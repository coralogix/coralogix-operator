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
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "integration",
		crdFile:  "coralogix.com_integrations.yaml",
		resource: "integrations",
		newFake:  newIntegrationFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.IntegrationReconciler{IntegrationsClient: cs.Integrations(), Interval: reconcileInterval}
		},
		scenarios: integrationScenarios,
		admission: integrationAdmission(),
		// The handwritten controller builds the parameter list from a Go map, so its order
		// changes from run to run.
		mask: sortIntegrationParameters,
		fidelity: fidelityConfig{
			captureField: "integrationId",
			deletePath:   func(id string) string { return integrationInstancePath + id },
			normalize:    normalizeIntegrationAnswer,
		},
	})
}

// The parameters are those of the e2e test of the released Integration.
const integrationSpec = `  integrationKey: aws-metrics-collector
  version: 0.1.0
  parameters:
    ApplicationName: cxo
    SubsystemName: aws-metrics-collector
    MetricNamespaces: [AWS/S3]
    AwsRoleArn: arn:aws:iam::123456789012:role/example-role
    IntegrationName: cxo-integration-setup
    AwsRegion: eu-north-1
    WithAggregations: false
    EnrichWithTags: true
`

func integrationManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: Integration\nmetadata:\n  name: integration\nspec:\n" + spec
}

func integrations(h *harness) *integrationFake {
	h.t.Helper()
	f, ok := h.API().(*integrationFake)
	if !ok {
		h.t.Fatal("expected *integrationFake")
	}
	return f
}

func integrationSpecMap(parameters map[string]any) map[string]any {
	return map[string]any{
		"integrationKey": "aws-metrics-collector",
		"version":        "0.1.0",
		"parameters":     parameters,
	}
}

func integrationParameters() map[string]any {
	return map[string]any{
		"ApplicationName":  "cxo",
		"SubsystemName":    "aws-metrics-collector",
		"MetricNamespaces": []any{"AWS/S3"},
		"AwsRoleArn":       "arn:aws:iam::123456789012:role/example-role",
		"IntegrationName":  "cxo-integration-setup",
		"AwsRegion":        "eu-north-1",
		"WithAggregations": false,
		"EnrichWithTags":   true,
	}
}

// putSecret creates or updates a Secret in the scenario namespace.
func putSecret(h *harness, name string, data map[string]string) {
	h.t.Helper()
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace}, StringData: data}
	err := h.client.Create(context.Background(), secret)
	if err == nil {
		return
	}
	existing := &corev1.Secret{}
	if getErr := h.client.Get(context.Background(), client.ObjectKeyFromObject(secret), existing); getErr != nil {
		h.t.Fatalf("create secret: %v", err)
	}
	existing.Data = nil
	existing.StringData = data
	if err := h.client.Update(context.Background(), existing); err != nil {
		h.t.Fatalf("update secret: %v", err)
	}
}

const integrationSecretSpec = integrationSpecWithoutRoleArn + `  parametersFromSecret:
    AwsRoleArn:
      name: aws-role
      key: role-arn
`

const integrationSpecWithoutRoleArn = `  integrationKey: aws-metrics-collector
  version: 0.1.0
  parameters:
    ApplicationName: cxo
    SubsystemName: aws-metrics-collector
    MetricNamespaces: [AWS/S3]
    IntegrationName: cxo-integration-setup
    AwsRegion: eu-north-1
    WithAggregations: false
    EnrichWithTags: true
`

var integrationScenarios = map[string]func(h *harness){
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the released sample", "config/samples/v1alpha1/integrations/aws-metrics-collector.yaml")
		h.Reconcile("resync: update sends the whole spec")
	},
	"create-sample-with-secret": func(h *harness) {
		raw, err := os.ReadFile(filepath.Join(repoRoot, "config/samples/v1alpha1/integrations/gcp-metrics-collector-with-secret.yaml"))
		if err != nil {
			h.t.Fatal(err)
		}
		docs := strings.Split(string(raw), "\n---\n")
		if len(docs) != 2 {
			h.t.Fatalf("expected a Secret and an Integration, got %d documents", len(docs))
		}
		putSecret(h, "gcp-metrics-collector-sa-key", map[string]string{"key.json": "{\"type\": \"service_account\"}"})
		h.Create("create the released sample with a secret", docs[1])
		h.Reconcile("resync")
	},
	"create-minimal": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		h.Reconcile("resync")
		h.Reconcile("resync again")
	},
	"create-without-parameters": func(h *harness) {
		h.Create("create without parameters sends an empty parameter object", integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n"))
	},
	"create-without-spec": func(h *harness) {
		h.Create("create without spec", "apiVersion: coralogix.com/v1alpha1\nkind: Integration\nmetadata:\n  name: integration\n")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-number-and-list": func(h *harness) {
		h.Create("numbers, nested lists, and list items that are not strings", integrationManifest(`  integrationKey: aws-metrics-collector
  version: 0.1.0
  parameters:
    ApplicationName: cxo
    PollIntervalMinutes: 5
    Ratio: 0.25
    MetricNamespaces: [AWS/S3, 7, true, AWS/EC2]
    EmptyList: []
`))
	},
	"create-unsupported-parameter": func(h *harness) {
		h.Create("an object parameter is not supported", integrationManifest(`  integrationKey: aws-metrics-collector
  version: 0.1.0
  parameters:
    ApplicationName: cxo
    Nested:
      inner: value
`))
	},
	"create-null-parameter": func(h *harness) {
		h.Create("a null parameter", integrationManifest(`  integrationKey: aws-metrics-collector
  version: 0.1.0
  parameters:
    ApplicationName: null
`))
	},
	"create-with-secret": func(h *harness) {
		putSecret(h, "aws-role", map[string]string{"role-arn": "arn:aws:iam::123456789012:role/example-role"})
		h.Create("the secret value is a string parameter", integrationManifest(integrationSecretSpec))
		h.Reconcile("resync reads the secret again")
	},
	"secret-rotation": func(h *harness) {
		putSecret(h, "aws-role", map[string]string{"role-arn": "arn:aws:iam::123456789012:role/example-role"})
		h.Create("create", integrationManifest(integrationSecretSpec))
		putSecret(h, "aws-role", map[string]string{"role-arn": "arn:aws:iam::123456789012:role/rotated-role"})
		h.Reconcile("resync sends the new secret value")
	},
	"secret-missing": func(h *harness) {
		h.Create("the secret does not exist", integrationManifest(integrationSecretSpec))
		putSecret(h, "aws-role", map[string]string{"role-arn": "arn:aws:iam::123456789012:role/example-role"})
		h.Reconcile("retry after the secret appears")
	},
	"secret-key-missing": func(h *harness) {
		putSecret(h, "aws-role", map[string]string{"other": "value"})
		h.Create("the secret has no such key", integrationManifest(integrationSecretSpec))
	},
	"secret-optional-missing": func(h *harness) {
		h.Create("an optional missing secret is skipped", integrationManifest(integrationSpecWithoutRoleArn+`  parametersFromSecret:
    AwsRoleArn:
      name: aws-role
      key: role-arn
      optional: true
`))
		putSecret(h, "aws-role", map[string]string{"other": "value"})
		h.Reconcile("an optional missing key is skipped")
	},
	"secret-conflict": func(h *harness) {
		putSecret(h, "aws-role", map[string]string{"role-arn": "arn:aws:iam::123456789012:role/example-role"})
		h.Create("a parameter in both sources", integrationManifest(integrationSpec+`  parametersFromSecret:
    AwsRoleArn:
      name: aws-role
      key: role-arn
`))
	},
	"create-rejected": func(h *harness) {
		h.Create("create with an invalid version", integrationManifest(strings.Replace(integrationSpec, "0.1.0", "9.9.9", 1)))
		h.Reconcile("retry")
		h.Delete("delete the CR without a remote id")
	},
	"create-unknown-key": func(h *harness) {
		h.Create("create with an unknown key answers 404", integrationManifest(strings.Replace(integrationSpec, "aws-metrics-collector\n", "cxo-no-such-integration\n", 1)))
		h.Reconcile("retry")
	},
	"create-empty-id": func(h *harness) {
		integrations(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{})
		h.Create("create succeeds without an id", integrationManifest(integrationSpec))
		h.Reconcile("retry creates the integration")
	},
	"update-parameters": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		params := integrationParameters()
		params["ApplicationName"] = "cxo-updated"
		params["MetricNamespaces"] = []any{"AWS/S3", "AWS/EC2"}
		h.SetSpec("change parameters", integrationSpecMap(params))
		h.RecordRemote("the remote holds the new values", integrations(h).snapshot())
	},
	"update-remove-parameter": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		params := integrationParameters()
		delete(params, "AwsRegion")
		h.SetSpec("remove a parameter: the update omits it", integrationSpecMap(params))
		h.RecordRemote("the remote keeps the removed parameter", integrations(h).snapshot())
	},
	"update-remove-all-parameters": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		h.SetSpec("remove every parameter", map[string]any{"integrationKey": "aws-metrics-collector", "version": "0.1.0"})
		h.RecordRemote("the remote keeps every parameter", integrations(h).snapshot())
	},
	"update-rejected": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		spec := integrationSpecMap(integrationParameters())
		spec["version"] = "9.9.9"
		h.SetSpec("update with an invalid version", spec)
		h.Reconcile("retry")
	},
	"update-when-gone": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		integrations(h).remove(h.StatusID())
		params := integrationParameters()
		params["ApplicationName"] = "cxo-updated"
		h.SetSpec("update after remote deletion", integrationSpecMap(params))
		h.Reconcile("recreate")
	},
	"resync-when-gone": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		integrations(h).remove(h.StatusID())
		h.Reconcile("resync after remote deletion")
		h.Reconcile("recreate")
	},
	"delete": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		h.Delete("delete")
		h.Reconcile("delete again")
	},
	// A chart upgrade: the released operator created the integration and wrote status.id and its
	// finalizer. The operator updates that integration, creates none, and deletes it on delete.
	"upgrade-from-released": func(h *harness) {
		id := integrations(h).seed("aws-metrics-collector", "0.1.0", map[string]map[string]any{
			"IntegrationName": {"stringValue": "released-name"},
		})
		h.CreateReleased("the released operator made the integration: reconcile the CR", integrationManifest(integrationSpec),
			[]string{"integration.coralogix.com/finalizer"}, map[string]any{"id": id})
		if got := h.StatusID(); got != id {
			h.t.Fatalf("status.id changed from %s to %s", id, got)
		}
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		integrations(h).remove(h.StatusID())
		h.Delete("delete after remote deletion")
	},
	"delete-rejected": func(h *harness) {
		h.Create("create", integrationManifest(integrationSpec))
		integrations(h).answerNext(http.MethodDelete, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: failed"})
		h.Delete("delete returns an API error")
		h.Reconcile("retry deletes the integration")
	},
	"selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
		h.Create("selector does not match: Reconcile still creates, as the event filter applies the selector", integrationManifest(integrationSpec))
		h.Reconcile("resync deletes the remote integration and clears the status")
		h.RecordRemote("the remote is gone", integrations(h).snapshot())
		h.Delete("delete")
	},
}

func integrationAdmission() map[string]string {
	return map[string]string{
		"sample":                   integrationManifest(integrationSpec),
		"without spec":             "apiVersion: coralogix.com/v1alpha1\nkind: Integration\nmetadata:\n  name: integration\n",
		"empty spec":               integrationManifest("  {}\n"),
		"key absent":               integrationManifest("  version: 0.1.0\n"),
		"version absent":           integrationManifest("  integrationKey: aws-metrics-collector\n"),
		"key empty":                integrationManifest("  integrationKey: \"\"\n  version: 0.1.0\n"),
		"version empty":            integrationManifest("  integrationKey: aws-metrics-collector\n  version: \"\"\n"),
		"parameters a list":        integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parameters: [a]\n"),
		"parameters a string":      integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parameters: a\n"),
		"parameters empty":         integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parameters: {}\n"),
		"parameters nested object": integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parameters:\n    Nested:\n      inner: value\n"),
		"secret ref without key":   integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parametersFromSecret:\n    AwsRoleArn:\n      name: aws-role\n"),
		"secret ref without name":  integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parametersFromSecret:\n    AwsRoleArn:\n      key: role-arn\n"),
		"secret ref unknown field": integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parametersFromSecret:\n    AwsRoleArn:\n      key: role-arn\n      namespace: other\n"),
		"secret refs empty":        integrationManifest("  integrationKey: aws-metrics-collector\n  version: 0.1.0\n  parametersFromSecret: {}\n"),
	}
}

// sortIntegrationParameters sorts each list named "parameters" whose items have a "key".
func sortIntegrationParameters(golden []byte) []byte {
	var v any
	if err := json.Unmarshal(golden, &v); err != nil {
		panic(err)
	}
	out, err := json.MarshalIndent(sortParameterLists(v), "", "  ")
	if err != nil {
		panic(err)
	}
	return append(out, '\n')
}

func sortParameterLists(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for key, value := range t {
			t[key] = sortParameterLists(value)
			if key != "parameters" {
				continue
			}
			list, ok := t[key].([]any)
			if !ok {
				continue
			}
			sort.SliceStable(list, func(i, j int) bool { return parameterKey(list[i]) < parameterKey(list[j]) })
		}
	case []any:
		for i := range t {
			t[i] = sortParameterLists(t[i])
		}
	}
	return v
}

func parameterKey(v any) string {
	m, _ := v.(map[string]any)
	key, _ := m["key"].(string)
	return key
}

// normalizeIntegrationAnswer sorts the parameters, which the API returns in no fixed order, and
// drops integrationStatus, which reports the connection to the cloud account and which the
// operator never reads.
func normalizeIntegrationAnswer(resp any) any {
	if m, ok := resp.(map[string]any); ok {
		if integration, ok := m["integration"].(map[string]any); ok {
			delete(integration, "integrationStatus")
		}
	}
	return sortParameterLists(resp)
}

func TestSortIntegrationParameters(t *testing.T) {
	got := string(sortIntegrationParameters([]byte(`{"body":{"parameters":[{"key":"b"},{"key":"a"}]},"other":[{"key":"b"},{"key":"a"}]}`)))
	want := "{\n  \"body\": {\n    \"parameters\": [\n      {\n        \"key\": \"a\"\n      },\n      {\n        \"key\": \"b\"\n      }\n    ]\n  },\n  \"other\": [\n    {\n      \"key\": \"b\"\n    },\n    {\n      \"key\": \"a\"\n    }\n  ]\n}\n"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
