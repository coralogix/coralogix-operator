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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
)

const ruleGroupPath = "/parsing-rules/rule-groups/v1"

// ruleParameterArms are the oneOf arms of a rule's parameters.
var ruleParameterArms = []string{
	"extractParameters", "jsonExtractParameters", "replaceParameters", "parseParameters",
	"allowParameters", "blockParameters", "extractTimestampParameters", "removeFieldsParameters",
	"jsonStringifyParameters", "jsonParseParameters",
}

// ruleGroupRecord is a stored rule group. The subgroups and their rules are kept as the request
// sent them, plus the ids that the service makes on every write.
type ruleGroupRecord struct {
	name        string
	description string
	creator     *string
	enabled     bool
	hidden      bool
	order       int64
	matchers    []any
	subgroups   []any
}

// ruleGroupFake is the parsing rule groups API, written from the SDK's openapi.yaml and the
// backend (RuleGroupsServiceImpl).
//
//	POST   /parsing-rules/rule-groups/v1       {name, description, enabled, hidden, creator, order,
//	                                            ruleMatchers, ruleSubgroups} -> {ruleGroup}
//	GET    /parsing-rules/rule-groups/v1/{id}  -> {ruleGroup}
//	PUT    /parsing-rules/rule-groups/v1/{id}  the Create body -> {ruleGroup}
//	DELETE /parsing-rules/rule-groups/v1/{id}  -> {}
//
// The status codes follow fidelity/recorded.json, which the real API answered:
//   - Update and Delete read the group first, so a missing group is a 404 for both.
//   - Update replaces the whole group. An omitted field takes its default, and the service sets a
//     new id for every subgroup and rule.
//   - A name that is empty or missing, a group without a subgroup, a subgroup without a rule, a
//     rule without a name, a source field, or exactly one parameters arm, and an empty regular
//     expression are 400s.
type ruleGroupFake struct {
	mu     sync.Mutex
	groups map[string]ruleGroupRecord
	forced map[string]forcedAnswer
}

func newRuleGroupFake() apiFake {
	return &ruleGroupFake{groups: map[string]ruleGroupRecord{}, forced: map[string]forcedAnswer{}}
}

// answerNext makes the next request with the method get this answer instead of the fake's own.
func (f *ruleGroupFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

// forget deletes a group without a request, as if it was deleted outside the operator.
func (f *ruleGroupFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.groups, id)
}

// put stores a group without a request, as if another client created it.
func (f *ruleGroupFake) put(id string, group ruleGroupRecord) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.groups[id] = group
}

func (f *ruleGroupFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.groups))
	for id, group := range f.groups {
		out[id] = group.answer(id)
	}
	return out
}

func (f *ruleGroupFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	in, _ := body.(map[string]any)

	if path == ruleGroupPath {
		switch method {
		case http.MethodPost:
			group, status, errBody := f.ruleGroupFromRequest(in, method)
			if status != 0 {
				return status, errBody
			}
			id := newUUID()
			f.groups[id] = group
			return http.StatusOK, map[string]any{"ruleGroup": group.answer(id)}
		case http.MethodGet:
			list := make([]any, 0, len(f.groups))
			for id, group := range f.groups {
				list = append(list, group.answer(id))
			}
			return http.StatusOK, map[string]any{"ruleGroups": list}
		}
		return routeNotFound()
	}

	id, ok := strings.CutPrefix(path, ruleGroupPath+"/")
	if !ok || id == "" || strings.Contains(id, "/") {
		return routeNotFound()
	}
	_, exists := f.groups[id]
	switch method {
	case http.MethodGet:
		if !exists {
			return ruleGroupNotFound(id)
		}
		return http.StatusOK, map[string]any{"ruleGroup": f.groups[id].answer(id)}
	case http.MethodPut:
		// The service reads the group before it writes, so a missing group is a 404 before the
		// body is checked.
		if !exists {
			return ruleGroupNotFound(id)
		}
		group, status, errBody := f.ruleGroupFromRequest(in, method)
		if status != 0 {
			return status, errBody
		}
		f.groups[id] = group
		return http.StatusOK, map[string]any{"ruleGroup": group.answer(id)}
	case http.MethodDelete:
		if !exists {
			return ruleGroupNotFound(id)
		}
		delete(f.groups, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

func ruleGroupNotFound(id string) (int, any) {
	return apiError(http.StatusNotFound, "RuleGroup not found: "+id)
}

// ruleGroupFromRequest reads the group of a create or replace body, in the order that the real API
// checks it: the gateway parses the JSON, the service converts the request, and the backend
// validates the group. A non-zero status is an error answer.
func (f *ruleGroupFake) ruleGroupFromRequest(in map[string]any, verb string) (ruleGroupRecord, int, any) {
	bad := func(reason string) (ruleGroupRecord, int, any) {
		status, body := apiError(http.StatusBadRequest, reason)
		return ruleGroupRecord{}, status, body
	}
	subgroups, _ := in["ruleSubgroups"].([]any)
	ruleOf := func(raw any) map[string]any { rule, _ := raw.(map[string]any); return rule }
	rulesOf := func(raw any) []any {
		subgroup, _ := raw.(map[string]any)
		rules, _ := subgroup["rules"].([]any)
		return rules
	}

	// 1. The gateway reads the JSON as the protobuf message.
	if order, ok := in["order"].(float64); ok && order < 0 {
		return bad(fmt.Sprintf("proto: (line 1:1): invalid value for uint32 field value: %d", int64(order)))
	}
	matchers, _ := in["ruleMatchers"].([]any)
	for _, raw := range matchers {
		matcher, _ := raw.(map[string]any)
		if severity, ok := matcher["severity"].(map[string]any); ok {
			if value, _ := severity["value"].(string); !ruleSeverityValues[value] {
				return bad(fmt.Sprintf("proto: (line 1:1): invalid value for enum field value: %q", value))
			}
		}
	}
	for _, subgroup := range subgroups {
		for _, raw := range rulesOf(subgroup) {
			parameters, _ := ruleOf(raw)["parameters"].(map[string]any)
			var set []string
			for _, arm := range ruleParameterArms {
				if _, ok := parameters[arm].(map[string]any); ok {
					set = append(set, arm)
				}
			}
			sort.Strings(set)
			if len(set) > 1 {
				return bad(fmt.Sprintf(`proto: (line 1:1): error parsing %q, oneof com.coralogix.rules.v1.RuleParameters.rule_parameters is already set`, set[1]))
			}
		}
	}

	// 2. The service converts the request and reports every problem at once.
	var problems []string
	if _, ok := in["name"].(string); !ok {
		problems = append(problems, "Missing CreateRuleGroup.name")
	}
	for _, raw := range matchers {
		matcher, _ := raw.(map[string]any)
		if matcher["applicationName"] == nil && matcher["subsystemName"] == nil && matcher["severity"] == nil {
			problems = append(problems, "RuleMatcher.Constraint.Empty")
		}
	}
	for _, subgroup := range subgroups {
		for _, raw := range rulesOf(subgroup) {
			rule := ruleOf(raw)
			if _, ok := rule["name"].(string); !ok {
				problems = append(problems, "Missing CreateRule.name")
			}
			if _, ok := rule["sourceField"].(string); !ok {
				problems = append(problems, "Missing CreateRule.sourceField")
			}
			if _, ok := rule["order"].(float64); !ok {
				problems = append(problems, "Missing CreateRule.order")
			}
			switch parameters, ok := rule["parameters"].(map[string]any); {
			case !ok:
				problems = append(problems, "Missing CreateRule.RuleParameters")
			case len(parameters) == 0:
				problems = append(problems, "Empty RuleParameters")
			}
		}
		if sub, _ := subgroup.(map[string]any); sub["order"] == nil {
			problems = append(problems, "Missing CreateRuleSubgroup.order")
		}
	}
	if len(problems) != 0 {
		return bad("Request errors:\n* " + strings.Join(problems, "\n* "))
	}

	// 3. The backend validates the group. An update names the check "validate".
	var failures []string
	if name, _ := in["name"].(string); name == "" {
		failures = append(failures, "{required ParsingThemes.Name string string  }")
	}
	if len(subgroups) == 0 {
		failures = append(failures, "{min ParsingThemes.RulesGroups slice []*shared.RulesGroup [] 1}")
	}
	for i, subgroup := range subgroups {
		rules := rulesOf(subgroup)
		path := fmt.Sprintf("ParsingThemes.RulesGroups[%d].Rules", i)
		if len(rules) == 0 {
			failures = append(failures, fmt.Sprintf("{min %s slice []*shared.Rule [] 1}", path))
		}
		for j, raw := range rules {
			rule := ruleOf(raw)
			rulePath := fmt.Sprintf("%s[%d]", path, j)
			if name, _ := rule["name"].(string); name == "" {
				failures = append(failures, fmt.Sprintf("{required %s.Name string string  }", rulePath))
			}
			if sourceField, _ := rule["sourceField"].(string); sourceField == "" {
				failures = append(failures, fmt.Sprintf("{source-field-validation %s.SourceField string string  }", rulePath))
			}
			parameters, _ := rule["parameters"].(map[string]any)
			if ruleParametersRegex(parameters) == "" {
				failures = append(failures, fmt.Sprintf("{required %s.Rule string string  }", rulePath))
			}
		}
	}
	if len(failures) != 0 {
		action := "create"
		if verb == http.MethodPut {
			action = "validate"
		}
		return bad(fmt.Sprintf("statusCode: 400, response: Failed to %s rule group err: [%s]", action, strings.Join(failures, " ")))
	}

	group := ruleGroupRecord{enabled: true, order: 1}
	group.name, _ = in["name"].(string)
	group.description, _ = in["description"].(string)
	creator, _ := in["creator"].(string)
	group.creator = &creator
	if enabled, ok := in["enabled"].(bool); ok {
		group.enabled = enabled
	}
	group.hidden, _ = in["hidden"].(bool)
	if order, ok := in["order"].(float64); ok {
		group.order = int64(order)
		if group.order == 0 {
			// An order of 0 places the group after every other one.
			group.order = 1
			for _, other := range f.groups {
				group.order = max(group.order, other.order+1)
			}
		}
	}
	group.matchers = matchers
	for _, raw := range subgroups {
		subgroup, _ := raw.(map[string]any)
		stored := map[string]any{"id": newUUID(), "enabled": true, "order": subgroup["order"]}
		if enabled, ok := subgroup["enabled"].(bool); ok {
			stored["enabled"] = enabled
		}
		var storedRules []any
		for _, rawRule := range rulesOf(raw) {
			rule := ruleOf(rawRule)
			storedRule := map[string]any{"id": newUUID(), "enabled": true}
			for _, key := range []string{"name", "sourceField", "order"} {
				storedRule[key] = rule[key]
			}
			storedRule["parameters"] = storedParameters(rule["parameters"].(map[string]any))
			storedRule["description"], _ = rule["description"].(string)
			if enabled, ok := rule["enabled"].(bool); ok {
				storedRule["enabled"] = enabled
			}
			storedRules = append(storedRules, storedRule)
		}
		stored["rules"] = storedRules
		group.subgroups = append(group.subgroups, stored)
	}
	return group, 0, nil
}

var ruleSeverityValues = map[string]bool{
	"VALUE_DEBUG_OR_UNSPECIFIED": true, "VALUE_VERBOSE": true, "VALUE_INFO": true,
	"VALUE_WARNING": true, "VALUE_ERROR": true, "VALUE_CRITICAL": true,
}

// ruleParametersRegex returns the regular expression of the one parameters arm, as the backend
// stores it in the field "rule": the joined fields of removeFields, and "-" for the arms that have
// none. A rule with an empty value fails the backend check.
func ruleParametersRegex(parameters map[string]any) string {
	for _, arm := range ruleParameterArms {
		value, ok := parameters[arm].(map[string]any)
		if !ok {
			continue
		}
		switch arm {
		case "removeFieldsParameters":
			var fields []string
			list, _ := value["fields"].([]any)
			for _, field := range list {
				text, _ := field.(string)
				fields = append(fields, text)
			}
			return strings.Join(fields, ",")
		case "extractTimestampParameters", "jsonStringifyParameters", "jsonParseParameters":
			return "-"
		}
		rule, _ := value["rule"].(string)
		return rule
	}
	return ""
}

// storedParameters drops what the service does not return: the escapedValue of a JSON parse rule.
func storedParameters(parameters map[string]any) map[string]any {
	out := map[string]any{}
	for arm, value := range parameters {
		if fields, ok := value.(map[string]any); ok && arm == "jsonParseParameters" {
			kept := map[string]any{}
			for key, v := range fields {
				if key != "escapedValue" {
					kept[key] = v
				}
			}
			value = kept
		}
		out[arm] = value
	}
	return out
}

func (g ruleGroupRecord) answer(id string) map[string]any {
	out := map[string]any{
		"id":            id,
		"name":          g.name,
		"description":   g.description,
		"enabled":       g.enabled,
		"hidden":        g.hidden,
		"order":         g.order,
		"ruleSubgroups": g.subgroups,
		"ruleMatchers":  []any{},
	}
	if g.creator != nil {
		out["creator"] = *g.creator
	} else {
		out["creator"] = ""
	}
	if len(g.matchers) != 0 {
		out["ruleMatchers"] = g.matchers
	}
	return out
}
