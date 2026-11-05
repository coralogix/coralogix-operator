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
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const e2mPath = "/events2metrics/events2metrics/v2"

func init() {
	register(kind{
		name:     "events2metric",
		crdFile:  "coralogix.com_events2metrics.yaml",
		resource: "events2metrics",
		newFake:  newE2MFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.Events2MetricReconciler{Client: cs.Events2Metrics(), Interval: reconcileInterval}
		},
		scenarios: events2MetricScenarios,
		fidelity: fidelityConfig{
			captureField: "e2m.id",
			deletePath:   func(id string) string { return e2mPath + "/" + id },
		},
	})
}

// A spec is built from these parts, so that each scenario shows only what it changes.
const e2mLogsManifest = `apiVersion: coralogix.com/v1alpha1
kind: Events2Metric
metadata:
  name: e2m
spec:
  name: baseline-e2m
  query:
    logs: {}
`

// e2mFullManifest sets every optional field of a logs E2M that a user can clear.
const e2mFullManifest = `apiVersion: coralogix.com/v1alpha1
kind: Events2Metric
metadata:
  name: e2m
spec:
  name: baseline-e2m
  description: Baseline description
  dataSource: system/baseline
  permutationsLimit: 250
  metricLabels:
    - targetLabel: status
      sourceField: status
  metricFields:
    - targetBaseMetricName: latency
      sourceField: latency
      aggregations:
        - aggType: min
          targetMetricName: latency_min
          aggMetadata: {}
  query:
    logs:
      lucene: 'status:200'
      alias: baseline-alias
      applicationNameFilters: [app]
      subsystemNameFilters: [sub]
      severityFilters: [error]
`

// e2mManifestWithFields is a logs E2M with the given metricFields, as YAML text.
func e2mManifestWithFields(metricFields string) string {
	return e2mLogsManifest + "  metricFields:\n" + metricFields
}

func e2mSpec(query map[string]any, fields map[string]any) map[string]any {
	spec := map[string]any{"name": "baseline-e2m", "query": query}
	maps.Copy(spec, fields)
	return spec
}

func e2mAPI(h *harness) *e2mFake {
	h.t.Helper()
	f, ok := h.API().(*e2mFake)
	if !ok {
		h.t.Fatal("expected *e2mFake")
	}
	return f
}

var events2MetricScenarios = map[string]func(h *harness){
	"create-logs-sample": func(h *harness) {
		h.CreateFromFile("create the logs sample", "config/samples/v1alpha1/events2metrics/logs2metric.yaml")
	},

	"create-spans-sample": func(h *harness) {
		h.CreateFromFile("create the spans sample", "config/samples/v1alpha1/events2metrics/spans2metric.yaml")
	},

	"create-minimal": func(h *harness) {
		h.Create("create with a name and an empty logs query", e2mLogsManifest)
	},

	"create-aggregation-types": func(h *harness) {
		h.Create("create with every simple aggregation type", e2mManifestWithFields(`    - targetBaseMetricName: request_count
      sourceField: request_count
      aggregations:
        - {aggType: count, targetMetricName: count_request_count, aggMetadata: {}}
        - {aggType: avg, targetMetricName: avg_request_count, aggMetadata: {}}
        - {aggType: sum, targetMetricName: sum_request_count, aggMetadata: {}}
        - {aggType: min, targetMetricName: min_request_count, aggMetadata: {}}
        - {aggType: max, targetMetricName: max_request_count, aggMetadata: {}}
`))
	},

	"create-samples-and-histogram": func(h *harness) {
		h.Create("create with a samples and a histogram aggregation", e2mManifestWithFields(`    - targetBaseMetricName: latency
      sourceField: latency
      aggregations:
        - aggType: samples
          targetMetricName: samples_latency
          aggMetadata:
            samples:
              sampleType: max
        - aggType: histogram
          targetMetricName: histogram_latency
          aggMetadata:
            histogram:
              buckets: ["0.1", "0.5", "1", "2.5", "5.5", "1e3"]
`))
	},

	"create-empty-aggregations": func(h *harness) {
		h.Create("create with a metric field that has no aggregations", e2mManifestWithFields(`    - targetBaseMetricName: field
      sourceField: field
`))
	},

	"create-severity-filters": func(h *harness) {
		h.Create("create with every severity filter", strings.Replace(e2mLogsManifest, "logs: {}", `logs:
      severityFilters: [debug, verbose, info, warn, error, critical]`, 1))
	},

	"enabled-false": func(h *harness) {
		h.Create("create with a disabled aggregation and one that takes the default", e2mManifestWithFields(`    - targetBaseMetricName: field
      sourceField: field
      aggregations:
        - {aggType: max, enabled: false, targetMetricName: max_field, aggMetadata: {}}
        - {aggType: min, targetMetricName: min_field, aggMetadata: {}}
`))
	},

	"resync": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		h.Reconcile("reconcile again with no change")
		h.Reconcile("reconcile a third time")
	},

	"update": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		h.SetSpec("rename, describe, set the limit, and add labels and a field", e2mSpec(
			map[string]any{"logs": map[string]any{"lucene": "status:500", "alias": "errors"}},
			map[string]any{
				"name":              "baseline-e2m-renamed",
				"description":       "Renamed",
				"permutationsLimit": int64(250),
				"metricLabels": []any{
					map[string]any{"targetLabel": "status", "sourceField": "status"},
					map[string]any{"targetLabel": "path", "sourceField": "path"},
				},
				"metricFields": []any{map[string]any{
					"targetBaseMetricName": "latency",
					"sourceField":          "latency",
					"aggregations": []any{
						map[string]any{"aggType": "min", "targetMetricName": "latency_min", "aggMetadata": map[string]any{}},
						map[string]any{"aggType": "max", "enabled": false, "targetMetricName": "latency_max", "aggMetadata": map[string]any{}},
					},
				}},
			}))
	},

	"clear-optional": func(h *harness) {
		h.Create("create with every optional field", e2mFullManifest)
		h.SetSpec("remove the description, data source, limit, lucene, and alias", e2mSpec(
			map[string]any{"logs": map[string]any{
				"applicationNameFilters": []any{"app"},
				"subsystemNameFilters":   []any{"sub"},
				"severityFilters":        []any{"error"},
			}},
			map[string]any{
				"metricLabels": []any{map[string]any{"targetLabel": "status", "sourceField": "status"}},
				"metricFields": []any{map[string]any{
					"targetBaseMetricName": "latency",
					"sourceField":          "latency",
					"aggregations": []any{
						map[string]any{"aggType": "min", "targetMetricName": "latency_min", "aggMetadata": map[string]any{}},
					},
				}},
			}))
		h.SetSpec("remove the labels, the fields, and the filters", e2mSpec(
			map[string]any{"logs": map[string]any{}}, nil))
	},

	"clear-optional-spans": func(h *harness) {
		h.Create("create with a spans query", `apiVersion: coralogix.com/v1alpha1
kind: Events2Metric
metadata:
  name: e2m
spec:
  name: baseline-e2m
  query:
    spans:
      lucene: 'service:nginx'
      applicationNameFilters: [app]
      subsystemNameFilters: [sub]
      actionFilters: [GET]
      serviceFilters: [nginx]
`)
		h.SetSpec("remove lucene and every filter", e2mSpec(map[string]any{"spans": map[string]any{}}, nil))
	},

	"switch-query": func(h *harness) {
		h.Create("create with a logs query", e2mFullManifest)
		h.SetSpec("switch the query to spans", e2mSpec(
			map[string]any{"spans": map[string]any{"lucene": "service:nginx", "serviceFilters": []any{"nginx"}}}, nil))
		h.SetSpec("switch the query back to logs", e2mSpec(
			map[string]any{"logs": map[string]any{"lucene": "status:200"}}, nil))
	},

	"delete": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		e2mAPI(h).forget(h.StatusID())
		h.Delete("delete: the E2M was deleted outside the operator")
	},

	"update-when-gone": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		e2mAPI(h).forget(h.StatusID())
		h.SetSpec("change the name: the E2M was deleted outside the operator", e2mSpec(
			map[string]any{"logs": map[string]any{}}, map[string]any{"name": "baseline-e2m-renamed"}))
		h.Reconcile("reconcile again")
	},

	"delete-after-update-when-gone": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		e2mAPI(h).forget(h.StatusID())
		h.SetSpec("change the name: the E2M was deleted outside the operator", e2mSpec(
			map[string]any{"logs": map[string]any{}}, map[string]any{"name": "baseline-e2m-renamed"}))
		h.Delete("delete")
	},

	// The shared reconciler creates when the status has no id, even while the CR is deleted. So the
	// controller's own "no id, nothing to delete" branch is not reached.
	"delete-without-id": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		clearE2MStatusID(h)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"delete-without-id-and-remote-gone": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		e2mAPI(h).forget(h.StatusID())
		clearE2MStatusID(h)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"create-rejected": func(h *harness) {
		h.Create("create with a metric name that the API rejects", e2mManifestWithFields(`    - targetBaseMetricName: has space
      sourceField: field
`))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-rejected-duplicate-name": func(h *harness) {
		e2mAPI(h).seed("baseline-e2m")
		h.Create("create with a name that another E2M has", e2mLogsManifest)
		h.Reconcile("reconcile again")
	},

	// The next answer is replaced, as if the API sent a body that the operator cannot use.
	// A body that the SDK cannot decode is a DeserializationError, and the shared reconciler does
	// not write the same condition twice.
	"create-empty-id": func(h *harness) {
		e2mAPI(h).answerNext(http.MethodPost, map[string]any{"e2m": map[string]any{"id": "", "name": "baseline-e2m", "type": "E2M_TYPE_LOGS2METRICS"}})
		h.Create("create: the answer has an empty id", e2mLogsManifest)
		e2mAPI(h).answerNext(http.MethodPost, map[string]any{})
		h.Reconcile("reconcile again: the answer has no E2M")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-malformed-response": func(h *harness) {
		e2mAPI(h).answerNext(http.MethodPost, "not an E2M")
		h.Create("create: the answer cannot be decoded", e2mLogsManifest)
		e2mAPI(h).answerNext(http.MethodPost, "not an E2M")
		h.Reconcile("reconcile again: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		e2mAPI(h).answerNext(http.MethodPut, "not an E2M")
		h.Reconcile("update: the answer cannot be decoded")
		e2mAPI(h).answerNext(http.MethodPut, "not an E2M")
		h.Reconcile("reconcile again: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-malformed-response": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		e2mAPI(h).answerNext(http.MethodDelete, "not an E2M")
		h.Delete("delete: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"bad-status-id": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		setE2MStatusID(h, "not-a-uuid")
		h.Reconcile("update with a status id that is not a UUID")
		h.Delete("delete with a status id that is not a UUID")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", e2mLogsManifest)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "metrics"})})
		h.Reconcile("reconcile after the selector stops matching the CR")
	},
}

// setE2MStatusID writes status.id of the CR, as a user who edits the status could.
func setE2MStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set Events2Metric status id: %v", err)
	}
}

// clearE2MStatusID removes status.id of the CR, as a user who edits the status could.
func clearE2MStatusID(h *harness) {
	h.t.Helper()
	obj := h.get()
	unstructured.RemoveNestedField(obj.Object, "status", "id")
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("clear Events2Metric status id: %v", err)
	}
}

// e2mFake is the events2metrics API, written from the SDK's openapi.yaml.
//
//	POST   /v2        E2MCreateParams -> {e2m}
//	PUT    /v2        E2M (with id)   -> {e2m}
//	GET    /v2/{id}                   -> {e2m}
//	DELETE /v2/{id}                   -> {id}
//
// The status codes, error texts, and stored defaults follow fidelity/recorded.json, which the
// real API answered. Each rule names the recorded case that shows it. A rule with no case name is
// not recorded: it is the simplest answer that the scenarios need.
type e2mFake struct {
	mu   sync.Mutex
	e2ms map[string]map[string]any // id -> stored E2M
	// dataSources are the data sets that exist. Only the rejection of an unknown one is recorded
	// ("rejections"). The scenarios use these two.
	dataSources map[string]bool
	// next holds one 200 body per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]any
}

// answerNext makes the next request with the method get a 200 answer with this body.
func (f *e2mFake) answerNext(method string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = body
}

func newE2MFake() apiFake {
	return &e2mFake{
		e2ms:        map[string]map[string]any{},
		dataSources: map[string]bool{"system/poc.error_tracking": true, "system/baseline": true},
		next:        map[string]any{},
	}
}

// seed stores an E2M with the name, as if another client made it.
func (f *e2mFake) seed(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := newUUID()
	f.e2ms[id] = map[string]any{"id": id, "name": name}
}

// forget deletes an E2M without a request, as if it was deleted outside the operator.
func (f *e2mFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.e2ms, id)
}

func (f *e2mFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if body, ok := f.next[method]; ok {
		delete(f.next, method)
		return http.StatusOK, body
	}
	rest, ok := strings.CutPrefix(path, e2mPath)
	if !ok {
		return routeNotFound()
	}
	id := strings.TrimPrefix(rest, "/")
	in, _ := body.(map[string]any)

	switch {
	case method == http.MethodPost && rest == "":
		return f.create(in)
	case method == http.MethodPut && rest == "":
		return f.replace(in)
	case method == http.MethodGet && id != "" && !strings.Contains(id, "/"):
		// "an id that is not a UUID": Get does not check the id. It answers 404.
		stored, ok := f.e2ms[id]
		if !ok {
			return apiError(http.StatusNotFound, fmt.Sprintf(`{"message":"NOT_FOUND: Events2Metrics with id %s not found","status":5}`, id))
		}
		return http.StatusOK, map[string]any{"e2m": stored}
	case method == http.MethodDelete && id != "" && !strings.Contains(id, "/"):
		if status, resp := f.existing(id); status != 0 {
			return status, resp
		}
		delete(f.e2ms, id)
		return http.StatusOK, map[string]any{"id": id}
	}
	return routeNotFound()
}

func (f *e2mFake) create(in map[string]any) (int, any) {
	if status, resp := f.validate(in, ""); status != 0 {
		return status, resp
	}
	id := newUUID()
	f.e2ms[id] = e2mStored(in, id, nil)
	return http.StatusOK, map[string]any{"e2m": f.e2ms[id]}
}

func (f *e2mFake) replace(in map[string]any) (int, any) {
	id, _ := in["id"].(string)
	if status, resp := f.existing(id); status != 0 {
		return status, resp
	}
	if status, resp := f.validate(in, id); status != 0 {
		return status, resp
	}
	f.e2ms[id] = e2mStored(in, id, f.e2ms[id])
	return http.StatusOK, map[string]any{"e2m": f.e2ms[id]}
}

var e2mUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// existing checks the id of a Replace or a Delete. A non-zero status is the error answer.
// "an id that is not a UUID": 400 "Invalid uuid". "lifecycle": an id that was deleted is also a
// 400, "L2M does not exist". It is not a 404.
func (f *e2mFake) existing(id string) (int, any) {
	if !e2mUUID.MatchString(id) {
		return e2mBadRequest(`{"validation":"uuid","code":"invalid_string","message":"Invalid uuid","path":["id"]}`)
	}
	if _, ok := f.e2ms[id]; !ok {
		return e2mBadRequest(`{"code":"custom","message":"L2M does not exist","path":["id"]}`)
	}
	return 0, nil
}

// e2mBadRequest answers 400 with one validation detail, as the real API does.
func e2mBadRequest(detail string) (int, any) {
	return apiError(http.StatusBadRequest, `{"message":"Bad Request","status":400,"details":[`+detail+`]}`)
}

var e2mMetricName = regexp.MustCompile(`^[\w/-]+$`)

// validate checks a create or replace body. A non-zero status is an error answer. skipID is the id
// of the E2M that the request replaces, which may keep its own name and metric names.
func (f *e2mFake) validate(in map[string]any, skipID string) (int, any) {
	// "rejections": a metric name with a space fails the regex of targetBaseMetricName.
	fields, _ := in["metricFields"].([]any)
	for i, raw := range fields {
		field, _ := raw.(map[string]any)
		if name, _ := field["targetBaseMetricName"].(string); !e2mMetricName.MatchString(name) {
			return e2mBadRequest(fmt.Sprintf(`{"validation":"regex","code":"invalid_string","message":"Invalid","path":["metricFields",%d,"targetBaseMetricName"]}`, i))
		}
	}
	// "rejections": an unknown data source.
	if ds, ok := in["dataSource"].(string); ok && !f.dataSources[ds] {
		return e2mBadRequest(fmt.Sprintf(`{"code":"custom","path":["e2m","dataSource"],"message":"dataSource '%s' does not exist"}`, ds))
	}
	// "rejections": a metric name that another E2M uses. This check comes before the name check:
	// the case sends both a used name and used metric names, and the API names the metric.
	for _, raw := range fields {
		field, _ := raw.(map[string]any)
		name, _ := field["targetBaseMetricName"].(string)
		for id, e := range f.e2ms {
			if id != skipID && e2mUsesMetric(e, name) {
				return e2mBadRequest(fmt.Sprintf(`{"code":"custom","path":[],"message":"Metric fields are already in use: %s"}`, name))
			}
		}
	}
	// "two E2Ms with the same name and different metric names".
	name, _ := in["name"].(string)
	for id, e := range f.e2ms {
		if id != skipID && e["name"] == name {
			return e2mBadRequest(fmt.Sprintf(`{"code":"custom","path":["e2m","name"],"message":"Name (%s) is not unique"}`, name))
		}
	}
	return 0, nil
}

func e2mUsesMetric(e map[string]any, name string) bool {
	fields, _ := e["metricFields"].([]any)
	for _, raw := range fields {
		if field, _ := raw.(map[string]any); field["targetBaseMetricName"] == name {
			return true
		}
	}
	return false
}

// e2mTime is the time of every stored E2M. The recorder replaces a time with <time>.
const e2mTime = "2026-01-01T00:00:00Z"

// e2mStored is the E2M that the API keeps for a Create or Replace body. old is the E2M that a
// Replace replaces, or nil.
func e2mStored(in map[string]any, id string, old map[string]any) map[string]any {
	stored := map[string]any{
		"id": id, "name": in["name"], "type": in["type"], "isInternal": false,
		"createTime": e2mTime, "updateTime": e2mTime,
		// "permutations": the default is 30000. Create takes the flat permutationsLimit. A Replace
		// without permutations resets the limit to the default. It does not keep the old one.
		"permutations": map[string]any{"limit": 30000, "hasExceededLimit": false},
		"metricLabels": e2mLabels(in["metricLabels"]),
		"metricFields": e2mFields(in["metricFields"]),
	}
	if old != nil {
		stored["createTime"] = old["createTime"]
	}
	if limit, ok := in["permutationsLimit"]; ok {
		stored["permutations"].(map[string]any)["limit"] = limit
	}
	if p, ok := in["permutations"].(map[string]any); ok {
		stored["permutations"].(map[string]any)["limit"] = p["limit"]
	}
	// "description kept when replace omits it": an omitted description keeps the old one.
	// "description on create": a description of "" is stored as "". An omitted one is not stored.
	if d, ok := in["description"]; ok {
		stored["description"] = d
	} else if old != nil && old["description"] != nil {
		stored["description"] = old["description"]
	}
	if ds, ok := in["dataSource"]; ok {
		stored["dataSource"] = ds
	}
	// "query shapes": each list is [] when it is not sent, and the API drops the logs alias.
	// "minimal create": lucene is "" when it is not sent. Only the logs case is recorded: spans
	// is assumed to do the same.
	if q, ok := in["logsQuery"].(map[string]any); ok {
		stored["logsQuery"] = e2mQuery(q, "applicationnameFilters", "subsystemnameFilters", "severityFilters")
	}
	if q, ok := in["spansQuery"].(map[string]any); ok {
		stored["spansQuery"] = e2mQuery(q, "applicationnameFilters", "subsystemnameFilters", "actionFilters", "serviceFilters")
	}
	return stored
}

func e2mQuery(q map[string]any, lists ...string) map[string]any {
	out := map[string]any{"lucene": ""}
	if lucene, ok := q["lucene"]; ok {
		out["lucene"] = lucene
	}
	for _, name := range lists {
		out[name] = []any{}
		if v, ok := q[name]; ok && v != nil {
			out[name] = v
		}
	}
	return out
}

func e2mLabels(raw any) []any {
	out := []any{}
	list, _ := raw.([]any)
	for _, l := range list {
		label, _ := l.(map[string]any)
		out = append(out, map[string]any{"sourceField": label["sourceField"], "targetLabel": label["targetLabel"]})
	}
	return out
}

func e2mFields(raw any) []any {
	out := []any{}
	list, _ := raw.([]any)
	for _, f := range list {
		field, _ := f.(map[string]any)
		out = append(out, map[string]any{
			"sourceField":          field["sourceField"],
			"targetBaseMetricName": field["targetBaseMetricName"],
			"aggregations":         e2mAggregations(field["aggregations"]),
		})
	}
	return out
}

// e2mAggregations is the fixed set of seven aggregations that the API stores for each metric
// field. "aggregations: empty list, absent, null, and every type": an empty list, an absent
// list, and null all store this set. The API ignores targetMetricName and always names the
// aggregations cx_<type>. A request changes only enabled, the histogram buckets, and the
// samples type of the entry with the same aggType.
func e2mAggregations(raw any) []any {
	set := []map[string]any{
		{"aggType": "AGG_TYPE_AVG", "enabled": true, "none": map[string]any{}, "targetMetricName": "cx_avg"},
		{"aggType": "AGG_TYPE_HISTOGRAM", "enabled": false, "histogram": map[string]any{"buckets": []any{}}, "targetMetricName": "cx_bucket"},
		{"aggType": "AGG_TYPE_COUNT", "enabled": true, "none": map[string]any{}, "targetMetricName": "cx_count"},
		{"aggType": "AGG_TYPE_MAX", "enabled": true, "none": map[string]any{}, "targetMetricName": "cx_max"},
		{"aggType": "AGG_TYPE_MIN", "enabled": true, "none": map[string]any{}, "targetMetricName": "cx_min"},
		{"aggType": "AGG_TYPE_SUM", "enabled": true, "none": map[string]any{}, "targetMetricName": "cx_sum"},
		{"aggType": "AGG_TYPE_SAMPLES", "enabled": true, "samples": map[string]any{"sampleType": "SAMPLE_TYPE_MAX"}, "targetMetricName": "cx_value"},
	}
	requested, _ := raw.([]any)
	for _, r := range requested {
		req, _ := r.(map[string]any)
		for _, agg := range set {
			if agg["aggType"] != req["aggType"] {
				continue
			}
			if enabled, ok := req["enabled"]; ok {
				agg["enabled"] = enabled
			}
			if h, ok := req["histogram"]; ok {
				agg["histogram"] = h
			}
			if sm, ok := req["samples"]; ok {
				agg["samples"] = sm
			}
		}
	}
	out := make([]any, len(set))
	for i, agg := range set {
		out[i] = agg
	}
	return out
}
