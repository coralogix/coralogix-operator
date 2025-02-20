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
	"k8s.io/utils/ptr"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// ApiKeySpec defines the desired state of ApiKey.
type ApiKeySpec struct {
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// +optional
	//+kubebuilder:default=true
	Active bool `json:"active"`

	Owner ApiKeyOwner `json:"owner"`

	// +optional
	Presets []string `json:"presets,omitempty"`

	// +optional
	Permissions []string `json:"permissions,omitempty"`
}

type ApiKeyOwner struct {
	// +optional
	UserId *string `json:"userId,omitempty"`

	// +optional
	TeamId *uint32 `json:"teamId,omitempty"`
}

// ApiKeyStatus defines the observed state of ApiKey.
type ApiKeyStatus struct {
	// +optional
	Id *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (a *ApiKey) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *ApiKey) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApiKey is the Schema for the apikeys API.
type ApiKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiKeySpec   `json:"spec,omitempty"`
	Status ApiKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApiKeyList contains a list of ApiKey.
type ApiKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApiKey `json:"items"`
}

func (s *ApiKeySpec) ExtractCreateApiKeyRequest() *cxsdk.CreateAPIKeyRequest {
	owner := &cxsdk.Owner{}
	if s.Owner.UserId != nil {
		owner.Owner = &cxsdk.OwnerUserID{UserId: *s.Owner.UserId}
	}
	if s.Owner.TeamId != nil {
		owner.Owner = &cxsdk.OwnerTeamID{TeamId: *s.Owner.TeamId}
	}

	return &cxsdk.CreateAPIKeyRequest{
		Name:  s.Name,
		Owner: owner,
		KeyPermissions: &cxsdk.APIKeyPermissions{
			Presets:     s.Presets,
			Permissions: s.Permissions,
		},
		Hashed: false,
	}
}

func (s *ApiKeySpec) ExtractUpdateApiKeyRequest(id string) *cxsdk.UpdateAPIKeyRequest {
	return &cxsdk.UpdateAPIKeyRequest{
		KeyId:       id,
		NewName:     ptr.To(s.Name),
		IsActive:    ptr.To(s.Active),
		Presets:     ptr.To(cxsdk.APIKeyPresetsUpdate{Presets: s.Presets}),
		Permissions: ptr.To(cxsdk.APIKeyPermissionsUpdate{Permissions: s.Permissions}),
	}
}

func init() {
	SchemeBuilder.Register(&ApiKey{}, &ApiKeyList{})
}
