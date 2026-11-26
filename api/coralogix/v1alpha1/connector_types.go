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

	// Type is the type of the connector. Can be one of slack, genericHttps, pagerDuty, pagerDutyIncidents, email, serviceNow, microsoftTeams, eventBridge, or incidentIo (preview).
	// +kubebuilder:validation:Enum=slack;genericHttps;pagerDuty;pagerDutyIncidents;email;serviceNow;microsoftTeams;eventBridge;incidentIo
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
	// EntityType is the entity type for the config override. Can be one of alerts or cases.
	// +kubebuilder:validation:Enum=alerts;cases
	EntityType string `json:"entityType"`

	// Fields are the templated fields for the config override.
	Fields []TemplatedConnectorConfigField `json:"fields,omitempty"`
}

// TemplatedConnectorConfigField defines a templated field in a config override.
// +kubebuilder:validation:XValidation:rule="(has(self.template) ? 1 : 0) + (has(self.templateSecretKeyRef) ? 1 : 0) == 1",message="Exactly one of template or templateSecretKeyRef must be set"
type TemplatedConnectorConfigField struct {
	// FieldName is the name of the field. e.g. "channel" for slack.
	FieldName string `json:"fieldName"`

	// Template is the literal template for the field. Conflicts with TemplateSecretKeyRef.
	// Use TemplateSecretKeyRef when the template holds a credential.
	// +optional
	Template string `json:"template,omitempty"`

	// TemplateSecretKeyRef is a reference to a secret key containing the template.
	// The secret must be in the same namespace as the Connector.
	// Conflicts with Template.
	// +optional
	TemplateSecretKeyRef *corev1.SecretKeySelector `json:"templateSecretKeyRef,omitempty"`
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
	schemaToOpenApiConnectorType = map[string]connectors.NotificationCenterConnectorType{
		"slack":              connectors.NOTIFICATIONCENTERCONNECTORTYPE_SLACK,
		"genericHttps":       connectors.NOTIFICATIONCENTERCONNECTORTYPE_GENERIC_HTTPS,
		"pagerDuty":          connectors.NOTIFICATIONCENTERCONNECTORTYPE_PAGERDUTY,
		"pagerDutyIncidents": connectors.NOTIFICATIONCENTERCONNECTORTYPE_PAGERDUTY_INCIDENTS,
		"email":              connectors.NOTIFICATIONCENTERCONNECTORTYPE_EMAIL,
		"serviceNow":         connectors.NOTIFICATIONCENTERCONNECTORTYPE_SERVICE_NOW,
		"microsoftTeams":     connectors.NOTIFICATIONCENTERCONNECTORTYPE_MICROSOFT_TEAMS,
		"eventBridge":        connectors.NOTIFICATIONCENTERCONNECTORTYPE_EVENTBRIDGE,
		"incidentIo":         connectors.NOTIFICATIONCENTERCONNECTORTYPE_INCIDENT_IO,
	}
	schemaToOpenApiEntityType = map[string]connectors.NotificationCenterEntityType{
		"alerts": connectors.NOTIFICATIONCENTERENTITYTYPE_ALERTS,
		"cases":  connectors.NOTIFICATIONCENTERENTITYTYPE_CASES,
	}
)

func (c *Connector) ExtractConnector(ctx context.Context) (*connectors.Connector, error) {
	connector := &connectors.Connector{
		Name:        c.Spec.Name,
		Description: connectors.PtrString(c.Spec.Description),
	}

	connectorType, ok := schemaToOpenApiConnectorType[c.Spec.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported connector type: %s", c.Spec.Type)
	}
	connector.Type = connectorType

	fields, err := ExtractConnectorConfigFields(ctx, c.Spec.ConnectorConfig.Fields, c.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to extract connector config fields: %w", err)
	}
	connector.ConnectorConfig = &connectors.ConnectorConfig{
		Fields: fields,
	}

	configOverrides, err := ExtractEntityTypeConfigOverrides(ctx, c.Spec.ConfigOverrides, c.Namespace)
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
			FieldName: field.FieldName,
			Value:     connectors.PtrString(value),
		})
	}

	return result, nil
}

func ExtractEntityTypeConfigOverrides(ctx context.Context, overrides []EntityTypeConfigOverrides, namespace string) ([]connectors.EntityTypeConfigOverrides, error) {
	var result []connectors.EntityTypeConfigOverrides
	for _, override := range overrides {
		entityType, ok := schemaToOpenApiEntityType[override.EntityType]
		if !ok {
			return nil, fmt.Errorf("invalid entity type %s", override.EntityType)
		}

		entityTypeConfigOverrides := connectors.EntityTypeConfigOverrides{
			EntityType: entityType,
		}

		fields, err := ExtractConfigOverridesFields(ctx, override.Fields, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to extract fields of entity type %s: %w", override.EntityType, err)
		}
		entityTypeConfigOverrides.Fields = fields
		result = append(result, entityTypeConfigOverrides)
	}

	return result, nil
}

func ExtractConfigOverridesFields(ctx context.Context, fields []TemplatedConnectorConfigField, namespace string) ([]connectors.TemplatedConnectorConfigField, error) {
	var result []connectors.TemplatedConnectorConfigField
	for _, field := range fields {
		template := field.Template
		if field.TemplateSecretKeyRef != nil {
			if template != "" {
				return nil, fmt.Errorf("field '%s' sets both template and templateSecretKeyRef; only one source is allowed", field.FieldName)
			}
			secretValue, err := readSecret(ctx, *field.TemplateSecretKeyRef, namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to read secret for template of field '%s': %w", field.FieldName, err)
			}
			template = secretValue
		}

		result = append(result, connectors.TemplatedConnectorConfigField{
			FieldName: field.FieldName,
			Template:  connectors.PtrString(template),
		})
	}

	return result, nil
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
