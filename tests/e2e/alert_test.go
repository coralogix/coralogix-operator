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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/coralogix/coralogix-operator/api/coralogix"
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
		Eventually(func() bool {
			getAlertRes, err := alertsClient.Get(ctx, &cxsdk.GetAlertDefRequest{Id: wrapperspb.String(alertID)})
			Expect(err).ToNot(HaveOccurred())
			return getAlertRes.GetAlertDef().GetUpdatedTime().AsTime().
				After(getAlertRes.GetAlertDef().GetCreatedTime().AsTime())
		}, time.Minute, time.Second).Should(BeTrue())
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
		Expect(err.Error()).To(ContainSubstring("only one alert type is allowed"))
	})
})
