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

// ViewFolderSpec defines the desired state of folder for views.
type ViewFolderSpec struct {
	// Name of the view folder
	Name string `json:"name"`
}

// ViewFolderStatus defines the observed state of ViewFolder.
type ViewFolderStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (v *ViewFolder) GetConditions() []metav1.Condition {
	return v.Status.Conditions
}

func (v *ViewFolder) SetConditions(conditions []metav1.Condition) {
	v.Status.Conditions = conditions
}

func (v *ViewFolder) HasIDInStatus() bool {
	return v.Status.ID != nil && *v.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ViewFolder is the Schema for the viewfolders API.
type ViewFolder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ViewFolderSpec   `json:"spec,omitempty"`
	Status ViewFolderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ViewFolderList contains a list of ViewFolder.
// See also https://coralogix.com/docs/user-guides/monitoring-and-insights/explore-screen/custom-views/
//
// **Added in v0.4.0**
type ViewFolderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ViewFolder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ViewFolder{}, &ViewFolderList{})
}
