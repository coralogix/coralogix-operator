/*
Copyright 2022 Coralogix Ltd..

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CoralogixLoggerSpec defines the desired state of CoralogixLogger
type CoralogixLoggerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Coralogix Region
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Region"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:Europe","urn:alm:descriptor:com.tectonic.ui:select:Europe2","urn:alm:descriptor:com.tectonic.ui:select:India","urn:alm:descriptor:com.tectonic.ui:select:Singapore","urn:alm:descriptor:com.tectonic.ui:select:US"}
	// +kubebuilder:validation:Enum=Europe;Europe2;India;Singapore;US
	// +kubebuilder:default:=Europe
	// +kubebuilder:validation:Optional
	Region string `json:"region"`
	// Coralogix Private Key
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Private Key"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:password"
	// +kubebuilder:validation:Format="password"
	// +kubebuilder:validation:Pattern="^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-z]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
	// +kubebuilder:validation:MinLength=36
	// +kubebuilder:validation:MaxLength=36
	// +kubebuilder:validation:Required
	PrivateKey string `json:"private_key"`
	// Current cluster name
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Cluster Name"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:com.tectonic.ui:advanced"
	// +kubebuilder:default:=cluster.local
	// +kubebuilder:validation:Optional
	ClusterName string `json:"cluster_name"`
}

// CoralogixLoggerStatus defines the observed state of CoralogixLogger
type CoralogixLoggerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ServiceAccount name
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="ServiceAccount"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes:ServiceAccount"
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// ClusterRole name
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="ClusterRole"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes:ClusterRole"
	ClusterRole string `json:"clusterRole,omitempty"`
	// ClusterRoleBinding name
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="ClusterRoleBinding"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes:ClusterRoleBinding"
	ClusterRoleBinding string `json:"clusterRoleBinding,omitempty"`
	// DaemonSet name
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="DaemonSet"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes:DaemonSet"
	DaemonSet string `json:"daemonSet,omitempty"`
	// Current state of logging agent
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="State"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:text"
	State string `json:"state,omitempty"`
	// Phase
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Phase"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes.phase"
	Phase string `json:"phase,omitempty"`
	// Reason
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Reason"
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors="urn:alm:descriptor:io.kubernetes.phase:reason"
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Coralogix logging agent.
// +operator-sdk:csv:customresourcedefinitions:displayName="Coralogix Logger"
// +operator-sdk:csv:customresourcedefinitions:resources={{DaemonSet,v1,""},{ServiceAccount,v1,fluentd-coralogix-service-account},{ClusterRole,v1,fluentd-coralogix-role},{ClusterRoleBinding,v1,fluentd-coralogix-role-binding}}
type CoralogixLogger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoralogixLoggerSpec   `json:"spec,omitempty"`
	Status CoralogixLoggerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CoralogixLoggerList contains a list of CoralogixLogger
type CoralogixLoggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoralogixLogger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CoralogixLogger{}, &CoralogixLoggerList{})
}
