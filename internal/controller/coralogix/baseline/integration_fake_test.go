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
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
)

const (
	integrationMetadataPath = "/integrations/metadata/v1"
	integrationDeployedPath = "/integrations/integrations/v1/deployed/"
	integrationInstancePath = "/integrations/integrations/v1/instances/"
)

// integrationDefinitions are the integration keys and versions that the fake knows, with the
// required parameters of each version in the order of the definition. The real API reads them
// from its catalog.
var integrationDefinitions = map[string]map[string][]string{
	"aws-metrics-collector": {"0.1.0": {
		"IntegrationName", "AwsRoleArn", "AwsRegion", "ApplicationName", "SubsystemName",
		"MetricNamespaces", "WithAggregations", "EnrichWithTags",
	}},
	"gcp-metrics-collector": {"1.0.0": {"MetricPrefixes", "ServiceAccountKey"}},
}

// integrationServiceAccountKeys are the parameters that hold a GCP service account key. The real
// API checks that the key is JSON with a string project_id, and Get shows the parameter as
// sensitiveData without its value.
var integrationServiceAccountKeys = map[string]string{"gcp-metrics-collector": "ServiceAccountKey"}

type integrationRecord struct {
	key        string
	version    string
	parameters map[string]map[string]any
}

// integrationFake keeps the remote state that the handwritten controller can change.
type integrationFake struct {
	mu           sync.Mutex
	integrations map[string]integrationRecord
	forced       map[string]forcedAnswer
}

func newIntegrationFake() apiFake {
	return &integrationFake{integrations: map[string]integrationRecord{}, forced: map[string]forcedAnswer{}}
}

func (f *integrationFake) answerNext(method string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.forced[method] = forcedAnswer{status: status, body: body}
}

func (f *integrationFake) remove(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.integrations, id)
}

// seed stores an integration, as if the released operator made it, and returns
// its id.
func (f *integrationFake) seed(key, version string, parameters map[string]map[string]any) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := newUUID()
	f.integrations[id] = integrationRecord{key: key, version: version, parameters: parameters}
	return id
}

func (f *integrationFake) snapshot() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]any, len(f.integrations))
	for id, rec := range f.integrations {
		out[id] = rec.answer(id)
	}
	return out
}

func (f *integrationFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.forced[method]; ok {
		delete(f.forced, method)
		return forced.status, forced.body
	}
	switch {
	case path == integrationMetadataPath && method == http.MethodPost:
		return f.create(body)
	case path == integrationMetadataPath && method == http.MethodPut:
		return f.update(body)
	case strings.HasPrefix(path, integrationDeployedPath) && method == http.MethodGet:
		id := strings.TrimPrefix(path, integrationDeployedPath)
		rec, ok := f.integrations[id]
		if !ok {
			return integrationNotFound(id)
		}
		return http.StatusOK, map[string]any{"integration": rec.answer(id)}
	case strings.HasPrefix(path, integrationInstancePath) && method == http.MethodDelete:
		id := strings.TrimPrefix(path, integrationInstancePath)
		if _, ok := f.integrations[id]; !ok {
			return integrationNotFound(id)
		}
		delete(f.integrations, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

func (f *integrationFake) create(body any) (int, any) {
	metadata, _ := mapField(body, "metadata")
	rec, status, answer := readIntegrationMetadata(metadata)
	if status != 0 {
		return status, answer
	}
	// Create needs every required parameter of the version.
	var missing []string
	for _, name := range integrationDefinitions[rec.key][rec.version] {
		if _, ok := rec.parameters[name]; !ok {
			missing = append(missing, "Required field "+name+" not provided")
		}
	}
	if len(missing) > 0 {
		return integrationBadRequest(strings.Join(missing, "\n"))
	}
	if status, answer := checkServiceAccountKey(rec, "create"); status != 0 {
		return status, answer
	}
	id := newUUID()
	f.integrations[id] = rec
	return http.StatusOK, map[string]any{"integrationId": id}
}

func (f *integrationFake) update(body any) (int, any) {
	m, _ := body.(map[string]any)
	id, _ := m["id"].(string)
	metadata, _ := mapField(body, "metadata")
	rec, status, answer := readIntegrationMetadata(metadata)
	if status != 0 {
		return status, answer
	}
	stored, ok := f.integrations[id]
	if !ok {
		return integrationNotFound(id)
	}
	if status, answer := checkServiceAccountKey(rec, "update"); status != 0 {
		return status, answer
	}
	// A parameter that the request omits keeps its stored value.
	for key, value := range rec.parameters {
		stored.parameters[key] = value
	}
	stored.key, stored.version = rec.key, rec.version
	f.integrations[id] = stored
	return http.StatusOK, map[string]any{}
}

// readIntegrationMetadata checks the metadata as the API does. A status other than 0 is the error
// answer.
func readIntegrationMetadata(metadata map[string]any) (integrationRecord, int, any) {
	key, _ := metadata["integrationKey"].(string)
	version, _ := metadata["version"].(string)
	if key == "" {
		status, answer := integrationBadRequest("integrationKey must not be empty")
		return integrationRecord{}, status, answer
	}
	versions, known := integrationDefinitions[key]
	if !known {
		status, answer := apiError(http.StatusNotFound, "Integration type by integration key not found: "+key)
		return integrationRecord{}, status, answer
	}
	if version == "" {
		status, answer := integrationBadRequest("version must not be empty")
		return integrationRecord{}, status, answer
	}
	required, ok := versions[version]
	if !ok {
		status, answer := integrationBadRequest("Invalid version")
		return integrationRecord{}, status, answer
	}
	params, ok := mapField(metadata, "integrationParameters")
	if !ok {
		status, answer := integrationBadRequest("Unknown integration metadata specific data: Empty")
		return integrationRecord{}, status, answer
	}
	rec := integrationRecord{key: key, version: version, parameters: map[string]map[string]any{}}
	list, _ := params["parameters"].([]any)
	for _, item := range list {
		p, _ := item.(map[string]any)
		name, _ := p["key"].(string)
		value := map[string]any{}
		for field, v := range p {
			if field != "key" {
				value[field] = v
			}
		}
		if len(value) == 0 {
			status, answer := integrationBadRequest("Not supported parameter type")
			return integrationRecord{}, status, answer
		}
		// A list without values is a parameter that the request does not provide.
		if list, ok := value["stringList"].(map[string]any); ok {
			if values, _ := list["values"].([]any); len(values) == 0 {
				if containsString(required, name) {
					status, answer := integrationBadRequest("Required field " + name + " not provided")
					return integrationRecord{}, status, answer
				}
				continue
			}
		}
		rec.parameters[name] = value
	}
	return rec, 0, nil
}

// checkServiceAccountKey answers 400 when the request sends a service account key that is not JSON
// with a string project_id. The real API does not check the key with GCP.
func checkServiceAccountKey(rec integrationRecord, call string) (int, any) {
	name, ok := integrationServiceAccountKeys[rec.key]
	if !ok {
		return 0, nil
	}
	value, ok := rec.parameters[name]
	if !ok {
		return 0, nil
	}
	text, _ := value["stringValue"].(string)
	var key map[string]any
	if err := json.Unmarshal([]byte(text), &key); err == nil {
		if _, ok := key["project_id"].(string); ok {
			return 0, nil
		}
	}
	return integrationBadRequest("Invalid argument: Calling service " + call + " for " + rec.key +
		" failed: code: 'Client specified an invalid argument', message: \"Service account key does not contain a string project_id\"")
}

func (r integrationRecord) answer(id string) map[string]any {
	return map[string]any{
		"id":                id,
		"definitionKey":     r.key,
		"definitionVersion": r.version,
		"parameters":        r.parameterList(integrationServiceAccountKeys[r.key]),
	}
}

// parameterList returns the parameters sorted by key. The sensitive parameter has no value.
func (r integrationRecord) parameterList(sensitive string) []any {
	keys := make([]string, 0, len(r.parameters))
	for key := range r.parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		p := map[string]any{"key": key}
		if key == sensitive {
			p["sensitiveData"] = map[string]any{}
			out = append(out, p)
			continue
		}
		for field, v := range r.parameters[key] {
			p[field] = v
		}
		out = append(out, p)
	}
	return out
}

func integrationNotFound(id string) (int, any) {
	return apiError(http.StatusNotFound, "Integration not found: "+id)
}

func integrationBadRequest(reason string) (int, any) {
	return apiError(http.StatusBadRequest, "Request errors:\n* "+reason)
}

func mapField(v any, name string) (map[string]any, bool) {
	m, _ := v.(map[string]any)
	field, ok := m[name].(map[string]any)
	return field, ok
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
