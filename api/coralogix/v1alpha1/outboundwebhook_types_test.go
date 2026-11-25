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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func webhookTestSecretRef(key string) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: "fake-webhook-values"},
		Key:                  key,
	}
}

func webhookTestClient(t *testing.T) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-webhook-values", Namespace: "test"},
		Data: map[string][]byte{
			"service-key": []byte("fake-service-key-value"),
			"api-token":   []byte("fake-api-token-value"),
			"url":         []byte("https://example.invalid/hook"),
			"header":      []byte("fake-header-value"),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	previousClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(previousClient) })
	config.InitClient(c)
	return c
}

func TestOutboundWebhookReadsSecretForCreateUpdateAndRotation(t *testing.T) {
	ctx := context.Background()
	c := webhookTestClient(t)
	webhook := &OutboundWebhook{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test"},
		Spec: OutboundWebhookSpec{OutboundWebhookType: OutboundWebhookType{
			PagerDuty: &PagerDuty{ServiceKeySecretKeyRef: webhookTestSecretRef("service-key")},
		}},
	}

	create, err := webhook.ExtractCreateOutboundWebhookRequestWithContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := create.Data.PagerDuty.ServiceKey; got != "fake-service-key-value" {
		t.Fatalf("create service key = %q", got)
	}

	secret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: "test", Name: "fake-webhook-values"}, secret); err != nil {
		t.Fatal(err)
	}
	secret.Data["service-key"] = []byte("fake-rotated-service-key-value")
	if err := c.Update(ctx, secret); err != nil {
		t.Fatal(err)
	}
	webhook.Status.ID = ptr.To("remote-id")
	update, err := webhook.ExtractUpdateOutboundWebhookRequestWithContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := update.Data.PagerDuty.ServiceKey; got != "fake-rotated-service-key-value" {
		t.Fatalf("update service key = %q", got)
	}
}

func TestOutboundWebhookResolvesURLAndCredentialHeader(t *testing.T) {
	webhookTestClient(t)
	ctx := context.Background()
	ref := webhookTestSecretRef("url")
	tests := []struct {
		name string
		kind OutboundWebhookType
	}{
		{"generic", OutboundWebhookType{GenericWebhook: &GenericWebhook{Method: GenericWebhookMethodTypePost, UrlSecretKeyRef: ref, Headers: map[string]string{"X-Name": "example"}, HeadersFromSecret: map[string]corev1.SecretKeySelector{"Authorization": *webhookTestSecretRef("header")}}}},
		{"slack", OutboundWebhookType{Slack: &Slack{UrlSecretKeyRef: ref}}},
		{"sendLog", OutboundWebhookType{SendLog: &SendLog{UrlSecretKeyRef: ref}}},
		{"microsoftTeams", OutboundWebhookType{MicrosoftTeams: &MicrosoftTeams{UrlSecretKeyRef: ref}}},
		{"jira", OutboundWebhookType{Jira: &Jira{UrlSecretKeyRef: ref, ApiTokenSecretKeyRef: webhookTestSecretRef("api-token")}}},
		{"opsgenie", OutboundWebhookType{Opsgenie: &Opsgenie{UrlSecretKeyRef: ref}}},
		{"demisto", OutboundWebhookType{Demisto: &Demisto{UrlSecretKeyRef: ref}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webhook := &OutboundWebhook{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}, Spec: OutboundWebhookSpec{OutboundWebhookType: test.kind}}
			request, err := webhook.ExtractCreateOutboundWebhookRequestWithContext(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got := *request.Data.Url; got != "https://example.invalid/hook" {
				t.Fatalf("resolved URL = %q", got)
			}
			if test.name == "generic" {
				if got := request.Data.GenericWebhook.Headers["Authorization"]; got != "fake-header-value" {
					t.Fatalf("resolved header = %q", got)
				}
			}
			if test.name == "jira" && request.Data.Jira.ApiToken != "fake-api-token-value" {
				t.Fatal("Jira API token was not resolved")
			}
		})
	}
}

func TestOutboundWebhookRejectsMissingSecretAndSourceCollision(t *testing.T) {
	webhookTestClient(t)
	ctx := context.Background()
	tests := []struct {
		name string
		kind OutboundWebhookType
		want string
	}{
		{"missing Secret", OutboundWebhookType{PagerDuty: &PagerDuty{ServiceKeySecretKeyRef: webhookTestSecretRef("missing")}}, "cannot find key"},
		{"two service key sources", OutboundWebhookType{PagerDuty: &PagerDuty{ServiceKey: "fake-literal-service-key", ServiceKeySecretKeyRef: webhookTestSecretRef("service-key")}}, "both"},
		{"two header sources", OutboundWebhookType{GenericWebhook: &GenericWebhook{Url: "https://example.invalid", Headers: map[string]string{"Authorization": "fake-literal-header-value"}, HeadersFromSecret: map[string]corev1.SecretKeySelector{"Authorization": *webhookTestSecretRef("header")}}}, "both"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			webhook := &OutboundWebhook{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}, Spec: OutboundWebhookSpec{OutboundWebhookType: test.kind}}
			_, err := webhook.ExtractCreateOutboundWebhookRequestWithContext(ctx)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v; want %q", err, test.want)
			}
		})
	}
}
