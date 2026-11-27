// Copyright 2026 Coralogix Ltd.
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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func groupWithReferences(t *testing.T) *coralogixv1alpha1.Group {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))
	role := &coralogixv1alpha1.CustomRole{
		ObjectMeta: metav1.ObjectMeta{Name: "role", Namespace: "default"},
		Status:     coralogixv1alpha1.CustomRoleStatus{ID: ptr.To("7001")},
	}
	scope := &coralogixv1alpha1.Scope{
		ObjectMeta: metav1.ObjectMeta{Name: "scope", Namespace: "default"},
		Status:     coralogixv1alpha1.ScopeStatus{ID: ptr.To("scope-id")},
	}
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(role, scope).Build())
	return &coralogixv1alpha1.Group{
		ObjectMeta: metav1.ObjectMeta{Name: "group", Namespace: "default"},
		Spec: coralogixv1alpha1.GroupSpec{
			Name:       "group",
			CustomRole: &coralogixv1alpha1.GroupCustomRole{ResourceRef: coralogixv1alpha1.ResourceRef{Name: "role"}},
			Scope:      &coralogixv1alpha1.GroupScope{ResourceRef: coralogixv1alpha1.ResourceRef{Name: "scope"}},
			Members:    coralogixv1alpha1.GroupMembers{{UserName: "Alice@Example.com"}},
		},
	}
}

func groupReconcilerWithUsers(t *testing.T) *GroupReconciler {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSearchUsers(t, w, 0, userJSON("id-1", "alice@example.com"))
	}))
	t.Cleanup(server.Close)
	return &GroupReconciler{ClientSet: openapicxsdk.NewClientSet(openapicxsdk.NewConfigBuilder().WithURL(server.URL).WithAPIKey("test").Build())}
}

func marshalGroupBody(t *testing.T, body any) string {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	return string(raw)
}

// The hooks run on every Create attempt and every reconcile. Running one twice on one body leaves
// the same body.
func TestGroupHooksAreSafeToRepeat(t *testing.T) {
	cr := groupWithReferences(t)
	r := groupReconcilerWithUsers(t)
	ctx := context.Background()

	create, err := cr.CreateRequest(ctx)
	require.NoError(t, err)
	require.NoError(t, r.beforeCreate(ctx, logr.Discard(), cr, create))
	want := `{"name":"group","roleId":7001,"scope":{"scopeId":"scope-id"},"userIds":["id-1"]}`
	require.JSONEq(t, want, marshalGroupBody(t, create))
	require.NoError(t, r.beforeCreate(ctx, logr.Discard(), cr, create))
	require.JSONEq(t, want, marshalGroupBody(t, create), "a second run must not add to the body")

	update, err := cr.UpdateRequest(ctx)
	require.NoError(t, err)
	require.NoError(t, r.beforeUpdate(ctx, logr.Discard(), cr, update))
	want = `{"name":"group",` +
		`"roleUpdate":{"action":{"actionType":"set_role_id","setRoleId":{"value":7001}}},` +
		`"scopeUpdate":{"action":{"actionType":"set_scope_id","setScopeId":{"value":"scope-id"}}},` +
		`"userUpdates":{"operation":{"operationType":"set","set":{"userIds":["id-1"]}}}}`
	require.JSONEq(t, want, marshalGroupBody(t, update))
	require.NoError(t, r.beforeUpdate(ctx, logr.Discard(), cr, update))
	require.JSONEq(t, want, marshalGroupBody(t, update), "a second run must not add to the body")
}

// Without references, Create sends an empty scope, and no role id: role id 0 does not exist, and
// the API rejects it with 403. Update sends no role update, so the remote role stays, a scope
// update with no value, which clears the scope, and an empty member set.
func TestGroupHooksWithoutReferences(t *testing.T) {
	cr := &coralogixv1alpha1.Group{Spec: coralogixv1alpha1.GroupSpec{Name: "group"}}
	r := groupReconcilerWithUsers(t)
	ctx := context.Background()

	create, err := cr.CreateRequest(ctx)
	require.NoError(t, err)
	require.NoError(t, r.beforeCreate(ctx, logr.Discard(), cr, create))
	require.JSONEq(t, `{"name":"group","scope":{}}`, marshalGroupBody(t, create))

	update, err := cr.UpdateRequest(ctx)
	require.NoError(t, err)
	require.NoError(t, r.beforeUpdate(ctx, logr.Discard(), cr, update))
	require.JSONEq(t, `{"name":"group","scopeUpdate":{"action":{"actionType":"set_scope_id","setScopeId":{}}},`+
		`"userUpdates":{"operation":{"operationType":"set","set":{}}}}`, marshalGroupBody(t, update))
}

// A role that is gone from the cluster, or has no status id, stops the hook before any request.
func TestGroupHooksFailForABrokenReference(t *testing.T) {
	cr := groupWithReferences(t)
	r := groupReconcilerWithUsers(t)
	ctx := context.Background()

	cr.Spec.CustomRole.ResourceRef.Name = "missing"
	create, err := cr.CreateRequest(ctx)
	require.NoError(t, err)
	require.Error(t, r.beforeCreate(ctx, logr.Discard(), cr, create))

	cr = groupWithReferences(t)
	cr.Spec.Scope.ResourceRef.Namespace = ptr.To("elsewhere")
	update, err := cr.UpdateRequest(ctx)
	require.NoError(t, err)
	require.Error(t, r.beforeUpdate(ctx, logr.Discard(), cr, update))
}
