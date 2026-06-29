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
// +kubebuilder:validation:XValidation:rule="(has(self.allowedTopics) ? 1 : 0) + (has(self.competition) ? 1 : 0) + (has(self.hallucinationCompleteness) ? 1 : 0) + (has(self.hallucinationContextAdherence) ? 1 : 0) + (has(self.hallucinationContextRelevance) ? 1 : 0) + (has(self.hallucinationCorrectness) ? 1 : 0) + (has(self.hallucinationTaskAdherence) ? 1 : 0) + (has(self.languageMismatch) ? 1 : 0) + (has(self.pii) ? 1 : 0) + (has(self.promptInjection) ? 1 : 0) + (has(self.restrictedTopics) ? 1 : 0) + (has(self.sexism) ? 1 : 0) + (has(self.sqlAllowedTables) ? 1 : 0) + (has(self.sqlHallucination) ? 1 : 0) + (has(self.sqlReadOnly) ? 1 : 0) + (has(self.sqlRestrictedTables) ? 1 : 0) + (has(self.toxicity) ? 1 : 0) == 1", message="Exactly one of the following AI evaluation configs must be set: allowedTopics, competition, hallucinationCompleteness, hallucinationContextAdherence, hallucinationContextRelevance, hallucinationCorrectness, hallucinationTaskAdherence, languageMismatch, pii, promptInjection, restrictedTopics, sexism, sqlAllowedTables, sqlHallucination, sqlReadOnly, sqlRestrictedTables, toxicity"
type AIEvaluationConfig struct {
	// Configuration for Allowed Topics evaluation.
	// +optional
	AllowedTopics *AIEvaluationAllowedTopicsConfig `json:"allowedTopics,omitempty"`

	// Configuration for Competition evaluation.
	// +optional
	Competition *AIEvaluationCompetitionConfig `json:"competition,omitempty"`

	// Configuration for Hallucination Completeness evaluation. Hallucination Completeness has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	HallucinationCompleteness *map[string]string `json:"hallucinationCompleteness,omitempty"`

	// Configuration for Hallucination Context Adherence evaluation. Hallucination Context Adherence has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	HallucinationContextAdherence *map[string]string `json:"hallucinationContextAdherence,omitempty"`

	// Configuration for Hallucination Context Relevance evaluation. Hallucination Context Relevance has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	HallucinationContextRelevance *map[string]string `json:"hallucinationContextRelevance,omitempty"`

	// Configuration for Hallucination Correctness evaluation. Hallucination Correctness has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	HallucinationCorrectness *map[string]string `json:"hallucinationCorrectness,omitempty"`

	// Configuration for Hallucination Task Adherence evaluation. Hallucination Task Adherence has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	HallucinationTaskAdherence *map[string]string `json:"hallucinationTaskAdherence,omitempty"`

	// Configuration for Language Mismatch evaluation. Language Mismatch has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	LanguageMismatch *map[string]string `json:"languageMismatch,omitempty"`

	// Configuration for PII evaluation.
	// +optional
	PII *AIEvaluationPIIConfig `json:"pii,omitempty"`

	// Configuration for Prompt Injection evaluation.
	// +optional
	PromptInjection *AIEvaluationPromptInjectionConfig `json:"promptInjection,omitempty"`

	// Configuration for Restricted Topics evaluation.
	// +optional
	RestrictedTopics *AIEvaluationRestrictedTopicsConfig `json:"restrictedTopics,omitempty"`

	// Configuration for Sexism evaluation. Sexism has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	Sexism *map[string]string `json:"sexism,omitempty"`

	// Configuration for SQL Allowed Tables evaluation.
	// +optional
	SQLAllowedTables *AIEvaluationSQLAllowedTablesConfig `json:"sqlAllowedTables,omitempty"`

	// Configuration for SQL Hallucination evaluation. SQL Hallucination has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	SQLHallucination *map[string]string `json:"sqlHallucination,omitempty"`

	// Configuration for SQL Read Only evaluation. SQL Read Only has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	SQLReadOnly *map[string]string `json:"sqlReadOnly,omitempty"`

	// Configuration for SQL Restricted Tables evaluation.
	// +optional
	SQLRestrictedTables *AIEvaluationSQLRestrictedTablesConfig `json:"sqlRestrictedTables,omitempty"`

	// Configuration for Toxicity evaluation. Toxicity has no nested fields and must be set to an empty object.
	// +optional
	// +kubebuilder:validation:MaxProperties=0
	Toxicity *map[string]string `json:"toxicity,omitempty"`
}

type AIEvaluationAllowedTopicsConfig struct {
	// Topics considered allowed.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1024
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:MaxLength=256
	// +listType=set
	Topics []string `json:"topics"`
}

type AIEvaluationCompetitionConfig struct {
	// Competitor names to watch for.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1024
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:MaxLength=256
	// +listType=set
	Competitors []string `json:"competitors"`
}

type AIEvaluationRestrictedTopicsConfig struct {
	// Topics that should not appear.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1024
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:MaxLength=256
	// +listType=set
	Topics []string `json:"topics"`
}

type AIEvaluationSQLAllowedTablesConfig struct {
	// SQL table names that are allowed.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1024
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:MaxLength=256
	// +listType=set
	Tables []string `json:"tables"`
}

type AIEvaluationSQLRestrictedTablesConfig struct {
	// SQL table names that are not allowed.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=1024
	// +kubebuilder:validation:items:MinLength=1
	// +kubebuilder:validation:items:MaxLength=256
	// +listType=set
	Tables []string `json:"tables"`
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

type AIEvaluationPromptInjectionConfig struct {
	// Additional context passed to the LLM evaluator.
	// +optional
	// +kubebuilder:default=""
	// +kubebuilder:validation:MaxLength=65536
	AdditionalContext string `json:"additionalContext,omitempty"`
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

	config, err := e.Spec.Config.ExtractAIEvaluationConfig()
	if err != nil {
		return nil, err
	}

	isEnabled := true
	if e.Spec.IsEnabled != nil {
		isEnabled = *e.Spec.IsEnabled
	}

	return &aievaluations.AiEvaluationsServiceCreateAiEvaluationRequest{
		Application: aievaluations.PtrString(e.Spec.Application),
		Config:      config,
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

	config, err := e.Spec.Config.ExtractAIEvaluationConfig()
	if err != nil {
		return nil, err
	}

	isEnabled := true
	if e.Spec.IsEnabled != nil {
		isEnabled = *e.Spec.IsEnabled
	}

	return &aievaluations.AiEvaluationsServiceUpdateAiEvaluationRequest{
		Config:    config,
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

func (c AIEvaluationConfig) ExtractAIEvaluationConfig() (*aievaluations.EvaluationConfig, error) {
	extractors := []func() *aievaluations.EvaluationConfig{}
	if c.AllowedTopics != nil {
		extractors = append(extractors, c.AllowedTopics.ExtractAIEvaluationConfig)
	}
	if c.Competition != nil {
		extractors = append(extractors, c.Competition.ExtractAIEvaluationConfig)
	}
	if c.HallucinationCompleteness != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationHallucinationCompletenessConfig)
	}
	if c.HallucinationContextAdherence != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationHallucinationContextAdherenceConfig)
	}
	if c.HallucinationContextRelevance != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationHallucinationContextRelevanceConfig)
	}
	if c.HallucinationCorrectness != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationHallucinationCorrectnessConfig)
	}
	if c.HallucinationTaskAdherence != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationHallucinationTaskAdherenceConfig)
	}
	if c.LanguageMismatch != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationLanguageMismatchConfig)
	}
	if c.PII != nil {
		extractors = append(extractors, c.PII.ExtractAIEvaluationConfig)
	}
	if c.PromptInjection != nil {
		extractors = append(extractors, c.PromptInjection.ExtractAIEvaluationConfig)
	}
	if c.RestrictedTopics != nil {
		extractors = append(extractors, c.RestrictedTopics.ExtractAIEvaluationConfig)
	}
	if c.Sexism != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationSexismConfig)
	}
	if c.SQLAllowedTables != nil {
		extractors = append(extractors, c.SQLAllowedTables.ExtractAIEvaluationConfig)
	}
	if c.SQLHallucination != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationSQLHallucinationConfig)
	}
	if c.SQLReadOnly != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationSQLReadOnlyConfig)
	}
	if c.SQLRestrictedTables != nil {
		extractors = append(extractors, c.SQLRestrictedTables.ExtractAIEvaluationConfig)
	}
	if c.Toxicity != nil {
		extractors = append(extractors, newOpenAPIAIEvaluationToxicityConfig)
	}

	if len(extractors) != 1 {
		return nil, fmt.Errorf("exactly one AI evaluation config must be set")
	}

	return extractors[0](), nil
}

func (c *AIEvaluationAllowedTopicsConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigAllowedTopicsAsEvaluationConfig(
		aievaluations.NewEvaluationConfigAllowedTopics(aievaluations.AllowedTopicsConfig{
			Topics: append([]string(nil), c.Topics...),
		}),
	)
	return &config
}

func (c *AIEvaluationCompetitionConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigCompetitionAsEvaluationConfig(
		aievaluations.NewEvaluationConfigCompetition(aievaluations.CompetitionConfig{
			Competitors: append([]string(nil), c.Competitors...),
		}),
	)
	return &config
}

func (c *AIEvaluationRestrictedTopicsConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigRestrictedTopicsAsEvaluationConfig(
		aievaluations.NewEvaluationConfigRestrictedTopics(aievaluations.RestrictedTopicsConfig{
			Topics: append([]string(nil), c.Topics...),
		}),
	)
	return &config
}

func (c *AIEvaluationSQLAllowedTablesConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigSqlAllowedTablesAsEvaluationConfig(
		aievaluations.NewEvaluationConfigSqlAllowedTables(aievaluations.SqlAllowedTablesConfig{
			Tables: append([]string(nil), c.Tables...),
		}),
	)
	return &config
}

func (c *AIEvaluationSQLRestrictedTablesConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigSqlRestrictedTablesAsEvaluationConfig(
		aievaluations.NewEvaluationConfigSqlRestrictedTables(aievaluations.SqlRestrictedTablesConfig{
			Tables: append([]string(nil), c.Tables...),
		}),
	)
	return &config
}

func (c *AIEvaluationPIIConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	categories := make([]aievaluations.PiiCategory, 0, len(c.Categories))
	for _, category := range c.Categories {
		categories = append(categories, schemaToOpenAPIAIEvaluationPIICategory[category])
	}

	config := aievaluations.EvaluationConfigPiiAsEvaluationConfig(
		aievaluations.NewEvaluationConfigPii(aievaluations.PiiConfig{Categories: categories}),
	)
	return &config
}

func (c *AIEvaluationPromptInjectionConfig) ExtractAIEvaluationConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigPromptInjectionAsEvaluationConfig(
		aievaluations.NewEvaluationConfigPromptInjection(aievaluations.PromptInjectionConfig{
			AdditionalContext: aievaluations.PtrString(c.AdditionalContext),
		}),
	)
	return &config
}

func newOpenAPIAIEvaluationHallucinationCompletenessConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigHallucinationCompletenessAsEvaluationConfig(
		aievaluations.NewEvaluationConfigHallucinationCompleteness(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationHallucinationContextAdherenceConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigHallucinationContextAdherenceAsEvaluationConfig(
		aievaluations.NewEvaluationConfigHallucinationContextAdherence(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationHallucinationContextRelevanceConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigHallucinationContextRelevanceAsEvaluationConfig(
		aievaluations.NewEvaluationConfigHallucinationContextRelevance(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationHallucinationCorrectnessConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigHallucinationCorrectnessAsEvaluationConfig(
		aievaluations.NewEvaluationConfigHallucinationCorrectness(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationHallucinationTaskAdherenceConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigHallucinationTaskAdherenceAsEvaluationConfig(
		aievaluations.NewEvaluationConfigHallucinationTaskAdherence(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationLanguageMismatchConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigLanguageMismatchAsEvaluationConfig(
		aievaluations.NewEvaluationConfigLanguageMismatch(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationSexismConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigSexismAsEvaluationConfig(
		aievaluations.NewEvaluationConfigSexism(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationSQLHallucinationConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigSqlHallucinationAsEvaluationConfig(
		aievaluations.NewEvaluationConfigSqlHallucination(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationSQLReadOnlyConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigSqlReadOnlyAsEvaluationConfig(
		aievaluations.NewEvaluationConfigSqlReadOnly(map[string]interface{}{}),
	)
	return &config
}

func newOpenAPIAIEvaluationToxicityConfig() *aievaluations.EvaluationConfig {
	config := aievaluations.EvaluationConfigToxicityAsEvaluationConfig(
		aievaluations.NewEvaluationConfigToxicity(map[string]interface{}{}),
	)
	return &config
}

// NewAIEvaluationHallucinationCompletenessConfig returns the empty object required to enable hallucination completeness evaluation.
func NewAIEvaluationHallucinationCompletenessConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationHallucinationContextAdherenceConfig returns the empty object required to enable hallucination context adherence evaluation.
func NewAIEvaluationHallucinationContextAdherenceConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationHallucinationContextRelevanceConfig returns the empty object required to enable hallucination context relevance evaluation.
func NewAIEvaluationHallucinationContextRelevanceConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationHallucinationCorrectnessConfig returns the empty object required to enable hallucination correctness evaluation.
func NewAIEvaluationHallucinationCorrectnessConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationHallucinationTaskAdherenceConfig returns the empty object required to enable hallucination task adherence evaluation.
func NewAIEvaluationHallucinationTaskAdherenceConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationLanguageMismatchConfig returns the empty object required to enable language mismatch evaluation.
func NewAIEvaluationLanguageMismatchConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationSexismConfig returns the empty object required to enable sexism evaluation.
func NewAIEvaluationSexismConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationSQLHallucinationConfig returns the empty object required to enable SQL hallucination evaluation.
func NewAIEvaluationSQLHallucinationConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationSQLReadOnlyConfig returns the empty object required to enable SQL read only evaluation.
func NewAIEvaluationSQLReadOnlyConfig() *map[string]string {
	return &map[string]string{}
}

// NewAIEvaluationToxicityConfig returns the empty object required to enable toxicity evaluation.
func NewAIEvaluationToxicityConfig() *map[string]string {
	return &map[string]string{}
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
