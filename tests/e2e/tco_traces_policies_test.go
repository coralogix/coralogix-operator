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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("TCOTracesPolicies", func() {
	var (
		crClient          client.Client
		tcoClient         *cxsdk.TCOPoliciesClient
		tracesPolicyName  = "tco-traces-policies-sample"
		TCOTracesPolicies *coralogixv1alpha1.TCOTracesPolicies
		policies          []*cxsdk.TCOPolicy
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		tcoClient = ClientsInstance.GetCoralogixClientSet().TCOPolicies()
		TCOTracesPolicies = &coralogixv1alpha1.TCOTracesPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tracesPolicyName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.TCOTracesPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOTracesPolicy{
					{
						Name:     "sample policy",
						Priority: "high",
						Applications: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"prod"},
							RuleType: "is",
						},
						Subsystems: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"mobile"},
							RuleType: "is",
						},
						Actions: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"action1", "action2"},
							RuleType: "is",
						},
						Services: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"service", "system"},
							RuleType: "includes",
						},
						Tags: []coralogixv1alpha1.TCOPolicyTag{
							{
								Name:     "tags.app",
								Values:   []string{"purchases", "signups"},
								RuleType: "start_with",
							},
							{
								Name:     "tags.http",
								Values:   []string{"GET", "POST"},
								RuleType: "is",
							},
						},
						ArchiveRetention: &coralogixv1alpha1.ArchiveRetention{
							BackendRef: coralogixv1alpha1.ArchiveRetentionBackendRef{
								Name: "Default",
							},
						},
					},
				},
			},
		}
	})

	It("Should create TCOTracesPolicies successfully", FlakeAttempts(3), func(ctx context.Context) {
		By("Creating TCOTracesPolicies")
		Expect(crClient.Create(ctx, TCOTracesPolicies)).To(Succeed())

		By("Verifying TCOTracesPolicies is synced")
		Eventually(func(g Gomega) {
			fetchedTCOTracesPolicies := &coralogixv1alpha1.TCOTracesPolicies{}
			g.Expect(crClient.Get(ctx, client.ObjectKey{Name: tracesPolicyName, Namespace: testNamespace}, fetchedTCOTracesPolicies)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedTCOTracesPolicies.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedTCOTracesPolicies.Status.PrintableStatus).To(Equal("RemoteSynced"))
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying policies exist in the Coralogix backend")
		Eventually(func() []*cxsdk.TCOPolicy {
			listRes, err := tcoClient.List(ctx, &cxsdk.GetCompanyPoliciesRequest{SourceType: ptr.To(cxsdk.TCOPolicySourceTypeSpans)})
			Expect(err).ToNot(HaveOccurred())
			policies = listRes.Policies
			return policies
		}, time.Minute, time.Second).Should(HaveLen(1))

		Expect(policies[0].Name.Value).To(Equal(TCOTracesPolicies.Spec.Policies[0].Name))

		By("Deleting the TCOTracesPolicies")
		Expect(crClient.Delete(ctx, TCOTracesPolicies)).To(Succeed())
		Eventually(func() []*cxsdk.TCOPolicy {
			listRes, err := tcoClient.List(ctx, &cxsdk.GetCompanyPoliciesRequest{SourceType: ptr.To(cxsdk.TCOPolicySourceTypeSpans)})
			Expect(err).ToNot(HaveOccurred())
			return listRes.Policies
		}, time.Minute, time.Second).Should(BeEmpty())
	})
})
