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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AIEvaluationTargetPrompt   = "prompt"
	AIEvaluationTargetResponse = "response"

	AIEvaluationPIICategoryPhoneNumber  AIEvaluationPIICategory = "PHONE_NUMBER"
	AIEvaluationPIICategoryEmailAddress AIEvaluationPIICategory = "EMAIL_ADDRESS"
	AIEvaluationPIICategoryCreditCard   AIEvaluationPIICategory = "CREDIT_CARD"
	AIEvaluationPIICategoryIBANCode     AIEvaluationPIICategory = "IBAN_CODE"
	AIEvaluationPIICategoryUSSSN        AIEvaluationPIICategory = "US_SSN"
)

var (
	schemaToOpenAPIAIEvaluationTarget = map[string]aievaluations.EvaluationTarget{
		AIEvaluationTargetPrompt:   aievaluations.EVALUATIONTARGET_PROMPT,
		AIEvaluationTargetResponse: aievaluations.EVALUATIONTARGET_RESPONSE,
	}
	schemaToOpenAPIAIEvaluationPIICategory = map[AIEvaluationPIICategory]aievaluations.PiiCategory{
		AIEvaluationPIICategoryPhoneNumber:  aievaluations.PIICATEGORY_PHONE_NUMBER,
		AIEvaluationPIICategoryEmailAddress: aievaluations.PIICATEGORY_EMAIL_ADDRESS,
		AIEvaluationPIICategoryCreditCard:   aievaluations.PIICATEGORY_CREDIT_CARD,
		AIEvaluationPIICategoryIBANCode:     aievaluations.PIICATEGORY_IBAN_CODE,
		AIEvaluationPIICategoryUSSSN:        aievaluations.PIICATEGORY_US_SSN,
	}
)

var maxAIEvaluationThreshold = resource.MustParse("1")

// AIEvaluationSpec defines the desired state of AIEvaluation.
type AIEvaluationSpec struct {
	// Name of the AI application this evaluation belongs to.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.application is immutable"
	Application string `json:"application"`

	// Subsystem within the application.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.subsystem is immutable"
	Subsystem string `json:"subsystem"`

	// Target span content the evaluation runs against.
	// +kubebuilder:validation:Enum=prompt;response
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.target is immutable"
	Target string `json:"target"`

	// Score threshold. Must be between 0.0 and 1.0 inclusive.
	// Fractional values must be supplied as quoted quantities, for example "0.8".
	Threshold resource.Quantity `json:"threshold"`

	// Whether the evaluation is active.
	// +optional
	// +kubebuilder:default=true
	IsEnabled *bool `json:"isEnabled,omitempty"`

	// AI evaluation configuration.
	Config AIEvaluationConfig `json:"config"`
}

// AIEvaluationConfig configures the AI evaluation type.
// +kubebuilder:validation:XValidation:rule="(has(self.pii) ? 1 : 0) == 1", message="Exactly one of the following AI evaluation configs must be set: pii"
type AIEvaluationConfig struct {
	// Configuration for PII evaluation.
	// +optional
	PII *AIEvaluationPIIConfig `json:"pii,omitempty"`
}

// +kubebuilder:validation:Enum=PHONE_NUMBER;EMAIL_ADDRESS;CREDIT_CARD;IBAN_CODE;US_SSN
type AIEvaluationPIICategory string

type AIEvaluationPIIConfig struct {
	// PII categories to detect.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1024
	// +listType=set
	Categories []AIEvaluationPIICategory `json:"categories"`
}

// AIEvaluationStatus defines the observed state of AIEvaluation.
type AIEvaluationStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (e *AIEvaluation) GetConditions() []metav1.Condition {
	return e.Status.Conditions
}

func (e *AIEvaluation) SetConditions(conditions []metav1.Condition) {
	e.Status.Conditions = conditions
}

func (e *AIEvaluation) GetPrintableStatus() string {
	return e.Status.PrintableStatus
}

func (e *AIEvaluation) SetPrintableStatus(printableStatus string) {
	e.Status.PrintableStatus = printableStatus
}

func (e *AIEvaluation) HasIDInStatus() bool {
	return e.Status.Id != nil && *e.Status.Id != ""
}

func (e *AIEvaluation) ExtractCreateAIEvaluationRequest() (*aievaluations.AiEvaluationsServiceCreateAiEvaluationRequest, error) {
	if err := e.Spec.ValidateAIEvaluationThreshold(); err != nil {
		return nil, err
	}

	isEnabled := true
	if e.Spec.IsEnabled != nil {
		isEnabled = *e.Spec.IsEnabled
	}

	return &aievaluations.AiEvaluationsServiceCreateAiEvaluationRequest{
		Application: aievaluations.PtrString(e.Spec.Application),
		Config:      e.Spec.Config.ExtractAIEvaluationConfig(),
		IsEnabled:   aievaluations.PtrBool(isEnabled),
		Subsystem:   aievaluations.PtrString(e.Spec.Subsystem),
		Target:      schemaToOpenAPIAIEvaluationTarget[e.Spec.Target].Ptr(),
		Threshold:   aievaluations.PtrFloat64(e.Spec.Threshold.AsApproximateFloat64()),
	}, nil
}

func (e *AIEvaluation) ExtractUpdateAIEvaluationRequest() (*aievaluations.AiEvaluationsServiceUpdateAiEvaluationRequest, error) {
	if err := e.Spec.ValidateAIEvaluationThreshold(); err != nil {
		return nil, err
	}

	isEnabled := true
	if e.Spec.IsEnabled != nil {
		isEnabled = *e.Spec.IsEnabled
	}

	return &aievaluations.AiEvaluationsServiceUpdateAiEvaluationRequest{
		Config:    e.Spec.Config.ExtractAIEvaluationConfig(),
		IsEnabled: aievaluations.PtrBool(isEnabled),
		Threshold: aievaluations.PtrFloat64(e.Spec.Threshold.AsApproximateFloat64()),
	}, nil
}

func (s *AIEvaluationSpec) ValidateAIEvaluationThreshold() error {
	if s.Threshold.Sign() < 0 || s.Threshold.Cmp(maxAIEvaluationThreshold) > 0 {
		return fmt.Errorf("spec.threshold must be between 0 and 1 inclusive")
	}
	return nil
}

func (c AIEvaluationConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	categories := make([]aievaluations.PiiCategory, 0, len(c.PII.Categories))
	for _, category := range c.PII.Categories {
		categories = append(categories, schemaToOpenAPIAIEvaluationPIICategory[category])
	}

	config := aievaluations.EvaluationConfigPiiAsEvaluationConfig(
		aievaluations.NewEvaluationConfigPii(aievaluations.PiiConfig{Categories: categories}),
	)
	return &config
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AIEvaluation is the Schema for the AI evaluations API.
type AIEvaluation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AIEvaluationSpec   `json:"spec,omitempty"`
	Status AIEvaluationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AIEvaluationList contains a list of AIEvaluation.
type AIEvaluationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AIEvaluation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AIEvaluation{}, &AIEvaluationList{})
}
