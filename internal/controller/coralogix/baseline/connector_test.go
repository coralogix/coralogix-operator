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

package baseline

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const connectorPath = "/notifications/notification-center/v1/connectors"

func init() {
	register(kind{
		name:     "connector",
		crdFile:  "coralogix.com_connectors.yaml",
		resource: "connectors",
		newFake:  newConnectorFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.ConnectorReconciler{ConnectorsClient: cs.Connectors(), Interval: reconcileInterval}
		},
		scenarios: connectorScenarios,
		admission: connectorAdmission(),
		fidelity: fidelityConfig{
			captureField: "connector.id",
			deletePath:   func(id string) string { return connectorPath + "/" + id },
		},
	})
}

// connectorSecret is the Secret that the scenarios read field values from.
const connectorSecret = "connector-secrets"

// connectorManifest returns a Connector CR named "connector". spec is YAML, indented by two spaces.
func connectorManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: Connector\nmetadata:\n  name: connector\nspec:\n" + spec
}

// A spec is built from these parts, so that each scenario shows only what it changes.
const (
	connectorSlackSpec = `  name: baseline-connector
  description: Baseline description
  type: slack
  connectorConfig:
    fields:
      - fieldName: channel
        value: general
`

	connectorFullSpec = `  name: baseline-connector
  description: Baseline description
  type: slack
  connectorConfig:
    fields:
      - fieldName: channel
        value: general
      - fieldName: integrationId
        value: some-integration-id
      - fieldName: fallbackChannel
        value: fallback_general
  configOverrides:
    - entityType: alerts
      fields:
        - fieldName: channel
          template: '{{alertDef.priority}}'
    - entityType: cases
      fields:
        - fieldName: channel
          template: cases-channel
`

	connectorSecretSpec = `  name: baseline-connector
  description: Baseline description
  type: pagerDuty
  connectorConfig:
    fields:
      - fieldName: integrationKey
        secretKeyRef:
          name: ` + connectorSecret + `
          key: integration-key
`
)

var (
	connectorMinimal    = connectorManifest(connectorSlackSpec)
	connectorFull       = connectorManifest(connectorFullSpec)
	connectorWithSecret = connectorManifest(connectorSecretSpec)
)

// connectorSpec builds a spec for SetSpec. extra holds the fields that the scenario adds or replaces.
func connectorSpec(typ string, fields []any, extra map[string]any) map[string]any {
	spec := map[string]any{
		"name":            "baseline-connector",
		"description":     "Baseline description",
		"type":            typ,
		"connectorConfig": map[string]any{"fields": fields},
	}
	maps.Copy(spec, extra)
	return spec
}

func valueField(name, value string) map[string]any {
	return map[string]any{"fieldName": name, "value": value}
}

func secretField(name, secret, key string) map[string]any {
	return map[string]any{"fieldName": name, "secretKeyRef": map[string]any{"name": secret, "key": key}}
}

func templateOverride(entityType, field, template string) map[string]any {
	return map[string]any{"entityType": entityType, "fields": []any{map[string]any{"fieldName": field, "template": template}}}
}

// createSecret creates the Secret in the scenario namespace. It writes Data, as the API server
// holds it: the reconciler reads Data.
func createSecret(h *harness, name string, data map[string]string) {
	h.t.Helper()
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace}, Data: secretData(data)}
	if err := h.client.Create(context.Background(), secret); err != nil {
		h.t.Fatalf("create Secret %s: %v", name, err)
	}
}

// updateSecret replaces the data of the Secret, as a user who rotates a key does.
func updateSecret(h *harness, name string, data map[string]string) {
	h.t.Helper()
	secret := &corev1.Secret{}
	if err := h.client.Get(context.Background(), types.NamespacedName{Namespace: h.namespace, Name: name}, secret); err != nil {
		h.t.Fatalf("get Secret %s: %v", name, err)
	}
	secret.Data = secretData(data)
	if err := h.client.Update(context.Background(), secret); err != nil {
		h.t.Fatalf("update Secret %s: %v", name, err)
	}
}

func deleteSecret(h *harness, name string) {
	h.t.Helper()
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace}}
	if err := h.client.Delete(context.Background(), secret); err != nil {
		h.t.Fatalf("delete Secret %s: %v", name, err)
	}
}

func secretData(data map[string]string) map[string][]byte {
	out := map[string][]byte{}
	for key, value := range data {
		out[key] = []byte(value)
	}
	return out
}

// createConnectorSampleWithSecret creates a sample that holds a Connector and its Secret as two
// YAML documents: the Secret first, then the CR.
func createConnectorSampleWithSecret(h *harness, name, path string) {
	h.t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, path))
	if err != nil {
		h.t.Fatal(err)
	}
	docs := strings.Split(string(raw), "\n---\n")
	if len(docs) != 2 {
		h.t.Fatalf("%s: want a Connector and a Secret, got %d documents", path, len(docs))
	}
	secret := &corev1.Secret{}
	if err := yaml.Unmarshal([]byte(docs[1]), secret); err != nil {
		h.t.Fatalf("%s: parse the Secret: %v", path, err)
	}
	// A sample writes stringData. The reconciler reads Data, so the scenario writes Data.
	data := map[string]string{}
	for key, value := range secret.StringData {
		data[key] = value
	}
	createSecret(h, secret.Name, data)
	h.Create(name, docs[0])
}

func createConnectorSample(file string) func(h *harness) {
	return func(h *harness) {
		h.CreateFromFile("create the sample", "config/samples/v1alpha1/connectors/"+file+".yaml")
	}
}

var connectorScenarios = map[string]func(h *harness){
	// One scenario for each sample file.
	"create-sample-slack":                createConnectorSample("slack"),
	"create-sample-generic-https":        createConnectorSample("generic-https"),
	"create-sample-pager-duty":           createConnectorSample("pager-duty"),
	"create-sample-pager-duty-incidents": createConnectorSample("pager-duty-incidents"),
	"create-sample-email":                createConnectorSample("email"),
	"create-sample-service-now":          createConnectorSample("service-now"),
	"create-sample-microsoft-teams":      createConnectorSample("microsoft-teams"),
	"create-sample-eventbridge":          createConnectorSample("eventbridge"),
	"create-sample-incident-io":          createConnectorSample("incident-io"),
	"create-sample-pager-duty-with-secret": func(h *harness) {
		createConnectorSampleWithSecret(h, "create the sample", "config/samples/v1alpha1/connectors/pager-duty-with-secret.yaml")
	},

	"create-minimal": func(h *harness) {
		h.Create("create with a name, a description, a type, and one field", connectorMinimal)
	},

	"create-full": func(h *harness) {
		h.Create("create with several fields and an override for each entity type", connectorFull)
	},

	// The CR requires description, so "" is the empty description. It is sent as "".
	"create-empty-description": func(h *harness) {
		h.Create("create with an empty description", connectorManifest(strings.Replace(connectorSlackSpec, "description: Baseline description", `description: ""`, 1)))
		requireSent(h, "POST", `"description":""`)
	},

	// An empty field list sends an empty connectorConfig: the request has no "fields" key.
	"create-empty-fields": func(h *harness) {
		h.Create("create with an empty list of fields", connectorManifest(`  name: baseline-connector
  description: Baseline description
  type: slack
  connectorConfig:
    fields: []
`))
		requireSent(h, "POST", `"connectorConfig":{}`)
	},

	// An override with no fields sends the entity type only.
	"create-override-without-fields": func(h *harness) {
		h.Create("create with an override that has no fields", connectorManifest(connectorSlackSpec+`  configOverrides:
    - entityType: alerts
`))
		requireSent(h, "POST", `"configOverrides":[{"entityType":"ALERTS"}]`)
	},

	// value: "" passes the exactly-one rule, because the field is set. It is sent as "".
	"create-empty-value": func(h *harness) {
		h.Create("create with a field whose value is empty", connectorManifest(`  name: baseline-connector
  description: Baseline description
  type: slack
  connectorConfig:
    fields:
      - fieldName: channel
        value: ""
`))
		requireSent(h, "POST", `"value":""`)
	},

	// The request keeps the order of the CR: fields, then overrides.
	"create-field-order": func(h *harness) {
		h.Create("create with fields and overrides in a fixed order", connectorManifest(`  name: baseline-connector
  description: Baseline description
  type: genericHttps
  connectorConfig:
    fields:
      - fieldName: url
        value: https://example.com/hook
      - fieldName: additionalBodyFields
        value: '{}'
      - fieldName: method
        value: POST
  configOverrides:
    - entityType: cases
      fields:
        - fieldName: url
          template: https://example.com/cases
        - fieldName: method
          template: PUT
    - entityType: alerts
      fields:
        - fieldName: method
          template: POST
`))
	},

	"create-secret": func(h *harness) {
		createSecret(h, connectorSecret, map[string]string{"integration-key": "secret-integration-key"})
		h.Create("create with a field from a Secret: the request holds the Secret value", connectorWithSecret)
		requireSent(h, "POST", `"value":"secret-integration-key"`)
	},

	// The Secret is read before any request. A missing Secret fails the create with no request.
	"create-secret-missing": func(h *harness) {
		h.Create("create with a Secret that does not exist: no request", connectorWithSecret)
		createSecret(h, connectorSecret, map[string]string{"integration-key": "secret-integration-key"})
		h.Reconcile("reconcile after the Secret exists")
	},

	"create-secret-missing-key": func(h *harness) {
		createSecret(h, connectorSecret, map[string]string{"other-key": "secret-integration-key"})
		h.Create("create with a Secret that has no such key: no request", connectorWithSecret)
	},

	// optional: true has no effect: a missing Secret is an error.
	"create-secret-optional-missing": func(h *harness) {
		h.Create("create with an optional Secret reference to a Secret that does not exist: no request",
			connectorManifest(strings.Replace(connectorSecretSpec, "key: integration-key", "key: integration-key\n          optional: true", 1)))
	},

	// The CRD defaults the Secret name to "". The reconciler fails to read a Secret with no name.
	"create-secret-empty-name": func(h *harness) {
		h.Create("create with a Secret reference that has no name: no request",
			connectorManifest(strings.Replace(connectorSecretSpec, "          name: "+connectorSecret+"\n", "", 1)))
	},

	"secret-rotated": func(h *harness) {
		createSecret(h, connectorSecret, map[string]string{"integration-key": "first-key"})
		h.Create("create", connectorWithSecret)
		updateSecret(h, connectorSecret, map[string]string{"integration-key": "second-key"})
		h.Reconcile("reconcile after the Secret changed: the update holds the new value")
		requireSent(h, "PUT", `"value":"second-key"`)
	},

	"update-secret-deleted": func(h *harness) {
		createSecret(h, connectorSecret, map[string]string{"integration-key": "secret-integration-key"})
		h.Create("create", connectorWithSecret)
		deleteSecret(h, connectorSecret)
		h.Reconcile("reconcile after the Secret was deleted: no request, and status.id stays")
	},

	"switch-value-to-secret": func(h *harness) {
		h.Create("create with a value", connectorManifest(strings.Replace(connectorSecretSpec,
			"        secretKeyRef:\n          name: "+connectorSecret+"\n          key: integration-key\n",
			"        value: literal-integration-key\n", 1)))
		createSecret(h, connectorSecret, map[string]string{"integration-key": "secret-integration-key"})
		h.SetSpec("switch the field to a Secret", connectorSpec("pagerDuty",
			[]any{secretField("integrationKey", connectorSecret, "integration-key")}, nil))
	},

	"switch-secret-to-value": func(h *harness) {
		createSecret(h, connectorSecret, map[string]string{"integration-key": "secret-integration-key"})
		h.Create("create with a Secret", connectorWithSecret)
		h.SetSpec("switch the field to a value", connectorSpec("pagerDuty",
			[]any{valueField("integrationKey", "literal-integration-key")}, nil))
	},

	"create-rejected": func(h *harness) {
		connectorAPI(h).answerNextError("POST", 400, "the API rejects this request")
		h.Create("create: the API answers 400", connectorMinimal)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-rejected-duplicate-name": func(h *harness) {
		connectorAPI(h).seed("baseline-connector")
		h.Create("create with a name that another connector has", connectorMinimal)
		h.Reconcile("reconcile again")
	},

	"create-route-not-found": func(h *harness) {
		connectorAPI(h).answerNextRouteNotFound("POST")
		h.Create("create: the API has no such route", connectorMinimal)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// A body that the SDK cannot decode is a DeserializationError. The shared reconciler does not
	// write the same condition twice.
	"create-malformed-response": func(h *harness) {
		connectorAPI(h).answerNext("POST", "not a connector")
		h.Create("create: the answer cannot be decoded", connectorMinimal)
		connectorAPI(h).answerNext("POST", "not a connector")
		h.Reconcile("reconcile again: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"resync": func(h *harness) {
		h.Create("create", connectorMinimal)
		h.Reconcile("reconcile again with no change: the operator sends a PUT")
		h.Reconcile("reconcile a third time")
	},

	"update": func(h *harness) {
		h.Create("create", connectorMinimal)
		h.SetSpec("rename, describe, change the fields, and add an override", connectorSpec("slack",
			[]any{valueField("channel", "alerts"), valueField("integrationId", "some-integration-id")},
			map[string]any{
				"name":            "baseline-connector-renamed",
				"description":     "Renamed",
				"configOverrides": []any{templateOverride("alerts", "channel", "{{alertDef.priority}}")},
			}))
	},

	"clear-optional": func(h *harness) {
		h.Create("create with every optional field", connectorFull)
		h.SetSpec("remove the overrides", connectorSpec("slack",
			[]any{valueField("channel", "general"), valueField("integrationId", "some-integration-id"), valueField("fallbackChannel", "fallback_general")}, nil))
		h.SetSpec("remove the fields", connectorSpec("slack", []any{}, nil))
	},

	"switch-type": func(h *harness) {
		h.Create("create a slack connector", connectorMinimal)
		h.SetSpec("switch the type to genericHttps", connectorSpec("genericHttps",
			[]any{valueField("url", "https://example.com/hook"), valueField("method", "POST")}, nil))
	},

	"update-rejected": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).answerNextError("PUT", 400, "the API rejects this request")
		h.Reconcile("resync: the API rejects the update")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// A 404 that names a resource on update is "not found": the shared reconciler removes the
	// status id, and the next reconcile creates again.
	"update-not-found": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).forget(h.StatusID())
		h.Reconcile("resync: the connector was deleted outside the operator, so the API answers 404")
		h.Reconcile("reconcile again: create")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).answerNext("PUT", "not a connector")
		h.Reconcile("resync: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// The update reads only the id of the answer, and the SDK does not require the connector. An
	// empty answer is success.
	"update-empty-response": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).answerNext("PUT", map[string]any{})
		h.Reconcile("resync: the answer has no connector")
	},

	"delete": func(h *harness) {
		h.Create("create", connectorMinimal)
		h.Delete("delete")
	},

	"delete-not-found": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).answerNextError("DELETE", 404, "connector not found")
		h.Delete("delete: a 404 that names a resource is success")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).forget(h.StatusID())
		h.Delete("delete: the connector was deleted outside the operator")
	},

	"delete-rejected": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).answerNextError("DELETE", 500, "try again")
		h.Delete("delete: the API answers 500")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-route-not-found": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).answerNextRouteNotFound("DELETE")
		h.Delete("delete: the API has no such route")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// The shared reconciler creates when the status has no id, even while the CR is deleted. So the
	// controller's own deletion is not reached. The first connector is still there, so the create
	// uses a name that exists.
	"delete-without-id": func(h *harness) {
		h.Create("create", connectorMinimal)
		setConnectorStatusID(h, nil)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"delete-without-id-and-remote-gone": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).forget(h.StatusID())
		setConnectorStatusID(h, nil)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", connectorMinimal)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "connectors"})})
		h.Reconcile("reconcile after the selector stops matching the CR: delete the connector")
		h.Reconcile("reconcile again")
	},

	// The operator sends the status id as it is. An id that the API does not know is a 404 on update.
	// The connector of the first create is gone, so the create after the 404 does not meet its name.
	"other-status-id": func(h *harness) {
		h.Create("create", connectorMinimal)
		connectorAPI(h).forget(h.StatusID())
		other := "another-id"
		setConnectorStatusID(h, &other)
		h.Reconcile("update with another status id: the API answers 404")
		h.Reconcile("reconcile again: create")
		h.Delete("delete")
	},
}

// setConnectorStatusID writes status.id of the CR, or removes it when id is nil, as a user who
// edits the status could.
func setConnectorStatusID(h *harness, id *string) {
	h.t.Helper()
	obj := h.get()
	if id == nil {
		unstructured.RemoveNestedField(obj.Object, "status", "id")
	} else if err := unstructured.SetNestedField(obj.Object, *id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set Connector status id: %v", err)
	}
}

// connectorAdmission lists the CRs that TestAdmission creates. Each case has one reason that the
// CRD accepts or rejects it.
func connectorAdmission() map[string]string {
	spec := func(fields string) string { return connectorManifest(fields) }
	// withFields is a spec whose connectorConfig.fields is the given YAML list.
	withFields := func(fields string) string {
		return spec("  name: c\n  description: d\n  type: slack\n  connectorConfig:\n    fields:\n" + fields)
	}
	withType := func(typ string) string {
		return spec("  name: c\n  description: d\n  type: " + typ + "\n  connectorConfig:\n    fields: []\n")
	}
	withOverrides := func(overrides string) string {
		return spec("  name: c\n  description: d\n  type: slack\n  connectorConfig:\n    fields: []\n  configOverrides:\n" + overrides)
	}
	out := map[string]string{
		"minimal":                            spec(connectorSlackSpec),
		"full":                               spec(connectorFullSpec),
		"secret field":                       spec(connectorSecretSpec),
		"empty fields":                       withType("slack"),
		"empty name":                         spec("  name: \"\"\n  description: d\n  type: slack\n  connectorConfig:\n    fields: []\n"),
		"empty description":                  spec("  name: c\n  description: \"\"\n  type: slack\n  connectorConfig:\n    fields: []\n"),
		"without a name":                     spec("  description: d\n  type: slack\n  connectorConfig:\n    fields: []\n"),
		"without a description":              spec("  name: c\n  type: slack\n  connectorConfig:\n    fields: []\n"),
		"without a type":                     spec("  name: c\n  description: d\n  connectorConfig:\n    fields: []\n"),
		"without a connector config":         spec("  name: c\n  description: d\n  type: slack\n"),
		"connector config without fields":    spec("  name: c\n  description: d\n  type: slack\n  connectorConfig: {}\n"),
		"type in upper case":                 withType("SLACK"),
		"type of the API":                    withType("GENERIC_HTTPS"),
		"type ibmEventNotifications":         withType("ibmEventNotifications"),
		"type unspecified":                   withType("unspecified"),
		"empty spec":                         "apiVersion: coralogix.com/v1alpha1\nkind: Connector\nmetadata:\n  name: connector\nspec: {}\n",
		"no spec":                            "apiVersion: coralogix.com/v1alpha1\nkind: Connector\nmetadata:\n  name: connector\n",
		"unknown field is not an error":      spec(connectorSlackSpec + "  labels: ignored\n"),
		"field with a value":                 withFields("      - fieldName: channel\n        value: general\n"),
		"field with an empty value":          withFields("      - fieldName: channel\n        value: \"\"\n"),
		"field with an empty field name":     withFields("      - fieldName: \"\"\n        value: general\n"),
		"field without a field name":         withFields("      - value: general\n"),
		"field with a secret reference":      withFields("      - fieldName: channel\n        secretKeyRef:\n          name: s\n          key: k\n"),
		"field with an optional secret":      withFields("      - fieldName: channel\n        secretKeyRef:\n          name: s\n          key: k\n          optional: true\n"),
		"secret reference without a name":    withFields("      - fieldName: channel\n        secretKeyRef:\n          key: k\n"),
		"secret reference without a key":     withFields("      - fieldName: channel\n        secretKeyRef:\n          name: s\n"),
		"secret reference with an empty key": withFields("      - fieldName: channel\n        secretKeyRef:\n          name: s\n          key: \"\"\n"),
		"empty secret reference":             withFields("      - fieldName: channel\n        secretKeyRef: {}\n"),
		// The exactly-one rule of ConnectorConfigField.
		"field with a value and a secret reference": withFields("      - fieldName: channel\n        value: general\n        secretKeyRef:\n          name: s\n          key: k\n"),
		"field with an empty value and a secret":    withFields("      - fieldName: channel\n        value: \"\"\n        secretKeyRef:\n          name: s\n          key: k\n"),
		"field with neither a value nor a secret":   withFields("      - fieldName: channel\n"),
		"one good field and one with neither":       withFields("      - fieldName: channel\n        value: general\n      - fieldName: other\n"),
		"override for alerts":                       withOverrides("    - entityType: alerts\n      fields:\n        - fieldName: channel\n          template: t\n"),
		"override for cases":                        withOverrides("    - entityType: cases\n      fields:\n        - fieldName: channel\n          template: t\n"),
		"override without fields":                   withOverrides("    - entityType: alerts\n"),
		"override with empty fields":                withOverrides("    - entityType: alerts\n      fields: []\n"),
		"override in upper case":                    withOverrides("    - entityType: ALERTS\n"),
		"override for testNotifications":            withOverrides("    - entityType: testNotifications\n"),
		"override for the API's TEST_NOTIFICATIONS": withOverrides("    - entityType: TEST_NOTIFICATIONS\n"),
		"override without an entity type":           withOverrides("    - fields: []\n"),
		"override field without a template":         withOverrides("    - entityType: alerts\n      fields:\n        - fieldName: channel\n"),
		"override field with an empty template":     withOverrides("    - entityType: alerts\n      fields:\n        - fieldName: channel\n          template: \"\"\n"),
		"override field without a field name":       withOverrides("    - entityType: alerts\n      fields:\n        - template: t\n"),
		"empty overrides":                           spec("  name: c\n  description: d\n  type: slack\n  connectorConfig:\n    fields: []\n  configOverrides: []\n"),
		"two overrides for one entity type":         withOverrides("    - entityType: alerts\n    - entityType: alerts\n"),
	}
	// Each type that the CRD accepts.
	for _, typ := range []string{"slack", "genericHttps", "pagerDuty", "pagerDutyIncidents", "email", "serviceNow", "microsoftTeams", "eventBridge", "incidentIo"} {
		out["type "+typ] = withType(typ)
	}
	return out
}
