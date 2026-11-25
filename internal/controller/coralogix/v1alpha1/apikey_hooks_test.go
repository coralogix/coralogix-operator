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
	"strings"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	apikeysservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/api_keys_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// apiKeyGetAPI answers the Get of a key with keyInfo. A nil value is a hashed key, which has none.
type apiKeyGetAPI struct {
	mu    sync.Mutex
	value *string
	gets  int
	// broken answers 200 with a key that the SDK cannot decode, holding value.
	broken bool
}

func (a *apiKeyGetAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	a.gets++
	info := map[string]any{"id": strings.TrimPrefix(r.URL.Path, "/mgmt/openapi/5/aaa/api-keys/v3/"), "name": "key", "active": true, "hashed": a.value == nil,
		"keyPermissions": map[string]any{"presets": []any{}, "permissions": []any{}}, "owner": map[string]any{"teamId": 1}}
	if a.value != nil {
		info["value"] = *a.value
	}
	if a.broken {
		delete(info, "keyPermissions")
	}
	a.mu.Unlock()
	out, err := json.Marshal(map[string]any{"keyInfo": info})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// newAPIKeyHooks returns a reconciler whose SDK client calls api, and a fake Kubernetes client that
// the hooks read through config.GetClient.
func newAPIKeyHooks(t *testing.T, api *apiKeyGetAPI, objects ...client.Object) (*ApiKeyReconciler, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	previous := config.GetClient()
	config.InitClient(c)
	t.Cleanup(func() { config.InitClient(previous) })
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)
	cs := cxsdk.NewClientSet(cxsdk.NewConfigBuilder().WithURL(server.URL + "/mgmt/openapi/5").Build())
	return &ApiKeyReconciler{Client: cs.APIKeys(), ClientSet: cs}, c
}

func apiKeyCR() *coralogixv1alpha1.ApiKey {
	return &coralogixv1alpha1.ApiKey{
		ObjectMeta: metav1.ObjectMeta{Name: "key", Namespace: "default", UID: "uid-1"},
		Status:     coralogixv1alpha1.ApiKeyStatus{ID: ptr.To("00000000-0000-4000-8000-000000000001")},
	}
}

func apiKeySecrets(t *testing.T, c client.Client) []corev1.Secret {
	t.Helper()
	var list corev1.SecretList
	require.NoError(t, c.List(context.Background(), &list))
	return list.Items
}

// afterCreate writes the secret with the value of the Create answer, owned by the CR. A second run
// leaves the one secret as it is.
func TestApiKeyAfterCreateIsSafeToRepeat(t *testing.T) {
	r, c := newAPIKeyHooks(t, &apiKeyGetAPI{})
	cr := apiKeyCR()
	resp := &apikeysservice.CreateApiKeyResponse{KeyId: "id", Name: "key", Value: "value-1"}
	require.NoError(t, r.afterCreate(context.Background(), logr.Discard(), cr, resp))
	first := apiKeySecrets(t, c)
	require.NoError(t, r.afterCreate(context.Background(), logr.Discard(), cr, resp))
	second := apiKeySecrets(t, c)
	require.Len(t, second, 1)
	require.Equal(t, first, second)
	secret := second[0]
	require.Equal(t, "key-secret", secret.Name)
	require.Equal(t, "value-1", string(secret.Data["key-value"]))
	require.Len(t, secret.OwnerReferences, 1)
	owner := secret.OwnerReferences[0]
	require.Equal(t, "coralogix.com/v1alpha1", owner.APIVersion)
	require.Equal(t, "ApiKey", owner.Kind)
	require.Equal(t, "key", owner.Name)
	require.True(t, ptr.Deref(owner.Controller, false))
}

// afterUpdate reads the key and makes the secret hold its value: it writes a missing or changed
// secret again, keeps the other entries of the secret, and a second run changes nothing.
func TestApiKeyAfterUpdateIsSafeToRepeat(t *testing.T) {
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "key-secret", Namespace: "default"},
		Data:       map[string][]byte{"key-value": []byte("changed-by-hand"), "other": []byte("kept")},
	}
	api := &apiKeyGetAPI{value: ptr.To("value-1")}
	r, c := newAPIKeyHooks(t, api, existing)
	cr := apiKeyCR()
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, nil))
	first := apiKeySecrets(t, c)
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, nil))
	second := apiKeySecrets(t, c)
	require.Equal(t, first, second)
	require.Len(t, second, 1)
	require.Equal(t, "value-1", string(second[0].Data["key-value"]))
	require.Equal(t, "kept", string(second[0].Data["other"]))
	require.Equal(t, 2, api.gets)

	require.NoError(t, c.Delete(context.Background(), &second[0]))
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, nil))
	recreated := apiKeySecrets(t, c)
	require.Len(t, recreated, 1)
	require.Equal(t, "value-1", string(recreated[0].Data["key-value"]))
	require.Len(t, recreated[0].OwnerReferences, 1)
}

// A hashed key has no value in Get: the secret keeps the value of the Create answer.
func TestApiKeyAfterUpdateKeepsTheValueOfAHashedKey(t *testing.T) {
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "key-secret", Namespace: "default"},
		Data:       map[string][]byte{"key-value": []byte("value-of-create")},
	}
	r, c := newAPIKeyHooks(t, &apiKeyGetAPI{}, existing)
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), apiKeyCR(), nil))
	secrets := apiKeySecrets(t, c)
	require.Len(t, secrets, 1)
	require.Equal(t, "value-of-create", string(secrets[0].Data["key-value"]))
}

// A Get answer that the SDK cannot decode fails the hook. The SDK error holds the response body, so
// the hook error, which reaches the condition of the CR, must not name the value of the key.
func TestApiKeyAfterUpdateErrorHoldsNoKeyValue(t *testing.T) {
	r, _ := newAPIKeyHooks(t, &apiKeyGetAPI{value: ptr.To("secret-value-1"), broken: true})
	err := r.afterUpdate(context.Background(), logr.Discard(), apiKeyCR(), nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-value-1")
	require.Contains(t, err.Error(), "status 200")
}
