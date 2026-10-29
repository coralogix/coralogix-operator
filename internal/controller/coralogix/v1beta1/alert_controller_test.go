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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	oapisdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

func testAlert(policy *string) *coralogixv1beta1.Alert {
	return &coralogixv1beta1.Alert{
		ObjectMeta: metav1.ObjectMeta{Name: "test-alert", Namespace: "default"},
		Spec: coralogixv1beta1.AlertSpec{
			Name:         "test-alert",
			Priority:     coralogixv1beta1.AlertPriorityP5,
			AccessPolicy: policy,
			TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
				LogsImmediate: &coralogixv1beta1.LogsImmediate{},
			},
		},
		Status: coralogixv1beta1.AlertStatus{ID: ptr.To("alert-id")},
	}
}

func testAlertReconciler(t *testing.T, server *httptest.Server) *AlertReconciler {
	t.Helper()
	return &AlertReconciler{
		ClientSet: oapisdk.NewClientSet(oapisdk.NewConfigBuilder().
			WithURL(server.URL).
			WithAPIKey("test").
			Build()),
	}
}

func initAlertClient(t *testing.T, alert *coralogixv1beta1.Alert) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1beta1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&coralogixv1beta1.Alert{}).
		WithObjects(alert).
		Build()
	originalClient := config.GetClient()
	originalScheme := config.GetScheme()
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.InitScheme(originalScheme)
	})
	config.InitClient(fakeClient)
	config.InitScheme(scheme)
	return fakeClient
}

func TestHandleUpdateSendsAccessPolicyAction(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	for _, tc := range []struct {
		name   string
		policy *string
		assert func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "nil omits the policy",
			policy: nil,
			assert: func(t *testing.T, body map[string]interface{}) {
				_, found := body["accessPolicy"]
				require.False(t, found, "body must not contain accessPolicy: %v", body)
			},
		},
		{
			name:   "empty deletes the policy",
			policy: ptr.To(""),
			assert: func(t *testing.T, body map[string]interface{}) {
				require.Equal(t, map[string]interface{}{"delete": map[string]interface{}{}}, body["accessPolicy"])
			},
		},
		{
			name:   "value upserts the policy",
			policy: ptr.To(policy),
			assert: func(t *testing.T, body map[string]interface{}) {
				require.Equal(t, map[string]interface{}{"upsert": policy}, body["accessPolicy"])
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var captured []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				mu.Lock()
				captured = body
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"alertDef":{"id":"alert-id"}}`))
			}))
			t.Cleanup(server.Close)

			alert := testAlert(tc.policy)
			initAlertClient(t, alert)

			reconciler := testAlertReconciler(t, server)
			require.NoError(t, reconciler.HandleUpdate(context.Background(), logr.Discard(), alert))

			mu.Lock()
			body := captured
			mu.Unlock()
			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(body, &parsed))
			tc.assert(t, parsed)
		})
	}
}

func TestHandleCreationSendsAccessPolicy(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	for _, tc := range []struct {
		name   string
		policy *string
		assert func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "nil omits the policy",
			policy: nil,
			assert: func(t *testing.T, body map[string]interface{}) {
				_, found := body["accessPolicy"]
				require.False(t, found, "body must not contain accessPolicy: %v", body)
			},
		},
		{
			name:   "empty omits the policy",
			policy: ptr.To(""),
			assert: func(t *testing.T, body map[string]interface{}) {
				_, found := body["accessPolicy"]
				require.False(t, found, "body must not contain accessPolicy: %v", body)
			},
		},
		{
			name:   "value sends the policy",
			policy: ptr.To(policy),
			assert: func(t *testing.T, body map[string]interface{}) {
				require.Equal(t, policy, body["accessPolicy"])
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var captured []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				mu.Lock()
				captured = body
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"alertDef":{"id":"alert-id"}}`))
			}))
			t.Cleanup(server.Close)

			alert := testAlert(tc.policy)
			initAlertClient(t, alert)

			reconciler := testAlertReconciler(t, server)
			require.NoError(t, reconciler.HandleCreation(context.Background(), logr.Discard(), alert))

			mu.Lock()
			body := captured
			mu.Unlock()
			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(body, &parsed))
			tc.assert(t, parsed)
			require.NotNil(t, alert.Status.ID)
			require.Equal(t, "alert-id", *alert.Status.ID)
		})
	}
}

// reconcileAlert drives the shared reconciler with the Alert controller against server.
func reconcileAlert(t *testing.T, server *httptest.Server, alert *coralogixv1beta1.Alert) error {
	t.Helper()
	originalSelector := config.GetConfig().Selector
	t.Cleanup(func() { config.GetConfig().Selector = originalSelector })
	config.GetConfig().Selector = config.Selector{}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: alert.Name, Namespace: alert.Namespace}}
	_, err := testAlertReconciler(t, server).Reconcile(context.Background(), req)
	return err
}

func TestReconcileRecreatesAlertWithAccessPolicyAfterRemoteNotFound(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	var mu sync.Mutex
	var created []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			// The API answers 404 only when the alert is absent; a denying policy answers 403.
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found: The requested Alert was not found"))
		case http.MethodPost:
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			mu.Lock()
			created = body
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"alertDef":{"id":"recreated-id"}}`))
		default:
			t.Errorf("unexpected %s request", r.Method)
		}
	}))
	t.Cleanup(server.Close)

	alert := testAlert(ptr.To(policy))
	fakeClient := initAlertClient(t, alert)
	key := client.ObjectKeyFromObject(alert)

	// Not checked: the condition write after dropping status.id conflicts, so this pass ends in a requeue.
	_ = reconcileAlert(t, server, alert)
	stored := &coralogixv1beta1.Alert{}
	require.NoError(t, fakeClient.Get(context.Background(), key, stored))
	require.Nil(t, stored.Status.ID, "a not-found replace must clear status.id so the alert is recreated")

	require.NoError(t, reconcileAlert(t, server, alert))
	require.NoError(t, fakeClient.Get(context.Background(), key, stored))
	require.NotNil(t, stored.Status.ID)
	require.Equal(t, "recreated-id", *stored.Status.ID)

	mu.Lock()
	body := created
	mu.Unlock()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.Equal(t, policy, parsed["accessPolicy"], "the recreated alert must carry the policy")
}

func TestReconcileKeepsIDWhenAccessPolicyDeniesOperator(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected %s request", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Permission Denied"))
	}))
	t.Cleanup(server.Close)

	alert := testAlert(ptr.To(policy))
	fakeClient := initAlertClient(t, alert)

	err := reconcileAlert(t, server, alert)
	require.ErrorContains(t, err, "Permission Denied")
	stored := &coralogixv1beta1.Alert{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(alert), stored))
	require.NotNil(t, stored.Status.ID, "a denied replace must not clear status.id")
	require.Equal(t, "alert-id", *stored.Status.ID)
	require.Equal(t, utils.ReasonRemoteUpdateFailed, utils.GetReasonForRemoteSyncedCondition(stored.Status.Conditions))
	require.Equal(t, "RemoteUnsynced", stored.Status.PrintableStatus)
}
