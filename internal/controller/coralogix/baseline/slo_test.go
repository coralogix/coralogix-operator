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
	"reflect"
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

const sloPath = "/slo/slos/v1"

func init() {
	register(kind{
		name:     "slo",
		crdFile:  "coralogix.com_slos.yaml",
		resource: "slos",
		newFake:  newSLOFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.SLOReconciler{Client: cs.SLOs(), ClientSet: cs, Interval: reconcileInterval}
		},
		scenarios: sloScenarios,
		fidelity: fidelityConfig{
			captureField: "slo.id",
			deletePath:   func(id string) string { return sloPath + "/" + id },
		},
	})
}

// A spec is built from these parts, so that each scenario shows only what it changes.
const sloRequestManifest = `apiVersion: coralogix.com/v1alpha1
kind: SLO
metadata:
  name: slo
spec:
  name: baseline-slo
  sliType:
    requestBasedMetric:
      goodEvents:
        query: sum(rate(good_total[1m]))
      totalEvents:
        query: sum(rate(all_total[1m]))
  window:
    timeFrame: 7d
  targetThresholdPercentage: 99
`

// sloFullManifest sets every optional field of a window-based SLO that a user can clear.
const sloFullManifest = `apiVersion: coralogix.com/v1alpha1
kind: SLO
metadata:
  name: slo
spec:
  name: baseline-slo
  description: Baseline description
  labels:
    team: baseline
  productType: unspecified
  ownershipTags:
    service:
      staticValues: [checkout]
    environment:
      staticValues: [production]
    team:
      labelKeys: [team]
  sliType:
    windowBasedMetric:
      query:
        query: sum(rate(errors_total[5m])) by (service)
      window: 5m
      comparisonOperator: lessThan
      threshold: "0.5"
      missingDataStrategy: bad
  window:
    timeFrame: 28d
  targetThresholdPercentage: "99.5"
`

// sloManifest is the request-based manifest with its sliType block replaced by sli, when sli is
// not empty, and the extra spec lines appended. Both are YAML text indented by two spaces.
func sloManifest(sli, extra string) string {
	m := sloRequestManifest
	if sli != "" {
		start := strings.Index(m, "  sliType:\n")
		end := strings.Index(m, "  window:\n")
		m = m[:start] + sli + m[end:]
	}
	return m + extra
}

func sloSpec(fields map[string]any) map[string]any {
	spec := map[string]any{
		"name": "baseline-slo",
		"sliType": map[string]any{"requestBasedMetric": map[string]any{
			"goodEvents":  map[string]any{"query": "sum(rate(good_total[1m]))"},
			"totalEvents": map[string]any{"query": "sum(rate(all_total[1m]))"},
		}},
		"window":                    map[string]any{"timeFrame": "7d"},
		"targetThresholdPercentage": int64(99),
	}
	maps.Copy(spec, fields)
	return spec
}

const sloAPMErrorSLI = `  sliType:
    apmSli:
      services: [checkout-service]
      errorConfig: {}
`

func sloAPI(h *harness) *sloFake {
	h.t.Helper()
	f, ok := h.API().(*sloFake)
	if !ok {
		h.t.Fatal("expected *sloFake")
	}
	return f
}

var sloScenarios = map[string]func(h *harness){
	"create-request-sample": func(h *harness) {
		h.CreateFromFile("create the request-based sample", "config/samples/v1alpha1/slos/request-based.yaml")
	},
	"create-window-sample": func(h *harness) {
		h.CreateFromFile("create the window-based sample", "config/samples/v1alpha1/slos/window-based.yaml")
	},
	"create-apm-error-sample": func(h *harness) {
		h.CreateFromFile("create the APM error-rate sample", "config/samples/v1alpha1/slos/apm-error.yaml")
	},
	"create-apm-latency-sample": func(h *harness) {
		h.CreateFromFile("create the APM latency sample", "config/samples/v1alpha1/slos/apm-latency.yaml")
	},

	"create-minimal": func(h *harness) {
		h.Create("create with only the required fields", sloRequestManifest)
	},

	"create-full": func(h *harness) {
		h.Create("create with every optional field", sloFullManifest)
	},

	// groupByLabels is deprecated: the request never has it.
	"create-group-by-labels": func(h *harness) {
		h.Create("create with groupByLabels", sloManifest(`  sliType:
    requestBasedMetric:
      goodEvents:
        query: sum(rate(good_total[1m]))
      totalEvents:
        query: sum(rate(all_total[1m]))
      groupByLabels: [service]
`, ""))
	},

	"create-window-without-threshold": func(h *harness) {
		h.Create("create a window-based SLO without threshold or missingDataStrategy", sloManifest(`  sliType:
    windowBasedMetric:
      query:
        query: sum(rate(errors_total[5m]))
      window: 1m
      comparisonOperator: greaterThanOrEquals
`, ""))
	},

	"create-time-frames": func(h *harness) {
		h.Create("create with timeFrame 14d", strings.Replace(sloRequestManifest, "timeFrame: 7d", "timeFrame: 14d", 1))
		for _, tf := range []string{"21d", "28d", "7d"} {
			h.SetSpec("change timeFrame to "+tf, sloSpec(map[string]any{"window": map[string]any{"timeFrame": tf}}))
		}
	},

	// 90d passes the CRD, but the API no longer has it, so the conversion rejects it.
	"time-frame-90d": func(h *harness) {
		h.Create("create with timeFrame 90d", strings.Replace(sloRequestManifest, "timeFrame: 7d", "timeFrame: 90d", 1))
		h.SetSpec("change timeFrame to 7d", sloSpec(nil))
	},

	"time-frame-unspecified": func(h *harness) {
		h.Create("create with timeFrame unspecified", strings.Replace(sloRequestManifest, "timeFrame: 7d", "timeFrame: unspecified", 1))
	},

	// The CRD does not require a time frame. The API does.
	"time-frame-omitted": func(h *harness) {
		h.Create("create without timeFrame", strings.Replace(sloRequestManifest, "  window:\n    timeFrame: 7d\n", "  window: {}\n", 1))
	},

	"product-type": func(h *harness) {
		h.Create("create an APM SLO with productType apm", sloManifest(sloAPMErrorSLI, "  productType: apm\n"))
		spec := sloSpec(map[string]any{"sliType": map[string]any{"apmSli": map[string]any{"services": []any{"checkout-service"}, "errorConfig": map[string]any{}}}})
		h.SetSpec("remove productType", spec)
	},

	"apm-filters-and-grouping": func(h *harness) {
		h.Create("create an APM SLO with filters and grouping keys", sloManifest(`  sliType:
    apmSli:
      services: [checkout-service]
      errorConfig: {}
      filters:
        - key: http.method
          values: [POST, PUT]
        - key: http.route
      groupingKeys: [http.route]
`, "  productType: apm\n"))
	},

	"apm-latency-average": func(h *harness) {
		h.Create("create an APM latency SLO with average and no threshold", sloManifest(`  sliType:
    apmSli:
      services: [checkout-service]
      latencyConfig:
        timeWindow: 1m
        average: {}
`, "  productType: apm\n"))
	},

	"apm-latency-quantile-without-percentile": func(h *harness) {
		h.Create("create an APM latency SLO with an empty quantile", sloManifest(`  sliType:
    apmSli:
      services: [checkout-service]
      latencyConfig:
        timeWindow: 5m
        threshold: "250"
        quantile: {}
`, "  productType: apm\n"))
	},

	// The CRD has no rule for the empty-object arms: the operator rejects these before any request.
	"apm-without-arm": func(h *harness) {
		h.Create("create an APM SLO with neither errorConfig nor latencyConfig", sloManifest(`  sliType:
    apmSli:
      services: [checkout-service]
`, "  productType: apm\n"))
	},
	"apm-both-arms": func(h *harness) {
		h.Create("create an APM SLO with errorConfig and latencyConfig", sloManifest(`  sliType:
    apmSli:
      services: [checkout-service]
      errorConfig: {}
      latencyConfig:
        timeWindow: 5m
        average: {}
`, "  productType: apm\n"))
	},
	"apm-latency-without-query-type": func(h *harness) {
		h.Create("create an APM latency SLO with neither quantile nor average", sloManifest(`  sliType:
    apmSli:
      services: [checkout-service]
      latencyConfig:
        timeWindow: 5m
`, "  productType: apm\n"))
	},
	"apm-latency-both-query-types": func(h *harness) {
		h.Create("create an APM latency SLO with quantile and average", sloManifest(`  sliType:
    apmSli:
      services: [checkout-service]
      latencyConfig:
        timeWindow: 5m
        quantile:
          percentile: "0.9"
        average: {}
`, "  productType: apm\n"))
	},

	"ownership-tags": func(h *harness) {
		h.Create("create with ownership tags and an empty dimension", sloManifest("", `  ownershipTags:
    service:
      staticValues: [checkout, checkout]
    environment:
      staticValues: []
      labelKeys: []
    team: {}
`))
		h.SetSpec("clear the static values of service", sloSpec(map[string]any{"ownershipTags": map[string]any{"service": map[string]any{"staticValues": []any{}}}}))
		h.SetSpec("set an empty ownershipTags block", sloSpec(map[string]any{"ownershipTags": map[string]any{}}))
	},

	"resync": func(h *harness) {
		h.Create("create", sloFullManifest)
		h.Reconcile("reconcile again with no change")
	},

	"update": func(h *harness) {
		h.Create("create", sloRequestManifest)
		h.SetSpec("change the name, description, labels, and target", sloSpec(map[string]any{
			"name":                      "baseline-slo-renamed",
			"description":               "Changed description",
			"labels":                    map[string]any{"team": "changed"},
			"targetThresholdPercentage": "99.9",
		}))
		h.SetSpec("change the queries: the remote revision moves", sloSpec(map[string]any{
			"sliType": map[string]any{"requestBasedMetric": map[string]any{
				"goodEvents":  map[string]any{"query": "sum(rate(good_v2_total[1m]))"},
				"totalEvents": map[string]any{"query": "sum(rate(all_v2_total[1m]))"},
			}},
		}))
	},

	"clear-optional": func(h *harness) {
		h.Create("create with every optional field", sloFullManifest)
		h.SetSpec("remove every optional field", map[string]any{
			"name": "baseline-slo",
			"sliType": map[string]any{"windowBasedMetric": map[string]any{
				"query":              map[string]any{"query": "sum(rate(errors_total[5m])) by (service)"},
				"window":             "5m",
				"comparisonOperator": "lessThan",
			}},
			"window":                    map[string]any{"timeFrame": "28d"},
			"targetThresholdPercentage": "99.5",
		})
		h.Reconcile("reconcile again")
	},

	"switch-sli": func(h *harness) {
		h.Create("create a request-based SLO", sloRequestManifest)
		h.SetSpec("switch to a window-based SLI", sloSpec(map[string]any{"sliType": map[string]any{"windowBasedMetric": map[string]any{
			"query":              map[string]any{"query": "sum(rate(errors_total[5m]))"},
			"window":             "1m",
			"comparisonOperator": "greaterThan",
			"threshold":          "1",
		}}}))
	},

	"delete": func(h *harness) {
		h.Create("create", sloRequestManifest)
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", sloRequestManifest)
		sloAPI(h).forget(h.StatusID())
		h.Delete("delete: the SLO was deleted outside the operator")
	},

	"update-when-gone": func(h *harness) {
		h.Create("create", sloRequestManifest)
		sloAPI(h).forget(h.StatusID())
		h.SetSpec("change the name: the SLO was deleted outside the operator", sloSpec(map[string]any{"name": "baseline-slo-renamed"}))
		h.Reconcile("reconcile again")
	},

	"delete-after-update-when-gone": func(h *harness) {
		h.Create("create", sloRequestManifest)
		sloAPI(h).forget(h.StatusID())
		h.SetSpec("change the name: the SLO was deleted outside the operator", sloSpec(map[string]any{"name": "baseline-slo-renamed"}))
		h.Delete("delete")
		h.Reconcile("reconcile again")
	},

	"delete-without-id": func(h *harness) {
		h.Create("create", sloRequestManifest)
		clearSLOStatusID(h)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"delete-without-id-and-remote-gone": func(h *harness) {
		h.Create("create", sloRequestManifest)
		sloAPI(h).forget(h.StatusID())
		clearSLOStatusID(h)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"create-rejected": func(h *harness) {
		h.Create("create with an empty good events query: the API rejects it", sloManifest(`  sliType:
    requestBasedMetric:
      goodEvents:
        query: ""
      totalEvents:
        query: sum(rate(all_total[1m]))
`, ""))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-empty-id": func(h *harness) {
		sloAPI(h).answerNext(http.MethodPost, map[string]any{"slo": map[string]any{"id": "", "name": "baseline-slo", "sloTimeFrame": "SLO_TIME_FRAME_7_DAYS"}})
		h.Create("create: the answer has an empty id", sloRequestManifest)
		sloAPI(h).answerNext(http.MethodPost, map[string]any{})
		h.Reconcile("reconcile again: the answer has no SLO")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-malformed-response": func(h *harness) {
		sloAPI(h).answerNext(http.MethodPost, "not an SLO")
		h.Create("create: the answer cannot be decoded", sloRequestManifest)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", sloRequestManifest)
		sloAPI(h).answerNext(http.MethodPut, "not an SLO")
		h.Reconcile("update: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-malformed-response": func(h *harness) {
		h.Create("create", sloRequestManifest)
		sloAPI(h).answerNext(http.MethodDelete, "not an SLO")
		h.Delete("delete: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"bad-status-id": func(h *harness) {
		h.Create("create", sloRequestManifest)
		setSLOStatusID(h, "not-a-uuid")
		h.Reconcile("update with a status id that is not a UUID")
		h.Delete("delete with a status id that is not a UUID")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", sloFullManifest)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "slos"})})
		h.Reconcile("reconcile after the selector stops matching the CR")
	},
}

// setSLOStatusID writes status.id of the CR, as a user who edits the status could.
func setSLOStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set SLO status id: %v", err)
	}
}

// clearSLOStatusID removes status.id of the CR, as a user who edits the status could.
func clearSLOStatusID(h *harness) {
	h.t.Helper()
	obj := h.get()
	unstructured.RemoveNestedField(obj.Object, "status", "id")
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("clear SLO status id: %v", err)
	}
}

// sloFake is the SLO API, written from the SDK's openapi.yaml.
//
//	POST   /slo/slos/v1        Slo           -> {slo, exclusionWindowIds}
//	PUT    /slo/slos/v1        Slo (with id) -> {slo, effectedSloAlertIds, exclusionWindowIds}
//	GET    /slo/slos/v1/{id}                 -> {slo}
//	DELETE /slo/slos/v1/{id}                 -> {effectedSloAlertIds}
//
// The status codes, error texts, and stored defaults follow fidelity/recorded.json, which the
// real API answered. The server sets id, grouping (the by clause of the query, or service_name
// for an APM SLI), sloType, type, creator, revision, sliDefinition (a copy of a metric SLI), and
// the times. An omitted targetThresholdPercentage or threshold reads as 0, an omitted
// missingDataStrategy as UNCOUNTED, and an omitted productType as UNSPECIFIED. A replace that
// changes nothing is not written: the stored SLO, with its description and revision, stays. Any
// other replace stores the request, so an omitted description or label is cleared, and moves the
// revision unless only the ownership tags change. A written ownership dimension returns
// staticValues and labelKeys as sent, and an empty resolvedValues; a dimension with no values is
// dropped. A request-based query must use 1-minute ranges. An APM service and a static service
// tag must be in the APM Service Catalog.
//
// The APM SLI is recorded only with an unknown service: the recording team has no APM service.
// The other APM rules follow what the released kind documents.
type sloFake struct {
	mu   sync.Mutex
	slos map[string]map[string]any // id -> stored SLO
	// services are the services of the APM Service Catalog.
	services map[string]bool
	clock    int
	// next holds one 200 body per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]any
}

func newSLOFake() apiFake {
	return &sloFake{slos: map[string]map[string]any{}, services: map[string]bool{"checkout-service": true, "checkout": true}, next: map[string]any{}}
}

// answerNext makes the next request with the method get a 200 answer with this body.
func (f *sloFake) answerNext(method string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = body
}

// forget deletes an SLO without a request, as if it was deleted outside the operator.
func (f *sloFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.slos, id)
}

func (f *sloFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if body, ok := f.next[method]; ok {
		delete(f.next, method)
		return http.StatusOK, body
	}
	rest, ok := strings.CutPrefix(path, sloPath)
	if !ok {
		return routeNotFound()
	}
	id := strings.TrimPrefix(rest, "/")
	in, _ := body.(map[string]any)

	switch {
	case method == http.MethodPost && rest == "":
		return f.write(in, false)
	case method == http.MethodPut && rest == "":
		return f.write(in, true)
	case method == http.MethodGet && id != "" && !strings.Contains(id, "/"):
		if !uuidPattern.MatchString(id) {
			return sloValidationMethod("GetSlo", `"id" must be a valid GUID`)
		}
		stored, found := f.slos[id]
		if !found {
			return sloNotFound(id)
		}
		return http.StatusOK, map[string]any{"slo": stored}
	case method == http.MethodDelete && id != "" && !strings.Contains(id, "/"):
		if !uuidPattern.MatchString(id) {
			return sloValidationMethod("DeleteSlo", `"id" must be a valid GUID`)
		}
		if _, found := f.slos[id]; !found {
			return sloNotFound(id)
		}
		delete(f.slos, id)
		return http.StatusOK, map[string]any{"effectedSloAlertIds": []any{}}
	}
	return routeNotFound()
}

// sloNotFound is the answer for an SLO that does not exist.
func sloNotFound(id string) (int, any) {
	return apiError(http.StatusNotFound, "Slo with id "+id+" not found for company <number>")
}

// sloValidation is the answer to a request that the validation layer refuses.
func sloValidation(replace bool, reason string) (int, any) {
	if replace {
		return sloValidationMethod("ReplaceSlo", reason)
	}
	return sloValidationMethod("CreateSlo", reason)
}

func sloValidationMethod(method, reason string) (int, any) {
	return apiError(http.StatusBadRequest, "validation failed for method /com.coralogixapis.slo.v1.SlosService/"+method+": ValidationError: "+reason)
}

var sloArms = []string{"apmSli", "requestBasedMetricSli", "windowBasedMetricSli"}

func (f *sloFake) write(in map[string]any, replace bool) (int, any) {
	if in == nil {
		return apiError(http.StatusBadRequest, "slo is required")
	}
	if status, resp := f.check(in, replace); status != 0 {
		return status, resp
	}
	stored := map[string]any{
		"name":                      in["name"],
		"targetThresholdPercentage": 0,
		"sloTimeFrame":              in["sloTimeFrame"],
		"productType":               "SLO_PRODUCT_TYPE_UNSPECIFIED",
		"labels":                    map[string]any{},
	}
	if v, ok := in["targetThresholdPercentage"]; ok {
		stored["targetThresholdPercentage"] = v
	}
	if desc, _ := in["description"].(string); desc != "" {
		stored["description"] = desc
	}
	if labels, _ := in["labels"].(map[string]any); len(labels) > 0 {
		stored["labels"] = labels
	}
	if pt, _ := in["productType"].(string); pt != "" {
		stored["productType"] = pt
	}
	if tags := sloOwnershipTags(in["ownershipTags"]); tags != nil {
		stored["ownershipTags"] = tags
	}
	var arm string
	switch {
	case in["requestBasedMetricSli"] != nil:
		arm = "requestBasedMetricSli"
		sli := in[arm].(map[string]any)
		stored[arm] = sli
		stored["sliDefinition"] = map[string]any{"requestBased": map[string]any{"custom": sli}}
		stored["grouping"] = map[string]any{"labels": sloGroupBy(sli["totalEvents"])}
	case in["windowBasedMetricSli"] != nil:
		arm = "windowBasedMetricSli"
		sli := maps.Clone(in[arm].(map[string]any))
		if sli["missingDataStrategy"] == nil {
			sli["missingDataStrategy"] = "MISSING_DATA_STRATEGY_UNCOUNTED"
		}
		if sli["threshold"] == nil {
			sli["threshold"] = 0
		}
		stored[arm] = sli
		stored["sliDefinition"] = map[string]any{"windowBased": map[string]any{
			"custom":              map[string]any{"comparisonOperator": sli["comparisonOperator"], "query": sli["query"], "threshold": sli["threshold"]},
			"missingDataStrategy": sli["missingDataStrategy"],
			"window":              sli["window"],
		}}
		stored["grouping"] = map[string]any{"labels": sloGroupBy(sli["query"])}
	default:
		arm = "apmSli"
		sli := sloAPMDefaults(in[arm].(map[string]any))
		stored[arm] = sli
		stored["apmSliMetadata"] = maps.Clone(sli)
		stored["grouping"] = map[string]any{"labels": []any{"service_name"}}
	}
	stored["sloType"], stored["type"] = sloTypeOf(in), "request"
	if stored["sloType"] == "SLO_TYPE_WINDOW" {
		stored["type"] = "window"
	}

	f.clock++
	now := fmt.Sprintf("2026-01-01T00:00:%02dZ", f.clock)
	id, _ := in["id"].(string)
	if replace {
		prev := f.slos[id]
		if sloSame(prev, stored, "description") {
			// "a replace without description keeps the stored one": nothing changed, so nothing
			// is written. The answer has the stored SLO without its description.
			return http.StatusOK, map[string]any{"slo": sloWithout(prev, "description"), "effectedSloAlertIds": []any{}, "exclusionWindowIds": []any{}}
		}
		stored["createTime"], stored["creator"] = prev["createTime"], prev["creator"]
		prevRevision, _ := prev["revision"].(map[string]any)
		revision, _ := prevRevision["revision"].(int)
		if !sloSame(prev, stored, "ownershipTags") {
			revision++
		}
		stored["revision"] = map[string]any{"revision": revision, "updateTime": now}
	} else {
		id = newUUID()
		stored["createTime"], stored["creator"] = now, "<email>"
		stored["revision"] = map[string]any{"revision": 1, "updateTime": now}
	}
	stored["id"] = id
	stored["updateTime"] = now
	f.slos[id] = stored
	if replace {
		return http.StatusOK, map[string]any{"slo": stored, "effectedSloAlertIds": []any{}, "exclusionWindowIds": []any{}}
	}
	return http.StatusOK, map[string]any{"slo": stored, "exclusionWindowIds": []any{}}
}

// sloServerFields are set by the server. They are not compared when a replace is checked for a
// change.
var sloServerFields = []string{"id", "createTime", "creator", "revision", "updateTime"}

// sloSame reports whether two stored SLOs are the same, apart from the server fields and the
// ignored field.
func sloSame(a, b map[string]any, ignored string) bool {
	return reflect.DeepEqual(sloWithout(a, append(sloServerFields, ignored)...), sloWithout(b, append(sloServerFields, ignored)...))
}

func sloWithout(slo map[string]any, fields ...string) map[string]any {
	out := maps.Clone(slo)
	for _, field := range fields {
		delete(out, field)
	}
	return out
}

// sloTypeOf is the SLO type that the server derives from the SLI. A window-based metric SLI
// reads SLO_TYPE_REQUEST; an APM latency SLI reads SLO_TYPE_WINDOW.
func sloTypeOf(in map[string]any) string {
	if apm, ok := in["apmSli"].(map[string]any); ok && apm["latencyConfig"] != nil {
		return "SLO_TYPE_WINDOW"
	}
	return "SLO_TYPE_REQUEST"
}

func (f *sloFake) check(in map[string]any, replace bool) (int, any) {
	var set []string
	for _, arm := range sloArms {
		if in[arm] != nil {
			set = append(set, arm)
		}
	}
	if name, _ := in["name"].(string); name == "" {
		return sloValidation(replace, `"slo.name" is not allowed to be empty`)
	}
	if tf, _ := in["sloTimeFrame"].(string); tf == "" {
		return sloValidation(replace, `"slo.window" is required`)
	}
	if replace {
		if id, _ := in["id"].(string); !uuidPattern.MatchString(id) {
			return sloValidation(replace, `"slo.id" must be a valid GUID`)
		}
	}
	if pt, _ := in["productType"].(string); pt == "SLO_PRODUCT_TYPE_APM" && in["apmSli"] == nil {
		return sloValidation(replace, "slo.apm_sli is required when slo.product_type is SLO_PRODUCT_TYPE_APM")
	}
	if len(set) == 0 {
		return sloValidation(replace, `"slo.sli" is required`)
	}
	if status, resp := sloCheckQueries(in); status != 0 {
		return status, resp
	}
	if window, ok := in["windowBasedMetricSli"].(map[string]any); ok {
		if op, _ := window["comparisonOperator"].(string); op == "" || op == "COMPARISON_OPERATOR_UNSPECIFIED" {
			return apiError(http.StatusBadRequest, "Invalid comparison operator: 0")
		}
		if w, _ := window["window"].(string); w == "" || w == "WINDOW_SLO_WINDOW_UNSPECIFIED" {
			return apiError(http.StatusBadRequest, "Window SLI window must be specified")
		}
	}
	if status, resp := f.checkOwnership(in); status != 0 {
		return status, resp
	}
	if apm, ok := in["apmSli"].(map[string]any); ok {
		if status, resp := f.checkAPM(apm); status != 0 {
			return status, resp
		}
	}
	if replace {
		id, _ := in["id"].(string)
		if _, found := f.slos[id]; !found {
			return apiError(http.StatusNotFound, "Cannot replace Slo with id "+id+" because it does not exist")
		}
	}
	if in["sloTimeFrame"] == "SLO_TIME_FRAME_UNSPECIFIED" {
		return apiError(http.StatusInternalServerError, "failed to create SLO: SLO time window must be specified")
	}
	return 0, nil
}

// checkOwnership refuses a static service tag that is not in the APM Service Catalog.
func (f *sloFake) checkOwnership(in map[string]any) (int, any) {
	tags, _ := in["ownershipTags"].(map[string]any)
	service, _ := tags["service"].(map[string]any)
	values, _ := service["staticValues"].([]any)
	for _, v := range values {
		if name, _ := v.(string); !f.services[name] {
			return apiError(http.StatusBadRequest, "Query: ownership service static values contain unknown services")
		}
	}
	return 0, nil
}

// sloRange is a range selector of PromQL, such as [5m].
var sloRange = regexp.MustCompile(`\[[^\]]*\]`)

const sloRangeProblem = "SLO queries must use a 1-minute range ([1m]) to ensure consistent time-window evaluation. Update all time ranges in your query to [1m]."

// sloCheckQueries refuses a metric SLI without a query, and a request-based query with a range
// other than [1m].
func sloCheckQueries(in map[string]any) (int, any) {
	query := func(metric any) string {
		m, _ := metric.(map[string]any)
		q, _ := m["query"].(string)
		return q
	}
	if sli, ok := in["requestBasedMetricSli"].(map[string]any); ok {
		if query(sli["goodEvents"]) == "" {
			return apiError(http.StatusBadRequest, "Good events query: Query is missing")
		}
		if query(sli["totalEvents"]) == "" {
			return apiError(http.StatusBadRequest, "Total events query: Query is missing")
		}
		var problems []string
		for _, q := range []struct{ name, query string }{{"Good events query", query(sli["goodEvents"])}, {"Total events query", query(sli["totalEvents"])}} {
			for _, r := range sloRange.FindAllString(q.query, -1) {
				if r != "[1m]" {
					problems = append(problems, q.name+": "+sloRangeProblem)
					break
				}
			}
		}
		if len(problems) != 0 {
			return apiError(http.StatusBadRequest, strings.Join(problems, "; "))
		}
	}
	missing := func(metric any) bool { return query(metric) == "" }
	if sli, ok := in["windowBasedMetricSli"].(map[string]any); ok && missing(sli["query"]) {
		return apiError(http.StatusBadRequest, "Window query: Query is missing")
	}
	return 0, nil
}

// checkAPM refuses an APM SLI that the released kind documents as invalid. These answers are
// not recorded.
func (f *sloFake) checkAPM(apm map[string]any) (int, any) {
	services, _ := apm["services"].([]any)
	if len(services) != 1 {
		return apiError(http.StatusBadRequest, "an APM SLI needs exactly one service")
	}
	// "an APM SLI with an unknown service"
	if name, _ := services[0].(string); !f.services[name] {
		return apiError(http.StatusBadRequest, "One or more APM SLO services not found in catalog")
	}
	if (apm["errorConfig"] == nil) == (apm["latencyConfig"] == nil) {
		return apiError(http.StatusBadRequest, "an APM SLI needs errorConfig or latencyConfig")
	}
	if latency, ok := apm["latencyConfig"].(map[string]any); ok {
		if latency["timeWindow"] == nil {
			return apiError(http.StatusBadRequest, "latencyConfig.timeWindow is required")
		}
		if (latency["quantile"] == nil) == (latency["average"] == nil) {
			return apiError(http.StatusBadRequest, "latencyConfig needs quantile or average")
		}
	}
	return 0, nil
}

// sloAPMDefaults stores the defaults of an APM SLI and the empty arrays that the API returns.
func sloAPMDefaults(in map[string]any) map[string]any {
	sli := maps.Clone(in)
	filters, _ := sli["filters"].([]any)
	withOperator := make([]any, len(filters))
	for i, f := range filters {
		filter, _ := f.(map[string]any)
		filter = maps.Clone(filter)
		if filter["matchOperator"] == nil {
			filter["matchOperator"] = "APM_FILTER_MATCH_OPERATOR_UNSPECIFIED"
		}
		withOperator[i] = filter
	}
	sli["filters"] = withOperator
	if sli["groupingKeys"] == nil {
		sli["groupingKeys"] = []any{}
	}
	if latency, ok := sli["latencyConfig"].(map[string]any); ok {
		latency = maps.Clone(latency)
		if latency["threshold"] == nil {
			latency["threshold"] = 0
		}
		if quantile, ok := latency["quantile"].(map[string]any); ok {
			quantile = maps.Clone(quantile)
			if quantile["percentile"] == nil {
				quantile["percentile"] = 0
			}
			latency["quantile"] = quantile
		}
		sli["latencyConfig"] = latency
	}
	return sli
}

// sloOwnershipTags drops a dimension with no values, and fills the arrays of the others. The
// static values stay as sent. resolvedValues is empty right after a write: the server resolves it
// later.
func sloOwnershipTags(in any) map[string]any {
	tags, _ := in.(map[string]any)
	out := map[string]any{}
	for _, dim := range []string{"service", "environment", "team"} {
		tag, _ := tags[dim].(map[string]any)
		static, _ := tag["staticValues"].([]any)
		keys, _ := tag["labelKeys"].([]any)
		if len(static) == 0 && len(keys) == 0 {
			continue
		}
		out[dim] = map[string]any{
			"staticValues":   append([]any{}, static...),
			"labelKeys":      append([]any{}, keys...),
			"resolvedValues": []any{},
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

var sloByClause = regexp.MustCompile(`by\s*\(([^)]*)\)`)

// sloGroupBy returns the labels of the by (...) clause of a metric query.
func sloGroupBy(metric any) []any {
	m, _ := metric.(map[string]any)
	query, _ := m["query"].(string)
	match := sloByClause.FindStringSubmatch(query)
	out := []any{}
	if match == nil {
		return out
	}
	for _, label := range strings.Split(match[1], ",") {
		if label = strings.TrimSpace(label); label != "" {
			out = append(out, label)
		}
	}
	return out
}
