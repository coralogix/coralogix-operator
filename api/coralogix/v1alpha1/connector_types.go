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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// ConnectorSpec defines the desired state of Connector.
type ConnectorSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	ConnectorType *ConnectorType `json:"connectorType"`
}

type ConnectorType struct {
	// +optional
	GenericHttps *ConnectorGenericHttps `json:"genericHttps,omitempty"`

	// +optional
	Slack *ConnectorSlack `json:"slack,omitempty"`
}

type ConnectorGenericHttps struct {
	Config *ConnectorGenericHttpsConfig `json:"config"`
}

type ConnectorGenericHttpsConfig struct {
	Url string `json:"url"`

	// +optional
	// +kubebuilder:validation:Enum=get;post;put
	Method *string `json:"method,omitempty"`

	// +optional
	AdditionalHeaders *string `json:"additionalHeaders,omitempty"`

	// +optional
	AdditionalBodyFields *string `json:"additionalBodyFields,omitempty"`
}

type ConnectorSlack struct {
	General *ConnectorSlackGeneral `json:"general"`

	// +optional
	Overrides []ConnectorSlackOverride `json:"overrides,omitempty"`
}

type ConnectorSlackGeneral struct {
	RawConfig *ConnectorSlackConfig `json:"rawConfig"`

	StructuredConfig *ConnectorSlackConfig `json:"structuredConfig"`
}

type ConnectorSlackConfig struct {
	IntegrationId string `json:"integrationId"`

	FallbackChannel string `json:"fallbackChannel"`

	// +optional
	Channel *string `json:"channel,omitempty"`
}

type ConnectorSlackOverride struct {
	RawConfig *ConnectorSlackConfigOverride `json:"rawConfig"`

	StructuredConfig *ConnectorSlackConfigOverride `json:"structuredConfig"`

	EntityType string `json:"entityType"`
}

type ConnectorSlackConfigOverride struct {
	Channel string `json:"channel"`
}

// ConnectorStatus defines the observed state of Connector.
type ConnectorStatus struct {
	Id *string `json:"id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Connector is the Schema for the connectors API.
type Connector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConnectorSpec   `json:"spec,omitempty"`
	Status ConnectorStatus `json:"status,omitempty"`
}

func (s *ConnectorSpec) ExtractCreateConnectorRequest() *cxsdk.CreateConnectorRequest {
	return &cxsdk.CreateConnectorRequest{
		Connector: s.ExtractConnector(),
	}
}

func (s *ConnectorSpec) ExtractReplaceConnectorRequest(id *string) *cxsdk.ReplaceConnectorRequest {
	connector := s.ExtractConnector()
	connector.Id = id

	return &cxsdk.ReplaceConnectorRequest{
		Connector: connector,
	}
}

func (s *ConnectorSpec) ExtractConnector() *cxsdk.Connector {
	connector := &cxsdk.Connector{
		Name:        s.Name,
		Description: s.Description,
	}

	if s.ConnectorType.GenericHttps != nil {
		connector.Type = cxsdk.ConnectorTypeGenericHTTPS
		connector.ConnectorConfigs = ExtractGenericHttpsConnectorConfigs(s.ConnectorType.GenericHttps.Config)
	}

	if s.ConnectorType.Slack != nil {
		connector.Type = cxsdk.ConnectorTypeSlack
		connector.ConnectorConfigs = ExtractSlackConnectorGeneralConfigs(s.ConnectorType.Slack.General)
		connector.ConfigOverrides = ExtractSlackConnectorOverrides(s.ConnectorType.Slack.Overrides)
	}

	return connector
}

const (
	FieldNameUrl                  = "url"
	FieldNameMethod               = "method"
	FieldNameAdditionalHeaders    = "additionalHeaders"
	FieldNameAdditionalBodyFields = "additionalBodyFields"
	FieldNameIntegrationId        = "integrationId"
	FieldNameChannel              = "channel"
	FieldNameFallbackChannel      = "fallbackChannel"
)

func ExtractGenericHttpsConnectorConfigs(config *ConnectorGenericHttpsConfig) []*cxsdk.ConnectorConfig {
	var connectorConfigFields []*cxsdk.ConnectorConfigField
	connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
		FieldName: FieldNameUrl,
		Template:  config.Url,
	})
	if config.Method != nil {
		connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
			FieldName: FieldNameMethod,
			Template:  strings.ToUpper(*config.Method),
		})
	}
	if config.AdditionalHeaders != nil {
		connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
			FieldName: FieldNameAdditionalHeaders,
			Template:  *config.AdditionalHeaders,
		})
	}
	if config.AdditionalBodyFields != nil {
		connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
			FieldName: FieldNameAdditionalBodyFields,
			Template:  *config.AdditionalBodyFields,
		})
	}

	return []*cxsdk.ConnectorConfig{
		{
			OutputSchemaId: DefaultOutputSchemaId,
			Fields:         connectorConfigFields,
		},
	}
}

func ExtractSlackConnectorGeneralConfigs(slackGeneral *ConnectorSlackGeneral) []*cxsdk.ConnectorConfig {
	var connectorConfigs []*cxsdk.ConnectorConfig

	if slackGeneral.RawConfig != nil {
		var connectorConfigFields []*cxsdk.ConnectorConfigField
		connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
			FieldName: FieldNameIntegrationId,
			Template:  slackGeneral.RawConfig.IntegrationId,
		})
		connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
			FieldName: FieldNameFallbackChannel,
			Template:  slackGeneral.RawConfig.FallbackChannel,
		})
		if slackGeneral.RawConfig.Channel != nil {
			connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
				FieldName: FieldNameChannel,
				Template:  *slackGeneral.RawConfig.Channel,
			})
		}
		connectorConfigs = append(connectorConfigs, &cxsdk.ConnectorConfig{
			OutputSchemaId: RawOutputSchemaId,
			Fields:         connectorConfigFields,
		})
	}

	if slackGeneral.StructuredConfig != nil {
		var connectorConfigFields []*cxsdk.ConnectorConfigField
		connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
			FieldName: FieldNameIntegrationId,
			Template:  slackGeneral.StructuredConfig.IntegrationId,
		})
		connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
			FieldName: FieldNameFallbackChannel,
			Template:  slackGeneral.StructuredConfig.FallbackChannel,
		})
		if slackGeneral.RawConfig.Channel != nil {
			connectorConfigFields = append(connectorConfigFields, &cxsdk.ConnectorConfigField{
				FieldName: FieldNameChannel,
				Template:  *slackGeneral.StructuredConfig.Channel,
			})
		}
		connectorConfigs = append(connectorConfigs, &cxsdk.ConnectorConfig{
			OutputSchemaId: StructuredOutputSchemaId,
			Fields:         connectorConfigFields,
		})
	}

	return connectorConfigs
}

func ExtractSlackConnectorOverrides(overrides []ConnectorSlackOverride) []*cxsdk.EntityTypeConfigOverrides {
	var connectorConfigOverrides []*cxsdk.EntityTypeConfigOverrides
	for _, override := range overrides {
		connectorConfigOverrides = append(connectorConfigOverrides, &cxsdk.EntityTypeConfigOverrides{
			EntityType:       override.EntityType,
			ConnectorConfigs: ExtractSlackConnectorOverrideConfigs(override),
		})
	}
	return connectorConfigOverrides
}

func ExtractSlackConnectorOverrideConfigs(override ConnectorSlackOverride) []*cxsdk.ConnectorConfig {
	var connectorConfigs []*cxsdk.ConnectorConfig

	if override.RawConfig != nil {
		connectorConfigFields := []*cxsdk.ConnectorConfigField{
			{
				FieldName: FieldNameChannel,
				Template:  override.RawConfig.Channel,
			},
		}
		connectorConfigs = append(connectorConfigs, &cxsdk.ConnectorConfig{
			OutputSchemaId: RawOutputSchemaId,
			Fields:         connectorConfigFields,
		})

	}

	if override.StructuredConfig != nil {
		connectorConfigFields := []*cxsdk.ConnectorConfigField{
			{
				FieldName: FieldNameChannel,
				Template:  override.StructuredConfig.Channel,
			},
		}
		connectorConfigs = append(connectorConfigs, &cxsdk.ConnectorConfig{
			OutputSchemaId: StructuredOutputSchemaId,
			Fields:         connectorConfigFields,
		})
	}

	return connectorConfigs
}

// +kubebuilder:object:root=true

// ConnectorList contains a list of Connector.
type ConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Connector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Connector{}, &ConnectorList{})
}
