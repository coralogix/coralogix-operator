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
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const dashboardPath = "/dashboards/dashboards/v1"

// Ids that the scenarios write themselves. maskIDs keeps the UUID one. A dashboard id is 21
// characters and not a UUID: the SDK rejects a path id of another length before it sends the
// request. The fake makes them in order (baselineDashboard0001, ...), so goldens need no mask.
const (
	dashboardFolderID   = fixedIDPrefix + "0000000000d1"
	importedDashboardID = "baselineImportedDash1"
)

// dashboardDefaultAccessPolicy is the policy that the API stores for a dashboard created
// without one (fidelity/recorded.json).
const dashboardDefaultAccessPolicy = `{"default":{"permissions":{"team-dashboards:Read":"grant","team-dashboards:ReadAccessPolicy":"grant","team-dashboards:Update":"grant","team-dashboards:UpdateAccessPolicy":"grant"}},"rules":[],"version":"2025-01-01"}`

const dashboardOwnAccessPolicy = `{"version":"2025-01-01","default":{"permissions":{"team-dashboards:Read":"grant","team-dashboards:Update":"deny"}},"rules":[]}`

func init() {
	register(kind{
		name:     "dashboard",
		crdFile:  "coralogix.com_dashboards.yaml",
		resource: "dashboards",
		newFake:  newDashboardFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.DashboardReconciler{Client: cs.Dashboards(), ClientSet: cs, Interval: reconcileInterval}
		},
		scenarios: dashboardScenarios,
		fidelity: fidelityConfig{
			captureField: "dashboardId",
			deletePath:   func(id string) string { return dashboardPath + "/" + id },
		},
	})
}

// dashboardContent is the smallest dashboard that the API accepts: a name and a layout.
func dashboardContent(name string, extra ...string) string {
	fields := append([]string{fmt.Sprintf(`"name":%q`, name), `"layout":{"sections":[]}`}, extra...)
	return "{" + strings.Join(fields, ",") + "}"
}

// dashboardManifest returns a Dashboard CR. spec is YAML, indented by two spaces.
func dashboardManifest(spec string, annotations ...string) string {
	meta := "  name: dashboard\n"
	if len(annotations) > 0 {
		meta += "  annotations:\n"
		for i := 0; i+1 < len(annotations); i += 2 {
			meta += fmt.Sprintf("    %s: %q\n", annotations[i], annotations[i+1])
		}
	}
	return "apiVersion: coralogix.com/v1alpha1\nkind: Dashboard\nmetadata:\n" + meta + "spec:\n" + spec
}

func dashboardJSONSpec(content string) string {
	return fmt.Sprintf("  json: %q\n", content)
}

func dashboardJSONManifest(content string) string {
	return dashboardManifest(dashboardJSONSpec(content))
}

func gzipBase64(t interface{ Fatal(...any) }, content string) string {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func dashboards(h *harness) *dashboardFake {
	h.t.Helper()
	f, ok := h.API().(*dashboardFake)
	if !ok {
		h.t.Fatal("expected *dashboardFake")
	}
	return f
}

// dashboardConfigMap is the ConfigMap that the scenarios read content from.
const dashboardConfigMap = "dashboard-definition"

// createConfigMap creates the ConfigMap in the scenario namespace.
func createConfigMap(h *harness, data map[string]string) {
	h.t.Helper()
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: dashboardConfigMap, Namespace: h.namespace}, Data: data}
	if err := h.client.Create(context.Background(), cm); err != nil {
		h.t.Fatalf("create ConfigMap %s: %v", dashboardConfigMap, err)
	}
}

// createDashboardsFolder creates a DashboardsFolder CR. A non-empty id is written to its status.
func createDashboardsFolder(h *harness, name, id string) {
	h.t.Helper()
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "coralogix.com/v1alpha1",
		"kind":       "DashboardsFolder",
		"metadata":   map[string]any{"name": name, "namespace": h.namespace},
		"spec":       map[string]any{"name": name},
	}}
	if err := h.client.Create(context.Background(), obj); err != nil {
		h.t.Fatalf("create DashboardsFolder %s: %v", name, err)
	}
	if id == "" {
		return
	}
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set DashboardsFolder %s status id: %v", name, err)
	}
}

// setDashboardStatusID writes status.id of the CR, as a user who edits the status could.
func setDashboardStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set status id: %v", err)
	}
}

const importAnnotation = "app.coralogix.com/import-id"

var dashboardScenarios = map[string]func(h *harness){
	// Content sources.

	"create-json-sample": func(h *harness) {
		h.CreateFromFile("create the JSON sample: an unknown key and an access policy", "config/samples/v1alpha1/dashboards/dashboard-json.yaml")
		h.Reconcile("reconcile again with no change")
	},

	"create-gzip": func(h *harness) {
		h.Create("create from gzip content", dashboardManifest(fmt.Sprintf("  gzipJson: %s\n", gzipBase64(h.t, dashboardContent("Gzip dashboard")))))
	},

	"create-gzip-invalid": func(h *harness) {
		h.Create("create from gzip content that is not gzip", dashboardManifest(fmt.Sprintf("  gzipJson: %s\n", base64.StdEncoding.EncodeToString([]byte("plain text")))))
	},

	"create-configmap": func(h *harness) {
		createConfigMap(h, map[string]string{"json": dashboardContent("ConfigMap dashboard")})
		h.Create("create from a ConfigMap", dashboardManifest("  configMapRef:\n    name: dashboard-definition\n    key: json\n"))
	},

	"create-configmap-missing": func(h *harness) {
		h.Create("create from a ConfigMap that does not exist", dashboardManifest("  configMapRef:\n    name: dashboard-definition\n    key: json\n"))
		createConfigMap(h, map[string]string{"json": dashboardContent("ConfigMap dashboard")})
		h.Reconcile("reconcile again after the ConfigMap is created")
	},

	"create-configmap-missing-key": func(h *harness) {
		createConfigMap(h, map[string]string{"other": dashboardContent("ConfigMap dashboard")})
		h.Create("create from a ConfigMap without the key", dashboardManifest("  configMapRef:\n    name: dashboard-definition\n    key: json\n"))
	},

	"create-json-and-gzip": func(h *harness) {
		h.Create("create with json and gzipJson: json wins", dashboardManifest(
			dashboardJSONSpec(dashboardContent("JSON wins"))+fmt.Sprintf("  gzipJson: %s\n", gzipBase64(h.t, dashboardContent("Gzip loses")))))
	},

	"create-without-content": func(h *harness) {
		h.Create("create with no content source", dashboardManifest("  accessPolicy: \"\"\n"))
		h.Delete("delete")
	},

	"create-invalid-json": func(h *harness) {
		h.Create("create with content that is not JSON", dashboardJSONManifest(`{"name":`))
		h.Delete("delete")
	},

	"create-with-content-id": func(h *harness) {
		h.Create("create with an id in the content: it is sent as it is", dashboardJSONManifest(dashboardContent("Content id", `"id":"contentDashboardId001"`)))
		h.Reconcile("update: the status id replaces the content id")
	},

	// The content decoder (from the released api/coralogix/v1alpha1/dashboard_types_test.go).

	"content-protobuf-spellings": func(h *harness) {
		h.Create("create with protobuf field spellings and a bare int64 number: both are accepted", dashboardJSONManifest(
			`{"name":"Spellings","layout":{"sections":[{"rows":[{"widgets":[{"definition":{"line_chart":{"query_definitions":[{"id":"q1","query":{},"series_count_limit":20}]}}}]}]}]}}`))
	},

	"content-unknown-enum": func(h *harness) {
		h.Create("create with an enum value that the API does not have: rejected before a request", dashboardJSONManifest(
			`{"name":"Enum","layout":{"sections":[{"rows":[{"widgets":[{"definition":{"gauge":{"thresholdBy":"THRESHOLD_BY_FUTURE"}}}]}]}]}}`))
	},

	"content-numeric-enum": func(h *harness) {
		h.Create("create with a numeric enum value: rejected before a request", dashboardJSONManifest(
			`{"name":"Enum","layout":{"sections":[{"rows":[{"widgets":[{"definition":{"gauge":{"thresholdBy":1}}}]}]}]}}`))
	},

	"content-folder-without-ref": func(h *harness) {
		h.Create("create with a folder in the content and no folderRef: the content folder is sent", dashboardJSONManifest(
			dashboardContent("Content folder", `"folderPath":{"segments":["from","content"]}`)))
	},

	// Folder references.

	"folder-backend-id": func(h *harness) {
		h.Create("create in a folder by backend id", dashboardManifest(
			dashboardJSONSpec(dashboardContent("Folder by id", `"folderPath":{"segments":["from","content"]}`))+
				fmt.Sprintf("  folderRef:\n    backendRef:\n      id: %s\n", dashboardFolderID)))
		h.Reconcile("update")
	},

	"folder-backend-path": func(h *harness) {
		h.Create("create in a folder by path", dashboardManifest(
			dashboardJSONSpec(dashboardContent("Folder by path", fmt.Sprintf(`"folderId":{"value":%q}`, dashboardFolderID)))+
				"  folderRef:\n    backendRef:\n      path: team/services/api\n"))
		h.Reconcile("update")
	},

	"folder-resource-ref": func(h *harness) {
		createDashboardsFolder(h, "folder", dashboardFolderID)
		h.Create("create in a folder by DashboardsFolder CR", dashboardManifest(
			dashboardJSONSpec(dashboardContent("Folder by CR"))+"  folderRef:\n    resourceRef:\n      name: folder\n"))
		h.Reconcile("update")
	},

	"folder-resource-ref-without-id": func(h *harness) {
		createDashboardsFolder(h, "folder", "")
		h.Create("create in a DashboardsFolder CR that has no status id", dashboardManifest(
			dashboardJSONSpec(dashboardContent("Folder by CR"))+"  folderRef:\n    resourceRef:\n      name: folder\n"))
	},

	"folder-resource-ref-missing": func(h *harness) {
		h.Create("create in a DashboardsFolder CR that does not exist", dashboardManifest(
			dashboardJSONSpec(dashboardContent("Folder by CR"))+"  folderRef:\n    resourceRef:\n      name: folder\n"))
	},

	"folder-unknown": func(h *harness) {
		h.Create("create in a folder that does not exist: the API rejects it", dashboardManifest(
			dashboardJSONSpec(dashboardContent("Unknown folder"))+
				"  folderRef:\n    backendRef:\n      id: 00000000-0000-4000-8000-0000000000f1\n"))
	},

	// Updates.

	"update": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard", `"description":"first"`)))
		h.SetSpec("change the content", map[string]any{"json": dashboardContent("Baseline dashboard renamed")})
		h.Reconcile("reconcile again with no change")
	},

	"update-access-policy": func(h *harness) {
		h.Create("create without an access policy: the API stores its default", dashboardJSONManifest(dashboardContent("Policy dashboard")))
		h.SetSpec("set the access policy", map[string]any{"json": dashboardContent("Policy dashboard"), "accessPolicy": dashboardOwnAccessPolicy})
		h.SetSpec("set an empty access policy", map[string]any{"json": dashboardContent("Policy dashboard"), "accessPolicy": ""})
		h.SetSpec("remove the access policy: it is not sent, and the API keeps the stored one", map[string]any{"json": dashboardContent("Policy dashboard")})
	},

	"switch-content-source": func(h *harness) {
		h.Create("create from json", dashboardJSONManifest(dashboardContent("From json")))
		createConfigMap(h, map[string]string{"json": dashboardContent("From ConfigMap")})
		h.SetSpec("switch to a ConfigMap", map[string]any{"configMapRef": map[string]any{"name": "dashboard-definition", "key": "json"}})
		h.SetSpec("switch to gzip", map[string]any{"gzipJson": gzipBase64(h.t, dashboardContent("From gzip"))})
	},

	"resync": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Resync dashboard")))
		h.Reconcile("resync")
		h.Reconcile("resync again")
	},

	// Deletes and remote state.

	"delete": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		dashboards(h).forget(h.StatusID())
		h.Delete("delete: the dashboard was deleted outside the operator")
	},

	"update-when-gone": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		dashboards(h).forget(h.StatusID())
		h.SetSpec("change the content: the dashboard was deleted outside the operator", map[string]any{"json": dashboardContent("Baseline dashboard renamed")})
		h.Reconcile("reconcile again: the dashboard is created again")
	},

	"delete-after-update-when-gone": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		dashboards(h).forget(h.StatusID())
		h.SetSpec("change the content: the dashboard was deleted outside the operator", map[string]any{"json": dashboardContent("Baseline dashboard renamed")})
		h.Delete("delete")
		h.Reconcile("reconcile again")
	},

	"create-rejected": func(h *harness) {
		h.Create("create with an empty name: the API rejects it", dashboardJSONManifest(dashboardContent("")))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-rejected-no-layout": func(h *harness) {
		h.Create("create without a layout: the API rejects it", dashboardJSONManifest(`{"name":"No layout"}`))
	},

	"create-empty-id": func(h *harness) {
		dashboards(h).answerNext(http.MethodPost, map[string]any{"dashboardId": ""})
		h.Create("create: the answer has an empty id", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		dashboards(h).answerNext(http.MethodPost, map[string]any{})
		h.Reconcile("reconcile again: the answer has no id")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-malformed-response": func(h *harness) {
		dashboards(h).answerNext(http.MethodPost, "not a dashboard")
		h.Create("create: the answer cannot be decoded", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		dashboards(h).answerNext(http.MethodPut, map[string]any{"unexpected": true})
		h.Reconcile("update: the answer has a field that the contract does not allow")
		dashboards(h).answerNext(http.MethodPut, "not an answer")
		h.Reconcile("update: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-malformed-response": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		dashboards(h).answerNext(http.MethodDelete, "not an answer")
		h.Delete("delete: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"bad-status-id": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		setDashboardStatusID(h, "not-a-dashboard")
		h.Reconcile("update with a status id that does not exist")
		h.Reconcile("reconcile again: the dashboard is created again")
		h.Delete("delete")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", dashboardJSONManifest(dashboardContent("Baseline dashboard")))
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "dashboards"})})
		h.Reconcile("reconcile after the selector stops matching the CR")
	},

	"selector-mismatch-imported": func(h *harness) {
		dashboards(h).seed(dashboardContent("Remote dashboard"))
		h.Create("import", dashboardManifest(dashboardJSONSpec(dashboardContent("Imported dashboard")), importAnnotation, importedDashboardID))
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "dashboards"})})
		h.Reconcile("reconcile after the selector stops matching the CR: status.imported stays")
	},

	// Import by annotation.

	"import": func(h *harness) {
		dashboards(h).seed(dashboardContent("Remote dashboard"))
		h.Create("import: the dashboard is read, not created", dashboardManifest(dashboardJSONSpec(dashboardContent("Imported dashboard")), importAnnotation, importedDashboardID))
		h.Reconcile("update: the spec replaces the remote content")
		h.Reconcile("resync")
		h.Delete("delete")
	},

	"import-with-access-policy": func(h *harness) {
		dashboards(h).seed(dashboardContent("Remote dashboard"))
		h.Create("import", dashboardManifest(dashboardJSONSpec(dashboardContent("Imported dashboard"))+
			fmt.Sprintf("  accessPolicy: %q\n", dashboardOwnAccessPolicy), importAnnotation, importedDashboardID))
		h.Reconcile("update: the access policy is sent")
	},

	"import-missing": func(h *harness) {
		h.Create("import a dashboard that does not exist", dashboardManifest(dashboardJSONSpec(dashboardContent("Imported dashboard")), importAnnotation, importedDashboardID))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"import-with-content-id": func(h *harness) {
		dashboards(h).seed(dashboardContent("Remote dashboard"))
		h.Create("import with an id in the content: rejected before a request", dashboardManifest(
			dashboardJSONSpec(dashboardContent("Imported dashboard", `"id":"contentDashboardId001"`)), importAnnotation, importedDashboardID))
		h.Delete("delete")
	},

	"import-then-gone": func(h *harness) {
		dashboards(h).seed(dashboardContent("Remote dashboard"))
		h.Create("import", dashboardManifest(dashboardJSONSpec(dashboardContent("Imported dashboard")), importAnnotation, importedDashboardID))
		dashboards(h).forget(importedDashboardID)
		h.Reconcile("update: the dashboard was deleted outside the operator")
		h.Reconcile("reconcile again: the dashboard is created, not imported again")
		h.Reconcile("resync")
	},

	"import-annotation-added-later": func(h *harness) {
		dashboards(h).seed(dashboardContent("Remote dashboard"))
		h.Create("create without the annotation", dashboardJSONManifest(dashboardContent("Created dashboard")))
		obj := h.get()
		obj.SetAnnotations(map[string]string{importAnnotation: importedDashboardID})
		if err := h.client.Update(context.Background(), obj); err != nil {
			h.t.Fatal(err)
		}
		h.Reconcile("add the annotation after create: the status id stays")
	},
}

// dashboardFake is the dashboards API, written from the SDK's openapi.yaml.
//
//	POST   /dashboards/dashboards/v1        {requestId, dashboard, accessPolicy?, isLocked?} -> {dashboardId}
//	PUT    /dashboards/dashboards/v1        {requestId, dashboard, accessPolicy?, isLocked?} -> {}
//	GET    /dashboards/dashboards/v1/{id}   -> {dashboard, accessPolicy, ...}
//	DELETE /dashboards/dashboards/v1/{id}   -> {}
//
// The status codes, error texts, defaults, and the stored access policy follow
// fidelity/recorded.json, which the real API answered.
type dashboardFake struct {
	mu         sync.Mutex
	created    int
	dashboards map[string]*storedDashboard
	// folders are the dashboard folders that exist.
	folders map[string]bool
	// next holds one 200 body per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]any
}

type storedDashboard struct {
	dashboard    map[string]any
	accessPolicy string
}

func newDashboardFake() apiFake {
	return &dashboardFake{
		dashboards: map[string]*storedDashboard{},
		folders:    map[string]bool{dashboardFolderID: true},
		next:       map[string]any{},
	}
}

func (f *dashboardFake) answerNext(method string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = body
}

// seed stores the dashboard that the import scenarios adopt, as if another client made it.
func (f *dashboardFake) seed(content string) {
	id := importedDashboardID
	f.mu.Lock()
	defer f.mu.Unlock()
	var d map[string]any
	if err := json.Unmarshal([]byte(content), &d); err != nil {
		panic(err)
	}
	d["id"] = id
	f.dashboards[id] = &storedDashboard{dashboard: d, accessPolicy: dashboardDefaultAccessPolicy}
}

// forget deletes a dashboard without a request, as if it was deleted outside the operator.
func (f *dashboardFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.dashboards, id)
}

func (f *dashboardFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if body, ok := f.next[method]; ok {
		delete(f.next, method)
		return http.StatusOK, body
	}
	rest, ok := strings.CutPrefix(path, dashboardPath)
	if !ok {
		return routeNotFound()
	}
	id := strings.TrimPrefix(rest, "/")
	in, _ := body.(map[string]any)

	switch {
	case method == http.MethodPost && rest == "":
		d, _ := in["dashboard"].(map[string]any)
		if status, errBody := f.check(d); status != 0 {
			return status, errBody
		}
		f.created++
		id := fmt.Sprintf("baselineDashboard%04d", f.created)
		// A copy: the request body stays as the operator sent it in the golden.
		d = maps.Clone(d)
		d["id"] = id
		policy := dashboardDefaultAccessPolicy
		if p, ok := in["accessPolicy"].(string); ok {
			policy = p
		}
		f.dashboards[id] = &storedDashboard{dashboard: d, accessPolicy: policy}
		return http.StatusOK, map[string]any{"dashboardId": id}

	case method == http.MethodPut && rest == "":
		d, _ := in["dashboard"].(map[string]any)
		if status, errBody := f.check(d); status != 0 {
			return status, errBody
		}
		id, _ := d["id"].(string)
		stored, ok := f.dashboards[id]
		if !ok {
			return dashboardNotFound(id)
		}
		// The dashboard is replaced as a whole. An omitted access policy keeps the stored one.
		stored.dashboard = d
		if p, ok := in["accessPolicy"].(string); ok {
			stored.accessPolicy = p
		}
		return http.StatusOK, map[string]any{}

	case method == http.MethodGet && id != "" && !strings.Contains(id, "/"):
		stored, ok := f.dashboards[id]
		if !ok {
			return dashboardNotFound(id)
		}
		return http.StatusOK, stored.answer()

	case method == http.MethodDelete && id != "" && !strings.Contains(id, "/"):
		if _, ok := f.dashboards[id]; !ok {
			return dashboardNotFound(id)
		}
		delete(f.dashboards, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

// check validates the dashboard of a create or replace request, in the order of the real API.
func (f *dashboardFake) check(d map[string]any) (int, any) {
	if name, _ := d["name"].(string); name == "" {
		return apiError(http.StatusBadRequest, "dashboard name cannot be empty")
	}
	if _, ok := d["layout"].(map[string]any); !ok {
		return apiError(http.StatusBadRequest, "dashboard layout not found")
	}
	if folder, ok := d["folderId"].(map[string]any); ok {
		if id, _ := folder["value"].(string); !f.folders[id] {
			return apiError(http.StatusBadRequest, fmt.Sprintf("dashboard folder with id DashboardFolderId(%s) does not exist", id))
		}
	}
	return 0, nil
}

// answer is the Get answer. The API adds empty lists for the parts that a dashboard omits.
func (s *storedDashboard) answer() map[string]any {
	d := map[string]any{}
	for k, v := range s.dashboard {
		d[k] = v
	}
	for _, k := range []string{"actions", "annotations", "filters", "variables", "variablesV2"} {
		if _, ok := d[k]; !ok {
			d[k] = []any{}
		}
	}
	if layout, ok := d["layout"].(map[string]any); ok {
		if _, ok := layout["sections"]; !ok {
			layout = map[string]any{"sections": []any{}}
			d["layout"] = layout
		}
	}
	return map[string]any{
		"accessPolicy":      s.accessPolicy,
		"accessSource":      "DASHBOARD_ACCESS_SOURCE_UNSPECIFIED",
		"authorId":          "<redacted>",
		"authorName":        "<redacted>",
		"createdAt":         "2026-01-02T03:04:05.678Z",
		"createdOriginType": "TOKEN_ORIGIN_TYPE_API",
		"dashboard":         d,
		"isLocked":          false,
		"permissions": map[string]any{
			"canRead": true, "canReadAccessPolicy": true, "canUpdate": true, "canUpdateAccessPolicy": true,
		},
		"resolvedWidgets":   map[string]any{},
		"updatedAt":         "2026-01-02T03:04:05.678Z",
		"updatedOriginType": "TOKEN_ORIGIN_TYPE_API",
		"updaterAuthorId":   "<redacted>",
		"updaterName":       "<redacted>",
	}
}

func dashboardNotFound(id string) (int, any) {
	return apiError(http.StatusNotFound, fmt.Sprintf("dashboard with id %s does not exist", id))
}
