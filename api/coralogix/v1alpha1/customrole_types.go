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
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// CustomRoleSpec defines the desired state of CustomRole.
type CustomRoleSpec struct {
	Name string `json:"name"`

	Description string `json:"description"`

	ParentRoleName string `json:"parentRoleName"`

	Permissions []string `json:"permissions"`
}

func (s *CustomRoleSpec) ExtractCreateCustomRoleRequest() *cxsdk.CreateRoleRequest {
	return &cxsdk.CreateRoleRequest{
		Name:        s.Name,
		Description: s.Description,
		ParentRole:  ptr.To(cxsdk.CreateRoleRequestParentRoleName{ParentRoleName: s.ParentRoleName}),
		Permissions: s.Permissions,
	}
}

func (s *CustomRoleSpec) ExtractUpdateCustomRoleRequest(id string) (*cxsdk.UpdateRoleRequest, error) {
	roleID, err := strconv.Atoi(id)
	if err != nil {
		return &cxsdk.UpdateRoleRequest{}, err
	}
	return &cxsdk.UpdateRoleRequest{
		RoleId:         uint32(roleID),
		NewName:        ptr.To(s.Name),
		NewDescription: ptr.To(s.Description),
		NewPermissions: ptr.To(cxsdk.RolePermissions{Permissions: s.Permissions}),
	}, nil
}

// CustomRoleStatus defines the observed state of CustomRole.
type CustomRoleStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (c *CustomRole) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *CustomRole) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CustomRole is the Schema for the customroles API.
type CustomRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomRoleSpec   `json:"spec,omitempty"`
	Status CustomRoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CustomRoleList contains a list of CustomRole.
type CustomRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CustomRole{}, &CustomRoleList{})
}
