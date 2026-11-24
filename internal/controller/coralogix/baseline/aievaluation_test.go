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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"unicode"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const aiEvaluationPath = "/ai/evaluations/v3"

// The AI application of the scenarios and of the fake's fidelity cases. The API refuses an
// evaluation of an application that the team does not have, and no API creates one.
const (
	aiEvalApplication = "cxo-baseline-app"
	aiEvalSubsystem   = "cxo-baseline-subsystem"
)

func init() {
	register(kind{
		name:     "aievaluation",
		crdFile:  "coralogix.com_aievaluations.yaml",
		resource: "aievaluations",
		newFake:  func() apiFake { return newAIEvaluationFake() },
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.AIEvaluationReconciler{Client: cs.AIEvaluations(), ClientSet: cs, Interval: reconcileInterval}
		},
		scenarios: aiEvaluationScenarios(),
		admission: aiEvaluationAdmission(),
		fidelity: fidelityConfig{
			captureField: "aiEvaluation.id",
			deletePath:   func(id string) string { return aiEvaluationPath + "/" + id },
			recordVars:   aiEvaluationRecordVars,
			fakeVars:     map[string]string{"application": aiEvalApplication, "subsystem": aiEvalSubsystem},
		},
	})
}

// aiEvaluationRecordVars names the AI application that the recorded cases use. Set
// CORALOGIX_AI_APPLICATION and CORALOGIX_AI_SUBSYSTEM, or the recorder takes the first application
// of the team that has no evaluation, so the cases never meet another evaluation.
func aiEvaluationRecordVars(t *testing.T, base, apiKey string) map[string]string {
	t.Helper()
	application, subsystem := os.Getenv("CORALOGIX_AI_APPLICATION"), os.Getenv("CORALOGIX_AI_SUBSYSTEM")
	if application != "" && subsystem != "" {
		return map[string]string{"application": application, "subsystem": subsystem}
	}
	status, apps, err := sendRecordRequest(base, apiKey, http.MethodGet, "/ai/applications/v3", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list AI applications: %d %v", status, err)
	}
	status, evals, err := sendRecordRequest(base, apiKey, http.MethodGet, aiEvaluationPath+"?page_size=1000", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list AI evaluations: %d %v", status, err)
	}
	used := map[string]bool{}
	for _, e := range listOf(evals, "aiEvaluations") {
		used[fmt.Sprint(e["application"])+"/"+fmt.Sprint(e["subsystem"])] = true
	}
	for _, app := range listOf(apps, "aiApplications") {
		application, _ := app["application"].(string)
		subsystem, _ := app["subsystem"].(string)
		if application != "" && subsystem != "" && !used[application+"/"+subsystem] {
			return map[string]string{"application": application, "subsystem": subsystem}
		}
	}
	t.Fatal("the team has no AI application without an evaluation: set CORALOGIX_AI_APPLICATION and CORALOGIX_AI_SUBSYSTEM")
	return nil
}

func listOf(resp any, field string) []map[string]any {
	m, _ := resp.(map[string]any)
	items, _ := m[field].([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if v, ok := item.(map[string]any); ok {
			out = append(out, v)
		}
	}
	return out
}

// aiEvaluationManifest returns an AIEvaluation CR named "evaluation". spec is YAML, indented by
// two spaces.
func aiEvaluationManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: AIEvaluation\nmetadata:\n  name: evaluation\nspec:\n" + spec
}

// aiEvalSpec returns the YAML of a spec with the application of the fake, a target, a threshold,
// and a config. config is YAML, indented by four spaces.
func aiEvalSpec(target, threshold, config string) string {
	return fmt.Sprintf("  application: %s\n  subsystem: %s\n  target: %s\n  threshold: %s\n  config:\n%s",
		aiEvalApplication, aiEvalSubsystem, target, threshold, config)
}

var aiEvalMinimalManifest = aiEvaluationManifest(aiEvalSpec("response", `"0.8"`, "    toxicity: {}\n"))

// aiEvalSpecMap is the spec of aiEvalMinimalManifest with the config and the fields given.
func aiEvalSpecMap(config map[string]any, fields ...any) map[string]any {
	spec := map[string]any{
		"application": aiEvalApplication,
		"subsystem":   aiEvalSubsystem,
		"target":      "response",
		"threshold":   "0.8",
		"config":      config,
	}
	for i := 0; i+1 < len(fields); i += 2 {
		spec[fields[i].(string)] = fields[i+1]
	}
	return spec
}

// aiEvalArms is every config arm of the released CRD, with a value for it.
var aiEvalArms = []struct {
	name  string
	value map[string]any
}{
	{"allowedTopics", map[string]any{"topics": []any{"billing", "shipping"}}},
	{"competition", map[string]any{"competitors": []any{"Example Corp"}}},
	{"hallucinationCompleteness", map[string]any{}},
	{"hallucinationContextAdherence", map[string]any{}},
	{"hallucinationContextRelevance", map[string]any{}},
	{"hallucinationCorrectness", map[string]any{}},
	{"hallucinationTaskAdherence", map[string]any{}},
	{"languageMismatch", map[string]any{}},
	{"pii", map[string]any{"categories": []any{"PHONE_NUMBER", "EMAIL_ADDRESS", "CREDIT_CARD", "IBAN_CODE", "US_SSN"}}},
	{"promptInjection", map[string]any{"additionalContext": "a support assistant"}},
	{"restrictedTopics", map[string]any{"topics": []any{"politics"}}},
	{"sexism", map[string]any{}},
	{"sqlAllowedTables", map[string]any{"tables": []any{"orders"}}},
	{"sqlHallucination", map[string]any{}},
	{"sqlReadOnly", map[string]any{}},
	{"sqlRestrictedTables", map[string]any{"tables": []any{"users"}}},
	{"toxicity", map[string]any{}},
}

func aiEvaluations(h *harness) *aiEvaluationFake {
	h.t.Helper()
	f, ok := h.API().(*aiEvaluationFake)
	if !ok {
		h.t.Fatal("expected *aiEvaluationFake")
	}
	return f
}

func aiEvaluationScenarios() map[string]func(h *harness) {
	scenarios := map[string]func(h *harness){
		"create-sample": func(h *harness) {
			h.CreateFromFile("create the sample", "config/samples/v1alpha1/aievaluations/pii-response.yaml")
			h.Reconcile("reconcile again with no change")
		},

		"create-minimal": func(h *harness) {
			h.Create("create with no isEnabled: the CRD default is true", aiEvalMinimalManifest)
			h.Reconcile("reconcile again with no change")
		},

		"create-disabled": func(h *harness) {
			h.Create("create with isEnabled false", aiEvaluationManifest(aiEvalSpec("prompt", `"0.5"`, "    sexism: {}\n")+"  isEnabled: false\n"))
		},

		"create-prompt-injection-without-context": func(h *harness) {
			h.Create("create a prompt injection with no additional context",
				aiEvaluationManifest(aiEvalSpec("prompt", `"0.5"`, "    promptInjection: {}\n")))
		},

		"update-every-arm": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			for _, arm := range aiEvalArms {
				h.SetSpec("switch the config to "+arm.name, aiEvalSpecMap(map[string]any{arm.name: arm.value}))
			}
		},

		"update-threshold-and-enabled": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			h.SetSpec("change the threshold", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0.25"))
			h.SetSpec("disable", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0.25", "isEnabled", false))
			h.SetSpec("remove isEnabled: the CRD default is true", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0.25"))
		},

		"update-list-items": func(h *harness) {
			h.Create("create", aiEvaluationManifest(aiEvalSpec("response", `"0.8"`, "    pii:\n      categories: [EMAIL_ADDRESS]\n")))
			h.SetSpec("add a category", aiEvalSpecMap(map[string]any{"pii": map[string]any{"categories": []any{"EMAIL_ADDRESS", "US_SSN"}}}))
			h.SetSpec("replace the categories", aiEvalSpecMap(map[string]any{"pii": map[string]any{"categories": []any{"CREDIT_CARD"}}}))
		},

		"update-prompt-injection-context": func(h *harness) {
			h.Create("create with additional context", aiEvaluationManifest(aiEvalSpec("prompt", `"0.5"`,
				"    promptInjection:\n      additionalContext: a support assistant\n")))
			h.SetSpec("remove the additional context", map[string]any{
				"application": aiEvalApplication, "subsystem": aiEvalSubsystem, "target": "prompt", "threshold": "0.5",
				"config": map[string]any{"promptInjection": map[string]any{}},
			})
		},

		"threshold-forms": func(h *harness) {
			h.Create("create with the threshold 1 as a number", aiEvaluationManifest(aiEvalSpec("response", "1", "    toxicity: {}\n")))
			h.SetSpec("threshold 0", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0"))
			h.SetSpec("threshold in milli units", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "750m"))
			h.SetSpec("threshold with many digits", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0.333333333"))
		},

		"create-threshold-out-of-range": func(h *harness) {
			h.Create("create with a threshold over 1: the operator rejects it", aiEvaluationManifest(aiEvalSpec("response", `"1.5"`, "    toxicity: {}\n")))
			h.Reconcile("reconcile again")
			h.SetSpec("fix the threshold", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}))
		},

		"create-threshold-negative": func(h *harness) {
			h.Create("create with a negative threshold: the operator rejects it", aiEvaluationManifest(aiEvalSpec("response", `"-0.1"`, "    toxicity: {}\n")))
		},

		"update-threshold-out-of-range": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			h.SetSpec("set a threshold over 1: the operator rejects it", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "2"))
			h.Delete("delete")
		},

		"immutable-fields": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			h.SetSpecRejected("change the application", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "application", "other-app"))
			h.SetSpecRejected("change the subsystem", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "subsystem", "other-subsystem"))
			h.SetSpecRejected("change the target", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "target", "prompt"))
		},

		"create-unknown-application": func(h *harness) {
			h.Create("create for an application that the team does not have", aiEvaluationManifest(
				"  application: unknown-app\n  subsystem: unknown-subsystem\n  target: response\n  threshold: \"0.8\"\n  config:\n    toxicity: {}\n"))
			h.Reconcile("reconcile again")
			h.Delete("delete")
		},

		"create-conflict": func(h *harness) {
			aiEvaluations(h).seed(map[string]any{
				"application": aiEvalApplication, "subsystem": aiEvalSubsystem, "target": "RESPONSE",
				"config": map[string]any{"toxicity": map[string]any{}}, "threshold": 0.5,
			})
			h.Create("create: another evaluation has the same application, subsystem, target, and type", aiEvalMinimalManifest)
			h.Reconcile("reconcile again")
			h.Delete("delete")
		},

		"create-rejected": func(h *harness) {
			aiEvaluations(h).answerNextError(http.MethodPost, http.StatusBadRequest, "the API rejects this request")
			h.Create("create: the API rejects the Create", aiEvalMinimalManifest)
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"create-empty-id": func(h *harness) {
			aiEvaluations(h).answerNext(http.MethodPost, map[string]any{"aiEvaluation": map[string]any{"id": ""}})
			h.Create("create: the answer has an empty id", aiEvalMinimalManifest)
			aiEvaluations(h).answerNext(http.MethodPost, map[string]any{})
			h.Reconcile("reconcile again: the answer has no evaluation")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"create-empty-id-after-remote-create": func(h *harness) {
			aiEvaluations(h).loseNextID()
			h.Create("create: the API stores the evaluation, and the answer has an empty id", aiEvalMinimalManifest)
			h.Reconcile("reconcile again: the stored evaluation takes the slot of the new one")
			h.Delete("delete: the operator has no id of the stored evaluation")
		},

		"create-malformed-response": func(h *harness) {
			aiEvaluations(h).answerNext(http.MethodPost, "not an evaluation")
			h.Create("create: the answer cannot be decoded", aiEvalMinimalManifest)
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"update-rejected": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).answerNextError(http.MethodPatch, http.StatusBadRequest, "the API rejects this request")
			h.SetSpec("change the threshold: the API rejects the Update", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0.4"))
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"update-malformed-response": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).answerNext(http.MethodPatch, "not an evaluation")
			h.Reconcile("update: the answer cannot be decoded")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"delete": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			h.Delete("delete")
		},

		"delete-when-gone": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).forget(h.StatusID())
			h.Delete("delete: the evaluation was deleted outside the operator")
		},

		"update-when-gone": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).forget(h.StatusID())
			h.SetSpec("change the threshold: the evaluation was deleted outside the operator", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0.4"))
			h.Reconcile("reconcile again")
		},

		"resync-when-gone": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).forget(h.StatusID())
			h.Reconcile("resync: the evaluation was deleted outside the operator")
			h.Reconcile("reconcile again")
		},

		"delete-after-update-when-gone": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).forget(h.StatusID())
			h.SetSpec("change the threshold: the evaluation was deleted outside the operator", aiEvalSpecMap(map[string]any{"toxicity": map[string]any{}}, "threshold", "0.4"))
			h.Delete("delete")
			h.Reconcile("reconcile again")
		},

		"delete-without-id": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			clearStatusID(h, h.key)
			h.Delete("delete a CR that has a finalizer and no status id")
			h.Reconcile("reconcile again")
		},

		"delete-rejected": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).answerNextError(http.MethodDelete, http.StatusBadRequest, "the API rejects this request")
			h.Delete("delete: the API rejects the Delete")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"delete-malformed-response": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			aiEvaluations(h).answerNext(http.MethodDelete, "not an answer")
			h.Delete("delete: the answer cannot be decoded")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"bad-status-id-update": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			setStatusID(h, h.key, "not-a-uuid")
			h.Reconcile("update with a status id that is not a UUID")
			h.Reconcile("reconcile again")
		},

		"bad-status-id-delete": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			setStatusID(h, h.key, "not-a-uuid")
			h.Delete("delete with a status id that is not a UUID")
			h.Reconcile("reconcile again")
		},

		"selector-mismatch": func(h *harness) {
			h.Create("create", aiEvalMinimalManifest)
			h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "ai"})})
			h.Reconcile("reconcile after the selector stops matching the CR")
		},
	}
	for _, arm := range aiEvalArms {
		value, err := json.Marshal(arm.value)
		if err != nil {
			panic(err)
		}
		target := "response"
		if arm.name == "promptInjection" {
			target = "prompt"
		}
		manifest := aiEvaluationManifest(aiEvalSpec(target, `"0.8"`, fmt.Sprintf("    %s: %s\n", arm.name, value)))
		scenarios["create-"+arm.name] = func(h *harness) {
			h.Create("create", manifest)
			h.Delete("delete")
		}
	}
	return scenarios
}

// aiEvaluationAdmission lists the CRs that TestAdmission creates. Each case has one reason that
// the CRD accepts or rejects it.
func aiEvaluationAdmission() map[string]string {
	base := func(application, subsystem, target, threshold, config string) string {
		spec := ""
		if application != "" {
			spec += "  application: " + application + "\n"
		}
		if subsystem != "" {
			spec += "  subsystem: " + subsystem + "\n"
		}
		if target != "" {
			spec += "  target: " + target + "\n"
		}
		if threshold != "" {
			spec += "  threshold: " + threshold + "\n"
		}
		if config != "" {
			spec += "  config:\n" + config
		}
		return aiEvaluationManifest(spec)
	}
	ok := func(config string) string { return base("app", "sub", "response", `"0.8"`, config) }
	withSpec := func(extra string) string { return ok("    toxicity: {}\n") + extra }
	cases := map[string]string{
		"minimal":                                ok("    toxicity: {}\n"),
		"no application":                         base("", "sub", "response", `"0.8"`, "    toxicity: {}\n"),
		"no subsystem":                           base("app", "", "response", `"0.8"`, "    toxicity: {}\n"),
		"no target":                              base("app", "sub", "", `"0.8"`, "    toxicity: {}\n"),
		"no threshold":                           base("app", "sub", "response", "", "    toxicity: {}\n"),
		"no config":                              base("app", "sub", "response", `"0.8"`, ""),
		"empty application":                      base(`""`, "sub", "response", `"0.8"`, "    toxicity: {}\n"),
		"empty subsystem":                        base("app", `""`, "response", `"0.8"`, "    toxicity: {}\n"),
		"application of spaces":                  base(`"   "`, "sub", "response", `"0.8"`, "    toxicity: {}\n"),
		"application of 256 characters":          base(strings.Repeat("a", 256), "sub", "response", `"0.8"`, "    toxicity: {}\n"),
		"application of 257 characters":          base(strings.Repeat("a", 257), "sub", "response", `"0.8"`, "    toxicity: {}\n"),
		"subsystem of 257 characters":            base("app", strings.Repeat("s", 257), "response", `"0.8"`, "    toxicity: {}\n"),
		"target prompt":                          base("app", "sub", "prompt", `"0.8"`, "    toxicity: {}\n"),
		"target conversation":                    base("app", "sub", "conversation", `"0.8"`, "    toxicity: {}\n"),
		"target in API spelling":                 base("app", "sub", "RESPONSE", `"0.8"`, "    toxicity: {}\n"),
		"threshold as a number":                  base("app", "sub", "response", "0.8", "    toxicity: {}\n"),
		"threshold 1.5":                          base("app", "sub", "response", `"1.5"`, "    toxicity: {}\n"),
		"threshold -1":                           base("app", "sub", "response", "-1", "    toxicity: {}\n"),
		"threshold that is not a quantity":       base("app", "sub", "response", `"high"`, "    toxicity: {}\n"),
		"isEnabled false":                        withSpec("  isEnabled: false\n"),
		"isEnabled as a string":                  withSpec("  isEnabled: \"yes\"\n"),
		"unknown spec field is not an error":     withSpec("  description: ignored\n"),
		"empty config":                           ok("    {}\n"),
		"two config arms":                        ok("    toxicity: {}\n    sexism: {}\n"),
		"empty arm with a field":                 ok("    toxicity:\n      level: high\n"),
		"empty arm as null":                      ok("    toxicity: null\n"),
		"arm the CRD does not have":              ok("    sqlLoad:\n      joinLimit: \"3\"\n"),
		"arm the CRD does not have and an arm":   ok("    sqlLoad: {}\n    toxicity: {}\n"),
		"custom evaluation arm":                  ok("    customEvaluation:\n      instructions: check {response}\n"),
		"pii without categories":                 ok("    pii: {}\n"),
		"pii with no categories":                 ok("    pii:\n      categories: []\n"),
		"pii with an unknown category":           ok("    pii:\n      categories: [PASSPORT]\n"),
		"pii with a category in lower case":      ok("    pii:\n      categories: [email_address]\n"),
		"pii with a repeated category":           ok("    pii:\n      categories: [US_SSN, US_SSN]\n"),
		"pii with every category":                ok("    pii:\n      categories: [PHONE_NUMBER, EMAIL_ADDRESS, CREDIT_CARD, IBAN_CODE, US_SSN]\n"),
		"allowed topics":                         ok("    allowedTopics:\n      topics: [billing]\n"),
		"allowed topics with no topics":          ok("    allowedTopics:\n      topics: []\n"),
		"allowed topics with an empty topic":     ok("    allowedTopics:\n      topics: [\"\"]\n"),
		"allowed topics with a repeated topic":   ok("    allowedTopics:\n      topics: [a, a]\n"),
		"allowed topics with a topic of 257":     ok("    allowedTopics:\n      topics: [" + strings.Repeat("t", 257) + "]\n"),
		"restricted topics with no topics":       ok("    restrictedTopics:\n      topics: []\n"),
		"competition with no competitors":        ok("    competition:\n      competitors: []\n"),
		"competition with a competitor":          ok("    competition:\n      competitors: [Example Corp]\n"),
		"sql allowed tables with no tables":      ok("    sqlAllowedTables:\n      tables: []\n"),
		"sql restricted tables with an empty":    ok("    sqlRestrictedTables:\n      tables: [\"\"]\n"),
		"prompt injection without context":       ok("    promptInjection: {}\n"),
		"prompt injection with empty context":    ok("    promptInjection:\n      additionalContext: \"\"\n"),
		"prompt injection with context of 65537": ok("    promptInjection:\n      additionalContext: " + strings.Repeat("c", 65537) + "\n"),
		"prompt injection with context of 65536": ok("    promptInjection:\n      additionalContext: " + strings.Repeat("c", 65536) + "\n"),
		"allowed topics with 1025 topics":        ok("    allowedTopics:\n      topics: [" + aiEvalItems(1025) + "]\n"),
		"allowed topics with 1024 topics":        ok("    allowedTopics:\n      topics: [" + aiEvalItems(1024) + "]\n"),
	}
	for _, arm := range aiEvalArms {
		value, err := json.Marshal(arm.value)
		if err != nil {
			panic(err)
		}
		cases["arm "+arm.name] = ok(fmt.Sprintf("    %s: %s\n", arm.name, value))
	}
	return cases
}

func aiEvalItems(n int) string {
	items := make([]string, n)
	for i := range items {
		items[i] = fmt.Sprintf("t%d", i)
	}
	return strings.Join(items, ", ")
}

// aiEvaluationFake is the AI evaluations API, written from the SDK's openapi.yaml.
//
//	POST   /ai/evaluations/v3        {application, subsystem, target, threshold?, isEnabled?, config} -> {aiEvaluation}
//	PATCH  /ai/evaluations/v3/{id}   {any of the fields, updateMask?}                                 -> {aiEvaluation}
//	GET    /ai/evaluations/v3/{id}                                                                    -> {aiEvaluation}
//	DELETE /ai/evaluations/v3/{id}                                                                    -> {id}
//
// The status codes, error texts, and stored values follow fidelity/recorded.json, which the real
// API answered:
//   - A PATCH with no updateMask updates each field in the body. A config in the body replaces
//     the stored config, also with another arm. With an updateMask, only the listed fields
//     change, and a listed threshold that the body leaves out is cleared.
//   - isEnabled is false when Create leaves it out. A promptInjection with no additionalContext
//     stores "".
//   - The API answers 404 for an evaluation that does not exist, and 400 for an id that is not a
//     UUID. An application that the team does not have is a 404 too.
//   - Only one evaluation can use an application, a subsystem, a target, and a type: a second
//     Create answers 409.
//   - The validation errors are the zod errors of the service, as text.
type aiEvaluationFake struct {
	mu           sync.Mutex
	evaluations  map[string]map[string]any
	applications map[string]bool
	seq, clock   int
	// next holds one forced answer per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]forcedAnswer
	// loseNextCreatedID makes the next Create store the evaluation and answer with an empty id.
	loseNextCreatedID bool
}

func newAIEvaluationFake() *aiEvaluationFake {
	return &aiEvaluationFake{
		evaluations: map[string]map[string]any{},
		applications: map[string]bool{
			aiEvalApplication + "/" + aiEvalSubsystem: true,
			"ai-center-demo/demo-runner":              true, // the sample
		},
		next: map[string]forcedAnswer{},
	}
}

// seed stores an evaluation, as if another client made it.
func (f *aiEvaluationFake) seed(body map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if status, resp := f.create(body); status != http.StatusOK {
		panic(fmt.Sprintf("seed: %d %v", status, resp))
	}
}

// forget deletes an evaluation without a request, as if it was deleted outside the operator.
func (f *aiEvaluationFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.evaluations, id)
}

// answerNext makes the next request with the method get a 200 answer with this body.
func (f *aiEvaluationFake) answerNext(method string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = forcedAnswer{status: http.StatusOK, body: body}
}

// loseNextID makes the next Create store the evaluation, as the real API does, and answer
// with an empty id, so the operator cannot know the id.
func (f *aiEvaluationFake) loseNextID() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loseNextCreatedID = true
}

// answerNextError makes the next request with the method get an error answer.
func (f *aiEvaluationFake) answerNextError(method string, status int, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := apiError(status, reason)
	f.next[method] = forcedAnswer{status: st, body: body}
}

func (f *aiEvaluationFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if forced, ok := f.next[method]; ok {
		delete(f.next, method)
		return forced.status, forced.body
	}
	rest, ok := strings.CutPrefix(path, aiEvaluationPath)
	if !ok {
		return routeNotFound()
	}
	id := strings.TrimPrefix(rest, "/")
	in, _ := body.(map[string]any)
	switch {
	case method == http.MethodPost && rest == "":
		status, resp := f.create(in)
		if f.loseNextCreatedID && status == http.StatusOK {
			f.loseNextCreatedID = false
			created := copyMap(resp.(map[string]any)["aiEvaluation"].(map[string]any))
			created["id"] = ""
			resp = map[string]any{"aiEvaluation": created}
		}
		return status, resp
	case method == http.MethodGet && id != "" && !strings.Contains(id, "/"):
		stored, status, resp := f.find(id)
		if stored == nil {
			return status, resp
		}
		return http.StatusOK, map[string]any{"aiEvaluation": copyMap(stored)}
	case method == http.MethodPatch && id != "" && !strings.Contains(id, "/"):
		return f.update(id, in)
	case method == http.MethodDelete && id != "" && !strings.Contains(id, "/"):
		stored, status, resp := f.find(id)
		if stored == nil {
			return status, resp
		}
		delete(f.evaluations, id)
		return http.StatusOK, map[string]any{"id": id}
	}
	return routeNotFound()
}

var aiEvalUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (f *aiEvaluationFake) find(id string) (map[string]any, int, any) {
	if !aiEvalUUID.MatchString(id) {
		status, resp := aiEvalZodFail([]zodIssue{zodUUID()})
		return nil, status, resp
	}
	stored, ok := f.evaluations[id]
	if !ok {
		status, resp := apiError(http.StatusNotFound, "Evaluation not found")
		return nil, status, resp
	}
	return stored, 0, nil
}

func (f *aiEvaluationFake) create(in map[string]any) (int, any) {
	if status, resp := aiEvalProtoCheck(in); status != 0 {
		return status, resp
	}
	if issues := aiEvalCheckCreate(in); len(issues) != 0 {
		return aiEvalZodFail(issues)
	}
	application, subsystem := in["application"].(string), in["subsystem"].(string)
	if !f.applications[application+"/"+subsystem] {
		return apiError(http.StatusNotFound, "AI application not found")
	}
	config := aiEvalNormalizeConfig(in["config"].(map[string]any))
	target := in["target"].(string)
	for _, other := range f.evaluations {
		if other["application"] == application && other["subsystem"] == subsystem &&
			other["target"] == target && aiEvalVariant(other["config"]) == aiEvalVariant(config) {
			return http.StatusConflict, map[string]any{"code": http.StatusConflict, "message": "Already Exists: An evaluation with these parameters already exists"}
		}
	}
	f.seq++
	f.clock++
	now := fmt.Sprintf("2026-01-01T00:00:%02d.000Z", f.clock)
	enabled, _ := in["isEnabled"].(bool)
	stored := map[string]any{
		"id":          newUUID(),
		"companyId":   "1",
		"application": application,
		"subsystem":   subsystem,
		"target":      target,
		"config":      config,
		"isEnabled":   enabled,
		"createdBy":   "",
		"createdAt":   now,
		"updatedAt":   now,
	}
	if threshold, ok := in["threshold"]; ok {
		stored["threshold"] = threshold
	}
	f.evaluations[stored["id"].(string)] = stored
	return http.StatusOK, map[string]any{"aiEvaluation": copyMap(stored)}
}

func (f *aiEvaluationFake) update(id string, in map[string]any) (int, any) {
	if status, resp := aiEvalProtoCheck(in); status != 0 {
		return status, resp
	}
	if issues := aiEvalCheckUpdate(id, in); len(issues) != 0 {
		return aiEvalZodFail(issues)
	}
	stored, status, resp := f.find(id)
	if stored == nil {
		return status, resp
	}
	// With an updateMask, only the listed fields change, and a listed threshold that the body
	// leaves out is cleared. Without one, each field in the body changes.
	if mask, _ := in["updateMask"].(string); mask != "" {
		masked := map[string]any{}
		for _, path := range strings.Split(mask, ",") {
			if v, ok := in[path]; ok {
				masked[path] = v
			} else if path == "threshold" {
				delete(stored, "threshold")
			}
		}
		in = masked
	}
	application, _ := in["application"].(string)
	subsystem, _ := in["subsystem"].(string)
	if application != "" || subsystem != "" {
		if application == "" {
			application = stored["application"].(string)
		}
		if subsystem == "" {
			subsystem = stored["subsystem"].(string)
		}
		if !f.applications[application+"/"+subsystem] {
			return apiError(http.StatusNotFound, "AI application not found")
		}
		stored["application"], stored["subsystem"] = application, subsystem
	}
	for _, k := range []string{"target", "isEnabled", "threshold"} {
		if v, ok := in[k]; ok {
			stored[k] = v
		}
	}
	if config, ok := in["config"].(map[string]any); ok {
		stored["config"] = aiEvalNormalizeConfig(config)
	}
	f.clock++
	stored["updatedAt"] = fmt.Sprintf("2026-01-01T00:00:%02d.000Z", f.clock)
	return http.StatusOK, map[string]any{"aiEvaluation": copyMap(stored)}
}

func copyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// aiEvalConfigLists names the list of each config arm that has one. The API requires at least
// one item, with no repeats.
var aiEvalConfigLists = map[string]string{
	"allowedTopics":       "topics",
	"restrictedTopics":    "topics",
	"competition":         "competitors",
	"pii":                 "categories",
	"sqlAllowedTables":    "tables",
	"sqlRestrictedTables": "tables",
}

// aiEvalMaskPaths are the update mask paths that the API accepts.
var aiEvalMaskPaths = map[string]bool{
	"application": true, "subsystem": true, "target": true, "config": true, "isEnabled": true, "threshold": true,
}

func aiEvalCheckCreate(in map[string]any) []zodIssue {
	var issues []zodIssue
	if _, ok := in["target"].(string); !ok {
		issues = append(issues, zodRequired("'PROMPT' | 'RESPONSE' | 'CONVERSATION'", "target"))
	}
	if config, ok := in["config"].(map[string]any); ok {
		issues = append(issues, aiEvalCheckConfig(config)...)
	} else {
		issues = append(issues, zodRequired("object", "config"))
	}
	for _, k := range []string{"application", "subsystem"} {
		if s, _ := in[k].(string); s == "" {
			issues = append(issues, zodTooSmall("string", "String must contain at least 1 character(s)", k))
		}
	}
	return append(issues, aiEvalCheckThreshold(in)...)
}

func aiEvalCheckUpdate(id string, in map[string]any) []zodIssue {
	var issues []zodIssue
	if !aiEvalUUID.MatchString(id) {
		issues = append(issues, zodUUID())
	}
	if config, ok := in["config"].(map[string]any); ok {
		issues = append(issues, aiEvalCheckConfig(config)...)
	}
	for _, k := range []string{"application", "subsystem"} {
		if s, ok := in[k].(string); ok && s == "" {
			issues = append(issues, zodTooSmall("string", "String must contain at least 1 character(s)", k))
		}
	}
	issues = append(issues, aiEvalCheckThreshold(in)...)
	if len(issues) != 0 {
		return issues
	}
	mask, _ := in["updateMask"].(string)
	if mask == "" {
		return nil
	}
	for _, path := range strings.Split(mask, ",") {
		if !aiEvalMaskPaths[path] {
			issues = append(issues, zodCustom(fmt.Sprintf("Unknown field path %q in update_mask.", snakeCase(path)), "updateMask"))
		}
	}
	if len(issues) != 0 {
		return issues
	}
	for _, path := range strings.Split(mask, ",") {
		if _, ok := in[path]; !ok && path != "threshold" {
			issues = append(issues, zodIssue{{"code", "custom"}, zodPath("updateMask"), {"message", fmt.Sprintf("Field %q is required and cannot be cleared via update_mask.", path)}})
		}
	}
	return issues
}

// snakeCase is the proto spelling of a field path, as the API names it in errors.
func snakeCase(path string) string {
	var b strings.Builder
	for _, r := range path {
		if unicode.IsUpper(r) {
			b.WriteByte('_')
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func aiEvalCheckThreshold(in map[string]any) []zodIssue {
	t, ok := in["threshold"].(float64)
	switch {
	case !ok:
		return nil
	case t < 0:
		return []zodIssue{zodTooSmall("number", "Number must be greater than or equal to 0", "threshold")}
	case t > 1:
		return []zodIssue{zodTooBig("threshold")}
	}
	return nil
}

func aiEvalCheckConfig(config map[string]any) []zodIssue {
	if len(config) == 0 {
		return []zodIssue{zodRequired("object", "config", "evaluation")}
	}
	variant := aiEvalVariant(config)
	list, ok := aiEvalConfigLists[variant]
	if !ok {
		return nil
	}
	value, _ := config[variant].(map[string]any)
	items, _ := value[list].([]any)
	path := []any{"config", "evaluation", "value", list}
	if len(items) == 0 {
		return []zodIssue{zodTooSmall("array", "Array must contain at least 1 element(s)", path...)}
	}
	seen := map[string]bool{}
	for _, item := range items {
		key := fmt.Sprint(item)
		if seen[key] {
			return []zodIssue{zodCustom("Items in array must be unique, no duplicates allowed.", path...)}
		}
		seen[key] = true
	}
	return nil
}

// aiEvalProtoCheck is the answer of the gateway to a config with two arms: it cannot decode the
// request into the protobuf oneOf.
func aiEvalProtoCheck(in map[string]any) (int, any) {
	config, ok := in["config"].(map[string]any)
	if !ok || len(config) < 2 {
		return 0, nil
	}
	keys := strings.Split(aiEvalVariant(config), ",")
	return apiError(http.StatusBadRequest, fmt.Sprintf("proto: (line 1:1): error parsing %q, oneof com.coralogixapis.ai.v3.EvaluationConfig.evaluation is already set", keys[1]))
}

// aiEvalVariant is the arm of a config, or the arms joined with "," when there are several.
func aiEvalVariant(config any) string {
	m, _ := config.(map[string]any)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func aiEvalNormalizeConfig(config map[string]any) map[string]any {
	variant := aiEvalVariant(config)
	value, _ := config[variant].(map[string]any)
	stored := copyMap(value)
	if variant == "promptInjection" {
		if _, ok := stored["additionalContext"]; !ok {
			stored["additionalContext"] = ""
		}
	}
	return map[string]any{variant: stored}
}

// zodIssue is one issue of a zod validation error, as ordered key-value pairs: the service prints
// the keys in the order of each issue kind.
type zodIssue [][2]any

func zodPath(path ...any) [2]any { return [2]any{"path", path} }

func zodRequired(expected string, path ...any) zodIssue {
	if strings.HasPrefix(expected, "'") {
		return zodIssue{{"expected", expected}, {"received", "undefined"}, {"code", "invalid_type"}, zodPath(path...), {"message", "Required"}}
	}
	return zodIssue{{"code", "invalid_type"}, {"expected", expected}, {"received", "undefined"}, zodPath(path...), {"message", "Required"}}
}

func zodTooSmall(kind, message string, path ...any) zodIssue {
	minimum := 1
	if kind == "number" {
		minimum = 0
	}
	return zodIssue{{"code", "too_small"}, {"minimum", minimum}, {"type", kind}, {"inclusive", true}, {"exact", false}, {"message", message}, zodPath(path...)}
}

func zodTooBig(path ...any) zodIssue {
	return zodIssue{{"code", "too_big"}, {"maximum", 1}, {"type", "number"}, {"inclusive", true}, {"exact", false}, {"message", "Number must be less than or equal to 1"}, zodPath(path...)}
}

func zodCustom(message string, path ...any) zodIssue {
	return zodIssue{{"code", "custom"}, {"message", message}, zodPath(path...)}
}

func zodUUID() zodIssue {
	return zodIssue{{"validation", "uuid"}, {"code", "invalid_string"}, {"message", "Invalid uuid"}, zodPath("id")}
}

// zodMessage prints the issues as the service does: JSON with two-space indents.
func zodMessage(issues []zodIssue) string {
	var b strings.Builder
	var write func(v any, indent string)
	write = func(v any, indent string) {
		switch x := v.(type) {
		case zodIssue:
			b.WriteString("{")
			for i, kv := range x {
				if i > 0 {
					b.WriteString(",")
				}
				b.WriteString("\n" + indent + "  " + mustJSON(kv[0]) + ": ")
				write(kv[1], indent+"  ")
			}
			b.WriteString("\n" + indent + "}")
		case []any:
			b.WriteString("[")
			for i, e := range x {
				if i > 0 {
					b.WriteString(",")
				}
				b.WriteString("\n" + indent + "  ")
				write(e, indent+"  ")
			}
			b.WriteString("\n" + indent + "]")
		default:
			b.WriteString(mustJSON(x))
		}
	}
	all := make([]any, len(issues))
	for i, issue := range issues {
		all[i] = issue
	}
	write(all, "")
	return b.String()
}

// mustJSON is the JSON text of a value of a zod issue: a string, a number, or a bool.
func mustJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func aiEvalZodFail(issues []zodIssue) (int, any) {
	return apiError(http.StatusBadRequest, zodMessage(issues))
}
