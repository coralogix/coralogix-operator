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
// It mirrors the configuration-groups API create request. This resource is in Preview.
type ConfigurationGroupSpec struct {
	// Display name. Group names are unique per team and cannot be cleared.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	Name string `json:"name"`

	// Human-readable description.
	// +optional
	// +kubebuilder:validation:MaxLength=4096
	Description *string `json:"description,omitempty"`

	// Tags attached to the configuration group.
	// +optional
	// +kubebuilder:validation:MaxItems=64
	// +kubebuilder:validation:items:MaxLength=256
	Tags []string `json:"tags,omitempty"`

	// Selection precedence. Higher values win on ties and 0 is the default.
	// +optional
	PriorityOrder *int32 `json:"priorityOrder,omitempty"`

	// Latest configuration family for this group.
	Family ConfigurationFamilySpec `json:"family"`
}

// ConfigurationFamilySpec is the latest configuration family of a group.
// Exactly one of preset or raw must be set. Updates replace the whole preset or raw
// content, so omitted optional fields are cleared rather than kept. A family cannot
// switch between preset and raw; delete and recreate the ConfigurationGroup instead.
// +kubebuilder:validation:XValidation:rule="has(self.preset) != has(self.raw)",message="Exactly one of preset or raw is required"
// +kubebuilder:validation:XValidation:rule="has(self.preset) == has(oldSelf.preset)",message="family cannot switch between preset and raw; delete and recreate the ConfigurationGroup"
type ConfigurationFamilySpec struct {
	// Whether this family is active. Defaults to false.
	// +optional
	Active *bool `json:"active,omitempty"`

	// Human-readable description.
	// +optional
	// +kubebuilder:validation:MaxLength=4096
	Description *string `json:"description,omitempty"`

	// Configuration template settings; Coralogix generates the remote configurations from them.
	// +optional
	Preset *PresetConfigurationFamilySpec `json:"preset,omitempty"`

	// Configuration family defined directly by its remote configurations, each with an
	// OTel Collector configuration and an agent selector.
	// +optional
	Raw *RawConfigurationFamilySpec `json:"raw,omitempty"`
}

// PresetConfigurationFamilySpec defines a family generated from a Coralogix configuration template.
type PresetConfigurationFamilySpec struct {
	// Configuration template type: CHART_NAME_OTEL_INTEGRATION for Kubernetes, CHART_NAME_OTEL_ECS_EC2 for ECS on EC2,
	// or CHART_NAME_OTEL_LINUX_STANDALONE, CHART_NAME_OTEL_WINDOWS_STANDALONE, or CHART_NAME_OTEL_MACOS_STANDALONE for hosts.
	// +kubebuilder:validation:Enum=CHART_NAME_OTEL_INTEGRATION;CHART_NAME_OTEL_LINUX_STANDALONE;CHART_NAME_OTEL_WINDOWS_STANDALONE;CHART_NAME_OTEL_MACOS_STANDALONE;CHART_NAME_OTEL_ECS_EC2
	ChartName string `json:"chartName"`

	// Configuration template version. It determines the OTel Collector version, the generated collector configuration,
	// and which integrationVersion values are supported. Unsupported versions are rejected.
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Pattern=`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`
	ChartVersion string `json:"chartVersion"`

	// Version of the observability features format. Omit it to use the default for the template type and version.
	// +optional
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Pattern=`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`
	IntegrationVersion *string `json:"integrationVersion,omitempty"`

	// Environment setup values for the configuration template, such as ClusterName, KubernetesRunningOn,
	// ApplicationName, or SubsystemName. Set the observability features in observabilityFeatures.
	// +optional
	// +kubebuilder:validation:MaxProperties=64
	// +kubebuilder:validation:XValidation:rule="self.all(k, size(self[k]) <= 4096)",message="metadata values must be at most 4096 characters"
	Metadata map[string]string `json:"metadata,omitempty"`

	// Observability feature settings for the configuration template, as a JSON object string;
	// the available features depend on chartName and integrationVersion.
	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=1048576
	ObservabilityFeatures string `json:"observabilityFeatures"`
}

// RawConfigurationFamilySpec defines a family by its remote OpenTelemetry Collector configurations.
type RawConfigurationFamilySpec struct {
	// Collector semantic version this family targets, without a leading v prefix.
	// Every agentSelector must then set service.version to this value.
	// +optional
	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Pattern=`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`
	CollectorVersion *string `json:"collectorVersion,omitempty"`

	// Metadata stored with this configuration family.
	// +optional
	// +kubebuilder:validation:MaxProperties=64
	// +kubebuilder:validation:XValidation:rule="self.all(k, size(self[k]) <= 4096)",message="metadata values must be at most 4096 characters"
	Metadata map[string]string `json:"metadata,omitempty"`

	// Remote configurations in this family.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=128
	RemoteConfigurations []RemoteConfigurationSpec `json:"remoteConfigurations"`
}

// RemoteConfigurationSpec is a remote OpenTelemetry Collector configuration.
type RemoteConfigurationSpec struct {
	// Remote configuration name.
	// +optional
	// +kubebuilder:validation:MaxLength=256
	Name *string `json:"name,omitempty"`

	// OpenTelemetry Collector configuration YAML. The supervisor-managed OpAMP extension must not be configured.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1048576
	RawConfiguration string `json:"rawConfiguration"`

	// Agent attributes that match agents for this configuration.
	// +optional
	AgentSelector *AgentSelectorSpec `json:"agentSelector,omitempty"`
}

// AgentSelectorSpec selects the agents a remote configuration applies to.
type AgentSelectorSpec struct {
	// Flat agent attributes to match. If collectorVersion is set, service.version must match it.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
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
// This resource is in Preview and may change in breaking ways. It uses the Fleet Manager configuration-groups API.
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
