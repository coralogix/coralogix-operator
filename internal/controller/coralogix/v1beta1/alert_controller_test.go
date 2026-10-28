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
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	oapisdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

type failStatusClient struct {
	client.Client
	err error
}

type failStatusWriter struct {
	client.SubResourceWriter
	parent *failStatusClient
}

func (c *failStatusClient) Status() client.SubResourceWriter {
	return failStatusWriter{SubResourceWriter: c.Client.Status(), parent: c}
}

func (w failStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if w.parent.err != nil {
		err := w.parent.err
		w.parent.err = nil
		return err
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

func testAlert(policy *string, managed bool) *coralogixv1beta1.Alert {
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
		Status: coralogixv1beta1.AlertStatus{ID: ptr.To("alert-id"), AccessPolicyManaged: managed},
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

func initAlertClient(t *testing.T, alert *coralogixv1beta1.Alert, statusClient func(client.Client) client.Client) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1beta1.AddToScheme(scheme))
	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&coralogixv1beta1.Alert{}).
		WithObjects(alert).
		Build()
	if statusClient != nil {
		wrapped := statusClient(baseClient)
		originalClient := config.GetClient()
		t.Cleanup(func() { config.InitClient(originalClient) })
		config.InitClient(wrapped)
		return baseClient
	}
	originalClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(originalClient) })
	config.InitClient(baseClient)
	return baseClient
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

			alert := testAlert(tc.policy, false)
			initAlertClient(t, alert, nil)

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

func TestHandleUpdateKeepsIDWhenPolicyManagedAlertIsNotFound(t *testing.T) {
	newNotFoundServer := func() *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found: No alert with the given id"))
		}))
		return server
	}

	t.Run("absent field with tombstone is not a not-found", func(t *testing.T) {
		server := newNotFoundServer()
		t.Cleanup(server.Close)

		alert := testAlert(nil, true)
		initAlertClient(t, alert, nil)

		reconciler := testAlertReconciler(t, server)
		err := reconciler.HandleUpdate(context.Background(), logr.Discard(), alert)
		require.Error(t, err)
		require.False(t, oapisdk.IsNotFound(err))
		require.NotEqual(t, codes.NotFound, cxsdk.Code(err))
		require.Contains(t, err.Error(), "not recreating it")
		require.NotNil(t, alert.Status.ID)
		require.Equal(t, "alert-id", *alert.Status.ID)
	})

	t.Run("absent field without tombstone stays a not-found", func(t *testing.T) {
		server := newNotFoundServer()
		t.Cleanup(server.Close)

		alert := testAlert(nil, false)
		initAlertClient(t, alert, nil)

		reconciler := testAlertReconciler(t, server)
		err := reconciler.HandleUpdate(context.Background(), logr.Discard(), alert)
		require.Error(t, err)
		require.True(t, oapisdk.IsNotFound(err))
	})
}

func TestHandleUpdateRecordsAccessPolicyBeforeSending(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	t.Run("tombstone is persisted before the request", func(t *testing.T) {
		var baseClient client.Client
		var mu sync.Mutex
		requestSent := false
		markRequestSent := func() {
			mu.Lock()
			requestSent = true
			mu.Unlock()
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markRequestSent()
			stored := &coralogixv1beta1.Alert{}
			assert.NoError(t, baseClient.Get(context.Background(), client.ObjectKey{Name: "test-alert", Namespace: "default"}, stored))
			assert.True(t, stored.Status.AccessPolicyManaged, "tombstone must be persisted before the upsert is sent")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"alertDef":{"id":"alert-id"}}`))
		}))
		t.Cleanup(server.Close)

		alert := testAlert(ptr.To(policy), false)
		baseClient = initAlertClient(t, alert, nil)

		reconciler := testAlertReconciler(t, server)
		require.NoError(t, reconciler.HandleUpdate(context.Background(), logr.Discard(), alert))
		mu.Lock()
		sent := requestSent
		mu.Unlock()
		require.True(t, sent)
	})

	t.Run("status update error sends no request", func(t *testing.T) {
		var mu sync.Mutex
		requestSent := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestSent = true
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(server.Close)

		alert := testAlert(ptr.To(policy), false)
		initAlertClient(t, alert, func(base client.Client) client.Client {
			return &failStatusClient{Client: base, err: context.DeadlineExceeded}
		})

		reconciler := testAlertReconciler(t, server)
		err := reconciler.HandleUpdate(context.Background(), logr.Discard(), alert)
		require.ErrorContains(t, err, "recording the alert access policy")
		mu.Lock()
		sent := requestSent
		mu.Unlock()
		require.False(t, sent)
	})
}

func TestHandleUpdateClearsAccessPolicyManagedAfterDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"alertDef":{"id":"alert-id"}}`))
	}))
	t.Cleanup(server.Close)

	alert := testAlert(ptr.To(""), true)
	alert.Annotations = map[string]string{coralogixv1beta1.AccessPolicyManagedAnnotation: `{"version":"2025-01-01"}`}
	initAlertClient(t, alert, nil)

	reconciler := testAlertReconciler(t, server)
	require.NoError(t, reconciler.HandleUpdate(context.Background(), logr.Discard(), alert))
	require.False(t, alert.Status.AccessPolicyManaged)
	require.NotContains(t, alert.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation,
		"the annotation must be cleared together with the status tombstone after a delete")
}

func TestHandleCreationSendsAccessPolicy(t *testing.T) {
	policy := `{"version":"2025-01-01","default":{"permissions":{"alerts:ReadConfig":"grant"}},"rules":[]}`

	for _, tc := range []struct {
		name        string
		policy      *string
		wantManaged bool
		assert      func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "nil omits the policy",
			policy:      nil,
			wantManaged: false,
			assert: func(t *testing.T, body map[string]interface{}) {
				_, found := body["accessPolicy"]
				require.False(t, found, "body must not contain accessPolicy: %v", body)
			},
		},
		{
			name:        "empty omits the policy",
			policy:      ptr.To(""),
			wantManaged: false,
			assert: func(t *testing.T, body map[string]interface{}) {
				_, found := body["accessPolicy"]
				require.False(t, found, "body must not contain accessPolicy: %v", body)
			},
		},
		{
			name:        "value sends the policy",
			policy:      ptr.To(policy),
			wantManaged: true,
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

			alert := testAlert(tc.policy, false)
			if tc.wantManaged {
				alert.Annotations = map[string]string{}
			}
			initAlertClient(t, alert, nil)

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
			require.Equal(t, tc.wantManaged, alert.Status.AccessPolicyManaged)
			if tc.wantManaged {
				require.Contains(t, alert.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation,
					"creation with a policy must record the annotation")
			} else {
				require.NotContains(t, alert.Annotations, coralogixv1beta1.AccessPolicyManagedAnnotation)
			}
		})
	}
}
