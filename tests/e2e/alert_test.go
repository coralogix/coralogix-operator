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

	"github.com/coralogix/coralogix-operator/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/api/coralogix"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
)

var _ = Describe("Alert", Ordered, func() {
	var (
		crClient     client.Client
		alertsClient *cxsdk.AlertsClient
		alertID      string
		alert        *coralogixv1beta1.Alert
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		alertsClient = ClientsInstance.GetCoralogixClientSet().Alerts()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Slack Connector")
		connectorName := "slack-connector-for-alert"
		connector := getSampleSlackConnector(connectorName, testNamespace)
		Expect(crClient.Create(ctx, connector)).To(Succeed())

		By("Creating Slack Preset")
		presetName := "slack-preset-for-alert"
		preset := getSampleSlackPreset(presetName, testNamespace)
		Expect(crClient.Create(ctx, preset)).To(Succeed())

		By("Creating Alert")
		alertName := "promql-alert"
		alert = &coralogixv1beta1.Alert{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1beta1.AlertSpec{
				Name:        alertName,
				Description: "alert from k8s operator",
				Priority:    coralogixv1beta1.AlertPriorityP1,
				NotificationGroup: &coralogixv1beta1.NotificationGroup{
					Webhooks: []coralogixv1beta1.WebhookSettings{
						{
							NotifyOn: coralogixv1beta1.NotifyOnTriggeredOnly,
							RetriggeringPeriod: coralogixv1beta1.RetriggeringPeriod{
								Minutes: pointer.Uint32(1),
							},
							Integration: coralogixv1beta1.IntegrationType{
								Recipients: []string{"example@coralogix.com"},
							},
						},
					},
					Destinations: []coralogixv1beta1.Destination{
						{
							NotifyOn: coralogixv1beta1.NotifyOnTriggeredOnly,
							DestinationType: coralogixv1beta1.DestinationType{
								Slack: &coralogixv1beta1.SlackDestination{
									ConnectorRef: &coralogixv1beta1.NCRef{
										ResourceRef: &coralogixv1beta1.ResourceRef{
											Name: connectorName,
										},
									},
									PresetRef: &coralogixv1beta1.NCRef{
										ResourceRef: &coralogixv1beta1.ResourceRef{
											Name: presetName,
										}},
									TriggeredRoutingOverride: &coralogixv1beta1.SlackRoutingOverride{
										ConnectorOverride: &coralogixv1beta1.SlackConnectorOverride{
											Channel: "override",
										},
										PresetOverride: &coralogixv1beta1.SlackPresetOverride{
											StructuredFields: &coralogixv1beta1.PresetSlackStructuredFields{
												Title:       ptr.To("Override Title"),
												Description: ptr.To("Override Description"),
												Footer:      ptr.To("Override Footer"),
											},
										},
									},
								},
							},
						},
					},
				},
				Schedule: &coralogixv1beta1.AlertSchedule{
					TimeZone: "UTC+02",
					ActiveOn: &coralogixv1beta1.ActiveOn{
						DayOfWeek: []coralogixv1beta1.DayOfWeek{coralogixv1beta1.DayOfWeekWednesday, coralogixv1beta1.DayOfWeekThursday},
						StartTime: ptr.To(coralogixv1beta1.TimeOfDay("08:30")),
						EndTime:   ptr.To(coralogixv1beta1.TimeOfDay("20:30")),
					},
				},
				TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
					MetricThreshold: &coralogixv1beta1.MetricThreshold{
						MissingValues: coralogixv1beta1.MetricMissingValues{
							MinNonNullValuesPct: pointer.Uint32(10),
						},
						MetricFilter: coralogixv1beta1.MetricFilter{
							Promql: "http_requests_total{status!~\"4..\"}",
						},
						Rules: []coralogixv1beta1.MetricThresholdRule{
							{
								Condition: coralogixv1beta1.MetricThresholdRuleCondition{
									Threshold:     coralogix.FloatToQuantity(3),
									ForOverPct:    50,
									ConditionType: coralogixv1beta1.MetricThresholdConditionTypeMoreThan,
									OfTheLast: coralogixv1beta1.MetricTimeWindow{
										SpecificValue: coralogixv1beta1.MetricTimeWindowValue12Hours,
									},
								},
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, alert)).To(Succeed())

		By("Fetching the Alert ID")
		fetchedAlert := &coralogixv1beta1.Alert{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: alertName, Namespace: testNamespace}, fetchedAlert)).To(Succeed())
			if fetchedAlert.Status.ID != nil {
				alertID = *fetchedAlert.Status.ID
				return nil
			}
			return fmt.Errorf("Alert ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Alert exists in Coralogix backend")
		Eventually(func() error {
			_, err := alertsClient.Get(ctx, &cxsdk.GetAlertDefRequest{Id: wrapperspb.String(alertID)})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the Alert")
		newAlertName := "promql-alert-updated"
		modifiedAlert := alert.DeepCopy()
		modifiedAlert.Spec.Name = newAlertName
		Expect(crClient.Patch(ctx, modifiedAlert, client.MergeFrom(alert))).To(Succeed())

		By("Verifying Alert is updated in Coralogix backend")
		Eventually(func() *wrapperspb.StringValue {
			getAlertRes, err := alertsClient.Get(ctx, &cxsdk.GetAlertDefRequest{Id: wrapperspb.String(alertID)})
			Expect(err).ToNot(HaveOccurred())
			return getAlertRes.GetAlertDef().AlertDefProperties.GetName()
		}, time.Minute, time.Second).Should(Equal(wrapperspb.String(newAlertName)))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Alert")
		Expect(crClient.Delete(ctx, alert)).To(Succeed())

		By("Verifying Alert is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := alertsClient.Get(ctx, &cxsdk.GetAlertDefRequest{Id: wrapperspb.String(alertID)})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})

	It("Should store an err condition in status", func(ctx context.Context) {
		By("Creating Alert with a non-existing webhook")
		alertName := "promql-alert"
		newAlert := &coralogixv1beta1.Alert{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1beta1.AlertSpec{
				Name:        alertName,
				Description: "alert from k8s operator",
				Priority:    coralogixv1beta1.AlertPriorityP1,
				NotificationGroup: &coralogixv1beta1.NotificationGroup{
					Webhooks: []coralogixv1beta1.WebhookSettings{
						{
							NotifyOn: coralogixv1beta1.NotifyOnTriggeredOnly,
							RetriggeringPeriod: coralogixv1beta1.RetriggeringPeriod{
								Minutes: pointer.Uint32(1),
							},
							Integration: coralogixv1beta1.IntegrationType{
								IntegrationRef: &coralogixv1beta1.IntegrationRef{
									ResourceRef: &coralogixv1beta1.ResourceRef{
										Name: "non-existing-webhook",
									},
								},
							},
						},
					},
				},
				Schedule: &coralogixv1beta1.AlertSchedule{
					TimeZone: "UTC+02",
					ActiveOn: &coralogixv1beta1.ActiveOn{
						DayOfWeek: []coralogixv1beta1.DayOfWeek{coralogixv1beta1.DayOfWeekWednesday, coralogixv1beta1.DayOfWeekThursday},
						StartTime: ptr.To(coralogixv1beta1.TimeOfDay("08:30")),
						EndTime:   ptr.To(coralogixv1beta1.TimeOfDay("20:30")),
					},
				},
				TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
					MetricThreshold: &coralogixv1beta1.MetricThreshold{
						MissingValues: coralogixv1beta1.MetricMissingValues{
							MinNonNullValuesPct: pointer.Uint32(10),
						},
						MetricFilter: coralogixv1beta1.MetricFilter{
							Promql: "http_requests_total{status!~\"4..\"}",
						},
						Rules: []coralogixv1beta1.MetricThresholdRule{
							{
								Condition: coralogixv1beta1.MetricThresholdRuleCondition{
									Threshold:     coralogix.FloatToQuantity(3),
									ForOverPct:    50,
									ConditionType: coralogixv1beta1.MetricThresholdConditionTypeMoreThan,
									OfTheLast: coralogixv1beta1.MetricTimeWindow{
										SpecificValue: coralogixv1beta1.MetricTimeWindowValue12Hours,
									},
								},
							},
						},
					},
				},
			},
		}
		newAlert.Spec.Name = alertName

		err := crClient.Create(ctx, newAlert)
		Expect(err).ToNot(HaveOccurred())

		By("Fetching the Alert")
		fetchedAlert := &coralogixv1beta1.Alert{}
		Eventually(func(g Gomega) {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: alert.Name, Namespace: alert.Namespace}, fetchedAlert)).To(Succeed())

			Expect(fetchedAlert.Status.Conditions).To(HaveLen(2))

			Expect(meta.IsStatusConditionTrue(fetchedAlert.Status.Conditions, utils.ConditionTypeError)).To(BeTrue())

			Expect(meta.IsStatusConditionFalse(fetchedAlert.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())

		}, time.Minute, time.Second).Should(Succeed())

		webhook := &coralogixv1alpha1.OutboundWebhook{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "webhook",
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.OutboundWebhookSpec{
				Name: "webhook",
				OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
					Slack: &coralogixv1alpha1.Slack{
						Url: "https://slack.com",
					},
				},
			},
		}
		Expect(crClient.Create(ctx, webhook)).To(Succeed())

		By("Updating the Alert with a valid webhook")
		updateAlert := newAlert.DeepCopy()
		updateAlert.Spec.NotificationGroup.Webhooks[0].Integration = coralogixv1beta1.IntegrationType{
			IntegrationRef: &coralogixv1beta1.IntegrationRef{
				ResourceRef: &coralogixv1beta1.ResourceRef{
					Name: webhook.Name,
				},
			},
		}
		Expect(crClient.Patch(ctx, updateAlert, client.MergeFrom(newAlert))).To(Succeed())

		By("Fetching the Alert again")
		fetchedAlert = &coralogixv1beta1.Alert{}
		Eventually(func(g Gomega) {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: newAlert.Name, Namespace: newAlert.Namespace}, fetchedAlert)).To(Succeed())

			Expect(fetchedAlert.Status.Conditions).To(HaveLen(1))

			Expect(meta.IsStatusConditionTrue(fetchedAlert.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())

		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should deny creation of Alert with more then one alert type", func(ctx context.Context) {
		By("Creating Alert")
		alert.Spec.TypeDefinition.MetricAnomaly = &coralogixv1beta1.MetricAnomaly{
			MetricFilter: coralogixv1beta1.MetricFilter{
				Promql: "http_requests_total{status!~\"4..\"}",
			},
			Rules: []coralogixv1beta1.MetricAnomalyRule{
				{
					Condition: coralogixv1beta1.MetricAnomalyCondition{
						Threshold:     coralogix.FloatToQuantity(3),
						ForOverPct:    50,
						ConditionType: coralogixv1beta1.MetricAnomalyConditionTypeMoreThanUsual,
						OfTheLast: coralogixv1beta1.MetricTimeWindow{
							SpecificValue: coralogixv1beta1.MetricTimeWindowValue12Hours,
						},
					},
				},
			},
		}
		err := crClient.Create(ctx, alert)
		Expect(err.Error()).To(ContainSubstring("only one alert type should be set"))
	})

	It("Should deny creation of Alert with destination with two types", func(ctx context.Context) {
		By("Creating Alert")

		alert.Spec.NotificationGroup.Destinations[0].DestinationType.GenericHttps = &coralogixv1beta1.GenericHttpsDestination{
			ConnectorRef: &coralogixv1beta1.NCRef{
				ResourceRef: &coralogixv1beta1.ResourceRef{
					Name: "generic-https-connector",
				},
			},
			PresetRef: &coralogixv1beta1.NCRef{
				ResourceRef: &coralogixv1beta1.ResourceRef{
					Name: "generic-https-preset",
				}},
		}
		err := crClient.Create(ctx, alert)
		Expect(err.Error()).To(ContainSubstring("only one destination type should be set"))
	})
})
