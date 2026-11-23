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
	"strconv"
	"strings"
	"sync"
)

const customRolePath = "/aaa/custom-roles/v1"

type roleRecord struct {
	name        string
	description string
	parent      string
	permissions []string
}

// customRoleFake keeps the remote state that the handwritten controller can change.
type customRoleFake struct {
	mu     sync.Mutex
	nextID int64
	roles  map[int64]roleRecord
	forced map[string]forcedAnswer
}

func newCustomRoleFake() apiFake {
	return &customRoleFake{nextID: 1001, roles: map[int64]roleRecord{}, forced: map[string]forcedAnswer{}}
}

func (f *customRoleFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

func (f *customRoleFake) remove(id int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.roles, id)
}

func (f *customRoleFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.roles))
	for id, role := range f.roles {
		out[strconv.FormatInt(id, 10)] = role.answer(id)
	}
	return out
}

func (f *customRoleFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	if path == customRolePath {
		switch method {
		case http.MethodPut:
			return f.create(body)
		}
	}
	idText, ok := strings.CutPrefix(path, customRolePath+"/")
	if !ok || strings.Contains(idText, "/") {
		return routeNotFound()
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id < 0 {
		return apiError(http.StatusBadRequest, "invalid role id")
	}
	role, exists := f.roles[id]
	if !exists {
		return apiError(http.StatusNotFound, "Role not found")
	}
	switch method {
	case http.MethodGet:
		return http.StatusOK, map[string]any{"role": role.answer(id)}
	case http.MethodPost:
		return f.update(id, role, body)
	case http.MethodDelete:
		delete(f.roles, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

func (f *customRoleFake) create(body any) (int, any) {
	in, _ := body.(map[string]any)
	name, _ := in["name"].(string)
	parent, _ := in["parentRoleName"].(string)
	if parent == "" {
		return apiError(http.StatusBadRequest, "Parent role is required")
	}
	if name == "" {
		return apiError(http.StatusBadRequest, "Role name is required")
	}
	if parent != "Standard User" {
		return apiError(http.StatusBadRequest, "parent role not found")
	}
	for _, role := range f.roles {
		if role.name == name {
			return http.StatusConflict, map[string]any{
				"code":    409,
				"message": fmt.Sprintf("Already Exists: Could not create custom role %s for team 1 as it already exists.", name),
			}
		}
	}
	id := f.nextID
	f.nextID++
	description, _ := in["description"].(string)
	f.roles[id] = roleRecord{name: name, description: description, parent: parent, permissions: rolePermissions(in["permissions"])}
	return http.StatusOK, map[string]any{"id": id}
}

func (f *customRoleFake) update(id int64, role roleRecord, body any) (int, any) {
	in, _ := body.(map[string]any)
	if name, ok := in["newName"].(string); ok {
		for otherID, other := range f.roles {
			if otherID != id && other.name == name {
				return apiError(http.StatusConflict, "role already exists")
			}
		}
		role.name = name
	}
	if description, ok := in["newDescription"].(string); ok {
		role.description = description
	}
	if permissions, ok := in["newPermissions"].(map[string]any); ok {
		role.permissions = rolePermissions(permissions["permissions"])
	}
	f.roles[id] = role
	return http.StatusOK, map[string]any{}
}

func (r roleRecord) answer(id int64) map[string]any {
	out := map[string]any{
		"roleId":         id,
		"teamId":         1,
		"name":           r.name,
		"description":    r.description,
		"parentRoleId":   1,
		"parentRoleName": r.parent,
		"permissions":    r.permissions,
	}
	return out
}

func rolePermissions(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if permission, ok := item.(string); ok {
			out = append(out, permission)
		}
	}
	return out
}
