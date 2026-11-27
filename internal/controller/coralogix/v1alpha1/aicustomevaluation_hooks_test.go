// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The SDK checks that an id in a path is a UUID.
const testAICustomEvaluationID = "3a9b8c7d-6e5f-4a3b-2c1d-0e9f8a7b6c5d"

// aiCustomEvaluationHookAPI answers the calls of the hooks: the List of AI applications, the List
// of custom evaluations with the links it holds, link, unlink, and update. It records each call.
type aiCustomEvaluationHookAPI struct {
	applications [][3]string // id, application, subsystem
	links        []string
	calls        []string
}

func (a *aiCustomEvaluationHookAPI) serve(t *testing.T) *AICustomEvaluationReconciler {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		path := strings.TrimPrefix(r.URL.Path, "/mgmt/openapi/5")
		call := r.Method + " " + path
		if r.URL.RawQuery != "" {
			call += "?" + r.URL.RawQuery
		}
		if len(body) != 0 {
			call += " " + strings.TrimSpace(string(body))
		}
		a.calls = append(a.calls, call)
		w.Header().Set("Content-Type", "application/json")
		var answer any
		switch {
		case r.Method == http.MethodGet && path == "/ai/applications/v3":
			size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
			offset, _ := strconv.Atoi(r.URL.Query().Get("page_offset"))
			page := []map[string]string{}
			for i := offset; i < len(a.applications) && i < offset+size; i++ {
				app := a.applications[i]
				page = append(page, map[string]string{"id": app[0], "application": app[1], "subsystem": app[2]})
			}
			answer = map[string]any{"aiApplications": page}
		case r.Method == http.MethodGet && path == "/ai/custom-evaluations/v3":
			answer = map[string]any{"items": []map[string]any{{"id": testAICustomEvaluationID, "name": "n", "applicationIds": a.links}}}
		case r.Method == http.MethodPatch:
			answer = map[string]any{"item": map[string]any{"id": testAICustomEvaluationID, "applicationIds": []string{}}}
		case strings.Contains(path, "/applications/"):
			applicationID := path[strings.LastIndex(path, "/")+1:]
			if r.Method == http.MethodPost {
				a.links = append(a.links, applicationID)
			} else {
				a.links = slices.DeleteFunc(a.links, func(id string) bool { return id == applicationID })
			}
			answer = map[string]any{"id": testAICustomEvaluationID, "applicationId": applicationID}
		default:
			w.WriteHeader(http.StatusNotFound)
			answer = map[string]any{"code": 404, "message": "Not Found: Not Found"}
		}
		require.NoError(t, json.NewEncoder(w).Encode(answer))
	}))
	t.Cleanup(server.Close)
	cs := cxsdk.NewClientSet(cxsdk.NewConfigBuilder().WithURL(server.URL + "/mgmt/openapi/5").Build())
	return &AICustomEvaluationReconciler{Client: cs.AIEvaluations(), ClientSet: cs}
}

func newTestAICustomEvaluation(applications ...string) *coralogixv1alpha1.AICustomEvaluation {
	cr := &coralogixv1alpha1.AICustomEvaluation{}
	cr.UID = types.UID("uid-" + strings.Join(applications, "-"))
	cr.Status.ID = ptr.To(testAICustomEvaluationID)
	for _, application := range applications {
		cr.Spec.Applications = append(cr.Spec.Applications, coralogixv1alpha1.AICustomEvaluationApplicationSelector{Application: application, Subsystem: "sub"})
	}
	return cr
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return string(raw)
}

// beforeCreate sets the criteria and the application ids. It builds each field whole, so a second
// run on the same body gives the same body.
func TestAICustomEvaluationBeforeCreateIsSafeToRepeat(t *testing.T) {
	api := &aiCustomEvaluationHookAPI{applications: [][3]string{{"id-b", "app-b", "sub"}, {"id-a", "app-a", "sub"}}}
	r := api.serve(t)
	cr := newTestAICustomEvaluation("app-b", "app-a")
	cr.Spec.Criteria = &coralogixv1alpha1.AICustomEvaluationCriteria{
		Acceptable: &coralogixv1alpha1.AICustomEvaluationCriterion{Flags: ptr.To("ok"), Examples: []string{"good"}},
		Prohibited: &coralogixv1alpha1.AICustomEvaluationCriterion{Examples: []string{"bad one", "bad two"}},
	}
	body := &aievaluations.AiEvaluationsServiceCreateCustomEvaluationRequest{}
	require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), cr, body))
	want := `{"applicationIds":["id-a","id-b"],"examples":[{"conversation":"good","score":"1"},{"conversation":"bad one","score":"0"},{"conversation":"bad two","score":"0"}],"safe":"ok","violates":""}`
	require.JSONEq(t, want, jsonOf(t, body))

	require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), cr, body), "a second run")
	require.JSONEq(t, want, jsonOf(t, body))
}

// With no criteria and no applications, beforeCreate sends empty flags and an empty example list,
// as the released kind did, and calls no API.
func TestAICustomEvaluationBeforeCreateWithoutCriteria(t *testing.T) {
	api := &aiCustomEvaluationHookAPI{}
	r := api.serve(t)
	body := &aievaluations.AiEvaluationsServiceCreateCustomEvaluationRequest{}
	require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), newTestAICustomEvaluation(), body))
	require.JSONEq(t, `{"examples":[],"safe":"","violates":""}`, jsonOf(t, body))
	require.Empty(t, api.calls)
}

// A selector must match exactly one application. The error comes before any write.
func TestAICustomEvaluationSelectorsMatchOneApplication(t *testing.T) {
	api := &aiCustomEvaluationHookAPI{applications: [][3]string{{"id-1", "twin", "sub"}, {"id-2", "twin", "sub"}}}
	r := api.serve(t)
	body := &aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest{}
	err := r.beforeUpdate(context.Background(), logr.Discard(), newTestAICustomEvaluation("twin"), body)
	require.ErrorContains(t, err, "ambiguous AI application selector")
	err = r.beforeUpdate(context.Background(), logr.Discard(), newTestAICustomEvaluation("missing"), body)
	require.ErrorContains(t, err, `no AI application named "missing" with subsystem "sub"`)
	for _, call := range api.calls {
		require.True(t, strings.HasPrefix(call, "GET /ai/applications/v3"), "only the List of applications: %s", call)
	}
}

// page_offset counts applications, so the second page starts after the first 200. An application
// on the second page resolves to one id.
func TestAICustomEvaluationApplicationsPageByItemOffset(t *testing.T) {
	api := &aiCustomEvaluationHookAPI{}
	for i := 0; i < 250; i++ {
		api.applications = append(api.applications, [3]string{fmt.Sprintf("id-%03d", i), fmt.Sprintf("app-%03d", i), "sub"})
	}
	r := api.serve(t)
	ids, err := r.resolveAICustomEvaluationApplications(context.Background(), newTestAICustomEvaluation("app-000", "app-249").Spec.Applications)
	require.NoError(t, err)
	require.Equal(t, []string{"id-000", "id-249"}, ids)
	require.Equal(t, []string{
		"GET /ai/applications/v3?page_offset=0&page_size=200",
		"GET /ai/applications/v3?page_offset=200&page_size=200",
	}, api.calls)
}

// afterUpdate clears the examples when the CR has none, reads the links, and links and unlinks
// to match the selectors. A second run finds the links right and changes nothing.
func TestAICustomEvaluationAfterUpdateIsSafeToRepeat(t *testing.T) {
	api := &aiCustomEvaluationHookAPI{applications: [][3]string{{"00000000-0000-4000-8000-0000000000c1", "app-a", "sub"}, {"00000000-0000-4000-8000-0000000000c2", "app-b", "sub"}}, links: []string{"00000000-0000-4000-8000-0000000000c3", "00000000-0000-4000-8000-0000000000c1"}}
	r := api.serve(t)
	cr := newTestAICustomEvaluation("app-a", "app-b")
	body := &aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest{}
	require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), cr, body))
	require.JSONEq(t, `{"examples":[],"safe":"","violates":""}`, jsonOf(t, body))
	api.calls = nil

	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, nil))
	require.Equal(t, []string{
		"PATCH /ai/custom-evaluations/v3/" + testAICustomEvaluationID + ` {"examples":[],"updateMask":"examples"}`,
		"GET /ai/custom-evaluations/v3",
		"POST /ai/custom-evaluations/v3/" + testAICustomEvaluationID + "/applications/00000000-0000-4000-8000-0000000000c2",
		"DELETE /ai/custom-evaluations/v3/" + testAICustomEvaluationID + "/applications/00000000-0000-4000-8000-0000000000c3",
	}, api.calls)
	require.ElementsMatch(t, []string{"00000000-0000-4000-8000-0000000000c1", "00000000-0000-4000-8000-0000000000c2"}, api.links)

	// The next reconcile: beforeUpdate resolves again, and afterUpdate makes no link change.
	api.calls = nil
	require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), cr, body))
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, nil), "a second run")
	require.Equal(t, []string{
		"GET /ai/applications/v3?page_offset=0&page_size=200",
		"PATCH /ai/custom-evaluations/v3/" + testAICustomEvaluationID + ` {"examples":[],"updateMask":"examples"}`,
		"GET /ai/custom-evaluations/v3",
	}, api.calls)
	require.ElementsMatch(t, []string{"00000000-0000-4000-8000-0000000000c1", "00000000-0000-4000-8000-0000000000c2"}, api.links)
}

// With examples, afterUpdate sends no second Update. Without beforeUpdate in the same reconcile, it
// resolves the selectors itself.
func TestAICustomEvaluationAfterUpdateKeepsExamples(t *testing.T) {
	api := &aiCustomEvaluationHookAPI{applications: [][3]string{{"00000000-0000-4000-8000-0000000000c1", "app-a", "sub"}}, links: []string{"00000000-0000-4000-8000-0000000000c1"}}
	r := api.serve(t)
	cr := newTestAICustomEvaluation("app-a")
	cr.Spec.Criteria = &coralogixv1alpha1.AICustomEvaluationCriteria{Acceptable: &coralogixv1alpha1.AICustomEvaluationCriterion{Examples: []string{"good"}}}
	require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), cr, nil))
	require.Equal(t, []string{"GET /ai/applications/v3?page_offset=0&page_size=200", "GET /ai/custom-evaluations/v3"}, api.calls)
}
