// Copyright 2026 Coralogix Ltd.
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

// ConfigurationGroupSpec defines the desired state of a Fleet Manager configuration group.
// This resource is in Beta and uses the Preview configuration-group API.
type ConfigurationGroupSpec struct {
	// Display name.
	Name string `json:"name"`

	// Human-readable description.
	// +optional
	Description *string `json:"description,omitempty"`

	// Tags attached to the configuration group.
	// +optional
	Tags []string `json:"tags,omitempty"`

	// Selection precedence. Higher values win on ties and 0 is the default.
	// +optional
	PriorityOrder *int32 `json:"priorityOrder,omitempty"`

	// Latest configuration family for this group.
	Family ConfigurationFamilySpec `json:"family"`
}

// ConfigurationFamilySpec is the latest family nested in a configuration group.
type ConfigurationFamilySpec struct {
	// Whether this family is active.
	// +kubebuilder:default=true
	// +optional
	Active *bool `json:"active,omitempty"`

	// Human-readable description.
	// +optional
	Description *string `json:"description,omitempty"`

	// Collector semantic version this family targets, without a leading v prefix.
	// +optional
	CollectorVersion *string `json:"collectorVersion,omitempty"`

	// Metadata stored with this configuration family.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// Remote OpenTelemetry Collector configurations in this family.
	// +kubebuilder:validation:MinItems=1
	RemoteConfigurations []RemoteConfigurationSpec `json:"remoteConfigurations"`
}

// RemoteConfigurationSpec is a remote OpenTelemetry Collector configuration.
type RemoteConfigurationSpec struct {
	// Remote configuration name.
	Name string `json:"name"`

	// OpenTelemetry Collector configuration YAML. The supervisor-managed OpAMP extension must not be configured.
	RawConfiguration string `json:"rawConfiguration"`

	// Flat agent attributes that match agents for this configuration.
	// +optional
	AgentSelector map[string]string `json:"agentSelector,omitempty"`
}

// ConfigurationGroupStatus defines the observed state of ConfigurationGroup.
type ConfigurationGroupStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (r *ConfigurationGroup) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *ConfigurationGroup) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

func (r *ConfigurationGroup) GetPrintableStatus() string {
	return r.Status.PrintableStatus
}

func (r *ConfigurationGroup) SetPrintableStatus(printableStatus string) {
	r.Status.PrintableStatus = printableStatus
}

func (r *ConfigurationGroup) HasIDInStatus() bool {
	return r.Status.ID != nil && *r.Status.ID != ""
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ConfigurationGroup is the Schema for the configurationgroups API.
// This resource is in Beta. It uses the Fleet Manager Preview configuration-group API.
type ConfigurationGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationGroupSpec   `json:"spec,omitempty"`
	Status ConfigurationGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigurationGroupList contains a list of ConfigurationGroup.
type ConfigurationGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigurationGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigurationGroup{}, &ConfigurationGroupList{})
}
