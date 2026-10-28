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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
)

// The fidelity files check that a kind's fake answers as the real API does.
//
//	testdata/<kind>/fidelity/cases.json     requests to send (handwritten)
//	testdata/<kind>/fidelity/recorded.json  answers of the real API (recorded)
//
// scripts/record-baseline-fidelity.sh records recorded.json. TestFakeMatchesRecordedAPI sends
// the same requests to the fake and compares the answers.

// fidelityConfig is what the recorder needs to know about a kind.
type fidelityConfig struct {
	// captureField is the response field that holds the id of a new resource.
	captureField string
	// deletePath returns the path that deletes a resource. The recorder calls it for each
	// captured id when it ends, so a failed run leaves nothing behind.
	deletePath func(id string) string
}

type fidelityStep struct {
	Method string `json:"method"`
	// Path is relative to the OpenAPI base URL. {{name}} is a captured id.
	Path string `json:"path"`
	Body any    `json:"body,omitempty"`
	// Capture names the id in the response, for use in later steps.
	Capture string `json:"capture,omitempty"`
}

type fidelityCases struct {
	Cases []struct {
		Name  string         `json:"name"`
		Steps []fidelityStep `json:"steps"`
	} `json:"cases"`
}

type fidelityAnswer struct {
	Status   int `json:"status"`
	Response any `json:"response"`
}

type recordedCase struct {
	Name  string           `json:"name"`
	Steps []fidelityAnswer `json:"steps"`
}

type fidelityRecorded struct {
	Cases []recordedCase `json:"cases"`
}

func fidelityDir(k kind) string { return filepath.Join(k.goldenDir(), "fidelity") }

func loadCases(t *testing.T, k kind) fidelityCases {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fidelityDir(k), "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases fidelityCases
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

// TestFakeMatchesRecordedAPI sends the fidelity cases to the fake and compares each answer with
// the recorded answer of the real API. It skips a kind with no recording.
func TestFakeMatchesRecordedAPI(t *testing.T) {
	for _, k := range sortedKinds() {
		t.Run(k.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(fidelityDir(k), "recorded.json"))
			if errors.Is(err, fs.ErrNotExist) {
				t.Skipf("%s/recorded.json does not exist. Record it with scripts/record-baseline-fidelity.sh %s <region-or-domain>.", fidelityDir(k), k.name)
			}
			if err != nil {
				t.Fatal(err)
			}
			var recorded fidelityRecorded
			if err := json.Unmarshal(raw, &recorded); err != nil {
				t.Fatal(err)
			}
			cases := loadCases(t, k)
			if len(cases.Cases) != len(recorded.Cases) {
				t.Fatalf("cases.json has %d cases and recorded.json has %d: record again", len(cases.Cases), len(recorded.Cases))
			}
			for ci, c := range cases.Cases {
				t.Run(c.Name, func(t *testing.T) {
					rc := recorded.Cases[ci]
					if rc.Name != c.Name || len(rc.Steps) != len(c.Steps) {
						t.Fatalf("recorded.json does not match cases.json for %q: record again", c.Name)
					}
					api := k.newFake()
					vars := map[string]string{}
					for si, s := range c.Steps {
						path, query := splitQuery(t, substString(s.Path, vars))
						status, resp := api.handle(s.Method, path, query, substJSON(t, s.Body, vars))
						got := normalizeAnswer(t, k, s, status, roundTrip(t, resp), vars)
						compareAnswer(t, fmt.Sprintf("step %d: %s %s", si+1, s.Method, s.Path), fidelityAnswer{Status: status, Response: got}, rc.Steps[si])
					}
				})
			}
		})
	}
}

func compareAnswer(t *testing.T, label string, got, want fidelityAnswer) {
	t.Helper()
	if got.Status != want.Status || !reflect.DeepEqual(got.Response, want.Response) {
		t.Errorf("%s: the fake differs from the real API\n  fake: %d %s\n  real: %d %s",
			label, got.Status, compact(got.Response), want.Status, compact(want.Response))
	}
}

// TestRecordFidelity sends the fidelity cases of one kind to a real Coralogix API and writes
// recorded.json. Run it through scripts/record-baseline-fidelity.sh, which asks for the API key.
func TestRecordFidelity(t *testing.T) {
	name := os.Getenv("RECORD_FIDELITY")
	if name == "" {
		t.Skip("set RECORD_FIDELITY=<kind>: run scripts/record-baseline-fidelity.sh")
	}
	k, ok := kinds[name]
	if !ok {
		t.Fatalf("unknown kind %q", name)
	}
	base := recordBaseURL(t)
	apiKey := os.Getenv("CORALOGIX_API_KEY")
	if apiKey == "" {
		t.Fatal("CORALOGIX_API_KEY is empty")
	}

	var created []string
	t.Cleanup(func() {
		for _, id := range created {
			// The cases may have deleted it already, so the answer does not matter.
			_, _, _ = sendRecordRequest(base, apiKey, http.MethodDelete, k.fidelity.deletePath(id), nil)
		}
	})

	out := fidelityRecorded{}
	for _, c := range loadCases(t, k).Cases {
		rec := recordedCase{Name: c.Name}
		vars := map[string]string{}
		for _, s := range c.Steps {
			status, resp, err := sendRecordRequest(base, apiKey, s.Method, substString(s.Path, vars), substJSON(t, s.Body, vars))
			if err != nil {
				t.Fatalf("%s: %s %s: %v", c.Name, s.Method, s.Path, err)
			}
			if status == http.StatusUnauthorized || status == http.StatusForbidden {
				t.Fatalf("%s: %s %s returned %d. The API key needs permission to manage %s.", c.Name, s.Method, s.Path, status, k.name)
			}
			if s.Capture != "" && status == http.StatusOK {
				if id := capturedID(resp, k); id != "" {
					created = append(created, id)
				}
			}
			rec.Steps = append(rec.Steps, fidelityAnswer{Status: status, Response: normalizeAnswer(t, k, s, status, resp, vars)})
		}
		out.Cases = append(out.Cases, rec)
	}
	recorded := marshalGolden(t, out)
	checkNoSecrets(t, recorded, apiKey, base)
	if err := os.WriteFile(filepath.Join(fidelityDir(k), "recorded.json"), recorded, 0o644); err != nil {
		t.Fatal(err)
	}
}

// checkNoSecrets stops the recording before it writes a file that holds the API key or the
// API host. The file goes into a public repository.
func checkNoSecrets(t *testing.T, recorded []byte, apiKey, base string) {
	t.Helper()
	if bytes.Contains(recorded, []byte(apiKey)) {
		t.Fatal("the recording holds the API key; nothing was written")
	}
	if u, err := url.Parse(base); err == nil && u.Host != "" && bytes.Contains(recorded, []byte(u.Host)) {
		t.Fatalf("the recording holds the API host %s; nothing was written", u.Host)
	}
}

// recordBaseURL reads the target as the operator does: exactly one of CORALOGIX_REGION and
// CORALOGIX_DOMAIN.
func recordBaseURL(t *testing.T) string {
	t.Helper()
	region, domain := os.Getenv("CORALOGIX_REGION"), os.Getenv("CORALOGIX_DOMAIN")
	switch {
	case region != "" && domain != "":
		t.Fatal("set only one of CORALOGIX_REGION and CORALOGIX_DOMAIN")
	case region != "":
		base, ok := openapicxsdk.URLFromRegion(strings.ToLower(region))
		if !ok {
			t.Fatalf("unknown region %q", region)
		}
		return base
	case domain != "":
		return openapicxsdk.URLFromDomain(domain)
	}
	t.Fatal("set CORALOGIX_REGION or CORALOGIX_DOMAIN")
	return ""
}

func sendRecordRequest(base, apiKey, method, path string, body any) (int, any, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, strings.TrimRight(base, "/")+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	parsed := parseBody(raw)
	if parsed == nil {
		parsed = map[string]any{}
	}
	return resp.StatusCode, parsed, nil
}

func capturedID(resp any, k kind) string {
	m, _ := resp.(map[string]any)
	id, _ := m[k.fidelity.captureField].(string)
	return id
}

// normalizeAnswer stores a captured id under its name, then writes {{name}} for each captured
// id and <id> for any other id that the API made. Each run makes new ids.
func normalizeAnswer(t *testing.T, k kind, s fidelityStep, status int, resp any, vars map[string]string) any {
	t.Helper()
	if s.Capture != "" && status == http.StatusOK {
		vars[s.Capture] = capturedID(resp, k)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for name, value := range vars {
		if value != "" {
			text = strings.ReplaceAll(text, value, "{{"+name+"}}")
		}
	}
	text = anyCaseUUIDPattern.ReplaceAllStringFunc(text, func(id string) string {
		if strings.HasPrefix(strings.ToLower(id), fixedIDPrefix) {
			return id
		}
		return "<id>"
	})
	text = redactText(text)
	var out any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}
	return redactFields(out)
}

// The recording goes into a public repository, so it keeps no value that names an account.
var (
	anyCaseUUIDPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	// CompanyId(123456) in an error message becomes CompanyId(<redacted>).
	labeledIDPattern = regexp.MustCompile(`(?i)\b((?:company|team|user|org|account)_?id)\(([^)]*)\)`)
	emailPattern     = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	// A run of five or more digits is a company, team, or user number.
	longNumberPattern = regexp.MustCompile(`\b[0-9]{5,}\b`)
	// Values of these response fields name an account or a single request.
	redactedFieldPattern = regexp.MustCompile(`(?i)^(companyid|teamid|userid|orgid|accountid|email|requestid|traceid|correlationid)$`)
)

// redactText hides account values inside free text, such as an error message.
func redactText(text string) string {
	text = labeledIDPattern.ReplaceAllString(text, "${1}(<redacted>)")
	text = emailPattern.ReplaceAllString(text, "<email>")
	return longNumberPattern.ReplaceAllString(text, "<number>")
}

// redactFields hides the values of response fields that name an account or a request.
func redactFields(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for key, value := range v {
			if redactedFieldPattern.MatchString(key) {
				v[key] = "<redacted>"
				continue
			}
			v[key] = redactFields(value)
		}
	case []any:
		for i := range v {
			v[i] = redactFields(v[i])
		}
	}
	return v
}

func splitQuery(t *testing.T, path string) (string, url.Values) {
	t.Helper()
	p, q, ok := strings.Cut(path, "?")
	if !ok {
		return p, nil
	}
	query, err := url.ParseQuery(q)
	if err != nil {
		t.Fatal(err)
	}
	return p, query
}

func substString(s string, vars map[string]string) string {
	for name, value := range vars {
		s = strings.ReplaceAll(s, "{{"+name+"}}", value)
	}
	return s
}

func substJSON(t *testing.T, v any, vars map[string]string) any {
	t.Helper()
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal([]byte(substString(string(raw), vars)), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func roundTrip(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func compact(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(raw)
}

func TestRedactHidesAccountValues(t *testing.T) {
	text := redactText(`Bad Request: parent folder for CompanyId(4012345) and team_id(77) of admin@example.com, request 123456789`)
	want := `Bad Request: parent folder for CompanyId(<redacted>) and team_id(<redacted>) of <email>, request <number>`
	if text != want {
		t.Fatalf("redactText:\n got %s\nwant %s", text, want)
	}

	got := redactFields(map[string]any{
		"folder": map[string]any{"name": "cxo-fidelity-a", "teamId": 4012345},
		"items":  []any{map[string]any{"requestId": "abc"}},
	})
	wantFields := map[string]any{
		"folder": map[string]any{"name": "cxo-fidelity-a", "teamId": "<redacted>"},
		"items":  []any{map[string]any{"requestId": "<redacted>"}},
	}
	if !reflect.DeepEqual(got, wantFields) {
		t.Fatalf("redactFields:\n got %v\nwant %v", got, wantFields)
	}
}
