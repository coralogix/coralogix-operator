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

package baseline

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// The scenarios call Reconcile themselves, so they cannot show that a Secret change starts a
// reconcile. Here the controller runs in a manager with the reconcile interval of the scenarios
// (10 minutes): only the Secret watch can send the rotated values within the timeout.
func TestOutboundWebhookSecretWatch(t *testing.T) {
	h := newHarness(t, kinds["outboundwebhook"], "secret-watch")
	fake := webhooks(h)
	remote := func() string {
		data, err := json.Marshal(fake.snapshot())
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/from-secret", "authorization": "Bearer fake-token"})
	putSecret(h, "other-values", map[string]string{"url": "https://example.com/other"})
	h.startManager()

	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(webhookManifest(webhookSecretSpec)), &obj.Object); err != nil {
		t.Fatal(err)
	}
	obj.SetNamespace(h.namespace)
	h.key = client.ObjectKeyFromObject(obj)
	h.gvk = obj.GroupVersionKind()
	if err := h.client.Create(context.Background(), obj); err != nil {
		t.Fatal(err)
	}
	h.eventually("the webhook is created with the URL of the Secret", func() bool {
		id, _, _ := unstructured.NestedString(h.get().Object, "status", "id")
		return strings.Contains(remote(), "https://example.com/from-secret") && id != ""
	})
	id := h.StatusID()

	putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/rotated", "authorization": "Bearer fake-rotated-token"})
	h.eventually("the rotated Secret reaches the webhook", func() bool {
		return strings.Contains(remote(), "https://example.com/rotated")
	})
	if got := h.StatusID(); got != id {
		t.Fatalf("status.id changed from %s to %s", id, got)
	}
	if snapshot := fake.snapshot(); len(snapshot) != 1 {
		t.Fatalf("the API has %d webhooks, want 1", len(snapshot))
	}

	if err := h.client.Delete(context.Background(), obj); err != nil {
		t.Fatal(err)
	}
	h.eventually("the webhook is deleted", func() bool { return len(fake.snapshot()) == 0 })
}
