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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var _ = Describe("AIEvaluation validation", func() {
	It("should accept a valid PII evaluation and default isEnabled", func(ctx context.Context) {
		aiEvaluation := validAIEvaluation("valid-pii")

		Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())
		Expect(aiEvaluation.Spec.IsEnabled).To(Equal(ptr.To(true)))
		Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
	})

	It("should accept a valid Allowed Topics evaluation", func(ctx context.Context) {
		aiEvaluation := validAIEvaluation("valid-allowed-topics")
		aiEvaluation.Spec.Config = coralogixv1alpha1.AIEvaluationConfig{
			AllowedTopics: &coralogixv1alpha1.AIEvaluationAllowedTopicsConfig{
				Topics: []string{"billing", "account settings"},
			},
		}

		Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())
		Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
	})

	It("should accept a valid Competition evaluation", func(ctx context.Context) {
		aiEvaluation := validAIEvaluation("valid-competition")
		aiEvaluation.Spec.Config = coralogixv1alpha1.AIEvaluationConfig{
			Competition: &coralogixv1alpha1.AIEvaluationCompetitionConfig{
				Competitors: []string{"CompetitorOne", "CompetitorTwo"},
			},
		}

		Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())
		Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
	})

	It("should accept a valid Restricted Topics evaluation", func(ctx context.Context) {
		aiEvaluation := validAIEvaluation("valid-restricted-topics")
		aiEvaluation.Spec.Config = coralogixv1alpha1.AIEvaluationConfig{
			RestrictedTopics: &coralogixv1alpha1.AIEvaluationRestrictedTopicsConfig{
				Topics: []string{"competitor mentions", "medical advice"},
			},
		}

		Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())
		Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
	})

	It("should accept a valid Toxicity evaluation", func(ctx context.Context) {
		aiEvaluation := validAIEvaluation("valid-toxicity")
		aiEvaluation.Spec.Config = coralogixv1alpha1.AIEvaluationConfig{
			Toxicity: coralogixv1alpha1.NewAIEvaluationToxicityConfig(),
		}

		Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())
		Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
	})

	It("should reject Toxicity config fields", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("toxicity-with-fields")
		config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		config["toxicity"] = map[string]interface{}{"unsupported": "value"}
		delete(config, "pii")

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too many"))
	})

	It("should accept an integer threshold quantity", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("integer-threshold")
		spec := aiEvaluation.Object["spec"].(map[string]interface{})
		spec["threshold"] = int64(1)

		Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())
		Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
	})

	It("should reject an evaluation without subsystem", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("missing-subsystem")
		spec := aiEvaluation.Object["spec"].(map[string]interface{})
		delete(spec, "subsystem")

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("subsystem"))
		Expect(err.Error()).To(ContainSubstring("Required"))
	})

	It("should reject application and subsystem values longer than 256 characters", func(ctx context.Context) {
		tooLong := strings.Repeat("a", 257)
		for _, field := range []string{"application", "subsystem"} {
			aiEvaluation := validUnstructuredAIEvaluation(fmt.Sprintf("too-long-%s", field))
			spec := aiEvaluation.Object["spec"].(map[string]interface{})
			spec[field] = tooLong

			err := k8sClient.Create(ctx, aiEvaluation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(field))
			Expect(err.Error()).To(ContainSubstring("Too long"))
		}
	})

	It("should reject unsupported targets", func(ctx context.Context) {
		aiEvaluation := validAIEvaluation("unsupported-target")
		aiEvaluation.Spec.Target = "conversation"

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unsupported value"))
		Expect(err.Error()).To(ContainSubstring("conversation"))
	})

	It("should accept syntactically valid quantity thresholds outside the semantic range", func(ctx context.Context) {
		for _, threshold := range []string{"-0.1", "1.1"} {
			aiEvaluation := validUnstructuredAIEvaluation(fmt.Sprintf("threshold-%s", strings.NewReplacer("-", "neg-", ".", "-").Replace(threshold)))
			spec := aiEvaluation.Object["spec"].(map[string]interface{})
			spec["threshold"] = threshold

			Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())
			Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
		}
	})

	It("should reject missing config variant", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("missing-config-variant")
		spec := aiEvaluation.Object["spec"].(map[string]interface{})
		spec["config"] = map[string]interface{}{}

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of the following AI evaluation configs must be set: allowedTopics, competition, pii, restrictedTopics, toxicity"))
	})

	It("should reject multiple config variants", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("multiple-config-variants")
		config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		config["toxicity"] = map[string]interface{}{}

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of the following AI evaluation configs must be set: allowedTopics, competition, pii, restrictedTopics, toxicity"))
	})

	It("should reject empty and oversized Allowed Topics topic sets", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("empty-allowed-topics")
		config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		config["allowedTopics"] = map[string]interface{}{"topics": []interface{}{}}
		delete(config, "pii")

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least 1 items"))

		aiEvaluation = validUnstructuredAIEvaluation("too-many-allowed-topics")
		config = aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		topics := make([]interface{}, 1025)
		for i := range topics {
			topics[i] = fmt.Sprintf("topic-%d", i)
		}
		config["allowedTopics"] = map[string]interface{}{"topics": topics}
		delete(config, "pii")

		err = k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too many"))
	})

	It("should reject Allowed Topics values longer than 256 characters", func(ctx context.Context) {
		for _, tt := range []struct {
			name  string
			topic string
			want  string
		}{
			{name: "empty-allowed-topic", topic: "", want: "at least 1 chars"},
			{name: "too-long-allowed-topic", topic: strings.Repeat("a", 257), want: "Too long"},
		} {
			aiEvaluation := validUnstructuredAIEvaluation(tt.name)
			config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
			config["allowedTopics"] = map[string]interface{}{"topics": []interface{}{tt.topic}}
			delete(config, "pii")

			err := k8sClient.Create(ctx, aiEvaluation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(tt.want))
		}
	})

	It("should reject empty and oversized Competition competitor sets", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("empty-competition")
		config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		config["competition"] = map[string]interface{}{"competitors": []interface{}{}}
		delete(config, "pii")

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least 1 items"))

		aiEvaluation = validUnstructuredAIEvaluation("too-many-competition")
		config = aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		competitors := make([]interface{}, 1025)
		for i := range competitors {
			competitors[i] = fmt.Sprintf("competitor-%d", i)
		}
		config["competition"] = map[string]interface{}{"competitors": competitors}
		delete(config, "pii")

		err = k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too many"))
	})

	It("should reject Competition competitor values longer than 256 characters", func(ctx context.Context) {
		for _, tt := range []struct {
			name       string
			competitor string
			want       string
		}{
			{name: "empty-competitor", competitor: "", want: "at least 1 chars"},
			{name: "too-long-competitor", competitor: strings.Repeat("a", 257), want: "Too long"},
		} {
			aiEvaluation := validUnstructuredAIEvaluation(tt.name)
			config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
			config["competition"] = map[string]interface{}{"competitors": []interface{}{tt.competitor}}
			delete(config, "pii")

			err := k8sClient.Create(ctx, aiEvaluation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(tt.want))
		}
	})

	It("should reject empty and oversized Restricted Topics topic sets", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("empty-restricted-topics")
		config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		config["restrictedTopics"] = map[string]interface{}{"topics": []interface{}{}}
		delete(config, "pii")

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least 1 items"))

		aiEvaluation = validUnstructuredAIEvaluation("too-many-restricted-topics")
		config = aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
		topics := make([]interface{}, 1025)
		for i := range topics {
			topics[i] = fmt.Sprintf("topic-%d", i)
		}
		config["restrictedTopics"] = map[string]interface{}{"topics": topics}
		delete(config, "pii")

		err = k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too many"))
	})

	It("should reject Restricted Topics values longer than 256 characters", func(ctx context.Context) {
		for _, tt := range []struct {
			name  string
			topic string
			want  string
		}{
			{name: "empty-restricted-topic", topic: "", want: "at least 1 chars"},
			{name: "too-long-restricted-topic", topic: strings.Repeat("a", 257), want: "Too long"},
		} {
			aiEvaluation := validUnstructuredAIEvaluation(tt.name)
			config := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})
			config["restrictedTopics"] = map[string]interface{}{"topics": []interface{}{tt.topic}}
			delete(config, "pii")

			err := k8sClient.Create(ctx, aiEvaluation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(tt.want))
		}
	})

	It("should reject invalid PII categories", func(ctx context.Context) {
		aiEvaluation := validAIEvaluation("invalid-category")
		aiEvaluation.Spec.Config.PII.Categories = []coralogixv1alpha1.AIEvaluationPIICategory{"PASSPORT_NUMBER"}

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Unsupported value"))
		Expect(err.Error()).To(ContainSubstring("PASSPORT_NUMBER"))
	})

	It("should reject empty and oversized PII category sets", func(ctx context.Context) {
		aiEvaluation := validUnstructuredAIEvaluation("empty-categories")
		pii := aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})["pii"].(map[string]interface{})
		pii["categories"] = []interface{}{}

		err := k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("at least 1 items"))

		aiEvaluation = validUnstructuredAIEvaluation("too-many-categories")
		pii = aiEvaluation.Object["spec"].(map[string]interface{})["config"].(map[string]interface{})["pii"].(map[string]interface{})
		categories := make([]interface{}, 1025)
		for i := range categories {
			categories[i] = fmt.Sprintf("CATEGORY_%d", i)
		}
		pii["categories"] = categories

		err = k8sClient.Create(ctx, aiEvaluation)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too many"))
	})

	It("should reject immutable field changes", func(ctx context.Context) {
		tests := []struct {
			name   string
			mutate func(*coralogixv1alpha1.AIEvaluation)
			want   string
		}{
			{
				name: "application",
				mutate: func(aiEvaluation *coralogixv1alpha1.AIEvaluation) {
					aiEvaluation.Spec.Application = "other-app"
				},
				want: "spec.application is immutable",
			},
			{
				name: "subsystem",
				mutate: func(aiEvaluation *coralogixv1alpha1.AIEvaluation) {
					aiEvaluation.Spec.Subsystem = "other-subsystem"
				},
				want: "spec.subsystem is immutable",
			},
			{
				name: "target",
				mutate: func(aiEvaluation *coralogixv1alpha1.AIEvaluation) {
					aiEvaluation.Spec.Target = coralogixv1alpha1.AIEvaluationTargetPrompt
				},
				want: "spec.target is immutable",
			},
		}

		for _, tt := range tests {
			aiEvaluation := validAIEvaluation(fmt.Sprintf("immutable-%s", tt.name))
			Expect(k8sClient.Create(ctx, aiEvaluation)).To(Succeed())

			tt.mutate(aiEvaluation)
			err := k8sClient.Update(ctx, aiEvaluation)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(tt.want))

			Expect(k8sClient.Delete(ctx, aiEvaluation)).To(Succeed())
		}
	})
})

func validAIEvaluation(name string) *coralogixv1alpha1.AIEvaluation {
	return &coralogixv1alpha1.AIEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: coralogixv1alpha1.AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      coralogixv1alpha1.AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			Config: coralogixv1alpha1.AIEvaluationConfig{
				PII: &coralogixv1alpha1.AIEvaluationPIIConfig{
					Categories: []coralogixv1alpha1.AIEvaluationPIICategory{
						coralogixv1alpha1.AIEvaluationPIICategoryEmailAddress,
						coralogixv1alpha1.AIEvaluationPIICategoryCreditCard,
					},
				},
			},
		},
	}
}

func validUnstructuredAIEvaluation(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "coralogix.com/v1alpha1",
			"kind":       "AIEvaluation",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"application": "my-chatbot",
				"subsystem":   "production",
				"target":      "response",
				"threshold":   "0.8",
				"config": map[string]interface{}{
					"pii": map[string]interface{}{
						"categories": []interface{}{"EMAIL_ADDRESS", "CREDIT_CARD"},
					},
				},
			},
		},
	}
}
