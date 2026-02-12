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
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	customenrichments "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/custom_enrichments_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("CustomEnrichment", Ordered, func() {
	var (
		crClient                client.Client
		customEnrichmentsClient *customenrichments.CustomEnrichmentsServiceAPIService
		customEnrichmentID      string
		customEnrichment        *coralogixv1alpha1.CustomEnrichment
		configMap               *corev1.ConfigMap
		customEnrichmentName    = "custom-enrichment-e2e-sample"
		configMapName           = "custom-enrichment-e2e-csv"
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		cfg := cxsdk.NewConfigBuilder().WithAPIKeyEnv().WithRegionEnv().Build()
		customEnrichmentsClient = cxsdk.NewClientSet(cfg).CustomEnrichments()
		configMap = newConfigMap(configMapName, testNamespace)
		customEnrichment = newCustomEnrichment(customEnrichmentName, configMapName, testNamespace)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating ConfigMap with CSV data")
		Expect(crClient.Create(ctx, configMap)).To(Succeed())

		By("Creating CustomEnrichment")
		Expect(crClient.Create(ctx, customEnrichment)).To(Succeed())

		By("Fetching the CustomEnrichment ID")
		fetched := &coralogixv1alpha1.CustomEnrichment{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: customEnrichmentName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetched.Status.Id != nil {
				customEnrichmentID = *fetched.Status.Id
				return nil
			}
			return fmt.Errorf("custom enrichment ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying CustomEnrichment exists in Coralogix backend")
		Eventually(func() error {
			id, err := strconv.ParseInt(customEnrichmentID, 10, 64)
			if err != nil {
				return err
			}
			_, _, err = customEnrichmentsClient.CustomEnrichmentServiceGetCustomEnrichment(ctx, id).Execute()
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the CustomEnrichment")
		newName := "custom-enrichment-e2e-updated"
		newDescription := "Updated description for e2e test"
		modified := customEnrichment.DeepCopy()
		modified.Spec.Name = newName
		modified.Spec.Description = newDescription
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(customEnrichment))).To(Succeed())

		By("Verifying CustomEnrichment is updated in Coralogix backend")
		Eventually(func(g Gomega) {
			id, err := strconv.ParseInt(customEnrichmentID, 10, 64)
			g.Expect(err).NotTo(HaveOccurred())
			resp, _, err := customEnrichmentsClient.CustomEnrichmentServiceGetCustomEnrichment(ctx, id).Execute()
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.CustomEnrichment.Name).ToNot(BeNil())
			g.Expect(*resp.CustomEnrichment.Name).To(Equal(newName))
			g.Expect(resp.CustomEnrichment.Description).ToNot(BeNil())
			g.Expect(*resp.CustomEnrichment.Description).To(Equal(newDescription))
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the CustomEnrichment")
		Expect(crClient.Delete(ctx, customEnrichment)).To(Succeed())

		By("Deleting the ConfigMap")
		Expect(crClient.Delete(ctx, configMap)).To(Succeed())

		By("Verifying CustomEnrichment is deleted from Coralogix backend")
		Eventually(func(g Gomega) bool {
			id, err := strconv.ParseInt(customEnrichmentID, 10, 64)
			g.Expect(err).NotTo(HaveOccurred())
			_, httpResp, err := customEnrichmentsClient.CustomEnrichmentServiceGetCustomEnrichment(ctx, id).Execute()
			if err != nil && httpResp != nil {
				return httpResp.StatusCode == 404
			}
			return false
		}, time.Minute, time.Second).Should(BeTrue())
	})
})

func newConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string]string{"csv": sampleCustomEnrichmentCSV},
	}
}

func newCustomEnrichment(name, configMapName, namespace string) *coralogixv1alpha1.CustomEnrichment {
	return &coralogixv1alpha1.CustomEnrichment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: coralogixv1alpha1.CustomEnrichmentSpec{
			Name:        name,
			Description: "E2E test custom enrichment that uses a ConfigMap as the source of enrichment data.",
			ConfigMapRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				Key:                  "csv",
			},
		},
	}
}

const sampleCustomEnrichmentCSV = `Date,day of week
7/30/21,Friday
7/31/21,Saturday
8/1/21,Sunday
8/2/21,Monday
8/4/21,Wednesday
8/5/21,Thursday
8/6/21,Friday
8/7/21,Saturday
8/8/21,Sunday
8/9/21,Monday
8/10/21,Tuesday
8/11/21,Wednesday
8/12/21,Thursday
8/13/21,Friday
8/14/21,Saturday
8/15/21,Sunday
8/16/21,Monday
8/17/21,Tuesday
8/18/21,Wednesday`
