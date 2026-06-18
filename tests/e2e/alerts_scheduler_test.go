// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/v2/api/coralogix"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("AlertScheduler", Ordered, func() {
	var (
		crClient             client.Client
		alertSchedulerClient *cxsdk.AlertSchedulerClient
		alertSchedulerID     string
		alertScheduler       *coralogixv1alpha1.AlertScheduler
		alertSchedulerName   = "alert-scheduler-sample"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		alertSchedulerClient = ClientsInstance.GetCoralogixClientSet().AlertSchedulers()
		alertScheduler = getSampleAlertScheduler(alertSchedulerName)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating AlertScheduler")
		Expect(crClient.Create(ctx, alertScheduler)).To(Succeed())

		By("Fetching the AlertScheduler ID")
		fetchedScheduler := &coralogixv1alpha1.AlertScheduler{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSchedulerName, Namespace: testNamespace}, fetchedScheduler)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedScheduler.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedScheduler.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedScheduler.Status.ID != nil {
				alertSchedulerID = *fetchedScheduler.Status.ID
				return nil
			}
			return fmt.Errorf("alert scheduler ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying AlertScheduler exists in Coralogix backend")
		Eventually(func() error {
			_, err := alertSchedulerClient.Get(ctx, &cxsdk.GetAlertSchedulerRuleRequest{AlertSchedulerRuleId: alertSchedulerID})
			return err
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying the meta-label filter is preserved in Coralogix backend")
		expectBackendMetaLabelsFilter(ctx, alertSchedulerClient, alertSchedulerID, alertScheduler.Spec.Filter.WhatExpression)
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the AlertScheduler")
		newSchedulerName := "alert-scheduler-updated"
		modifiedScheduler := alertScheduler.DeepCopy()
		modifiedScheduler.Spec.Name = newSchedulerName
		Expect(crClient.Patch(ctx, modifiedScheduler, client.MergeFrom(alertScheduler))).To(Succeed())

		By("Verifying AlertScheduler is updated in Coralogix backend")
		Eventually(func() string {
			getSchedulerRes, err := alertSchedulerClient.Get(ctx, &cxsdk.GetAlertSchedulerRuleRequest{AlertSchedulerRuleId: alertSchedulerID})
			Expect(err).ToNot(HaveOccurred())
			if getSchedulerRes.AlertSchedulerRule == nil {
				return ""
			}
			return getSchedulerRes.AlertSchedulerRule.Name
		}, time.Minute, time.Second).Should(Equal(newSchedulerName))
	})

	It("Should be rejected when filter has both MetaLabels and Alerts", func(ctx context.Context) {
		By("Creating an AlertScheduler with both MetaLabels and Alerts")
		invalidScheduler := alertScheduler.DeepCopy()
		invalidScheduler.Spec.Filter.Alerts = []coralogixv1alpha1.AlertRef{
			{
				ResourceRef: &coralogixv1alpha1.ResourceRef{
					Name: "invalid-alert",
				},
			},
		}
		err := crClient.Create(ctx, invalidScheduler)
		Expect(err.Error()).To(ContainSubstring("Exactly one of allAlerts, metaLabels, alerts, or alertUniqueIds must be set"))
	})

	It("Should be rejected when allAlerts is false", func(ctx context.Context) {
		By("Creating an AlertScheduler with allAlerts set to false")
		invalidScheduler := alertScheduler.DeepCopy()
		invalidScheduler.Name = "alert-scheduler-all-alerts-false"
		invalidScheduler.Spec.Name = invalidScheduler.Name
		invalidScheduler.Spec.Filter.MetaLabels = nil
		invalidScheduler.Spec.Filter.AllAlerts = ptr.To(false)

		err := crClient.Create(ctx, invalidScheduler)
		Expect(err.Error()).To(ContainSubstring("allAlerts must be true when set"))
	})

	It("Should sync an all-alert filter", func(ctx context.Context) {
		scheduler := getSampleAlertScheduler(fmt.Sprintf("alert-scheduler-all-alerts-%d", time.Now().UnixNano()))
		scheduler.Spec.Filter.MetaLabels = nil
		scheduler.Spec.Filter.AllAlerts = ptr.To(true)

		By("Creating an AlertScheduler that applies to all alerts")
		id := createAlertSchedulerAndWait(ctx, crClient, scheduler)

		By("Verifying the all-alert filter is preserved in Coralogix backend")
		expectBackendUniqueIDsFilter(ctx, alertSchedulerClient, id, scheduler.Spec.Filter.WhatExpression, nil)

		By("Deleting the all-alert AlertScheduler")
		Expect(crClient.Delete(ctx, scheduler)).To(Succeed())
	})

	It("Should sync resource-ref and direct alert-ID filters", func(ctx context.Context) {
		alertName := fmt.Sprintf("alert-for-scheduler-%d", time.Now().UnixNano())
		alert := getSampleAlertForScheduler(alertName, testNamespace)

		By("Creating a referenced Alert")
		Expect(crClient.Create(ctx, alert)).To(Succeed())
		alertID := waitForAlertID(ctx, crClient, alertName)

		resourceRefScheduler := getSampleAlertScheduler(fmt.Sprintf("alert-scheduler-resource-ref-%d", time.Now().UnixNano()))
		resourceRefScheduler.Spec.Filter.MetaLabels = nil
		resourceRefScheduler.Spec.Filter.Alerts = []coralogixv1alpha1.AlertRef{
			{ResourceRef: &coralogixv1alpha1.ResourceRef{Name: alertName}},
		}

		By("Creating an AlertScheduler that selects an Alert resource reference")
		resourceRefSchedulerID := createAlertSchedulerAndWait(ctx, crClient, resourceRefScheduler)
		expectBackendUniqueIDsFilter(ctx, alertSchedulerClient, resourceRefSchedulerID, resourceRefScheduler.Spec.Filter.WhatExpression, []string{alertID})

		directIDScheduler := getSampleAlertScheduler(fmt.Sprintf("alert-scheduler-direct-id-%d", time.Now().UnixNano()))
		directIDScheduler.Spec.Filter.MetaLabels = nil
		directIDScheduler.Spec.Filter.AlertUniqueIDs = []string{alertID}

		By("Creating an AlertScheduler that selects a direct backend alert ID")
		directIDSchedulerID := createAlertSchedulerAndWait(ctx, crClient, directIDScheduler)
		expectBackendUniqueIDsFilter(ctx, alertSchedulerClient, directIDSchedulerID, directIDScheduler.Spec.Filter.WhatExpression, []string{alertID})

		By("Deleting resource-ref and direct-ID AlertSchedulers")
		Expect(crClient.Delete(ctx, resourceRefScheduler)).To(Succeed())
		Expect(crClient.Delete(ctx, directIDScheduler)).To(Succeed())
		Expect(crClient.Delete(ctx, alert)).To(Succeed())
	})

	It("Should be rejected when schedule has both OneTime and Recurring", func(ctx context.Context) {
		By("Creating an AlertScheduler with both OneTime and Recurring")
		invalidScheduler := alertScheduler.DeepCopy()
		invalidScheduler.Spec.Schedule.OneTime = &coralogixv1alpha1.TimeFrame{
			StartTime: "2026-03-14T00:00:00.000",
			Timezone:  "UTC+0",
			Duration: &coralogixv1alpha1.Duration{
				ForOver:   2,
				Frequency: "hours",
			},
		}

		err := crClient.Create(ctx, invalidScheduler)
		Expect(err.Error()).To(ContainSubstring("Exactly one of oneTime or recurring must be set"))
	})

	It("Should be rejected when recurring has both always and dynamic", func(ctx context.Context) {
		By("Creating an AlertScheduler with both always and dynamic")
		invalidScheduler := alertScheduler.DeepCopy()
		invalidScheduler.Spec.Schedule.Recurring.Always = &coralogixv1alpha1.Always{}
		err := crClient.Create(ctx, invalidScheduler)
		Expect(err.Error()).To(ContainSubstring("Exactly one of always or dynamic must be set"))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the AlertScheduler")
		Expect(crClient.Delete(ctx, alertScheduler)).To(Succeed())

		By("Verifying AlertScheduler is deleted from Coralogix backend")
		Eventually(func() *cxsdk.AlertSchedulerRule {
			getRes, _ := alertSchedulerClient.Get(ctx, &cxsdk.GetAlertSchedulerRuleRequest{AlertSchedulerRuleId: alertSchedulerID})
			return getRes.AlertSchedulerRule
		}, time.Minute, time.Second).Should(BeNil())
	})
})

func createAlertSchedulerAndWait(ctx context.Context, crClient client.Client, scheduler *coralogixv1alpha1.AlertScheduler) string {
	Expect(crClient.Create(ctx, scheduler)).To(Succeed())

	fetchedScheduler := &coralogixv1alpha1.AlertScheduler{}
	var schedulerID string
	Eventually(func(g Gomega) {
		g.Expect(crClient.Get(ctx, types.NamespacedName{Name: scheduler.Name, Namespace: scheduler.Namespace}, fetchedScheduler)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(fetchedScheduler.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
		g.Expect(fetchedScheduler.Status.PrintableStatus).To(Equal("RemoteSynced"))
		g.Expect(fetchedScheduler.Status.ID).ToNot(BeNil())
		schedulerID = *fetchedScheduler.Status.ID
	}, time.Minute, time.Second).Should(Succeed())

	return schedulerID
}

func waitForAlertID(ctx context.Context, crClient client.Client, name string) string {
	fetchedAlert := &coralogixv1beta1.Alert{}
	var alertID string
	Eventually(func(g Gomega) {
		g.Expect(crClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, fetchedAlert)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(fetchedAlert.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
		g.Expect(fetchedAlert.Status.PrintableStatus).To(Equal("RemoteSynced"))
		g.Expect(fetchedAlert.Status.ID).ToNot(BeNil())
		alertID = *fetchedAlert.Status.ID
	}, time.Minute, time.Second).Should(Succeed())

	return alertID
}

func expectBackendMetaLabelsFilter(ctx context.Context, alertSchedulerClient *cxsdk.AlertSchedulerClient, schedulerID, whatExpression string) {
	Eventually(func(g Gomega) {
		getSchedulerRes, err := alertSchedulerClient.Get(ctx, &cxsdk.GetAlertSchedulerRuleRequest{AlertSchedulerRuleId: schedulerID})
		g.Expect(err).ToNot(HaveOccurred())
		filter := getSchedulerRes.AlertSchedulerRule.GetFilter()
		metaLabels := filter.GetAlertMetaLabels()
		g.Expect(metaLabels).ToNot(BeNil())
		g.Expect(filter.GetWhatExpression()).To(Equal(whatExpression))
		g.Expect(metaLabels.GetValue()).ToNot(BeEmpty())
	}, time.Minute, time.Second).Should(Succeed())
}

func expectBackendUniqueIDsFilter(ctx context.Context, alertSchedulerClient *cxsdk.AlertSchedulerClient, schedulerID, whatExpression string, wantIDs []string) {
	Eventually(func(g Gomega) {
		getSchedulerRes, err := alertSchedulerClient.Get(ctx, &cxsdk.GetAlertSchedulerRuleRequest{AlertSchedulerRuleId: schedulerID})
		g.Expect(err).ToNot(HaveOccurred())
		filter := getSchedulerRes.AlertSchedulerRule.GetFilter()
		alertUniqueIDs := filter.GetAlertUniqueIds()
		g.Expect(alertUniqueIDs).ToNot(BeNil())
		g.Expect(filter.GetWhatExpression()).To(Equal(whatExpression))
		gotIDs := alertUniqueIDs.GetValue()
		if wantIDs == nil {
			g.Expect(gotIDs).To(BeEmpty())
			return
		}
		g.Expect(reflect.DeepEqual(gotIDs, wantIDs)).To(BeTrue())
	}, time.Minute, time.Second).Should(Succeed())
}

func getSampleAlertScheduler(name string) *coralogixv1alpha1.AlertScheduler {
	return &coralogixv1alpha1.AlertScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.AlertSchedulerSpec{
			Name:        name,
			Description: "This is a sample alert scheduler",
			Enabled:     true,
			Filter: coralogixv1alpha1.Filter{
				WhatExpression: "source logs | filter $d.cpodId:string == '122'",
				MetaLabels: []coralogixv1alpha1.MetaLabel{
					{Key: "environment", Value: ptr.To("production")},
				},
			},
			Schedule: coralogixv1alpha1.Schedule{
				Operation: "mute",
				Recurring: &coralogixv1alpha1.Recurring{
					Dynamic: &coralogixv1alpha1.Dynamic{
						RepeatEvery: 1,
						Frequency: &coralogixv1alpha1.Frequency{
							Weekly: &coralogixv1alpha1.Weekly{
								Days: []coralogixv1alpha1.Day{"Monday", "Thursday"},
							},
						},
						TimeFrame: &coralogixv1alpha1.TimeFrame{
							StartTime: "2026-03-14T00:00:00.000",
							Timezone:  "UTC+0",
							Duration: &coralogixv1alpha1.Duration{
								ForOver:   2,
								Frequency: "hours",
							},
						},
					},
				},
			},
		},
	}
}

func getSampleAlertForScheduler(name, namespace string) *coralogixv1beta1.Alert {
	return &coralogixv1beta1.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1beta1.AlertSpec{
			Name:        name,
			Description: "alert for alert scheduler e2e",
			Priority:    coralogixv1beta1.AlertPriorityP1,
			NotificationGroup: &coralogixv1beta1.NotificationGroup{
				Webhooks: []coralogixv1beta1.WebhookSettings{
					{
						NotifyOn: coralogixv1beta1.NotifyOnTriggeredOnly,
						RetriggeringPeriod: coralogixv1beta1.RetriggeringPeriod{
							Minutes: ptr.To(int64(1)),
						},
						Integration: coralogixv1beta1.IntegrationType{
							Recipients: []string{"example@coralogix.com"},
						},
					},
				},
			},
			TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
				MetricThreshold: &coralogixv1beta1.MetricThreshold{
					MissingValues: coralogixv1beta1.MetricMissingValues{
						MinNonNullValuesPct: ptr.To(int64(10)),
					},
					MetricFilter: coralogixv1beta1.MetricFilter{
						Promql: "http_requests_total{status!~\"4..\"}",
					},
					NoDataPolicy: &coralogixv1beta1.NoDataPolicy{
						State:             coralogixv1beta1.NoDataPolicyStateAlerting,
						AutoRetireSeconds: ptr.To(int32(1800)),
					},
					EvaluationDelayMs: ptr.To(int32(60000)),
					Rules: []coralogixv1beta1.MetricThresholdRule{
						{
							Condition: coralogixv1beta1.MetricThresholdRuleCondition{
								Threshold:     coralogix.FloatToQuantity(3),
								ForOverPct:    50,
								ConditionType: coralogixv1beta1.MetricThresholdConditionTypeMoreThan,
								OfTheLast: coralogixv1beta1.MetricTimeWindow{
									DynamicDuration: ptr.To("12h"),
								},
							},
						},
					},
				},
			},
		},
	}
}
