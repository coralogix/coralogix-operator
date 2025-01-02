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

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

// GroupSpec defines the desired state of Group.
type GroupSpec struct {
	Name string `json:"name"`

	Description *string `json:"description,omitempty"`

	// +optional
	Members []Member `json:"members,omitempty"`

	CustomRoles []GroupCustomRole `json:"customRoles,omitempty"`

	// +optional
	Scope *GroupScope `json:"scope"`
}

type Member struct {
	Email string `json:"email"`
}

type GroupCustomRole struct {
	CustomRoleRef string `json:"customRoleRef"`
}

type GroupScope struct {
	ScopeRef string `json:"scopeRef"`
}

func (g *Group) ExtractCreateGroupRequest(
	ctx context.Context, log logr.Logger,
	k8sClient client.Client, cxClient *cxsdk.ClientSet) (*cxsdk.CreateTeamGroupRequest, error) {
	usersIds, err := g.ExtractUsersIds(ctx, log, cxClient)
	if err != nil {
		return nil, err
	}

	rolesIds, err := g.ExtractRolesIds(log, k8sClient)
	if err != nil {
		return nil, err
	}

	scopeId, err := g.ExtractScopeId(k8sClient)
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
	ctx context.Context, log logr.Logger, k8sClient client.Client,
	cxClient *cxsdk.ClientSet, groupID *string) (*cxsdk.UpdateTeamGroupRequest, error) {
	usersIds, err := g.ExtractUsersIds(ctx, log, cxClient)
	if err != nil {
		return nil, err
	}

	rolesIds, err := g.ExtractRolesIds(log, k8sClient)
	if err != nil {
		return nil, err
	}

	scopeId, err := g.ExtractScopeId(k8sClient)
	if err != nil {
		return nil, err
	}

	id, err := strconv.Atoi(*groupID)
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

func (g *Group) ExtractUsersIds(ctx context.Context, log logr.Logger, cxClient *cxsdk.ClientSet) ([]*cxsdk.UserID, error) {
	users, err := cxClient.Users().List(ctx)
	if err != nil {
		return nil, err
	}

	var usersIds []*cxsdk.UserID
	for _, member := range g.Spec.Members {
		found := false
		for _, user := range users {
			if user.UserName == member.Email {
				found = true
				usersIds = append(usersIds, &cxsdk.UserID{Id: *user.ID})
				break
			}
		}
		if !found {
			log.Error(errors.New("user not found"), "user not found", "email", member.Email)
		}
	}

	return usersIds, nil
}

func (g *Group) ExtractRolesIds(log logr.Logger, k8sClient client.Client) ([]*cxsdk.RoleID, error) {
	var rolesIds []*cxsdk.RoleID
	var errs error
	for _, customRole := range g.Spec.CustomRoles {
		cr := &CustomRole{}
		if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: customRole.CustomRoleRef, Namespace: g.Namespace}, cr); err != nil {
			log.Error(err, "error on getting CustomRole")
			continue
		}
		if cr.Status.ID != nil {
			roleID, err := strconv.Atoi(*cr.Status.ID)
			if err != nil {
				errors.Join(errs, err)
			} else {
				rolesIds = append(rolesIds, &cxsdk.RoleID{Id: uint32(roleID)})
			}
		}
	}

	if errs != nil {
		return nil, errs
	}

	if len(rolesIds) == 0 {
		return nil, fmt.Errorf("no roles found for Group %s", g.Name)
	}

	return rolesIds, nil
}

func (g *Group) ExtractScopeId(k8sClient client.Client) (*string, error) {
	if g.Spec.Scope == nil {
		return nil, nil
	}

	sc := &Scope{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: g.Spec.Scope.ScopeRef, Namespace: g.Namespace}, sc); err != nil {
		return nil, err
	}

	if sc.Status.ID != nil {
		return sc.Status.ID, nil
	}

	return nil, nil
}

// GroupStatus defines the observed state of Group.
type GroupStatus struct {
	ID *string `json:"id,"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Group is the Schema for the groups API.
type Group struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSpec   `json:"spec,omitempty"`
	Status GroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupList contains a list of Group.
type GroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Group `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Group{}, &GroupList{})
}
