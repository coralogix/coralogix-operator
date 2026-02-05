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
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	groups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/team_groups_management_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

var (
	groupTypeSchemaToOpenAPI = map[string]groups.GroupType{
		"unspecified": groups.GROUPTYPE_GROUP_TYPE_UNSPECIFIED,
		"open":        groups.GROUPTYPE_GROUP_TYPE_OPEN,
		"closed":      groups.GROUPTYPE_GROUP_TYPE_CLOSED,
		"restricted":  groups.GROUPTYPE_GROUP_TYPE_RESTRICTED,
	}
)

// GroupSpec defines the desired state of Coralogix Group.
type GroupSpec struct {
	// Name of the group.
	Name string `json:"name"`

	// Description of the group.
	Description *string `json:"description,omitempty"`

	// Type of the group.
	// +optional
	// +kubebuilder:validation:Enum=unspecified;open;closed;restricted
	GroupType *string `json:"groupType,omitempty"`

	// Members of the group.
	// +optional
	Members []Member `json:"members,omitempty"`

	// Custom roles applied to the group.
	// +optional
	CustomRole *GroupCustomRole `json:"customRole,omitempty"`

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
	cxClient *cxsdk.ClientSet) (*groups.CreateTeamGroupRequest, error) {
	var groupType *groups.GroupType
	if g.Spec.GroupType != nil {
		groupType = groupTypeSchemaToOpenAPI[*g.Spec.GroupType].Ptr()
	}

	usersIds, err := g.ExtractUsersIDs(ctx, cxClient)
	if err != nil {
		return nil, err
	}

	roleId, err := g.ExtractRoleId()
	if err != nil {
		return nil, err
	}

	scopeId, err := g.ExtractScopeId()
	if err != nil {
		return nil, err
	}

	return &groups.CreateTeamGroupRequest{
		Name:        groups.PtrString(g.Spec.Name),
		Description: g.Spec.Description,
		GroupType:   groupType,
		UserIds:     usersIds,
		RoleId:      &roleId,
		Scope: &groups.V2Scope{
			ScopeId: scopeId,
		},
	}, nil
}

func (g *Group) ExtractUpdateGroupRequest(
	ctx context.Context, cxClient *cxsdk.ClientSet) (*groups.UpdateTeamGroupRequest, error) {
	var groupType *groups.GroupType
	if g.Spec.GroupType != nil {
		groupType = groupTypeSchemaToOpenAPI[*g.Spec.GroupType].Ptr()
	}

	usersIds, err := g.ExtractUsersIDs(ctx, cxClient)
	if err != nil {
		return nil, err
	}

	roleId, err := g.ExtractRoleId()
	if err != nil {
		return nil, err
	}

	scopeId, err := g.ExtractScopeId()
	if err != nil {
		return nil, err
	}

	return &groups.UpdateTeamGroupRequest{
		Name:        groups.PtrString(g.Spec.Name),
		Description: g.Spec.Description,
		GroupType:   groupType,
		UserUpdates: &groups.UserUpdates{
			Operation: &groups.UserUpdatesOperation{
				UserUpdatesOperationSet: &groups.UserUpdatesOperationSet{
					Set: &groups.UserIdList{
						UserIds: usersIds,
					},
				},
			},
		},
		RoleUpdate: &groups.RoleUpdate{
			Action: &groups.RoleUpdateAction{
				RoleUpdateActionSetRoleId: &groups.RoleUpdateActionSetRoleId{
					SetRoleId: &groups.SetRoleId{
						Value: &roleId,
					},
				},
			},
		},
		ScopeUpdate: &groups.ScopeUpdate{
			Action: &groups.ScopeUpdateAction{
				ScopeUpdateActionSetScopeId: &groups.ScopeUpdateActionSetScopeId{
					SetScopeId: &groups.SetScopeId{
						Value: scopeId,
					},
				},
			},
		},
	}, nil
}

func (g *Group) ExtractUsersIDs(ctx context.Context, cxClient *cxsdk.ClientSet) ([]string, error) {
	if g.Spec.Members == nil {
		return nil, nil
	}

	users, err := cxClient.Users().List(ctx)
	if err != nil {
		return nil, err
	}

	var usersIDs []string
	var errs error
	for _, member := range g.Spec.Members {
		found := false
		for _, user := range users {
			if user.UserName == member.UserName {
				found = true
				usersIDs = append(usersIDs, *user.ID)
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

func (g *Group) ExtractRoleId() (int64, error) {
	if g.Spec.CustomRole == nil {
		return 0, nil
	}
	var namespace string
	if ns := g.Spec.CustomRole.ResourceRef.Namespace; ns != nil {
		namespace = *ns
	} else {
		namespace = g.Namespace
	}

	cr := &CustomRole{}
	if err := config.GetClient().Get(context.Background(), client.ObjectKey{Name: g.Spec.CustomRole.ResourceRef.Name, Namespace: namespace}, cr); err != nil {
		return 0, err
	}

	if !config.GetConfig().Selector.Matches(cr.Labels, cr.Namespace) {
		return 0, fmt.Errorf("custom role %s does not match selector", cr.Name)
	}

	if cr.Status.ID == nil {
		return 0, fmt.Errorf("ID is not populated for CustomRole %s", g.Spec.CustomRole.ResourceRef.Name)
	}

	roleID, err := strconv.Atoi(*cr.Status.ID)
	if err != nil {
		return 0, err
	}

	return int64(roleID), nil
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
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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
