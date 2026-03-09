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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	connectors "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/connectors_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// ConnectorSpec defines the desired state of Connector.
// See also https://coralogix.com/docs/user-guides/notification-center/introduction/connectors-explained/
type ConnectorSpec struct {
	// Name is the name of the connector.
	Name string `json:"name"`

	// Description is the description of the connector.
	Description string `json:"description"`

	// Type is the type of the connector. Can be one of slack, genericHttps, pagerDuty, email, or serviceNow.
	// +kubebuilder:validation:Enum=slack;genericHttps;pagerDuty;email;serviceNow
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

// ConnectorConfigField defines a field in the connector configuration.
// +kubebuilder:validation:XValidation:rule="(has(self.value) ? 1 : 0) + (has(self.secretKeyRef) ? 1 : 0) == 1",message="Exactly one of value or secretKeyRef must be set"
type ConnectorConfigField struct {
	// FieldName is the name of the field. e.g. "channel" for slack.
	FieldName string `json:"fieldName"`

	// Value is the literal value of the field. Conflicts with SecretKeyRef.
	// +optional
	Value *string `json:"value,omitempty"`

	// SecretKeyRef is a reference to a secret key containing the field value.
	// Use this for sensitive data like API keys, integration keys, or tokens.
	// Conflicts with Value.
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
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
		"email":        connectors.CONNECTORTYPE_EMAIL,
		"serviceNow":   connectors.CONNECTORTYPE_SERVICE_NOW,
	}
	schemaToOpenApiEntityType = map[string]connectors.NotificationCenterEntityType{
		"alerts": connectors.NOTIFICATIONCENTERENTITYTYPE_ALERTS,
	}
)

func (c *Connector) ExtractConnector(ctx context.Context) (*connectors.Connector, error) {
	connector := &connectors.Connector{
		Name:        connectors.PtrString(c.Spec.Name),
		Description: connectors.PtrString(c.Spec.Description),
	}

	connectorType, ok := schemaToOpenApiConnectorType[c.Spec.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported connector type: %s", c.Spec.Type)
	}
	connector.Type = connectorType.Ptr()

	fields, err := ExtractConnectorConfigFields(ctx, c.Spec.ConnectorConfig.Fields, c.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to extract connector config fields: %w", err)
	}
	connector.ConnectorConfig = &connectors.ConnectorConfig{
		Fields: fields,
	}

	configOverrides, err := ExtractEntityTypeConfigOverrides(c.Spec.ConfigOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to extract config overrides: %w", err)
	}
	connector.ConfigOverrides = configOverrides

	return connector, nil
}

func ExtractConnectorConfigFields(ctx context.Context, fields []ConnectorConfigField, namespace string) ([]connectors.NotificationCenterConnectorConfigField, error) {
	var result []connectors.NotificationCenterConnectorConfigField
	for _, field := range fields {
		var value string
		if field.Value != nil {
			value = *field.Value
		} else if field.SecretKeyRef != nil {
			secretValue, err := readSecret(ctx, *field.SecretKeyRef, namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to read secret for field '%s': %w", field.FieldName, err)
			}
			value = secretValue
		} else {
			return nil, fmt.Errorf("field '%s' must have either value or secretKeyRef set", field.FieldName)
		}

		result = append(result, connectors.NotificationCenterConnectorConfigField{
			FieldName: connectors.PtrString(field.FieldName),
			Value:     connectors.PtrString(value),
		})
	}

	return result, nil
}

func ExtractEntityTypeConfigOverrides(overrides []EntityTypeConfigOverrides) ([]connectors.EntityTypeConfigOverrides, error) {
	var result []connectors.EntityTypeConfigOverrides
	for _, override := range overrides {
		entityType, ok := schemaToOpenApiEntityType[override.EntityType]
		if !ok {
			return nil, fmt.Errorf("invalid entity type %s", override.EntityType)
		}

		entityTypeConfigOverrides := connectors.EntityTypeConfigOverrides{
			EntityType: entityType.Ptr(),
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

func readSecret(ctx context.Context, secretKeyRef corev1.SecretKeySelector, namespace string) (string, error) {
	secret := &corev1.Secret{}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: secretKeyRef.Name}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret '%s': %w", secretKeyRef.Name, err)
	}

	if value, ok := secret.Data[secretKeyRef.Key]; ok {
		return string(value), nil
	}

	return "", fmt.Errorf("cannot find key '%s' in secret '%s'", secretKeyRef.Key, secretKeyRef.Name)
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
