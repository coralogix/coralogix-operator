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
	"strings"
	"sync"
)

// alertSchedulerFake is the alert scheduler rule API, written from the SDK's openapi.yaml.
//
//	POST   /alerts/suppression-rules/v1        {alertSchedulerRule}                    -> {alertSchedulerRule}
//	PUT    /alerts/suppression-rules/v1        {alertSchedulerRule: {uniqueIdentifier}} -> {alertSchedulerRule}
//	GET    /alerts/suppression-rules/v1/{id}   -> {alertSchedulerRule}
//	DELETE /alerts/suppression-rules/v1/{id}   -> {}
//
// The status codes, error texts, and stored values follow fidelity/recorded.json. Each rule names
// the recorded case that shows it. The real API moves a start time that is not a valid occurrence
// of a recurring schedule to the next one; the fake keeps it, and the cases use valid start times.
type alertSchedulerFake struct {
	mu     sync.Mutex
	rules  map[string]map[string]any
	alerts map[string]bool
	// next holds one forced answer per method. The next request with the method gets it, and the
	// fake does not run the request.
	next map[string]forcedAnswer
}

func newAlertSchedulerFake() apiFake {
	return &alertSchedulerFake{
		rules:  map[string]map[string]any{},
		alerts: map[string]bool{asAlertID: true, asOtherAlertID: true},
		next:   map[string]forcedAnswer{},
	}
}

func (f *alertSchedulerFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = forcedAnswer{status: status, body: body}
}

func (f *alertSchedulerFake) answerNextError(method string, status int, reason string) {
	st, body := apiError(status, reason)
	f.answerNext(method, st, body)
}

func (f *alertSchedulerFake) answerNextRouteNotFound(method string) {
	st, body := routeNotFound()
	f.answerNext(method, st, body)
}

// seed stores the rule that a released operator created for asMinimal, and returns its
// uniqueIdentifier.
func (f *alertSchedulerFake) seed() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	status, resp := f.create(map[string]any{"alertSchedulerRule": map[string]any{
		"name":        "Baseline scheduler",
		"description": "",
		"enabled":     true,
		"filter": map[string]any{
			"whatExpression":  "source logs | filter true",
			"alertMetaLabels": map[string]any{"value": []any{map[string]any{"key": "app", "value": "cxo"}}},
		},
		"schedule": map[string]any{
			"scheduleOperation": "SCHEDULE_OPERATION_MUTE",
			"oneTime": map[string]any{"timeframe": map[string]any{
				"startTime": "2027-01-04T00:00:00.000",
				"timezone":  "UTC+0",
				"endTime":   "2027-01-05T00:00:00.000",
			}},
		},
	}})
	if status != http.StatusOK {
		panic(fmt.Sprintf("seed: %d %v", status, resp))
	}
	return resp.(map[string]any)["alertSchedulerRule"].(map[string]any)["uniqueIdentifier"].(string)
}

// remove deletes a rule, as a user of the UI could.
func (f *alertSchedulerFake) remove(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rules, id)
}

// rule returns a copy of the stored rule, or nil.
func (f *alertSchedulerFake) rule(id string) any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.rules[id]; ok {
		return deepCopyJSON(r)
	}
	return nil
}

func (f *alertSchedulerFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.next[method]; ok && strings.HasPrefix(path, alertSchedulerPath) {
		delete(f.next, method)
		return forced.status, forced.body
	}
	switch {
	case path == alertSchedulerPath && method == http.MethodPost:
		return f.create(body)
	case path == alertSchedulerPath && method == http.MethodPut:
		return f.replace(body)
	case strings.HasPrefix(path, alertSchedulerPath+"/") && !strings.Contains(strings.TrimPrefix(path, alertSchedulerPath+"/"), "/"):
		id := strings.TrimPrefix(path, alertSchedulerPath+"/")
		switch method {
		case http.MethodGet:
			return f.get(id)
		case http.MethodDelete:
			return f.delete(id)
		}
	}
	return routeNotFound()
}

func (f *alertSchedulerFake) get(id string) (int, any) {
	if !isUUID(id) {
		return apiError(http.StatusBadRequest, "[ alertSchedulerRuleId ] Invalid uuid")
	}
	r, ok := f.rules[id]
	if !ok {
		return apiError(http.StatusNotFound, "No suppression rule found with unique identifier "+id)
	}
	// Case "every schedule shape": a read has no submittedStartTime.
	out := deepCopyJSON(r).(map[string]any)
	dropSubmittedStartTime(out)
	return http.StatusOK, map[string]any{"alertSchedulerRule": out}
}

func (f *alertSchedulerFake) delete(id string) (int, any) {
	status, resp := f.get(id)
	if status != http.StatusOK {
		return status, resp
	}
	delete(f.rules, id)
	return http.StatusOK, map[string]any{}
}

func (f *alertSchedulerFake) create(body any) (int, any) {
	in, status, resp := f.validate(body)
	if in == nil {
		return status, resp
	}
	id := newUUID()
	in["uniqueIdentifier"] = id
	f.rules[id] = f.store(in)
	return http.StatusOK, map[string]any{"alertSchedulerRule": deepCopyJSON(f.rules[id])}
}

// replace follows the cases "create, read, replace, and delete ..." and "unknown and malformed ids".
func (f *alertSchedulerFake) replace(body any) (int, any) {
	in, status, resp := f.validate(body)
	if in == nil {
		return status, resp
	}
	const required = "[ alertSchedulerRule ] uniqueIdentifier is required; the value provided in the id field is a version id and cannot be used to update a rule"
	id, present := in["uniqueIdentifier"].(string)
	switch {
	case !present:
		return apiError(http.StatusBadRequest, required)
	case id == "":
		return apiError(http.StatusBadRequest, "[ alertSchedulerRule.uniqueIdentifier ] Invalid uuid, "+required)
	case !isUUID(id):
		return apiError(http.StatusBadRequest, "[ alertSchedulerRule.uniqueIdentifier ] Invalid uuid")
	}
	if _, ok := f.rules[id]; !ok {
		return apiError(http.StatusNotFound, "No result found")
	}
	f.rules[id] = f.store(in)
	return http.StatusOK, map[string]any{"alertSchedulerRule": deepCopyJSON(f.rules[id])}
}

// validate returns the rule of the request, or nil and the error answer. It follows the case
// "rejected rules".
func (f *alertSchedulerFake) validate(body any) (map[string]any, int, any) {
	root, _ := body.(map[string]any)
	in, ok := root["alertSchedulerRule"].(map[string]any)
	if !ok {
		st, resp := apiError(http.StatusBadRequest, "[ alertSchedulerRule ] Required")
		return nil, st, resp
	}
	in = deepCopyJSON(in).(map[string]any)
	if name, _ := in["name"].(string); name == "" {
		st, resp := apiError(http.StatusBadRequest, "[ alertSchedulerRule.name ] String must contain at least 1 character(s)")
		return nil, st, resp
	}
	filter, _ := in["filter"].(map[string]any)
	if what, _ := filter["whatExpression"].(string); !strings.HasPrefix(what, "source logs | filter") {
		st, resp := apiError(http.StatusBadRequest, `[ alertSchedulerRule.filter.whatExpression ] Invalid dataPrime expression : {"syntax_error":"expression does not start with 'source logs | filter'"}`)
		return nil, st, resp
	}
	labels, hasLabels := filter["alertMetaLabels"].(map[string]any)
	ids, hasIDs := filter["alertUniqueIds"].(map[string]any)
	if !hasLabels && !hasIDs {
		st, resp := apiError(http.StatusBadRequest, "[ alertSchedulerRule.filter.whichAlerts ] Invalid input")
		return nil, st, resp
	}
	if hasIDs {
		values, _ := ids["value"].([]any)
		for _, v := range values {
			if id, _ := v.(string); !f.alerts[id] {
				st, resp := apiError(http.StatusBadRequest, "[ alertSchedulerRule.filter.whichAlerts.alertUniqueIds.value ] invalid alert uniqe ids")
				return nil, st, resp
			}
		}
	}
	if hasLabels {
		labels["value"] = withLabelIDs(labels["value"])
	}
	schedule, _ := in["schedule"].(map[string]any)
	if op, _ := schedule["scheduleOperation"].(string); op == "" || op == "SCHEDULE_OPERATION_UNSPECIFIED" {
		st, resp := apiError(http.StatusBadRequest, "[ alertSchedulerRule.schedule.scheduleOperation ] ScheduleOperation value UNSPECIFIED is not allowed")
		return nil, st, resp
	}
	if st, resp := normalizeSchedule(schedule); st != 0 {
		return nil, st, resp
	}
	return in, 0, nil
}

// normalizeSchedule checks a schedule and writes the values that the API adds. It returns a
// status and an answer for a schedule that the API rejects, or 0.
func normalizeSchedule(schedule map[string]any) (int, any) {
	if oneTime, ok := schedule["oneTime"].(map[string]any); ok {
		tf, _ := oneTime["timeframe"].(map[string]any)
		return normalizeTimeframe(tf)
	}
	recurring, _ := schedule["recurring"].(map[string]any)
	dynamic, ok := recurring["schedule"].(map[string]any)
	if !ok {
		return 0, nil
	}
	// Case "a recurring schedule with repeatEvery 0 becomes always active".
	if every, _ := dynamic["repeatEvery"].(float64); every == 0 {
		schedule["recurring"] = map[string]any{"alwaysActive": map[string]any{}}
		return 0, nil
	}
	if weekly, ok := dynamic["weekly"].(map[string]any); ok {
		days, _ := weekly["daysOfWeek"].([]any)
		for i, d := range days {
			if n, _ := d.(float64); n < 1 || n > 7 {
				return apiError(http.StatusBadRequest, fmt.Sprintf("[ alertSchedulerRule.schedule.scheduler.recurring.condition.schedule.frequency.weekly.daysOfWeek.%d ] Invalid days in week (accept 1-7)", i))
			}
		}
	}
	if monthly, ok := dynamic["monthly"].(map[string]any); ok {
		days, _ := monthly["daysOfMonth"].([]any)
		for i, d := range days {
			if n, _ := d.(float64); n < 1 || n > 31 {
				return apiError(http.StatusBadRequest, fmt.Sprintf("[ alertSchedulerRule.schedule.scheduler.recurring.condition.schedule.frequency.monthly.daysOfMonth.%d ] Invalid days in month (accept 1-31)", i))
			}
		}
	}
	if _, ok := dynamic["terminationDate"]; !ok {
		dynamic["terminationDate"] = ""
	}
	tf, _ := dynamic["timeframe"].(map[string]any)
	return normalizeTimeframe(tf)
}

func normalizeTimeframe(tf map[string]any) (int, any) {
	_, hasEnd := tf["endTime"]
	duration, hasDuration := tf["duration"].(map[string]any)
	if !hasEnd && !hasDuration {
		return apiError(http.StatusBadRequest, "[ alertSchedulerRule.schedule.scheduler ] Invalid input")
	}
	if hasDuration {
		if _, ok := duration["forOver"]; !ok {
			duration["forOver"] = float64(0)
		}
	}
	start, _ := tf["startTime"].(string)
	tf["submittedStartTime"] = start
	return 0, nil
}

// withLabelIDs gives each meta label a new id, as the API does.
func withLabelIDs(v any) any {
	labels, _ := v.([]any)
	out := make([]any, 0, len(labels))
	for _, l := range labels {
		m, _ := l.(map[string]any)
		label := map[string]any{"id": newUUID()}
		for k, val := range m {
			if k != "id" {
				label[k] = val
			}
		}
		if _, ok := label["key"]; !ok {
			label["key"] = ""
		}
		if value, _ := label["value"].(string); value == "" {
			delete(label, "value")
		}
		out = append(out, label)
	}
	return out
}

// store returns the rule as the API stores it: a new version id, and defaults for the fields that
// the request left out (case "optional fields left out").
func (f *alertSchedulerFake) store(in map[string]any) map[string]any {
	in["id"] = newUUID()
	if _, ok := in["description"]; !ok {
		in["description"] = ""
	}
	if _, ok := in["enabled"]; !ok {
		in["enabled"] = false
	}
	in["metaLabels"] = withLabelIDs(in["metaLabels"])
	in["createdAt"] = "2026-01-01T00:00:00.000Z"
	in["updatedAt"] = "2026-01-01T00:00:00.000Z"
	return in
}

func isUUID(id string) bool {
	return len(id) == 36 && uuidPattern.MatchString(strings.ToLower(id))
}

func dropSubmittedStartTime(rule map[string]any) {
	schedule, _ := rule["schedule"].(map[string]any)
	if oneTime, ok := schedule["oneTime"].(map[string]any); ok {
		if tf, ok := oneTime["timeframe"].(map[string]any); ok {
			delete(tf, "submittedStartTime")
		}
	}
	recurring, _ := schedule["recurring"].(map[string]any)
	if dynamic, ok := recurring["schedule"].(map[string]any); ok {
		if tf, ok := dynamic["timeframe"].(map[string]any); ok {
			delete(tf, "submittedStartTime")
		}
	}
}

func deepCopyJSON(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}
