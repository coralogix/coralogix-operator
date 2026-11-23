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
)

// connectorFake is the connectors API, written from the SDK's openapi.yaml.
//
//	POST   /connectors        {connector: {name, type, description?, connectorConfig?, configOverrides?}} -> {connector}
//	PUT    /connectors        {connector: {id, ...}}  -> {connector}
//	GET    /connectors/{id}   -> {connector}
//	DELETE /connectors/{id}   -> {}
//
// The operator never calls the sibling routes (all/summaries, all/list, list/summaries,
// types/summaries, connector-schemas, connectors:test, connectors:testConfig). The fake answers
// them as a route that the API does not have.
//
// Nothing here is recorded yet: fidelity/recorded.json does not exist, so each status code, error
// text, and stored value is the simplest answer that the scenarios need. TestFakeMatchesRecordedAPI
// shows each difference once the recording lands.
type connectorFake struct {
	mu         sync.Mutex
	connectors map[string]map[string]any // id -> stored connector
	// next holds one forced answer per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]forcedAnswer
}

func newConnectorFake() apiFake {
	return &connectorFake{connectors: map[string]map[string]any{}, next: map[string]forcedAnswer{}}
}

func connectorAPI(h *harness) *connectorFake {
	h.t.Helper()
	f, ok := h.API().(*connectorFake)
	if !ok {
		h.t.Fatal("expected *connectorFake")
	}
	return f
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
	st, body := apiError(status, reason)
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

// seed stores a connector with the name, as if another client made it.
func (f *connectorFake) seed(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := newUUID()
	f.connectors[id] = connectorStored(map[string]any{"name": name, "type": "SLACK"}, id, nil)
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
		stored, ok := f.connectors[id]
		if !ok {
			return connectorNotFound(id)
		}
		return http.StatusOK, map[string]any{"connector": stored}
	case method == http.MethodDelete && id != "" && !strings.ContainsAny(id, "/:"):
		if _, ok := f.connectors[id]; !ok {
			return connectorNotFound(id)
		}
		delete(f.connectors, id)
		return http.StatusOK, map[string]any{}
	}
	// Includes the sibling routes: all/summaries, all/list, list/summaries, types/summaries,
	// connectors:test, and connectors:testConfig.
	return routeNotFound()
}

// connectorNotFound is the answer for an id that no connector has. The text starts with "Not
// Found:" and names the id, so the SDK reads it as a missing resource.
func connectorNotFound(id string) (int, any) {
	return apiError(http.StatusNotFound, fmt.Sprintf("connector with id %s does not exist", id))
}

func (f *connectorFake) create(in map[string]any) (int, any) {
	connector, ok := in["connector"].(map[string]any)
	if !ok {
		return apiError(http.StatusBadRequest, "connector is required")
	}
	if status, resp := f.validate(connector, ""); status != 0 {
		return status, resp
	}
	id := newUUID()
	f.connectors[id] = connectorStored(connector, id, nil)
	return http.StatusOK, map[string]any{"connector": f.connectors[id]}
}

func (f *connectorFake) replace(in map[string]any) (int, any) {
	connector, ok := in["connector"].(map[string]any)
	if !ok {
		return apiError(http.StatusBadRequest, "connector is required")
	}
	id, _ := connector["id"].(string)
	if id == "" {
		return apiError(http.StatusBadRequest, "connector id is required")
	}
	old, ok := f.connectors[id]
	if !ok {
		return connectorNotFound(id)
	}
	if status, resp := f.validate(connector, id); status != 0 {
		return status, resp
	}
	f.connectors[id] = connectorStored(connector, id, old)
	return http.StatusOK, map[string]any{"connector": f.connectors[id]}
}

var connectorTypes = map[string]bool{
	"SLACK": true, "GENERIC_HTTPS": true, "PAGERDUTY": true, "PAGERDUTY_INCIDENTS": true, "EMAIL": true,
	"SERVICE_NOW": true, "MICROSOFT_TEAMS": true, "EVENTBRIDGE": true, "INCIDENT_IO": true,
}

// validate checks a create or replace body. A non-zero status is an error answer. skipID is the id
// of the connector that the request replaces, which may keep its own name.
func (f *connectorFake) validate(connector map[string]any, skipID string) (int, any) {
	typ, _ := connector["type"].(string)
	if !connectorTypes[typ] {
		return apiError(http.StatusBadRequest, fmt.Sprintf("unsupported connector type %q", typ))
	}
	name, _ := connector["name"].(string)
	if name == "" {
		return apiError(http.StatusBadRequest, "connector name is required")
	}
	for id, stored := range f.connectors {
		if id != skipID && stored["name"] == name {
			return apiError(http.StatusConflict, fmt.Sprintf("connector with name %s already exists", name))
		}
	}
	return 0, nil
}

// connectorTime is the time of every stored connector. The recorder replaces a time with <time>.
const connectorTime = "2026-01-01T00:00:00Z"

// connectorStored is the connector that the API keeps for a Create or Replace body. old is the
// connector that a Replace replaces, or nil. It is a full replace: a field that the body omits is
// not kept. The server sets the id, the team, the times, and the resolved config; a value that the
// client sends for them is ignored.
func connectorStored(in map[string]any, id string, old map[string]any) map[string]any {
	stored := map[string]any{
		"id": id, "name": in["name"], "type": in["type"],
		"teamId": 1000000, "createTime": connectorTime, "updateTime": connectorTime,
	}
	if old != nil {
		stored["createTime"] = old["createTime"]
	}
	for _, key := range []string{"description", "connectorConfig", "configOverrides"} {
		if value, ok := in[key]; ok {
			stored[key] = value
		}
	}
	if config, ok := in["connectorConfig"]; ok {
		stored["resolvedConnectorConfig"] = config
	}
	return stored
}
