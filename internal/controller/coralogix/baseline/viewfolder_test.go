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
	"strings"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const viewFolderPath = "/data-exploration/views/v1/folders"

func init() {
	register(kind{
		name:     "viewfolder",
		crdFile:  "coralogix.com_viewfolders.yaml",
		resource: "viewfolders",
		newFake:  func() apiFake { return &viewFolderFake{folders: map[string]string{}} },
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.ViewFolderReconciler{Client: cs.ViewsFolders(), Interval: reconcileInterval}
		},
		scenarios: viewFolderScenarios,
		fidelity: fidelityConfig{
			captureField: "id",
			deletePath:   func(id string) string { return viewFolderPath + "/" + id },
		},
	})
}

func viewFolderManifest(specName string) string {
	return fmt.Sprintf(`apiVersion: coralogix.com/v1alpha1
kind: ViewFolder
metadata:
  name: folder
spec:
  name: %q
`, specName)
}

func viewFolders(h *harness) *viewFolderFake {
	h.t.Helper()
	f, ok := h.API().(*viewFolderFake)
	if !ok {
		h.t.Fatal("expected *viewFolderFake")
	}
	return f
}

var viewFolderScenarios = map[string]func(h *harness){
	"create": func(h *harness) {
		h.CreateFromFile("create the sample", "config/samples/v1alpha1/views/view-folder.yaml")
		h.Reconcile("reconcile again with no change")
	},

	"rename": func(h *harness) {
		h.Create("create", viewFolderManifest("Team views"))
		h.SetSpec("change the name", map[string]any{"name": "Team views renamed"})
	},

	"delete": func(h *harness) {
		h.Create("create", viewFolderManifest("Team views"))
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", viewFolderManifest("Team views"))
		viewFolders(h).forget(h.StatusID())
		h.Delete("delete: the folder was deleted outside the operator")
	},

	"update-when-gone": func(h *harness) {
		h.Create("create", viewFolderManifest("Team views"))
		viewFolders(h).forget(h.StatusID())
		h.SetSpec("change the name: the folder was deleted outside the operator", map[string]any{"name": "Team views renamed"})
		h.Reconcile("reconcile again")
	},

	"delete-after-update-when-gone": func(h *harness) {
		h.Create("create", viewFolderManifest("Team views"))
		viewFolders(h).forget(h.StatusID())
		h.SetSpec("change the name: the folder was deleted outside the operator", map[string]any{"name": "Team views renamed"})
		h.Delete("delete")
		h.Reconcile("reconcile again")
	},

	"create-rejected": func(h *harness) {
		h.Create("create with an empty name: the API rejects it", viewFolderManifest(""))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},
}

// viewFolderFake is the folders-for-views API, written from the SDK's openapi.yaml.
//
//	POST   /folders       {name}      -> {id, name}
//	PUT    /folders       {id, name}  -> {id, name}
//	GET    /folders/{id}              -> {id, name}
//	GET    /folders                   -> {folders: [...]}
//	DELETE /folders/{id}              -> {}
//
// The status codes and error texts follow fidelity/recorded.json, which the real API answered.
// The operator reads only the status code, and whether a 404 names a resource or a missing route.
type viewFolderFake struct {
	mu      sync.Mutex
	folders map[string]string // id -> name
}

// forget deletes a folder without a request, as if it was deleted outside the operator.
func (f *viewFolderFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.folders, id)
}

func (f *viewFolderFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	rest, ok := strings.CutPrefix(path, viewFolderPath)
	if !ok {
		return routeNotFound()
	}
	id := strings.TrimPrefix(rest, "/")
	in, _ := body.(map[string]any)

	switch {
	case method == http.MethodPost && rest == "":
		name, status, errBody := viewFolderName(in)
		if status != 0 {
			return status, errBody
		}
		id := newUUID()
		f.folders[id] = name
		return http.StatusOK, map[string]any{"id": id, "name": name}

	case method == http.MethodPut && rest == "":
		id, _ := in["id"].(string)
		name, status, errBody := viewFolderName(in)
		if status != 0 {
			return status, errBody
		}
		if _, ok := f.folders[id]; !ok {
			return viewFolderNotFound("")
		}
		f.folders[id] = name
		return http.StatusOK, map[string]any{"id": id, "name": name}

	case method == http.MethodGet && rest == "":
		list := []any{}
		for id, name := range f.folders {
			list = append(list, map[string]any{"id": id, "name": name})
		}
		return http.StatusOK, map[string]any{"folders": list}

	case method == http.MethodGet && id != "" && !strings.Contains(id, "/"):
		name, ok := f.folders[id]
		if !ok {
			return viewFolderNotFound(id)
		}
		return http.StatusOK, map[string]any{"id": id, "name": name}

	case method == http.MethodDelete && id != "" && !strings.Contains(id, "/"):
		if _, ok := f.folders[id]; !ok {
			return viewFolderNotFound("")
		}
		delete(f.folders, id)
		return http.StatusOK, map[string]any{}
	}
	return routeNotFound()
}

// viewFolderName checks the name of a create or replace request. A non-zero status is an error
// answer. A name over 100 characters is cut to 100: the Terraform provider documents that the
// API does this.
func viewFolderName(in map[string]any) (string, int, any) {
	name, _ := in["name"].(string)
	if name == "" {
		status, body := apiError(http.StatusBadRequest, `{"message":"INVALID_ARGUMENT: validation error: folder name not found","status":3}`)
		return "", status, body
	}
	if r := []rune(name); len(r) > 100 {
		name = string(r[:100])
	}
	return name, 0, nil
}

// viewFolderNotFound is the answer for a folder that does not exist. Only Get names the id.
func viewFolderNotFound(id string) (int, any) {
	if id == "" {
		return apiError(http.StatusNotFound, `{"message":"NOT_FOUND: folder not found","status":5}`)
	}
	return apiError(http.StatusNotFound, fmt.Sprintf(`{"message":"NOT_FOUND: folder not found","status":5,"details":{"folderId":%q}}`, id))
}
