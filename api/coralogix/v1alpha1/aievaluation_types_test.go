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
	"testing"

	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestAIEvaluationExtractRequestsCoverTerraformPIIScenario(t *testing.T) {
	aiEvaluation := AIEvaluation{
		Spec: AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			IsEnabled:   ptr.To(true),
			Config: AIEvaluationConfig{PII: &AIEvaluationPIIConfig{
				Categories: []AIEvaluationPIICategory{AIEvaluationPIICategoryEmailAddress, AIEvaluationPIICategoryCreditCard},
			}},
		},
	}

	createRequest, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "my-chatbot", createRequest.GetApplication())
	require.Equal(t, "production", createRequest.GetSubsystem())
	require.Equal(t, aievaluations.EVALUATIONTARGET_RESPONSE, createRequest.GetTarget())
	require.InDelta(t, 0.8, createRequest.GetThreshold(), 0.000001)
	require.True(t, createRequest.GetIsEnabled())
	assertPII(t, createRequest.GetConfig(), aievaluations.PIICATEGORY_EMAIL_ADDRESS, aievaluations.PIICATEGORY_CREDIT_CARD)

	aiEvaluation.Status = AIEvaluationStatus{
		Id: ptr.To("evaluation-id"),
	}
	aiEvaluation.Spec.Threshold = resource.MustParse("0.9")
	aiEvaluation.Spec.IsEnabled = ptr.To(false)
	aiEvaluation.Spec.Config = AIEvaluationConfig{PII: &AIEvaluationPIIConfig{
		Categories: []AIEvaluationPIICategory{AIEvaluationPIICategoryPhoneNumber, AIEvaluationPIICategoryUSSSN},
	}}

	updateRequest, err := aiEvaluation.ExtractUpdateAIEvaluationRequest()
	require.NoError(t, err)
	require.InDelta(t, 0.9, updateRequest.GetThreshold(), 0.000001)
	require.False(t, updateRequest.GetIsEnabled())
	assertPII(t, updateRequest.GetConfig(), aievaluations.PIICATEGORY_PHONE_NUMBER, aievaluations.PIICATEGORY_US_SSN)
}

func TestAIEvaluationExtractRequestsCoverTerraformAllowedTopicsScenario(t *testing.T) {
	aiEvaluation := AIEvaluation{
		Spec: AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			IsEnabled:   ptr.To(true),
			Config: AIEvaluationConfig{AllowedTopics: &AIEvaluationAllowedTopicsConfig{
				Topics: []string{"billing", "account settings"},
			}},
		},
	}

	createRequest, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "my-chatbot", createRequest.GetApplication())
	require.Equal(t, "production", createRequest.GetSubsystem())
	require.Equal(t, aievaluations.EVALUATIONTARGET_RESPONSE, createRequest.GetTarget())
	require.InDelta(t, 0.8, createRequest.GetThreshold(), 0.000001)
	require.True(t, createRequest.GetIsEnabled())
	assertAllowedTopics(t, createRequest.GetConfig(), "billing", "account settings")

	aiEvaluation.Status = AIEvaluationStatus{
		Id: ptr.To("evaluation-id"),
	}
	aiEvaluation.Spec.Threshold = resource.MustParse("0.9")
	aiEvaluation.Spec.IsEnabled = ptr.To(false)
	aiEvaluation.Spec.Config = AIEvaluationConfig{AllowedTopics: &AIEvaluationAllowedTopicsConfig{
		Topics: []string{"observability", "incident response"},
	}}

	updateRequest, err := aiEvaluation.ExtractUpdateAIEvaluationRequest()
	require.NoError(t, err)
	require.InDelta(t, 0.9, updateRequest.GetThreshold(), 0.000001)
	require.False(t, updateRequest.GetIsEnabled())
	assertAllowedTopics(t, updateRequest.GetConfig(), "observability", "incident response")
}

func TestAIEvaluationExtractRequestsCoverTerraformCompetitionScenario(t *testing.T) {
	aiEvaluation := AIEvaluation{
		Spec: AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			IsEnabled:   ptr.To(true),
			Config: AIEvaluationConfig{Competition: &AIEvaluationCompetitionConfig{
				Competitors: []string{"CompetitorOne", "CompetitorTwo"},
			}},
		},
	}

	createRequest, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "my-chatbot", createRequest.GetApplication())
	require.Equal(t, "production", createRequest.GetSubsystem())
	require.Equal(t, aievaluations.EVALUATIONTARGET_RESPONSE, createRequest.GetTarget())
	require.InDelta(t, 0.8, createRequest.GetThreshold(), 0.000001)
	require.True(t, createRequest.GetIsEnabled())
	assertCompetition(t, createRequest.GetConfig(), "CompetitorOne", "CompetitorTwo")

	aiEvaluation.Status = AIEvaluationStatus{
		Id: ptr.To("evaluation-id"),
	}
	aiEvaluation.Spec.Threshold = resource.MustParse("0.9")
	aiEvaluation.Spec.IsEnabled = ptr.To(false)
	aiEvaluation.Spec.Config = AIEvaluationConfig{Competition: &AIEvaluationCompetitionConfig{
		Competitors: []string{"CompetitorThree", "CompetitorFour"},
	}}

	updateRequest, err := aiEvaluation.ExtractUpdateAIEvaluationRequest()
	require.NoError(t, err)
	require.InDelta(t, 0.9, updateRequest.GetThreshold(), 0.000001)
	require.False(t, updateRequest.GetIsEnabled())
	assertCompetition(t, updateRequest.GetConfig(), "CompetitorThree", "CompetitorFour")
}

func TestAIEvaluationExtractRequestsCoverTerraformRestrictedTopicsScenario(t *testing.T) {
	aiEvaluation := AIEvaluation{
		Spec: AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			IsEnabled:   ptr.To(true),
			Config: AIEvaluationConfig{RestrictedTopics: &AIEvaluationRestrictedTopicsConfig{
				Topics: []string{"competitor mentions", "medical advice"},
			}},
		},
	}

	createRequest, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "my-chatbot", createRequest.GetApplication())
	require.Equal(t, "production", createRequest.GetSubsystem())
	require.Equal(t, aievaluations.EVALUATIONTARGET_RESPONSE, createRequest.GetTarget())
	require.InDelta(t, 0.8, createRequest.GetThreshold(), 0.000001)
	require.True(t, createRequest.GetIsEnabled())
	assertRestrictedTopics(t, createRequest.GetConfig(), "competitor mentions", "medical advice")

	aiEvaluation.Status = AIEvaluationStatus{
		Id: ptr.To("evaluation-id"),
	}
	aiEvaluation.Spec.Threshold = resource.MustParse("0.9")
	aiEvaluation.Spec.IsEnabled = ptr.To(false)
	aiEvaluation.Spec.Config = AIEvaluationConfig{RestrictedTopics: &AIEvaluationRestrictedTopicsConfig{
		Topics: []string{"pricing promises", "legal advice"},
	}}

	updateRequest, err := aiEvaluation.ExtractUpdateAIEvaluationRequest()
	require.NoError(t, err)
	require.InDelta(t, 0.9, updateRequest.GetThreshold(), 0.000001)
	require.False(t, updateRequest.GetIsEnabled())
	assertRestrictedTopics(t, updateRequest.GetConfig(), "pricing promises", "legal advice")
}

func TestAIEvaluationExtractRequestsCoverTerraformSexismScenario(t *testing.T) {
	aiEvaluation := AIEvaluation{
		Spec: AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			IsEnabled:   ptr.To(true),
			Config: AIEvaluationConfig{
				Sexism: NewAIEvaluationSexismConfig(),
			},
		},
	}

	createRequest, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "my-chatbot", createRequest.GetApplication())
	require.Equal(t, "production", createRequest.GetSubsystem())
	require.Equal(t, aievaluations.EVALUATIONTARGET_RESPONSE, createRequest.GetTarget())
	require.InDelta(t, 0.8, createRequest.GetThreshold(), 0.000001)
	require.True(t, createRequest.GetIsEnabled())
	assertSexism(t, createRequest.GetConfig())

	aiEvaluation.Status = AIEvaluationStatus{
		Id: ptr.To("evaluation-id"),
	}
	aiEvaluation.Spec.Threshold = resource.MustParse("0.9")
	aiEvaluation.Spec.IsEnabled = ptr.To(false)

	updateRequest, err := aiEvaluation.ExtractUpdateAIEvaluationRequest()
	require.NoError(t, err)
	require.InDelta(t, 0.9, updateRequest.GetThreshold(), 0.000001)
	require.False(t, updateRequest.GetIsEnabled())
	assertSexism(t, updateRequest.GetConfig())
}

func TestAIEvaluationExtractRequestsCoverTerraformToxicityScenario(t *testing.T) {
	aiEvaluation := AIEvaluation{
		Spec: AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			IsEnabled:   ptr.To(true),
			Config: AIEvaluationConfig{
				Toxicity: NewAIEvaluationToxicityConfig(),
			},
		},
	}

	createRequest, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "my-chatbot", createRequest.GetApplication())
	require.Equal(t, "production", createRequest.GetSubsystem())
	require.Equal(t, aievaluations.EVALUATIONTARGET_RESPONSE, createRequest.GetTarget())
	require.InDelta(t, 0.8, createRequest.GetThreshold(), 0.000001)
	require.True(t, createRequest.GetIsEnabled())
	assertToxicity(t, createRequest.GetConfig())

	aiEvaluation.Status = AIEvaluationStatus{
		Id: ptr.To("evaluation-id"),
	}
	aiEvaluation.Spec.Threshold = resource.MustParse("0.9")
	aiEvaluation.Spec.IsEnabled = ptr.To(false)

	updateRequest, err := aiEvaluation.ExtractUpdateAIEvaluationRequest()
	require.NoError(t, err)
	require.InDelta(t, 0.9, updateRequest.GetThreshold(), 0.000001)
	require.False(t, updateRequest.GetIsEnabled())
	assertToxicity(t, updateRequest.GetConfig())
}

func TestAIEvaluationExtractRequestsRejectThresholdOutsideSupportedRange(t *testing.T) {
	for _, threshold := range []string{"-0.1", "1.1"} {
		aiEvaluation := AIEvaluation{
			Spec: AIEvaluationSpec{
				Application: "my-chatbot",
				Subsystem:   "production",
				Target:      AIEvaluationTargetResponse,
				Threshold:   resource.MustParse(threshold),
				Config: AIEvaluationConfig{PII: &AIEvaluationPIIConfig{
					Categories: []AIEvaluationPIICategory{AIEvaluationPIICategoryEmailAddress},
				}},
			},
		}

		_, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
		require.ErrorContains(t, err, "spec.threshold must be between 0 and 1 inclusive")

		_, err = aiEvaluation.ExtractUpdateAIEvaluationRequest()
		require.ErrorContains(t, err, "spec.threshold must be between 0 and 1 inclusive")
	}
}

func TestAIEvaluationExtractRequestsRejectEmptyConfig(t *testing.T) {
	aiEvaluation := AIEvaluation{
		Spec: AIEvaluationSpec{
			Application: "my-chatbot",
			Subsystem:   "production",
			Target:      AIEvaluationTargetResponse,
			Threshold:   resource.MustParse("0.8"),
			Config:      AIEvaluationConfig{},
		},
	}

	_, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	require.ErrorContains(t, err, "exactly one AI evaluation config must be set")

	_, err = aiEvaluation.ExtractUpdateAIEvaluationRequest()
	require.ErrorContains(t, err, "exactly one AI evaluation config must be set")
}

func assertPII(t *testing.T, config aievaluations.EvaluationConfig, values ...aievaluations.PiiCategory) {
	actual, ok := config.GetActualInstance().(*aievaluations.EvaluationConfigPii)
	require.True(t, ok)
	pii := actual.GetPii()
	require.ElementsMatch(t, values, (&pii).GetCategories())
}

func assertAllowedTopics(t *testing.T, config aievaluations.EvaluationConfig, values ...string) {
	actual, ok := config.GetActualInstance().(*aievaluations.EvaluationConfigAllowedTopics)
	require.True(t, ok)
	allowedTopics := actual.GetAllowedTopics()
	require.ElementsMatch(t, values, allowedTopics.GetTopics())
}

func assertCompetition(t *testing.T, config aievaluations.EvaluationConfig, values ...string) {
	actual, ok := config.GetActualInstance().(*aievaluations.EvaluationConfigCompetition)
	require.True(t, ok)
	competition := actual.GetCompetition()
	require.ElementsMatch(t, values, competition.GetCompetitors())
}

func assertRestrictedTopics(t *testing.T, config aievaluations.EvaluationConfig, values ...string) {
	actual, ok := config.GetActualInstance().(*aievaluations.EvaluationConfigRestrictedTopics)
	require.True(t, ok)
	restrictedTopics := actual.GetRestrictedTopics()
	require.ElementsMatch(t, values, restrictedTopics.GetTopics())
}

func assertSexism(t *testing.T, config aievaluations.EvaluationConfig) {
	actual, ok := config.GetActualInstance().(*aievaluations.EvaluationConfigSexism)
	require.True(t, ok)
	require.Empty(t, actual.GetSexism())
}

func assertToxicity(t *testing.T, config aievaluations.EvaluationConfig) {
	actual, ok := config.GetActualInstance().(*aievaluations.EvaluationConfigToxicity)
	require.True(t, ok)
	require.Empty(t, actual.GetToxicity())
}
