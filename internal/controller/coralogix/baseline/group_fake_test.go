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

const (
	groupPath = "/aaa/team-groups/v2"
	usersPath = "/aaa/users/v2"
	// groupFakeTeam is the team number in the error texts of the fake. The recording redacts it.
	groupFakeTeam = 4000001
	// A recorded answer has the same timestamp in every field. The recording redacts it.
	groupFakeTime = "2026-01-01T00:00:00Z"
)

// A role the fake knows, as the API shows it in the role of a group.
type groupRole struct {
	name        string
	description string
}

type groupUser struct {
	id       string
	username string
}

type groupRecord struct {
	name        string
	description *string
	groupType   string
	roleID      *int64
	scopeID     *string
	userIDs     []string
}

// groupFake is the team groups API and the users search API, written from the SDK's openapi.yaml.
//
//	POST   /team-groups/v2              {name, description, groupType, roleId, scope, userIds} -> {group}
//	GET    /team-groups/v2/id/{id}      -> {group}
//	PUT    /team-groups/v2/{id}         {name, description, groupType, roleUpdate, scopeUpdate, userUpdates} -> {group}
//	DELETE /team-groups/v2/{id}         -> {groupId}
//	GET    /users/v2?page_size&page_token -> {users, nextPageToken, totalCount}
//
// The status codes and error texts follow fidelity/recorded.json, which the real API answered:
//   - Create and Update answer the group without its users: userIds is always empty there. Get
//     answers the users.
//   - A group name is unique in the team: Create answers 409. Update to a taken name answers 500.
//     An empty name is a 400.
//   - An omitted groupType is the open type, and an unspecified one on Create or Update is
//     open too. Update keeps the stored type when the field is omitted.
//   - Update clears an omitted description. It keeps the name, the role, the scope, and the
//     users when the body omits them. The name is kept when the field is omitted.
//   - roleUpdate set_role_id with the id 0, no id, or an id that is not a role of the team is a
//     403. A clear action removes the role.
//   - scopeUpdate set_scope_id with no value clears the scope. An unknown scope is a 404.
//   - A user that is not in the team is a 403. Duplicate user ids are stored once.
//   - Update, Get, and Delete of a group that does not exist answer 404.
//   - The users search answers no users when page_size is omitted. nextPageToken is the offset of
//     the next page, and it is omitted on the last page.
type groupFake struct {
	mu     sync.Mutex
	nextID int64
	groups map[int64]groupRecord
	users  []groupUser
	roles  map[int64]groupRole
	scopes map[string]bool
	forced map[string]forcedAnswer
}

func newGroupFake() apiFake {
	f := &groupFake{
		nextID: 9001,
		groups: map[int64]groupRecord{},
		roles:  map[int64]groupRole{},
		scopes: map[string]bool{},
		forced: map[string]forcedAnswer{},
	}
	// The fidelity cases name these values as {{userA}}, {{userB}}, {{scopeId}}, and {{roleId}}.
	f.users = []groupUser{
		{id: groupFakeUserA, username: "member-a@example.com"},
		{id: groupFakeUserB, username: "member-b@example.com"},
	}
	f.roles[groupFakeRole] = groupRole{name: "cxo-fidelity-group-role", description: "Fidelity role"}
	f.scopes[groupFakeScope] = true
	return f
}

// The values that the fidelity cases name. They are not UUIDs with the fixed test prefix, so the
// recording and the fake both write <id> for them.
const (
	groupFakeUserA = "3a1f0e55-8c1b-4f0c-9d0e-1a2b3c4d5e01"
	groupFakeUserB = "3a1f0e55-8c1b-4f0c-9d0e-1a2b3c4d5e02"
	groupFakeScope = "3a1f0e55-8c1b-4f0c-9d0e-1a2b3c4d5e03"
	groupFakeRole  = int64(7001)
)

// answerNext makes the next request with the method get this answer instead of the fake's own.
func (f *groupFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

// forget deletes a group without a request, as if it was deleted outside the operator.
func (f *groupFake) forget(id int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.groups, id)
}

// put stores a group without a request, as if another client created it.
func (f *groupFake) put(id int64, group groupRecord) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.groups[id] = group
	if id >= f.nextID {
		f.nextID = id + 1
	}
}

// setUsers replaces the users of the team.
func (f *groupFake) setUsers(users ...groupUser) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users = users
}

func (f *groupFake) addRole(id int64, role groupRole) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.roles[id] = role
}

func (f *groupFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.groups))
	for id, group := range f.groups {
		out[strconv.FormatInt(id, 10)] = f.answer(id, group, true)
	}
	return out
}

func (f *groupFake) handle(method, path string, query url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	in, _ := body.(map[string]any)

	if path == usersPath && method == http.MethodGet {
		return http.StatusOK, f.searchUsers(query)
	}
	if path == groupPath && method == http.MethodPost {
		return f.create(in)
	}
	if idText, ok := strings.CutPrefix(path, groupPath+"/id/"); ok && method == http.MethodGet {
		return f.get(idText)
	}
	idText, ok := strings.CutPrefix(path, groupPath+"/")
	if !ok || idText == "" || strings.Contains(idText, "/") {
		return routeNotFound()
	}
	switch method {
	case http.MethodPut:
		return f.update(idText, in)
	case http.MethodDelete:
		id, _, status, errBody := f.find(idText)
		if status != 0 {
			return status, errBody
		}
		delete(f.groups, id)
		return http.StatusOK, map[string]any{"groupId": id}
	}
	return routeNotFound()
}

// searchUsers answers one page. The real API answers no users when page_size is omitted.
func (f *groupFake) searchUsers(query url.Values) any {
	size, _ := strconv.Atoi(query.Get("page_size"))
	offset, _ := strconv.Atoi(query.Get("page_token"))
	users := []any{}
	if size > 0 && offset < len(f.users) {
		end := min(offset+size, len(f.users))
		for _, user := range f.users[offset:end] {
			users = append(users, map[string]any{
				"userId": user.id, "username": user.username, "status": "USER_STATUS_ACTIVE",
				"firstName": "", "lastName": "", "userAccountId": 1,
				"accessType": map[string]any{"accessType": "permanent", "permanentAccess": map[string]any{}},
				"groupIds":   []any{}, "allowedLoginMode": []any{},
			})
		}
		out := map[string]any{"users": users, "totalCount": len(f.users)}
		if end < len(f.users) {
			out["nextPageToken"] = end
		}
		return out
	}
	return map[string]any{"users": users, "totalCount": len(f.users)}
}

func (f *groupFake) create(in map[string]any) (int, any) {
	name, _ := in["name"].(string)
	if name == "" {
		return groupError(http.StatusBadRequest, "Bad Request", "Group name cannot be empty")
	}
	for _, other := range f.groups {
		if other.name == name {
			return groupError(http.StatusConflict, "Already Exists", fmt.Sprintf("Team group %s already exists for team %d", name, groupFakeTeam))
		}
	}
	group := groupRecord{groupType: "GROUP_TYPE_OPEN", userIDs: []string{}}
	if description, ok := in["description"].(string); ok {
		group.description = &description
	}
	if groupType, ok := in["groupType"].(string); ok {
		if status, errBody := checkGroupType(groupType); status != 0 {
			return status, errBody
		}
		if groupType != "GROUP_TYPE_UNSPECIFIED" {
			group.groupType = groupType
		}
	}
	group.name = name
	if raw, ok := in["roleId"]; ok {
		id, status, errBody := f.checkRole(raw)
		if status != 0 {
			return status, errBody
		}
		group.roleID = &id
	}
	if scope, ok := in["scope"].(map[string]any); ok {
		if scopeID, _ := scope["scopeId"].(string); scopeID != "" {
			if status, errBody := f.checkScope(scopeID); status != 0 {
				return status, errBody
			}
			group.scopeID = &scopeID
		}
	}
	if raw, ok := in["userIds"].([]any); ok {
		ids, status, errBody := f.checkUsers(raw, "create")
		if status != 0 {
			return status, errBody
		}
		group.userIDs = ids
	}
	id := f.nextID
	f.nextID++
	f.groups[id] = group
	return http.StatusOK, map[string]any{"group": f.answer(id, group, false)}
}

func (f *groupFake) get(idText string) (int, any) {
	id, group, status, errBody := f.find(idText)
	if status != 0 {
		return status, errBody
	}
	return http.StatusOK, map[string]any{"group": f.answer(id, group, true)}
}

func (f *groupFake) update(idText string, in map[string]any) (int, any) {
	id, group, status, errBody := f.find(idText)
	if status != 0 {
		return status, errBody
	}
	if name, ok := in["name"].(string); ok {
		if name == "" {
			return groupError(http.StatusBadRequest, "Bad Request", "Group name cannot be empty")
		}
		for otherID, other := range f.groups {
			if otherID != id && other.name == name {
				return groupError(http.StatusInternalServerError, "Internal Server Error", "Internal Error - service error")
			}
		}
		group.name = name
	}
	group.description = nil
	if description, ok := in["description"].(string); ok {
		group.description = &description
	}
	if groupType, ok := in["groupType"].(string); ok {
		if status, errBody := checkGroupType(groupType); status != 0 {
			return status, errBody
		}
		group.groupType = "GROUP_TYPE_OPEN"
		if groupType != "GROUP_TYPE_UNSPECIFIED" {
			group.groupType = groupType
		}
	}
	if update, ok := in["roleUpdate"].(map[string]any); ok {
		action, _ := update["action"].(map[string]any)
		switch actionType, _ := action["actionType"].(string); {
		case action["clear"] != nil || actionType == "clear":
			group.roleID = nil
		case action["setRoleId"] != nil || actionType == "set_role_id":
			set, _ := action["setRoleId"].(map[string]any)
			roleID, status, errBody := f.checkRole(set["value"])
			if status != 0 {
				return status, errBody
			}
			group.roleID = &roleID
		}
	}
	if update, ok := in["scopeUpdate"].(map[string]any); ok {
		action, _ := update["action"].(map[string]any)
		switch actionType, _ := action["actionType"].(string); {
		case action["clear"] != nil || actionType == "clear":
			group.scopeID = nil
		case action["setScopeId"] != nil || actionType == "set_scope_id":
			set, _ := action["setScopeId"].(map[string]any)
			scopeID, _ := set["value"].(string)
			if scopeID == "" {
				group.scopeID = nil
				break
			}
			if status, errBody := f.checkScope(scopeID); status != 0 {
				return status, errBody
			}
			group.scopeID = &scopeID
		}
	}
	if update, ok := in["userUpdates"].(map[string]any); ok {
		operation, _ := update["operation"].(map[string]any)
		if set, ok := operation["set"].(map[string]any); ok {
			raw, _ := set["userIds"].([]any)
			ids, status, errBody := f.checkUsers(raw, "update")
			if status != 0 {
				return status, errBody
			}
			group.userIDs = ids
		}
	}
	f.groups[id] = group
	return http.StatusOK, map[string]any{"group": f.answer(id, group, false)}
}

// find reads the group of a path id. A non-zero status is an error answer.
func (f *groupFake) find(idText string) (int64, groupRecord, int, any) {
	id, err := strconv.ParseUint(idText, 10, 63)
	if err != nil {
		status, body := groupError(http.StatusBadRequest, "Bad Request",
			fmt.Sprintf("type mismatch, parameter: group_id, error: strconv.ParseUint: parsing %q: invalid syntax", idText))
		return 0, groupRecord{}, status, body
	}
	group, ok := f.groups[int64(id)]
	if !ok {
		status, body := groupError(http.StatusNotFound, "Not Found", fmt.Sprintf("group with id TeamGroupId(%d) not found", id))
		return 0, groupRecord{}, status, body
	}
	return int64(id), group, 0, nil
}

func (f *groupFake) checkRole(raw any) (int64, int, any) {
	var id int64
	switch v := raw.(type) {
	case float64:
		id = int64(v)
	case string:
		id, _ = strconv.ParseInt(v, 10, 64)
	}
	if _, ok := f.roles[id]; !ok {
		status, body := groupError(http.StatusForbidden, "Permission Denied",
			"Trying to perform operation on roles which are not of type team or do not belong to this team")
		return 0, status, body
	}
	return id, 0, nil
}

func (f *groupFake) checkScope(id string) (int, any) {
	if !f.scopes[id] {
		return groupError(http.StatusNotFound, "Not Found", "Next gen scope not found")
	}
	return 0, nil
}

// checkUsers reads a list of user ids and keeps each id once, in order.
func (f *groupFake) checkUsers(raw []any, verb string) ([]string, int, any) {
	known := map[string]bool{}
	for _, user := range f.users {
		known[user.id] = true
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, item := range raw {
		id, _ := item.(string)
		if !known[id] {
			status, body := groupError(http.StatusForbidden, "Permission Denied", fmt.Sprintf(
				"Could not %s group for team %d. Some users, requested to be added to %sgroup, are not part of the group's team.",
				verb, groupFakeTeam, map[string]string{"create": "", "update": "updated "}[verb]))
			return nil, status, body
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, 0, nil
}

func checkGroupType(value string) (int, any) {
	switch value {
	case "GROUP_TYPE_UNSPECIFIED", "GROUP_TYPE_OPEN", "GROUP_TYPE_CLOSED", "GROUP_TYPE_RESTRICTED":
		return 0, nil
	}
	return groupError(http.StatusBadRequest, "Bad Request",
		fmt.Sprintf("proto: (line 1:38): invalid value for enum field groupType: %q", value))
}

// answer is the group as the API shows it. Create and Update leave the users out.
func (f *groupFake) answer(id int64, group groupRecord, withUsers bool) map[string]any {
	out := map[string]any{
		"groupId":     id,
		"name":        group.name,
		"teamId":      groupFakeTeam,
		"groupOrigin": "GROUP_ORIGIN_USER_DEFINED",
		"groupType":   group.groupType,
		"createdAt":   groupFakeTime,
		"updatedAt":   groupFakeTime,
		"userIds":     []any{},
	}
	if group.description != nil {
		out["description"] = *group.description
	}
	if group.roleID != nil {
		role := f.roles[*group.roleID]
		out["role"] = map[string]any{"roleId": *group.roleID, "name": role.name, "description": role.description}
	}
	if group.scopeID != nil {
		out["scope"] = map[string]any{"scopeId": *group.scopeID}
	}
	if withUsers {
		ids := make([]any, len(group.userIDs))
		for i, userID := range group.userIDs {
			ids[i] = userID
		}
		out["userIds"] = ids
	}
	return out
}

// groupError is an error answer of the gateway. The label is the gRPC code name that the real
// API writes before the reason, which is not always the HTTP status text.
func groupError(status int, label, reason string) (int, any) {
	return status, map[string]any{"code": status, "message": label + ": " + reason}
}
