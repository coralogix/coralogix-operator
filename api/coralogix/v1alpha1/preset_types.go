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

	presets "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/presets_service"
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

var (
	schemaToOpenApiPresetConnectorType = map[string]presets.ConnectorType{
		"slack":        presets.CONNECTORTYPE_SLACK,
		"genericHttps": presets.CONNECTORTYPE_GENERIC_HTTPS,
		"pagerDuty":    presets.CONNECTORTYPE_PAGERDUTY,
	}
	schemaToOpenApiPresetsEntityType = map[string]presets.NotificationCenterEntityType{
		"alerts": presets.NOTIFICATIONCENTERENTITYTYPE_ALERTS,
	}
)

func (p *Preset) ExtractPreset() (*presets.Preset1, error) {
	preset := &presets.Preset1{
		Name:        p.Spec.Name,
		Description: presets.PtrString(p.Spec.Description),
		ParentId:    p.Spec.ParentId,
	}

	connectorType, ok := schemaToOpenApiPresetConnectorType[p.Spec.ConnectorType]
	if !ok {
		return nil, fmt.Errorf("invalid connector type %s", p.Spec.ConnectorType)
	}
	preset.ConnectorType = connectorType.Ptr()

	entityType, ok := schemaToOpenApiPresetsEntityType[p.Spec.EntityType]
	if !ok {
		return nil, fmt.Errorf("invalid entity type %s", p.Spec.EntityType)
	}

	preset.EntityType = entityType
	preset.ConfigOverrides = ExtractConfigOverrides(p.Spec.ConfigOverrides)
	return preset, nil
}

func ExtractConfigOverrides(overrides []ConfigOverride) []presets.ConfigOverrides {
	var result []presets.ConfigOverrides
	for _, override := range overrides {
		configOverride := presets.ConfigOverrides{
			PayloadType: override.PayloadType,
		}

		configOverride.MessageConfig = ExtractMessageConfig(override.MessageConfig)

		if override.ConditionType.MatchEntityType != nil {
			configOverride.ConditionType = &presets.NotificationCenterConditionType{
				NotificationCenterConditionTypeMatchEntityType: &presets.NotificationCenterConditionTypeMatchEntityType{
					MatchEntityType: map[string]interface{}{},
				},
			}
		} else if override.ConditionType.MatchEntityTypeAndSubType != nil {
			configOverride.ConditionType = &presets.NotificationCenterConditionType{
				NotificationCenterConditionTypeMatchEntityTypeAndSubType: &presets.NotificationCenterConditionTypeMatchEntityTypeAndSubType{
					MatchEntityTypeAndSubType: &presets.MatchEntityTypeAndSubTypeCondition{
						EntitySubType: presets.PtrString(override.ConditionType.MatchEntityTypeAndSubType.EntitySubType),
					},
				},
			}
		}

		result = append(result, configOverride)
	}

	return result
}

func ExtractMessageConfig(messageConfig MessageConfig) *presets.MessageConfig {
	var fields []presets.NotificationCenterMessageConfigField
	for _, field := range messageConfig.Fields {
		fields = append(fields, presets.NotificationCenterMessageConfigField{
			FieldName: field.FieldName,
			Template:  field.Template,
		})
	}

	return &presets.MessageConfig{
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
