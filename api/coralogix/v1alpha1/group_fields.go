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
	"fmt"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// The handwritten parts of the generated Group kind: the spec fields that name members, a custom
// role, and a scope by reference. The beforeCreate and beforeUpdate hooks turn them into the user
// ids, the role, and the scope of the API.

// Members of the group.
type GroupMembers []Member

// Deprecated: use customRole. Kept so Groups created against Helm chart 1.0
// (spec.customRoles) keep their role after CRD upgrade. The operator uses
// customRole when set, otherwise the first customRoles entry.
type GroupCustomRoles []GroupCustomRole

// User on Coralogix.
type Member struct {
	// User's name.
	UserName string `json:"userName"`
}

// Custom role applied to the group.
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

func (g *Group) customRoleRef() *GroupCustomRole {
	if g.Spec.CustomRole != nil {
		return g.Spec.CustomRole
	}
	if len(g.Spec.CustomRoles) > 0 {
		return &g.Spec.CustomRoles[0]
	}
	return nil
}

// ExtractRoleId returns the remote id of the referenced custom role, or nil when the Group
// references no custom role.
func (g *Group) ExtractRoleId(ctx context.Context) (*int64, error) {
	ref := g.customRoleRef()
	if ref == nil {
		return nil, nil
	}
	var namespace string
	if ns := ref.ResourceRef.Namespace; ns != nil {
		namespace = *ns
	} else {
		namespace = g.Namespace
	}

	cr := &CustomRole{}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: ref.ResourceRef.Name, Namespace: namespace}, cr); err != nil {
		return nil, err
	}

	if !config.GetConfig().Selector.Matches(cr.Labels, cr.Namespace) {
		return nil, fmt.Errorf("custom role %s does not match selector", cr.Name)
	}

	if cr.Status.ID == nil {
		return nil, fmt.Errorf("ID is not populated for CustomRole %s", ref.ResourceRef.Name)
	}

	roleID, err := strconv.ParseInt(*cr.Status.ID, 10, 64)
	if err != nil {
		return nil, err
	}

	return &roleID, nil
}

// ExtractScopeId returns the remote id of the referenced scope, or nil when the Group references
// no scope.
func (g *Group) ExtractScopeId(ctx context.Context) (*string, error) {
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
	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: g.Spec.Scope.ResourceRef.Name, Namespace: namespace}, sc); err != nil {
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
