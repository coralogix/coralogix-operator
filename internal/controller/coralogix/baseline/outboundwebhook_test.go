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
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "outboundwebhook",
		crdFile:  "coralogix.com_outboundwebhooks.yaml",
		resource: "outboundwebhooks",
		// The released kind reads the Secrets that a webhook names, and watches them.
		rbacExtra: []string{"secrets"},
		newFake:   newWebhookFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.OutboundWebhookReconciler{Client: cs.Webhooks(), ClientSet: cs, Interval: reconcileInterval}
		},
		scenarios: webhookScenarios,
		admission: webhookAdmission(),
		fidelity: fidelityConfig{
			captureField: "id",
			deletePath:   func(id string) string { return webhookPath + "/" + id },
		},
	})
}

const webhookGenericSpec = `  name: baseline-webhook
  outboundWebhookType:
    genericWebhook:
      url: https://example.com/hook
      method: Post
      headers:
        X-Team: platform
      payload: '{"alert": "$ALERT_NAME"}'
`

func webhookManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: OutboundWebhook\nmetadata:\n  name: webhook\nspec:\n" + spec
}

// webhookSpec is a whole spec with one webhook type.
func webhookSpec(name, arm string, config map[string]any) map[string]any {
	return map[string]any{"name": name, "outboundWebhookType": map[string]any{arm: config}}
}

func webhooks(h *harness) *webhookFake {
	h.t.Helper()
	f, ok := h.API().(*webhookFake)
	if !ok {
		h.t.Fatal("expected *webhookFake")
	}
	return f
}

func setWebhookStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatal(err)
	}
}

// webhookSampleSecrets are the Secrets that the released samples name, with test values.
func webhookSampleSecrets(h *harness) {
	h.t.Helper()
	putSecret(h, "webhook-credentials", map[string]string{"url": "https://example.com/from-secret", "authorization": "Bearer fake-token"})
	putSecret(h, "jira-credentials", map[string]string{"api-token": "fake-jira-token"})
	putSecret(h, "pagerduty-credentials", map[string]string{"service-key": "fake-service-key"})
}

func webhookSample(file string) func(h *harness) {
	return func(h *harness) {
		webhookSampleSecrets(h)
		h.CreateFromFile("create the released sample", "config/samples/v1alpha1/outboundwebhooks/"+file)
		h.Reconcile("resync: update sends the whole spec")
	}
}

const webhookSecretSpec = `  name: baseline-webhook
  outboundWebhookType:
    genericWebhook:
      urlSecretKeyRef:
        name: webhook-values
        key: url
      method: Post
      headers:
        X-Team: platform
      headersFromSecret:
        Authorization:
          name: webhook-values
          key: authorization
`

var webhookScenarios = map[string]func(h *harness){
	"create-sample-aws-event-bridge":            webhookSample("aws-event-bridge.yaml"),
	"create-sample-demisto":                     webhookSample("demisto.yaml"),
	"create-sample-email-group":                 webhookSample("email-group.yaml"),
	"create-sample-generic-webhook":             webhookSample("generic-webhook.yaml"),
	"create-sample-generic-webhook-with-secret": webhookSample("generic-webhook-with-secret.yaml"),
	"create-sample-jira":                        webhookSample("jira.yaml"),
	"create-sample-microsoft-teams":             webhookSample("microsoft-teams.yaml"),
	"create-sample-opsgenie":                    webhookSample("opsgenie.yaml"),
	"create-sample-pagerduty":                   webhookSample("pagerduty.yaml"),
	"create-sample-sendlog":                     webhookSample("sendlog.yaml"),
	"create-sample-slack":                       webhookSample("slack.yaml"),
	"create-generic": func(h *harness) {
		h.Create("create: POST, then GET for the external id", webhookManifest(webhookGenericSpec))
		h.Reconcile("resync: update sends a new uuid")
		h.Reconcile("resync again")
	},
	"create-literal-credentials": func(h *harness) {
		h.Create("create with the deprecated literal service key", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    pagerDuty:\n      serviceKey: fake-literal-key\n"))
		h.SetSpec("jira with the deprecated literal token", webhookSpec("baseline-webhook", "jira", map[string]any{
			"apiToken": "fake-literal-token", "email": "user@example.com", "projectKey": "OPS", "url": "https://example.atlassian.net",
		}))
	},
	"create-slack-every-value": func(h *harness) {
		h.Create("create with each digest and attachment type", webhookManifest(`  name: baseline-webhook
  outboundWebhookType:
    slack:
      url: https://hooks.slack.com/services/baseline
      digests:
        - {type: ErrorAndCriticalLogs, isActive: true}
        - {type: FlowAnomalies, isActive: false}
        - {type: SpikeAnomalies, isActive: true}
        - {type: DataUsage, isActive: true}
      attachments:
        - {type: MetricSnapshot, isActive: true}
        - {type: Logs, isActive: false}
`))
	},
	// Admission accepts the zero values of the released enums. The API rejects them.
	"create-zero-enums": func(h *harness) {
		h.Create("a method Unknown", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    genericWebhook:\n      url: https://example.com/hook\n      method: Unknown\n"))
		h.SetSpec("a digest Unknown and an attachment Empty", webhookSpec("baseline-webhook", "slack", map[string]any{
			"url":         "https://hooks.slack.com/services/baseline",
			"digests":     []any{map[string]any{"type": "Unknown", "isActive": true}},
			"attachments": []any{map[string]any{"type": "Empty", "isActive": true}},
		}))
		h.SetSpec("each method", webhookSpec("baseline-webhook", "genericWebhook", map[string]any{"url": "https://example.com/hook", "method": "Get"}))
		h.SetSpec("put", webhookSpec("baseline-webhook", "genericWebhook", map[string]any{"url": "https://example.com/hook", "method": "Put"}))
	},
	"create-every-type": func(h *harness) {
		h.Create("email group", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    emailGroup:\n      emailAddresses: [first@example.com, second@example.com, first@example.com]\n"))
		h.Delete("delete")
		h.Create("an empty email group", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    emailGroup:\n      emailAddresses: []\n"))
		h.Delete("delete")
		h.Create("opsgenie", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    opsgenie:\n      url: https://api.opsgenie.com/v1/json/coralogix\n"))
		h.Delete("delete")
		h.Create("demisto", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    demisto:\n      uuid: 00000000-0000-4000-8000-000000000001\n      payload: '{}'\n      url: https://example.com/demisto\n"))
		h.Delete("delete")
		h.Create("aws event bridge", webhookManifest(`  name: baseline-webhook
  outboundWebhookType:
    awsEventBridge:
      eventBusArn: arn:aws:events:us-east-1:000000000000:event-bus/default
      detail: '{}'
      detailType: alert
      source: coralogix
      roleName: coralogix-role
`))
		h.Delete("delete")
		h.Create("send log without a URL", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    sendLog:\n      payload: '{}'\n      url: \"\"\n"))
		h.Delete("delete")
	},
	"secret-references": func(h *harness) {
		putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/from-secret", "authorization": "Bearer fake-token"})
		h.Create("create: the URL and a header come from the Secret", webhookManifest(webhookSecretSpec))
		putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/rotated", "authorization": "Bearer fake-rotated-token"})
		h.Reconcile("the Secret changed: update sends the new values")
		h.Delete("delete")
	},
	"secret-references-every-type": func(h *harness) {
		putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/from-secret", "token": "fake-token", "key": "fake-key"})
		ref := func(key string) map[string]any { return map[string]any{"name": "webhook-values", "key": key} }
		h.Create("pagerduty service key", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    pagerDuty:\n      serviceKeySecretKeyRef: {name: webhook-values, key: key}\n"))
		h.Delete("delete")
		h.Create("jira token and URL", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    jira:\n      apiTokenSecretKeyRef: {name: webhook-values, key: token}\n      urlSecretKeyRef: {name: webhook-values, key: url}\n      email: user@example.com\n      projectKey: OPS\n"))
		h.SetSpec("slack URL", webhookSpec("baseline-webhook", "slack", map[string]any{"urlSecretKeyRef": ref("url")}))
		h.Delete("delete")
		for _, arm := range []string{"slack", "opsgenie", "microsoftTeams"} {
			h.Create(arm+" URL", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    "+arm+":\n      urlSecretKeyRef: {name: webhook-values, key: url}\n"))
			h.Delete("delete")
		}
		h.Create("demisto URL", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    demisto:\n      uuid: 00000000-0000-4000-8000-000000000001\n      payload: '{}'\n      urlSecretKeyRef: {name: webhook-values, key: url}\n"))
		h.Delete("delete")
		h.Create("send log URL", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    sendLog:\n      payload: '{}'\n      urlSecretKeyRef: {name: webhook-values, key: url}\n"))
		h.Delete("delete")
	},
	"secret-missing": func(h *harness) {
		h.Create("the Secret does not exist: no request", webhookManifest(webhookSecretSpec))
		putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/from-secret", "authorization": "Bearer fake-token"})
		h.Reconcile("the Secret exists: create")
	},
	"secret-key-missing": func(h *harness) {
		putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/from-secret"})
		h.Create("the Secret has no authorization key: no request", webhookManifest(webhookSecretSpec))
	},
	"secret-key-removed-after-create": func(h *harness) {
		putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/from-secret", "authorization": "Bearer fake-token"})
		h.Create("create", webhookManifest(webhookSecretSpec))
		putSecret(h, "webhook-values", map[string]string{"url": "https://example.com/from-secret"})
		h.Reconcile("the key is gone: no request, and the status id stays")
		h.Delete("delete")
	},
	// A chart upgrade: the released operator created the webhook and wrote status.id, the external
	// id, and its finalizer. The operator updates that webhook, creates none, and deletes it on
	// delete.
	"upgrade-from-released": func(h *harness) {
		const id = "22222222-2222-4222-8222-222222222222"
		webhooks(h).putRequest(id, map[string]any{
			"name": "baseline-webhook", "url": "https://example.com/hook",
			"genericWebhook": map[string]any{"uuid": "33333333-3333-4333-8333-333333333333", "method": "POST"},
		})
		h.CreateReleased("the released operator made the webhook: reconcile the CR", webhookManifest(webhookGenericSpec),
			[]string{"outbound-webhook.coralogix.com/finalizer"}, map[string]any{"id": id, "externalId": "1001", "printableStatus": "RemoteSynced"})
		if got := h.StatusID(); got != id {
			h.t.Fatalf("status.id changed from %s to %s", id, got)
		}
		h.Reconcile("reconcile again")
		h.Delete("delete: the webhook is deleted by its released id")
		h.RecordRemote("no webhook remains", webhooks(h).snapshot())
	},
	"create-without-spec": func(h *harness) {
		h.Create("create without spec: admission accepts it, no request", "apiVersion: coralogix.com/v1alpha1\nkind: OutboundWebhook\nmetadata:\n  name: webhook\n")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-rejected": func(h *harness) {
		webhooks(h).answerNext(http.MethodPost, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Create("create returns an API error", webhookManifest(webhookGenericSpec))
		h.Reconcile("retry creates the webhook")
	},
	"create-api-rejected": func(h *harness) {
		h.Create("an empty name: the API rejects it", webhookManifest("  name: \"\"\n  outboundWebhookType:\n    opsgenie:\n      url: https://api.opsgenie.com/v1/json/coralogix\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-empty-id": func(h *harness) {
		webhooks(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{})
		h.Create("create succeeds without an id", webhookManifest(webhookGenericSpec))
		h.Reconcile("retry")
	},
	"create-get-fails": func(h *harness) {
		webhooks(h).answerNext(http.MethodGet, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Create("create succeeds, the GET for the external id fails", webhookManifest(webhookGenericSpec))
		h.Reconcile("retry creates a second webhook")
		h.Delete("delete")
		h.RecordRemote("the first webhook remains", webhooks(h).snapshot())
	},
	"create-get-empty-id": func(h *harness) {
		webhooks(h).answerNext(http.MethodGet, http.StatusOK, map[string]any{"webhook": map[string]any{
			"id": "", "type": "GENERIC", "name": "baseline-webhook", "url": "https://example.com/hook", "externalId": 1001,
			"createdAt": webhookTime, "updatedAt": webhookTime,
			"genericWebhook": map[string]any{"uuid": fixedIDPrefix + "000000000003", "method": "POST"},
		}})
		h.Create("create succeeds, the GET answers a webhook with an empty id", webhookManifest(webhookGenericSpec))
		h.Reconcile("retry")
		h.Delete("delete")
		h.RecordRemote("the remote webhooks", webhooks(h).snapshot())
	},
	"create-malformed-response": func(h *harness) {
		webhooks(h).answerNext(http.MethodPost, http.StatusOK, "not an object")
		h.Create("create answers a string", webhookManifest(webhookGenericSpec))
		h.Reconcile("retry")
	},
	"update-fields": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		h.SetSpec("rename, change the URL and method, drop headers and payload", webhookSpec("renamed-webhook", "genericWebhook", map[string]any{
			"url": "https://example.com/hook-updated", "method": "Put",
		}))
		h.RecordRemote("the remote webhook", webhooks(h).snapshot())
	},
	"update-slack-lists": func(h *harness) {
		const url = "https://hooks.slack.com/services/baseline"
		h.Create("create with a digest and an attachment", webhookManifest(`  name: baseline-webhook
  outboundWebhookType:
    slack:
      url: https://hooks.slack.com/services/baseline
      digests:
        - {type: ErrorAndCriticalLogs, isActive: true}
      attachments:
        - {type: MetricSnapshot, isActive: true}
`))
		h.SetSpec("change the digest and the attachment", webhookSpec("baseline-webhook", "slack", map[string]any{
			"url":         url,
			"digests":     []any{map[string]any{"type": "DataUsage", "isActive": false}},
			"attachments": []any{map[string]any{"type": "Logs", "isActive": true}},
		}))
		h.SetSpec("empty lists", webhookSpec("baseline-webhook", "slack", map[string]any{"url": url, "digests": []any{}, "attachments": []any{}}))
		h.RecordRemote("the lists are empty", webhooks(h).snapshot())
		h.SetSpec("set the lists again", webhookSpec("baseline-webhook", "slack", map[string]any{
			"url":         url,
			"digests":     []any{map[string]any{"type": "SpikeAnomalies", "isActive": true}},
			"attachments": []any{map[string]any{"type": "MetricSnapshot", "isActive": true}},
		}))
		h.SetSpec("leave the lists out", webhookSpec("baseline-webhook", "slack", map[string]any{"url": url}))
		h.RecordRemote("the lists are empty again", webhooks(h).snapshot())
		h.Delete("delete")
	},
	"update-email-group": func(h *harness) {
		h.Create("create", webhookManifest("  name: baseline-webhook\n  outboundWebhookType:\n    emailGroup:\n      emailAddresses: [first@example.com, second@example.com]\n"))
		h.SetSpec("replace an address", webhookSpec("baseline-webhook", "emailGroup", map[string]any{
			"emailAddresses": []any{"second@example.com", "third@example.com"},
		}))
		h.RecordRemote("the new addresses", webhooks(h).snapshot())
		h.SetSpec("add a duplicate", webhookSpec("baseline-webhook", "emailGroup", map[string]any{
			"emailAddresses": []any{"second@example.com", "third@example.com", "second@example.com"},
		}))
		h.SetSpec("remove every address", webhookSpec("baseline-webhook", "emailGroup", map[string]any{"emailAddresses": []any{}}))
		h.RecordRemote("no address remains", webhooks(h).snapshot())
		h.Delete("delete")
	},
	"update-change-type": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		h.SetSpec("a generic webhook becomes a PagerDuty webhook: the API rejects it", webhookSpec("baseline-webhook", "pagerDuty", map[string]any{"serviceKey": "fake-literal-key"}))
		h.Reconcile("retry")
		h.RecordRemote("the remote webhook is unchanged", webhooks(h).snapshot())
	},
	"update-rejected": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		h.SetSpec("a method Unknown", webhookSpec("baseline-webhook", "genericWebhook", map[string]any{"url": "https://example.com/hook", "method": "Unknown"}))
		h.Reconcile("retry")
	},
	"update-malformed-response": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		webhooks(h).answerNext(http.MethodPut, http.StatusOK, "not an object")
		h.Reconcile("update answers a string")
	},
	"update-when-gone": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		webhooks(h).forget(h.StatusID())
		h.SetSpec("update after remote deletion", webhookSpec("renamed-webhook", "genericWebhook", map[string]any{"url": "https://example.com/hook", "method": "Post"}))
		h.Reconcile("recreate")
	},
	// The update 404 removes status.id. A delete before the retry finds a CR with a finalizer
	// and no status id. It names no remote resource: the shared reconciler removes the finalizer
	// with no API call, and no new webhook is created.
	"delete-after-update-gone": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		webhooks(h).forget(h.StatusID())
		h.SetSpec("update after remote deletion: the 404 removes status.id", webhookSpec("renamed-webhook", "genericWebhook", map[string]any{"url": "https://example.com/hook", "method": "Post"}))
		h.Delete("delete a CR that has a finalizer and no status id: no webhook is created")
		h.RecordRemote("no remote webhook exists", webhooks(h).snapshot())
	},
	// The API answers 503 for an id that is not a UUID, so the status id stays.
	"bad-status-id": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		setWebhookStatusID(h, "not-a-uuid")
		h.Reconcile("update of an id that is not a UUID is a 503")
		h.Delete("delete is a 503 too: the finalizer stays")
	},
	"delete": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		h.Delete("delete")
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		webhooks(h).forget(h.StatusID())
		h.Delete("delete after remote deletion: the API answers 404")
	},
	"delete-rejected": func(h *harness) {
		h.Create("create", webhookManifest(webhookGenericSpec))
		webhooks(h).answerNext(http.MethodDelete, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Delete("delete returns an API error")
		h.Reconcile("retry deletes the webhook")
	},
	"selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
		h.Create("create: a direct reconcile does not run the event predicate", webhookManifest(webhookGenericSpec))
		h.Reconcile("the selector does not match: delete the remote webhook and clear the status")
		h.Delete("delete")
	},
}

func webhookAdmission() map[string]string {
	m := func(arm, config string) string {
		return webhookManifest("  name: w\n  outboundWebhookType:\n    " + arm + ":\n" + config)
	}
	return map[string]string{
		"sample":                         webhookManifest(webhookGenericSpec),
		"secret sample":                  webhookManifest(webhookSecretSpec),
		"empty spec":                     webhookManifest("  {}\n"),
		"name absent":                    webhookManifest("  outboundWebhookType:\n    opsgenie:\n      url: https://example.com\n"),
		"name empty":                     webhookManifest("  name: \"\"\n  outboundWebhookType:\n    opsgenie:\n      url: https://example.com\n"),
		"type absent":                    webhookManifest("  name: w\n"),
		"type empty":                     webhookManifest("  name: w\n  outboundWebhookType: {}\n"),
		"two types":                      webhookManifest("  name: w\n  outboundWebhookType:\n    opsgenie:\n      url: https://example.com\n    pagerDuty:\n      serviceKey: k\n"),
		"generic url and ref":            m("genericWebhook", "      method: Post\n      url: https://example.com\n      urlSecretKeyRef: {name: s, key: url}\n"),
		"generic no url":                 m("genericWebhook", "      method: Post\n"),
		"generic empty url":              m("genericWebhook", "      method: Post\n      url: \"\"\n"),
		"generic method absent":          m("genericWebhook", "      url: https://example.com\n"),
		"generic method other":           m("genericWebhook", "      url: https://example.com\n      method: Delete\n"),
		"generic method unknown":         m("genericWebhook", "      url: https://example.com\n      method: Unknown\n"),
		"generic method upper case":      m("genericWebhook", "      url: https://example.com\n      method: POST\n"),
		"generic header in both sources": m("genericWebhook", "      url: https://example.com\n      method: Post\n      headers: {Authorization: x}\n      headersFromSecret:\n        Authorization: {name: s, key: a}\n"),
		"generic header ref no key":      m("genericWebhook", "      url: https://example.com\n      method: Post\n      headersFromSecret:\n        Authorization: {name: s}\n"),
		"generic url ref no key":         m("genericWebhook", "      method: Post\n      urlSecretKeyRef: {name: s}\n"),
		"generic url ref optional":       m("genericWebhook", "      method: Post\n      urlSecretKeyRef: {name: s, key: url, optional: true}\n"),
		"generic empty payload":          m("genericWebhook", "      url: https://example.com\n      method: Post\n      payload: \"\"\n"),
		"slack url":                      m("slack", "      url: https://example.com\n"),
		"slack no url":                   m("slack", "      digests: []\n"),
		"slack digest type other":        m("slack", "      url: https://example.com\n      digests: [{type: Other, isActive: true}]\n"),
		"slack digest unknown":           m("slack", "      url: https://example.com\n      digests: [{type: Unknown, isActive: true}]\n"),
		"slack digest no isActive":       m("slack", "      url: https://example.com\n      digests: [{type: DataUsage}]\n"),
		"slack attachment empty":         m("slack", "      url: https://example.com\n      attachments: [{type: Empty, isActive: true}]\n"),
		"slack attachment other":         m("slack", "      url: https://example.com\n      attachments: [{type: Other, isActive: true}]\n"),
		"pagerduty key":                  m("pagerDuty", "      serviceKey: k\n"),
		"pagerduty empty key":            m("pagerDuty", "      serviceKey: \"\"\n"),
		"pagerduty key and ref":          m("pagerDuty", "      serviceKey: k\n      serviceKeySecretKeyRef: {name: s, key: k}\n"),
		"pagerduty neither":              m("pagerDuty", "      {}\n"),
		"sendlog no payload":             m("sendLog", "      url: https://example.com\n"),
		"sendlog no url":                 m("sendLog", "      payload: '{}'\n"),
		"email group":                    m("emailGroup", "      emailAddresses: [a@example.com]\n"),
		"email group empty":              m("emailGroup", "      emailAddresses: []\n"),
		"email group absent":             m("emailGroup", "      {}\n"),
		"teams url":                      m("microsoftTeams", "      url: https://example.com\n"),
		"teams neither":                  m("microsoftTeams", "      {}\n"),
		"jira":                           m("jira", "      apiToken: t\n      email: a@example.com\n      projectKey: P\n      url: https://example.com\n"),
		"jira token and ref":             m("jira", "      apiToken: t\n      apiTokenSecretKeyRef: {name: s, key: t}\n      email: a@example.com\n      projectKey: P\n      url: https://example.com\n"),
		"jira no token":                  m("jira", "      email: a@example.com\n      projectKey: P\n      url: https://example.com\n"),
		"jira no email":                  m("jira", "      apiToken: t\n      projectKey: P\n      url: https://example.com\n"),
		"jira no project key":            m("jira", "      apiToken: t\n      email: a@example.com\n      url: https://example.com\n"),
		"opsgenie neither":               m("opsgenie", "      {}\n"),
		"demisto":                        m("demisto", "      uuid: u\n      payload: p\n      url: https://example.com\n"),
		"demisto no uuid":                m("demisto", "      payload: p\n      url: https://example.com\n"),
		"demisto no payload":             m("demisto", "      uuid: u\n      url: https://example.com\n"),
		"aws":                            m("awsEventBridge", "      eventBusArn: a\n      detail: d\n      detailType: t\n      source: s\n      roleName: r\n"),
		"aws empty values":               m("awsEventBridge", "      eventBusArn: \"\"\n      detail: \"\"\n      detailType: \"\"\n      source: \"\"\n      roleName: \"\"\n"),
		"aws no role":                    m("awsEventBridge", "      eventBusArn: a\n      detail: d\n      detailType: t\n      source: s\n"),
		"unknown type":                   webhookManifest("  name: w\n  outboundWebhookType:\n    msTeamsWorkflow:\n      url: https://example.com\n"),
	}
}
