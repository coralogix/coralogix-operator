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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// ConnectorSpec defines the desired state of Connector.
type ConnectorSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	// +kubebuilder:validation:Enum=slack;genericHttps;pagerduty
	Type string `json:"type"`

	ConnectorConfig ConnectorConfig `json:"connectorConfig"`

	// +optional
	ConfigOverrides []EntityTypeConfigOverrides `json:"configOverrides,omitempty"`
}

type ConnectorConfig struct {
	Fields []ConnectorConfigField `json:"fields"`
}

type ConnectorConfigField struct {
	FieldName string `json:"fieldName"`

	Value string `json:"value"`
}

type EntityTypeConfigOverrides struct {
	// +kubebuilder:validation:Enum=alerts
	EntityType string `json:"entityType"`

	Fields []TemplatedConnectorConfigField `json:"fields,omitempty"`
}

type TemplatedConnectorConfigField struct {
	FieldName string `json:"fieldName"`

	Template string `json:"template"`
}

// ConnectorStatus defines the observed state of Connector.
type ConnectorStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (c *Connector) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *Connector) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

var (
	schemaToProtoConnectorType = map[string]cxsdk.ConnectorType{
		"slack":        cxsdk.ConnectorTypeSlack,
		"genericHttps": cxsdk.ConnectorTypeGenericHTTPS,
		"pagerduty":    cxsdk.ConnectorTypePagerDuty,
	}
	schemaToProtoEntityType = map[string]cxsdk.EntityType{
		"alerts": cxsdk.EntityTypeAlerts,
	}
)

func (c *Connector) ExtractCreateConnectorRequest() (*cxsdk.CreateConnectorRequest, error) {
	connector, err := c.ExtractConnector()
	if err != nil {
		return nil, fmt.Errorf("failed to extract connector: %w", err)
	}
	return &cxsdk.CreateConnectorRequest{
		Connector: connector,
	}, nil
}

func (c *Connector) ExtractUpdateConnectorRequest() (*cxsdk.ReplaceConnectorRequest, error) {
	connector, err := c.ExtractConnector()
	if err != nil {
		return nil, fmt.Errorf("failed to extract connector: %w", err)
	}

	connector.Id = c.Status.Id
	return &cxsdk.ReplaceConnectorRequest{
		Connector: connector,
	}, nil
}

func (c *Connector) ExtractConnector() (*cxsdk.Connector, error) {
	connector := &cxsdk.Connector{
		Name:        c.Spec.Name,
		Description: c.Spec.Description,
	}

	if connectorType, ok := schemaToProtoConnectorType[c.Spec.Type]; ok {
		connector.Type = connectorType
	} else {
		return nil, fmt.Errorf("unsupported connector type: %s", c.Spec.Type)
	}

	connector.ConnectorConfig = &cxsdk.ConnectorConfig{
		Fields: ExtractConnectorConfigFields(c.Spec.ConnectorConfig.Fields),
	}

	connector.ConfigOverrides = ExtractEntityTypeConfigOverrides(c.Spec.ConfigOverrides)
	return connector, nil
}

func ExtractConnectorConfigFields(fields []ConnectorConfigField) []*cxsdk.ConnectorConfigField {
	var result []*cxsdk.ConnectorConfigField
	for _, field := range fields {
		result = append(result, &cxsdk.ConnectorConfigField{
			FieldName: field.FieldName,
			Value:     field.Value,
		})
	}

	return result
}

func ExtractEntityTypeConfigOverrides(overrides []EntityTypeConfigOverrides) []*cxsdk.EntityTypeConfigOverrides {
	var result []*cxsdk.EntityTypeConfigOverrides
	for _, override := range overrides {
		entityType := &cxsdk.EntityTypeConfigOverrides{
			EntityType: override.EntityType,
		}
		for _, field := range override.Fields {
			entityType.Fields = append(entityType.Fields, &cxsdk.TemplatedConnectorConfigField{
				FieldName: field.FieldName,
				Template:  field.Template,
			})
		}
		result = append(result, entityType)
	}
	return result
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
