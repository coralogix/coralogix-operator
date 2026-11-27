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

// The spec fields of AICustomEvaluation that the API cannot state. The hooks of the controller
// read them: applications are sent as AI application ids, and criteria as the fields safe,
// violates, and examples.

// The policy types that spec.policyType accepts.
const (
	AICustomEvaluationPolicyTypeQuality  = "quality"
	AICustomEvaluationPolicyTypeSecurity = "security"
)

// AI applications to link this custom evaluation to, selected by application and subsystem. Defaults to no linked applications.
// +kubebuilder:validation:MaxItems=1024
// +listType=map
// +listMapKey=application
// +listMapKey=subsystem
type AICustomEvaluationApplications []AICustomEvaluationApplicationSelector

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

// The hooks send acceptable and prohibited examples as one list, which the API caps at 100 items.

// Acceptable and prohibited criteria for this custom evaluation. Defaults to empty criteria.
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
	// +kubebuilder:validation:MaxLength=65536
	Flags *string `json:"flags,omitempty"`

	// Example conversations for this criterion.
	// +optional
	// +kubebuilder:validation:MaxItems=100
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:MaxLength=65536
	Examples []string `json:"examples,omitempty"`
}
