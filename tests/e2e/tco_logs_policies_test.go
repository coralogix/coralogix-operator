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

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("TCOLogsPolicies schema validation", func() {
	It("should reject a policy with more than 50 subsystem names", func(ctx context.Context) {
		names := make([]string, 51)
		for i := range names {
			names[i] = fmt.Sprintf("subsystem-%d", i)
		}
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "too-many-subsystems",
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "over-limit",
					Priority:   ptr.To("low"),
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Subsystems: &coralogixv1alpha1.TCOPolicyRule{
						Names:    names,
						RuleType: "is",
					},
				}},
			},
		}
		err := ClientsInstance.GetControllerRuntimeClient().Create(ctx, policy)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too many"))
	})
})

var _ = Describe("TCOLogsPolicies", Serial, func() {
	var (
		crClient        client.Client
		tcoClient       *cxsdk.TCOPoliciesClient
		logsPolicyName  = "tco-logs-policies-sample"
		TCOLogsPolicies *coralogixv1alpha1.TCOLogsPolicies
		policies        []*cxsdk.TCOPolicy
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		tcoClient = ClientsInstance.GetCoralogixClientSet().TCOPolicies()
		TCOLogsPolicies = &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      logsPolicyName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{
					{
						Name:       "sample policy",
						Priority:   ptr.To("high"),
						Severities: []coralogixv1alpha1.TCOPolicySeverity{"critical", "error"},
						Applications: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"prod"},
							RuleType: "start_with",
						},
						Subsystems: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"mobile"},
							RuleType: "is",
						},
						ArchiveRetention: &coralogixv1alpha1.ArchiveRetention{
							BackendRef: coralogixv1alpha1.ArchiveRetentionBackendRef{
								Name: "Default",
							},
						},
					},
					{
						Name:       "sample policy 2",
						Priority:   ptr.To("high"),
						Disabled:   ptr.To(true),
						Severities: []coralogixv1alpha1.TCOPolicySeverity{"critical", "error"},
						Applications: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"prod"},
							RuleType: "start_with",
						},
						Subsystems: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"mobile"},
							RuleType: "is",
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

	It("Should create TCOLogsPolicies successfully", func(ctx context.Context) {
		By("Creating TCOLogsPolicies")
		Expect(crClient.Create(ctx, TCOLogsPolicies)).To(Succeed())

		By("Verifying TCOLogsPolicies is synced")
		Eventually(func(g Gomega) {
			fetchedTCOLogsPolicies := &coralogixv1alpha1.TCOLogsPolicies{}
			g.Expect(crClient.Get(ctx, client.ObjectKey{Name: logsPolicyName, Namespace: testNamespace}, fetchedTCOLogsPolicies)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedTCOLogsPolicies.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedTCOLogsPolicies.Status.PrintableStatus).To(Equal("RemoteSynced"))
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying policies exist in the Coralogix backend")
		Eventually(func() []*cxsdk.TCOPolicy {
			listRes, err := tcoClient.List(ctx, &cxsdk.GetCompanyPoliciesRequest{SourceType: ptr.To(cxsdk.TCOPolicySourceTypeLogs)})
			Expect(err).ToNot(HaveOccurred())
			policies = listRes.Policies
			return policies
		}, time.Minute, time.Second).Should(HaveLen(2))

		Expect(policies[0].Name.Value).To(Equal(TCOLogsPolicies.Spec.Policies[0].Name))

		By("Deleting the TCOLogsPolicies")
		Expect(crClient.Delete(ctx, TCOLogsPolicies)).To(Succeed())
		Eventually(func() []*cxsdk.TCOPolicy {
			listRes, err := tcoClient.List(ctx, &cxsdk.GetCompanyPoliciesRequest{SourceType: ptr.To(cxsdk.TCOPolicySourceTypeLogs)})
			Expect(err).ToNot(HaveOccurred())
			return listRes.Policies
		}, time.Minute, time.Second).Should(BeEmpty())
	})
})

var _ = Describe("TCOLogsPolicies with targets", Serial, func() {
	var (
		crClient       client.Client
		policiesClient *tcopolicies.PoliciesServiceAPIService
		policyName     = "tco-logs-policies-targets-sample"
		cr             *coralogixv1alpha1.TCOLogsPolicies
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		cfg := openapicxsdk.NewConfigBuilder().WithAPIKeyEnv().WithRegionEnv().Build()
		policiesClient = openapicxsdk.NewClientSet(cfg).TCOPolicies()
		cr = &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      policyName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{
					{
						Name:       "targets-policy",
						Severities: []coralogixv1alpha1.TCOPolicySeverity{"info", "debug"},
						Targets: []coralogixv1alpha1.TCOPolicyTarget{
							{
								Dataset:  "myDataset",
								Priority: ptr.To("low"),
							},
							{
								Dataset:  "myDataset2",
								Priority: ptr.To("medium"),
							},
						},
					},
				},
			},
		}
	})

	It("should create a policy with targets and verify targets in the backend", func(ctx context.Context) {
		By("Creating TCOLogsPolicies with targets")
		Expect(crClient.Create(ctx, cr)).To(Succeed())

		By("Verifying the CR is synced")
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.TCOLogsPolicies{}
			g.Expect(crClient.Get(ctx, client.ObjectKey{Name: policyName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying targets exist in the backend via REST API")
		Eventually(func(g Gomega) {
			resp, _, err := policiesClient.PoliciesServiceGetCompanyPolicies(ctx).
				SourceType(tcopolicies.V1SOURCETYPE_SOURCE_TYPE_LOGS).
				Execute()
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.Policies).NotTo(BeEmpty())
			found := false
			for _, p := range resp.Policies {
				if p.Name == "targets-policy" {
					g.Expect(p.Targets).To(HaveLen(2))
					found = true
				}
			}
			g.Expect(found).To(BeTrue(), "policy 'targets-policy' not found in backend")
		}, time.Minute, time.Second).Should(Succeed())

		By("Deleting the CR")
		Expect(crClient.Delete(ctx, cr)).To(Succeed())
		Eventually(func(g Gomega) {
			resp, _, err := policiesClient.PoliciesServiceGetCompanyPolicies(ctx).
				SourceType(tcopolicies.V1SOURCETYPE_SOURCE_TYPE_LOGS).
				Execute()
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(resp.Policies).To(BeEmpty())
		}, time.Minute, time.Second).Should(Succeed())
	})
})
