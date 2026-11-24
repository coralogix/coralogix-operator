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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

const (
	identityTestUID     = types.UID("7d1c2b9e-5a4f-4e0b-9c1d-2f3a4b5c6d7e")
	existingRemoteID    = "9a5fea05-0000-4000-8000-000000000001"
	unrelatedRemoteID   = "11111111-0000-4000-8000-000000000002"
	secondMatchRemoteID = "9f4dd591-0000-4000-8000-000000000003"
)

// identityAlert returns an Alert that lost its status.id: it has a UID but no remote id.
func identityAlert() *coralogixv1beta1.Alert {
	alert := testAlert(nil)
	alert.UID = identityTestUID
	alert.Status.ID = nil
	return alert
}

// remoteAlertJSON renders one alert definition of a ListAlertDefs response.
func remoteAlertJSON(t *testing.T, id string, labels map[string]string) map[string]interface{} {
	t.Helper()
	return map[string]interface{}{
		"id": id,
		"alertDefProperties": map[string]interface{}{
			"name":         "test-alert",
			"entityLabels": labels,
		},
	}
}

// fakeAlertsAPI serves ListAlertDefs from pages, and records Create and Replace bodies.
type fakeAlertsAPI struct {
	t      *testing.T
	pages  [][]map[string]interface{}
	mu     sync.Mutex
	posts  [][]byte
	puts   [][]byte
	lists  int
	listOK bool
}

func newFakeAlertsAPI(t *testing.T, pages ...[]map[string]interface{}) (*fakeAlertsAPI, *httptest.Server) {
	t.Helper()
	api := &fakeAlertsAPI{t: t, pages: pages, listOK: true}
	server := httptest.NewServer(http.HandlerFunc(api.serve))
	t.Cleanup(server.Close)
	return api, server
}

func (f *fakeAlertsAPI) serve(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	assert.NoError(f.t, err)
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		f.lists++
		if !f.listOK {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("list failed"))
			return
		}
		page := 0
		if token := r.URL.Query().Get("pagination[pageToken]"); token != "" {
			_, err := fmt.Sscanf(token, "page-%d", &page)
			assert.NoError(f.t, err)
		}
		resp := map[string]interface{}{"alertDefs": []map[string]interface{}{}}
		if page < len(f.pages) {
			resp["alertDefs"] = f.pages[page]
		}
		if page+1 < len(f.pages) {
			resp["pagination"] = map[string]interface{}{"nextPageToken": fmt.Sprintf("page-%d", page+1)}
		}
		assert.NoError(f.t, json.NewEncoder(w).Encode(resp))
	case http.MethodPost:
		f.posts = append(f.posts, body)
		_, _ = w.Write([]byte(`{"alertDef":{"id":"created-id"}}`))
	case http.MethodPut:
		f.puts = append(f.puts, body)
		_, _ = w.Write([]byte(`{"alertDef":{"id":"alert-id"}}`))
	default:
		f.t.Errorf("unexpected %s request", r.Method)
	}
}

func (f *fakeAlertsAPI) recorded() (posts, puts [][]byte, lists int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts, f.puts, f.lists
}

func entityLabelsOf(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &parsed))
	props, ok := parsed["alertDefProperties"].(map[string]interface{})
	require.True(t, ok, "body has no alertDefProperties: %s", body)
	labels, _ := props["entityLabels"].(map[string]interface{})
	return labels
}

func ownedLabels(alert *coralogixv1beta1.Alert) map[string]string {
	return map[string]string{
		utils.RemoteIdentityUIDLabelKey:       string(alert.UID),
		utils.RemoteIdentityNamespaceLabelKey: alert.Namespace,
		utils.RemoteIdentityNameLabelKey:      alert.Name,
	}
}

// A lost status.id must not create a second remote alert: the operator adopts the remote
// alert that carries the identity label of this Alert.
func TestReconcileAdoptsRemoteAlertAfterStatusIDIsLost(t *testing.T) {
	alert := identityAlert()
	api, server := newFakeAlertsAPI(t, []map[string]interface{}{
		remoteAlertJSON(t, unrelatedRemoteID, map[string]string{utils.RemoteIdentityUIDLabelKey: "another-uid"}),
		remoteAlertJSON(t, existingRemoteID, ownedLabels(alert)),
	})
	fakeClient := initAlertClient(t, alert)

	require.NoError(t, reconcileAlert(t, server, alert))

	posts, _, lists := api.recorded()
	require.Empty(t, posts, "a remote alert with this Alert's identity exists; it must not be created again")
	require.Equal(t, 1, lists)
	stored := &coralogixv1beta1.Alert{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(alert), stored))
	require.NotNil(t, stored.Status.ID)
	require.Equal(t, existingRemoteID, *stored.Status.ID)
	require.Contains(t, stored.Finalizers, "alert.coralogix.com/finalizer")
	require.Equal(t, "RemoteSynced", stored.Status.PrintableStatus)
}

func TestHandleCreationAdoptsMatchOnALaterPage(t *testing.T) {
	alert := identityAlert()
	api, server := newFakeAlertsAPI(t,
		[]map[string]interface{}{remoteAlertJSON(t, unrelatedRemoteID, nil)},
		[]map[string]interface{}{remoteAlertJSON(t, existingRemoteID, ownedLabels(alert))},
	)
	initAlertClient(t, alert)

	require.NoError(t, testAlertReconciler(t, server).HandleCreation(context.Background(), logr.Discard(), alert))

	posts, _, lists := api.recorded()
	require.Empty(t, posts)
	require.Equal(t, 2, lists)
	require.NotNil(t, alert.Status.ID)
	require.Equal(t, existingRemoteID, *alert.Status.ID)
}

// The API does not apply the entity label filter as the SDK encodes it, so the operator must
// match the label itself: a remote alert with the same name and no identity label is not adopted.
func TestHandleCreationCreatesWhenNoRemoteAlertCarriesTheIdentity(t *testing.T) {
	alert := identityAlert()
	alert.Spec.EntityLabels = map[string]string{"team": "platform"}
	api, server := newFakeAlertsAPI(t, []map[string]interface{}{
		remoteAlertJSON(t, unrelatedRemoteID, map[string]string{"team": "platform"}),
	})
	initAlertClient(t, alert)

	require.NoError(t, testAlertReconciler(t, server).HandleCreation(context.Background(), logr.Discard(), alert))

	posts, _, _ := api.recorded()
	require.Len(t, posts, 1)
	labels := entityLabelsOf(t, posts[0])
	require.Equal(t, "platform", labels["team"])
	require.Equal(t, string(identityTestUID), labels[utils.RemoteIdentityUIDLabelKey])
	require.Equal(t, alert.Namespace, labels[utils.RemoteIdentityNamespaceLabelKey])
	require.Equal(t, alert.Name, labels[utils.RemoteIdentityNameLabelKey])
	require.Equal(t, map[string]string{"team": "platform"}, alert.Spec.EntityLabels, "the spec must not be changed")
	require.NotNil(t, alert.Status.ID)
	require.Equal(t, "created-id", *alert.Status.ID)
}

// Two remote alerts with one identity can exist only through an earlier bug or a manual copy.
// The operator adopts one, the same one each time, and creates nothing.
func TestHandleCreationAdoptsOneOfSeveralMatches(t *testing.T) {
	alert := identityAlert()
	api, server := newFakeAlertsAPI(t, []map[string]interface{}{
		remoteAlertJSON(t, secondMatchRemoteID, ownedLabels(alert)),
		remoteAlertJSON(t, existingRemoteID, ownedLabels(alert)),
	})
	initAlertClient(t, alert)

	require.NoError(t, testAlertReconciler(t, server).HandleCreation(context.Background(), logr.Discard(), alert))

	posts, _, _ := api.recorded()
	require.Empty(t, posts)
	require.NotNil(t, alert.Status.ID)
	require.Equal(t, existingRemoteID, *alert.Status.ID, "the lowest id is adopted so the choice is stable")
}

func TestHandleCreationDoesNotCreateWhenTheLookupFails(t *testing.T) {
	alert := identityAlert()
	api, server := newFakeAlertsAPI(t)
	api.listOK = false
	initAlertClient(t, alert)

	err := testAlertReconciler(t, server).HandleCreation(context.Background(), logr.Discard(), alert)
	require.ErrorContains(t, err, "list failed")

	posts, _, _ := api.recorded()
	require.Empty(t, posts, "a failed lookup must not create a remote alert blindly")
	require.Nil(t, alert.Status.ID)
}

// Update replaces the whole alert, so it must send the identity labels too: otherwise the next
// Update removes them, and alerts created before this change get them on their next Update.
func TestHandleUpdateSendsIdentityLabels(t *testing.T) {
	alert := identityAlert()
	alert.Status.ID = &[]string{existingRemoteID}[0]
	alert.Spec.EntityLabels = map[string]string{
		"team":                          "platform",
		utils.RemoteIdentityUIDLabelKey: "spoofed",
	}
	api, server := newFakeAlertsAPI(t)
	initAlertClient(t, alert)

	require.NoError(t, testAlertReconciler(t, server).HandleUpdate(context.Background(), logr.Discard(), alert))

	_, puts, lists := api.recorded()
	require.Zero(t, lists, "update must not list alerts")
	require.Len(t, puts, 1)
	labels := entityLabelsOf(t, puts[0])
	require.Equal(t, "platform", labels["team"])
	require.Equal(t, string(identityTestUID), labels[utils.RemoteIdentityUIDLabelKey], "the operator's identity wins over a spec label")
	require.Equal(t, alert.Namespace, labels[utils.RemoteIdentityNamespaceLabelKey])
	require.Equal(t, alert.Name, labels[utils.RemoteIdentityNameLabelKey])
	require.Equal(t, "spoofed", alert.Spec.EntityLabels[utils.RemoteIdentityUIDLabelKey], "the spec must not be changed")
}
