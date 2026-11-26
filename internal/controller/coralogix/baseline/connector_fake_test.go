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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
)

// connectorFake is the connectors API, written from the SDK's openapi.yaml.
//
//	POST   /connectors        {connector: {name, type, description?, connectorConfig, configOverrides?}} -> {connector}
//	PUT    /connectors        {connector: {id, ...}}  -> {connector}
//	GET    /connectors/{id}   -> {connector}
//	DELETE /connectors/{id}   -> {}
//
// The operator never calls the sibling routes (all/summaries, all/list, list/summaries,
// types/summaries, connector-schemas, connectors:test, connectors:testConfig). The fake answers
// them as a route that the API does not have.
//
// The status codes, error texts, and stored values follow fidelity/recorded.json, which the real
// API answered. Each rule names the recorded case that shows it. A rule with no case name is not
// recorded: it is the simplest answer that the scenarios need.
type connectorFake struct {
	mu         sync.Mutex
	connectors map[string]map[string]any // id -> stored connector
	// integrations are the ids of the integrations that a request can name in an integrationId field.
	// The recorded team has none, so the fake starts with none. A scenario that needs a working
	// PagerDuty Incidents, Microsoft Teams, or EventBridge connector, or a Slack connector that names
	// an integration, adds one.
	//
	// A Slack connector that names no integration still resolves one: the recording shows the API
	// adding an integrationId to resolvedConnectorConfig (a UUID that the recorder shows as <id>). The
	// API chooses that value, and the recording does not say from where. Whatever it is, a request
	// cannot name it: the recorded team has no integration that a request is allowed to name. The fake
	// makes up a UUID for it. It is not in the integrations set.
	integrations map[string]bool
	// serviceNowInstances are the instance URLs that resolve. The recorded team has none.
	serviceNowInstances map[string]bool
	// next holds one forced answer per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]forcedAnswer
}

func newConnectorFake() apiFake {
	return &connectorFake{
		connectors:          map[string]map[string]any{},
		integrations:        map[string]bool{},
		serviceNowInstances: map[string]bool{},
		next:                map[string]forcedAnswer{},
	}
}

func connectorAPI(h *harness) *connectorFake {
	h.t.Helper()
	f, ok := h.API().(*connectorFake)
	if !ok {
		h.t.Fatal("expected *connectorFake")
	}
	return f
}

// addIntegration makes the team have the integration.
func (f *connectorFake) addIntegration(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.integrations[id] = true
}

// addServiceNowInstance makes the instance URL resolve.
func (f *connectorFake) addServiceNowInstance(instanceURL string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.serviceNowInstances[instanceURL] = true
}

// answerNext makes the next request with the method get a 200 answer with this body.
func (f *connectorFake) answerNext(method string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = forcedAnswer{status: http.StatusOK, body: body}
}

// answerNextError makes the next request with the method get an error answer.
func (f *connectorFake) answerNextError(method string, status int, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := connectorReason(status, reason)
	f.next[method] = forcedAnswer{status: st, body: body}
}

// answerNextRouteNotFound makes the next request with the method get the answer for a route that
// the API does not have.
func (f *connectorFake) answerNextRouteNotFound(method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := routeNotFound()
	f.next[method] = forcedAnswer{status: st, body: body}
}

// seed stores a Slack connector with the name, as if another client made it.
func (f *connectorFake) seed(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := newUUID()
	f.connectors[id] = map[string]any{"id": id, "name": name, "type": "SLACK"}
}

// forget deletes a connector without a request, as if it was deleted outside the operator.
func (f *connectorFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.connectors, id)
}

func (f *connectorFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if forced, ok := f.next[method]; ok {
		delete(f.next, method)
		return forced.status, forced.body
	}
	rest, ok := strings.CutPrefix(path, connectorPath)
	if !ok {
		return routeNotFound()
	}
	id := strings.TrimPrefix(rest, "/")
	in, _ := body.(map[string]any)

	switch {
	case method == http.MethodPost && rest == "":
		return f.create(in)
	case method == http.MethodPut && rest == "":
		return f.replace(in)
	case method == http.MethodGet && id != "" && !strings.ContainsAny(id, "/:"):
		// "unknown id": a well-formed id that no connector has is a 404. "malformed id ...": any
		// other id is a 400.
		if !connectorIDPattern.MatchString(id) {
			return connectorBadRequest(fmt.Sprintf("'%s' is not a valid identifier", id))
		}
		stored, ok := f.connectors[id]
		if !ok {
			return connectorNotFound(id)
		}
		return http.StatusOK, map[string]any{"connector": stored}
	case method == http.MethodDelete && id != "" && !strings.ContainsAny(id, "/:"):
		// "unknown id" and "delete twice": a delete of an id that no connector has is a 200.
		if !connectorIDPattern.MatchString(id) {
			return connectorBadRequest(fmt.Sprintf("'%s' is not a valid identifier", id))
		}
		delete(f.connectors, id)
		return http.StatusOK, map[string]any{}
	}
	// Includes the sibling routes: all/summaries, all/list, list/summaries, types/summaries,
	// connectors:test, and connectors:testConfig.
	return routeNotFound()
}

var connectorIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// connectorReason is an error answer of the connectors API: the text is a JSON object with a
// "reason", as in {"code": 400, "message": "Bad Request: {\"reason\":\"name must not be empty\"}"}.
func connectorReason(status int, reason string) (int, any) {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string]string{"reason": reason}); err != nil {
		panic(err)
	}
	return apiError(status, strings.TrimSpace(out.String()))
}

func connectorBadRequest(reason string) (int, any) {
	return connectorReason(http.StatusBadRequest, reason)
}

// connectorNotFound is the answer for an id that no connector has. The text starts with "Not
// Found:" and names the id, so the SDK reads it as a missing resource.
func connectorNotFound(id string) (int, any) {
	return connectorReason(http.StatusNotFound, fmt.Sprintf("Connector with id %s not found", id))
}

func (f *connectorFake) create(in map[string]any) (int, any) {
	if status, resp := connectorProtoError(in); status != 0 {
		return status, resp
	}
	connector, ok := in["connector"].(map[string]any)
	if !ok {
		return connectorBadRequest("Connector input is required")
	}
	resolved, status, resp := f.validate(connector)
	if status != 0 {
		return status, resp
	}
	if f.nameTaken(connector, "") {
		return connectorBadRequest(fmt.Sprintf("Connector with name %s already exists", connector["name"]))
	}
	id := newUUID()
	f.connectors[id] = connectorStored(connector, id, nil, resolved)
	return http.StatusOK, map[string]any{"connector": f.connectors[id]}
}

func (f *connectorFake) replace(in map[string]any) (int, any) {
	if status, resp := connectorProtoError(in); status != 0 {
		return status, resp
	}
	connector, ok := in["connector"].(map[string]any)
	if !ok {
		return connectorBadRequest("Connector data is required")
	}
	// "replace without an id": the API looks for an id of its own making, and does not find it.
	id, hasID := connector["id"].(string)
	switch {
	case !hasID:
		id = newUUID()
	case id == "":
		return connectorBadRequest("id must not be empty")
	case !connectorIDPattern.MatchString(id):
		return connectorBadRequest(fmt.Sprintf("Failed to validate id: '%s' does not match regex %s", id, connectorIDPattern))
	}
	old, ok := f.connectors[id]
	if !ok {
		return connectorNotFound(id)
	}
	resolved, status, resp := f.validate(connector)
	if status != 0 {
		return status, resp
	}
	// "replace changes the type and the fields": the fields are checked first, then the type.
	if old["type"] != connector["type"] {
		return connectorBadRequest("Changing connector type is not allowed. Original: " + connectorSchemas[old["type"].(string)].display)
	}
	if f.nameTaken(connector, id) {
		return connectorBadRequest(fmt.Sprintf("Connector with name %s already exists", connector["name"]))
	}
	f.connectors[id] = connectorStored(connector, id, old, resolved)
	return http.StatusOK, map[string]any{"connector": f.connectors[id]}
}

// nameTaken follows "duplicate name: ...": a name is unique within a type, and it is case
// sensitive. skipID is the connector that the request replaces.
func (f *connectorFake) nameTaken(connector map[string]any, skipID string) bool {
	for id, stored := range f.connectors {
		if id != skipID && stored["type"] == connector["type"] && stored["name"] == connector["name"] {
			return true
		}
	}
	return false
}

// connectorKeys are the properties of a connector that the API knows.
var connectorKeys = []string{
	"configOverrides", "connectorConfig", "createTime", "description", "diagnostics", "id", "name",
	"resolvedConnectorConfig", "teamId", "type", "updateTime",
}

// connectorEnumTypes are the type names of the API enum, with the number of each. The numbers of
// the types that no recorded case rejects are in the order of the enum.
var connectorEnumTypes = map[string]int{
	"CONNECTOR_TYPE_UNSPECIFIED": 0, "SLACK": 1, "GENERIC_HTTPS": 2, "PAGERDUTY": 3, "IBM_EVENT_NOTIFICATIONS": 5,
	"SERVICE_NOW": 6, "EMAIL": 7, "PAGERDUTY_INCIDENTS": 8, "MICROSOFT_TEAMS": 9, "EVENTBRIDGE": 10, "INCIDENT_IO": 11,
}

func jsonQuoted(s string) string {
	raw, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// connectorProtoError follows "rejection: a type that the enum does not have", "a type in lower
// case", and "a connector with a property that the API does not know". The gateway decodes the
// JSON body first. Its error names the column of the bad token in the body it received, which is
// compact JSON with the keys in order.
func connectorProtoError(in map[string]any) (int, any) {
	connector, _ := in["connector"].(map[string]any)
	raw, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	text := string(raw)
	keys := make([]string, 0, len(connector))
	for key := range connector {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !slices.Contains(connectorKeys, key) {
			quoted := jsonQuoted(key)
			return apiError(http.StatusBadRequest, fmt.Sprintf("proto: (line 1:%d): unknown field %s", strings.Index(text, quoted+":")+1, quoted))
		}
	}
	if typ, ok := connector["type"].(string); ok {
		if _, ok := connectorEnumTypes[typ]; !ok {
			quoted := jsonQuoted(typ)
			column := strings.Index(text, `"type":`+quoted) + len(`"type":`) + 1
			return apiError(http.StatusBadRequest, fmt.Sprintf("proto: (line 1:%d): invalid value for enum field type: %s", column, quoted))
		}
	}
	return 0, nil
}

// fieldKind says how the API checks the value of a connector config field.
type fieldKind int

const (
	fieldText fieldKind = iota
	fieldNonEmpty
	fieldURL
	fieldMethod
	fieldJSONObject
	fieldEmails
	// fieldIntegration is the id of an integration of the team. Slack takes the one of the team
	// when the field is not set.
	fieldIntegration
)

// connectorSchema is the output schema that the API checks the config of a type against. The
// schemas of Slack, generic HTTPS, email, and incident.io, and the name "Slack", are recorded
// ("rejection: ...", "replace changes the type and the fields"). The rest is the simplest value
// that fits the samples.
type connectorSchema struct {
	id       string
	display  string
	fields   map[string]fieldKind
	required []string
	// templated are the fields that a config override can set.
	templated []string
}

var connectorSchemas = map[string]connectorSchema{
	"SLACK": {
		id: "slack_structured", display: "Slack",
		fields:    map[string]fieldKind{"channel": fieldNonEmpty, "fallbackChannel": fieldText, "integrationId": fieldIntegration},
		required:  []string{"channel"},
		templated: []string{"channel"},
	},
	"GENERIC_HTTPS": {
		id: "generic_https_default", display: "Generic HTTPS",
		fields: map[string]fieldKind{
			"url": fieldURL, "method": fieldMethod, "additionalHeaders": fieldJSONObject, "additionalBodyFields": fieldText,
		},
		required:  []string{"url"},
		templated: []string{"url", "additionalHeaders", "additionalBodyFields"},
	},
	"PAGERDUTY": {
		id: "pager_duty_default", display: "PagerDuty",
		fields:    map[string]fieldKind{"integrationKey": fieldNonEmpty},
		required:  []string{"integrationKey"},
		templated: []string{"integrationKey"},
	},
	"PAGERDUTY_INCIDENTS": {
		id: "pager_duty_incidents_default", display: "PagerDuty Incidents",
		fields:   map[string]fieldKind{"integrationId": fieldIntegration, "service": fieldText},
		required: []string{"integrationId", "service"},
	},
	"EMAIL": {
		id: "email_default", display: "Email",
		fields:    map[string]fieldKind{"emailAddresses": fieldEmails},
		required:  []string{"emailAddresses"},
		templated: []string{"emailAddresses"},
	},
	"SERVICE_NOW": {
		id: "service_now_default", display: "ServiceNow",
		fields:   map[string]fieldKind{"instanceUrl": fieldURL, "username": fieldText, "password": fieldText},
		required: []string{"instanceUrl", "username", "password"},
	},
	"MICROSOFT_TEAMS": {
		id: "microsoft_teams_default", display: "Microsoft Teams",
		fields:   map[string]fieldKind{"integrationId": fieldIntegration, "teamId": fieldText, "channelId": fieldText},
		required: []string{"integrationId", "teamId", "channelId"},
	},
	"EVENTBRIDGE": {
		id: "eventbridge_default", display: "EventBridge",
		fields:   map[string]fieldKind{"integrationId": fieldIntegration, "additionalDetail": fieldText},
		required: []string{"integrationId"},
	},
	"INCIDENT_IO": {
		id: "incident_io_default", display: "incident.io",
		fields:   map[string]fieldKind{"apiKey": fieldNonEmpty},
		required: []string{"apiKey"},
	},
}

// connectorEntityTypes are the entity types that accept config overrides, by API name.
var connectorEntityTypes = map[string]string{"ALERTS": "alerts", "CASES": "cases"}

// validate checks a create or replace body, in the order of the recorded answers. A non-zero status
// is an error answer. On success it returns the resolved config.
func (f *connectorFake) validate(connector map[string]any) (resolved []any, status int, resp any) {
	fail := func(reason string) ([]any, int, any) {
		st, body := connectorBadRequest(reason)
		return nil, st, body
	}
	// "rejection: an empty name", "a connector without a name".
	if name, _ := connector["name"].(string); name == "" {
		return fail("name must not be empty")
	}
	// "rejection: type CONNECTOR_TYPE_UNSPECIFIED", "type IBM_EVENT_NOTIFICATIONS", "a connector
	// without a type": the enum value must be one that the API can make a connector of.
	typ, _ := connector["type"].(string)
	schema, ok := connectorSchemas[typ]
	if !ok {
		return fail(fmt.Sprintf("unknown enumeration value %d", connectorEnumTypes[typ]))
	}
	// "rejection: a connector with a name and a type only, no connectorConfig": the config is
	// required. An empty one passes this check and fails the required fields of the schema below
	// ("rejection: an empty connectorConfig"): every schema has a required field.
	config, ok := connector["connectorConfig"].(map[string]any)
	if !ok {
		return fail("connector_config is required")
	}

	values := map[string]string{}
	var order []string
	list, _ := config["fields"].([]any)
	for _, raw := range list {
		field, _ := raw.(map[string]any)
		name, _ := field["fieldName"].(string)
		value, _ := field["value"].(string)
		// "create with the same field name twice": the last one wins.
		if _, seen := values[name]; !seen {
			order = append(order, name)
		}
		values[name] = value
	}
	prefix := fmt.Sprintf("Error while validating config for output schema with id '%s': ", schema.id)
	// "replace changes the type and keeps the fields of the old type": an unknown field comes
	// before a missing required one.
	for _, name := range order {
		if _, ok := schema.fields[name]; !ok {
			return fail(prefix + "Unknown connector config field: " + name)
		}
	}
	for _, name := range schema.required {
		if _, ok := values[name]; !ok {
			return fail(prefix + fmt.Sprintf("Required field '%s' is missing in connector config", name))
		}
	}
	for _, name := range order {
		if reason := checkFieldValue(name, schema.fields[name], values[name]); reason != "" {
			return fail(prefix + reason)
		}
	}
	if _, ok := schema.fields["emailAddresses"]; ok {
		if reason := checkEmails(values["emailAddresses"]); reason != "" {
			return fail(fmt.Sprintf("Error while validating email addresses for output schema with id '%s': %s", schema.id, reason))
		}
	}
	for _, name := range order {
		if schema.fields[name] == fieldIntegration && !f.integrations[values[name]] {
			return fail(fmt.Sprintf("Integration '%s' not found or not authorized for this team", values[name]))
		}
	}
	if instance, ok := values["instanceUrl"]; ok && !f.serviceNowInstances[instance] {
		return fail("Failed to resolve ServiceNow instance url")
	}

	if reason := checkOverrides(connector["configOverrides"], schema); reason != "" {
		return fail(reason)
	}

	// "create a slack connector": the API adds the integration of the team to the resolved config.
	names := slices.Clone(order)
	if typ == "SLACK" {
		if _, set := values["integrationId"]; !set {
			names = append(names, "integrationId")
			values["integrationId"] = newUUID()
		}
	}
	sort.Strings(names)
	resolved = []any{}
	for _, name := range names {
		resolved = append(resolved, map[string]any{"fieldName": name, "value": values[name]})
	}
	return resolved, 0, nil
}

// checkFieldValue returns the reason that the value is not one for the field, or "".
func checkFieldValue(name string, kind fieldKind, value string) string {
	switch kind {
	case fieldText, fieldIntegration:
		// Any value passes. An integration is looked up apart from the syntax.
	case fieldNonEmpty:
		if value == "" {
			return fmt.Sprintf("Failed to parse field %s into type NonEmptyString: Template must be a non-empty string", name)
		}
	case fieldURL:
		if u, err := url.Parse(value); err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || strings.Contains(value, " ") {
			return fmt.Sprintf("Failed to parse field %s into type Url: url %s is not a valid URL", name, value)
		}
	case fieldMethod:
		if !slices.Contains([]string{"POST", "GET", "PUT"}, value) {
			return fmt.Sprintf(`Field '%s' has invalid value. Allowed values: ["POST", "GET", "PUT"]`, name)
		}
	case fieldJSONObject:
		var object map[string]string
		if err := json.Unmarshal([]byte(value), &object); err != nil {
			return fmt.Sprintf("Failed to parse field %s into type KeyValuePairs: Template must be a valid JSON object with top-level string fields only", name)
		}
	case fieldEmails:
		var emails []string
		if err := json.Unmarshal([]byte(value), &emails); err != nil {
			return fmt.Sprintf("Failed to parse field %s into type ArrayOfStrings: Template must be an array of strings", name)
		}
	}
	return ""
}

// checkEmails returns the reason that an address of the JSON list is not one, or "".
func checkEmails(value string) string {
	var emails []string
	_ = json.Unmarshal([]byte(value), &emails)
	var invalid []string
	for _, email := range emails {
		if _, err := mail.ParseAddress(email); err != nil {
			invalid = append(invalid, email)
		}
	}
	if len(invalid) > 0 {
		return "The following emails are invalid: " + strings.Join(invalid, ", ")
	}
	return ""
}

var connectorTemplateTag = regexp.MustCompile(`\{%\s*(\w+)\s*%\}`)

// checkOverrides follows the "rejection: override ..." cases. It returns the reason that a config
// override is not valid, or "".
func checkOverrides(raw any, schema connectorSchema) string {
	overrides, _ := raw.([]any)
	for _, o := range overrides {
		override, _ := o.(map[string]any)
		entity, _ := override["entityType"].(string)
		switch entity {
		case "TEST_NOTIFICATIONS":
			return "Config overrides for entity type 'testNotifications' are not supported."
		case "OLLY_SCHEDULED_TASKS":
			return "Config overrides for entity type 'ollyScheduledTasks' are not supported."
		}
		name, ok := connectorEntityTypes[entity]
		if !ok {
			return "Valid entity_type must be provided"
		}
		prefix := fmt.Sprintf("Error while validating connector config overrides for entity type: '%s', output schema with id '%s': ", name, schema.id)
		fields, _ := override["fields"].([]any)
		for _, f := range fields {
			field, _ := f.(map[string]any)
			fieldName, _ := field["fieldName"].(string)
			if _, ok := schema.fields[fieldName]; !ok {
				return prefix + "Unknown connector config field: " + fieldName
			}
			if !slices.Contains(schema.templated, fieldName) {
				return prefix + "Connector config override field is not templated: " + fieldName
			}
			// A control tag with no condition, such as {% if %}, does not parse.
			template, _ := field["template"].(string)
			for _, tag := range connectorTemplateTag.FindAllStringSubmatch(template, -1) {
				if tag[1] == "if" || tag[1] == "elif" {
					return fmt.Sprintf("Failed to validate connector config templates for entity_type '%s' and field '%s': invalid template syntax", name, fieldName)
				}
			}
		}
	}
	return ""
}

// connectorTime is the time of every stored connector. The recorder replaces a time with <time>.
const connectorTime = "2026-01-01T00:00:00Z"

// connectorStored is the connector that the API keeps for a Create or Replace body. old is the
// connector that a Replace replaces, or nil. It is a full replace: a field that the body omits is
// cleared ("replace without a description"). The server sets the id, the team, the times, and the
// resolved config; a value that the client sends for them is ignored ("server fields").
func connectorStored(in map[string]any, id string, old map[string]any, resolved []any) map[string]any {
	stored := map[string]any{
		"id": id, "name": in["name"], "type": in["type"],
		"teamId": 1000000, "createTime": connectorTime, "updateTime": connectorTime,
		// "create without a description": an omitted description is "".
		"description":             "",
		"resolvedConnectorConfig": map[string]any{"fields": resolved},
	}
	if old != nil {
		stored["createTime"] = old["createTime"]
	}
	if description, ok := in["description"].(string); ok {
		stored["description"] = description
	}
	stored["connectorConfig"] = connectorConfigAnswer(in["connectorConfig"])
	stored["configOverrides"] = connectorOverridesAnswer(in["configOverrides"])
	return stored
}

// connectorConfigAnswer follows "create with fields in reverse order": the API sorts the fields by
// name, and keeps the last of two fields with one name.
func connectorConfigAnswer(raw any) map[string]any {
	config, _ := raw.(map[string]any)
	list, _ := config["fields"].([]any)
	byName := map[string]any{}
	for _, f := range list {
		field, _ := f.(map[string]any)
		name, _ := field["fieldName"].(string)
		value, _ := field["value"].(string)
		byName[name] = map[string]any{"fieldName": name, "value": value}
	}
	return map[string]any{"fields": sortedFields(byName)}
}

// connectorOverridesAnswer follows "create without overrides" and "create with an override that
// has no fields": no overrides is [], an override with no fields has [], and the fields of an
// override are sorted by name. The overrides keep their order.
func connectorOverridesAnswer(raw any) []any {
	out := []any{}
	list, _ := raw.([]any)
	for _, o := range list {
		override, _ := o.(map[string]any)
		byName := map[string]any{}
		fields, _ := override["fields"].([]any)
		for _, f := range fields {
			field, _ := f.(map[string]any)
			name, _ := field["fieldName"].(string)
			template, _ := field["template"].(string)
			byName[name] = map[string]any{"fieldName": name, "template": template}
		}
		out = append(out, map[string]any{"entityType": override["entityType"], "fields": sortedFields(byName)})
	}
	return out
}

func sortedFields(byName map[string]any) []any {
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, byName[name])
	}
	return out
}
