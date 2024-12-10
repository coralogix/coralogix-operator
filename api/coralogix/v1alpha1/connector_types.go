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
)

// ConnectorSpec defines the desired state of Connector.
type ConnectorSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	ConnectorType ConnectorType `json:"connectorType"`
}

// ConnectorStatus defines the observed state of Connector.
type ConnectorStatus struct {
	Id *string `json:"id"`
}

type ConnectorType struct {
	// +optional
	ConnectorGenericHttpsType *ConnectorGenericHttpsType `json:"genericHttps,omitempty"`

	// +optional
	ConnectorSlackType *ConnectorSlackType `json:"slack,omitempty"`
}

type ConnectorGenericHttpsType struct {
	Url string `json:"url"`

	// +kubebuilder:validation:Enum=Get;Post;Put
	Method string `json:"method"`

	Headers map[string]string `json:"headers"`

	Body string `json:"body"`
}

type ConnectorSlackType struct {
	// +optional
	ConnectorRawSlack *ConnectorRawSlack `json:"raw,omitempty"`

	// +optional
	ConnectorStructuredSlack *ConnectorStructuredSlack `json:"structured,omitempty"`
}

type ConnectorRawSlack struct {
	IntegrationId string `json:"integrationId"`

	FallbackChannel string `json:"fallbackChannel"`

	// +optional
	Channel string `json:"channel,omitempty"`
}

type ConnectorStructuredSlack struct {
	IntegrationId string `json:"integrationId"`

	FallbackChannel string `json:"fallbackChannel"`

	// +optional
	Channel string `json:"channel,omitempty"`
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
