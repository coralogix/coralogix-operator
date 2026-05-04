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

package v1alpha1

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var _ = Describe("TCOLogsPolicies validation", func() {
	It("should reject a policy with more than 50 subsystem names", func(ctx context.Context) {
		names := make([]string, 51)
		for i := range names {
			names[i] = fmt.Sprintf("subsystem-%d", i)
		}
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "too-many-subsystems",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
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
		err := k8sClient.Create(ctx, policy)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too many"))
	})

	It("should accept a policy with exactly 50 subsystem names", func(ctx context.Context) {
		names := make([]string, 50)
		for i := range names {
			names[i] = fmt.Sprintf("subsystem-%d", i)
		}
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "max-subsystems",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "at-limit",
					Priority:   "low",
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Subsystems: &coralogixv1alpha1.TCOPolicyRule{
						Names:    names,
						RuleType: "is",
					},
				}},
			},
		}
		Expect(k8sClient.Create(ctx, policy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, policy)).To(Succeed())
	})
})
