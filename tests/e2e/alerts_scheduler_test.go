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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
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
		alertScheduler = getSampleAlertScheduler(alertSchedulerName, testNamespace)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating AlertScheduler")
		Expect(crClient.Create(ctx, alertScheduler)).To(Succeed())

		By("Fetching the AlertScheduler ID")
		fetchedScheduler := &coralogixv1alpha1.AlertScheduler{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: alertSchedulerName, Namespace: testNamespace}, fetchedScheduler)).To(Succeed())
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
		Expect(err.Error()).To(ContainSubstring("Exactly one of metaLabels or alerts must be set"))
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

func getSampleAlertScheduler(name, namespace string) *coralogixv1alpha1.AlertScheduler {
	return &coralogixv1alpha1.AlertScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
