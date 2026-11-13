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
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/yaml"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// The TCO policies kinds have no remote id. Each CR overwrites every policy of its signal in the
// team, so each reconcile sends one PUT with the whole list, and a delete sends the same PUT
// with no policies. A policy may name its archive retention, and then the reconcile first reads
// the retentions of the team.
const retentionsPath = "/dataengine/retention-tags/v1"

// The retentions that every fake team has. The ids are fixed, so maskIDs keeps them. The
// fidelity cases do not read retentions: the names and ids of a real team are its own.
const (
	defaultRetentionID  = fixedIDPrefix + "0000000000a1"
	longTermRetentionID = fixedIDPrefix + "0000000000a2"
	// unknownRetentionID names no retention of the team.
	unknownRetentionID = "12345678-1234-4234-8234-123456789abc"
)

// tcoSignal is what differs between the log, span, and RUM policy APIs.
type tcoSignal struct {
	// path is the overwrite path.
	path string
	// rulesKey is the request and response field that holds the rules of the signal.
	rulesKey string
	// maxPolicies is the most policies one overwrite takes.
	maxPolicies int
	// defaultDataset is the dataset of the one target that a policy without targets reads back
	// with, or "" when it reads back with none.
	defaultDataset string
	// alwaysDefaultTarget is true when the policy reads back with the default target even when the
	// request has targets.
	alwaysDefaultTarget bool
	// rejectsTargets is true when the API rejects a policy with targets.
	rejectsTargets bool
	// severities is true when the rules hold severities.
	severities bool
}

var (
	tcoLogs  = tcoSignal{path: "/dataplans/log-policies/v1", rulesKey: "logRules", maxPolicies: 200, defaultDataset: "logs", severities: true}
	tcoSpans = tcoSignal{path: "/dataplans/span-policies/v1", rulesKey: "spanRules", maxPolicies: 200, defaultDataset: "spans", alwaysDefaultTarget: true}
	tcoRum   = tcoSignal{path: "/dataplans/rum-policies/v1", rulesKey: "rumRules", maxPolicies: 10000, rejectsTargets: true, severities: true}
)

// tcoFake is the atomic overwrite API of one signal and the retentions API, written from the
// SDK's openapi.yaml.
//
//	PUT /dataplans/{log,span,rum}-policies/v1  {policies: [...]}  -> {createResponses: [{policy}]}
//	GET /dataengine/retention-tags/v1                             -> {retentions: [...]}
//
// Nothing here is recorded yet: fidelity/recorded.json does not exist, because the cases replace
// every policy of the team. Each rule is the documented behavior of openapi.yaml, or the
// simplest answer that the scenarios need.
type tcoFake struct {
	mu     sync.Mutex
	signal tcoSignal
	// policies are the stored policies, in order.
	policies []map[string]any
	// lastRequest is the policies list of the last overwrite that succeeded, as JSON.
	lastRequest string
	retentions  []map[string]any
	// next holds one answer per "METHOD path". The next request to it gets the answer, and the
	// fake does not run the request.
	next map[string]tcoAnswer
}

type tcoAnswer struct {
	status int
	body   any
}

func newTCOFake(signal tcoSignal) func() apiFake {
	return func() apiFake {
		return &tcoFake{
			signal: signal,
			retentions: []map[string]any{
				{"id": defaultRetentionID, "name": "Default", "order": 1, "editable": false},
				{"id": longTermRetentionID, "name": "Long_term", "order": 2, "editable": true},
			},
			next: map[string]tcoAnswer{},
		}
	}
}

// answerNext makes the next request with the method and path get this answer.
func (f *tcoFake) answerNext(method, path string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method+" "+path] = tcoAnswer{status: status, body: body}
}

func tcoAPI(h *harness) *tcoFake {
	h.t.Helper()
	f, ok := h.API().(*tcoFake)
	if !ok {
		h.t.Fatal("expected *tcoFake")
	}
	return f
}

func (f *tcoFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if answer, ok := f.next[method+" "+path]; ok {
		delete(f.next, method+" "+path)
		return answer.status, answer.body
	}
	switch {
	case method == http.MethodGet && path == retentionsPath:
		list := make([]any, len(f.retentions))
		for i, r := range f.retentions {
			list[i] = r
		}
		return http.StatusOK, map[string]any{"retentions": list}
	case method == http.MethodPut && path == f.signal.path:
		in, _ := body.(map[string]any)
		return f.overwrite(in)
	}
	return routeNotFound()
}

// tcoTime is the time of every stored policy.
const tcoTime = "2026-01-01T00:00:00Z"

func (f *tcoFake) overwrite(in map[string]any) (int, any) {
	// The operator sends {"policies": null} for an empty list. It is read as an empty list.
	requested, _ := in["policies"].([]any)
	if len(requested) > f.signal.maxPolicies {
		return apiError(http.StatusBadRequest, fmt.Sprintf("policies must have at most %d items", f.signal.maxPolicies))
	}
	names := map[string]bool{}
	for i, raw := range requested {
		p, _ := raw.(map[string]any)
		policy, _ := p["policy"].(map[string]any)
		name, _ := policy["name"].(string)
		// openapi.yaml: "the service rejects a repeated name with 409".
		if names[name] {
			return apiError(http.StatusConflict, fmt.Sprintf("policy name %q is used more than once", name))
		}
		names[name] = true
		if status, resp := f.validate(i, p); status != 0 {
			return status, resp
		}
	}

	raw := compact(requested)
	// openapi.yaml: "one that sends the stored policies again keeps their ids".
	if raw != f.lastRequest {
		f.policies = make([]map[string]any, len(requested))
		for i, r := range requested {
			p, _ := r.(map[string]any)
			f.policies[i] = f.stored(i, p)
		}
		f.lastRequest = raw
	}
	responses := make([]any, len(f.policies))
	for i, p := range f.policies {
		responses[i] = map[string]any{"policy": p}
	}
	return http.StatusOK, map[string]any{"createResponses": responses}
}

// validate checks one policy of an overwrite. A non-zero status is the error answer.
func (f *tcoFake) validate(i int, p map[string]any) (int, any) {
	policy, _ := p["policy"].(map[string]any)
	rules, _ := p[f.signal.rulesKey].(map[string]any)
	if name, _ := policy["name"].(string); name == "" {
		return apiError(http.StatusBadRequest, fmt.Sprintf("policies[%d].policy.name must not be empty", i))
	}
	// openapi.yaml: the expression "starts with a version tag ... the service rejects an
	// expression without one".
	if dpxl, ok := rules["dpxlExpression"].(string); ok && !strings.HasPrefix(dpxl, "<v") {
		return apiError(http.StatusBadRequest, fmt.Sprintf("policies[%d]: the DPXL expression has no version tag", i))
	}
	// openapi.yaml: "A request without dpxlExpression needs at least one" severity.
	if f.signal.severities {
		severities, _ := rules["severities"].([]any)
		if _, dpxl := rules["dpxlExpression"]; !dpxl && len(severities) == 0 {
			return apiError(http.StatusBadRequest, fmt.Sprintf("policies[%d]: set severities or a DPXL expression", i))
		}
	}
	targets, _ := policy["targets"].([]any)
	// openapi.yaml: "RUM policies reject targets".
	if f.signal.rejectsTargets && len(targets) > 0 {
		return apiError(http.StatusBadRequest, fmt.Sprintf("policies[%d]: a RUM policy has no targets", i))
	}
	// openapi.yaml: "The service rejects a policy without a priority, an override, or targets
	// that each have a priority."
	if !tcoHasPriority(policy["priority"]) && policy["priorityOverride"] == nil {
		every := len(targets) > 0
		for _, raw := range targets {
			target, _ := raw.(map[string]any)
			every = every && tcoHasPriority(target["priority"])
		}
		if !every {
			return apiError(http.StatusBadRequest, fmt.Sprintf("policies[%d]: the policy has no priority", i))
		}
	}
	return 0, nil
}

func tcoHasPriority(v any) bool {
	p, _ := v.(string)
	return p != "" && p != "PRIORITY_TYPE_UNSPECIFIED"
}

// stored is the policy that the API keeps for one policy of an overwrite.
func (f *tcoFake) stored(i int, p map[string]any) map[string]any {
	policy, _ := p["policy"].(map[string]any)
	disabled, _ := policy["disabled"].(bool)
	out := map[string]any{
		"id":        newUUID(),
		"companyId": 1,
		"name":      policy["name"],
		// openapi.yaml: "A create or atomic-overwrite request sets it as the opposite of its
		// disabled value".
		"enabled": !disabled,
		"deleted": false,
		// openapi.yaml: "An overwrite sets it to the position of the policy in the request,
		// starting at 1".
		"order":     i + 1,
		"priority":  policy["priority"],
		"createdAt": tcoTime,
		"updatedAt": tcoTime,
	}
	// openapi.yaml: "With priorityOverride and no priority, the policy gets PRIORITY_TYPE_BLOCK".
	if !tcoHasPriority(policy["priority"]) && policy["priorityOverride"] != nil {
		out["priority"] = "PRIORITY_TYPE_BLOCK"
	}
	if out["priority"] == nil {
		out["priority"] = "PRIORITY_TYPE_UNSPECIFIED"
	}
	// openapi.yaml: "The atomic overwrite does not default an omitted description".
	for _, key := range []string{"description", "applicationRule", "subsystemRule", "archiveRetention", "priorityOverride"} {
		if v, ok := policy[key]; ok {
			out[key] = v
		}
	}

	rules := map[string]any{}
	if r, ok := p[f.signal.rulesKey].(map[string]any); ok {
		for k, v := range r {
			rules[k] = v
		}
	}
	if f.signal.severities {
		// openapi.yaml: "A response with dpxlExpression has an empty list."
		if _, ok := rules["severities"]; !ok {
			rules["severities"] = []any{}
		}
	} else if _, ok := rules["tagRules"]; !ok {
		rules["tagRules"] = []any{}
	}
	out[f.signal.rulesKey] = rules

	// openapi.yaml: a log policy without targets reads back with the logs target of the default
	// dataspace, a span policy always reads back with one spans target, and a RUM policy with none.
	targets, _ := policy["targets"].([]any)
	switch {
	case len(targets) > 0 && !f.signal.alwaysDefaultTarget:
		out["targets"] = targets
	case f.signal.defaultDataset != "":
		target := map[string]any{"dataset": f.signal.defaultDataset, "dataspace": "default", "priority": out["priority"]}
		if o, ok := policy["priorityOverride"]; ok {
			target["priorityOverride"] = o
		}
		out["targets"] = []any{target}
	}
	return out
}

// tcoSpec parses a spec written as YAML, for SetSpec.
func tcoSpec(h *harness, spec string) map[string]any {
	h.t.Helper()
	out := map[string]any{}
	if err := yaml.Unmarshal([]byte(spec), &out); err != nil {
		h.t.Fatalf("parse spec: %v", err)
	}
	return out
}

// tcoManifest is a CR of the kind with the spec, written as YAML at the indent of a spec field.
func tcoManifest(kind, spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: " + kind + "\nmetadata:\n  name: policies\nspec:\n" + spec
}

// tcoCommonScenarios are the scenarios that every TCO policies kind has. minimal is a spec that
// the API accepts, updated is another one, and rejected is one that the API rejects with 400.
// signal is the overwrite path of the kind.
func tcoCommonScenarios(kind string, signal tcoSignal, minimal, updated, rejected string) map[string]func(h *harness) {
	return map[string]func(h *harness){
		"create-minimal": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
		},

		"create-empty-list": func(h *harness) {
			h.Create("create with an empty list: it deletes every policy", tcoManifest(kind, "  policies: []\n"))
		},

		"resync": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			h.Reconcile("reconcile again with no change: the overwrite runs again")
			h.Reconcile("reconcile a third time")
		},

		"update": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			h.SetSpec("change the policies", tcoSpec(h, updated))
		},

		"create-rejected": func(h *harness) {
			h.Create("create with a DPXL expression that has no version tag: the API rejects it", tcoManifest(kind, rejected))
			h.Reconcile("reconcile again")
			h.Delete("delete: the CR has no finalizer")
		},

		"create-rejected-duplicate-names": func(h *harness) {
			h.Create("create with two policies of the same name: the API answers 409", tcoManifest(kind, tcoDuplicate(minimal)))
		},

		"retentions-unavailable": func(h *harness) {
			tcoAPI(h).answerNext(http.MethodGet, retentionsPath, http.StatusInternalServerError,
				map[string]any{"code": 500, "message": "Internal Server Error: retentions unavailable"})
			h.Create("create with a retention name: the retentions read fails", tcoManifest(kind, tcoWithRetentionName(minimal, "Default")))
			h.Reconcile("reconcile again: the read works")
		},

		"archive-retention-by-id": func(h *harness) {
			h.Create("create with a retention id: no retentions read", tcoManifest(kind, tcoWithRetentionID(minimal, longTermRetentionID)))
		},

		"archive-retention-by-name": func(h *harness) {
			h.Create("create with a retention name: one retentions read, then the overwrite", tcoManifest(kind, tcoWithRetentionName(minimal, "Long_term")))
			h.Reconcile("reconcile again: the retentions are read again")
		},

		"archive-retention-name-not-found": func(h *harness) {
			h.Create("create with a retention name that the team does not have: no overwrite", tcoManifest(kind, tcoWithRetentionName(minimal, "Missing")))
			h.Delete("delete: the CR has no finalizer")
		},

		"archive-retention-unknown-id": func(h *harness) {
			h.Create("create with a retention id that the team does not have: the operator sends it", tcoManifest(kind, tcoWithRetentionID(minimal, unknownRetentionID)))
		},

		"delete": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			h.Delete("delete: the overwrite sends no policies")
		},

		// A 404 that names a resource is "not found": the delete succeeds.
		"delete-when-gone": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusNotFound,
				map[string]any{"code": 404, "message": "Not Found: no policies"})
			h.Delete("delete: the API answers 404")
		},

		// A 404 for a missing route is not "not found": the delete fails and keeps the finalizer.
		"delete-route-not-found": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusNotFound,
				map[string]any{"code": 404, "message": "Not Found: Not Found"})
			h.Delete("delete: the API answers 404 for the route")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"delete-rejected": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusInternalServerError,
				map[string]any{"code": 500, "message": "Internal Server Error: try again"})
			h.Delete("delete: the API answers 500")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		// The kind has no status id, so the shared reconciler's "remove status.id" step changes
		// nothing, and the next reconcile overwrites again.
		"update-when-gone": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusNotFound,
				map[string]any{"code": 404, "message": "Not Found: no policies"})
			h.SetSpec("change the policies: the API answers 404", tcoSpec(h, updated))
			h.Reconcile("reconcile again")
		},

		"selector-mismatch": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "tco"})})
			h.Reconcile("reconcile after the selector stops matching the CR: the overwrite sends no policies")
			h.Reconcile("reconcile again")
		},

		// A body that the SDK cannot decode is a DeserializationError, and the shared reconciler
		// does not write the same condition twice.
		"malformed-response": func(h *harness) {
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusOK, "not an overwrite answer")
			h.Create("create: the answer cannot be decoded", tcoManifest(kind, minimal))
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusOK, "not an overwrite answer")
			h.Reconcile("reconcile again: the answer cannot be decoded")
			h.Reconcile("reconcile again: the API answers as it should")
		},

		"malformed-policy-response": func(h *harness) {
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusOK,
				map[string]any{"createResponses": []any{map[string]any{"policy": map[string]any{"name": "no id"}}}})
			h.Create("create: the answer has a policy without its required fields", tcoManifest(kind, minimal))
		},

		"delete-malformed-response": func(h *harness) {
			h.Create("create", tcoManifest(kind, minimal))
			tcoAPI(h).answerNext(http.MethodPut, signal.path, http.StatusOK, "not an overwrite answer")
			h.Delete("delete: the answer cannot be decoded")
			h.Reconcile("reconcile again: the API answers as it should")
		},
	}
}

// tcoDuplicate repeats the first policy of a spec. The spec must hold one policy.
func tcoDuplicate(spec string) string {
	_, policy, _ := strings.Cut(spec, "  policies:\n")
	return spec + policy
}

// tcoWithRetentionName adds an archive retention by name to the first policy of a spec.
func tcoWithRetentionName(spec, name string) string {
	return tcoAddToFirstPolicy(spec, "      archiveRetention:\n        backendRef:\n          name: "+name+"\n")
}

// tcoWithRetentionID adds an archive retention by id to the first policy of a spec.
func tcoWithRetentionID(spec, id string) string {
	return tcoAddToFirstPolicy(spec, "      archiveRetention:\n        backendRef:\n          id: "+id+"\n")
}

// tcoAddToFirstPolicy adds fields after the name of the first policy of a spec.
func tcoAddToFirstPolicy(spec, fields string) string {
	before, after, ok := strings.Cut(spec, "    - name: ")
	if !ok {
		panic("the spec has no policy")
	}
	nameLine, rest, _ := strings.Cut(after, "\n")
	return before + "    - name: " + nameLine + "\n" + fields + rest
}
