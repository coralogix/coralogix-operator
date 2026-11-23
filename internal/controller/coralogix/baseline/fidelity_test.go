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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
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
	// captureField is the response field that holds the id of a new resource. A dotted path
	// reads a nested field: "e2m.id" is the id in {"e2m": {"id": "..."}}.
	captureField string
	// deletePath returns the path that deletes a resource. The recorder calls it for each
	// captured id when it ends, so a failed run leaves nothing behind.
	deletePath func(id string) string
	// numericID is true when Create returns an integer id. The recorder stores the integer as a
	// path variable and masks it in response bodies.
	numericID bool
	// cleanup returns the requests that remove a resource, for an API that has no DELETE, such as
	// one that archives with a POST. The recorder sends them in order instead of a DELETE of
	// deletePath. A 404 of any request is no problem: a case already removed the resource.
	cleanup func(id string) []fidelityStep
	// overwritesTeam is true for a kind whose API replaces every resource of the kind in the
	// team, such as the TCO policies. Its cases delete what the team had. The recorder runs them
	// only when RECORD_FIDELITY_OVERWRITE_TEAM names the kind.
	overwritesTeam bool
	// normalize rewrites an answer of the kind before it is compared or recorded, for a value that
	// the fake cannot make and the operator never reads, such as text that the API renders from a
	// template. It runs on the answers of the fake and of the real API alike.
	normalize func(resp any) any
	// teamState is set for a kind that has one value per team, such as the archive target. The
	// recorder reads the value before the first case and writes it back when it ends.
	teamState *teamState
	// recordVars returns values that the cases name as {{name}} from their first step, such as an
	// AI application of the team, which the API cannot create. It reads them from the real API.
	// fakeVars are the same names for the fake. The recorder writes {{name}} for each value, so
	// the two compare equal.
	recordVars func(t *testing.T, base, apiKey string) map[string]string
	fakeVars   map[string]string
}

// teamState reads the one value of a team and writes it back.
type teamState struct {
	// readPath is the path of the GET that returns the value.
	readPath string
	// restore returns the request that writes back the value that the GET returned.
	restore func(original any) fidelityStep
	// require checks the team before the first case, with read sending a GET. A case can change
	// what restore does not write back, so the recorder runs only on a team where that change is
	// none: it stops with the error of require.
	require func(read func(path string) (any, error)) error
}

// startVars returns a copy of vars, the values that each case starts with.
func startVars(vars map[string]string) map[string]string {
	out := map[string]string{}
	for name, value := range vars {
		out[name] = value
	}
	return out
}

// cleanupSteps returns the requests that remove the resource with the id: the kind's cleanup, or
// a DELETE of deletePath.
func (c fidelityConfig) cleanupSteps(id string) []fidelityStep {
	if c.cleanup != nil {
		return c.cleanup(id)
	}
	return []fidelityStep{{Method: http.MethodDelete, Path: c.deletePath(id)}}
}

type fidelityStep struct {
	Method string `json:"method"`
	// Path is relative to the OpenAPI base URL. {{name}} is a captured id.
	Path string `json:"path"`
	Body any    `json:"body,omitempty"`
	// Capture names the id in the response, for use in later steps.
	Capture string `json:"capture,omitempty"`
	// CaptureFields names values of the response, by a dotted path, for use in later steps as
	// {{name}}. The recording shows each value as {{name}}, so a value of the team, such as its
	// bucket, stays out of the file.
	CaptureFields map[string]string `json:"captureFields,omitempty"`
	// AllowForbidden records a 403 answer instead of stopping. By default the recorder reads a 403
	// as a missing permission. Set it on a step where the API itself refuses with 403.
	AllowForbidden bool `json:"allowForbidden,omitempty"`
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
					vars := runVars(k.fidelity.fakeVars, fakeRunToken)
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
	if k.fidelity.overwritesTeam && os.Getenv("RECORD_FIDELITY_OVERWRITE_TEAM") != name {
		t.Fatalf("the %s cases replace every %s resource of the team. Use a team that holds nothing you need, and set RECORD_FIDELITY_OVERWRITE_TEAM=%s.", name, name, name)
	}
	base := recordBaseURL(t)
	apiKey := os.Getenv("CORALOGIX_API_KEY")
	if apiKey == "" {
		t.Fatal("CORALOGIX_API_KEY is empty")
	}

	if ts := k.fidelity.teamState; ts != nil {
		if ts.require != nil {
			read := func(path string) (any, error) {
				status, resp, err := sendRecordRequest(base, apiKey, http.MethodGet, path, nil)
				if err == nil && !succeeded(status) {
					err = fmt.Errorf("GET %s returned %d", path, status)
				}
				return resp, err
			}
			if err := ts.require(read); err != nil {
				t.Fatalf("the team cannot record %s: %v", k.name, err)
			}
		}
		status, original, err := sendRecordRequest(base, apiKey, http.MethodGet, ts.readPath, nil)
		if err != nil || !succeeded(status) {
			t.Fatalf("read the team value before the cases: GET %s returned %d: %v", ts.readPath, status, err)
		}
		// t.Cleanup runs last in, first out, so this runs after the deletes below.
		t.Cleanup(func() {
			s := ts.restore(original)
			status, resp, err := sendRecordRequest(base, apiKey, s.Method, s.Path, s.Body)
			if err != nil || !succeeded(status) {
				t.Errorf("cleanup: %s %s returned %d %s: %v; write back the team value by hand: %s", s.Method, s.Path, status, compact(resp), err, compact(original))
			}
		})
	}

	var created []string
	t.Cleanup(func() {
		// The last resource goes first, so a child is deleted before its parent.
		for i := len(created) - 1; i >= 0; i-- {
			for _, s := range k.fidelity.cleanupSteps(created[i]) {
				status, resp, err := sendRecordRequest(base, apiKey, s.Method, s.Path, s.Body)
				if problem := cleanupProblem(status, resp, err); problem != "" {
					t.Errorf("cleanup: %s %s %s; delete it by hand", s.Method, s.Path, problem)
				}
			}
		}
	})

	var initial map[string]string
	if k.fidelity.recordVars != nil {
		initial = k.fidelity.recordVars(t, base, apiKey)
	}
	run := newRunToken()
	out := fidelityRecorded{}
	for _, c := range loadCases(t, k).Cases {
		rec := recordedCase{Name: c.Name}
		vars := runVars(initial, run)
		for _, s := range c.Steps {
			status, resp, err := sendRecordRequest(base, apiKey, s.Method, substString(s.Path, vars), substJSON(t, s.Body, vars))
			if err != nil {
				t.Fatalf("%s: %s %s: %v", c.Name, s.Method, s.Path, err)
			}
			if status == http.StatusUnauthorized || (status == http.StatusForbidden && !s.AllowForbidden) {
				t.Fatalf("%s: %s %s returned %d. The API key needs permission to manage %s.", c.Name, s.Method, s.Path, status, k.name)
			}
			if s.Capture != "" && succeeded(status) {
				if id := capturedID(resp, k); id != "" {
					created = append(created, id)
				}
			}
			normalized := normalizeAnswer(t, k, s, status, resp, vars)
			if name := leakedStartingValue(normalized, substJSON(t, s.Body, vars), initial); name != "" {
				t.Fatalf("%s: %s %s: the answer holds the value of {{%s}} outside its field, for example in an error text; nothing was written. The value can be private: choose another one", c.Name, s.Method, s.Path, name)
			}
			rec.Steps = append(rec.Steps, fidelityAnswer{Status: status, Response: normalized})
		}
		out.Cases = append(out.Cases, rec)
	}
	recorded := marshalGolden(t, out)
	checkNoSecrets(t, recorded, apiKey, base)
	if err := os.WriteFile(filepath.Join(fidelityDir(k), "recorded.json"), recorded, 0o644); err != nil {
		t.Fatal(err)
	}
}

// A case can write {{run}} in a path or a body, such as in a name that must be new on each
// recording: the API of some kinds keeps the name of a removed resource, so a second recording with
// the same name fails. The recorder sends a new token on each run, and the fake gets a fixed one.
// normalizeAnswer writes {{run}} for the token in both, so the answers compare. A token has
// letters that are not hex digits, so it is never part of a UUID.
const fakeRunToken = "fakerunq"

const runTokenLetters = "ghjkmnpqrstvwxyz"

// newRunToken returns a token of eight letters from runTokenLetters.
func newRunToken() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = runTokenLetters[int(b[i])%len(runTokenLetters)]
	}
	return string(b[:])
}

// runVars returns the variables that a case starts with: a copy of start, and the run token.
func runVars(start map[string]string, token string) map[string]string {
	vars := startVars(start)
	vars["run"] = token
	return vars
}

// cleanupProblem says why a cleanup delete failed, or "" when it worked. A 404 is no problem: a
// case already deleted the resource. Some APIs answer a second delete with a 400 that says the
// resource does not exist. That is no problem either.
func cleanupProblem(status int, resp any, err error) string {
	switch {
	case err != nil:
		return fmt.Sprintf("failed: %v", err)
	case succeeded(status), status == http.StatusNotFound:
		return ""
	case status == http.StatusBadRequest && strings.Contains(compact(resp), "does not exist"):
		return ""
	default:
		return fmt.Sprintf("returned %d", status)
	}
}

// succeeded is true for every 2xx answer. Some Create operations answer 201.
func succeeded(status int) bool {
	return status >= 200 && status < 300
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

// capturedID reads the id of a new resource from a response, or "" when the path has no string.
func capturedID(resp any, k kind) string {
	if k.fidelity.numericID {
		if id, ok := valueAt(resp, k.fidelity.captureField).(float64); ok {
			return strconv.FormatInt(int64(id), 10)
		}
	}
	return stringAt(resp, k.fidelity.captureField)
}

// stringAt reads the string at a dotted path of a response, or "" when the path has no string.
func stringAt(resp any, path string) string {
	value, _ := valueAt(resp, path).(string)
	return value
}

func valueAt(resp any, path string) any {
	current := resp
	for _, key := range strings.Split(path, ".") {
		m, _ := current.(map[string]any)
		current = m[key]
	}
	return current
}

// normalizeAnswer stores a captured id under its name, then writes {{name}} for each captured
// id and <id> for any other id that the API made. Each run makes new ids.
func normalizeAnswer(t *testing.T, k kind, s fidelityStep, status int, resp any, vars map[string]string) any {
	t.Helper()
	if s.Capture != "" && succeeded(status) {
		vars[s.Capture] = capturedID(resp, k)
	}
	if succeeded(status) {
		for name, path := range s.CaptureFields {
			vars[name] = stringAt(resp, path)
		}
	}
	// A starting value, such as an application name, is replaced only as the whole value of the
	// field of its name. A substring replace would rewrite a key that contains it, and two starting
	// values that are equal would get the same placeholder.
	resp = replaceFieldVars(roundTrip(t, resp), k.fidelity.fakeVars, vars)
	if k.fidelity.numericID {
		resp = maskNumericIDs(resp, vars)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for name, value := range vars {
		if _, field := k.fidelity.fakeVars[name]; field || value == "" {
			continue
		}
		if k.fidelity.numericID {
			if _, err := strconv.ParseUint(value, 10, 32); err == nil {
				continue // numeric ids were replaced in the decoded response
			}
		}
		text = strings.ReplaceAll(text, value, "{{"+name+"}}")
	}
	text = anyCaseUUIDPattern.ReplaceAllStringFunc(text, func(id string) string {
		if strings.HasPrefix(strings.ToLower(id), fixedIDPrefix) {
			return id
		}
		return "<id>"
	})
	var out any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatal(err)
	}
	out = redactFields(redactStrings(canonicalJSONStrings(out)))
	if k.fidelity.normalize != nil {
		out = k.fidelity.normalize(out)
	}
	return out
}

// maskNumericIDs replaces captured integer ids before JSON text substitution. A bare number
// cannot hold the string placeholder in JSON.
func maskNumericIDs(v any, vars map[string]string) any {
	switch v := v.(type) {
	case float64:
		if v != math.Trunc(v) {
			return v
		}
		for name, value := range vars {
			if value != "" && strconv.FormatInt(int64(v), 10) == value {
				return "{{" + name + "}}"
			}
		}
	case string:
		for name, value := range vars {
			if _, err := strconv.ParseUint(value, 10, 32); err == nil {
				v = regexp.MustCompile(`\b`+regexp.QuoteMeta(value)+`\b`).ReplaceAllString(v, "{{"+name+"}}")
			}
		}
		return v
	case map[string]any:
		for key, value := range v {
			v[key] = maskNumericIDs(value, vars)
		}
	case []any:
		for i := range v {
			v[i] = maskNumericIDs(v[i], vars)
		}
	}
	return v
}

// replaceFieldVars writes {{name}} for each string field named after a starting value that holds
// exactly that value.
func replaceFieldVars(v any, names map[string]string, vars map[string]string) any {
	switch v := v.(type) {
	case map[string]any:
		for key, value := range v {
			if _, ok := names[key]; ok {
				if text, ok := value.(string); ok && text != "" && text == vars[key] {
					v[key] = "{{" + key + "}}"
					continue
				}
			}
			v[key] = replaceFieldVars(value, names, vars)
		}
	case []any:
		for i := range v {
			v[i] = replaceFieldVars(v[i], names, vars)
		}
	}
	return v
}

// leakedStartingValue returns the name of a starting value that a normalized answer still holds,
// or "". The value can be private, such as an application name of the team. It reads string
// values only, not keys. It ignores placeholders, and strings that the request of the step sent,
// such as a topic of a config that the answer repeats.
func leakedStartingValue(answer, request any, vars map[string]string) string {
	sent := map[string]bool{}
	var collect func(v any)
	collect = func(v any) {
		switch v := v.(type) {
		case string:
			sent[v] = true
		case map[string]any:
			for _, value := range v {
				collect(value)
			}
		case []any:
			for _, value := range v {
				collect(value)
			}
		}
	}
	collect(request)
	names := make([]string, 0, len(vars))
	for name, value := range vars {
		if value != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var leaked func(v any) string
	leaked = func(v any) string {
		switch v := v.(type) {
		case string:
			if sent[v] {
				return ""
			}
			text := placeholderPattern.ReplaceAllString(v, "")
			for _, name := range names {
				if regexp.MustCompile(`\b` + regexp.QuoteMeta(vars[name]) + `\b`).MatchString(text) {
					return name
				}
			}
		case map[string]any:
			for _, value := range v {
				if name := leaked(value); name != "" {
					return name
				}
			}
		case []any:
			for _, value := range v {
				if name := leaked(value); name != "" {
					return name
				}
			}
		}
		return ""
	}
	return leaked(answer)
}

var placeholderPattern = regexp.MustCompile(`\{\{[^}]*\}\}`)

// redactStrings hides account values inside each string value. A number that is not in a string
// stays as it is: a limit of 30000 is a setting, and "<number>" would not be JSON.
func redactStrings(v any) any {
	switch v := v.(type) {
	case string:
		return redactText(v)
	case map[string]any:
		for key, value := range v {
			v[key] = redactStrings(value)
		}
	case []any:
		for i := range v {
			v[i] = redactStrings(v[i])
		}
	}
	return v
}

// The recording goes into a public repository, so it keeps no value that names an account.
var (
	anyCaseUUIDPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	// CompanyId(123456) in an error message becomes CompanyId(<redacted>).
	labeledIDPattern = regexp.MustCompile(`(?i)\b((?:company|team|user|org|account)_?id)\(([^)]*)\)`)
	emailPattern     = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	// A time that the API sets, such as 2026-01-02T03:04:05.678Z. Each run gets a new one.
	timestampPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`)
	// A run of five or more digits is a company, team, or user number.
	longNumberPattern = regexp.MustCompile(`\b[0-9]{5,}\b`)
	// Values of these response fields name an account, a person, or a single request.
	redactedFieldPattern = regexp.MustCompile(`(?i)^(companyid|teamid|userid|orgid|accountid|externalid|email|requestid|traceid|correlationid|authorid|authorname|updaterauthorid|updatername|lockerauthorid|lockername)$`)
)

// redactText hides account values inside free text, such as an error message. It also writes a
// no-break space as a space: protojson picks one of the two at random in its error messages, so
// two recordings of one answer differ.
func redactText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = labeledIDPattern.ReplaceAllString(text, "${1}(<redacted>)")
	text = emailPattern.ReplaceAllString(text, "<email>")
	text = timestampPattern.ReplaceAllString(text, "<time>")
	return replaceLongNumbers(text)
}

// replaceLongNumbers hides each long number, but not a digit group of a UUID, such as the
// 00000000 of a fixed test id.
func replaceLongNumbers(text string) string {
	var b strings.Builder
	last := 0
	for _, m := range longNumberPattern.FindAllStringIndex(text, -1) {
		if (m[0] > 0 && text[m[0]-1] == '-') || (m[1] < len(text) && text[m[1]] == '-') {
			continue
		}
		b.WriteString(text[last:m[0]])
		b.WriteString("<number>")
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

// canonicalJSONStrings rewrites each string that holds a JSON object with sorted keys. The API
// can answer the same stored JSON text with its keys in another order on each call, such as the
// default access policy of a dashboard.
func canonicalJSONStrings(v any) any {
	switch v := v.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if !strings.HasPrefix(trimmed, "{") {
			return v
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return v
		}
		raw, err := json.Marshal(parsed)
		if err != nil {
			return v
		}
		return string(raw)
	case map[string]any:
		for key, value := range v {
			v[key] = canonicalJSONStrings(value)
		}
	case []any:
		for i := range v {
			v[i] = canonicalJSONStrings(v[i])
		}
	}
	return v
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

// substJSON writes each {{name}} of a request body. It substitutes inside the decoded keys and
// string values, so a value with a quote or a backslash, such as an application name, stays the
// exact string that it is.
func substJSON(t *testing.T, v any, vars map[string]string) any {
	t.Helper()
	if v == nil {
		return nil
	}
	var subst func(v any) any
	subst = func(v any) any {
		switch v := v.(type) {
		case string:
			return substString(v, vars)
		case map[string]any:
			out := make(map[string]any, len(v))
			for key, value := range v {
				out[substString(key, vars)] = subst(value)
			}
			return out
		case []any:
			out := make([]any, len(v))
			for i, value := range v {
				out[i] = subst(value)
			}
			return out
		}
		return v
	}
	return subst(roundTrip(t, v))
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
	text := redactText(`Bad Request: parent folder for CompanyId(4012345) and team_id(77) of admin@example.com, request 123456789, folder 00000000-0000-4000-8000-00000000dead, created 2026-01-02T03:04:05.678Z`)
	want := `Bad Request: parent folder for CompanyId(<redacted>) and team_id(<redacted>) of <email>, request <number>, folder 00000000-0000-4000-8000-00000000dead, created <time>`
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

func TestRedactTextWritesANoBreakSpaceAsASpace(t *testing.T) {
	if got := redactText("proto:\u00a0(line 1:2)"); got != "proto: (line 1:2)" {
		t.Fatalf("redactText = %q", got)
	}
}

func TestCanonicalJSONStringsSortsKeys(t *testing.T) {
	got := canonicalJSONStrings(map[string]any{
		"policy": `{"b":1,"a":{"d":2,"c":3}}`,
		"text":   "{not json",
		"list":   []any{`{"y":1,"x":2}`},
	})
	want := map[string]any{
		"policy": `{"a":{"c":3,"d":2},"b":1}`,
		"text":   "{not json",
		"list":   []any{`{"x":2,"y":1}`},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonicalJSONStrings = %v, want %v", got, want)
	}
}

func TestSucceededAcceptsEvery2xx(t *testing.T) {
	for status, want := range map[int]bool{199: false, 200: true, 201: true, 204: true, 299: true, 300: false, 404: false} {
		if got := succeeded(status); got != want {
			t.Errorf("succeeded(%d) = %v, want %v", status, got, want)
		}
	}
}

func TestNormalizeAnswerCapturesIDOf201(t *testing.T) {
	k := kinds["viewfolder"]
	vars := map[string]string{}
	resp := map[string]any{"id": "3dc02998-0b50-4ea8-b68a-4779d716fa1f", "name": "cxo-fidelity-a"}
	got := normalizeAnswer(t, k, fidelityStep{Capture: "id"}, http.StatusCreated, resp, vars)
	if vars["id"] != "3dc02998-0b50-4ea8-b68a-4779d716fa1f" {
		t.Fatalf("id not captured from a 201 answer: vars = %v", vars)
	}
	if want := map[string]any{"id": "{{id}}", "name": "cxo-fidelity-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAnswer = %v, want %v", got, want)
	}
}

// A starting value is replaced by field, so a short value does not rewrite a key, and two equal
// values keep their own placeholders.
func TestNormalizeAnswerReplacesStartingValuesByField(t *testing.T) {
	k := kind{fidelity: fidelityConfig{fakeVars: map[string]string{"application": "a", "subsystem": "s"}}}
	vars := map[string]string{"application": "app", "subsystem": "app"}
	resp := map[string]any{"aiEvaluation": map[string]any{"application": "app", "subsystem": "app", "note": "apple app"}}
	got := normalizeAnswer(t, k, fidelityStep{}, http.StatusOK, resp, vars)
	want := map[string]any{"aiEvaluation": map[string]any{"application": "{{application}}", "subsystem": "{{subsystem}}", "note": "apple app"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAnswer = %v, want %v", got, want)
	}
}

// A starting value leaks only when an answer string that the request did not send holds it. Keys,
// placeholders, and repeated request values are no leak.
func TestLeakedStartingValue(t *testing.T) {
	vars := map[string]string{"application": "application", "subsystem": "billing"}
	request := map[string]any{"application": "application", "subsystem": "billing", "config": map[string]any{"allowedTopics": map[string]any{"topics": []any{"billing"}}}}
	clean := map[string]any{"aiEvaluation": map[string]any{"application": "{{application}}", "subsystem": "{{subsystem}}", "config": map[string]any{"allowedTopics": map[string]any{"topics": []any{"billing"}}}}}
	if name := leakedStartingValue(clean, request, vars); name != "" {
		t.Fatalf("a normalized answer leaks {{%s}}", name)
	}
	vars = map[string]string{"application": "team-app-20"}
	leak := map[string]any{"message": "Not Found: application team-app-20 not found"}
	if name := leakedStartingValue(leak, map[string]any{"application": "team-app-20"}, vars); name != "application" {
		t.Fatalf("an error text that names the application is not reported: got %q", name)
	}
}

// A value with a quote or a backslash is sent as the exact string.
func TestSubstJSONKeepsQuotesAndBackslashes(t *testing.T) {
	vars := map[string]string{"application": `app"na\me`, "id": "3dc02998-0b50-4ea8-b68a-4779d716fa1f"}
	body := map[string]any{"application": "{{application}}", "note": "of {{application}}", "ids": []any{"{{id}}"}}
	got := substJSON(t, body, vars)
	want := map[string]any{"application": `app"na\me`, "note": `of app"na\me`, "ids": []any{"3dc02998-0b50-4ea8-b68a-4779d716fa1f"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("substJSON = %v, want %v", got, want)
	}
}

func TestCleanupProblemReportsFailedDeletes(t *testing.T) {
	for _, c := range []struct {
		status int
		resp   any
		err    error
		want   string
	}{
		{status: 200, want: ""},
		{status: 204, want: ""},
		{status: 404, want: ""},
		{status: 400, resp: map[string]any{"message": "Bad Request: L2M does not exist"}, want: ""},
		{status: 400, resp: map[string]any{"message": "Bad Request: invalid id"}, want: "returned 400"},
		{status: 500, want: "returned 500"},
		{status: 403, want: "returned 403"},
		{err: errors.New("connection reset"), want: "failed: connection reset"},
	} {
		if got := cleanupProblem(c.status, c.resp, c.err); got != c.want {
			t.Errorf("cleanupProblem(%d, %v) = %q, want %q", c.status, c.err, got, c.want)
		}
	}
}

func TestCapturedIDReadsADottedPath(t *testing.T) {
	for _, c := range []struct {
		field string
		resp  any
		want  string
	}{
		{"id", map[string]any{"id": "a"}, "a"},
		{"e2m.id", map[string]any{"e2m": map[string]any{"id": "b"}}, "b"},
		{"e2m.id", map[string]any{"id": "top-level only"}, ""},
		{"e2m.id", map[string]any{"e2m": "not an object"}, ""},
		{"e2m.id", map[string]any{"e2m": map[string]any{"id": 7}}, ""},
		{"e2m.id", nil, ""},
	} {
		k := kind{fidelity: fidelityConfig{captureField: c.field}}
		if got := capturedID(c.resp, k); got != c.want {
			t.Errorf("capturedID(%v, %q) = %q, want %q", c.resp, c.field, got, c.want)
		}
	}
}

func TestNormalizeAnswerCapturesANestedID(t *testing.T) {
	k := kind{fidelity: fidelityConfig{captureField: "e2m.id"}}
	vars := map[string]string{}
	id := "3dc02998-0b50-4ea8-b68a-4779d716fa1f"
	resp := map[string]any{"e2m": map[string]any{"id": id, "createTime": "2026-01-02T03:04:05.678Z"}}
	got := normalizeAnswer(t, k, fidelityStep{Capture: "id"}, http.StatusOK, resp, vars)
	if vars["id"] != id {
		t.Fatalf("id not captured from a nested field: vars = %v", vars)
	}
	want := map[string]any{"e2m": map[string]any{"id": "{{id}}", "createTime": "<time>"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAnswer = %v, want %v", got, want)
	}
}

func TestNormalizeAnswerKeepsANumberThatIsNotInAString(t *testing.T) {
	k := kind{fidelity: fidelityConfig{captureField: "e2m.id"}}
	resp := map[string]any{"e2m": map[string]any{"permutations": map[string]any{"limit": float64(30000)}, "note": "request 123456789"}}
	got := normalizeAnswer(t, k, fidelityStep{}, http.StatusOK, resp, map[string]string{})
	want := map[string]any{"e2m": map[string]any{"permutations": map[string]any{"limit": float64(30000)}, "note": "request <number>"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAnswer = %v, want %v", got, want)
	}
}

func TestNormalizeAnswerCapturesFields(t *testing.T) {
	vars := map[string]string{}
	resp := map[string]any{"target": map[string]any{"s3": map[string]any{"bucket": "team-bucket", "region": "eu-north-1", "externalId": "eu2-4012345"}}}
	step := fidelityStep{CaptureFields: map[string]string{"bucket": "target.s3.bucket", "region": "target.s3.region"}}
	got := normalizeAnswer(t, kind{}, step, http.StatusOK, resp, vars)
	if vars["bucket"] != "team-bucket" || vars["region"] != "eu-north-1" {
		t.Fatalf("fields not captured: vars = %v", vars)
	}
	want := map[string]any{"target": map[string]any{"s3": map[string]any{"bucket": "{{bucket}}", "region": "{{region}}", "externalId": "<redacted>"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAnswer = %v, want %v", got, want)
	}

	failed := map[string]string{}
	normalizeAnswer(t, kind{}, step, http.StatusBadRequest, resp, failed)
	if len(failed) != 0 {
		t.Fatalf("a failed answer captured fields: %v", failed)
	}
}

func TestRestoreArchiveTargetWritesBackTheTeamTarget(t *testing.T) {
	original := map[string]any{"target": map[string]any{
		"s3":          map[string]any{"bucket": "b", "region": "r", "externalId": "e"},
		"archiveSpec": map[string]any{"isActive": true},
	}}
	got := restoreArchiveTarget(original)
	want := fidelityStep{Method: http.MethodPost, Path: archiveTargetPath, Body: map[string]any{
		"isActive": true, "s3": map[string]any{"bucket": "b", "region": "r", "externalId": "e"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restoreArchiveTarget = %+v, want %+v", got, want)
	}
}

func TestRequireArchiveTeam(t *testing.T) {
	reader := func(target, tags any) func(string) (any, error) {
		return func(path string) (any, error) {
			if path == archiveTagsEnabledURL {
				return tags, nil
			}
			return target, nil
		}
	}
	s3 := map[string]any{"target": map[string]any{"s3": map[string]any{"bucket": "b"}}}
	on, off := map[string]any{"enableTags": true}, map[string]any{"enableTags": false}
	if err := requireArchiveTeam(reader(s3, on)); err != nil {
		t.Errorf("a team with an S3 target and enabled tags: %v", err)
	}
	if err := requireArchiveTeam(reader(map[string]any{"target": map[string]any{}}, on)); err == nil {
		t.Error("a team without an S3 target must be refused")
	}
	if err := requireArchiveTeam(reader(s3, off)); err == nil {
		t.Error("a team without enabled tags must be refused")
	}
}

func TestRunTokenIsNeverPartOfAUUID(t *testing.T) {
	for range 100 {
		token := newRunToken()
		if len(token) != 8 || strings.ContainsAny(token, "0123456789abcdef") {
			t.Fatalf("newRunToken() = %q, want 8 letters that are not hex digits", token)
		}
	}
	if strings.Trim(fakeRunToken, "0123456789abcdef") == "" || !strings.ContainsAny(fakeRunToken, "ghijklmnopqrstuvwxyz") {
		t.Fatalf("fakeRunToken %q must hold a letter that is not a hex digit", fakeRunToken)
	}
}
