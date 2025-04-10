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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// PresetSpec defines the desired state of Preset.
type PresetSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	ParentId string `json:"parentId"`

	// +kubebuilder:validation:Enum=slack;genericHttps;pagerDuty
	ConnectorType string `json:"connectorType"`

	// +kubebuilder:validation:Enum=alerts
	EntityType string `json:"entityType"`

	ConfigOverrides []ConfigOverride `json:"configOverrides,omitempty"`
}

type ConfigOverride struct {
	ConditionType ConditionType `json:"conditionType"`

	PayloadType string `json:"payloadType"`

	MessageConfig MessageConfig `json:"messageConfig"`
}

// ConditionType defines the condition type for the config override.
// One of matchEntityType or matchEntityTypeAndSubType must be set.
type ConditionType struct {
	// +optional
	MatchEntityType *MatchEntityType `json:"matchEntityType,omitempty"`

	// +optional
	MatchEntityTypeAndSubType *MatchEntityTypeAndSubType `json:"matchEntityTypeAndSubType,omitempty"`
}

type MatchEntityType struct{}

type MatchEntityTypeAndSubType struct {
	EntitySubType string `json:"entitySubType"`
}

type MessageConfig struct {
	Fields []MessageConfigField `json:"fields"`
}

type MessageConfigField struct {
	FieldName string `json:"fieldName"`

	Template string `json:"template"`
}

// PresetStatus defines the observed state of Preset.
type PresetStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (p *Preset) GetConditions() []metav1.Condition {
	return p.Status.Conditions
}

func (p *Preset) SetConditions(conditions []metav1.Condition) {
	p.Status.Conditions = conditions
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
		Parent: &cxsdk.Preset{
			Id: ptr.To(p.Spec.ParentId),
		},
	}

	if connectorType, ok := schemaToProtoConnectorType[p.Spec.ConnectorType]; ok {
		preset.ConnectorType = connectorType
	} else {
		return nil, fmt.Errorf("invalid connector type %s", p.Spec.ConnectorType)
	}

	if entityType, ok := schemaToProtoEntityType[p.Spec.EntityType]; ok {
		preset.EntityType = entityType
	} else {
		return nil, fmt.Errorf("invalid entity type %s", p.Spec.EntityType)
	}

	preset.ConfigOverrides = ExtractConfigOverrides(p.Spec.ConfigOverrides)
	return preset, nil
}

func ExtractConfigOverrides(overrides []ConfigOverride) []*cxsdk.ConfigOverrides {
	var result []*cxsdk.ConfigOverrides
	for _, override := range overrides {
		messageConfig := ExtractMessageConfig(override.MessageConfig)
		configOverride := &cxsdk.ConfigOverrides{
			PayloadType:   override.PayloadType,
			MessageConfig: messageConfig,
		}

		if override.ConditionType.MatchEntityType != nil {
			configOverride.ConditionType = &cxsdk.ConditionType{
				MatchEntityType: &cxsdk.MatchEntityType{},
			}
		} else if override.ConditionType.MatchEntityTypeAndSubType != nil {
			configOverride.ConditionType = &cxsdk.ConditionType{
				MatchEntityTypeAndSubType: &cxsdk.MatchEntityTypeAndSubType{
					EntitySubType: override.ConditionType.MatchEntityTypeAndSubType.EntitySubType,
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

// Preset is the Schema for the presets API.
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
