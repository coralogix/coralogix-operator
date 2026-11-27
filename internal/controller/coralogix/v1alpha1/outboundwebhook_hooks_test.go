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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	webhooks "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/outgoing_webhooks_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// webhookHookServer answers each Get of a webhook with the given status and body, and counts them.
func webhookHookServer(t *testing.T, status int, body string) (*OutboundWebhookReconciler, *atomic.Int32) {
	t.Helper()
	gets := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/integrations/webhooks/v1/webhook-id") {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		gets.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	cs := cxsdk.NewClientSet(cxsdk.NewConfigBuilder().WithURL(server.URL + "/mgmt/openapi/5").Build())
	return &OutboundWebhookReconciler{Client: cs.Webhooks(), ClientSet: cs}, gets
}

const webhookGetAnswer = `{"webhook": {"id": "webhook-id", "type": "GENERIC", "name": "w", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z", "externalId": 42, "genericWebhook": {"uuid": "u", "method": "POST"}}}`

func webhookWithID() *coralogixv1alpha1.OutboundWebhook {
	return &coralogixv1alpha1.OutboundWebhook{Status: coralogixv1alpha1.OutboundWebhookStatus{ID: ptr.To("webhook-id")}}
}

// The UUID hooks fill only a UUID that the CR leaves out. A second run on the same body changes
// nothing, and a UUID of the CR stays as it is.
func TestOutboundWebhookUUIDHooksAreSafeToRepeat(t *testing.T) {
	r := &OutboundWebhookReconciler{}
	body := &webhooks.CreateOutgoingWebhookRequest{Data: webhooks.OutgoingWebhookInputData{
		GenericWebhook: &webhooks.GenericWebhookConfig{},
		SendLog:        &webhooks.SendLogConfig{Uuid: ptr.To("from-the-cr")},
	}}
	for range 2 {
		require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), nil, body))
	}
	first := ptr.Deref(body.Data.GenericWebhook.Uuid, "")
	require.Len(t, first, 36)
	require.Equal(t, "from-the-cr", ptr.Deref(body.Data.SendLog.Uuid, ""))
	require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), nil, body))
	require.Equal(t, first, ptr.Deref(body.Data.GenericWebhook.Uuid, ""))

	// An empty UUID is a 400 in the API, so it is filled as a missing one.
	update := &webhooks.UpdateOutgoingWebhookRequest{Data: webhooks.OutgoingWebhookInputData{SendLog: &webhooks.SendLogConfig{Uuid: ptr.To("")}}}
	require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), nil, update))
	require.Len(t, ptr.Deref(update.Data.SendLog.Uuid, ""), 36)

	// A type without a UUID gets none.
	other := &webhooks.UpdateOutgoingWebhookRequest{Data: webhooks.OutgoingWebhookInputData{PagerDuty: &webhooks.PagerDutyConfig{ServiceKey: "k"}}}
	require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), nil, other))
	require.Nil(t, other.Data.GenericWebhook)
	require.Nil(t, other.Data.SendLog)
}

// afterCreate reads the external id. Running it twice sets the same id.
func TestOutboundWebhookAfterCreateReadsTheExternalID(t *testing.T) {
	r, gets := webhookHookServer(t, http.StatusOK, webhookGetAnswer)
	cr := webhookWithID()
	for range 2 {
		require.NoError(t, r.afterCreate(context.Background(), logr.Discard(), cr, &webhooks.CreateOutgoingWebhookResponse{Id: "webhook-id"}))
		require.Equal(t, "42", ptr.Deref(cr.Status.ExternalId, ""))
	}
	require.EqualValues(t, 2, gets.Load())
}

// afterUpdate reads the external id only when the status has none, so a resync sends no Get.
func TestOutboundWebhookAfterUpdateRepairsAMissingExternalID(t *testing.T) {
	r, gets := webhookHookServer(t, http.StatusOK, webhookGetAnswer)
	cr := webhookWithID()
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, map[string]interface{}{}))
	require.Equal(t, "42", ptr.Deref(cr.Status.ExternalId, ""))
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, map[string]interface{}{}))
	require.EqualValues(t, 1, gets.Load())
}

// A failed Get names the HTTP status, never the body, which can hold a credential.
func TestOutboundWebhookExternalIDErrorHasNoBody(t *testing.T) {
	r, _ := webhookHookServer(t, http.StatusInternalServerError, `{"code": 500, "message": "fake-secret-in-the-body"}`)
	cr := webhookWithID()
	err := r.afterCreate(context.Background(), logr.Discard(), cr, &webhooks.CreateOutgoingWebhookResponse{Id: "webhook-id"})
	require.ErrorContains(t, err, "HTTP 500")
	require.NotContains(t, err.Error(), "fake-secret-in-the-body")
	require.Nil(t, cr.Status.ExternalId)

	require.ErrorContains(t, (&OutboundWebhookReconciler{}).readWebhookExternalID(context.Background(), &coralogixv1alpha1.OutboundWebhook{}), "no status id")
}
