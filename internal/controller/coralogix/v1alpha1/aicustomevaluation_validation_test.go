// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"k8s.io/utils/ptr"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var _ = Describe("AICustomEvaluation validation", func() {
	It("should accept a full custom evaluation and default optional fields", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("valid-ai-custom-evaluation")

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).To(Succeed())
		Expect(aiCustomEvaluation.Spec.Description).To(Equal(""))
		Expect(aiCustomEvaluation.Spec.ShouldIncludeSystemPrompt).To(Equal(ptr.To(false)))
		Expect(k8sClient.Delete(ctx, aiCustomEvaluation)).To(Succeed())
	})

	It("should accept a custom evaluation without applications or criteria", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("valid-minimal-ai-custom-evaluation")
		aiCustomEvaluation.Spec.Applications = nil
		aiCustomEvaluation.Spec.Criteria = nil

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).To(Succeed())
		Expect(k8sClient.Delete(ctx, aiCustomEvaluation)).To(Succeed())
	})

	It("should reject missing required fields", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("missing-required-ai-custom-evaluation")
		aiCustomEvaluation.Spec.Name = ""
		aiCustomEvaluation.Spec.PolicyType = ""
		aiCustomEvaluation.Spec.Instructions = ""

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})

	It("should reject invalid policy type", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("invalid-policy-ai-custom-evaluation")
		aiCustomEvaluation.Spec.PolicyType = "other"

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})

	It("should reject instructions without a supported placeholder", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("invalid-instructions-ai-custom-evaluation")
		aiCustomEvaluation.Spec.Instructions = "Score whether the response matches the policy."

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})

	It("should reject empty application selectors", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("empty-application-ai-custom-evaluation")
		aiCustomEvaluation.Spec.Applications = []coralogixv1alpha1.AICustomEvaluationApplicationSelector{
			{},
		}

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})

	It("should reject empty examples", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("empty-example-ai-custom-evaluation")
		aiCustomEvaluation.Spec.Criteria.Acceptable.Examples = []string{""}

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})

	It("should reject more than 100 total examples", func(ctx context.Context) {
		aiCustomEvaluation := validAICustomEvaluation("too-many-examples-ai-custom-evaluation")
		aiCustomEvaluation.Spec.Criteria.Acceptable.Examples = numberedAICustomEvaluationExamples("acceptable", 51)
		aiCustomEvaluation.Spec.Criteria.Prohibited.Examples = numberedAICustomEvaluationExamples("prohibited", 50)

		Expect(k8sClient.Create(ctx, aiCustomEvaluation)).ToNot(Succeed())
	})
})

func validAICustomEvaluation(name string) *coralogixv1alpha1.AICustomEvaluation {
	return &coralogixv1alpha1.AICustomEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: coralogixv1alpha1.AICustomEvaluationSpec{
			Name:         "competitor-policy",
			PolicyType:   coralogixv1alpha1.AICustomEvaluationPolicyTypeQuality,
			Instructions: "Score whether {response} mentions competitor products.",
			Applications: []coralogixv1alpha1.AICustomEvaluationApplicationSelector{
				{
					Application: "ai-center-demo",
					Subsystem:   "demo-runner",
				},
			},
			Criteria: &coralogixv1alpha1.AICustomEvaluationCriteria{
				Acceptable: &coralogixv1alpha1.AICustomEvaluationCriterion{
					Flags: "Does not mention competitor products.",
					Examples: []string{
						"User: which tool should I use?\nAssistant: Our product is a strong fit.",
					},
				},
				Prohibited: &coralogixv1alpha1.AICustomEvaluationCriterion{
					Flags: "Mentions a competitor product.",
					Examples: []string{
						"User: which tool should I use?\nAssistant: CompetitorX is a strong fit.",
					},
				},
			},
		},
	}
}

func numberedAICustomEvaluationExamples(prefix string, count int) []string {
	examples := make([]string, count)
	for i := range examples {
		examples[i] = fmt.Sprintf("%s-example-%d", prefix, i)
	}
	return examples
}
