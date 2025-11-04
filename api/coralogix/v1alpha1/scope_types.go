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

	scopes "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/scopes_service"
)

// ScopeSpec defines the desired state of a Coralogix Scope.
type ScopeSpec struct {
	// Scope display name.
	Name string `json:"name"`

	// Description of the scope. Optional.
	// +optional
	Description *string `json:"description,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// Filters applied to include data in the scope.
	Filters []ScopeFilter `json:"filters"`

	// +kubebuilder:validation:Enum=<v1>true;<v1>false
	// Default expression to use when no filter matches the query. Until further notice, this is limited to `true` (everything is included) or `false` (nothing is included). Use a version tag (e.g `<v1>true` or `<v1>false`)
	DefaultExpression string `json:"defaultExpression"`
}

var entityTypeSchemaToOpenAPI = map[string]*scopes.V1EntityType{
	"logs":        scopes.V1ENTITYTYPE_ENTITY_TYPE_LOGS.Ptr(),
	"spans":       scopes.V1ENTITYTYPE_ENTITY_TYPE_SPANS.Ptr(),
	"unspecified": scopes.V1ENTITYTYPE_ENTITY_TYPE_UNSPECIFIED.Ptr(),
}

// ScopeFilter defines a filter to include data in a scope.
type ScopeFilter struct {
	// +kubebuilder:validation:Enum=logs;spans;unspecified
	// Entity type to apply the expression on.
	EntityType string `json:"entityType"`

	// Expression to run.
	Expression string `json:"expression"`
}

func (s *ScopeSpec) ExtractCreateScopeRequest() (*scopes.CreateScopeRequest, error) {
	filters, err := s.ExtractScopeFilters()
	if err != nil {
		return nil, err
	}

	return &scopes.CreateScopeRequest{
		DisplayName:       s.Name,
		Description:       s.Description,
		Filters:           filters,
		DefaultExpression: scopes.PtrString(s.DefaultExpression),
	}, nil
}

func (s *ScopeSpec) ExtractUpdateScopeRequest(id string) (*scopes.UpdateScopeRequest, error) {
	filters, err := s.ExtractScopeFilters()
	if err != nil {
		return nil, err
	}

	return &scopes.UpdateScopeRequest{
		Id:                id,
		DisplayName:       s.Name,
		Description:       s.Description,
		Filters:           filters,
		DefaultExpression: s.DefaultExpression,
	}, nil
}

func (s *ScopeSpec) ExtractScopeFilters() ([]scopes.ScopesV1Filter, error) {
	var filters []scopes.ScopesV1Filter
	for _, f := range s.Filters {
		filters = append(filters, scopes.ScopesV1Filter{
			EntityType: entityTypeSchemaToOpenAPI[f.EntityType],
			Expression: scopes.PtrString(f.Expression),
		})
	}

	return filters, nil
}

// ScopeStatus defines the observed state of Coralogix Scope.
type ScopeStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (s *Scope) GetConditions() []metav1.Condition {
	return s.Status.Conditions
}

func (s *Scope) SetConditions(conditions []metav1.Condition) {
	s.Status.Conditions = conditions
}

func (s *Scope) GetPrintableStatus() string {
	return s.Status.PrintableStatus
}

func (s *Scope) SetPrintableStatus(printableStatus string) {
	s.Status.PrintableStatus = printableStatus
}

func (s *Scope) HasIDInStatus() bool {
	return s.Status.ID != nil && *s.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Scope is the Schema for the scopes API.
// See also https://coralogix.com/docs/user-guides/account-management/user-management/scopes/
//
// **Added in v0.4.0**
type Scope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScopeSpec   `json:"spec,omitempty"`
	Status ScopeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScopeList contains a list of Scopes.
type ScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Scope{}, &ScopeList{})
}
