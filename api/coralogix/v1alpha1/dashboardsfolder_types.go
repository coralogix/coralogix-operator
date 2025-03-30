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

	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/internal/config"
)

// DashboardsFolderSpec defines the desired state of DashboardsFolder.
// +kubebuilder:validation:XValidation:rule="!(has(self.parentFolderId) && has(self.parentFolderRef))",message="Only one of parentFolderID or parentFolderRef can be declared at the same time"
type DashboardsFolderSpec struct {
	Name string `json:"name"`
	// A custom ID for the folder. If not provided, a random UUID will be generated. The custom ID is immutable.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.customId is immutable"
	// +kubebuilder:validation:Pattern=`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	CustomID *string `json:"customId,omitempty"`
	// A reference to an existing folder by its backend's ID.
	// +optional
	ParentFolderID *string `json:"parentFolderId,omitempty"`
	// A reference to an existing DashboardsFolder CR.
	// +optional
	ParentFolderRef *ResourceRef `json:"parentFolderRef,omitempty"`
}

func (in *DashboardsFolderSpec) ExtractDashboardsFolderFromSpec(ctx context.Context, namespace string) (*cxsdk.DashboardFolder, error) {
	dashboardFolder := new(cxsdk.DashboardFolder)
	dashboardFolder.Name = wrapperspb.String(in.Name)

	if parentID := in.ParentFolderID; parentID != nil {
		dashboardFolder.ParentId = wrapperspb.String(*parentID)
	} else if parentRef := in.ParentFolderRef; parentRef != nil {
		var err error
		parentId, err := GetFolderIdFromFolderCR(ctx, namespace, *parentRef)
		if err != nil {
			return nil, err
		}
		dashboardFolder.ParentId = wrapperspb.String(parentId)
	}
	return dashboardFolder, nil
}

func GetFolderIdFromFolderCR(ctx context.Context, namespace string, parentRef ResourceRef) (string, error) {
	df := &DashboardsFolder{}
	if parentRef.Namespace != nil {
		namespace = *parentRef.Namespace
	}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: parentRef.Name, Namespace: namespace}, df); err != nil {
		return "", fmt.Errorf("failed to get DashboardsFolder: %w", err)
	}
	if df.Status.ID == nil || *df.Status.ID == "" {
		return "", fmt.Errorf("failed to get DashboardsFolder ID")
	}
	return *df.Status.ID, nil
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) || has(self.resourceRef)", message="One of backendRef or resourceRef is required"
// +kubebuilder:validation:XValidation:rule="!(has(self.backendRef) && has(self.resourceRef))", message="Only one of backendRef or resourceRef can be declared at the same time"
type DashboardFolderRef struct {
	// +optional
	// +kubebuilder:validation:XValidation:rule="has(self.id) || has(self.path)",message="One of id or path is required"
	// +kubebuilder:validation:XValidation:rule="!(has(self.id) && has(self.path))",message="Only one of id or path can be declared at the same time"
	BackendRef *DashboardFolderRefBackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

type DashboardFolderRefBackendRef struct {
	// Reference to a folder by its backend's ID.
	// +optional
	ID *string `json:"id,omitempty"`
	// Reference to a folder by its path (<parent-folder-name-1>/<parent-folder-name-2>/<folder-name>).
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
