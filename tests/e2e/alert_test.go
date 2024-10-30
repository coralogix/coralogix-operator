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
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("Alert", Ordered, func() {
	var (
		crClient     client.Client
		alertsClient *cxsdk.AlertsClient
		alertID      string
		alert        *coralogixv1alpha1.Alert
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		alertsClient = ClientsInstance.GetCoralogixClientSet().Alerts()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Alert")
		alertName := "promql-alert"
		alert = &coralogixv1alpha1.Alert{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.AlertSpec{
				Name:        alertName,
				Description: "alert from k8s operator",
				Severity:    "Critical",
				NotificationGroups: []coralogixv1alpha1.NotificationGroup{
					{
						GroupByFields: []string{"coralogix.metadata.sdkId"},
						Notifications: []coralogixv1alpha1.Notification{
							{
								NotifyOn:                  "TriggeredOnly",
								IntegrationName:           ptr.To("Email"),
								RetriggeringPeriodMinutes: 1,
							},
						},
					},
				},
				Scheduling: &coralogixv1alpha1.Scheduling{
					DaysEnabled: []coralogixv1alpha1.Day{"Wednesday", "Thursday"},
					TimeZone:    "UTC+02",
					StartTime:   ptr.To(coralogixv1alpha1.Time("08:30")),
					EndTime:     ptr.To(coralogixv1alpha1.Time("20:30")),
				},
				AlertType: coralogixv1alpha1.AlertType{
					Metric: &coralogixv1alpha1.Metric{
						Promql: &coralogixv1alpha1.Promql{
							SearchQuery: "http_requests_total{status!~\"4..\"}",
							Conditions: coralogixv1alpha1.PromqlConditions{
								AlertWhen:                  "More",
								Threshold:                  utils.FloatToQuantity(3),
								SampleThresholdPercentage:  50,
								TimeWindow:                 "TwelveHours",
								MinNonNullValuesPercentage: ptr.To(10),
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, alert)).To(Succeed())

		By("Fetching the Alert ID")
		fetchedAlert := &coralogixv1alpha1.Alert{}
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
			return status.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})
