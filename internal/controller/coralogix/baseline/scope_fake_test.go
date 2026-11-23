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
	"strings"
	"sync"
)

const scopePath = "/aaa/team-scopes/v1"

type scopeFilter struct {
	entityType string
	expression string
}

type scopeRecord struct {
	displayName       string
	description       *string
	filters           []scopeFilter
	defaultExpression string
}

// scopeFake is the team scopes API, written from the SDK's openapi.yaml.
//
//	POST   /team-scopes/v1           {displayName, description, filters, defaultExpression} -> {scope}
//	PUT    /team-scopes/v1           {id, displayName, description, filters, defaultExpression} -> {scope}
//	GET    /team-scopes/v1?ids=<id>  -> {scopes: [...]}
//	DELETE /team-scopes/v1/{id}      -> {}
//
// The status codes and error texts follow fidelity/recorded.json, which the real API answered:
//   - Replace clears an omitted displayName, description, or filters.
//   - Get answers 404 when no id matches, also for an id that is not a UUID.
//   - Update of a missing scope answers 404. Delete of a missing scope answers 200.
//   - An unspecified entity type is a 400. An expression without a version tag, or a default
//     expression other than true or false, is a 412.
//   - The fake accepts <v1>false as the contract says. The real API rejected it with the
//     "true or false" 412 in most attempts during recording, so no case sends it.
//   - Display names need not be unique.
type scopeFake struct {
	mu     sync.Mutex
	scopes map[string]scopeRecord
	forced map[string]forcedAnswer
}

func newScopeFake() apiFake {
	return &scopeFake{scopes: map[string]scopeRecord{}, forced: map[string]forcedAnswer{}}
}

// answerNext makes the next request with the method get this answer instead of the fake's own.
func (f *scopeFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

// forget deletes a scope without a request, as if it was deleted outside the operator.
func (f *scopeFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.scopes, id)
}

// put stores a scope without a request, as if another client created it.
func (f *scopeFake) put(id string, scope scopeRecord) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scopes[id] = scope
}

func (f *scopeFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.scopes))
	for id, scope := range f.scopes {
		out[id] = scope.answer(id)
	}
	return out
}

func (f *scopeFake) handle(method, path string, query url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	in, _ := body.(map[string]any)

	if path == scopePath {
		switch method {
		case http.MethodPost:
			scope, status, errBody := scopeFromRequest(in)
			if status != 0 {
				return status, errBody
			}
			id := newUUID()
			f.scopes[id] = scope
			return http.StatusOK, map[string]any{"scope": scope.answer(id)}

		case http.MethodPut:
			id, _ := in["id"].(string)
			scope, status, errBody := scopeFromRequest(in)
			if status != 0 {
				return status, errBody
			}
			if _, ok := f.scopes[id]; !ok {
				return apiError(http.StatusNotFound, "Scope not found")
			}
			f.scopes[id] = scope
			return http.StatusOK, map[string]any{"scope": scope.answer(id)}

		case http.MethodGet:
			found := []any{}
			for _, id := range query["ids"] {
				if scope, ok := f.scopes[id]; ok {
					found = append(found, scope.answer(id))
				}
			}
			if len(found) == 0 {
				return apiError(http.StatusNotFound, "No scopes found")
			}
			return http.StatusOK, map[string]any{"scopes": found}
		}
		return routeNotFound()
	}

	id, ok := strings.CutPrefix(path, scopePath+"/")
	if ok && id != "" && !strings.Contains(id, "/") && method == http.MethodDelete {
		delete(f.scopes, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

// scopeFromRequest reads the scope of a create or replace body. A non-zero status is an error
// answer.
func scopeFromRequest(in map[string]any) (scopeRecord, int, any) {
	var scope scopeRecord
	scope.displayName, _ = in["displayName"].(string)
	if description, ok := in["description"].(string); ok {
		scope.description = &description
	}
	items, _ := in["filters"].([]any)
	scope.filters = []scopeFilter{}
	for _, item := range items {
		filter, _ := item.(map[string]any)
		entityType, _ := filter["entityType"].(string)
		expression, _ := filter["expression"].(string)
		if entityType == "" || entityType == "ENTITY_TYPE_UNSPECIFIED" {
			status, body := apiError(http.StatusBadRequest, "Incoming entity type is unspecified")
			return scopeRecord{}, status, body
		}
		if !strings.HasPrefix(expression, "<v1>") {
			status, body := scopeExpressionError(expression)
			return scopeRecord{}, status, body
		}
		scope.filters = append(scope.filters, scopeFilter{entityType: entityType, expression: expression})
	}
	scope.defaultExpression, _ = in["defaultExpression"].(string)
	value, tagged := strings.CutPrefix(scope.defaultExpression, "<v1>")
	if !tagged {
		status, body := scopeExpressionError(scope.defaultExpression)
		return scopeRecord{}, status, body
	}
	if value = strings.TrimSpace(value); value != "true" && value != "false" {
		status, body := scopePrecondition("The default expression must be either 'true' or 'false")
		return scopeRecord{}, status, body
	}
	return scope, 0, nil
}

func scopeExpressionError(expression string) (int, any) {
	return scopePrecondition(fmt.Sprintf(
		`Failed to compile expression. Expression: [%s], Errors: [expected version tag. e.g. "<v1> foo == 'bar'"]`, expression))
}

// scopePrecondition is a 412. The gateway names it with the gRPC code, not the HTTP status text.
func scopePrecondition(reason string) (int, any) {
	return http.StatusPreconditionFailed, map[string]any{"code": http.StatusPreconditionFailed, "message": "Failed Precondition: " + reason}
}

func (s scopeRecord) answer(id string) map[string]any {
	filters := make([]any, 0, len(s.filters))
	for _, filter := range s.filters {
		filters = append(filters, map[string]any{"entityType": filter.entityType, "expression": filter.expression})
	}
	out := map[string]any{
		"id":                id,
		"displayName":       s.displayName,
		"teamId":            1,
		"filters":           filters,
		"defaultExpression": s.defaultExpression,
	}
	if s.description != nil {
		out["description"] = *s.description
	}
	return out
}
