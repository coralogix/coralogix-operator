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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// apiBasePath is the path of the management OpenAPI on every Coralogix domain.
const apiBasePath = "/mgmt/openapi/5"

// apiFake answers the requests of one kind as the real API does. path is relative to
// apiBasePath. body is the parsed JSON body, or nil. The fake guards its own state.
type apiFake interface {
	handle(method, path string, query url.Values, body any) (status int, response any)
}

// recordedRequest is one request that the SDK client sent.
type recordedRequest struct {
	Method string     `json:"method"`
	Path   string     `json:"path"`
	Query  url.Values `json:"query,omitempty"`
	Body   any        `json:"body,omitempty"`
	Status int        `json:"status"`
}

// fakeServer is an HTTP server in front of an apiFake. It records each request.
type fakeServer struct {
	api     apiFake
	baseURL string

	mu       sync.Mutex
	requests []recordedRequest
}

func newFakeServer(t *testing.T, api apiFake) *fakeServer {
	t.Helper()
	s := &fakeServer{api: api}
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	s.baseURL = srv.URL + apiBasePath
	return s
}

func (s *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body := parseBody(raw)
	status, resp := s.api.handle(r.Method, strings.TrimPrefix(r.URL.Path, apiBasePath), r.URL.Query(), body)

	rec := recordedRequest{Method: r.Method, Path: r.URL.Path, Body: body, Status: status}
	if len(r.URL.Query()) > 0 {
		rec.Query = r.URL.Query()
	}
	s.mu.Lock()
	s.requests = append(s.requests, rec)
	s.mu.Unlock()

	out, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}

// takeRequests returns the requests since the last call.
func (s *fakeServer) takeRequests() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.requests
	s.requests = nil
	if out == nil {
		out = []recordedRequest{}
	}
	return out
}

// parseBody returns the JSON body, the raw text when it is not JSON, or nil when it is empty.
func parseBody(raw []byte) any {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var body any
	if err := json.Unmarshal(raw, &body); err != nil {
		return string(raw)
	}
	return body
}

// apiError is an error answer of the API gateway: {"code": 404, "message": "Not Found: <reason>"}.
// The SDK reads a 404 as "resource not found" only when the reason is not "Not Found".
func apiError(status int, reason string) (int, any) {
	return status, map[string]any{"code": status, "message": http.StatusText(status) + ": " + reason}
}

// routeNotFound is the answer for a path that the API does not have.
func routeNotFound() (int, any) {
	return apiError(http.StatusNotFound, "Not Found")
}

// newUUID returns a random version 4 UUID, as the API makes for a new resource.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
