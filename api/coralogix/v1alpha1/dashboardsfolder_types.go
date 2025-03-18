/*
Copyright 2024.

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

package v1alpha1

import (
	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DashboardsFolderSpec defines the desired state of DashboardsFolder.
type DashboardsFolderSpec struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	CustomID string `json:"customId,omitempty"`
	// +optional
	ParentFolderBackendID *string `json:"parentFolderBackendId,omitempty"`
}

func (in *DashboardsFolderSpec) ExtractDashboardsFolderFromSpec() *cxsdk.DashboardFolder {
	dashboardFolder := new(cxsdk.DashboardFolder)
	dashboardFolder.Name = wrapperspb.String(in.Name)
	dashboardFolder.Id = wrapperspb.String(in.CustomID)
	if parentID := in.ParentFolderBackendID; parentID != nil {
		dashboardFolder.ParentId = wrapperspb.String(*parentID)
	}
	return dashboardFolder
}

type DashboardFolderRef struct {
	// +optional
	BackendRef *DashboardFolderRefBackendRef `json:"backendRef"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef"`
}

type DashboardFolderRefBackendRef struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Path *string `json:"path,omitempty"`
}

// DashboardsFolderStatus defines the observed state of DashboardsFolder.
type DashboardsFolderStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	ID *string `json:"id,omitempty"`
}

func (df *DashboardsFolder) GetConditions() []metav1.Condition {
	return df.Status.Conditions
}

func (df *DashboardsFolder) SetConditions(conditions []metav1.Condition) {
	df.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DashboardsFolder is the Schema for the dashboardsfolders API.
type DashboardsFolder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DashboardsFolderSpec   `json:"spec,omitempty"`
	Status DashboardsFolderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DashboardsFolderList contains a list of DashboardsFolder.
type DashboardsFolderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DashboardsFolder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DashboardsFolder{}, &DashboardsFolderList{})
}
