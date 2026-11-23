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
	"net/http"
	"regexp"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "customrole",
		crdFile:  "coralogix.com_customroles.yaml",
		resource: "customroles",
		newFake:  newCustomRoleFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.CustomRoleReconciler{CustomRolesClient: cs.CustomRoles(), Interval: reconcileInterval}
		},
		scenarios: customRoleScenarios,
		admission: customRoleAdmission(),
		fidelity: fidelityConfig{
			captureField: "id",
			numericID:    true,
			deletePath:   func(id string) string { return customRolePath + "/" + id },
			normalize:    normalizeCustomRoleAnswer,
		},
	})
}

// The parent name and permission come from the published CustomRole sample.
const customRoleSpec = "  name: baseline-role\n  description: Baseline role\n  parentRoleName: Standard User\n  permissions:\n    - team-actions:UpdateConfig\n"

func customRoleManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: CustomRole\nmetadata:\n  name: role\nspec:\n" + spec
}

func customRoles(h *harness) *customRoleFake {
	h.t.Helper()
	f, ok := h.API().(*customRoleFake)
	if !ok {
		h.t.Fatal("expected *customRoleFake")
	}
	return f
}

func customRoleSpecMap(name, description, parent string, permissions ...string) map[string]any {
	return map[string]any{
		"name": name, "description": description, "parentRoleName": parent, "permissions": permissions,
	}
}

func setCustomRoleStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatal(err)
	}
}

var customRoleScenarios = map[string]func(h *harness){
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the released sample", "config/samples/v1alpha1/customroles/example-custom-role.yaml")
		h.Reconcile("resync: update sends the whole mutable spec")
	},
	"create-minimal": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		h.Reconcile("resync")
		h.Reconcile("resync again")
	},
	"create-without-spec": func(h *harness) {
		h.Create("create without spec", "apiVersion: coralogix.com/v1alpha1\nkind: CustomRole\nmetadata:\n  name: role\n")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-rejected": func(h *harness) {
		customRoles(h).answerNext(http.MethodPut, http.StatusBadRequest, map[string]any{"code": 400, "message": "Bad Request: rejected"})
		h.Create("create returns an API error", customRoleManifest(customRoleSpec))
		h.Reconcile("retry creates the role")
	},
	"create-empty-id": func(h *harness) {
		customRoles(h).answerNext(http.MethodPut, http.StatusOK, map[string]any{})
		h.Create("create succeeds without an id", customRoleManifest(customRoleSpec))
		h.Reconcile("retry creates the role")
	},
	"create-empty-id-after-remote-create": func(h *harness) {
		fake := customRoles(h)
		fake.roles[1001] = roleRecord{name: "baseline-role", description: "Baseline role", parent: "Standard User", permissions: []string{"team-actions:UpdateConfig"}}
		fake.nextID = 1002
		fake.answerNext(http.MethodPut, http.StatusOK, map[string]any{})
		h.Create("create stores a role but returns no id", customRoleManifest(customRoleSpec))
		h.Reconcile("retry meets the stored role")
		h.Delete("delete the CR without a status id")
		h.RecordRemote("the role remains", fake.snapshot())
	},
	"create-duplicate-name": func(h *harness) {
		customRoles(h).roles[900] = roleRecord{name: "baseline-role", parent: "Standard User"}
		h.Create("name already in use", customRoleManifest(customRoleSpec))
		h.Reconcile("retry")
		h.Delete("delete the CR without a remote id")
	},
	"update-fields": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		h.SetSpec("rename and replace permissions", customRoleSpecMap("renamed-role", "Changed", "Standard User", "TEAM-CUSTOM-API-KEYS:READCONFIG"))
		h.Reconcile("resync")
	},
	"update-parent": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		h.SetSpec("change parent: update request has no parent", customRoleSpecMap("baseline-role", "Baseline role", "Admin", "team-actions:UpdateConfig"))
		h.RecordRemote("the remote parent stays", customRoles(h).snapshot())
	},
	"update-rejected": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		customRoles(h).answerNext(http.MethodPost, http.StatusBadRequest, map[string]any{"code": 400, "message": "Bad Request: rejected"})
		h.SetSpec("update returns an API error", customRoleSpecMap("renamed-role", "Changed", "Standard User", "team-actions:UpdateConfig"))
		h.Reconcile("retry")
	},
	"update-when-gone": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		id, err := strconv.ParseInt(h.StatusID(), 10, 64)
		if err != nil {
			h.t.Fatal(err)
		}
		customRoles(h).remove(id)
		h.SetSpec("update after remote deletion", customRoleSpecMap("renamed-role", "Changed", "Standard User", "team-actions:UpdateConfig"))
		h.Reconcile("recreate")
	},
	"bad-status-id": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		setCustomRoleStatusID(h, "not-an-integer")
		h.Reconcile("update cannot parse status id")
		h.Delete("delete cannot parse status id")
	},
	"delete": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		h.Delete("delete")
		h.Reconcile("delete again")
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", customRoleManifest(customRoleSpec))
		id, err := strconv.ParseInt(h.StatusID(), 10, 64)
		if err != nil {
			h.t.Fatal(err)
		}
		customRoles(h).remove(id)
		h.Delete("delete after remote deletion")
	},
	"selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
		h.Create("selector does not match", customRoleManifest(customRoleSpec))
		h.Reconcile("no remote call")
		h.Delete("delete")
	},
}

func customRoleAdmission() map[string]string {
	return map[string]string{
		"sample":             customRoleManifest(customRoleSpec),
		"without spec":       "apiVersion: coralogix.com/v1alpha1\nkind: CustomRole\nmetadata:\n  name: role\n",
		"empty spec":         customRoleManifest("  {}\n"),
		"name absent":        customRoleManifest("  description: Baseline role\n  parentRoleName: Standard User\n  permissions: [team-actions:UpdateConfig]\n"),
		"parent absent":      customRoleManifest("  name: baseline-role\n  description: Baseline role\n  permissions: [team-actions:UpdateConfig]\n"),
		"permissions absent": customRoleManifest("  name: baseline-role\n  description: Baseline role\n  parentRoleName: Standard User\n"),
		"permissions empty":  customRoleManifest("  name: baseline-role\n  description: Baseline role\n  parentRoleName: Standard User\n  permissions: []\n"),
		"name empty":         customRoleManifest("  name: \"\"\n  description: Baseline role\n  parentRoleName: Standard User\n  permissions: [team-actions:UpdateConfig]\n"),
		"parent empty":       customRoleManifest("  name: baseline-role\n  description: Baseline role\n  parentRoleName: \"\"\n  permissions: [team-actions:UpdateConfig]\n"),
		"description absent": customRoleManifest("  name: baseline-role\n  parentRoleName: Standard User\n  permissions: [team-actions:UpdateConfig]\n"),
	}
}

func normalizeCustomRoleAnswer(resp any) any {
	switch v := resp.(type) {
	case map[string]any:
		for key, value := range v {
			if key == "parentRoleId" {
				v[key] = "<parent-role-id>"
				continue
			}
			if key == "message" {
				if message, ok := value.(string); ok {
					v[key] = customRoleTeamNumber.ReplaceAllString(message, "for team <redacted>")
					continue
				}
			}
			v[key] = normalizeCustomRoleAnswer(value)
		}
	case []any:
		for i := range v {
			v[i] = normalizeCustomRoleAnswer(v[i])
		}
	}
	return resp
}

var customRoleTeamNumber = regexp.MustCompile(`for team (?:<number>|[0-9]+)`)

func TestCustomRoleNumericIDRecording(t *testing.T) {
	k := kinds["customrole"]
	response := map[string]any{
		"id": float64(1234), "otherCount": float64(12345),
		"message": "role 1234 was created; role 12345 was not",
		"role":    map[string]any{"roleId": float64(1234), "teamId": float64(5678)},
	}
	if got := capturedID(response, k); got != "1234" {
		t.Fatalf("captured ID = %q", got)
	}
	vars := map[string]string{}
	got := normalizeAnswer(t, k, fidelityStep{Capture: "role"}, http.StatusOK, response, vars)
	if want := `{"id":"{{role}}","message":"role {{role}} was created; role \u003cnumber\u003e was not","otherCount":12345,"role":{"roleId":"{{role}}","teamId":"\u003credacted\u003e"}}`; compact(got) != want {
		t.Fatalf("normalized response = %s, want %s", compact(got), want)
	}
}
