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

package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// AlertSetSpec defines a set of Coralogix alerts.
type AlertSetSpec struct {
	// Alerts contains the alerts that this resource manages.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=100
	// +listType=map
	// +listMapKey=key
	Alerts []AlertSetItem `json:"alerts"`
}

// AlertSetItem defines one alert in an AlertSet.
type AlertSetItem struct {
	// Key is the stable identity of the alert in this AlertSet.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Key string `json:"key"`

	// Spec defines the alert.
	Spec AlertSpec `json:"spec"`
}

// AlertSetItemState is the synchronization state of one alert.
// +kubebuilder:validation:Enum=Pending;Synced;Failed;Deleting
type AlertSetItemState string

const (
	AlertSetItemStatePending  AlertSetItemState = "Pending"
	AlertSetItemStateSynced   AlertSetItemState = "Synced"
	AlertSetItemStateFailed   AlertSetItemState = "Failed"
	AlertSetItemStateDeleting AlertSetItemState = "Deleting"
)

// AlertSetItemStatus defines the observed state of one alert.
type AlertSetItemStatus struct {
	// Key is the stable identity of the alert in this AlertSet.
	Key string `json:"key"`

	// ID is the remote Coralogix alert ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// State is the latest synchronization state.
	// +optional
	State AlertSetItemState `json:"state,omitempty"`

	// Message describes the latest synchronization failure.
	// +optional
	Message string `json:"message,omitempty"`
}

// AlertSetStatus defines the observed state of an AlertSet.
type AlertSetStatus struct {
	// Alerts contains the observed state of each managed alert.
	// +optional
	// +listType=map
	// +listMapKey=key
	Alerts []AlertSetItemStatus `json:"alerts,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// AlertSet is the Schema for the AlertSets API.
type AlertSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertSetSpec   `json:"spec"`
	Status AlertSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertSetList contains a list of AlertSet resources.
type AlertSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertSet{}, &AlertSetList{})
}

func (a *AlertSet) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *AlertSet) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
}

func (a *AlertSet) HasIDInStatus() bool {
	for _, alert := range a.Status.Alerts {
		if alert.ID != nil && *alert.ID != "" {
			return true
		}
	}
	return false
}

func (a *AlertSet) GetPrintableStatus() string {
	return a.Status.PrintableStatus
}

func (a *AlertSet) SetPrintableStatus(printableStatus string) {
	a.Status.PrintableStatus = printableStatus
}
