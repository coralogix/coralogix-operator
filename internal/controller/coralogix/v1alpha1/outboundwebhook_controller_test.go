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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	webhooks "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/outgoing_webhooks_service"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOutboundWebhookUpdateSendsRotatedSecret(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-webhook-values", Namespace: "test"},
		Data:       map[string][]byte{"service-key": []byte("fake-initial-service-key-value")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	previousClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(previousClient) })
	config.InitClient(c)

	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		bodies = append(bodies, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	sdkConfig := webhooks.NewConfiguration()
	sdkConfig.Servers = webhooks.ServerConfigurations{{URL: server.URL}}
	reconciler := &OutboundWebhookReconciler{OutboundWebhooksClient: webhooks.NewAPIClient(sdkConfig).OutgoingWebhooksServiceAPI}
	webhook := &coralogixv1alpha1.OutboundWebhook{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook", Namespace: "test"},
		Spec: coralogixv1alpha1.OutboundWebhookSpec{OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
			PagerDuty: &coralogixv1alpha1.PagerDuty{ServiceKeySecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "fake-webhook-values"}, Key: "service-key"}},
		}},
		Status: coralogixv1alpha1.OutboundWebhookStatus{ID: ptr.To("remote-id")},
	}
	if err := reconciler.HandleUpdate(ctx, logr.Discard(), webhook); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		t.Fatal(err)
	}
	secret.Data["service-key"] = []byte("fake-rotated-service-key-value")
	if err := c.Update(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if err := reconciler.HandleUpdate(ctx, logr.Discard(), webhook); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 || !strings.Contains(bodies[0], "fake-initial-service-key-value") || !strings.Contains(bodies[1], "fake-rotated-service-key-value") {
		t.Fatalf("update requests did not use the current Secret value")
	}
}

func TestOutboundWebhookSecretWatchSelectsReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := coralogixv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	webhooks := []*coralogixv1alpha1.OutboundWebhook{
		{ObjectMeta: metav1.ObjectMeta{Name: "first", Namespace: "test"}, Spec: coralogixv1alpha1.OutboundWebhookSpec{OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{PagerDuty: &coralogixv1alpha1.PagerDuty{ServiceKeySecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "shared"}, Key: "key"}}}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "second", Namespace: "test"}, Spec: coralogixv1alpha1.OutboundWebhookSpec{OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{Jira: &coralogixv1alpha1.Jira{UrlSecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "other"}, Key: "url"}}}}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(webhooks[0], webhooks[1]).
		WithIndex(&coralogixv1alpha1.OutboundWebhook{}, outboundWebhookSecretIndex, func(obj client.Object) []string {
			return outboundWebhookSecretNames(obj.(*coralogixv1alpha1.OutboundWebhook))
		}).Build()
	previousClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(previousClient) })
	config.InitClient(c)
	previousSelector := config.GetConfig().Selector
	t.Cleanup(func() { config.GetConfig().Selector = previousSelector })
	requests := (&OutboundWebhookReconciler{}).webhooksForSecret(context.Background(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "test"}})
	if len(requests) != 1 || requests[0].Name != "first" {
		t.Fatalf("Secret event requests = %v", requests)
	}
	requests = (&OutboundWebhookReconciler{}).webhooksForSecret(context.Background(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "another"}})
	if len(requests) != 0 {
		t.Fatalf("cross-namespace Secret event requests = %v", requests)
	}
	config.GetConfig().Selector = config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"selected": "true"})}
	requests = (&OutboundWebhookReconciler{}).webhooksForSecret(context.Background(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "test"}})
	if len(requests) != 0 {
		t.Fatalf("unselected webhook requests = %v", requests)
	}
}

func TestOutboundWebhookSecretIndexDeduplicatesNames(t *testing.T) {
	ref := &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "shared"}, Key: "field"}
	webhook := &coralogixv1alpha1.OutboundWebhook{Spec: coralogixv1alpha1.OutboundWebhookSpec{OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
		Jira: &coralogixv1alpha1.Jira{UrlSecretKeyRef: ref, ApiTokenSecretKeyRef: ref},
	}}}
	names := outboundWebhookSecretNames(webhook)
	if len(names) != 1 || names[0] != "shared" {
		t.Fatalf("Secret names = %v", names)
	}
}

type webhookTestAPIError struct {
	body string
}

func (e webhookTestAPIError) Error() string { return "request failed: fake-sensitive-value" }
func (e webhookTestAPIError) Body() []byte  { return []byte(e.body) }
func (e webhookTestAPIError) Model() any    { return nil }

func TestWebhookAPIErrorDoesNotExposeResponseBody(t *testing.T) {
	err := safeWebhookAPIError("create", &http.Response{StatusCode: http.StatusBadRequest}, webhookTestAPIError{body: "fake-sensitive-value"})
	if strings.Contains(err.Error(), "fake-sensitive-value") {
		t.Fatalf("unsafe error: %v", err)
	}
	if got := cxsdk.Code(err); got != http.StatusBadRequest {
		t.Fatalf("HTTP status = %d", got)
	}

	notFound := safeWebhookAPIError("update", &http.Response{StatusCode: http.StatusNotFound}, webhookTestAPIError{body: "Not Found: No webhook with the given id"})
	if !cxsdk.IsNotFound(notFound) {
		t.Fatal("resource not found classification was lost")
	}
	if strings.Contains(notFound.Error(), "given id") {
		t.Fatalf("unsafe error: %v", notFound)
	}

	transport := safeWebhookAPIError("create", nil, errors.New("fake-sensitive-value"))
	if strings.Contains(transport.Error(), "fake-sensitive-value") {
		t.Fatalf("unsafe transport error: %v", transport)
	}
}
