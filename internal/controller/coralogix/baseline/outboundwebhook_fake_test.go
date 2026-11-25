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
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

const webhookPath = "/integrations/webhooks/v1"

// webhookTime is the creation and update time of every webhook of the fake, so goldens stay the same.
const webhookTime = "2026-01-01T00:00:00Z"

// webhookArms maps each config of a webhook to its type, in the order of the contract.
var webhookArms = []struct{ field, webhookType string }{
	{"genericWebhook", "GENERIC"},
	{"slack", "SLACK"},
	{"pagerDuty", "PAGERDUTY"},
	{"sendLog", "SEND_LOG"},
	{"emailGroup", "EMAIL_GROUP"},
	{"microsoftTeams", "MICROSOFT_TEAMS"},
	{"jira", "JIRA"},
	{"opsgenie", "OPSGENIE"},
	{"demisto", "DEMISTO"},
	{"awsEventBridge", "AWS_EVENT_BRIDGE"},
	{"ibmEventNotifications", "IBM_EVENT_NOTIFICATIONS"},
	{"msTeamsWorkflow", "MS_TEAMS_WORKFLOW"},
}

// webhookTypeNames are the names of the types in the type-mismatch error of Update.
var webhookTypeNames = map[string]string{"GENERIC": "WebHook", "PAGERDUTY": "PagerDuty"}

// webhooksWithURL are the types that need a URL.
var webhooksWithURL = map[string]bool{
	"GENERIC": true, "SLACK": true, "JIRA": true, "OPSGENIE": true, "DEMISTO": true, "MS_TEAMS_WORKFLOW": true,
}

type webhookRecord struct {
	webhookType string
	name        string
	url         *string
	// config is the config object of the type, as the API answers it.
	config     map[string]any
	externalID int64
	createdAt  string
	updatedAt  string
}

// webhookFake is the outgoing webhooks API, written from the SDK's openapi.yaml.
//
//	POST   /integrations/webhooks/v1        {data: {name, type, url, <config>}} -> {id}
//	PUT    /integrations/webhooks/v1        {id, data: {...}}                   -> {}
//	GET    /integrations/webhooks/v1/{id}   -> {webhook: {id, type, name, url, externalId, <config>}}
//	DELETE /integrations/webhooks/v1/{id}   -> {}
//
// The status codes and error texts follow fidelity/recorded.json, which the real API answered:
//   - The config sets the type. The type in the request is ignored, also when it names another type.
//   - Update cannot change the type: 400 "Webhook type mismatch". It replaces the whole config,
//     so an omitted payload or headers are cleared.
//   - A missing webhook is a 404 on Get, Update, and Delete. An id that is not a UUID is a 503.
//   - Generic, Send Log, and Demisto need a UUID. A method UNKNOWN, a digest UNKNOWN, and an
//     attachment EMPTY are a 400.
//   - Generic, Slack, Jira, Opsgenie, Demisto, and MS Teams workflow need a URL. Send Log takes no
//     URL. Microsoft Teams is deprecated, so every write is a 400.
//   - Slack keeps one digest and one attachment of each type: the last one. Email Group removes
//     duplicate addresses. An empty address list is accepted.
//   - Names need not be unique.
type webhookFake struct {
	mu       sync.Mutex
	webhooks map[string]*webhookRecord
	nextExt  int64
	forced   map[string]forcedAnswer
}

func newWebhookFake() apiFake {
	return &webhookFake{webhooks: map[string]*webhookRecord{}, nextExt: 1000, forced: map[string]forcedAnswer{}}
}

// answerNext makes the next request with the method get this answer instead of the fake's own.
func (f *webhookFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

// forget deletes a webhook without a request, as if it was deleted outside the operator.
func (f *webhookFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.webhooks, id)
}

// putRequest stores a webhook from a request body without a request, as if another client made it.
func (f *webhookFake) putRequest(id string, data map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	record, status, _ := webhookFromRequest(data)
	if status != 0 {
		panic("putRequest: the webhook is not valid")
	}
	f.nextExt++
	record.externalID = f.nextExt
	record.createdAt = webhookTime
	record.updatedAt = record.createdAt
	f.webhooks[id] = record
}

func (f *webhookFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.webhooks))
	for id, w := range f.webhooks {
		out[id] = w.answer(id)
	}
	return out
}

func (f *webhookFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	in, _ := body.(map[string]any)

	if path == webhookPath {
		data, _ := in["data"].(map[string]any)
		switch method {
		case http.MethodPost:
			record, status, errBody := webhookFromRequest(data)
			if status != 0 {
				return status, errBody
			}
			f.nextExt++
			record.externalID = f.nextExt
			record.createdAt = webhookTime
			record.updatedAt = record.createdAt
			id := newUUID()
			f.webhooks[id] = record
			return http.StatusOK, map[string]any{"id": id}

		case http.MethodPut:
			id, _ := in["id"].(string)
			existing, status, errBody := f.lookup(id)
			if status != 0 {
				return status, errBody
			}
			record, status, errBody := webhookFromRequest(data)
			if status != 0 {
				return status, errBody
			}
			if record.webhookType != existing.webhookType {
				_, status, errBody := webhookBadRequest("Webhook type mismatch: expected " + webhookTypeName(existing.webhookType) + ", got " + webhookTypeName(record.webhookType))
				return status, errBody
			}
			record.externalID = existing.externalID
			record.createdAt = existing.createdAt
			record.updatedAt = webhookTime
			f.webhooks[id] = record
			return http.StatusOK, map[string]any{}
		}
		return routeNotFound()
	}

	id, ok := strings.CutPrefix(path, webhookPath+"/")
	if !ok || id == "" || strings.Contains(id, "/") {
		return routeNotFound()
	}
	switch method {
	case http.MethodGet:
		record, status, errBody := f.lookup(id)
		if status != 0 {
			return status, errBody
		}
		return http.StatusOK, map[string]any{"webhook": record.answer(id)}
	case http.MethodDelete:
		if _, status, errBody := f.lookup(id); status != 0 {
			return status, errBody
		}
		delete(f.webhooks, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

// lookup finds a webhook. A non-zero status is an error answer.
func (f *webhookFake) lookup(id string) (*webhookRecord, int, any) {
	if !webhookUUID.MatchString(id) {
		return nil, http.StatusServiceUnavailable, map[string]any{"code": http.StatusServiceUnavailable, "message": "Service Unavailable: Upstream service error"}
	}
	record, ok := f.webhooks[id]
	if !ok {
		status, body := apiError(http.StatusNotFound, "Outgoing webhook not found: "+id)
		return nil, status, body
	}
	return record, 0, nil
}

var webhookUUID = regexp.MustCompile(`^(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func webhookTypeName(webhookType string) string {
	if name, ok := webhookTypeNames[webhookType]; ok {
		return name
	}
	return webhookType
}

// webhookFromRequest reads the webhook of a create or update body. A non-zero status is an error
// answer.
func webhookFromRequest(data map[string]any) (*webhookRecord, int, any) {
	record := &webhookRecord{webhookType: "UNKNOWN"}
	var errs []string
	record.name, _ = data["name"].(string)
	if record.name == "" {
		errs = append(errs, "data must not be empty")
	}
	for _, arm := range webhookArms {
		if config, ok := data[arm.field].(map[string]any); ok {
			record.webhookType = arm.webhookType
			record.config = roundTripMap(config)
			break
		}
	}
	if record.webhookType == "UNKNOWN" {
		return webhookBadRequest(append(errs, configError("Unknown config for webhook type: UNKNOWN}"))...)
	}
	if record.webhookType == "MICROSOFT_TEAMS" {
		return webhookBadRequest("Webhook type MICROSOFT_TEAMS is deprecated due to Office 365 connectors deprecation within Microsoft Teams. Please create the webhook of type MS_TEAMS_WORKFLOW type instead. Refer to this guide for details: https://coralogix.com/docs/workflow-based-microsoft-teams-outbound-webhooks/.")
	}

	if raw, ok := data["url"].(string); ok {
		record.url = &raw
	}
	switch {
	case record.webhookType == "SEND_LOG" && record.url != nil && *record.url != "":
		errs = append(errs, "URL not allowed: Configuration of URL not supported for SendLog webhook type")
	case record.url != nil && *record.url == "" && (webhooksWithURL[record.webhookType] || record.webhookType == "SEND_LOG"):
		errs = append(errs, configError("data must be a valid uri"))
	case record.url == nil && webhooksWithURL[record.webhookType]:
		errs = append(errs, configError("Missing URL"))
	case record.url != nil && webhooksWithURL[record.webhookType]:
		if parsed, err := url.Parse(*record.url); err != nil || !parsed.IsAbs() {
			errs = append(errs, "Failed to parse provided URL: relative URL without a base")
		}
	}

	c := record.config
	switch record.webhookType {
	case "GENERIC":
		errs = append(errs, uuidErrors(c)...)
		if method, _ := c["method"].(string); method == "" || method == "UNKNOWN" {
			errs = append(errs, configError("Unknown method type: UNKNOWN"))
		}
		if payload, ok := c["payload"].(string); ok && payload == "" {
			errs = append(errs, configError("payload must not be empty"))
		}
		if _, ok := c["headers"]; !ok {
			c["headers"] = map[string]any{}
		}
	case "SLACK":
		for _, list := range []struct{ field, zero, label string }{{"digests", "UNKNOWN", "digest"}, {"attachments", "EMPTY", "attachment"}} {
			items, ok := c[list.field].([]any)
			if !ok {
				continue
			}
			for _, item := range items {
				m, _ := item.(map[string]any)
				if t, _ := m["type"].(string); t == "" || t == list.zero {
					errs = append(errs, configError("Unknown "+list.label+" type: "+list.zero))
				}
			}
			c[list.field] = lastOfEachType(items)
		}
	case "PAGERDUTY":
		errs = append(errs, emptyErrors(c, "serviceKey")...)
	case "SEND_LOG":
		errs = append(errs, uuidErrors(c)...)
		errs = append(errs, emptyErrors(c, "payload")...)
	case "EMAIL_GROUP":
		addresses, _ := c["emailAddresses"].([]any)
		kept := []any{}
		seen := map[string]bool{}
		for _, a := range addresses {
			s, _ := a.(string)
			if _, err := mail.ParseAddress(s); err != nil {
				errs = append(errs, configError("data must be email address"))
			}
			if !seen[s] {
				seen[s] = true
				kept = append(kept, s)
			}
		}
		c["emailAddresses"] = kept
	case "JIRA":
		errs = append(errs, emptyErrors(c, "apiToken")...)
		if email, _ := c["email"].(string); email == "" {
			errs = append(errs, configError("data must be email address"))
		}
		errs = append(errs, emptyErrors(c, "projectKey")...)
	case "DEMISTO":
		errs = append(errs, uuidErrors(c)...)
	case "AWS_EVENT_BRIDGE":
		errs = append(errs, emptyErrors(c, "eventBusArn", "detail", "detailType", "source", "roleName")...)
	}
	if len(errs) > 0 {
		return webhookBadRequest(errs...)
	}
	return record, 0, nil
}

func configError(message string) string { return "Failed to validate webhook config: " + message }

func uuidErrors(c map[string]any) []string {
	uuid, _ := c["uuid"].(string)
	if webhookUUID.MatchString(uuid) {
		return nil
	}
	return []string{configError("data must be a valid UUID, was: " + uuid)}
}

func emptyErrors(c map[string]any, fields ...string) []string {
	var errs []string
	for _, field := range fields {
		if value, _ := c[field].(string); value == "" {
			errs = append(errs, configError("data must not be empty"))
		}
	}
	return errs
}

// lastOfEachType keeps the last item of each type, in the order of first appearance. The answer
// leaves out isActive when it is false.
func lastOfEachType(items []any) []any {
	var order []string
	last := map[string]map[string]any{}
	for _, item := range items {
		m, _ := item.(map[string]any)
		t, _ := m["type"].(string)
		if _, ok := last[t]; !ok {
			order = append(order, t)
		}
		last[t] = m
	}
	out := make([]any, 0, len(order))
	for _, t := range order {
		kept := map[string]any{"type": t}
		if active, _ := last[t]["isActive"].(bool); active {
			kept["isActive"] = true
		}
		out = append(out, kept)
	}
	return out
}

// webhookBadRequest is a 400 that lists each error, as the API does.
func webhookBadRequest(errs ...string) (*webhookRecord, int, any) {
	return nil, http.StatusBadRequest, map[string]any{"code": http.StatusBadRequest, "message": "Bad Request: Request errors:\n* " + strings.Join(errs, "\n* ")}
}

func roundTripMap(m map[string]any) map[string]any {
	raw, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func (w *webhookRecord) answer(id string) map[string]any {
	out := map[string]any{
		"id":         id,
		"type":       w.webhookType,
		"name":       w.name,
		"externalId": w.externalID,
		"createdAt":  w.createdAt,
		"updatedAt":  w.updatedAt,
	}
	if w.url != nil {
		out["url"] = *w.url
	}
	for _, arm := range webhookArms {
		if arm.webhookType == w.webhookType {
			out[arm.field] = roundTripMap(w.config)
		}
	}
	return out
}
