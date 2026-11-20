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
	"sort"
	"sync"
)

// The team of the fake. Its region is the region of the Coralogix cluster: the API takes a bucket
// only in that region. archiveBucket exists in the region; archiveOtherBucket does too.
const (
	archiveTeamRegion  = "eu-north-1"
	archiveBucket      = "cxo-fidelity-archive"
	archiveOtherBucket = "cxo-fidelity-archive-2"
)

// archiveLogsTargetFake is the archive target API and the retentions activation, written from
// the SDK's openapi.yaml.
//
//	GET  /logs/data-setup/v2                      -> {target: {s3, archiveSpec, managedBy, ...}}
//	POST /logs/data-setup/v2                      {isActive, s3: {bucket, region?, ...}, format?} -> {target}
//	POST /dataengine/retention-tags/v1/activate   -> {activateRetentions: true}
//	GET  /dataengine/retention-tags/v1/enabled    -> {enableTags}
//
// The team has one target. It starts as the team of the recording: an inactive S3 target. The
// status codes, error texts, and stored values follow fidelity/recorded.json. Each rule names the
// recorded case that shows it.
type archiveLogsTargetFake struct {
	mu       sync.Mutex
	buckets  map[string]bool
	bucket   string
	region   string
	external string
	active   bool
	tags     bool
	// next holds one forced answer per path of a POST. The next POST to the path gets it, and the
	// fake does not run the request. Every call of the operator is a POST.
	next map[string]forcedAnswer
}

func newArchiveLogsTargetFake() apiFake {
	return &archiveLogsTargetFake{
		buckets:  map[string]bool{archiveBucket: true, archiveOtherBucket: true},
		bucket:   archiveBucket,
		region:   archiveTeamRegion,
		external: "eu2-0000000",
		tags:     true,
		next:     map[string]forcedAnswer{},
	}
}

func archiveTargets(h *harness) *archiveLogsTargetFake {
	h.t.Helper()
	f, ok := h.API().(*archiveLogsTargetFake)
	if !ok {
		h.t.Fatal("expected *archiveLogsTargetFake")
	}
	return f
}

// answerNext makes the next POST to the path get this answer.
func (f *archiveLogsTargetFake) answerNext(path string, status int, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[path] = forcedAnswer{status: status, body: body}
}

// answerNextError makes the next POST to the path get an error answer.
func (f *archiveLogsTargetFake) answerNextError(path string, status int, reason string) {
	st, body := apiError(status, reason)
	f.answerNext(path, st, body)
}

// answerNextRouteNotFound makes the next POST to the path get the answer for a route that the API
// does not have.
func (f *archiveLogsTargetFake) answerNextRouteNotFound(path string) {
	st, body := routeNotFound()
	f.answerNext(path, st, body)
}

func (f *archiveLogsTargetFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if forced, ok := f.next[path]; ok && method == http.MethodPost {
		delete(f.next, path)
		return forced.status, forced.body
	}
	switch {
	case method == http.MethodGet && path == archiveTargetPath:
		return http.StatusOK, f.answer()
	case method == http.MethodPost && path == archiveTargetPath:
		in, _ := body.(map[string]any)
		return f.set(in)
	case method == http.MethodPost && path == archiveActivatePath:
		// Case "activate retentions twice": a second activation answers the same.
		f.tags = true
		return http.StatusOK, map[string]any{"activateRetentions": true}
	case method == http.MethodGet && path == archiveTagsEnabledURL:
		return http.StatusOK, map[string]any{"enableTags": f.tags}
	}
	return routeNotFound()
}

// set follows the cases "set the team bucket ...", "a target without a region, and an empty
// region", "rejected targets", "a body without isActive deactivates", and "a format and an
// external id in the request".
func (f *archiveLogsTargetFake) set(in map[string]any) (int, any) {
	// An unknown field, such as ibmCos, stops the JSON decoder of the gateway.
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key != "isActive" && key != "s3" && key != "format" {
			return apiError(http.StatusBadRequest, fmt.Sprintf("proto: (line 1:2): unknown field %q", key))
		}
	}
	s3, ok := in["s3"].(map[string]any)
	if !ok {
		return apiError(http.StatusBadRequest, `"targetSpec" is required`)
	}
	bucket, _ := s3["bucket"].(string)
	region, hasRegion := s3["region"].(string)
	if bucket == "" || hasRegion && region == "" {
		return apiError(http.StatusBadRequest, `"targetSpec" does not match any of the allowed types`)
	}
	if hasRegion && region != archiveTeamRegion {
		return archiveDeadline(fmt.Sprintf("Region %s is not supported, must use %s.", region, archiveTeamRegion))
	}
	if !f.buckets[bucket] {
		return archiveDeadline(fmt.Sprintf("Failed to update archive target. Please provide a bucket in region %s and try again.", archiveTeamRegion))
	}
	f.bucket = bucket
	if hasRegion {
		f.region = region
	}
	if external, ok := s3["externalId"].(string); ok && external != "" {
		f.external = external
	}
	// A body without isActive sets the target inactive.
	f.active, _ = in["isActive"].(bool)
	return http.StatusOK, f.answer()
}

// archiveDeadline is the answer when the API cannot use the bucket: 408 Deadline Exceeded.
func archiveDeadline(text string) (int, any) {
	return http.StatusRequestTimeout, map[string]any{"code": http.StatusRequestTimeout, "message": "Deadline Exceeded: " + text}
}

func (f *archiveLogsTargetFake) answer() map[string]any {
	return map[string]any{"target": map[string]any{
		"s3": map[string]any{"bucket": f.bucket, "region": f.region, "externalId": f.external},
		"archiveSpec": map[string]any{
			"archivingFormatId": "cx_data_v1",
			"format":            "TARGET_FORMAT_CX_DATA_V1",
			"isActive":          f.active,
			"enableTags":        f.tags,
		},
		"managedBy":             "MANAGED_BY_CUSTOMER",
		"managedRetentionClass": "MANAGED_RETENTION_CLASS_UNSPECIFIED",
	}}
}
