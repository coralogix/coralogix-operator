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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("Slo", Ordered, func() {
	var (
		crClient   client.Client
		slosClient *cxsdk.SLOsClient
		sloID      string
		slo        *coralogixv1alpha1.SLO
		sloName    = "slo-sample"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		slosClient = ClientsInstance.GetCoralogixClientSet().SLOs()
		slo = getSampleSlo(sloName, testNamespace)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating SLO")
		Expect(crClient.Create(ctx, slo)).To(Succeed())

		By("Fetching the SLO ID")
		fetchedSlo := &coralogixv1alpha1.SLO{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: sloName, Namespace: testNamespace}, fetchedSlo)).To(Succeed())
			if fetchedSlo.Status.ID != nil {
				sloID = *fetchedSlo.Status.ID
				return nil
			}
			return fmt.Errorf("slo ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying SLO exists in Coralogix backend")
		Eventually(func() error {
			_, err := slosClient.Get(ctx, &cxsdk.GetServiceSloRequest{
				Id: sloID,
			})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the slo")
		newSloName := "slo-sample-updated"
		modifiedSlo := slo.DeepCopy()
		modifiedSlo.Spec.Name = newSloName
		Expect(crClient.Patch(ctx, modifiedSlo, client.MergeFrom(slo))).To(Succeed())

		By("Verifying slo is updated in Coralogix backend")
		Eventually(func() string {
			getSloRes, err := slosClient.Get(ctx, &cxsdk.GetServiceSloRequest{
				Id: sloID,
			})
			Expect(err).ToNot(HaveOccurred())
			return getSloRes.Slo.Name
		}, time.Minute, time.Second).Should(Equal(newSloName))
	})

	It("After deleted from Coralogix backend directly, it should be recreated based on configured interval",
		func(ctx context.Context) {
			By("Deleting the slo from Coralogix backend")
			_, err := slosClient.Delete(ctx, &cxsdk.DeleteServiceSloRequest{Id: sloID})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying slo is populated with a new ID after configured interval")
			var newSloID string
			Eventually(func() bool {
				fetchedSlo := &coralogixv1alpha1.SLO{}
				Expect(crClient.Get(ctx, types.NamespacedName{Name: sloName, Namespace: testNamespace}, fetchedSlo)).To(Succeed())
				if fetchedSlo.Status.ID == nil {
					return false
				}
				newSloID = *fetchedSlo.Status.ID
				return newSloID != sloID
			}, 2*time.Minute, time.Second).Should(BeTrue())

			By("Verifying slo with new the ID exists in Coralogix backend")
			Eventually(func() error {
				_, err := slosClient.Get(ctx, &cxsdk.GetServiceSloRequest{
					Id: newSloID,
				})
				return err
			}, time.Minute, time.Second).Should(Succeed())

		})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the slo")
		Expect(crClient.Delete(ctx, slo)).To(Succeed())

		By("Verifying slo is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := slosClient.Get(ctx, &cxsdk.GetServiceSloRequest{
				Id: sloID,
			})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})

func getSampleSlo(name, namespace string) *coralogixv1alpha1.SLO {
	timeFrame := coralogixv1alpha1.SloTimeFrame7d
	return &coralogixv1alpha1.SLO{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.SLOSpec{
			Name:        name,
			Description: ptr.To("This is a sample slo"),
			Labels: map[string]string{
				"team": "e2e-test",
			},
			TargetThresholdPercentage: *resource.NewQuantity(10, resource.DecimalSI),
			SliType: coralogixv1alpha1.SliType{
				RequestBasedMetricSli: &coralogixv1alpha1.RequestBasedMetricSli{
					GoodEvents: &coralogixv1alpha1.SloMetricEvent{
						Query: "sum(rate(coralogix_logs_events_total{app=\"coralogix-slo-example\", status=\"success\"}[5m]))",
					},
				},
			},
			Window: coralogixv1alpha1.SloWindow{
				TimeFrame: &timeFrame,
			},
		},
	}
}
