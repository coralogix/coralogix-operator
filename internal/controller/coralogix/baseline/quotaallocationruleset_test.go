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
	"fmt"
	"net/http"
	"net/url"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const quotaRulesPath = "/dataplan/quota-rules/v1"

func init() {
	register(kind{
		name:     "quotaallocationruleset",
		crdFile:  "coralogix.com_quotaallocationrulesets.yaml",
		resource: "quotaallocationrulesets",
		newFake:  func() apiFake { return &quotaRulesFake{} },
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.QuotaAllocationRuleSetReconciler{
				QuotaAllocationRulesClient: cs.Quotas(),
				Interval:                   reconcileInterval,
			}
		},
		scenarios: quotaRulesScenariosWithCleanup(),
		admission: quotaRulesAdmission(),
		// This API replaces the account's rule set. Live fidelity recording needs an isolated
		// account and a recorder that restores its initial rule set on every exit path.
		fidelity: fidelityConfig{overwritesTeam: true},
	})
}

const quotaRulesMinimal = `apiVersion: coralogix.com/v1alpha1
kind: QuotaAllocationRuleSet
metadata:
  name: quota-rules
spec:
  rules:
    - entityType: logs
      allocation: 50
      enabled: true
      canOverflow: false
`

const quotaRulesFull = `apiVersion: coralogix.com/v1alpha1
kind: QuotaAllocationRuleSet
metadata:
  name: quota-rules
spec:
  rules:
    - entityType: logs
      allocation: "12.5"
      allocationType: percentage
      enabled: true
      canOverflow: true
    - entityType: spans
      allocation: "1.25"
      allocationType: lockedUnits
      enabled: false
      canOverflow: false
    - entityType: metrics
      allocation: 0
      allocationType: unspecified
      enabled: true
      canOverflow: false
`

func quotaRulesSpec(h *harness, manifest string) map[string]any {
	h.t.Helper()
	var obj map[string]any
	if err := yaml.Unmarshal([]byte(manifest), &obj); err != nil {
		h.t.Fatal(err)
	}
	return obj["spec"].(map[string]any)
}

func quotaRulesAPI(h *harness) *quotaRulesFake {
	h.t.Helper()
	f, ok := h.API().(*quotaRulesFake)
	if !ok {
		h.t.Fatal("expected *quotaRulesFake")
	}
	return f
}

var quotaRulesScenarios = map[string]func(h *harness){
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the released sample", "config/samples/v1alpha1/quotaallocationruleset/example-quota-allocation-rule-set.yaml")
	},
	"create-minimal": func(h *harness) {
		h.Create("create with the default allocation type", quotaRulesMinimal)
	},
	"create-full": func(h *harness) {
		h.Create("create with both allocation types, an alias, and a fractional quantity", quotaRulesFull)
	},
	"create-empty": func(h *harness) {
		h.Create("create with an empty rule list", "apiVersion: coralogix.com/v1alpha1\nkind: QuotaAllocationRuleSet\nmetadata:\n  name: quota-rules\nspec:\n  rules: []\n")
	},
	"resync": func(h *harness) {
		h.Create("create", quotaRulesMinimal)
		h.Reconcile("resync the same rule set")
	},
	"update": func(h *harness) {
		h.Create("create", quotaRulesMinimal)
		h.SetSpec("replace the full rule list", quotaRulesSpec(h, quotaRulesFull))
	},
	"clear-rules": func(h *harness) {
		h.Create("create", quotaRulesFull)
		h.SetSpec("clear customer-managed rules with an empty list", map[string]any{"rules": []any{}})
		h.RecordRemote("the remote has an empty rule list", quotaRulesAPI(h).snapshot())
	},
	"managed-rules": func(h *harness) {
		quotaRulesAPI(h).seed([]any{
			quotaRule("metrics", 25, true),
			quotaRule("spans", 10, false),
		})
		h.Create("replace user rules and preserve the managed rule", quotaRulesMinimal)
		h.RecordRemote("the remote has the new user rule and the managed rule", quotaRulesAPI(h).snapshot())
		h.Delete("delete only customer-managed rules")
		h.RecordRemote("the managed rule remains", quotaRulesAPI(h).snapshot())
	},
	"managed-rule-collision": func(h *harness) {
		quotaRulesAPI(h).seed([]any{quotaRule("logs", 25, true)})
		h.Create("reject a user rule for a managed entity type", quotaRulesMinimal)
		h.RecordRemote("leave the managed rule unchanged", quotaRulesAPI(h).snapshot())
	},
	"invalid-allocation": func(h *harness) {
		h.Create("reject a percentage over 100 before the API call", "apiVersion: coralogix.com/v1alpha1\nkind: QuotaAllocationRuleSet\nmetadata:\n  name: quota-rules\nspec:\n  rules:\n    - {entityType: logs, allocation: 101, enabled: true, canOverflow: false}\n")
	},
	"second-selected-rule-set": func(h *harness) {
		h.Create("create the selected rule set", quotaRulesMinimal)
		other := &coralogixv1alpha1.QuotaAllocationRuleSet{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: h.namespace},
			Spec:       coralogixv1alpha1.QuotaAllocationRuleSetSpec{Rules: []coralogixv1alpha1.QuotaAllocationRule{}},
		}
		if err := h.client.Create(context.Background(), other); err != nil {
			h.t.Fatal(err)
		}
		h.Reconcile("reject a second selected resource without changing the remote")
		h.RecordRemote("the first rule set stays remote", quotaRulesAPI(h).snapshot())
	},
	"selector-mismatch": func(h *harness) {
		h.Create("create", quotaRulesMinimal)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "selected"})})
		h.Reconcile("reconcile after the selector stops matching")
		h.RecordRemote("the remote after selector cleanup", quotaRulesAPI(h).snapshot())
	},
	"create-get-failed-then-retried": func(h *harness) {
		quotaRulesAPI(h).answerNext(http.MethodGet)
		h.Create("fail to read before replacing", quotaRulesMinimal)
		h.Reconcile("retry the full replacement")
	},
	"create-replace-failed-then-retried": func(h *harness) {
		quotaRulesAPI(h).answerNext(http.MethodPut)
		h.Create("fail to replace", quotaRulesMinimal)
		h.Reconcile("retry the full replacement")
	},
	"update-failed-then-retried": func(h *harness) {
		h.Create("create", quotaRulesMinimal)
		quotaRulesAPI(h).answerNext(http.MethodPut)
		h.SetSpec("fail to replace the updated rule set", quotaRulesSpec(h, quotaRulesFull))
		h.Reconcile("retry the full replacement")
	},
	"delete": func(h *harness) {
		h.Create("create", quotaRulesMinimal)
		h.Delete("delete the rule set")
		h.RecordRemote("the remote has no customer-managed rules", quotaRulesAPI(h).snapshot())
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", quotaRulesMinimal)
		quotaRulesAPI(h).forget()
		h.Delete("delete after the remote rules were cleared outside the operator")
	},
	"delete-failed-then-retried": func(h *harness) {
		h.Create("create", quotaRulesMinimal)
		quotaRulesAPI(h).answerNext(http.MethodDelete)
		h.Delete("fail to delete the remote")
		h.Reconcile("retry deletion")
	},
}

// The envtest harness keeps namespaces between scenarios. Remove this singleton kind after
// each case, or the next case will record a cross-namespace conflict instead of its API calls.
func quotaRulesScenariosWithCleanup() map[string]func(h *harness) {
	out := make(map[string]func(h *harness), len(quotaRulesScenarios))
	for name, scenario := range quotaRulesScenarios {
		out[name] = func(h *harness) {
			defer cleanupQuotaRules(h)
			scenario(h)
		}
	}
	return out
}

func cleanupQuotaRules(h *harness) {
	h.t.Helper()
	ctx := context.Background()
	list := &coralogixv1alpha1.QuotaAllocationRuleSetList{}
	if err := h.client.List(ctx, list, client.InNamespace(h.namespace)); err != nil {
		h.t.Fatalf("list quota rule sets for cleanup: %v", err)
	}
	for i := range list.Items {
		obj := &list.Items[i]
		if len(obj.Finalizers) != 0 {
			obj.Finalizers = nil
			if err := h.client.Update(ctx, obj); err != nil {
				h.t.Fatalf("clear quota rule set finalizers: %v", err)
			}
		}
		if err := h.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			h.t.Fatalf("delete quota rule set after scenario: %v", err)
		}
	}
}

func quotaRulesAdmission() map[string]string {
	withRules := func(rules string) string {
		return "apiVersion: coralogix.com/v1alpha1\nkind: QuotaAllocationRuleSet\nmetadata:\n  name: quota-rules\nspec:\n  rules:\n" + rules
	}
	return map[string]string{
		"minimal":                 quotaRulesMinimal,
		"full":                    quotaRulesFull,
		"empty rules":             withRules("    []\n"),
		"missing rules":           "apiVersion: coralogix.com/v1alpha1\nkind: QuotaAllocationRuleSet\nmetadata:\n  name: quota-rules\nspec: {}\n",
		"duplicate entity":        withRules("    - {entityType: logs, allocation: 10, enabled: true, canOverflow: false}\n    - {entityType: logs, allocation: 20, enabled: true, canOverflow: false}\n"),
		"empty entity":            withRules("    - {entityType: \"\", allocation: 10, enabled: true, canOverflow: false}\n"),
		"missing allocation":      withRules("    - {entityType: logs, enabled: true, canOverflow: false}\n"),
		"negative allocation":     withRules("    - {entityType: logs, allocation: -1, enabled: true, canOverflow: false}\n"),
		"over 100 percent":        withRules("    - {entityType: logs, allocation: 101, allocationType: percentage, enabled: true, canOverflow: false}\n"),
		"unknown allocation type": withRules("    - {entityType: logs, allocation: 10, allocationType: other, enabled: true, canOverflow: false}\n"),
	}
}

// quotaRulesFake models only the account-level GET, PUT, and DELETE paths used by the
// handwritten controller. It does not make unverified claims about backend validation.
type quotaRulesFake struct {
	mu    sync.Mutex
	rules []any
	next  map[string]int
}

func quotaRule(entityType string, allocation float64, managed bool) map[string]any {
	return map[string]any{
		"entityType": entityType, "allocation": allocation,
		"allocationType": "QUOTA_ALLOCATION_TYPE_PERCENTAGE",
		"enabled":        true, "canOverflow": false, "cxManaged": managed,
	}
}

func (f *quotaRulesFake) seed(rules []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rules = rules
}

func (f *quotaRulesFake) forget() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rules = nil
}

func (f *quotaRulesFake) answerNext(method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.next == nil {
		f.next = map[string]int{}
	}
	f.next[method] = http.StatusInternalServerError
}

func (f *quotaRulesFake) snapshot() any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ruleSet()
}

func (f *quotaRulesFake) ruleSet() map[string]any {
	rules := make([]any, len(f.rules))
	copy(rules, f.rules)
	return map[string]any{"id": "quota-allocation-rule-set", "rules": rules}
}

func (f *quotaRulesFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if path != quotaRulesPath {
		return routeNotFound()
	}
	if status := f.next[method]; status != 0 {
		delete(f.next, method)
		return apiError(status, fmt.Sprintf("quota rules request failed with %d", status))
	}
	switch method {
	case http.MethodGet:
		// EU2 returns an empty rule set with HTTP 200 before any customer rule exists.
		return http.StatusOK, map[string]any{"ruleSet": f.ruleSet()}
	case http.MethodPut:
		in, _ := body.(map[string]any)
		set, _ := in["ruleSet"].(map[string]any)
		rules, _ := set["rules"].([]any)
		managed := map[string]bool{}
		for _, raw := range f.rules {
			rule, _ := raw.(map[string]any)
			isManaged, _ := rule["cxManaged"].(bool)
			if isManaged {
				entityType, _ := rule["entityType"].(string)
				managed[entityType] = true
			}
		}
		f.rules = make([]any, 0, len(rules))
		for _, raw := range rules {
			rule, _ := raw.(map[string]any)
			stored := make(map[string]any, len(rule)+1)
			for key, value := range rule {
				stored[key] = value
			}
			entityType, _ := stored["entityType"].(string)
			stored["cxManaged"] = managed[entityType]
			f.rules = append(f.rules, stored)
		}
		return http.StatusOK, map[string]any{"ruleSet": f.ruleSet()}
	case http.MethodDelete:
		kept := make([]any, 0, len(f.rules))
		for _, raw := range f.rules {
			rule, _ := raw.(map[string]any)
			isManaged, _ := rule["cxManaged"].(bool)
			if isManaged {
				kept = append(kept, raw)
			}
		}
		f.rules = kept
		return http.StatusOK, map[string]any{"id": "quota-allocation-rule-set"}
	}
	return routeNotFound()
}
