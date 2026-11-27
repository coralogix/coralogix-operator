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

	"github.com/go-logr/logr"

	groups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/team_groups_management_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The hooks read spec.members, spec.customRole (or spec.customRoles), and spec.scope, and write
// the user ids, the role, and the scope of the request. Each run sets whole fields, so a hook that
// runs again sends the same body.

// +kubebuilder:rbac:groups=coralogix.com,resources=customroles;scopes,verbs=get

func (r *GroupReconciler) beforeCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Group, body *groups.CreateTeamGroupRequest) error {
	userIDs, err := resolveMemberUserIDs(ctx, r.ClientSet.Users(), cr.Spec.Members)
	if err != nil {
		return err
	}
	roleID, err := cr.ExtractRoleId(ctx)
	if err != nil {
		return err
	}
	scopeID, err := cr.ExtractScopeId(ctx)
	if err != nil {
		return err
	}
	body.UserIds = userIDs
	body.RoleId = roleID
	body.Scope = &groups.V2Scope{ScopeId: scopeID}
	return nil
}

func (r *GroupReconciler) beforeUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.Group, body *groups.UpdateTeamGroupRequest) error {
	userIDs, err := resolveMemberUserIDs(ctx, r.ClientSet.Users(), cr.Spec.Members)
	if err != nil {
		return err
	}
	roleID, err := cr.ExtractRoleId(ctx)
	if err != nil {
		return err
	}
	scopeID, err := cr.ExtractScopeId(ctx)
	if err != nil {
		return err
	}
	body.UserUpdates = &groups.UserUpdates{
		Operation: &groups.UserUpdatesOperation{
			OperationType: "set",
			Set:           &groups.UserIdList{UserIds: userIDs},
		},
	}
	// Without a custom role reference, the remote role stays: role id 0 does not exist, and the
	// API rejects it with 403.
	body.RoleUpdate = nil
	if roleID != nil {
		body.RoleUpdate = &groups.RoleUpdate{
			Action: &groups.RoleUpdateAction{
				ActionType: "set_role_id",
				SetRoleId:  &groups.SetRoleId{Value: roleID},
			},
		}
	}
	// Without a scope reference, the scope update has no value, and the API clears the scope.
	body.ScopeUpdate = &groups.ScopeUpdate{
		Action: &groups.ScopeUpdateAction{
			ActionType: "set_scope_id",
			SetScopeId: &groups.SetScopeId{Value: scopeID},
		},
	}
	return nil
}
