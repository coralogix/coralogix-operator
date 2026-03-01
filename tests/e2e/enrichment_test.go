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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	enrichments "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/enrichments_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

const (
	enrichmentName          = "enrichment-sample"
	customEnrichmentRefName = "custom-enrichment-csv-sample"
)

var _ = PDescribe("Enrichment", Ordered, func() {
	var (
		crClient           client.Client
		enrichmentsClient  *enrichments.EnrichmentsServiceAPIService
		enrichment         *coralogixv1alpha1.Enrichment
		customEnrichmentCR *coralogixv1alpha1.CustomEnrichment
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		cfg := cxsdk.NewConfigBuilder().WithAPIKeyEnv().WithRegionEnv().Build()
		enrichmentsClient = cxsdk.NewClientSet(cfg).Enrichments()
		customEnrichmentCR = newCSVCustomEnrichment(testNamespace)
		enrichment = newEnrichmentFromSample(testNamespace)
	})

	It("Should create Enrichment successfully and override remote enrichments", func(ctx context.Context) {
		By("Creating CustomEnrichment (required for the custom enrichment type)")
		Expect(crClient.Create(ctx, customEnrichmentCR)).To(Succeed())

		By("Creating Enrichment (override resource)")
		Expect(crClient.Create(ctx, enrichment)).To(Succeed())

		By("Verifying Enrichment is synced")
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.Enrichment{}
			g.Expect(crClient.Get(ctx, client.ObjectKey{Name: enrichmentName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying there are 3 enrichments in the Coralogix backend (override replaces all)")
		Eventually(func(g Gomega) {
			resp, _, err := enrichmentsClient.EnrichmentServiceGetEnrichments(ctx).Execute()
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.Enrichments).To(HaveLen(3))
		}, time.Minute, time.Second).Should(Succeed())

		By("Deleting the Enrichment")
		Expect(crClient.Delete(ctx, enrichment)).To(Succeed())

		By("Verifying there are 0 enrichments in the Coralogix backend after delete (override cleared)")
		Eventually(func(g Gomega) {
			resp, _, err := enrichmentsClient.EnrichmentServiceGetEnrichments(ctx).Execute()
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.Enrichments).To(BeEmpty())
		}, time.Minute, time.Second).Should(Succeed())
	})
})

func newEnrichmentFromSample(namespace string) *coralogixv1alpha1.Enrichment {
	return &coralogixv1alpha1.Enrichment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      enrichmentName,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.EnrichmentSpec{
			Enrichments: []coralogixv1alpha1.EnrichmentType{
				{
					SuspiciousIp: &coralogixv1alpha1.SuspiciousIpEnrichmentType{
						FieldName: "attributes.event.namespace",
					},
				},
				{
					GeoIp: &coralogixv1alpha1.GeoIpEnrichmentType{
						FieldName: "resource.attributes.service.name",
						WithAsn:   ptr.To(true),
					},
				},
				{
					Custom: &coralogixv1alpha1.CustomEnrichmentType{
						FieldName: "resource.attributes.service.name",
						CustomEnrichmentRef: coralogixv1alpha1.CustomEnrichmentRef{
							ResourceRef: &coralogixv1alpha1.ResourceRef{
								Name: customEnrichmentRefName,
							},
						},
					},
				},
			},
		},
	}
}

func newCSVCustomEnrichment(namespace string) *coralogixv1alpha1.CustomEnrichment {
	return &coralogixv1alpha1.CustomEnrichment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      customEnrichmentRefName,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.CustomEnrichmentSpec{
			Name:        fmt.Sprintf("custom-enrichment-%d", time.Now().UnixNano()),
			Description: "Sample custom enrichment for e2e that uses an inline CSV.",
			CSV: ptr.To(`Date,day of week
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
8/18/21,Wednesday`),
		},
	}
}
