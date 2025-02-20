// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// PresetSpec defines the desired state of Preset.
type PresetSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	// +kubebuilder:validation:Enum=alerts
	EntityType string `json:"entityType"`

	ParentId string `json:"parentId"`

	ConnectorType *PresetConnectorType `json:"connectorType"`
}

type PresetConnectorType struct {
	// +optional
	GenericHttps *PresetGenericHttps `json:"genericHttps,omitempty"`

	// +optional
	Slack *PresetSlack `json:"slack,omitempty"`
}

type PresetGenericHttps struct {
	// +optional
	General *PresetGenericHttpsGeneral `json:"general,omitempty"`

	// +optional
	Overrides []PresetGenericHttpsOverride `json:"overrides,omitempty"`
}

type PresetGenericHttpsGeneral struct {
	Fields *PresetGenericHttpsFields `json:"fields"`
}

type PresetGenericHttpsOverride struct {
	Fields *PresetGenericHttpsFields `json:"fields"`

	EntitySubType string `json:"entitySubType"`
}

type PresetGenericHttpsFields struct {
	// +optional
	Headers *string `json:"headers,omitempty"`

	// +optional
	Body *string `json:"body,omitempty"`
}

type PresetSlack struct {
	// +optional
	General *PresetSlackGeneral `json:"general,omitempty"`

	// +optional
	Overrides []PresetSlackOverride `json:"overrides,omitempty"`
}

type PresetSlackGeneral struct {
	// +optional
	RawFields *PresetSlackRawFields `json:"rawFields,omitempty"`

	// +optional
	StructuredFields *PresetSlackStructuredFields `json:"structuredFields,omitempty"`
}

type PresetSlackOverride struct {
	// +optional
	RawFields *PresetSlackRawFields `json:"rawFields,omitempty"`

	// +optional
	StructuredFields *PresetSlackStructuredFields `json:"structuredFields,omitempty"`

	EntitySubType string `json:"entitySubType"`
}

type PresetSlackRawFields struct {
	Payload string `json:"payload"`
}

type PresetSlackStructuredFields struct {
	// +optional
	Title *string `json:"title,omitempty"`

	// +optional
	Description *string `json:"description,omitempty"`

	// +optional
	Footer *string `json:"footer,omitempty"`
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Preset is the Schema for the presets API.
// NOTE: This CRD exposes a new feature and may have breaking changes in future releases.
type Preset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PresetSpec   `json:"spec,omitempty"`
	Status PresetStatus `json:"status,omitempty"`
}

func (s *PresetSpec) ExtractCreatePresetRequest() *cxsdk.CreateCustomPresetRequest {
	return &cxsdk.CreateCustomPresetRequest{
		Preset: s.ExtractPreset(),
	}
}

func (s *PresetSpec) ExtractReplacePresetRequest(id *string) *cxsdk.ReplaceCustomPresetRequest {
	preset := s.ExtractPreset()
	preset.Id = id

	return &cxsdk.ReplaceCustomPresetRequest{
		Preset: preset,
	}
}

func (s *PresetSpec) ExtractPreset() *cxsdk.Preset {
	preset := &cxsdk.Preset{
		Name:        s.Name,
		Description: s.Description,
		EntityType:  s.EntityType,
		Parent: &cxsdk.Preset{
			UserFacingId: ptr.To(s.ParentId),
		},
	}
	if s.ConnectorType.GenericHttps != nil {
		preset.ConnectorType = cxsdk.ConnectorTypeGenericHTTPS
		preset.ConfigOverrides = s.ExtractGenericHttpsConfigOverrides()

	} else if s.ConnectorType.Slack != nil {
		preset.ConnectorType = cxsdk.ConnectorTypeSlack
		preset.ConfigOverrides = s.ExtractSlackConfigOverrides()

	}
	return preset
}

const (
	DefaultOutputSchemaId    = "default"
	RawOutputSchemaId        = "raw"
	StructuredOutputSchemaId = "structured"
	FieldNameHeaders         = "headers"
	FieldNameBody            = "body"
	FieldNamePayload         = "payload"
	FieldNameTitle           = "title"
	FieldNameDescription     = "description"
	FieldNameFooter          = "footer"
)

func (s *PresetSpec) ExtractGenericHttpsConfigOverrides() []*cxsdk.ConfigOverrides {
	var configOverrides []*cxsdk.ConfigOverrides
	if s.ConnectorType.GenericHttps.General != nil {
		configOverrides = append(configOverrides, getGenericHttpsConfigOverride(
			s.ConnectorType.GenericHttps.General.Fields,
			s.EntityType,
			"",
		))
	}

	if s.ConnectorType.GenericHttps.Overrides != nil {
		for _, override := range s.ConnectorType.GenericHttps.Overrides {
			configOverrides = append(configOverrides, getGenericHttpsConfigOverride(
				override.Fields,
				s.EntityType,
				override.EntitySubType,
			))
		}
	}

	return configOverrides
}

func getGenericHttpsConfigOverride(
	fields *PresetGenericHttpsFields,
	entityType, entitySubType string,
) *cxsdk.ConfigOverrides {
	return &cxsdk.ConfigOverrides{
		OutputSchemaId: DefaultOutputSchemaId,
		MessageConfig: &cxsdk.MessageConfig{
			Fields: getGenericHttpsMessageConfigFields(fields),
		},
		ConditionType: getConditionType(entityType, entitySubType),
	}
}

func getGenericHttpsMessageConfigFields(fields *PresetGenericHttpsFields) []*cxsdk.MessageConfigField {
	var msgConfigFields []*cxsdk.MessageConfigField
	if fields.Headers != nil {
		msgConfigFields = append(msgConfigFields, &cxsdk.MessageConfigField{
			FieldName: FieldNameHeaders,
			Template:  *fields.Headers,
		})
	}
	if fields.Body != nil {
		msgConfigFields = append(msgConfigFields, &cxsdk.MessageConfigField{
			FieldName: FieldNameBody,
			Template:  *fields.Body,
		})
	}

	return msgConfigFields
}

func (s *PresetSpec) ExtractSlackConfigOverrides() []*cxsdk.ConfigOverrides {
	var configOverrides []*cxsdk.ConfigOverrides
	if s.ConnectorType.Slack.General != nil {
		configOverrides = append(configOverrides, getSlackConfigOverride(
			s.ConnectorType.Slack.General.RawFields,
			s.ConnectorType.Slack.General.StructuredFields,
			s.EntityType,
			"",
		))
	}

	if s.ConnectorType.Slack.Overrides != nil {
		for _, override := range s.ConnectorType.Slack.Overrides {
			configOverrides = append(configOverrides, getSlackConfigOverride(
				override.RawFields,
				override.StructuredFields,
				s.EntityType,
				override.EntitySubType,
			))
		}
	}

	return configOverrides
}

func getSlackConfigOverride(
	rawFields *PresetSlackRawFields,
	structuredFields *PresetSlackStructuredFields,
	entityType, entitySubType string,
) *cxsdk.ConfigOverrides {
	var outputSchemaId string
	var messageFields []*cxsdk.MessageConfigField

	if rawFields != nil {
		outputSchemaId = RawOutputSchemaId
		messageFields = getSlackMessageConfigRawFields(rawFields)
	} else if structuredFields != nil {
		outputSchemaId = StructuredOutputSchemaId
		messageFields = getSlackMessageConfigStructuredFields(structuredFields)
	}

	return &cxsdk.ConfigOverrides{
		OutputSchemaId: outputSchemaId,
		MessageConfig: &cxsdk.MessageConfig{
			Fields: messageFields,
		},
		ConditionType: getConditionType(entityType, entitySubType),
	}
}

func getSlackMessageConfigRawFields(fields *PresetSlackRawFields) []*cxsdk.MessageConfigField {
	return []*cxsdk.MessageConfigField{
		{
			FieldName: FieldNamePayload,
			Template:  fields.Payload,
		},
	}
}

func getSlackMessageConfigStructuredFields(fields *PresetSlackStructuredFields) []*cxsdk.MessageConfigField {
	var msgConfigFields []*cxsdk.MessageConfigField
	if fields.Title != nil {
		msgConfigFields = append(msgConfigFields, &cxsdk.MessageConfigField{
			FieldName: FieldNameTitle,
			Template:  *fields.Title,
		})
	}
	if fields.Description != nil {
		msgConfigFields = append(msgConfigFields, &cxsdk.MessageConfigField{
			FieldName: FieldNameDescription,
			Template:  *fields.Description,
		})
	}
	if fields.Footer != nil {
		msgConfigFields = append(msgConfigFields, &cxsdk.MessageConfigField{
			FieldName: FieldNameFooter,
			Template:  *fields.Footer,
		})
	}

	return msgConfigFields
}

func getConditionType(entityType, entitySubType string) *cxsdk.ConditionType {
	if entitySubType == "" {
		return &cxsdk.ConditionType{
			Condition: &cxsdk.ConditionTypeMatchEntityType{
				MatchEntityType: &cxsdk.MatchEntityTypeCondition{
					EntityType: entityType,
				},
			},
		}
	}

	return &cxsdk.ConditionType{
		Condition: &cxsdk.ConditionTypeMatchEntityTypeAndSubType{
			MatchEntityTypeAndSubType: &cxsdk.MatchEntityTypeAndSubTypeCondition{
				EntityType:    entityType,
				EntitySubType: entitySubType,
			},
		},
	}
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
