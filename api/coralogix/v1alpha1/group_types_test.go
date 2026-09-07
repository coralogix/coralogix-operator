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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomRoleRefPrefersCustomRoleOverLegacyList(t *testing.T) {
	group := &Group{
		Spec: GroupSpec{
			CustomRole: &GroupCustomRole{ResourceRef: ResourceRef{Name: "current"}},
			CustomRoles: []GroupCustomRole{
				{ResourceRef: ResourceRef{Name: "legacy"}},
			},
		},
	}
	require.Equal(t, "current", group.customRoleRef().ResourceRef.Name)
}

func TestCustomRoleRefFallsBackToFirstCustomRolesEntry(t *testing.T) {
	group := &Group{
		Spec: GroupSpec{
			CustomRoles: []GroupCustomRole{
				{ResourceRef: ResourceRef{Name: "legacy-a"}},
				{ResourceRef: ResourceRef{Name: "legacy-b"}},
			},
		},
	}
	require.Equal(t, "legacy-a", group.customRoleRef().ResourceRef.Name)
}

func TestCustomRoleRefEmpty(t *testing.T) {
	require.Nil(t, (&Group{}).customRoleRef())
}
