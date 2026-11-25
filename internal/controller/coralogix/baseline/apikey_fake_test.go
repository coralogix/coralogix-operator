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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const apiKeyPath = "/aaa/api-keys/v3"

// apiKeyFakeTeam is the team of the caller. The fake accepts a key that this team owns, as the
// real API accepts a key only for the team of the caller. It is the team of the released sample
// config/samples/v1alpha1/apikeys/team_key.yaml.
const apiKeyFakeTeam = 4013254

// apiKeyFakeUser is a member of the caller's team, so a key that this user owns is created. It is
// the user of the released sample config/samples/v1alpha1/apikeys/user_key.yaml. The recording key
// could not create a key of a user, so this answer is from the contract, not from EU2.
const apiKeyFakeUser = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

// apiKeyPresets are the presets that the fake knows, with the permissions that Get shows for
// them. APM is as EU2 answered it.
var apiKeyPresets = map[string][]string{
	"APM": {
		"apm-views-folders:ReadAccessPolicy", "apm-views-folders:UpdateAccessPolicy", "serverless:Read",
		"service-catalog:Read", "service-catalog:ReadApdexConfig", "service-catalog:ReadDimensionsConfig",
		"service-catalog:ReadSLIConfig", "service-catalog:Update", "service-catalog:UpdateApdexConfig",
		"service-catalog:UpdateDimensionsConfig", "service-catalog:UpdateSLIConfig", "service-map:Read",
	},
	"SendData": {
		"cloud-metadata-ingress:SendData", "contextual-data:SendData", "logs.data-ingress:SendData",
		"metrics.data-ingress:SendData", "profiles-data-ingress:SendData", "profiles.data-ingress:SendData",
		"spans.data-ingress:SendData",
	},
}

// apiKeyPermissions are the permissions that the fake knows. A legacy name maps to the name that
// Get shows.
var apiKeyPermissions = map[string]string{
	"livetail:Read":   "livetail:Read",
	"serverless:Read": "serverless:Read",
	"alerts:MapRead":  "alerts:MapRead",
	"ALERTS-MAP:READ": "alerts:MapRead",
}

// apiKeyTeamPolicyPermissions are the permissions that the policy of a key that a team owns may
// hold. The API adds each one that a policy leaves out, as deny.
var apiKeyTeamPolicyPermissions = []string{
	"team-custom-api-keys:Manage", "team-custom-api-keys:ReadAccessPolicy",
	"team-custom-api-keys:ReadConfig", "team-custom-api-keys:UpdateAccessPolicy",
}

type apiKeyRecord struct {
	name        string
	teamID      int64
	userID      string
	active      bool
	hashed      bool
	presets     []string
	permissions []string
	// accessPolicy is the stored policy, or "" for none.
	accessPolicy string
	value        string
}

// apiKeyFake is the API keys service, written from the SDK's openapi.yaml.
//
//	POST   /aaa/api-keys/v3           {name, owner, keyPermissions, hashed, accessPolicy} -> {keyId, name, value}
//	GET    /aaa/api-keys/v3/{key_id}  -> {keyInfo}
//	PUT    /aaa/api-keys/v3/{key_id}  {newName, isActive, presets, permissions, accessPolicy} -> {}
//	DELETE /aaa/api-keys/v3/{key_id}  -> {}
//
// The status codes and error texts follow fidelity/recorded.json, which EU2 answered to a key of
// a team:
//   - Create needs a name and a preset or a permission. A new key is active.
//   - A key of another team or of a user that is not a member is a 403. An empty owner is a 400.
//   - A preset matches in any case and reads back by its name. A legacy permission name reads
//     back by its new name. An unknown preset or permission is a 400. A repeated permission is a 500.
//   - Update changes only what the body has. A presets or permissions object replaces the list,
//     and an object without the list clears it: Update may leave a key with neither.
//   - Update with an empty new name is a 400.
//   - The API stores a policy as JSON, without the fields it does not know, with the version
//     without its "v", and with each team permission that the policy leaves out added as deny.
//     "" clears the policy on Update, and is no policy on Create. Each invalid JSON gets the
//     recorded error text, which EU2 gave for "not json".
//   - Get of a hashed key has no value.
//   - Get, Update, and Delete of a missing key answer 404. An id that is not a UUID is a 400.
//
// The recording key belongs to a team, so the fake answers a key of an organisation with a 400
// that was not recorded, and accepts a key of one user, apiKeyFakeUser.
type apiKeyFake struct {
	mu     sync.Mutex
	keys   map[string]apiKeyRecord
	forced map[string]forcedAnswer
	// next numbers the secret values, so the goldens stay the same on each run.
	next int
}

func newAPIKeyFake() apiFake {
	return &apiKeyFake{keys: map[string]apiKeyRecord{}, forced: map[string]forcedAnswer{}}
}

// answerNext makes the next request with the method get this answer instead of the fake's own.
func (f *apiKeyFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

// forget deletes a key without a request, as if it was deleted outside the operator.
func (f *apiKeyFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.keys, id)
}

// put stores a key without a request, as if another client created it.
func (f *apiKeyFake) put(id string, key apiKeyRecord) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys[id] = key
}

func (f *apiKeyFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.keys))
	for id, key := range f.keys {
		out[id] = key.answer(id)
	}
	return out
}

func (f *apiKeyFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	in, _ := body.(map[string]any)

	if path == apiKeyPath {
		if method == http.MethodPost {
			return f.create(in)
		}
		return routeNotFound()
	}
	id, ok := strings.CutPrefix(path, apiKeyPath+"/")
	if !ok || id == "" || strings.Contains(id, "/") {
		return routeNotFound()
	}
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodDelete:
	default:
		return routeNotFound()
	}
	if status, errBody := apiKeyIDError(id); status != 0 {
		return status, errBody
	}
	key, found := f.keys[id]
	switch method {
	case http.MethodGet:
		if !found {
			return apiError(http.StatusNotFound, fmt.Sprintf("Key with id KeyId(%s) does not exist", id))
		}
		return http.StatusOK, map[string]any{"keyInfo": key.answer(id)}
	case http.MethodPut:
		if !found {
			return apiError(http.StatusNotFound, fmt.Sprintf("Key %s not found", id))
		}
		return f.update(id, key, in)
	default:
		if !found {
			return apiError(http.StatusNotFound, fmt.Sprintf("Key with id KeyId(%s) does not exist", id))
		}
		delete(f.keys, id)
		return http.StatusOK, map[string]any{}
	}
}

func (f *apiKeyFake) create(in map[string]any) (int, any) {
	var key apiKeyRecord
	key.name, _ = in["name"].(string)
	if key.name == "" {
		return apiError(http.StatusBadRequest, "name must be not empty")
	}
	permissions, _ := in["keyPermissions"].(map[string]any)
	presets, _ := permissions["presets"].([]any)
	direct, _ := permissions["permissions"].([]any)
	if len(presets) == 0 && len(direct) == 0 {
		return apiError(http.StatusBadRequest, "preset_ids or permission_ids must be provided")
	}
	owner, _ := in["owner"].(map[string]any)
	switch {
	case owner["teamId"] != nil:
		team, ok := apiKeyInt(owner["teamId"])
		if !ok || team != apiKeyFakeTeam {
			return apiKeyDenied(`"Caller is not authorize to perform this action"`)
		}
		key.teamID = team
	case owner["userId"] != nil:
		user, _ := owner["userId"].(string)
		if user == apiKeyFakeUser {
			key.userID = user
			break
		}
		return apiKeyDenied(fmt.Sprintf(`"The user specified as the API key owner, '%s', either does not exist or is not a member of the current team."`, user))
	case owner["organisationId"] != nil:
		return apiError(http.StatusBadRequest, "an organisation cannot own a key")
	default:
		return apiError(http.StatusBadRequest, "Missing Owner.")
	}
	var status int
	var errBody any
	if key.presets, status, errBody = apiKeyPresetNames(presets); status != 0 {
		return status, errBody
	}
	if key.permissions, status, errBody = apiKeyPermissionNames(direct); status != 0 {
		return status, errBody
	}
	if policy, ok := in["accessPolicy"].(string); ok && policy != "" {
		if key.accessPolicy, status, errBody = apiKeyPolicy(policy); status != 0 {
			return status, errBody
		}
	}
	key.hashed, _ = in["hashed"].(bool)
	key.active = true
	f.next++
	key.value = fmt.Sprintf("cxo-fake-key-value-%d", f.next)
	id := newUUID()
	f.keys[id] = key
	return http.StatusOK, map[string]any{"keyId": id, "name": key.name, "value": key.value}
}

func (f *apiKeyFake) update(id string, key apiKeyRecord, in map[string]any) (int, any) {
	if name, ok := in["newName"].(string); ok {
		if name == "" {
			return apiError(http.StatusBadRequest, "new_name must be not empty")
		}
		key.name = name
	}
	if active, ok := in["isActive"].(bool); ok {
		key.active = active
	}
	var status int
	var errBody any
	if presets, ok := in["presets"].(map[string]any); ok {
		list, _ := presets["presets"].([]any)
		if key.presets, status, errBody = apiKeyPresetNames(list); status != 0 {
			return status, errBody
		}
	}
	if permissions, ok := in["permissions"].(map[string]any); ok {
		list, _ := permissions["permissions"].([]any)
		if key.permissions, status, errBody = apiKeyPermissionNames(list); status != 0 {
			return status, errBody
		}
	}
	if policy, ok := in["accessPolicy"].(string); ok {
		key.accessPolicy = ""
		if policy != "" {
			if key.accessPolicy, status, errBody = apiKeyPolicy(policy); status != 0 {
				return status, errBody
			}
		}
	}
	f.keys[id] = key
	return http.StatusOK, map[string]any{}
}

// apiKeyDenied is a 403. The gateway names it with the gRPC code, not the HTTP status text.
func apiKeyDenied(reason string) (int, any) {
	return http.StatusForbidden, map[string]any{"code": http.StatusForbidden, "message": "Permission Denied: Permission denied " + reason}
}

// apiKeyIDError is the 400 for an id that is not a UUID, or 0.
func apiKeyIDError(id string) (int, any) {
	if uuidPattern.MatchString(strings.ToLower(id)) && len(id) == 36 {
		return 0, nil
	}
	for i, c := range id {
		if !strings.ContainsRune("0123456789abcdefABCDEF-", c) {
			return apiError(http.StatusBadRequest, fmt.Sprintf("key_id invalid character: found `%c` at %d", c, i+1))
		}
	}
	return apiError(http.StatusBadRequest, "key_id invalid length")
}

// apiKeyInt reads a team id, which the SDK sends as a number and proto JSON may send as text.
func apiKeyInt(v any) (int64, bool) {
	switch v := v.(type) {
	case float64:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	}
	return 0, false
}

func apiKeyPresetNames(list []any) ([]string, int, any) {
	out := []string{}
	for _, item := range list {
		name, _ := item.(string)
		found := ""
		for preset := range apiKeyPresets {
			if strings.EqualFold(preset, name) {
				found = preset
			}
		}
		if found == "" {
			status, body := apiError(http.StatusBadRequest, fmt.Sprintf("Preset %s does not exist or is not a valid preset for the key being created.", name))
			return nil, status, body
		}
		out = append(out, found)
	}
	return out, 0, nil
}

func apiKeyPermissionNames(list []any) ([]string, int, any) {
	out := []string{}
	seen := map[string]bool{}
	for _, item := range list {
		name, _ := item.(string)
		permission, ok := apiKeyPermissions[name]
		if !ok {
			status, body := apiError(http.StatusBadRequest, fmt.Sprintf("BadRequest Permission %s not found", name))
			return nil, status, body
		}
		if seen[permission] {
			status, body := apiError(http.StatusInternalServerError, "Internal Error - service error")
			return nil, status, body
		}
		seen[permission] = true
		out = append(out, permission)
	}
	return out, 0, nil
}

// apiKeyPolicy reads a policy of a key that a team owns and returns it as the API stores it.
func apiKeyPolicy(text string) (string, int, any) {
	var in struct {
		Version string `json:"version"`
		Default struct {
			Permissions map[string]string `json:"permissions"`
		} `json:"default"`
		Rules []any `json:"rules"`
	}
	if err := json.Unmarshal([]byte(text), &in); err != nil {
		status, body := apiError(http.StatusBadRequest, "invalid argument: provided access_policy_raw_json is not a valid json: expected ident at line 1 column 2")
		return "", status, body
	}
	allowed := map[string]bool{}
	for _, p := range apiKeyTeamPolicyPermissions {
		allowed[p] = true
	}
	var invalid []string
	for p := range in.Default.Permissions {
		if !allowed[p] {
			invalid = append(invalid, apiKeyPolicyPermissionName(p))
		}
	}
	if len(invalid) > 0 {
		sort.Strings(invalid)
		var b strings.Builder
		for _, p := range invalid {
			fmt.Fprintf(&b, "    %q,\n", p)
		}
		status, body := apiError(http.StatusBadRequest, fmt.Sprintf(
			"invalid argument: access policy contains permissions [[\n%s]] which are not valid for resource type: resource_team_custom_api_keys", b.String()))
		return "", status, body
	}
	permissions := map[string]string{}
	for _, p := range apiKeyTeamPolicyPermissions {
		permissions[p] = "deny"
	}
	for p, grant := range in.Default.Permissions {
		permissions[p] = grant
	}
	rules := in.Rules
	if rules == nil {
		rules = []any{}
	}
	out, err := json.Marshal(map[string]any{
		"version": strings.TrimPrefix(in.Version, "v"),
		"default": map[string]any{"permissions": permissions},
		"rules":   rules,
	})
	if err != nil {
		panic(err)
	}
	return string(out), 0, nil
}

// apiKeyPolicyPermissionName is the name of a permission in an error text of the API:
// data-ingest-api-keys:ReadConfig is permission_data_ingest_api_keys_read_config.
func apiKeyPolicyPermissionName(permission string) string {
	resource, action, _ := strings.Cut(permission, ":")
	var b strings.Builder
	for i, c := range action {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			c += 'a' - 'A'
		}
		b.WriteRune(c)
	}
	return "permission_" + strings.ReplaceAll(resource, "-", "_") + "_" + b.String()
}

func (k apiKeyRecord) answer(id string) map[string]any {
	presets := make([]any, 0, len(k.presets))
	for _, name := range k.presets {
		permissions := make([]any, 0, len(apiKeyPresets[name]))
		for _, p := range apiKeyPresets[name] {
			permissions = append(permissions, p)
		}
		presets = append(presets, map[string]any{"name": name, "permissions": permissions})
	}
	permissions := make([]any, 0, len(k.permissions))
	for _, p := range k.permissions {
		permissions = append(permissions, p)
	}
	owner := map[string]any{}
	if k.userID != "" {
		owner["userId"] = k.userID
	} else {
		owner["teamId"] = k.teamID
	}
	out := map[string]any{
		"id":             id,
		"name":           k.name,
		"owner":          owner,
		"active":         k.active,
		"hashed":         k.hashed,
		"keyPermissions": map[string]any{"presets": presets, "permissions": permissions},
	}
	if !k.hashed {
		out["value"] = k.value
	}
	if k.accessPolicy != "" {
		out["accessPolicy"] = k.accessPolicy
	}
	return out
}
