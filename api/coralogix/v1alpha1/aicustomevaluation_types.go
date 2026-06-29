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

	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AICustomEvaluationPolicyTypeQuality  = "quality"
	AICustomEvaluationPolicyTypeSecurity = "security"

	aiCustomEvaluationAcceptableScore    = "1"
	aiCustomEvaluationProhibitedScore    = "0"
	aiCustomEvaluationExamplesUpdateMask = "examples"
)

// AICustomEvaluationSpec defines the desired state of AICustomEvaluation.
type AICustomEvaluationSpec struct {
	// Display name of the custom evaluation.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Name string `json:"name"`

	// Policy type identifier.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Enum=quality;security
	PolicyType string `json:"policyType"`

	// Human-readable description. Defaults to an empty string.
	// +optional
	// +kubebuilder:default=""
	// +kubebuilder:validation:MaxLength=65536
	Description string `json:"description,omitempty"`

	// Instructions sent to the LLM evaluator. Must contain at least one of {prompt}, {response}, or {chat_history}.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1048576
	// +kubebuilder:validation:Pattern=`\{(prompt|response|chat_history)\}`
	Instructions string `json:"instructions"`

	// Whether to include the system prompt in the LLM input. Defaults to false.
	// +optional
	// +kubebuilder:default=false
	ShouldIncludeSystemPrompt *bool `json:"shouldIncludeSystemPrompt,omitempty"`

	// AI applications to link this custom evaluation to, selected by application and subsystem. Defaults to no linked applications.
	// +optional
	// +kubebuilder:validation:MaxItems=1024
	// +listType=map
	// +listMapKey=application
	// +listMapKey=subsystem
	Applications []AICustomEvaluationApplicationSelector `json:"applications,omitempty"`

	// Acceptable and prohibited criteria for this custom evaluation. Defaults to empty criteria.
	// +optional
	Criteria *AICustomEvaluationCriteria `json:"criteria,omitempty"`
}

type AICustomEvaluationApplicationSelector struct {
	// AI application name.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Application string `json:"application"`

	// AI application subsystem.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Subsystem string `json:"subsystem"`
}

// +kubebuilder:validation:XValidation:rule="(!has(self.acceptable) || !has(self.acceptable.examples) || !has(self.prohibited) || !has(self.prohibited.examples)) || size(self.acceptable.examples) + size(self.prohibited.examples) <= 100",message="criteria can include at most 100 total examples across acceptable and prohibited criteria"
type AICustomEvaluationCriteria struct {
	// Criteria and examples for acceptable responses.
	// +optional
	Acceptable *AICustomEvaluationCriterion `json:"acceptable,omitempty"`

	// Criteria and examples for prohibited responses.
	// +optional
	Prohibited *AICustomEvaluationCriterion `json:"prohibited,omitempty"`
}

type AICustomEvaluationCriterion struct {
	// Criterion flags.
	// +optional
	// +kubebuilder:default=""
	// +kubebuilder:validation:MaxLength=65536
	Flags string `json:"flags,omitempty"`

	// Example conversations for this criterion.
	// +optional
	// +kubebuilder:validation:MaxItems=100
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:MaxLength=65536
	Examples []string `json:"examples,omitempty"`
}

// AICustomEvaluationStatus defines the observed state of AICustomEvaluation.
type AICustomEvaluationStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`

	// Resolved AI application IDs linked to this custom evaluation.
	// +optional
	// +listType=set
	ApplicationIds []string `json:"applicationIds,omitempty"`

	// Resolved AI application mappings linked to this custom evaluation.
	// +optional
	// +listType=map
	// +listMapKey=id
	Applications []AICustomEvaluationApplicationStatus `json:"applications,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

type AICustomEvaluationApplicationStatus struct {
	// Resolved AI application ID.
	Id string `json:"id"`

	// AI application name.
	Application string `json:"application,omitempty"`

	// AI application subsystem.
	Subsystem string `json:"subsystem,omitempty"`
}

func (e *AICustomEvaluation) GetConditions() []metav1.Condition {
	return e.Status.Conditions
}

func (e *AICustomEvaluation) SetConditions(conditions []metav1.Condition) {
	e.Status.Conditions = conditions
}

func (e *AICustomEvaluation) GetPrintableStatus() string {
	return e.Status.PrintableStatus
}

func (e *AICustomEvaluation) SetPrintableStatus(printableStatus string) {
	e.Status.PrintableStatus = printableStatus
}

func (e *AICustomEvaluation) HasIDInStatus() bool {
	return e.Status.Id != nil && *e.Status.Id != ""
}

func (e *AICustomEvaluation) ExtractCreateAICustomEvaluationRequest(applicationIDs []string) (*aievaluations.AiEvaluationsServiceCreateCustomEvaluationRequest, error) {
	examples, safe, violates, err := e.Spec.ExtractAICustomEvaluationCriteria()
	if err != nil {
		return nil, err
	}

	return &aievaluations.AiEvaluationsServiceCreateCustomEvaluationRequest{
		ApplicationIds:            append([]string(nil), applicationIDs...),
		Description:               aievaluations.PtrString(e.Spec.Description),
		Examples:                  examples,
		Instructions:              aievaluations.PtrString(e.Spec.Instructions),
		Name:                      aievaluations.PtrString(e.Spec.Name),
		PolicyType:                aievaluations.PtrString(e.Spec.PolicyType),
		Safe:                      aievaluations.PtrString(safe),
		ShouldIncludeSystemPrompt: aievaluations.PtrBool(e.Spec.ShouldIncludeSystemPromptValue()),
		Violates:                  aievaluations.PtrString(violates),
	}, nil
}

func (e *AICustomEvaluation) ExtractUpdateAICustomEvaluationRequest() (*aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest, error) {
	examples, safe, violates, err := e.Spec.ExtractAICustomEvaluationCriteria()
	if err != nil {
		return nil, err
	}

	return &aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest{
		Description:               aievaluations.PtrString(e.Spec.Description),
		Examples:                  examples,
		Instructions:              aievaluations.PtrString(e.Spec.Instructions),
		Name:                      aievaluations.PtrString(e.Spec.Name),
		PolicyType:                aievaluations.PtrString(e.Spec.PolicyType),
		Safe:                      aievaluations.PtrString(safe),
		ShouldIncludeSystemPrompt: aievaluations.PtrBool(e.Spec.ShouldIncludeSystemPromptValue()),
		Violates:                  aievaluations.PtrString(violates),
	}, nil
}

func (e *AICustomEvaluation) ExtractUpdateAICustomEvaluationExamplesRequest() (*aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest, error) {
	examples, _, _, err := e.Spec.ExtractAICustomEvaluationCriteria()
	if err != nil {
		return nil, err
	}
	if len(examples) != 0 {
		return nil, nil
	}

	return &aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest{
		Examples:   examples,
		UpdateMask: aievaluations.PtrString(aiCustomEvaluationExamplesUpdateMask),
	}, nil
}

func (s *AICustomEvaluationSpec) ShouldIncludeSystemPromptValue() bool {
	if s.ShouldIncludeSystemPrompt == nil {
		return false
	}
	return *s.ShouldIncludeSystemPrompt
}

func (s *AICustomEvaluationSpec) ExtractAICustomEvaluationCriteria() ([]aievaluations.CustomEvaluationExample, string, string, error) {
	acceptable := AICustomEvaluationCriterion{}
	prohibited := AICustomEvaluationCriterion{}
	if s.Criteria != nil {
		if s.Criteria.Acceptable != nil {
			acceptable = *s.Criteria.Acceptable
		}
		if s.Criteria.Prohibited != nil {
			prohibited = *s.Criteria.Prohibited
		}
	}

	if len(acceptable.Examples)+len(prohibited.Examples) > 100 {
		return nil, "", "", fmt.Errorf("criteria can include at most 100 total examples across acceptable and prohibited criteria")
	}

	examples := make([]aievaluations.CustomEvaluationExample, 0, len(acceptable.Examples)+len(prohibited.Examples))
	for _, example := range acceptable.Examples {
		examples = append(examples, aievaluations.CustomEvaluationExample{
			Conversation: aievaluations.PtrString(example),
			Score:        aievaluations.PtrString(aiCustomEvaluationAcceptableScore),
		})
	}
	for _, example := range prohibited.Examples {
		examples = append(examples, aievaluations.CustomEvaluationExample{
			Conversation: aievaluations.PtrString(example),
			Score:        aievaluations.PtrString(aiCustomEvaluationProhibitedScore),
		})
	}

	return examples, acceptable.Flags, prohibited.Flags, nil
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AICustomEvaluation is the Schema for the AI custom evaluations API.
type AICustomEvaluation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AICustomEvaluationSpec   `json:"spec,omitempty"`
	Status AICustomEvaluationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AICustomEvaluationList contains a list of AICustomEvaluation.
type AICustomEvaluationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AICustomEvaluation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AICustomEvaluation{}, &AICustomEvaluationList{})
}
