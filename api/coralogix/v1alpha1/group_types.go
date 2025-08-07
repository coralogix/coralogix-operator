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
	"errors"
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/coralogix/coralogix-operator/internal/config"
)

// GroupSpec defines the desired state of Coralogix Group.
type GroupSpec struct {
	// Name of the group.
	Name string `json:"name"`

	// Description of the group.
	Description *string `json:"description,omitempty"`

	// Members of the group.
	// +optional
	Members []Member `json:"members,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// Custom roles applied to the group.
	CustomRoles []GroupCustomRole `json:"customRoles"`

	// Scope attached to the group.
	// +optional
	Scope *GroupScope `json:"scope,omitempty"`
}

// User on Coralogix.
type Member struct {
	// User's name.
	UserName string `json:"userName"`
}

// Custom role reference.
type GroupCustomRole struct {
	// Reference to the custom role within the cluster.
	ResourceRef ResourceRef `json:"resourceRef"`
}

// Scope attached to the group.
type GroupScope struct {
	// Scope reference.
	ResourceRef ResourceRef `json:"resourceRef"`
}

// Reference to a Coralogix resource within the cluster.
type ResourceRef struct {
	// Name of the resource (not id).
	Name string `json:"name"`

	// Kubernetes namespace.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

func (g *Group) ExtractCreateGroupRequest(
	ctx context.Context,
	cxClient *cxsdk.ClientSet) (*cxsdk.CreateTeamGroupRequest, error) {
	usersIds, err := g.ExtractUsersIDs(ctx, cxClient)
	if err != nil {
		return nil, err
	}

	rolesIds, err := g.ExtractRolesIds()
	if err != nil {
		return nil, err
	}

	scopeId, err := g.ExtractScopeId()
	if err != nil {
		return nil, err
	}

	return &cxsdk.CreateTeamGroupRequest{
		Name:           g.Spec.Name,
		Description:    g.Spec.Description,
		UserIds:        usersIds,
		RoleIds:        rolesIds,
		NextGenScopeId: scopeId,
	}, nil
}

func (g *Group) ExtractUpdateGroupRequest(
	ctx context.Context, cxClient *cxsdk.ClientSet,
	groupID string) (*cxsdk.UpdateTeamGroupRequest, error) {
	usersIds, err := g.ExtractUsersIDs(ctx, cxClient)
	if err != nil {
		return nil, err
	}

	rolesIds, err := g.ExtractRolesIds()
	if err != nil {
		return nil, err
	}

	scopeId, err := g.ExtractScopeId()
	if err != nil {
		return nil, err
	}

	id, err := strconv.Atoi(groupID)
	if err != nil {
		return nil, err
	}

	return &cxsdk.UpdateTeamGroupRequest{
		GroupId:        ptr.To(cxsdk.TeamGroupID{Id: uint32(id)}),
		Name:           g.Spec.Name,
		Description:    g.Spec.Description,
		UserUpdates:    ptr.To(cxsdk.UpdateTeamGroupRequestUserUpdates{UserIds: usersIds}),
		RoleUpdates:    ptr.To(cxsdk.UpdateTeamGroupRequestRoleUpdates{RoleIds: rolesIds}),
		NextGenScopeId: scopeId,
	}, nil
}

func (g *Group) ExtractUsersIDs(ctx context.Context, cxClient *cxsdk.ClientSet) ([]*cxsdk.UserID, error) {
	if g.Spec.Members == nil {
		return nil, nil
	}

	users, err := cxClient.Users().List(ctx)
	if err != nil {
		return nil, err
	}

	var usersIDs []*cxsdk.UserID
	var errs error
	for _, member := range g.Spec.Members {
		found := false
		for _, user := range users {
			if user.UserName == member.UserName {
				found = true
				usersIDs = append(usersIDs, &cxsdk.UserID{Id: *user.ID})
				break
			}
		}
		if !found {
			errs = errors.Join(errs, fmt.Errorf("user %s not found", member.UserName))
		}
	}

	if errs != nil {
		return nil, errs
	}

	return usersIDs, nil
}

func (g *Group) ExtractRolesIds() ([]*cxsdk.RoleID, error) {
	var rolesIds []*cxsdk.RoleID
	var errs error
	for _, customRole := range g.Spec.CustomRoles {
		roleID, err := g.getRoleIDFromCustomRole(customRole)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		rolesIds = append(rolesIds, roleID)
	}

	if errs != nil {
		return nil, errs
	}

	return rolesIds, nil
}

func (g *Group) getRoleIDFromCustomRole(customRole GroupCustomRole) (*cxsdk.RoleID, error) {
	var namespace string
	if customRole.ResourceRef.Namespace != nil {
		namespace = *customRole.ResourceRef.Namespace
	} else {
		namespace = g.Namespace
	}

	cr := &CustomRole{}
	if err := config.GetClient().Get(context.Background(), client.ObjectKey{Name: customRole.ResourceRef.Name, Namespace: namespace}, cr); err != nil {
		return nil, err
	}

	if !config.GetConfig().Selector.Matches(cr.Labels, cr.Namespace) {
		return nil, fmt.Errorf("custom role %s does not match selector", cr.Name)
	}

	if cr.Status.ID == nil {
		return nil, fmt.Errorf("ID is not populated for CustomRole %s", customRole.ResourceRef.Name)
	}

	roleID, err := strconv.Atoi(*cr.Status.ID)
	if err != nil {
		return nil, err
	}

	return &cxsdk.RoleID{Id: uint32(roleID)}, nil
}

func (g *Group) ExtractScopeId() (*string, error) {
	if g.Spec.Scope == nil {
		return nil, nil
	}

	var namespace string
	if g.Spec.Scope.ResourceRef.Namespace != nil {
		namespace = *g.Spec.Scope.ResourceRef.Namespace
	} else {
		namespace = g.Namespace
	}

	sc := &Scope{}
	if err := config.GetClient().Get(context.Background(), client.ObjectKey{Name: g.Spec.Scope.ResourceRef.Name, Namespace: namespace}, sc); err != nil {
		return nil, err
	}

	if !config.GetConfig().Selector.Matches(sc.Labels, sc.Namespace) {
		return nil, fmt.Errorf("scope %s does not match selector", sc.Name)
	}

	if sc.Status.ID == nil {
		return nil, fmt.Errorf("ID is not populated for Scope %s", g.Spec.Scope.ResourceRef.Name)
	}

	return sc.Status.ID, nil
}

// GroupStatus defines the observed state of Group.
type GroupStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (g *Group) GetConditions() []metav1.Condition {
	return g.Status.Conditions
}

func (g *Group) SetConditions(conditions []metav1.Condition) {
	g.Status.Conditions = conditions
}

func (g *Group) GetPrintableStatus() string {
	return g.Status.PrintableStatus
}

func (g *Group) SetPrintableStatus(printableStatus string) {
	g.Status.PrintableStatus = printableStatus
}

func (g *Group) HasIDInStatus() bool {
	return g.Status.ID != nil && *g.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// Group is the Schema for the Groups API.
// See also https://coralogix.com/docs/user-guides/account-management/user-management/assign-user-roles-and-scopes-via-groups/
//
// **Added in v0.4.0**
type Group struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSpec   `json:"spec,omitempty"`
	Status GroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupList contains a list of Groups.
type GroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Group `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Group{}, &GroupList{})
}
