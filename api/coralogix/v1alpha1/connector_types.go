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

	connectors "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/connectors_service"
)

// ConnectorSpec defines the desired state of Connector.
// See also https://coralogix.com/docs/user-guides/notification-center/introduction/connectors-explained/
type ConnectorSpec struct {
	// Name is the name of the connector.
	Name string `json:"name"`

	// Description is the description of the connector.
	Description string `json:"description"`

	// Type is the type of the connector. Can be one of slack, genericHttps, or pagerDuty.
	// +kubebuilder:validation:Enum=slack;genericHttps;pagerDuty
	Type string `json:"type"`

	// ConnectorConfig is the configuration of the connector.
	ConnectorConfig ConnectorConfig `json:"connectorConfig"`

	// ConfigOverrides are the entity type config overrides for the connector.
	// +optional
	ConfigOverrides []EntityTypeConfigOverrides `json:"configOverrides,omitempty"`
}

type ConnectorConfig struct {
	// Fields are the fields of the connector config.
	Fields []ConnectorConfigField `json:"fields"`
}

type ConnectorConfigField struct {
	// FieldName is the name of the field. e.g. "channel" for slack.
	FieldName string `json:"fieldName"`

	// Value is the value of the field.
	Value string `json:"value"`
}

type EntityTypeConfigOverrides struct {
	// EntityType is the entity type for the config override. Should equal "alerts".
	// +kubebuilder:validation:Enum=alerts
	EntityType string `json:"entityType"`

	// Fields are the templated fields for the config override.
	Fields []TemplatedConnectorConfigField `json:"fields,omitempty"`
}

type TemplatedConnectorConfigField struct {
	// FieldName is the name of the field. e.g. "channel" for slack.
	FieldName string `json:"fieldName"`

	// Template is the template for the field.
	Template string `json:"template"`
}

// ConnectorStatus defines the observed state of Connector.
type ConnectorStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (c *Connector) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *Connector) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

func (c *Connector) GetPrintableStatus() string {
	return c.Status.PrintableStatus
}

func (c *Connector) SetPrintableStatus(printableStatus string) {
	c.Status.PrintableStatus = printableStatus
}

func (c *Connector) HasIDInStatus() bool {
	return c.Status.Id != nil && *c.Status.Id != ""
}

var (
	schemaToOpenApiConnectorType = map[string]connectors.ConnectorType{
		"slack":        connectors.CONNECTORTYPE_SLACK,
		"genericHttps": connectors.CONNECTORTYPE_GENERIC_HTTPS,
		"pagerDuty":    connectors.CONNECTORTYPE_PAGERDUTY,
	}
	schemaToOpenApiEntityType = map[string]*connectors.NotificationCenterEntityType{
		"alerts": connectors.NOTIFICATIONCENTERENTITYTYPE_ALERTS.Ptr(),
	}
)

func (c *Connector) ExtractConnector() (*connectors.Connector1, error) {
	connector := &connectors.Connector1{
		Name:        c.Spec.Name,
		Description: connectors.PtrString(c.Spec.Description),
	}

	connectorType, ok := schemaToOpenApiConnectorType[c.Spec.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported connector type: %s", c.Spec.Type)
	}
	connector.Type = connectorType

	connector.ConnectorConfig = &connectors.ConnectorConfig{
		Fields: ExtractConnectorConfigFields(c.Spec.ConnectorConfig.Fields),
	}

	configOverrides, err := ExtractEntityTypeConfigOverrides(c.Spec.ConfigOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to extract config overrides: %w", err)
	}
	connector.ConfigOverrides = configOverrides

	return connector, nil
}

func ExtractConnectorConfigFields(fields []ConnectorConfigField) []connectors.NotificationCenterConnectorConfigField {
	var result []connectors.NotificationCenterConnectorConfigField
	for _, field := range fields {
		result = append(result, connectors.NotificationCenterConnectorConfigField{
			FieldName: connectors.PtrString(field.FieldName),
			Value:     connectors.PtrString(field.Value),
		})
	}

	return result
}

func ExtractEntityTypeConfigOverrides(overrides []EntityTypeConfigOverrides) ([]connectors.EntityTypeConfigOverrides, error) {
	var result []connectors.EntityTypeConfigOverrides
	for _, override := range overrides {
		entityType, ok := schemaToOpenApiEntityType[override.EntityType]
		if !ok {
			return nil, fmt.Errorf("invalid entity type %s", override.EntityType)
		}

		entityTypeConfigOverrides := connectors.EntityTypeConfigOverrides{
			EntityType: entityType,
		}

		entityTypeConfigOverrides.Fields = ExtractConfigOverridesFields(override.Fields)
		result = append(result, entityTypeConfigOverrides)
	}

	return result, nil
}

func ExtractConfigOverridesFields(fields []TemplatedConnectorConfigField) []connectors.TemplatedConnectorConfigField {
	var result []connectors.TemplatedConnectorConfigField
	for _, field := range fields {
		result = append(result, connectors.TemplatedConnectorConfigField{
			FieldName: connectors.PtrString(field.FieldName),
			Template:  connectors.PtrString(field.Template),
		})
	}

	return result
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Connector is the Schema for the connectors API.
//
// **Added in v0.4.0**
// NOTE: This CRD exposes a new feature and may have breaking changes in future releases.
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
