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
	"net/http"
	"net/url"
	"strconv"
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

const viewPath = "/data-exploration/views/v1/views"

// The status ids of the ViewFolder CRs that the scenarios create. They are fixed, so maskIDs
// keeps them.
const (
	sampleFolderID = fixedIDPrefix + "000000000001"
	otherFolderID  = fixedIDPrefix + "000000000002"
)

func init() {
	register(kind{
		name:     "view",
		crdFile:  "coralogix.com_views.yaml",
		resource: "views",
		newFake:  newViewFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.ViewReconciler{ViewsClient: cs.Views(), Interval: reconcileInterval}
		},
		scenarios: viewScenarios,
	})
}

// viewQuickManifest has no search query and no folder.
const viewQuickManifest = `apiVersion: coralogix.com/v1alpha1
kind: View
metadata:
  name: view
spec:
  name: Baseline view
  timeSelection:
    quickSelection:
      seconds: 3600
  filters:
    filters:
      - name: severity
        selectedValues:
          ERROR: true
          INFO: false
`

// viewCustomTimeManifest has times with a zone offset and fractional seconds.
const viewCustomTimeManifest = `apiVersion: coralogix.com/v1alpha1
kind: View
metadata:
  name: view
spec:
  name: Custom time view
  searchQuery:
    query: "region:us-west-2"
  timeSelection:
    customSelection:
      fromTime: "2024-01-25T13:31:43.152+02:00"
      toTime: "2024-01-25T11:35:43Z"
  filters:
    filters:
      - name: applicationName
        selectedValues:
          sample-app: true
`

// viewFolderRefManifest refers to the ViewFolder "folder". It sets the reference namespace when
// namespace is not empty.
func viewFolderRefManifest(namespace string) string {
	ref := "      name: folder\n"
	if namespace != "" {
		ref += "      namespace: " + namespace + "\n"
	}
	return `apiVersion: coralogix.com/v1alpha1
kind: View
metadata:
  name: view
  labels:
    team: views
spec:
  name: Folder view
  timeSelection:
    quickSelection:
      seconds: 900
  filters:
    filters:
      - name: severity
        selectedValues:
          ERROR: true
  folder:
    resourceRef:
` + ref
}

var viewScenarios = map[string]func(h *harness){
	"create-quick-selection-sample": func(h *harness) {
		createViewFolder(h, h.namespace, "sample-folder", sampleFolderID)
		h.CreateFromFile("create the sample", "config/samples/v1alpha1/views/quick-selection-view.yaml")
		h.Reconcile("reconcile again with no change")
	},

	"create-custom-selection-sample": func(h *harness) {
		createViewFolder(h, h.namespace, "sample-folder", sampleFolderID)
		h.CreateFromFile("create the sample", "config/samples/v1alpha1/views/custom-selection-view.yaml")
	},

	"custom-time-offset": func(h *harness) {
		h.Create("create with a zone offset and fractional seconds", viewCustomTimeManifest)
	},

	"no-search-query": func(h *harness) {
		h.Create("create without a search query and a folder", viewQuickManifest)
	},

	"folder-other-namespace": func(h *harness) {
		folders := createNamespace(h, h.namespace+"-folders")
		createViewFolder(h, folders, "folder", otherFolderID)
		h.Create("create with a folder in another namespace", viewFolderRefManifest(folders))
	},

	"folder-missing": func(h *harness) {
		h.Create("do not call the API when the folder does not exist", viewFolderRefManifest(""))
		h.Delete("delete")
	},

	"folder-without-id": func(h *harness) {
		createViewFolder(h, h.namespace, "folder", "")
		h.Create("do not call the API before the folder has a status id", viewFolderRefManifest(""))
	},

	"folder-not-selected": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "views"})})
		createViewFolder(h, h.namespace, "folder", sampleFolderID)
		h.Create("do not call the API when the selector does not match the folder", viewFolderRefManifest(""))
	},

	"update": func(h *harness) {
		h.Create("create", viewQuickManifest)
		h.SetSpec("change the name, the time, and the filters", map[string]any{
			"name":          "Baseline view renamed",
			"searchQuery":   map[string]any{"query": "level:error"},
			"timeSelection": map[string]any{"quickSelection": map[string]any{"seconds": int64(86400)}},
			"filters": map[string]any{"filters": []any{
				map[string]any{"name": "subsystemName", "selectedValues": map[string]any{"api": true}},
			}},
		})
	},

	"delete": func(h *harness) {
		h.Create("create", viewQuickManifest)
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", viewQuickManifest)
		viewsFake(h).forget(h.StatusID())
		h.Delete("delete: the view was deleted outside the operator")
	},

	"update-when-gone": func(h *harness) {
		h.Create("create", viewQuickManifest)
		viewsFake(h).forget(h.StatusID())
		h.SetSpec("change the name: the view was deleted outside the operator", map[string]any{
			"name":          "Baseline view renamed",
			"timeSelection": map[string]any{"quickSelection": map[string]any{"seconds": int64(3600)}},
			"filters": map[string]any{"filters": []any{
				map[string]any{"name": "severity", "selectedValues": map[string]any{"ERROR": true}},
			}},
		})
		h.Reconcile("reconcile again")
	},

	"update-when-gone-404": func(h *harness) {
		h.Create("create", viewQuickManifest)
		f := viewsFake(h)
		f.forget(h.StatusID())
		f.replaceMissingNotFound = true
		h.SetSpec("change the name: the view was deleted outside the operator, and PUT answers 404", map[string]any{
			"name":          "Baseline view renamed",
			"timeSelection": map[string]any{"quickSelection": map[string]any{"seconds": int64(3600)}},
			"filters": map[string]any{"filters": []any{
				map[string]any{"name": "severity", "selectedValues": map[string]any{"ERROR": true}},
			}},
		})
		h.Reconcile("reconcile again: create the view again")
	},

	"remove-search-query": func(h *harness) {
		h.Create("create with a search query", viewCustomTimeManifest)
		h.SetSpec("remove the search query", map[string]any{
			"name":          "Custom time view",
			"timeSelection": map[string]any{"quickSelection": map[string]any{"seconds": int64(3600)}},
			"filters": map[string]any{"filters": []any{
				map[string]any{"name": "applicationName", "selectedValues": map[string]any{"sample-app": true}},
			}},
		})
	},

	"negative-seconds": func(h *harness) {
		h.Create("create with negative seconds: admission accepts it", strings.Replace(viewQuickManifest, "seconds: 3600", "seconds: -1", 1))
		h.Reconcile("reconcile again")
	},

	"delete-after-update-when-gone": func(h *harness) {
		h.Create("create", viewQuickManifest)
		viewsFake(h).forget(h.StatusID())
		h.SetSpec("change the name: the view was deleted outside the operator", map[string]any{
			"name":          "Baseline view renamed",
			"timeSelection": map[string]any{"quickSelection": map[string]any{"seconds": int64(3600)}},
			"filters": map[string]any{"filters": []any{
				map[string]any{"name": "severity", "selectedValues": map[string]any{"ERROR": true}},
			}},
		})
		h.Delete("delete")
	},

	"create-rejected": func(h *harness) {
		h.Create("create with an empty name: the API rejects it", strings.Replace(viewQuickManifest, "name: Baseline view", `name: ""`, 1))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"bad-status-id": func(h *harness) {
		h.Create("create", viewQuickManifest)
		setViewStatusID(h, "not-a-number")
		h.Reconcile("update with a status id that is not a number")
		setViewStatusID(h, "3000000000")
		h.Reconcile("update with a status id over the int32 range")
		h.Delete("delete with a status id over the int32 range")
	},
}

// createNamespace creates a namespace and returns its name.
func createNamespace(h *harness, name string) string {
	h.t.Helper()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := h.client.Create(context.Background(), ns); err != nil {
		h.t.Fatalf("create namespace %s: %v", name, err)
	}
	return name
}

// createViewFolder creates a ViewFolder CR without a reconciler. A non-empty id is set as its
// status id, as the ViewFolder controller does after Create.
func createViewFolder(h *harness, namespace, name, id string) {
	h.t.Helper()
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "coralogix.com/v1alpha1",
		"kind":       "ViewFolder",
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"spec":       map[string]any{"name": name},
	}}
	if err := h.client.Create(context.Background(), obj); err != nil {
		h.t.Fatalf("create ViewFolder %s: %v", name, err)
	}
	if id == "" {
		return
	}
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set ViewFolder %s status id: %v", name, err)
	}
}

// setViewStatusID writes status.id of the View, as a user who edits the status could.
func setViewStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set View status id: %v", err)
	}
}

func viewsFake(h *harness) *viewFake {
	h.t.Helper()
	f, ok := h.API().(*viewFake)
	if !ok {
		h.t.Fatal("expected *viewFake")
	}
	return f
}

// viewFake is the views API, written from the SDK's openapi.yaml and checked against the real API.
//
//	POST   /views       {name, searchQuery, timeSelection, filters, folderId}  -> View
//	GET    /views/{id}                                                         -> View
//	PUT    /views/{id}  {name, searchQuery, timeSelection, filters, folderId}  -> View
//	DELETE /views/{id}                                                         -> {}
//
// The service stores a view in its own form: see storedView. A replace of a view that does not
// exist answers 500, not 404; replaceMissingNotFound makes it answer 404, as the backend should.
// The operator reads only the id of a Create answer, and the status
// code of the others.
type viewFake struct {
	mu     sync.Mutex
	views  map[int]map[string]any
	nextID int
	// replaceMissingNotFound makes a replace of a missing view answer 404 instead of 500.
	replaceMissingNotFound bool
}

// viewFolderIDs are the folders that exist in the fake: the status ids that the scenarios give
// their ViewFolder CRs.
var viewFolderIDs = map[string]bool{sampleFolderID: true, otherFolderID: true}

func newViewFake() apiFake { return &viewFake{views: map[int]map[string]any{}} }

// forget deletes a view without a request, as if it was deleted outside the operator.
func (f *viewFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, _ := strconv.Atoi(id)
	delete(f.views, n)
}

func (f *viewFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	rest, ok := strings.CutPrefix(path, viewPath)
	if !ok {
		return routeNotFound()
	}
	in, _ := body.(map[string]any)
	if rest == "" {
		if method != http.MethodPost {
			return routeNotFound()
		}
		if status, errBody := checkView(in); status != 0 {
			return status, errBody
		}
		f.nextID++
		f.views[f.nextID] = storedView(f.nextID, in)
		return http.StatusOK, f.views[f.nextID]
	}

	rawID := strings.TrimPrefix(rest, "/")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return apiError(http.StatusBadRequest, fmt.Sprintf("type mismatch, parameter: id, error: strconv.ParseInt: parsing %q: invalid syntax", rawID))
	}
	_, exists := f.views[id]
	switch method {
	case http.MethodGet:
		if !exists {
			return apiError(http.StatusNotFound, fmt.Sprintf(`{"message":"NOT_FOUND: view not found","status":5,"details":{"viewId":%d}}`, id))
		}
		return http.StatusOK, f.views[id]
	case http.MethodPut:
		if status, errBody := checkView(in); status != 0 {
			return status, errBody
		}
		if !exists && f.replaceMissingNotFound {
			return apiError(http.StatusNotFound, fmt.Sprintf(`{"message":"NOT_FOUND: view not found","status":5,"details":{"viewId":%d}}`, id))
		}
		if !exists {
			return apiError(http.StatusInternalServerError, `{"message":"INTERNAL: Request failed","status":13}`)
		}
		f.views[id] = storedView(id, in)
		return http.StatusOK, f.views[id]
	case http.MethodDelete:
		if !exists {
			return apiError(http.StatusNotFound, `{"message":"NOT_FOUND: View not found","status":5}`)
		}
		delete(f.views, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

// checkView checks a create or replace request. A non-zero status is an error answer.
func checkView(in map[string]any) (int, any) {
	if name, _ := in["name"].(string); name == "" {
		return apiError(http.StatusBadRequest, `{"message":"INVALID_ARGUMENT: validation error: view name not found","status":3}`)
	}
	selection, ok := in["timeSelection"].(map[string]any)
	if !ok {
		return apiError(http.StatusBadRequest, `{"message":"INVALID_ARGUMENT: Invalid time selection","status":3}`)
	}
	// seconds is a uint32 in the API, so the gateway cannot read a negative value. The real
	// answer also names the position in the body, which the fake leaves out.
	if quick, ok := selection["quickSelection"].(map[string]any); ok {
		if seconds, _ := quick["seconds"].(float64); seconds < 0 {
			return apiError(http.StatusBadRequest, fmt.Sprintf("proto: invalid value for uint32 field seconds: %d", int64(seconds)))
		}
	}
	if folderID, ok := in["folderId"].(string); ok && !viewFolderIDs[folderID] {
		return apiError(http.StatusBadRequest, `{"message":"INVALID_ARGUMENT: folder not found","status":3}`)
	}
	return 0, nil
}

// storedView is the view as the service stores and returns it:
//   - searchQuery defaults to {query: ""}, and syntaxType to SYNTAX_TYPE_LUCENE.
//   - isCompactMode defaults to false, and viewType to VIEW_TYPE_LOGS.
//   - a filter keeps only the values set to true.
//   - quickSelection.seconds becomes the nearest supported range, with its caption.
//   - a replace stores an omitted searchQuery, syntaxType, or viewType as its default.
func storedView(id int, in map[string]any) map[string]any {
	out := map[string]any{"id": id, "isCompactMode": false, "viewType": "VIEW_TYPE_LOGS"}
	for k, v := range in {
		out[k] = v
	}
	query, _ := in["searchQuery"].(map[string]any)
	stored := map[string]any{"query": "", "syntaxType": "SYNTAX_TYPE_LUCENE"}
	for k, v := range query {
		stored[k] = v
	}
	out["searchQuery"] = stored
	if filters, ok := in["filters"].(map[string]any); ok {
		out["filters"] = map[string]any{"filters": storedFilters(filters["filters"])}
	}
	selection, _ := in["timeSelection"].(map[string]any)
	if quick, ok := selection["quickSelection"].(map[string]any); ok {
		seconds, _ := quick["seconds"].(float64)
		rounded := nearestQuickRange(int64(seconds))
		storedQuick := map[string]any{"seconds": rounded}
		if caption, ok := quickRangeCaptions[rounded]; ok {
			storedQuick["caption"] = caption
		}
		out["timeSelection"] = map[string]any{"quickSelection": storedQuick}
	}
	return out
}

func storedFilters(v any) []any {
	filters, _ := v.([]any)
	out := make([]any, 0, len(filters))
	for _, item := range filters {
		filter, _ := item.(map[string]any)
		values, _ := filter["selectedValues"].(map[string]any)
		kept := map[string]any{}
		for name, selected := range values {
			if b, _ := selected.(bool); b {
				kept[name] = true
			}
		}
		out = append(out, map[string]any{"name": filter["name"], "selectedValues": kept})
	}
	return out
}

// quickRanges are the ranges that the service stores, from the openapi.yaml description.
var quickRanges = []int64{60, 120, 300, 900, 1800, 3600, 7200, 21600, 43200, 86400, 172800, 259200, 432000, 604800}

// quickRangeCaptions holds only the captions that the real API returned in a check.
var quickRangeCaptions = map[int64]string{60: "Last 1 minute", 3600: "Last 1 hour", 86400: "Last 24 hours"}

// nearestQuickRange returns the supported range nearest to seconds. A tie picks the shorter one.
func nearestQuickRange(seconds int64) int64 {
	best := quickRanges[0]
	for _, r := range quickRanges[1:] {
		if abs64(r-seconds) < abs64(best-seconds) {
			best = r
		}
	}
	return best
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
