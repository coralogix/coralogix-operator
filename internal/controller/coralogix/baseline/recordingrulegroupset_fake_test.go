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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const recordingRulesPath = "/recording-rules/recording-rules/v1"

// ruleGroupRecord is one rule group of a set, as the API stores it.
type ruleGroupRecord struct {
	id       string
	name     string
	interval int64
	limit    string
	rules    []any
	version  int64
}

type ruleGroupSetRecord struct {
	name   string
	groups []ruleGroupRecord
}

// recordingRuleGroupSetFake keeps the remote state that the handwritten controller can change.
type recordingRuleGroupSetFake struct {
	mu     sync.Mutex
	nextID int
	sets   map[string]ruleGroupSetRecord
	forced map[string]forcedAnswer
}

func newRecordingRuleGroupSetFake() apiFake {
	return &recordingRuleGroupSetFake{nextID: 1, sets: map[string]ruleGroupSetRecord{}, forced: map[string]forcedAnswer{}}
}

func (f *recordingRuleGroupSetFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

// newID returns an id of the shape that the API makes (26 characters, a ULID), with a counter so
// that the goldens stay the same on each run.
func (f *recordingRuleGroupSetFake) newID() string {
	id := fmt.Sprintf("01FAKE%020d", f.nextID)
	f.nextID++
	return id
}

// seed adds a set that the operator did not create, and returns its id.
func (f *recordingRuleGroupSetFake) seed(name string, groupNames ...string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.newID()
	set := ruleGroupSetRecord{name: name}
	for _, groupName := range groupNames {
		set.groups = append(set.groups, ruleGroupRecord{id: f.newID(), name: groupName, interval: 60, limit: "0", rules: []any{}})
	}
	f.sets[id] = set
	return id
}

func (f *recordingRuleGroupSetFake) remove(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sets, id)
}

func (f *recordingRuleGroupSetFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.sets))
	for id, set := range f.sets {
		out[id] = set.answer(id)
	}
	return out
}

func (f *recordingRuleGroupSetFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	if path == recordingRulesPath {
		switch method {
		case http.MethodPost:
			return f.create(body)
		}
		return routeNotFound()
	}
	id, ok := strings.CutPrefix(path, recordingRulesPath+"/")
	if !ok || strings.Contains(id, "/") {
		return routeNotFound()
	}
	set, exists := f.sets[id]
	if !exists {
		return apiError(http.StatusNotFound, "No rule group set with the given id")
	}
	switch method {
	case http.MethodGet:
		return http.StatusOK, set.answer(id)
	case http.MethodPut:
		return f.update(id, set, body)
	case http.MethodDelete:
		delete(f.sets, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

var limitPattern = regexp.MustCompile(`^[0-9]+$`)

func (f *recordingRuleGroupSetFake) create(body any) (int, any) {
	in, _ := body.(map[string]any)
	groups, status, answer := f.groupsOf(in, nil)
	if answer != nil {
		return status, answer
	}
	id := f.newID()
	name, hasName := in["name"].(string)
	if !hasName {
		name = id
	}
	if status, answer := f.checkSetName(id, name); answer != nil {
		return status, answer
	}
	f.sets[id] = ruleGroupSetRecord{name: name, groups: groups}
	return http.StatusOK, map[string]any{"id": id}
}

func (f *recordingRuleGroupSetFake) update(id string, set ruleGroupSetRecord, body any) (int, any) {
	in, _ := body.(map[string]any)
	groups, status, answer := f.groupsOf(in, set.groups)
	if answer != nil {
		return status, answer
	}
	if name, ok := in["name"].(string); ok {
		if status, answer := f.checkSetName(id, name); answer != nil {
			return status, answer
		}
		set.name = name
	}
	set.groups = groups
	f.sets[id] = set
	return http.StatusOK, map[string]any{}
}

// checkSetName rejects a set name that is empty, too long, or used by another set.
func (f *recordingRuleGroupSetFake) checkSetName(id, name string) (int, any) {
	if n := len(name); n < 1 || n > 255 {
		return validationError(map[string]any{"name": []any{map[string]any{"code": "length"}}})
	}
	for otherID, other := range f.sets {
		if otherID != id && other.name == name {
			return apiError(http.StatusBadRequest, "duplicate_rule_group_set_name")
		}
	}
	return 0, nil
}

// groupsOf reads and checks the groups of a request. stored holds the groups of the set that an
// update replaces: a group with the name of a stored group keeps its id and counts a new version.
func (f *recordingRuleGroupSetFake) groupsOf(in map[string]any, stored []ruleGroupRecord) ([]ruleGroupRecord, int, any) {
	items, _ := in["groups"].([]any)
	if len(items) == 0 {
		status, answer := apiError(http.StatusBadRequest, "No rule groups in the set")
		return nil, status, answer
	}
	// The API reads the body as protobuf JSON before it checks any rule.
	for _, item := range items {
		group, _ := item.(map[string]any)
		if limit, ok := group["limit"].(string); ok && !limitPattern.MatchString(limit) {
			status, answer := apiError(http.StatusBadRequest, fmt.Sprintf("proto: (line 1:1): invalid value for uint64 field limit: %q", limit))
			return nil, status, answer
		}
		if interval, ok := group["interval"].(float64); ok && (interval < 0 || interval > 4294967295) {
			status, answer := apiError(http.StatusBadRequest, fmt.Sprintf("proto: (line 1:1): invalid value for uint32 field interval: %s", strconv.FormatFloat(interval, 'f', -1, 64)))
			return nil, status, answer
		}
	}
	byName := map[string]ruleGroupRecord{}
	for _, group := range stored {
		byName[group.name] = group
	}
	problems := map[string]any{}
	groupProblems := map[string]any{}
	var out []ruleGroupRecord
	seen := map[string]bool{}
	duplicate := false
	for i, item := range items {
		group, _ := item.(map[string]any)
		name, _ := group["name"].(string)
		fieldProblems := map[string]any{}
		if n := len(name); n < 1 || n > 255 {
			fieldProblems["name"] = []any{map[string]any{"code": "length"}}
		}
		record := ruleGroupRecord{name: name, interval: 60, limit: "0", rules: []any{}}
		if interval, ok := group["interval"].(float64); ok {
			if interval < 60 {
				fieldProblems["interval"] = []any{map[string]any{"code": "range"}}
			}
			record.interval = int64(interval)
		}
		if limit, ok := group["limit"].(string); ok {
			record.limit = limit
		}
		ruleProblems := map[string]any{}
		for r, ruleItem := range asList(group["rules"]) {
			rule, _ := ruleItem.(map[string]any)
			problem := map[string]any{}
			for _, field := range []string{"record", "expr"} {
				value, _ := rule[field].(string)
				if value == "" {
					// An empty value breaks the length rule and is no PromQL either.
					problem[field] = []any{map[string]any{"code": "length"}, map[string]any{"code": "invalid_promql"}}
				}
			}
			if len(problem) != 0 {
				ruleProblems[strconv.Itoa(r)] = problem
			}
			labels, _ := rule["labels"].(map[string]any)
			if labels == nil {
				labels = map[string]any{}
			}
			stored := map[string]any{"record": rule["record"], "expr": rule["expr"], "labels": labels}
			if delay, ok := rule["evaluationDelayMs"]; ok {
				stored["evaluationDelayMs"] = delay
			}
			record.rules = append(record.rules, stored)
		}
		if len(ruleProblems) != 0 {
			fieldProblems["rules"] = ruleProblems
		}
		if len(fieldProblems) != 0 {
			groupProblems[strconv.Itoa(i)] = fieldProblems
		}
		if seen[name] {
			duplicate = true
		}
		seen[name] = true
		if previous, ok := byName[name]; ok {
			record.id, record.version = previous.id, previous.version+1
		} else {
			record.id = f.newID()
		}
		out = append(out, record)
	}
	if len(groupProblems) != 0 {
		problems["groups"] = groupProblems
		status, answer := validationError(problems)
		return nil, status, answer
	}
	if duplicate {
		status, answer := validationError(map[string]any{"groups": []any{map[string]any{"code": "duplicate_rule_group_name"}}})
		return nil, status, answer
	}
	// The API does not keep the order of the groups.
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, 0, nil
}

func asList(v any) []any {
	items, _ := v.([]any)
	return items
}

// validationError is the 400 of a rule that a request breaks: the API writes the failed rules as
// JSON text into the message.
func validationError(problems map[string]any) (int, any) {
	raw, err := json.Marshal(problems)
	if err != nil {
		panic(err)
	}
	return apiError(http.StatusBadRequest, string(raw))
}

func (s ruleGroupSetRecord) answer(id string) map[string]any {
	groups := make([]any, 0, len(s.groups))
	for _, g := range s.groups {
		rules := g.rules
		if rules == nil {
			rules = []any{}
		}
		groups = append(groups, map[string]any{
			"id": g.id, "name": g.name, "interval": g.interval, "limit": g.limit, "rules": rules, "version": g.version,
		})
	}
	return map[string]any{"id": id, "name": s.name, "groups": groups}
}

// normalizeRecordingRuleGroupSetAnswer drops what the fake cannot make and the operator never
// reads: the ids of the groups, and the text of a validation error. It keeps which rules failed,
// by path and code.
func normalizeRecordingRuleGroupSetAnswer(resp any) any {
	m, ok := resp.(map[string]any)
	if !ok {
		return resp
	}
	if groups, ok := m["groups"].([]any); ok {
		for _, g := range groups {
			if group, ok := g.(map[string]any); ok {
				if _, has := group["id"]; has {
					group["id"] = "<group-id>"
				}
			}
		}
	}
	if message, ok := m["message"].(string); ok {
		m["message"] = normalizeValidationMessage(message)
	}
	return m
}

var protoPositionPattern = regexp.MustCompile(`\(line \d+:\d+\)`)

// normalizeValidationMessage rewrites a validation error to the list of failed paths and codes,
// and the position of a protobuf JSON error to a fixed text.
func normalizeValidationMessage(message string) string {
	text, ok := strings.CutPrefix(message, "Bad Request: ")
	if !ok {
		return message
	}
	text = protoPositionPattern.ReplaceAllString(text, "(line N:N)")
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return "Bad Request: " + text
	}
	var found []string
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch v := v.(type) {
		case map[string]any:
			for key, value := range v {
				walk(path+"."+key, value)
			}
		case []any:
			for i, value := range v {
				if m, ok := value.(map[string]any); ok {
					if code, ok := m["code"].(string); ok {
						found = append(found, strings.TrimPrefix(path, ".")+": "+code)
						continue
					}
				}
				walk(fmt.Sprintf("%s[%d]", path, i), value)
			}
		}
	}
	walk("", parsed)
	sort.Strings(found)
	return "Bad Request: validation failed: " + strings.Join(found, "; ")
}
