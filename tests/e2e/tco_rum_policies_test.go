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

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("TCORumPolicies schema validation", func() {
	It("should reject a policy with more than 50 subsystem names", func(ctx context.Context) {
		names := make([]string, 51)
		for i := range names {
			names[i] = fmt.Sprintf("subsystem-%d", i)
		}
		policy := &coralogixv1alpha1.TCORumPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rum-too-many-subsystems",
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.TCORumPoliciesSpec{
				Policies: []coralogixv1alpha1.TCORumPolicy{{
					Name:       "over-limit",
					Priority:   "low",
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

	It("should reject a policy that sets both severities and dpxlExpression", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCORumPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rum-both-rules",
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.TCORumPoliciesSpec{
				Policies: []coralogixv1alpha1.TCORumPolicy{{
					Name:           "conflicting-rules",
					Priority:       "low",
					Severities:     []coralogixv1alpha1.TCOPolicySeverity{"info"},
					DpxlExpression: ptr.To("<v1>$d.applicationname == 'prod'"),
				}},
			},
		}
		err := ClientsInstance.GetControllerRuntimeClient().Create(ctx, policy)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mutually exclusive"))
	})
})

var _ = Describe("TCORumPolicies", Serial, func() {
	var (
		crClient       client.Client
		policiesClient *tcopolicies.PoliciesServiceAPIService
		rumPolicyName  = "tco-rum-policies-sample"
		TCORumPolicies *coralogixv1alpha1.TCORumPolicies
	)

	// listRumPolicies reads the RUM policy collection from the Coralogix backend.
	listRumPolicies := func(ctx context.Context) []tcopolicies.Policy {
		resp, _, err := policiesClient.PoliciesServiceGetCompanyPolicies(ctx).
			SourceType(tcopolicies.V1SOURCETYPE_SOURCE_TYPE_RUM).
			Execute()
		Expect(err).ToNot(HaveOccurred())
		return resp.Policies
	}

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		cfg := openapicxsdk.NewConfigBuilder().WithAPIKeyEnv().WithRegionEnv().Build()
		policiesClient = openapicxsdk.NewClientSet(cfg).TCOPolicies()
		TCORumPolicies = &coralogixv1alpha1.TCORumPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rumPolicyName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.TCORumPoliciesSpec{
				Policies: []coralogixv1alpha1.TCORumPolicy{
					{
						Name:       "sample policy",
						Priority:   "medium",
						Severities: []coralogixv1alpha1.TCOPolicySeverity{"critical", "error"},
						Applications: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"prod"},
							RuleType: "start_with",
						},
						Subsystems: &coralogixv1alpha1.TCOPolicyRule{
							Names:    []string{"mobile"},
							RuleType: "is",
						},
					},
					{
						Name:           "dpxl policy",
						Priority:       "medium",
						DpxlExpression: ptr.To("<v1>$d.applicationname == 'prod'"),
					},
				},
			},
		}
	})

	It("Should create TCORumPolicies successfully", func(ctx context.Context) {
		By("Creating TCORumPolicies")
		Expect(crClient.Create(ctx, TCORumPolicies)).To(Succeed())

		By("Verifying TCORumPolicies is synced")
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.TCORumPolicies{}
			g.Expect(crClient.Get(ctx, client.ObjectKey{Name: rumPolicyName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying policies exist in the Coralogix backend")
		Eventually(func() []tcopolicies.Policy {
			return listRumPolicies(ctx)
		}, time.Minute, time.Second).Should(HaveLen(2))

		By("Deleting the TCORumPolicies")
		Expect(crClient.Delete(ctx, TCORumPolicies)).To(Succeed())
		Eventually(func() []tcopolicies.Policy {
			return listRumPolicies(ctx)
		}, time.Minute, time.Second).Should(BeEmpty())
	})
})
