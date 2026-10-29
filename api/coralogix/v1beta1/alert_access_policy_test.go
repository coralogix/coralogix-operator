// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestAlertSpecAccessPolicyMapping(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	t.Run("nil leaves the remote policy unchanged", func(t *testing.T) {
		spec := &AlertSpec{}
		require.Nil(t, spec.ExtractCreateAccessPolicy())
		require.Nil(t, spec.ExtractAccessPolicyAction())
	})

	t.Run("empty string deletes the remote policy", func(t *testing.T) {
		spec := &AlertSpec{AccessPolicy: ptr.To("")}
		require.Nil(t, spec.ExtractCreateAccessPolicy())
		action := spec.ExtractAccessPolicyAction()
		require.NotNil(t, action)
		require.Nil(t, action.Upsert)
		require.NotNil(t, action.Delete)
		raw, err := json.Marshal(action)
		require.NoError(t, err)
		require.JSONEq(t, `{"delete":{}}`, string(raw))
	})

	t.Run("value is upserted", func(t *testing.T) {
		spec := &AlertSpec{AccessPolicy: ptr.To(policy)}
		require.Equal(t, policy, *spec.ExtractCreateAccessPolicy())
		action := spec.ExtractAccessPolicyAction()
		require.NotNil(t, action)
		require.Nil(t, action.Delete)
		require.NotNil(t, action.Upsert)
		require.Equal(t, policy, *action.Upsert)
		raw, err := json.Marshal(action)
		require.NoError(t, err)
		want, err := json.Marshal(map[string]string{"upsert": policy})
		require.NoError(t, err)
		require.JSONEq(t, string(want), string(raw))
	})
}

func TestAlertSpecAccessPolicyMayDenyOperator(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	for _, tc := range []struct {
		name        string
		spec        *AlertSpec
		managed     bool
		annotations map[string]string
		want        bool
	}{
		{name: "absent without tombstone or annotation", spec: &AlertSpec{}, managed: false, annotations: nil, want: false},
		{name: "absent with tombstone", spec: &AlertSpec{}, managed: true, annotations: nil, want: true},
		{name: "absent with annotation", spec: &AlertSpec{}, managed: false, annotations: map[string]string{AccessPolicyManagedAnnotation: policy}, want: true},
		{name: "empty without tombstone or annotation", spec: &AlertSpec{AccessPolicy: ptr.To("")}, managed: false, annotations: nil, want: false},
		{name: "empty with tombstone", spec: &AlertSpec{AccessPolicy: ptr.To("")}, managed: true, annotations: nil, want: true},
		{name: "empty with annotation (delete in flight, still managed)", spec: &AlertSpec{AccessPolicy: ptr.To("")}, managed: false, annotations: map[string]string{AccessPolicyManagedAnnotation: policy}, want: true},
		{name: "set without tombstone or annotation", spec: &AlertSpec{AccessPolicy: ptr.To(policy)}, managed: false, annotations: nil, want: true},
		{name: "set with tombstone", spec: &AlertSpec{AccessPolicy: ptr.To(policy)}, managed: true, annotations: nil, want: true},
		{name: "nil annotations map", spec: &AlertSpec{}, managed: false, annotations: nil, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.spec.AccessPolicyMayDenyOperator(tc.managed, tc.annotations))
		})
	}
}
