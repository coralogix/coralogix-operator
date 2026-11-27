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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "recordingrulegroupset",
		crdFile:  "coralogix.com_recordingrulegroupsets.yaml",
		resource: "recordingrulegroupsets",
		newFake:  newRecordingRuleGroupSetFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.RecordingRuleGroupSetReconciler{RecordingRulesClient: cs.RecordingRules(), Interval: reconcileInterval}
		},
		scenarios: recordingRuleGroupSetScenarios,
		admission: recordingRuleGroupSetAdmission(),
		fidelity: fidelityConfig{
			captureField: "id",
			deletePath:   func(id string) string { return recordingRulesPath + "/" + id },
			normalize:    normalizeRecordingRuleGroupSetAnswer,
		},
	})
}

// releasedRecordingRuleGroupSetFinalizer is the finalizer that the released operator adds.
const releasedRecordingRuleGroupSetFinalizer = "recordingrulegroupset.coralogix.com/finalizer"

const recordingRuleGroupSetSpec = "  groups:\n    - name: baseline.rules\n      intervalSeconds: 120\n      limit: 100\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n          labels:\n            team: baseline\n"

func recordingRuleGroupSetManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: RecordingRuleGroupSet\nmetadata:\n  name: rules\nspec:\n" + spec
}

func recordingRuleGroupSets(h *harness) *recordingRuleGroupSetFake {
	h.t.Helper()
	f, ok := h.API().(*recordingRuleGroupSetFake)
	if !ok {
		h.t.Fatal("expected *recordingRuleGroupSetFake")
	}
	return f
}

func setRecordingRuleGroupSetSuffix(h *harness, suffix string) {
	h.t.Helper()
	r, ok := h.reconciler.(*v1alpha1controllers.RecordingRuleGroupSetReconciler)
	if !ok {
		h.t.Fatal("expected *RecordingRuleGroupSetReconciler")
	}
	r.RecordingRuleGroupSetSuffix = suffix
}

func setRecordingRuleGroupSetStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatal(err)
	}
}

func ruleGroupSpec(name string, rules ...any) map[string]any {
	return map[string]any{"name": name, "rules": rules}
}

func ruleSpec(record, expr string) map[string]any {
	return map[string]any{"record": record, "expr": expr}
}

func recordingRuleGroupSetSpecMap(groups ...any) map[string]any {
	return map[string]any{"groups": groups}
}

var recordingRuleGroupSetScenarios = map[string]func(h *harness){
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the released sample", "config/samples/v1alpha1/recordingrulegroupset/coralogix_v1alpha1_recordingrulegroupset.yaml")
		h.Reconcile("resync: update sends the groups, never the name")
	},
	"create-minimal": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.Reconcile("resync")
		h.Reconcile("resync again")
	},
	"create-defaults": func(h *harness) {
		h.Create("a group without interval and limit: the CRD defaults the interval, Create sends limit 0",
			recordingRuleGroupSetManifest("  groups:\n    - name: defaults\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n"))
		h.Reconcile("resync")
	},
	"create-suffix": func(h *harness) {
		setRecordingRuleGroupSetSuffix(h, "-eu2")
		h.Create("the operator suffix is part of the set name", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.Reconcile("resync: update sends no name")
	},
	"create-group-without-rules": func(h *harness) {
		h.Create("a group with no rules", recordingRuleGroupSetManifest("  groups:\n    - name: empty\n"))
		h.Reconcile("resync")
	},
	"create-without-spec": func(h *harness) {
		h.Create("create without spec", "apiVersion: coralogix.com/v1alpha1\nkind: RecordingRuleGroupSet\nmetadata:\n  name: rules\n")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-empty-group-name": func(h *harness) {
		h.Create("a group with an empty name", recordingRuleGroupSetManifest("  groups:\n    - name: \"\"\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-group-without-name": func(h *harness) {
		h.Create("a group without a name", recordingRuleGroupSetManifest("  groups:\n    - rules:\n        - record: baseline_record\n          expr: vector(1)\n"))
		h.Delete("delete")
	},
	"create-empty-record-and-expr": func(h *harness) {
		h.Create("a rule without record and expr", recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      rules:\n        - labels:\n            team: baseline\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-interval-below-minimum": func(h *harness) {
		h.Create("an interval of 30 seconds", recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      intervalSeconds: 30\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-interval-zero": func(h *harness) {
		h.Create("an interval of 0: Create sends it and the API answers 400", recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      intervalSeconds: 0\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-interval-above-int32": func(h *harness) {
		h.Create("an interval above int32: the Go type cannot decode the CR, so no request is sent", recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      intervalSeconds: 2147483648\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-limit-negative": func(h *harness) {
		h.Create("a limit of -1: Create sends it as text and the API answers 400", recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      limit: -1\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-duplicate-group-names": func(h *harness) {
		h.Create("two groups with one name", recordingRuleGroupSetManifest("  groups:\n    - name: same\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n    - name: same\n      rules:\n        - record: baseline_record2\n          expr: vector(2)\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-duplicate-set-name": func(h *harness) {
		recordingRuleGroupSets(h).seed("rules", "other")
		h.Create("another set has the name of the CR", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-rejected": func(h *harness) {
		recordingRuleGroupSets(h).answerNext(http.MethodPost, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Create("create returns an API error", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.Reconcile("retry creates the set")
	},
	"create-empty-id": func(h *harness) {
		recordingRuleGroupSets(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{})
		h.Create("create succeeds without an id", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.Reconcile("retry creates the set")
	},
	"update-groups": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.SetSpec("change a rule, add a group, drop nothing", recordingRuleGroupSetSpecMap(
			map[string]any{"name": "baseline.rules", "intervalSeconds": 180, "limit": 5, "rules": []any{
				map[string]any{"record": "baseline_record", "expr": "vector(3)", "labels": map[string]any{"team": "changed"}},
			}},
			ruleGroupSpec("second.rules", ruleSpec("second_record", "vector(2)")),
		))
		h.SetSpec("drop the first group: the API deletes it", recordingRuleGroupSetSpecMap(
			ruleGroupSpec("second.rules", ruleSpec("second_record", "vector(2)")),
		))
		h.RecordRemote("the remote set after the updates", recordingRuleGroupSets(h).snapshot())
	},
	"update-remove-optional-fields": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.SetSpec("remove interval, limit, and labels: the API stores 60, 0, and no labels",
			recordingRuleGroupSetSpecMap(ruleGroupSpec("baseline.rules", ruleSpec("baseline_record", "vector(1)"))))
		h.RecordRemote("the remote set after the update", recordingRuleGroupSets(h).snapshot())
	},
	"update-clear-rules": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.SetSpec("remove every rule of the group", recordingRuleGroupSetSpecMap(map[string]any{"name": "baseline.rules"}))
		h.RecordRemote("the remote group has no rules", recordingRuleGroupSets(h).snapshot())
	},
	"update-rejected": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		recordingRuleGroupSets(h).answerNext(http.MethodPut, http.StatusBadRequest, map[string]any{"code": 400, "message": "Bad Request: rejected"})
		h.SetSpec("update returns an API error", recordingRuleGroupSetSpecMap(ruleGroupSpec("baseline.rules", ruleSpec("baseline_record", "vector(2)"))))
		h.Reconcile("retry")
	},
	"update-invalid-group": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.SetSpec("update with an empty expression", recordingRuleGroupSetSpecMap(ruleGroupSpec("baseline.rules", ruleSpec("baseline_record", ""))))
		h.RecordRemote("the remote set keeps the old rule", recordingRuleGroupSets(h).snapshot())
	},
	"update-when-gone": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		recordingRuleGroupSets(h).remove(h.StatusID())
		h.SetSpec("update after remote deletion", recordingRuleGroupSetSpecMap(ruleGroupSpec("baseline.rules", ruleSpec("baseline_record", "vector(2)"))))
		h.Reconcile("recreate")
	},
	"upgrade-from-released": func(h *harness) {
		id := recordingRuleGroupSets(h).seed("rules", "baseline.rules")
		h.CreateReleased("the released operator made the set: reconcile the CR", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec),
			[]string{releasedRecordingRuleGroupSetFinalizer}, map[string]any{"id": id})
		h.Reconcile("resync")
		h.Delete("delete")
	},
	"bad-status-id": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		setRecordingRuleGroupSetStatusID(h, "not-an-id")
		h.Reconcile("update: the API answers 404 for an id that is no set, so the status id is cleared")
		h.Reconcile("recreate: the first set still has the name, so the API answers 400")
		setRecordingRuleGroupSetStatusID(h, "not-an-id")
		h.Delete("delete: a 404 is success")
	},
	"delete": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.Delete("delete")
		h.Reconcile("delete again")
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		recordingRuleGroupSets(h).remove(h.StatusID())
		h.Delete("delete after remote deletion")
	},
	"delete-rejected": func(h *harness) {
		h.Create("create", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		recordingRuleGroupSets(h).answerNext(http.MethodDelete, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Delete("delete returns an API error")
		h.Reconcile("retry deletes the set")
	},
	"selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
		h.Create("selector does not match", recordingRuleGroupSetManifest(recordingRuleGroupSetSpec))
		h.Reconcile("no remote call")
		h.Delete("delete")
	},
}

func recordingRuleGroupSetAdmission() map[string]string {
	rule := "        - record: baseline_record\n          expr: vector(1)\n"
	group := func(fields string) string {
		return recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n" + fields + "      rules:\n" + rule)
	}
	return map[string]string{
		"sample":                    recordingRuleGroupSetManifest(recordingRuleGroupSetSpec),
		"without spec":              "apiVersion: coralogix.com/v1alpha1\nkind: RecordingRuleGroupSet\nmetadata:\n  name: rules\n",
		"empty spec":                recordingRuleGroupSetManifest("  {}\n"),
		"groups empty":              recordingRuleGroupSetManifest("  groups: []\n"),
		"group without name":        recordingRuleGroupSetManifest("  groups:\n    - rules:\n" + rule),
		"group name empty":          recordingRuleGroupSetManifest("  groups:\n    - name: \"\"\n      rules:\n" + rule),
		"group without rules":       recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n"),
		"rules empty":               recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      rules: []\n"),
		"rule without record":       recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      rules:\n        - expr: vector(1)\n"),
		"rule without expr":         recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      rules:\n        - record: baseline_record\n"),
		"record empty":              recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      rules:\n        - record: \"\"\n          expr: vector(1)\n"),
		"expr empty":                recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      rules:\n        - record: baseline_record\n          expr: \"\"\n"),
		"labels with a number":      recordingRuleGroupSetManifest("  groups:\n    - name: baseline.rules\n      rules:\n        - record: baseline_record\n          expr: vector(1)\n          labels:\n            team: 1\n"),
		"interval 0":                group("      intervalSeconds: 0\n"),
		"interval below minimum":    group("      intervalSeconds: 30\n"),
		"interval negative":         group("      intervalSeconds: -1\n"),
		"interval above int32":      group("      intervalSeconds: 2147483648\n"),
		"interval as string":        group("      intervalSeconds: \"60\"\n"),
		"limit 0":                   group("      limit: 0\n"),
		"limit negative":            group("      limit: -1\n"),
		"limit as string":           group("      limit: \"5\"\n"),
		"duplicate group names":     recordingRuleGroupSetManifest("  groups:\n    - name: same\n      rules:\n" + rule + "    - name: same\n      rules:\n" + rule),
		"unknown field of a group":  group("      unknown: true\n"),
		"unknown field of the spec": recordingRuleGroupSetManifest(recordingRuleGroupSetSpec + "  name: set\n"),
	}
}
