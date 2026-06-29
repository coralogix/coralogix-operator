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
	"fmt"
	"testing"

	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestAICustomEvaluationExtractRequestsCoverTerraformScenario(t *testing.T) {
	require.Equal(t, "1", aiCustomEvaluationAcceptableScore)
	require.Equal(t, "0", aiCustomEvaluationProhibitedScore)

	aiCustomEvaluation := AICustomEvaluation{
		Spec: AICustomEvaluationSpec{
			Name:                      "competitor-policy",
			PolicyType:                AICustomEvaluationPolicyTypeQuality,
			Description:               "Flags competitor references in assistant responses.",
			Instructions:              "Score whether {response} mentions competitor products.\nTreat each assistant answer independently.",
			ShouldIncludeSystemPrompt: ptr.To(false),
			Applications: []AICustomEvaluationApplicationSelector{
				{
					Application: "ai-center-demo",
					Subsystem:   "demo-runner",
				},
			},
			Criteria: &AICustomEvaluationCriteria{
				Acceptable: &AICustomEvaluationCriterion{
					Flags: "Does not mention competitor products.\nAnswer stays focused on our product.",
					Examples: []string{
						"User: which tool should I use?\nAssistant: Our product is a strong fit.",
					},
				},
				Prohibited: &AICustomEvaluationCriterion{
					Flags: "Mentions a competitor product.\nNames another vendor as the recommended option.",
					Examples: []string{
						"User: which tool should I use?\nAssistant: CompetitorX is a strong fit.",
					},
				},
			},
		},
	}

	createRequest, err := aiCustomEvaluation.ExtractCreateAICustomEvaluationRequest([]string{"application-id"})
	require.NoError(t, err)
	require.Equal(t, []string{"application-id"}, createRequest.GetApplicationIds())
	require.Equal(t, "competitor-policy", createRequest.GetName())
	require.Equal(t, AICustomEvaluationPolicyTypeQuality, createRequest.GetPolicyType())
	require.Equal(t, "Flags competitor references in assistant responses.", createRequest.GetDescription())
	require.Equal(t, "Score whether {response} mentions competitor products.\nTreat each assistant answer independently.", createRequest.GetInstructions())
	require.False(t, createRequest.GetShouldIncludeSystemPrompt())
	require.Equal(t, "Does not mention competitor products.\nAnswer stays focused on our product.", createRequest.GetSafe())
	require.Equal(t, "Mentions a competitor product.\nNames another vendor as the recommended option.", createRequest.GetViolates())
	requireCustomEvaluationExamples(t, createRequest.GetExamples(),
		expectedAICustomEvaluationExample{
			conversation: "User: which tool should I use?\nAssistant: Our product is a strong fit.",
			score:        aiCustomEvaluationAcceptableScore,
		},
		expectedAICustomEvaluationExample{
			conversation: "User: which tool should I use?\nAssistant: CompetitorX is a strong fit.",
			score:        aiCustomEvaluationProhibitedScore,
		},
	)

	aiCustomEvaluation.Spec.Name = "competitor-policy-updated"
	aiCustomEvaluation.Spec.PolicyType = AICustomEvaluationPolicyTypeSecurity
	aiCustomEvaluation.Spec.Description = "Flags responses that recommend competitor tools."
	aiCustomEvaluation.Spec.Instructions = "Score whether {response} recommends competitor products.\nOnly evaluate the final assistant response."
	aiCustomEvaluation.Spec.ShouldIncludeSystemPrompt = ptr.To(true)
	aiCustomEvaluation.Spec.Criteria = &AICustomEvaluationCriteria{
		Acceptable: &AICustomEvaluationCriterion{
			Flags: "Does not recommend competitor products.\nMentions only our product or neutral guidance.",
			Examples: []string{
				"User: what should I buy?\nAssistant: Our product covers that workflow.",
			},
		},
		Prohibited: &AICustomEvaluationCriterion{
			Flags: "Recommends a competitor product.\nNames a competitor as the best choice.",
			Examples: []string{
				"User: what should I buy?\nAssistant: You should buy CompetitorY.",
			},
		},
	}

	updateRequest, err := aiCustomEvaluation.ExtractUpdateAICustomEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "competitor-policy-updated", updateRequest.GetName())
	require.Equal(t, AICustomEvaluationPolicyTypeSecurity, updateRequest.GetPolicyType())
	require.Equal(t, "Flags responses that recommend competitor tools.", updateRequest.GetDescription())
	require.Equal(t, "Score whether {response} recommends competitor products.\nOnly evaluate the final assistant response.", updateRequest.GetInstructions())
	require.True(t, updateRequest.GetShouldIncludeSystemPrompt())
	require.Equal(t, "Does not recommend competitor products.\nMentions only our product or neutral guidance.", updateRequest.GetSafe())
	require.Equal(t, "Recommends a competitor product.\nNames a competitor as the best choice.", updateRequest.GetViolates())
	require.Empty(t, updateRequest.GetUpdateMask())
	requireCustomEvaluationExamples(t, updateRequest.GetExamples(),
		expectedAICustomEvaluationExample{
			conversation: "User: what should I buy?\nAssistant: Our product covers that workflow.",
			score:        aiCustomEvaluationAcceptableScore,
		},
		expectedAICustomEvaluationExample{
			conversation: "User: what should I buy?\nAssistant: You should buy CompetitorY.",
			score:        aiCustomEvaluationProhibitedScore,
		},
	)
}

func TestAICustomEvaluationExtractRequestsCoverTerraformMinimalScenario(t *testing.T) {
	aiCustomEvaluation := AICustomEvaluation{
		Spec: AICustomEvaluationSpec{
			Name:         "minimal-policy",
			PolicyType:   AICustomEvaluationPolicyTypeQuality,
			Instructions: "Score whether {response} matches the policy.",
		},
	}

	createRequest, err := aiCustomEvaluation.ExtractCreateAICustomEvaluationRequest(nil)
	require.NoError(t, err)
	require.Empty(t, createRequest.GetApplicationIds())
	require.Equal(t, "minimal-policy", createRequest.GetName())
	require.Equal(t, AICustomEvaluationPolicyTypeQuality, createRequest.GetPolicyType())
	require.Empty(t, createRequest.GetDescription())
	require.Equal(t, "Score whether {response} matches the policy.", createRequest.GetInstructions())
	require.False(t, createRequest.GetShouldIncludeSystemPrompt())
	require.Empty(t, createRequest.GetSafe())
	require.Empty(t, createRequest.GetViolates())
	require.Empty(t, createRequest.GetExamples())

	updateRequest, err := aiCustomEvaluation.ExtractUpdateAICustomEvaluationRequest()
	require.NoError(t, err)
	require.Equal(t, "minimal-policy", updateRequest.GetName())
	require.Equal(t, AICustomEvaluationPolicyTypeQuality, updateRequest.GetPolicyType())
	require.Empty(t, updateRequest.GetDescription())
	require.Equal(t, "Score whether {response} matches the policy.", updateRequest.GetInstructions())
	require.False(t, updateRequest.GetShouldIncludeSystemPrompt())
	require.Empty(t, updateRequest.GetSafe())
	require.Empty(t, updateRequest.GetViolates())
	require.Empty(t, updateRequest.GetExamples())
	require.Empty(t, updateRequest.GetUpdateMask())
}

func TestAICustomEvaluationExtractUpdateRequestReplacesExamplesWhenOneCriterionIsEmpty(t *testing.T) {
	tests := []struct {
		name                    string
		criteria                *AICustomEvaluationCriteria
		expectedAcceptableFlags string
		expectedProhibitedFlags string
		expectedCustomExamples  []expectedAICustomEvaluationExample
	}{
		{
			name: "acceptable has examples and prohibited is empty",
			criteria: &AICustomEvaluationCriteria{
				Acceptable: &AICustomEvaluationCriterion{
					Flags: "Does not mention competitor products.",
					Examples: []string{
						"User: which tool should I use?\nAssistant: Our product is a strong fit for that workflow.",
					},
				},
				Prohibited: &AICustomEvaluationCriterion{
					Flags:    "Mentions a competitor product.",
					Examples: nil,
				},
			},
			expectedAcceptableFlags: "Does not mention competitor products.",
			expectedProhibitedFlags: "Mentions a competitor product.",
			expectedCustomExamples: []expectedAICustomEvaluationExample{
				{
					conversation: "User: which tool should I use?\nAssistant: Our product is a strong fit for that workflow.",
					score:        aiCustomEvaluationAcceptableScore,
				},
			},
		},
		{
			name: "prohibited has examples and acceptable is empty",
			criteria: &AICustomEvaluationCriteria{
				Acceptable: &AICustomEvaluationCriterion{
					Flags:    "Does not mention competitor products.",
					Examples: nil,
				},
				Prohibited: &AICustomEvaluationCriterion{
					Flags: "Mentions a competitor product.",
					Examples: []string{
						"User: which tool should I use?\nAssistant: CompetitorX is a strong fit.",
					},
				},
			},
			expectedAcceptableFlags: "Does not mention competitor products.",
			expectedProhibitedFlags: "Mentions a competitor product.",
			expectedCustomExamples: []expectedAICustomEvaluationExample{
				{
					conversation: "User: which tool should I use?\nAssistant: CompetitorX is a strong fit.",
					score:        aiCustomEvaluationProhibitedScore,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aiCustomEvaluation := AICustomEvaluation{
				Spec: AICustomEvaluationSpec{
					Name:         "one-sided-examples-policy",
					PolicyType:   AICustomEvaluationPolicyTypeSecurity,
					Description:  "Flags responses that mention or recommend competitor products.",
					Instructions: "Evaluate whether {response} mentions or recommends competitor products.",
					Criteria:     tt.criteria,
				},
			}

			updateRequest, err := aiCustomEvaluation.ExtractUpdateAICustomEvaluationRequest()
			require.NoError(t, err)
			require.Empty(t, updateRequest.GetUpdateMask())
			require.Equal(t, tt.expectedAcceptableFlags, updateRequest.GetSafe())
			require.Equal(t, tt.expectedProhibitedFlags, updateRequest.GetViolates())
			requireCustomEvaluationExamples(t, updateRequest.GetExamples(), tt.expectedCustomExamples...)

			examplesUpdateRequest, err := aiCustomEvaluation.ExtractUpdateAICustomEvaluationExamplesRequest()
			require.NoError(t, err)
			require.Nil(t, examplesUpdateRequest)
		})
	}
}

func TestAICustomEvaluationExtractUpdateExamplesRequestClearsEmptyExamples(t *testing.T) {
	aiCustomEvaluation := AICustomEvaluation{
		Spec: AICustomEvaluationSpec{
			Name:         "empty-examples-policy",
			PolicyType:   AICustomEvaluationPolicyTypeSecurity,
			Description:  "Flags responses that mention or recommend competitor products.",
			Instructions: "Evaluate whether {response} mentions or recommends competitor products.",
			Criteria: &AICustomEvaluationCriteria{
				Acceptable: &AICustomEvaluationCriterion{
					Flags:    "Does not mention competitor products.",
					Examples: nil,
				},
				Prohibited: &AICustomEvaluationCriterion{
					Flags:    "Mentions a competitor product.",
					Examples: nil,
				},
			},
		},
	}

	examplesUpdateRequest, err := aiCustomEvaluation.ExtractUpdateAICustomEvaluationExamplesRequest()
	require.NoError(t, err)
	require.NotNil(t, examplesUpdateRequest)
	require.Equal(t, aiCustomEvaluationExamplesUpdateMask, examplesUpdateRequest.GetUpdateMask())
	require.Empty(t, examplesUpdateRequest.GetName())
	require.Empty(t, examplesUpdateRequest.GetDescription())
	require.Empty(t, examplesUpdateRequest.GetInstructions())
	require.Empty(t, examplesUpdateRequest.GetSafe())
	require.Empty(t, examplesUpdateRequest.GetViolates())
	require.Empty(t, examplesUpdateRequest.GetPolicyType())
	require.False(t, examplesUpdateRequest.GetShouldIncludeSystemPrompt())
	require.NotNil(t, examplesUpdateRequest.Examples)
	require.Empty(t, examplesUpdateRequest.GetExamples())
}

func TestAICustomEvaluationExtractRequestsRejectTooManyExamples(t *testing.T) {
	examples := make([]string, 101)
	for i := range examples {
		examples[i] = fmt.Sprintf("conversation-%d", i)
	}

	aiCustomEvaluation := AICustomEvaluation{
		Spec: AICustomEvaluationSpec{
			Name:         "too-many-examples",
			PolicyType:   AICustomEvaluationPolicyTypeQuality,
			Instructions: "Score whether {response} matches the policy.",
			Criteria: &AICustomEvaluationCriteria{
				Acceptable: &AICustomEvaluationCriterion{
					Examples: examples,
				},
			},
		},
	}

	_, err := aiCustomEvaluation.ExtractCreateAICustomEvaluationRequest(nil)
	require.ErrorContains(t, err, "at most 100 total examples")

	_, err = aiCustomEvaluation.ExtractUpdateAICustomEvaluationRequest()
	require.ErrorContains(t, err, "at most 100 total examples")
}

type expectedAICustomEvaluationExample struct {
	conversation string
	score        string
}

func requireCustomEvaluationExamples(t *testing.T, actual []aievaluations.CustomEvaluationExample, expected ...expectedAICustomEvaluationExample) {
	t.Helper()

	require.Len(t, actual, len(expected))
	for i := range expected {
		require.Equal(t, expected[i].conversation, actual[i].GetConversation())
		require.Equal(t, expected[i].score, actual[i].GetScore())
	}
}
