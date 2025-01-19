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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// ScopeSpec defines the desired state of Scope.
type ScopeSpec struct {
	Name string `json:"name"`

	// +optional
	Description *string `json:"description,omitempty"`

	Filters []ScopeFilter `json:"filters"`

	// +kubebuilder:validation:Enum=<v1>true;<v1>false
	DefaultExpression string `json:"defaultExpression"`
}

// ScopeFilter defines a filter for a scope
type ScopeFilter struct {
	// +kubebuilder:validation:Enum=logs;spans;unspecified
	EntityType string `json:"entityType"`

	Expression string `json:"expression"`
}

func (s *ScopeSpec) ExtractCreateScopeRequest() (*cxsdk.CreateScopeRequest, error) {
	filters, err := s.ExtractScopeFilters()
	if err != nil {
		return nil, err
	}

	return &cxsdk.CreateScopeRequest{
		DisplayName:       s.Name,
		Description:       s.Description,
		Filters:           filters,
		DefaultExpression: s.DefaultExpression,
	}, nil
}

func (s *ScopeSpec) ExtractUpdateScopeRequest(id string) (*cxsdk.UpdateScopeRequest, error) {
	filters, err := s.ExtractScopeFilters()
	if err != nil {
		return nil, err
	}

	return &cxsdk.UpdateScopeRequest{
		Id:                id,
		DisplayName:       s.Name,
		Description:       s.Description,
		Filters:           filters,
		DefaultExpression: s.DefaultExpression,
	}, nil
}

func (s *ScopeSpec) ExtractScopeFilters() ([]*cxsdk.ScopeFilter, error) {
	var filters []*cxsdk.ScopeFilter
	for _, f := range s.Filters {
		entityType, ok := cxsdk.EntityTypeValueLookup["ENTITY_TYPE_"+strings.ToUpper(f.EntityType)]
		if !ok {
			return nil, fmt.Errorf("invalid entity type: %s", f.EntityType)
		}
		filters = append(filters, &cxsdk.ScopeFilter{
			EntityType: cxsdk.EntityType(entityType),
			Expression: f.Expression,
		})
	}

	return filters, nil
}

// ScopeStatus defines the observed state of Scope.
type ScopeStatus struct {
	ID *string `json:"id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Scope is the Schema for the scopes API.
type Scope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScopeSpec   `json:"spec,omitempty"`
	Status ScopeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScopeList contains a list of Scope.
type ScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Scope{}, &ScopeList{})
}
