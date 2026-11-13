// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

// The handwritten parts of the generated Dashboard kind: the folder reference that the
// beforeCreate and beforeUpdate hooks read, and the import annotation that the adopt hook reads.

// DashboardFolderRef places the dashboard in a folder. It replaces the folder of the JSON content.
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

// ImportDashboardIDAnnotationKey adopts a pre-existing remote Dashboard into a Dashboard CR by ID.
// Once the dashboard is adopted it is treated exactly like a normally-created Dashboard CR,
// and the CR's spec becomes the source of truth and overwrites the remote dashboard's content
// on the next reconcile. The annotation itself is not removed after adoption (mirroring
// adoption annotations in tools like Crossplane and AWS ACK); instead status.imported on the
// Dashboard tracks that adoption already happened, so that if the remote dashboard is later
// deleted outside the operator, it is recreated from spec instead of the import path
// retrying a Get for an ID that no longer exists.
const ImportDashboardIDAnnotationKey = "app.coralogix.com/import-id"
