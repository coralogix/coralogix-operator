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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "scope",
		crdFile:  "coralogix.com_scopes.yaml",
		resource: "scopes",
		newFake:  newScopeFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.ScopeReconciler{ScopesClient: cs.Scopes(), Interval: reconcileInterval}
		},
		scenarios: scopeScenarios,
		admission: scopeAdmission(),
		fidelity: fidelityConfig{
			captureField: "scope.id",
			deletePath:   func(id string) string { return scopePath + "/" + id },
		},
	})
}

const scopeSpec = `  name: baseline-scope
  description: Baseline scope
  filters:
    - entityType: logs
      expression: <v1>subsystemName == 'purchases'
  defaultExpression: <v1>true
`

func scopeManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: Scope\nmetadata:\n  name: scope\nspec:\n" + spec
}

func scopes(h *harness) *scopeFake {
	h.t.Helper()
	f, ok := h.API().(*scopeFake)
	if !ok {
		h.t.Fatal("expected *scopeFake")
	}
	return f
}

// scopeSpecMap is a whole spec. Each filter is an entity type and an expression.
func scopeSpecMap(name string, description *string, defaultExpression string, filters ...[2]string) map[string]any {
	items := make([]any, 0, len(filters))
	for _, f := range filters {
		items = append(items, map[string]any{"entityType": f[0], "expression": f[1]})
	}
	spec := map[string]any{"name": name, "filters": items, "defaultExpression": defaultExpression}
	if description != nil {
		spec["description"] = *description
	}
	return spec
}

func setScopeStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatal(err)
	}
}

var scopeScenarios = map[string]func(h *harness){
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the released sample", "config/samples/v1alpha1/scopes/example-scope.yaml")
		h.Reconcile("resync: update sends the whole spec")
	},
	"create-minimal": func(h *harness) {
		h.Create("create without a description", scopeManifest("  name: baseline-scope\n  filters:\n    - entityType: spans\n      expression: <v1>true\n  defaultExpression: <v1>false\n"))
		h.Reconcile("resync")
		h.Reconcile("resync again")
	},
	"create-empty-description": func(h *harness) {
		h.Create("an empty description is sent and stored", scopeManifest("  name: baseline-scope\n  description: \"\"\n  filters:\n    - entityType: logs\n      expression: <v1>true\n  defaultExpression: <v1>true\n"))
		h.RecordRemote("the remote description is empty", scopes(h).snapshot())
	},
	"create-without-spec": func(h *harness) {
		h.Create("create without spec", "apiVersion: coralogix.com/v1alpha1\nkind: Scope\nmetadata:\n  name: scope\n")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-unspecified-entity-type": func(h *harness) {
		h.Create("admission accepts unspecified, the API rejects it", scopeManifest("  name: baseline-scope\n  filters:\n    - entityType: unspecified\n      expression: <v1>true\n  defaultExpression: <v1>true\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-rejected": func(h *harness) {
		scopes(h).answerNext(http.MethodPost, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Create("create returns an API error", scopeManifest(scopeSpec))
		h.Reconcile("retry creates the scope")
	},
	"create-empty-id": func(h *harness) {
		scopes(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{"scope": map[string]any{}})
		h.Create("create succeeds without an id", scopeManifest(scopeSpec))
		h.Reconcile("retry")
	},
	"create-without-scope": func(h *harness) {
		scopes(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{})
		h.Create("create answers without the scope", scopeManifest(scopeSpec))
		h.Reconcile("retry creates the scope")
	},
	"create-empty-id-after-remote-create": func(h *harness) {
		fake := scopes(h)
		fake.put("11111111-1111-4111-8111-111111111111", scopeRecord{
			displayName: "baseline-scope", description: ptr.To("Baseline scope"),
			filters:           []scopeFilter{{entityType: "ENTITY_TYPE_LOGS", expression: "<v1>subsystemName == 'purchases'"}},
			defaultExpression: "<v1>true",
		})
		fake.answerNext(http.MethodPost, http.StatusOK, map[string]any{})
		h.Create("create stores a scope but returns no scope", scopeManifest(scopeSpec))
		h.Reconcile("retry creates a second scope with the same name")
		h.Delete("delete")
		h.RecordRemote("the first scope remains", fake.snapshot())
	},
	"update-fields": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		h.SetSpec("rename, replace filters, change the default", scopeSpecMap("renamed-scope", ptr.To("Changed"), "<v1>false",
			[2]string{"spans", "<v1>serviceName == 'checkout'"}, [2]string{"logs", "<v1>true"}))
		h.RecordRemote("the remote scope", scopes(h).snapshot())
	},
	"update-clear-description": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		h.SetSpec("remove the description: update omits it", scopeSpecMap("baseline-scope", nil, "<v1>true",
			[2]string{"logs", "<v1>subsystemName == 'purchases'"}))
		h.RecordRemote("the remote description is cleared", scopes(h).snapshot())
	},
	"update-rejected": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		h.SetSpec("an expression without a version tag", scopeSpecMap("baseline-scope", ptr.To("Baseline scope"), "<v1>true",
			[2]string{"logs", "subsystemName == 'purchases'"}))
		h.Reconcile("retry")
		h.RecordRemote("the remote scope is unchanged", scopes(h).snapshot())
	},
	"update-when-gone": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		scopes(h).forget(h.StatusID())
		h.SetSpec("update after remote deletion", scopeSpecMap("renamed-scope", ptr.To("Changed"), "<v1>true",
			[2]string{"logs", "<v1>true"}))
		h.Reconcile("recreate")
	},
	"bad-status-id": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		setScopeStatusID(h, "not-a-uuid")
		h.Reconcile("update of an unknown id is a 404")
		h.Reconcile("recreate")
		h.Delete("delete")
	},
	"delete": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		h.Delete("delete")
		h.Reconcile("delete again")
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		scopes(h).forget(h.StatusID())
		h.Delete("delete after remote deletion: the API answers 200")
	},
	"delete-rejected": func(h *harness) {
		h.Create("create", scopeManifest(scopeSpec))
		scopes(h).answerNext(http.MethodDelete, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Delete("delete returns an API error")
		h.Reconcile("retry deletes the scope")
	},
	"selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
		h.Create("create: a direct reconcile does not run the event predicate", scopeManifest(scopeSpec))
		h.Reconcile("the selector does not match: delete the remote scope and clear the status")
		h.Delete("delete")
	},
}

func scopeAdmission() map[string]string {
	const filters = "  filters:\n    - entityType: logs\n      expression: <v1>true\n"
	return map[string]string{
		"sample":                      scopeManifest(scopeSpec),
		"without spec":                "apiVersion: coralogix.com/v1alpha1\nkind: Scope\nmetadata:\n  name: scope\n",
		"empty spec":                  scopeManifest("  {}\n"),
		"name absent":                 scopeManifest(filters + "  defaultExpression: <v1>true\n"),
		"name empty":                  scopeManifest("  name: \"\"\n" + filters + "  defaultExpression: <v1>true\n"),
		"description empty":           scopeManifest("  name: s\n  description: \"\"\n" + filters + "  defaultExpression: <v1>true\n"),
		"filters absent":              scopeManifest("  name: s\n  defaultExpression: <v1>true\n"),
		"filters empty":               scopeManifest("  name: s\n  filters: []\n  defaultExpression: <v1>true\n"),
		"default expression absent":   scopeManifest("  name: s\n" + filters),
		"default expression false":    scopeManifest("  name: s\n" + filters + "  defaultExpression: <v1>false\n"),
		"default expression other":    scopeManifest("  name: s\n" + filters + "  defaultExpression: <v1>maybe\n"),
		"default expression untagged": scopeManifest("  name: s\n" + filters + "  defaultExpression: \"true\"\n"),
		"entity type spans":           scopeManifest("  name: s\n  filters:\n    - entityType: spans\n      expression: <v1>true\n  defaultExpression: <v1>true\n"),
		"entity type unspecified":     scopeManifest("  name: s\n  filters:\n    - entityType: unspecified\n      expression: <v1>true\n  defaultExpression: <v1>true\n"),
		"entity type unknown":         scopeManifest("  name: s\n  filters:\n    - entityType: metrics\n      expression: <v1>true\n  defaultExpression: <v1>true\n"),
		"entity type absent":          scopeManifest("  name: s\n  filters:\n    - expression: <v1>true\n  defaultExpression: <v1>true\n"),
		"expression absent":           scopeManifest("  name: s\n  filters:\n    - entityType: logs\n  defaultExpression: <v1>true\n"),
		"expression empty":            scopeManifest("  name: s\n  filters:\n    - entityType: logs\n      expression: \"\"\n  defaultExpression: <v1>true\n"),
	}
}
