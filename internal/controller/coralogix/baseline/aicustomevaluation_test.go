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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const (
	aiCustomEvalPath     = "/ai/custom-evaluations/v3"
	aiApplicationsPath   = "/ai/applications/v3"
	aiCustomEvalNotFound = "Custom evaluation not found"
)

// The AI applications of the fake. The API links a custom evaluation only to an application that
// the team has, and no API creates one. The ids of the other applications start with
// fixedIDPrefix, so the goldens keep them, and a scenario shows which application a link names.
// The fidelity cases link the first one, and the recorder masks its id, so it has a usual UUID.
const (
	aiCustomEvalApplication   = "cxo-baseline-app"
	aiCustomEvalSubsystem     = "cxo-baseline-subsystem"
	aiCustomEvalApplicationID = "5e1ec7ed-0000-4000-8000-0000000000a1"
	aiCustomEvalOtherApp      = "cxo-baseline-other-app"
	aiCustomEvalOtherAppID    = fixedIDPrefix + "0000000000a2"
	aiCustomEvalTwinApp       = "cxo-baseline-twin-app"
	aiCustomEvalTwinAppID1    = fixedIDPrefix + "0000000000a3"
	aiCustomEvalTwinAppID2    = fixedIDPrefix + "0000000000a4"
	aiCustomEvalSampleAppID   = fixedIDPrefix + "0000000000a5"
)

func init() {
	register(kind{
		name:     "aicustomevaluation",
		crdFile:  "coralogix.com_aicustomevaluations.yaml",
		resource: "aicustomevaluations",
		newFake:  func() apiFake { return newAICustomEvaluationFake() },
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.AICustomEvaluationReconciler{Client: cs.AIEvaluations(), ClientSet: cs, Interval: reconcileInterval}
		},
		scenarios: aiCustomEvaluationScenarios(),
		admission: aiCustomEvaluationAdmission(),
		fidelity: fidelityConfig{
			captureField: "item.id",
			deletePath:   func(id string) string { return aiCustomEvalPath + "/" + id },
			recordVars:   aiCustomEvaluationRecordVars,
			fakeVars: map[string]string{
				"application":   aiCustomEvalApplication,
				"subsystem":     aiCustomEvalSubsystem,
				"applicationId": aiCustomEvalApplicationID,
			},
			normalize: aiCustomEvaluationNormalize,
		},
	})
}

// aiCustomEvaluationRecordVars names the AI application that the recorded cases link. Set
// CORALOGIX_AI_APPLICATION and CORALOGIX_AI_SUBSYSTEM, or the recorder takes the first application
// of the team.
func aiCustomEvaluationRecordVars(t *testing.T, base, apiKey string) map[string]string {
	t.Helper()
	status, apps, err := sendRecordRequest(base, apiKey, http.MethodGet, aiApplicationsPath+"?page_size=1000", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list AI applications: %d %v", status, err)
	}
	application, subsystem := os.Getenv("CORALOGIX_AI_APPLICATION"), os.Getenv("CORALOGIX_AI_SUBSYSTEM")
	for _, app := range listOf(apps, "aiApplications") {
		name, _ := app["application"].(string)
		sub, _ := app["subsystem"].(string)
		id, _ := app["id"].(string)
		if id == "" || name == "" || sub == "" || (application != "" && (name != application || sub != subsystem)) {
			continue
		}
		return map[string]string{"application": name, "subsystem": sub, "applicationId": id}
	}
	t.Fatal("the team has no AI application: set CORALOGIX_AI_APPLICATION and CORALOGIX_AI_SUBSYSTEM to one it has")
	return nil
}

// aiCustomEvaluationNormalize keeps the items of a List that a case made: the List of the API holds
// every custom evaluation of the team, and the team can have other ones. It writes each AI
// application as its field names and value types, once: the team has its own applications, and
// the List returns them in a new order on each call, so only the shape of a page compares.
func aiCustomEvaluationNormalize(resp any) any {
	m, ok := resp.(map[string]any)
	if !ok {
		return resp
	}
	keep := func(field string, match func(map[string]any) bool) {
		items, ok := m[field].([]any)
		if !ok {
			return
		}
		kept := []any{}
		for _, item := range items {
			if v, ok := item.(map[string]any); ok && match(v) {
				kept = append(kept, v)
			}
		}
		m[field] = kept
	}
	keep("items", func(v map[string]any) bool {
		name, _ := v["name"].(string)
		return strings.Contains(name, "{{run}}")
	})
	if apps, ok := m["aiApplications"].([]any); ok {
		shapes, seen := []any{}, map[string]bool{}
		for _, app := range apps {
			v, ok := app.(map[string]any)
			if !ok {
				continue
			}
			shape := map[string]any{}
			for field, value := range v {
				shape[field] = fmt.Sprintf("<%T>", value)
			}
			key := fmt.Sprint(shape) // fmt prints map keys sorted
			if !seen[key] {
				seen[key] = true
				shapes = append(shapes, shape)
			}
		}
		m["aiApplications"] = shapes
	}
	return m
}

// aiCustomEvaluationManifest returns an AICustomEvaluation CR named "custom-evaluation". spec is
// YAML, indented by two spaces.
func aiCustomEvaluationManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: AICustomEvaluation\nmetadata:\n  name: custom-evaluation\nspec:\n" + spec
}

const aiCustomEvalMinimalSpec = "  name: Competitor policy\n  policyType: quality\n  instructions: Check whether {response} names a competitor.\n"

var aiCustomEvalMinimalManifest = aiCustomEvaluationManifest(aiCustomEvalMinimalSpec)

// aiCustomEvalSpecMap is the minimal spec with the fields given.
func aiCustomEvalSpecMap(fields ...any) map[string]any {
	spec := map[string]any{
		"name":         "Competitor policy",
		"policyType":   "quality",
		"instructions": "Check whether {response} names a competitor.",
	}
	for i := 0; i+1 < len(fields); i += 2 {
		spec[fields[i].(string)] = fields[i+1]
	}
	return spec
}

func aiCustomEvalSelector(application, subsystem string) map[string]any {
	return map[string]any{"application": application, "subsystem": subsystem}
}

var aiCustomEvalFullCriteria = map[string]any{
	"acceptable": map[string]any{"flags": "Stays on our product.", "examples": []any{"User: which tool?\nAssistant: Ours.", "User: help\nAssistant: Sure."}},
	"prohibited": map[string]any{"flags": "Names a competitor.", "examples": []any{"User: which tool?\nAssistant: CompetitorX."}},
}

func aiCustomEvaluations(h *harness) *aiCustomEvaluationFake {
	h.t.Helper()
	f, ok := h.API().(*aiCustomEvaluationFake)
	if !ok {
		h.t.Fatal("expected *aiCustomEvaluationFake")
	}
	return f
}

func aiCustomEvaluationScenarios() map[string]func(h *harness) {
	withApp := aiCustomEvaluationManifest(aiCustomEvalMinimalSpec +
		"  applications:\n  - application: " + aiCustomEvalApplication + "\n    subsystem: " + aiCustomEvalSubsystem + "\n")
	return map[string]func(h *harness){
		"create-sample": func(h *harness) {
			h.CreateFromFile("create the sample", "config/samples/v1alpha1/aicustomevaluations/competitor-policy.yaml")
			h.Reconcile("reconcile again with no change")
		},

		"create-minimal": func(h *harness) {
			h.Create("create with no description, criteria, or applications: shouldIncludeSystemPrompt defaults to false", aiCustomEvalMinimalManifest)
			h.Reconcile("reconcile again with no change: the examples are cleared again")
		},

		"create-full": func(h *harness) {
			h.Create("create with every field", aiCustomEvaluationManifest(aiCustomEvalMinimalSpec+
				"  description: Flags competitors.\n  shouldIncludeSystemPrompt: true\n"+
				"  applications:\n  - application: "+aiCustomEvalOtherApp+"\n    subsystem: "+aiCustomEvalSubsystem+
				"\n  - application: "+aiCustomEvalApplication+"\n    subsystem: "+aiCustomEvalSubsystem+"\n"+
				"  criteria:\n    acceptable:\n      flags: Stays on our product.\n      examples: [\"good one\", \"good two\"]\n"+
				"    prohibited:\n      flags: Names a competitor.\n      examples: [\"bad one\"]\n"))
			h.Reconcile("reconcile again with no change")
		},

		"criteria-forms": func(h *harness) {
			h.Create("create with only acceptable flags", aiCustomEvaluationManifest(aiCustomEvalMinimalSpec+"  criteria:\n    acceptable:\n      flags: Stays on our product.\n"))
			h.SetSpec("only prohibited examples", aiCustomEvalSpecMap("criteria", map[string]any{"prohibited": map[string]any{"examples": []any{"bad one"}}}))
			h.SetSpec("only acceptable examples", aiCustomEvalSpecMap("criteria", map[string]any{"acceptable": map[string]any{"examples": []any{"good one"}}}))
			h.SetSpec("empty criteria", aiCustomEvalSpecMap("criteria", map[string]any{}))
			h.SetSpec("empty criterion objects", aiCustomEvalSpecMap("criteria", map[string]any{"acceptable": map[string]any{}, "prohibited": map[string]any{}}))
		},

		"update-fields": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			h.SetSpec("change every field", aiCustomEvalSpecMap(
				"name", "Security policy", "policyType", "security", "instructions", "Check {prompt} and {chat_history}.",
				"description", "Flags leaks.", "shouldIncludeSystemPrompt", true, "criteria", aiCustomEvalFullCriteria))
			h.SetSpec("change the examples", aiCustomEvalSpecMap(
				"name", "Security policy", "policyType", "security", "instructions", "Check {prompt} and {chat_history}.",
				"description", "Flags leaks.", "shouldIncludeSystemPrompt", true, "criteria", map[string]any{
					"acceptable": map[string]any{"flags": "Stays on our product.", "examples": []any{"good three"}},
					"prohibited": map[string]any{"flags": "Names a competitor.", "examples": []any{"bad two", "bad three"}},
				}))
			h.SetSpec("remove the optional fields: the description and the flags become empty, and the examples are cleared", aiCustomEvalSpecMap())
		},

		"update-applications": func(h *harness) {
			h.Create("create with one application", withApp)
			h.SetSpec("add an application", aiCustomEvalSpecMap("applications", []any{
				aiCustomEvalSelector(aiCustomEvalApplication, aiCustomEvalSubsystem), aiCustomEvalSelector(aiCustomEvalOtherApp, aiCustomEvalSubsystem)}))
			h.SetSpec("replace the applications", aiCustomEvalSpecMap("applications", []any{aiCustomEvalSelector(aiCustomEvalOtherApp, aiCustomEvalSubsystem)}))
			h.SetSpec("remove the applications", aiCustomEvalSpecMap())
		},

		"resync-relinks": func(h *harness) {
			h.Create("create with one application", withApp)
			aiCustomEvaluations(h).setLinks(h.StatusID(), aiCustomEvalOtherAppID)
			h.Reconcile("resync: another client replaced the link")
		},

		"create-unknown-application": func(h *harness) {
			h.Create("create for an application that the team does not have", aiCustomEvaluationManifest(aiCustomEvalMinimalSpec+
				"  applications:\n  - application: unknown-app\n    subsystem: "+aiCustomEvalSubsystem+"\n"))
			h.Reconcile("reconcile again")
			h.Delete("delete")
		},

		"update-unknown-application": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			h.SetSpec("link an application that the team does not have", aiCustomEvalSpecMap("applications", []any{aiCustomEvalSelector(aiCustomEvalApplication, "unknown-subsystem")}))
		},

		"ambiguous-application": func(h *harness) {
			h.Create("create for an application and subsystem that two applications of the team have", aiCustomEvaluationManifest(aiCustomEvalMinimalSpec+
				"  applications:\n  - application: "+aiCustomEvalTwinApp+"\n    subsystem: "+aiCustomEvalSubsystem+"\n"))
		},

		"applications-over-one-page": func(h *harness) {
			aiCustomEvaluations(h).addApplications(250)
			h.Create("create when the team has more applications than one page holds", withApp)
		},

		"list-applications-rejected": func(h *harness) {
			aiCustomEvaluations(h).answerNextError("applications", http.StatusInternalServerError, "the API fails")
			h.Create("create: the List of AI applications fails", withApp)
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"create-conflict": func(h *harness) {
			aiCustomEvaluations(h).seed("Competitor policy")
			h.Create("create: another custom evaluation has the name", aiCustomEvalMinimalManifest)
			h.Reconcile("reconcile again")
			h.Delete("delete")
		},

		"update-conflict": func(h *harness) {
			aiCustomEvaluations(h).seed("Taken name")
			h.Create("create", aiCustomEvalMinimalManifest)
			h.SetSpec("rename to the name of another custom evaluation", aiCustomEvalSpecMap("name", "Taken name"))
		},

		"create-rejected": func(h *harness) {
			aiCustomEvaluations(h).answerNextError("create", http.StatusBadRequest, "the API rejects this request")
			h.Create("create: the API rejects the Create", aiCustomEvalMinimalManifest)
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"create-empty-id": func(h *harness) {
			aiCustomEvaluations(h).answerNext("create", map[string]any{"item": map[string]any{"id": ""}})
			h.Create("create: the answer has an empty id", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).answerNext("create", map[string]any{})
			h.Reconcile("reconcile again: the answer has no custom evaluation")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"create-empty-id-after-remote-create": func(h *harness) {
			aiCustomEvaluations(h).loseNextID()
			h.Create("create: the API stores the custom evaluation, and the answer has an empty id", aiCustomEvalMinimalManifest)
			h.Reconcile("reconcile again: the stored custom evaluation has the name")
			h.Delete("delete: the operator has no id of the stored custom evaluation")
		},

		"create-malformed-response": func(h *harness) {
			aiCustomEvaluations(h).answerNext("create", "not a custom evaluation")
			h.Create("create: the answer cannot be decoded", aiCustomEvalMinimalManifest)
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"update-rejected": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).answerNextError("update", http.StatusBadRequest, "the API rejects this request")
			h.SetSpec("change the name: the API rejects the Update", aiCustomEvalSpecMap("name", "Renamed"))
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"update-malformed-response": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).answerNext("update", "not a custom evaluation")
			h.Reconcile("update: the answer cannot be decoded")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"list-rejected": func(h *harness) {
			h.Create("create", withApp)
			aiCustomEvaluations(h).answerNextError("list", http.StatusInternalServerError, "the API fails")
			h.Reconcile("resync: the List of custom evaluations fails")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"list-misses-the-custom-evaluation": func(h *harness) {
			h.Create("create", withApp)
			aiCustomEvaluations(h).answerNext("list", map[string]any{"items": []any{}})
			h.Reconcile("resync: the List has no custom evaluation with the id")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"link-rejected": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).answerNextError("link", http.StatusConflict, "Custom evaluation is already linked to this application")
			h.SetSpec("add an application: the API rejects the link", aiCustomEvalSpecMap("applications", []any{aiCustomEvalSelector(aiCustomEvalApplication, aiCustomEvalSubsystem)}))
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"link-partly-rejected": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).answerErrorAfter("link", 1, http.StatusInternalServerError, "the API fails")
			h.SetSpec("add two applications: the first link is stored, and the API fails the second", aiCustomEvalSpecMap("applications", []any{
				aiCustomEvalSelector(aiCustomEvalApplication, aiCustomEvalSubsystem), aiCustomEvalSelector(aiCustomEvalOtherApp, aiCustomEvalSubsystem)}))
			aiCustomEvaluations(h).expectLinks(h, aiCustomEvalOtherAppID)
			h.Reconcile("reconcile again: the List has the first link, so only the second is sent")
			aiCustomEvaluations(h).expectLinks(h, aiCustomEvalOtherAppID, aiCustomEvalApplicationID)
		},

		"unlink-rejected": func(h *harness) {
			h.Create("create", withApp)
			aiCustomEvaluations(h).answerNextError("unlink", http.StatusNotFound, "Custom evaluation is not linked to this application")
			h.SetSpec("remove the application: the unlink answers 404, a hook error, so the status id stays", aiCustomEvalSpecMap())
			h.Reconcile("reconcile again: the unlink is sent again")
		},

		"delete": func(h *harness) {
			h.Create("create", withApp)
			h.Delete("delete")
		},

		"delete-when-gone": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).forget(h.StatusID())
			h.Delete("delete: the custom evaluation was deleted outside the operator")
		},

		"update-when-gone": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).forget(h.StatusID())
			h.SetSpec("change the name: the custom evaluation was deleted outside the operator", aiCustomEvalSpecMap("name", "Renamed"))
			h.Reconcile("reconcile again")
		},

		"resync-when-gone": func(h *harness) {
			h.Create("create", withApp)
			aiCustomEvaluations(h).forget(h.StatusID())
			h.Reconcile("resync: the custom evaluation was deleted outside the operator")
			h.Reconcile("reconcile again")
		},

		"delete-after-update-when-gone": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).forget(h.StatusID())
			h.SetSpec("change the name: the custom evaluation was deleted outside the operator", aiCustomEvalSpecMap("name", "Renamed"))
			h.Delete("delete")
			h.Reconcile("reconcile again")
		},

		"delete-without-id": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			clearStatusID(h, h.key)
			h.Delete("delete a CR that has a finalizer and no status id")
			h.Reconcile("reconcile again")
		},

		"delete-rejected": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).answerNextError("delete", http.StatusBadRequest, "the API rejects this request")
			h.Delete("delete: the API rejects the Delete")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"delete-malformed-response": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			aiCustomEvaluations(h).answerNext("delete", "not an answer")
			h.Delete("delete: the answer cannot be decoded")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"bad-status-id-update": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			setStatusID(h, h.key, "not-a-uuid")
			h.Reconcile("update with a status id that is not a UUID")
			h.Reconcile("reconcile again")
		},

		"bad-status-id-delete": func(h *harness) {
			h.Create("create", aiCustomEvalMinimalManifest)
			setStatusID(h, h.key, "not-a-uuid")
			h.Delete("delete with a status id that is not a UUID")
			h.Reconcile("reconcile again")
		},

		"selector-mismatch": func(h *harness) {
			h.Create("create", withApp)
			h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "ai"})})
			h.Reconcile("reconcile after the selector stops matching the CR")
		},
	}
}

// aiCustomEvaluationAdmission lists the CRs that TestAdmission creates. Each case has one reason
// that the CRD accepts or rejects it.
func aiCustomEvaluationAdmission() map[string]string {
	spec := func(name, policyType, instructions string) string {
		s := ""
		if name != "" {
			s += "  name: " + name + "\n"
		}
		if policyType != "" {
			s += "  policyType: " + policyType + "\n"
		}
		if instructions != "" {
			s += "  instructions: " + instructions + "\n"
		}
		return s
	}
	ok := aiCustomEvalMinimalSpec
	with := func(extra string) string { return aiCustomEvaluationManifest(ok + extra) }
	criterion := func(side, body string) string { return with("  criteria:\n    " + side + ":\n" + body) }
	return map[string]string{
		"minimal":                               aiCustomEvaluationManifest(ok),
		"no name":                               aiCustomEvaluationManifest(spec("", "quality", "Check {response}.")),
		"no policyType":                         aiCustomEvaluationManifest(spec("n", "", "Check {response}.")),
		"no instructions":                       aiCustomEvaluationManifest(spec("n", "quality", "")),
		"empty name":                            aiCustomEvaluationManifest(spec(`""`, "quality", "Check {response}.")),
		"name of 256 characters":                aiCustomEvaluationManifest(spec(strings.Repeat("n", 256), "quality", "Check {response}.")),
		"name of 257 characters":                aiCustomEvaluationManifest(spec(strings.Repeat("n", 257), "quality", "Check {response}.")),
		"name of spaces":                        aiCustomEvaluationManifest(spec(`"   "`, "quality", "Check {response}.")),
		"policyType security":                   aiCustomEvaluationManifest(spec("n", "security", "Check {response}.")),
		"policyType other":                      aiCustomEvaluationManifest(spec("n", "other", "Check {response}.")),
		"policyType in upper case":              aiCustomEvaluationManifest(spec("n", "QUALITY", "Check {response}.")),
		"instructions with {prompt}":            aiCustomEvaluationManifest(spec("n", "quality", "Check {prompt}.")),
		"instructions with {chat_history}":      aiCustomEvaluationManifest(spec("n", "quality", "Check {chat_history}.")),
		"instructions without a placeholder":    aiCustomEvaluationManifest(spec("n", "quality", "Check the answer.")),
		"instructions with an unknown one":      aiCustomEvaluationManifest(spec("n", "quality", "Check {answer}.")),
		"instructions of only a placeholder":    aiCustomEvaluationManifest(spec("n", "quality", `"{response}"`)),
		"description":                           with("  description: Flags competitors.\n"),
		"empty description":                     with("  description: \"\"\n"),
		"description of 65537 characters":       with("  description: " + strings.Repeat("d", 65537) + "\n"),
		"shouldIncludeSystemPrompt true":        with("  shouldIncludeSystemPrompt: true\n"),
		"shouldIncludeSystemPrompt as a string": with("  shouldIncludeSystemPrompt: \"yes\"\n"),
		"one application":                       with("  applications:\n  - application: app\n    subsystem: sub\n"),
		"no applications":                       with("  applications: []\n"),
		"application without subsystem":         with("  applications:\n  - application: app\n"),
		"application with empty name":           with("  applications:\n  - application: \"\"\n    subsystem: sub\n"),
		"application of 257 characters":         with("  applications:\n  - application: " + strings.Repeat("a", 257) + "\n    subsystem: sub\n"),
		"repeated application":                  with("  applications:\n  - application: app\n    subsystem: sub\n  - application: app\n    subsystem: sub\n"),
		"one application in two subsystems":     with("  applications:\n  - application: app\n    subsystem: one\n  - application: app\n    subsystem: two\n"),
		"1025 applications":                     with("  applications:\n" + aiCustomEvalSelectors(1025)),
		"empty criteria":                        with("  criteria: {}\n"),
		"criteria with an unknown side":         with("  criteria:\n    neutral:\n      flags: f\n"),
		"acceptable flags":                      criterion("acceptable", "      flags: Stays on our product.\n"),
		"empty acceptable flags":                criterion("acceptable", "      flags: \"\"\n"),
		"flags of 65537 characters":             criterion("prohibited", "      flags: "+strings.Repeat("f", 65537)+"\n"),
		"empty example":                         criterion("acceptable", "      examples: [\"\"]\n"),
		"example of 65537 characters":           criterion("acceptable", "      examples: ["+strings.Repeat("e", 65537)+"]\n"),
		"repeated example":                      criterion("acceptable", "      examples: [a, a]\n"),
		"no examples":                           criterion("acceptable", "      examples: []\n"),
		"101 acceptable examples":               criterion("acceptable", "      examples: ["+aiCustomEvalExamples("a", 101)+"]\n"),
		"100 acceptable examples":               criterion("acceptable", "      examples: ["+aiCustomEvalExamples("a", 100)+"]\n"),
		"100 examples on both sides together": with("  criteria:\n    acceptable:\n      examples: [" + aiCustomEvalExamples("a", 50) +
			"]\n    prohibited:\n      examples: [" + aiCustomEvalExamples("p", 50) + "]\n"),
		"101 examples on both sides together": with("  criteria:\n    acceptable:\n      examples: [" + aiCustomEvalExamples("a", 51) +
			"]\n    prohibited:\n      examples: [" + aiCustomEvalExamples("p", 50) + "]\n"),
		"unknown spec field is not an error": with("  threshold: \"0.5\"\n"),
	}
}

func aiCustomEvalExamples(prefix string, n int) string {
	items := make([]string, n)
	for i := range items {
		items[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return strings.Join(items, ", ")
}

func aiCustomEvalSelectors(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "  - application: app%d\n    subsystem: sub\n", i)
	}
	return b.String()
}

// aiCustomEvaluationFake is the AI custom evaluations API and the List of AI applications,
// written from the SDK's openapi.yaml.
//
//	GET    /ai/custom-evaluations/v3                                      -> {items}
//	POST   /ai/custom-evaluations/v3            {name, description?, instructions?, policyType?,
//	                                             safe?, violates?, examples?,
//	                                             shouldIncludeSystemPrompt?, applicationIds?} -> {item}
//	PATCH  /ai/custom-evaluations/v3/{id}       {any of the fields but applicationIds, updateMask?} -> {item}
//	DELETE /ai/custom-evaluations/v3/{id}                                 -> {id}
//	POST   /ai/custom-evaluations/v3/{id}/applications/{applicationId}   -> {id, applicationId}
//	DELETE /ai/custom-evaluations/v3/{id}/applications/{applicationId}   -> {id, applicationId}
//	GET    /ai/applications/v3?page_size=&page_offset=                     -> {aiApplications}
//
// The status codes, error texts, and stored values follow fidelity/recorded.json, which the real
// API answered:
//   - No call reads one custom evaluation: the List holds every custom evaluation of the team.
//   - The response nests instructions, policyType, safe, violates, examples, and
//     shouldIncludeSystemPrompt in config. A field that Create leaves out is "", false, or [].
//   - A PATCH with no updateMask updates each field in the body. An empty examples list is the
//     same as no list, so it keeps the stored examples. With an updateMask, only the listed fields
//     change, and a listed field that the body leaves out is cleared.
//   - The answer of a PATCH has no links: its applicationIds is always empty.
//   - The name is unique in the team: Create and a rename answer 409.
//   - An application id that the team does not have is a 404, on Create and on a link. A link that
//     exists is a 409, and an unlink of a link that does not exist is a 404.
//   - page_offset of the List of AI applications counts applications, not pages.
type aiCustomEvaluationFake struct {
	mu           sync.Mutex
	evaluations  map[string]map[string]any
	order        []string
	applications []map[string]any
	// next holds one forced answer per route: create, list, update, delete, link, unlink, or
	// applications. The next request of the route gets it, and the fake does not run the request.
	next map[string]forcedAnswer
	// pass holds, per route, how many requests run before the forced answer of next applies.
	pass map[string]int
	// loseNextCreatedID makes the next Create store the custom evaluation and answer with an
	// empty id.
	loseNextCreatedID bool
}

func newAICustomEvaluationFake() *aiCustomEvaluationFake {
	app := func(id, application, subsystem string) map[string]any {
		return aiCustomEvalApp(id, application, subsystem)
	}
	return &aiCustomEvaluationFake{
		evaluations: map[string]map[string]any{},
		applications: []map[string]any{
			app(aiCustomEvalApplicationID, aiCustomEvalApplication, aiCustomEvalSubsystem),
			app(aiCustomEvalOtherAppID, aiCustomEvalOtherApp, aiCustomEvalSubsystem),
			app(aiCustomEvalTwinAppID1, aiCustomEvalTwinApp, aiCustomEvalSubsystem),
			app(aiCustomEvalTwinAppID2, aiCustomEvalTwinApp, aiCustomEvalSubsystem),
			app(aiCustomEvalSampleAppID, "ai-center-demo", "demo-runner"), // the sample
		},
		next: map[string]forcedAnswer{},
		pass: map[string]int{},
	}
}

// seed stores a custom evaluation with the name, as if another client made it.
func (f *aiCustomEvaluationFake) seed(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if status, resp := f.create(map[string]any{"name": name, "instructions": "x", "policyType": "quality"}); status != http.StatusOK {
		panic(fmt.Sprintf("seed: %d %v", status, resp))
	}
}

// addApplications adds n AI applications before the others, as a team with many applications has.
func (f *aiCustomEvaluationFake) addApplications(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	extra := make([]map[string]any, 0, n+len(f.applications))
	for i := 0; i < n; i++ {
		extra = append(extra, aiCustomEvalApp(fmt.Sprintf("%s%012d", fixedIDPrefix, i), fmt.Sprintf("cxo-baseline-app-%03d", i), aiCustomEvalSubsystem))
	}
	f.applications = append(extra, f.applications...)
}

// setLinks replaces the links of a custom evaluation without a request, as another client could.
func (f *aiCustomEvaluationFake) setLinks(id string, applicationIDs ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]any, len(applicationIDs))
	for i, a := range applicationIDs {
		ids[i] = a
	}
	f.evaluations[id]["applicationIds"] = ids
}

// expectLinks fails the test unless the custom evaluation of the CR is linked to exactly the
// applications given, in that order.
func (f *aiCustomEvaluationFake) expectLinks(h *harness, applicationIDs ...string) {
	h.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	want := make([]any, len(applicationIDs))
	for i, a := range applicationIDs {
		want[i] = a
	}
	got, _ := f.evaluations[h.StatusID()]["applicationIds"].([]any)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		h.t.Errorf("links: got %v, want %v", got, want)
	}
}

// forget deletes a custom evaluation without a request, as if it was deleted outside the operator.
func (f *aiCustomEvaluationFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.evaluations, id)
}

func (f *aiCustomEvaluationFake) answerNext(route string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[route] = forcedAnswer{status: http.StatusOK, body: body}
}

func (f *aiCustomEvaluationFake) loseNextID() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loseNextCreatedID = true
}

func (f *aiCustomEvaluationFake) answerNextError(route string, status int, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := aiCustomEvalError(status, reason)
	f.next[route] = forcedAnswer{status: st, body: body}
}

// answerErrorAfter lets n requests of the route run, then answers the next one with the error.
func (f *aiCustomEvaluationFake) answerErrorAfter(route string, n, status int, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := aiCustomEvalError(status, reason)
	f.next[route] = forcedAnswer{status: st, body: body}
	f.pass[route] = n
}

// aiCustomEvalError is an error answer of the API. A 409 starts with "Already Exists", as the
// gRPC code that the gateway maps to it.
func aiCustomEvalError(status int, reason string) (int, any) {
	if status == http.StatusConflict {
		return status, map[string]any{"code": status, "message": "Already Exists: " + reason}
	}
	return apiError(status, reason)
}

// route names the call of a request, and returns the custom evaluation id and the application id
// of its path.
func aiCustomEvalRoute(method, path string) (route, id, applicationID string) {
	if path == aiApplicationsPath && method == http.MethodGet {
		return "applications", "", ""
	}
	rest, ok := strings.CutPrefix(path, aiCustomEvalPath)
	if !ok {
		return "", "", ""
	}
	if rest == "" {
		switch method {
		case http.MethodGet:
			return "list", "", ""
		case http.MethodPost:
			return "create", "", ""
		}
		return "", "", ""
	}
	parts := strings.Split(strings.TrimPrefix(rest, "/"), "/")
	switch {
	case len(parts) == 1 && method == http.MethodPatch:
		return "update", parts[0], ""
	case len(parts) == 1 && method == http.MethodDelete:
		return "delete", parts[0], ""
	case len(parts) == 3 && parts[1] == "applications" && method == http.MethodPost:
		return "link", parts[0], parts[2]
	case len(parts) == 3 && parts[1] == "applications" && method == http.MethodDelete:
		return "unlink", parts[0], parts[2]
	}
	return "", "", ""
}

func (f *aiCustomEvaluationFake) handle(method, path string, query url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	route, id, applicationID := aiCustomEvalRoute(method, path)
	if route == "" {
		return routeNotFound()
	}
	if forced, ok := f.next[route]; ok && f.pass[route] > 0 {
		f.pass[route]--
	} else if ok {
		delete(f.next, route)
		return forced.status, forced.body
	}
	in, _ := body.(map[string]any)
	switch route {
	case "applications":
		return f.listApplications(query)
	case "list":
		items := make([]any, 0, len(f.order))
		for _, id := range f.order {
			if stored, ok := f.evaluations[id]; ok {
				items = append(items, aiCustomEvalCopy(stored))
			}
		}
		return http.StatusOK, map[string]any{"items": items}
	case "create":
		status, resp := f.create(in)
		if f.loseNextCreatedID && status == http.StatusOK {
			f.loseNextCreatedID = false
			created := copyMap(resp.(map[string]any)["item"].(map[string]any))
			created["id"] = ""
			resp = map[string]any{"item": created}
		}
		return status, resp
	case "update":
		return f.update(id, in)
	case "delete":
		if !aiEvalUUID.MatchString(id) {
			return apiError(http.StatusNotFound, "Resource not found: invalid ID")
		}
		if _, ok := f.evaluations[id]; !ok {
			return apiError(http.StatusNotFound, aiCustomEvalNotFound)
		}
		delete(f.evaluations, id)
		return http.StatusOK, map[string]any{"id": id}
	case "link", "unlink":
		return f.link(route == "link", id, applicationID)
	}
	return routeNotFound()
}

// listApplications pages the AI applications. page_offset is the number of applications to skip.
func (f *aiCustomEvaluationFake) listApplications(query url.Values) (int, any) {
	size, offset := len(f.applications), 0
	if v, err := strconv.Atoi(query.Get("page_size")); err == nil {
		size = v
	}
	if v, err := strconv.Atoi(query.Get("page_offset")); err == nil {
		offset = v
	}
	page := []any{}
	for i := offset; i < len(f.applications) && i < offset+size; i++ {
		page = append(page, copyMap(f.applications[i]))
	}
	return http.StatusOK, map[string]any{"aiApplications": page}
}

// aiCustomEvalApp is an AI application as the List returns it.
func aiCustomEvalApp(id, application, subsystem string) map[string]any {
	return map[string]any{
		"id": id, "companyId": "1", "application": application, "subsystem": subsystem,
		"createdAt": "2026-01-01T00:00:00.000Z", "updatedAt": "2026-01-01T00:00:00.000Z",
		"guardrailsIntegrated": false, "icon": "ICON_SYSTEM_KUBERNETES", "color": "COLOR_SOFT_ORANGE",
	}
}

func (f *aiCustomEvaluationFake) hasApplication(id string) bool {
	for _, app := range f.applications {
		if app["id"] == id {
			return true
		}
	}
	return false
}

func (f *aiCustomEvaluationFake) nameTaken(name, except string) bool {
	for id, other := range f.evaluations {
		if id != except && other["name"] == name {
			return true
		}
	}
	return false
}

func aiCustomEvalConflict(name string) (int, any) {
	return aiCustomEvalError(http.StatusConflict, fmt.Sprintf("A custom evaluation with name '%s' already exists", name))
}

// aiCustomEvalConfigFields are the request fields that the response nests in config, with the
// value that a missing field gets.
var aiCustomEvalConfigFields = []struct {
	name string
	zero any
}{
	{"instructions", ""},
	{"violates", ""},
	{"safe", ""},
	{"examples", []any{}},
	{"shouldIncludeSystemPrompt", false},
	{"policyType", ""},
}

func (f *aiCustomEvaluationFake) create(in map[string]any) (int, any) {
	name, _ := in["name"].(string)
	if name == "" {
		return apiError(http.StatusBadRequest, "Name is required")
	}
	if f.nameTaken(name, "") {
		return aiCustomEvalConflict(name)
	}
	links := []any{}
	if ids, ok := in["applicationIds"].([]any); ok {
		for _, id := range ids {
			if !f.hasApplication(fmt.Sprint(id)) {
				return apiError(http.StatusNotFound, "AI application not found")
			}
			links = append(links, id)
		}
	}
	config := map[string]any{}
	for _, field := range aiCustomEvalConfigFields {
		if v, ok := in[field.name]; ok {
			config[field.name] = v
		} else {
			config[field.name] = field.zero
		}
	}
	description, _ := in["description"].(string)
	stored := map[string]any{
		"id":             newUUID(),
		"name":           name,
		"description":    description,
		"config":         config,
		"applicationIds": links,
	}
	id := stored["id"].(string)
	f.evaluations[id] = stored
	f.order = append(f.order, id)
	return http.StatusOK, map[string]any{"item": aiCustomEvalCopy(stored)}
}

func (f *aiCustomEvaluationFake) update(id string, in map[string]any) (int, any) {
	if !aiEvalUUID.MatchString(id) {
		return aiEvalZodFail([]zodIssue{zodUUID()})
	}
	stored, ok := f.evaluations[id]
	if !ok {
		return apiError(http.StatusNotFound, aiCustomEvalNotFound)
	}
	// With an updateMask, only the listed fields change, and a listed field that the body leaves
	// out is cleared. Without one, each field in the body changes, but an empty list is no list.
	fields := map[string]any{}
	if mask, _ := in["updateMask"].(string); mask != "" {
		for _, path := range strings.Split(mask, ",") {
			fields[path] = in[path]
		}
	} else {
		for k, v := range in {
			if items, isList := v.([]any); isList && len(items) == 0 {
				continue
			}
			fields[k] = v
		}
	}
	if name, ok := fields["name"].(string); ok && f.nameTaken(name, id) {
		return aiCustomEvalConflict(name)
	}
	if v, ok := fields["name"]; ok {
		stored["name"] = aiCustomEvalOrZero(v, "")
	}
	if v, ok := fields["description"]; ok {
		stored["description"] = aiCustomEvalOrZero(v, "")
	}
	config := stored["config"].(map[string]any)
	for _, field := range aiCustomEvalConfigFields {
		if v, ok := fields[field.name]; ok {
			config[field.name] = aiCustomEvalOrZero(v, field.zero)
		}
	}
	answer := aiCustomEvalCopy(stored)
	answer["applicationIds"] = []any{}
	return http.StatusOK, map[string]any{"item": answer}
}

func (f *aiCustomEvaluationFake) link(link bool, id, applicationID string) (int, any) {
	stored, ok := f.evaluations[id]
	if !ok {
		return apiError(http.StatusNotFound, aiCustomEvalNotFound)
	}
	if !f.hasApplication(applicationID) {
		return apiError(http.StatusNotFound, "AI application not found")
	}
	links, _ := stored["applicationIds"].([]any)
	at := -1
	for i, linked := range links {
		if linked == applicationID {
			at = i
		}
	}
	switch {
	case link && at >= 0:
		return aiCustomEvalError(http.StatusConflict, "Custom evaluation is already linked to this application")
	case link:
		stored["applicationIds"] = append(append([]any{}, links...), applicationID)
	case at < 0:
		return apiError(http.StatusNotFound, "Custom evaluation is not linked to this application")
	default:
		stored["applicationIds"] = append(append([]any{}, links[:at]...), links[at+1:]...)
	}
	return http.StatusOK, map[string]any{"id": id, "applicationId": applicationID}
}

func aiCustomEvalOrZero(v, zero any) any {
	if v == nil {
		return zero
	}
	return v
}

// aiCustomEvalCopy copies a stored custom evaluation, its config, and its links.
func aiCustomEvalCopy(stored map[string]any) map[string]any {
	out := copyMap(stored)
	out["config"] = copyMap(stored["config"].(map[string]any))
	links, _ := stored["applicationIds"].([]any)
	out["applicationIds"] = append([]any{}, links...)
	return out
}
