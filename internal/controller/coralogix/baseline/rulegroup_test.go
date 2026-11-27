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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "rulegroup",
		crdFile:  "coralogix.com_rulegroups.yaml",
		resource: "rulegroups",
		newFake:  newRuleGroupFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.RuleGroupReconciler{RuleGroupClient: cs.RuleGroups(), Interval: reconcileInterval}
		},
		scenarios: ruleGroupScenarios(),
		admission: ruleGroupAdmission(),
		fidelity: fidelityConfig{
			captureField: "ruleGroup.id",
			deletePath:   func(id string) string { return ruleGroupPath + "/" + id },
			normalize:    normalizeRuleGroupAnswer,
		},
	})
}

var protoPosition = regexp.MustCompile(`\(line \d+:\d+\)`)

// normalizeRuleGroupAnswer hides the position in the request text that a protobuf parse error
// names. It depends on how the client wrote the JSON, and the fake cannot know it.
func normalizeRuleGroupAnswer(resp any) any {
	answer, ok := resp.(map[string]any)
	if !ok {
		return resp
	}
	if message, ok := answer["message"].(string); ok {
		answer["message"] = protoPosition.ReplaceAllString(message, "(line N:N)")
	}
	return answer
}

// ruleGroupSpec is a rule group with one subgroup of one rule.
const ruleGroupSpec = `  name: baseline-rule-group
  description: Baseline rule group
  subgroups:
    - rules:
        - name: worker-to-category
          description: Extracts the worker key
          jsonExtract:
            jsonKey: worker
            destinationField: Category
`

// ruleGroupAllRuleTypesSpec has a rule of every type with its non-default flags, and every enum value.
const ruleGroupAllRuleTypesSpec = `  name: baseline-rule-group
  subgroups:
    - rules:
        - name: parse
          parse: {sourceField: text, destinationField: text, regex: 'a(?P<b>\d+)'}
        - name: block-keep
          block: {sourceField: text, regex: secret, keepBlockedLogs: true}
        - name: allow
          block: {sourceField: text, regex: keep-me, blockingAllMatchingBlocks: false, keepBlockedLogs: true}
        - name: allow-default-keep
          block: {sourceField: text, regex: keep-me-too, blockingAllMatchingBlocks: false}
        - name: replace
          replace: {sourceField: text, destinationField: text, regex: 'x', replacementString: ''}
        - name: remove-fields
          removeFields: {excludedFields: [a, b.c]}
        - name: stringify-keep
          jsonStringify: {sourceField: text.a, destinationField: text.b, keepSourceField: true}
        - name: stringify-delete
          jsonStringify: {sourceField: text.a, destinationField: text.b}
        - name: extract
          extract: {sourceField: text, regex: '(?P<a>\d+)'}
    - rules:
        - name: parse-json-keep-both
          parseJsonField: {sourceField: text.a, destinationField: text.b, keepSourceField: true, keepDestinationField: true}
        - name: parse-json-keep-none
          parseJsonField: {sourceField: text.a, destinationField: text.b, keepSourceField: false, keepDestinationField: false}
        - name: category
          jsonExtract: {jsonKey: a, destinationField: Category}
        - name: class
          jsonExtract: {jsonKey: a, destinationField: CLASSNAME}
        - name: method
          jsonExtract: {jsonKey: a, destinationField: METHODNAME}
        - name: thread
          jsonExtract: {jsonKey: a, destinationField: THREADID}
        - name: severity
          jsonExtract: {jsonKey: a, destinationField: SEVERITY}
    - rules:
        - {name: ts-strftime, extractTimestamp: {sourceField: text, fieldFormatStandard: Strftime, timeFormat: '%Y'}}
        - {name: ts-java, extractTimestamp: {sourceField: text, fieldFormatStandard: JavaSDF, timeFormat: yyyy}}
        - {name: ts-golang, extractTimestamp: {sourceField: text, fieldFormatStandard: Golang, timeFormat: '2006'}}
        - {name: ts-seconds, extractTimestamp: {sourceField: text, fieldFormatStandard: SecondTS, timeFormat: ''}}
        - {name: ts-milli, extractTimestamp: {sourceField: text, fieldFormatStandard: MilliTS, timeFormat: ''}}
        - {name: ts-micro, extractTimestamp: {sourceField: text, fieldFormatStandard: MicroTS, timeFormat: ''}}
        - {name: ts-nano, extractTimestamp: {sourceField: text, fieldFormatStandard: NanoTS, timeFormat: ''}}
`

func ruleGroupManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: RuleGroup\nmetadata:\n  name: rulegroup\nspec:\n" + spec
}

func ruleGroups(h *harness) *ruleGroupFake {
	h.t.Helper()
	f, ok := h.API().(*ruleGroupFake)
	if !ok {
		h.t.Fatal("expected *ruleGroupFake")
	}
	return f
}

// ruleGroupSpecMap parses the YAML of a whole spec.
func ruleGroupSpecMap(h *harness, spec string) map[string]any {
	h.t.Helper()
	var out map[string]any
	if err := yaml.Unmarshal([]byte(spec), &out); err != nil {
		h.t.Fatal(err)
	}
	return out
}

func setRuleGroupStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatal(err)
	}
}

// ruleGroupSampleScenarios makes one scenario for each released sample: create the sample, then
// resync, which sends the whole group again with the id in the path.
func ruleGroupSampleScenarios() map[string]func(h *harness) {
	files, err := filepath.Glob(filepath.Join(repoRoot, "config", "samples", "v1alpha1", "rulegroups", "*.yaml"))
	if err != nil || len(files) == 0 {
		panic("the RuleGroup samples are missing")
	}
	sort.Strings(files)
	out := map[string]func(h *harness){}
	for _, file := range files {
		relative := strings.TrimPrefix(file, filepath.Join(repoRoot)+string(filepath.Separator))
		name := "sample-" + strings.ReplaceAll(strings.TrimSuffix(filepath.Base(file), ".yaml"), "_", "-")
		out[name] = func(h *harness) {
			h.CreateFromFile("create the released sample", filepath.ToSlash(relative))
			h.Reconcile("resync: update sends the whole group")
			h.Delete("delete")
		}
	}
	return out
}

func ruleGroupScenarios() map[string]func(h *harness) {
	scenarios := map[string]func(h *harness){
		"create-minimal": func(h *harness) {
			h.Create("create with the required fields only", ruleGroupManifest(`  name: baseline-rule-group
  subgroups:
    - rules:
        - name: worker-to-category
          jsonExtract:
            jsonKey: worker
            destinationField: Category
`))
			h.Reconcile("resync")
			h.Reconcile("resync again")
		},
		// Every field of the group. The matchers are the applications, then the subsystems, then
		// the severities, each as its own matcher.
		"create-all-fields": func(h *harness) {
			h.Create("create with every group field", ruleGroupManifest(`  name: baseline-rule-group
  description: Baseline rule group
  active: false
  hidden: true
  creator: baseline
  order: 3
  applications: [shop, checkout]
  subsystems: [payments]
  severities: [Debug, Verbose, Info, Warning, Error, Critical]
  subgroups:
    - active: false
      rules:
        - name: worker-to-category
          active: false
          jsonExtract:
            jsonKey: worker
            destinationField: Category
`))
			h.RecordRemote("the remote group", ruleGroups(h).snapshot())
		},
		// Every rule type with its non-default flags, and every enum value.
		"create-all-rule-types": func(h *harness) {
			h.Create("create with a rule of each type", ruleGroupManifest(ruleGroupAllRuleTypesSpec))
			h.RecordRemote("the remote group", ruleGroups(h).snapshot())
		},
		// Update sends the same rules: every type, flag and enum value reaches the Update body.
		"update-all-rule-types": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			h.SetSpec("update to a rule of each type", ruleGroupSpecMap(h, ruleGroupAllRuleTypesSpec))
			h.RecordRemote("the remote group", ruleGroups(h).snapshot())
		},
		// The spec can hold an id and an order for a subgroup. The requests send neither id: the
		// order is the position of the subgroup, and of the rule inside it.
		"create-ignores-subgroup-id-and-order": func(h *harness) {
			h.Create("subgroup id and order are not sent", ruleGroupManifest(`  name: baseline-rule-group
  subgroups:
    - id: ignored-id-1
      order: 7
      rules:
        - {name: first, extract: {sourceField: text, regex: 'a'}}
        - {name: second, extract: {sourceField: text, regex: 'b'}}
    - id: ignored-id-2
      order: 3
      rules:
        - {name: third, extract: {sourceField: text, regex: 'c'}}
`))
			h.RecordRemote("the remote group", ruleGroups(h).snapshot())
		},
		// A group without subgroups is a 400: the API needs at least one.
		"create-without-subgroups": func(h *harness) {
			h.Create("create without subgroups", ruleGroupManifest("  name: baseline-rule-group\n"))
			h.Reconcile("retry")
			h.Delete("delete")
		},
		"create-empty-subgroup": func(h *harness) {
			h.Create("create with a subgroup that has no rules", ruleGroupManifest("  name: baseline-rule-group\n  subgroups:\n    - {}\n"))
			h.Reconcile("retry")
			h.Delete("delete")
		},
		"create-empty-name": func(h *harness) {
			h.Create("admission accepts an empty name, the API rejects it", ruleGroupManifest(`  name: ""
  subgroups:
    - rules:
        - {name: a, extract: {sourceField: text, regex: 'a'}}
`))
			h.Reconcile("retry")
			h.Delete("delete")
		},
		// Admission accepts an empty rule name and an empty removeFields list, and the API rejects
		// both with a 400. The reconcile fails, the CR keeps no status id, and each retry sends the
		// same POST.
		"create-empty-rule-name": func(h *harness) {
			h.Create("admission accepts an empty rule name, the API rejects it", ruleGroupManifest(`  name: baseline-rule-group
  subgroups:
    - rules:
        - {name: "", extract: {sourceField: text, regex: 'a'}}
`))
			h.Reconcile("retry")
			h.Delete("delete")
		},
		"create-empty-remove-fields": func(h *harness) {
			h.Create("admission accepts an empty removeFields list, the API rejects it", ruleGroupManifest(`  name: baseline-rule-group
  subgroups:
    - rules:
        - {name: a, removeFields: {excludedFields: []}}
`))
			h.Reconcile("retry")
			h.Delete("delete")
		},
		// The same two rejections on an update: the CR keeps its status id, and the retry sends the
		// same PUT.
		"update-empty-rule-name-and-remove-fields": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			h.SetSpec("an empty rule name", ruleGroupSpecMap(h, `name: baseline-rule-group
subgroups:
  - rules:
      - {name: "", extract: {sourceField: text, regex: a}}
`))
			h.SetSpec("an empty removeFields list", ruleGroupSpecMap(h, `name: baseline-rule-group
subgroups:
  - rules:
      - {name: a, removeFields: {excludedFields: []}}
`))
			h.Reconcile("retry")
			h.RecordRemote("the remote group is unchanged", ruleGroups(h).snapshot())
		},
		"create-without-spec": func(h *harness) {
			h.Create("create without spec", "apiVersion: coralogix.com/v1alpha1\nkind: RuleGroup\nmetadata:\n  name: rulegroup\n")
			h.Reconcile("retry")
			h.Delete("delete")
		},
		"create-rejected": func(h *harness) {
			ruleGroups(h).answerNext(http.MethodPost, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
			h.Create("create returns an API error", ruleGroupManifest(ruleGroupSpec))
			h.Reconcile("retry creates the group")
		},
		"create-empty-id": func(h *harness) {
			ruleGroups(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{"ruleGroup": map[string]any{"id": ""}})
			h.Create("create succeeds with an empty id", ruleGroupManifest(ruleGroupSpec))
			h.Reconcile("retry")
		},
		"update-fields": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			h.SetSpec("rename, change the flags and the matchers, replace the subgroups", ruleGroupSpecMap(h, `name: renamed-rule-group
description: Changed
active: false
hidden: true
creator: someone
order: 2
applications: [shop]
severities: [Error]
subgroups:
  - rules:
      - {name: block, block: {sourceField: text, regex: secret}}
  - rules:
      - {name: replace, replace: {sourceField: text, destinationField: text, regex: x, replacementString: "y"}}
`))
			h.RecordRemote("the remote group", ruleGroups(h).snapshot())
		},
		// The API replaces the group, so each field that the update omits takes its default.
		"update-clear-optional-fields": func(h *harness) {
			h.Create("create with every optional field", ruleGroupManifest(`  name: baseline-rule-group
  description: Baseline rule group
  hidden: true
  creator: baseline
  order: 3
  applications: [shop]
  subsystems: [payments]
  severities: [Warning]
  subgroups:
    - rules:
        - {name: a, description: A rule, extract: {sourceField: text, regex: 'a'}}
`))
			h.SetSpec("remove the optional fields: update omits the order and the matchers", ruleGroupSpecMap(h, `name: baseline-rule-group
subgroups:
  - rules:
      - {name: a, extract: {sourceField: text, regex: a}}
`))
			h.RecordRemote("the remote group", ruleGroups(h).snapshot())
		},
		"update-rejected": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			h.SetSpec("an empty regular expression", ruleGroupSpecMap(h, `name: baseline-rule-group
subgroups:
  - rules:
      - {name: a, extract: {sourceField: text, regex: ""}}
`))
			h.Reconcile("retry")
			h.RecordRemote("the remote group is unchanged", ruleGroups(h).snapshot())
		},
		"update-when-gone": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			ruleGroups(h).forget(h.StatusID())
			h.SetSpec("update after remote deletion: the 404 removes status.id", ruleGroupSpecMap(h, `name: renamed-rule-group
subgroups:
  - rules:
      - {name: a, extract: {sourceField: text, regex: a}}
`))
			h.Reconcile("recreate")
		},
		// The update 404 removes status.id. A delete before the retry finds a CR with a finalizer
		// and no status id. It names no remote resource: the shared reconciler removes the
		// finalizer with no API call, and no new group is created.
		"delete-after-update-gone": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			ruleGroups(h).forget(h.StatusID())
			h.SetSpec("update after remote deletion: the 404 removes status.id", ruleGroupSpecMap(h, `name: renamed-rule-group
subgroups:
  - rules:
      - {name: a, extract: {sourceField: text, regex: a}}
`))
			h.Delete("delete a CR that has a finalizer and no status id: no group is created")
			h.RecordRemote("no remote group exists", ruleGroups(h).snapshot())
		},
		"bad-status-id": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			setRuleGroupStatusID(h, "not-a-uuid")
			h.Reconcile("update of an unknown id is a 404")
			h.Reconcile("recreate")
			h.Delete("delete")
		},
		// A chart upgrade: the released operator created the group and wrote status.id and its
		// finalizer. The operator updates that group, creates none, and deletes it on delete.
		"upgrade-from-released": func(h *harness) {
			const id = "22222222-2222-4222-8222-222222222222"
			ruleGroups(h).put(id, ruleGroupRecord{
				name: "baseline-rule-group", description: "Baseline rule group", enabled: true, order: 1,
				subgroups: []any{map[string]any{
					"id": "33333333-3333-4333-8333-333333333333", "enabled": true, "order": 1,
					"rules": []any{map[string]any{
						"id": "44444444-4444-4444-8444-444444444444", "name": "worker-to-category", "description": "Extracts the worker key",
						"enabled": true, "order": 1, "sourceField": "text",
						"parameters": map[string]any{"jsonExtractParameters": map[string]any{"destinationFieldType": "DESTINATION_FIELD_CATEGORY_OR_UNSPECIFIED", "rule": "worker"}},
					}},
				}},
			})
			h.CreateReleased("the released operator made the group: reconcile the CR", ruleGroupManifest(ruleGroupSpec),
				[]string{"rulegroup.coralogix.com/finalizer"}, map[string]any{"id": id, "printableStatus": "RemoteSynced"})
			if got := h.StatusID(); got != id {
				h.t.Fatalf("status.id changed from %s to %s", id, got)
			}
			h.Reconcile("reconcile again")
			h.Delete("delete: the group is deleted by its released id")
			h.RecordRemote("no group remains", ruleGroups(h).snapshot())
		},
		"delete": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			h.Delete("delete")
			h.Reconcile("delete again")
		},
		// The service reads the group before it deletes it, so a missing group is a 404, which the
		// operator accepts.
		"delete-when-gone": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			ruleGroups(h).forget(h.StatusID())
			h.Delete("delete after remote deletion: the API answers 404")
		},
		"delete-rejected": func(h *harness) {
			h.Create("create", ruleGroupManifest(ruleGroupSpec))
			ruleGroups(h).answerNext(http.MethodDelete, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
			h.Delete("delete returns an API error")
			h.Reconcile("retry deletes the group")
		},
		"selector-mismatch": func(h *harness) {
			h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
			h.Create("create: a direct reconcile does not run the event predicate", ruleGroupManifest(ruleGroupSpec))
			h.Reconcile("the selector does not match: delete the remote group and clear the status")
			h.Delete("delete")
		},
	}
	for name, scenario := range ruleGroupSampleScenarios() {
		scenarios[name] = scenario
	}
	return scenarios
}

func ruleGroupAdmission() map[string]string {
	const rule = "    - rules:\n        - name: r\n          extract: {sourceField: text, regex: a}\n"
	one := func(arm string) string {
		return ruleGroupManifest("  name: g\n  subgroups:\n    - rules:\n        - name: r\n          " + arm + "\n")
	}
	return map[string]string{
		"sample":                 ruleGroupManifest(ruleGroupSpec),
		"without spec":           "apiVersion: coralogix.com/v1alpha1\nkind: RuleGroup\nmetadata:\n  name: rulegroup\n",
		"empty spec":             ruleGroupManifest("  {}\n"),
		"name absent":            ruleGroupManifest("  subgroups:\n" + rule),
		"name empty":             ruleGroupManifest("  name: \"\"\n  subgroups:\n" + rule),
		"name only":              ruleGroupManifest("  name: g\n"),
		"subgroups empty":        ruleGroupManifest("  name: g\n  subgroups: []\n"),
		"subgroup without rules": ruleGroupManifest("  name: g\n  subgroups:\n    - {}\n"),
		"description empty":      ruleGroupManifest("  name: g\n  description: \"\"\n  subgroups:\n" + rule),
		"active false":           ruleGroupManifest("  name: g\n  active: false\n  subgroups:\n" + rule),
		"hidden true":            ruleGroupManifest("  name: g\n  hidden: true\n  subgroups:\n" + rule),
		"creator":                ruleGroupManifest("  name: g\n  creator: someone\n  subgroups:\n" + rule),
		"order 0":                ruleGroupManifest("  name: g\n  order: 0\n  subgroups:\n" + rule),
		"order 1":                ruleGroupManifest("  name: g\n  order: 1\n  subgroups:\n" + rule),
		"order negative":         ruleGroupManifest("  name: g\n  order: -1\n  subgroups:\n" + rule),
		"order too large":        ruleGroupManifest("  name: g\n  order: 2147483648\n  subgroups:\n" + rule),
		"applications":           ruleGroupManifest("  name: g\n  applications: [a, \"\"]\n  subgroups:\n" + rule),
		"subsystems":             ruleGroupManifest("  name: g\n  subsystems: [a]\n  subgroups:\n" + rule),
		"severity warning":       ruleGroupManifest("  name: g\n  severities: [Warning]\n  subgroups:\n" + rule),
		"severity unknown":       ruleGroupManifest("  name: g\n  severities: [Fatal]\n  subgroups:\n" + rule),
		"severity lower case":    ruleGroupManifest("  name: g\n  severities: [warning]\n  subgroups:\n" + rule),
		"subgroup id":            ruleGroupManifest("  name: g\n  subgroups:\n    - id: x\n      rules:\n        - {name: r, extract: {sourceField: text, regex: a}}\n"),
		"subgroup order 0":       ruleGroupManifest("  name: g\n  subgroups:\n    - order: 0\n      rules:\n        - {name: r, extract: {sourceField: text, regex: a}}\n"),
		"subgroup inactive":      ruleGroupManifest("  name: g\n  subgroups:\n    - active: false\n      rules:\n        - {name: r, extract: {sourceField: text, regex: a}}\n"),
		"rule name absent":       ruleGroupManifest("  name: g\n  subgroups:\n    - rules:\n        - extract: {sourceField: text, regex: a}\n"),
		"rule name empty":        ruleGroupManifest("  name: g\n  subgroups:\n    - rules:\n        - name: \"\"\n          extract: {sourceField: text, regex: a}\n"),
		"rule without type":      ruleGroupManifest("  name: g\n  subgroups:\n    - rules:\n        - name: r\n"),
		"rule with two types":    one("extract: {sourceField: text, regex: a}\n          parse: {sourceField: text, destinationField: text, regex: a}"),
		"rule inactive":          one("active: false\n          extract: {sourceField: text, regex: a}"),
		"parse":                  one("parse: {sourceField: text, destinationField: text, regex: a}"),
		"parse without regex":    one("parse: {sourceField: text, destinationField: text}"),
		"block":                  one("block: {sourceField: text, regex: a}"),
		"block allow":            one("block: {sourceField: text, regex: a, blockingAllMatchingBlocks: false, keepBlockedLogs: true}"),
		"block without source":   one("block: {regex: a}"),
		"json extract":           one("jsonExtract: {jsonKey: a, destinationField: Category}"),
		"json extract unknown":   one("jsonExtract: {jsonKey: a, destinationField: Text}"),
		"json extract no key":    one("jsonExtract: {destinationField: Category}"),
		"replace":                one("replace: {sourceField: text, destinationField: text, regex: a, replacementString: b}"),
		"replace no string":      one("replace: {sourceField: text, destinationField: text, regex: a}"),
		"extract timestamp":      one("extractTimestamp: {sourceField: text, fieldFormatStandard: Golang, timeFormat: x}"),
		"timestamp unknown":      one("extractTimestamp: {sourceField: text, fieldFormatStandard: Unix, timeFormat: x}"),
		"timestamp no format":    one("extractTimestamp: {sourceField: text, fieldFormatStandard: Golang}"),
		"remove fields":          one("removeFields: {excludedFields: [a]}"),
		"remove no fields":       one("removeFields: {}"),
		"remove empty fields":    one("removeFields: {excludedFields: []}"),
		"json stringify":         one("jsonStringify: {sourceField: text, destinationField: text}"),
		"json stringify keep":    one("jsonStringify: {sourceField: text, destinationField: text, keepSourceField: true}"),
		"extract":                one("extract: {sourceField: text, regex: a}"),
		"parse json field":       one("parseJsonField: {sourceField: text, destinationField: text, keepSourceField: true, keepDestinationField: false}"),
		"parse json no keep":     one("parseJsonField: {sourceField: text, destinationField: text}"),
	}
}

// TestRuleGroupCreateWithoutRuleGroup records a bug of the handwritten controller: it reads the
// id from createResponse.RuleGroup without a nil check, so a Create answer without ruleGroup
// panics the reconcile. controller-runtime recovers the panic and retries. A scenario cannot show
// it, because a panic ends the step before the golden has it.
func TestRuleGroupCreateWithoutRuleGroup(t *testing.T) {
	h := newHarness(t, kinds["rulegroup"], "create-without-rule-group")
	ruleGroups(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{})
	recovered := func() (value any) {
		defer func() { value = recover() }()
		h.Create("create answers without the ruleGroup", ruleGroupManifest(ruleGroupSpec))
		return nil
	}()
	if recovered == nil {
		t.Fatal("the reconcile did not panic: the controller now handles a Create answer without ruleGroup. Record the new behavior in a scenario.")
	}
}
