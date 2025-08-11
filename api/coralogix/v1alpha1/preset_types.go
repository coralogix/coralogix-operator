/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PresetSpec defines the desired state of Preset.
type PresetSpec struct {
	// Name is the name of the preset.
	Name string `json:"name"`

	// Description is the description of the preset.
	Description string `json:"description"`

	// ParentId is the ID of the parent preset. For example, "preset_system_slack_alerts_basic".
	// +optional
	ParentId *string `json:"parentId,omitempty"`

	// ConnectorType is the type of the connector. Can be one of slack, genericHttps, or pagerDuty.
	// +kubebuilder:validation:Enum=slack;genericHttps;pagerDuty
	ConnectorType string `json:"connectorType"`

	// EntityType is the entity type for the preset. Should equal "alerts".
	// +kubebuilder:validation:Enum=alerts
	EntityType string `json:"entityType"`

	// ConfigOverrides are the entity type configs, allowing entity type templating.
	ConfigOverrides []ConfigOverride `json:"configOverrides,omitempty"`
}

type ConfigOverride struct {
	// ConditionType is the condition type for the config override.
	ConditionType ConditionType `json:"conditionType"`

	// PayloadType is the payload type for the config override.
	// +optional
	PayloadType *string `json:"payloadType,omitempty"`

	// MessageConfig is the message config for the config override.
	MessageConfig MessageConfig `json:"messageConfig"`
}

// ConditionType defines the condition type for the config override.
// One of matchEntityType or matchEntityTypeAndSubType must be set.
// +kubebuilder:validation:XValidation:rule="has(self.matchEntityType) != has(self.matchEntityTypeAndSubType)", message="exactly one of matchEntityType or matchEntityTypeAndSubType must be set"
type ConditionType struct {
	// MatchEntityType is used for matching entity types.
	// +optional
	MatchEntityType *MatchEntityType `json:"matchEntityType,omitempty"`

	// MatchEntityTypeAndSubType is used for matching entity subtypes.
	// +optional
	MatchEntityTypeAndSubType *MatchEntityTypeAndSubType `json:"matchEntityTypeAndSubType,omitempty"`
}

type MatchEntityType struct{}

type MatchEntityTypeAndSubType struct {
	// EntitySubType is the entity subtype for the config override. For example, "logsImmediateTriggered".
	EntitySubType string `json:"entitySubType"`
}

type MessageConfig struct {
	// Fields are the fields of the message config.
	Fields []MessageConfigField `json:"fields"`
}

type MessageConfigField struct {
	// FieldName is the name of the field. e.g. "title" for slack.
	FieldName string `json:"fieldName"`

	// Template is the template for the field.
	Template string `json:"template"`
}

// PresetStatus defines the observed state of Preset.
type PresetStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (p *Preset) GetConditions() []metav1.Condition {
	return p.Status.Conditions
}

func (p *Preset) SetConditions(conditions []metav1.Condition) {
	p.Status.Conditions = conditions
}

func (p *Preset) GetPrintableStatus() string {
	return p.Status.PrintableStatus
}

func (p *Preset) SetPrintableStatus(printableStatus string) {
	p.Status.PrintableStatus = printableStatus
}

func (p *Preset) HasIDInStatus() bool {
	return p.Status.Id != nil && *p.Status.Id != ""
}

func (p *Preset) ExtractCreateCustomPresetRequest() (*cxsdk.CreateCustomPresetRequest, error) {
	preset, err := p.extractPreset()
	if err != nil {
		return nil, fmt.Errorf("failed to extract preset: %w", err)
	}

	return &cxsdk.CreateCustomPresetRequest{
		Preset: preset,
	}, nil
}

func (p *Preset) ExtractUpdateCustomPresetRequest() (*cxsdk.ReplaceCustomPresetRequest, error) {
	preset, err := p.extractPreset()
	if err != nil {
		return nil, fmt.Errorf("failed to extract preset: %w", err)
	}

	preset.Id = p.Status.Id
	return &cxsdk.ReplaceCustomPresetRequest{
		Preset: preset,
	}, nil
}

func (p *Preset) extractPreset() (*cxsdk.Preset, error) {
	preset := &cxsdk.Preset{
		Name:        p.Spec.Name,
		Description: p.Spec.Description,
		ParentId:    p.Spec.ParentId,
	}

	connectorType, ok := schemaToProtoConnectorType[p.Spec.ConnectorType]
	if !ok {
		return nil, fmt.Errorf("invalid connector type %s", p.Spec.ConnectorType)
	}
	preset.ConnectorType = connectorType

	entityType, ok := schemaToProtoEntityType[p.Spec.EntityType]
	if !ok {
		return nil, fmt.Errorf("invalid entity type %s", p.Spec.EntityType)
	}

	preset.EntityType = entityType
	preset.ConfigOverrides = ExtractConfigOverrides(p.Spec.ConfigOverrides)
	return preset, nil
}

func ExtractConfigOverrides(overrides []ConfigOverride) []*cxsdk.ConfigOverrides {
	var result []*cxsdk.ConfigOverrides
	for _, override := range overrides {
		configOverride := &cxsdk.ConfigOverrides{
			PayloadType: override.PayloadType,
		}

		configOverride.MessageConfig = ExtractMessageConfig(override.MessageConfig)

		if override.ConditionType.MatchEntityType != nil {
			configOverride.ConditionType = &cxsdk.ConditionType{
				Condition: &cxsdk.ConditionTypeMatchEntityType{},
			}
		} else if override.ConditionType.MatchEntityTypeAndSubType != nil {
			configOverride.ConditionType = &cxsdk.ConditionType{
				Condition: &cxsdk.ConditionTypeMatchEntityTypeAndSubType{
					MatchEntityTypeAndSubType: &cxsdk.MatchEntityTypeAndSubTypeCondition{
						EntitySubType: override.ConditionType.MatchEntityTypeAndSubType.EntitySubType,
					},
				},
			}
		}

		result = append(result, configOverride)
	}

	return result
}

func ExtractMessageConfig(messageConfig MessageConfig) *cxsdk.MessageConfig {
	var fields []*cxsdk.MessageConfigField
	for _, field := range messageConfig.Fields {
		fields = append(fields, &cxsdk.MessageConfigField{
			FieldName: field.FieldName,
			Template:  field.Template,
		})
	}

	return &cxsdk.MessageConfig{
		Fields: fields,
	}
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Preset is the Schema for the presets API.
// NOTE: This CRD exposes a new feature and may have breaking changes in future releases.
// See also https://coralogix.com/docs/user-guides/notification-center/presets/introduction/
//
// **Added in v0.4.0**
type Preset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PresetSpec   `json:"spec,omitempty"`
	Status PresetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PresetList contains a list of Preset.
type PresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Preset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Preset{}, &PresetList{})
}
