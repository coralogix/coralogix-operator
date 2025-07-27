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

// ApiKeySpec defines the desired state of a Coralogix ApiKey.
// +kubebuilder:validation:XValidation:rule="has(self.presets) || has(self.permissions)",message="At least one of presets or permissions must be set"
type ApiKeySpec struct {

	//+kubebuilder:validation:MinLength=0
	// Name of the ApiKey
	Name string `json:"name"`

	//+kubebuilder:default=true
	// Whether the ApiKey Is active.
	// +optional
	// TODO: add validation for active to be true on create
	Active bool `json:"active"`

	// Owner of the ApiKey.
	Owner ApiKeyOwner `json:"owner"`

	// Permission Presets that the ApiKey uses.
	// +optional
	Presets []string `json:"presets,omitempty"`

	// Permissions of the ApiKey
	// +optional
	Permissions []string `json:"permissions,omitempty"`
}

// Owner of an ApiKey.
// +kubebuilder:validation:XValidation:rule="has(self.userId) != has(self.teamId)",message="Exactly one of userId or teamId must be set"
type ApiKeyOwner struct {
	// User that owns the key.
	// +optional
	UserId *string `json:"userId,omitempty"`

	// Team that owns the key.
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

func (a *ApiKey) HasIDInStatus() bool {
	return a.Status.Id != nil && *a.Status.Id != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// ApiKey is the Schema for the ApiKeys API.
// See also https://coralogix.com/docs/user-guides/account-management/api-keys/api-keys/
//
// **Added in v0.4.0**
type ApiKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiKeySpec   `json:"spec,omitempty"`
	Status ApiKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ApiKeyList contains a list of ApiKeys.
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
