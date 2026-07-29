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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

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
					Priority:   ptr.To("low"),
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
					Priority:   ptr.To("low"),
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

	It("should reject a target with an empty dataset", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "empty-dataset-target",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "bad-target",
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Targets: []coralogixv1alpha1.TCOPolicyTarget{{
						Dataset:  "",
						Priority: ptr.To("low"),
					}},
				}},
			},
		}
		err := k8sClient.Create(ctx, policy)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("should be at least 1 chars long"))
	})

	It("should accept a policy with valid targets", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid-targets",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "with-targets",
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Targets: []coralogixv1alpha1.TCOPolicyTarget{{
						Dataset:  "myDataset",
						Priority: ptr.To("low"),
					}},
				}},
			},
		}
		Expect(k8sClient.Create(ctx, policy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, policy)).To(Succeed())
	})

	// CEL cross-field: priority must be present somewhere
	It("should reject a policy with no priority and no targets", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-priority-no-targets",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "missing-priority",
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
				}},
			},
		}
		err := k8sClient.Create(ctx, policy)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("priority is required"))
	})

	It("should reject a policy with no priority and a target that lacks its own priority", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-priority-target-no-priority",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "missing-target-priority",
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Targets: []coralogixv1alpha1.TCOPolicyTarget{{
						Dataset: "myDataset",
						// no Priority set
					}},
				}},
			},
		}
		err := k8sClient.Create(ctx, policy)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("priority is required"))
	})

	It("should accept a policy with no top-level priority when all targets have their own priority", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "targets-with-priority",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "targets-own-priority",
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Targets: []coralogixv1alpha1.TCOPolicyTarget{
						{Dataset: "ds1", Priority: ptr.To("low")},
						{Dataset: "ds2", Priority: ptr.To("medium")},
					},
				}},
			},
		}
		Expect(k8sClient.Create(ctx, policy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, policy)).To(Succeed())
	})

	It("should accept a policy with a top-level priority and targets that have no per-target priority", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "top-level-priority-with-targets",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "inherited-priority",
					Priority:   ptr.To("low"),
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Targets: []coralogixv1alpha1.TCOPolicyTarget{{
						Dataset: "myDataset",
					}},
				}},
			},
		}
		Expect(k8sClient.Create(ctx, policy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, policy)).To(Succeed())
	})

	// resource.Quantity for DailyQuotaPercentage (supports fractional values)
	It("should accept a priorityOverride with a fractional dailyQuotaPercentage", func(ctx context.Context) {
		policy := &coralogixv1alpha1.TCOLogsPolicies{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fractional-quota",
				Namespace: "default",
			},
			Spec: coralogixv1alpha1.TCOLogsPoliciesSpec{
				Policies: []coralogixv1alpha1.TCOLogsPolicy{{
					Name:       "quota-override",
					Severities: []coralogixv1alpha1.TCOPolicySeverity{"info"},
					Targets: []coralogixv1alpha1.TCOPolicyTarget{{
						Dataset:  "myDataset",
						Priority: ptr.To("low"),
						PriorityOverride: &coralogixv1alpha1.TCOPolicyPriorityOverride{
							QuotaBased: &coralogixv1alpha1.TCOPolicyQuotaBased{
								UsageTiers: []coralogixv1alpha1.TCOPolicyUsageTier{{
									DailyQuotaPercentage: resource.MustParse("60.5"),
									Priority:             "medium",
								}},
							},
						},
					}},
				}},
			},
		}
		Expect(k8sClient.Create(ctx, policy)).To(Succeed())
		Expect(k8sClient.Delete(ctx, policy)).To(Succeed())
	})
})
