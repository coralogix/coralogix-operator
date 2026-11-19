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
	"regexp"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const dashboardsFolderPath = "/dashboards/folders/v1"

// The ids that the scenarios use for folders that they write themselves. They are fixed, so
// maskIDs keeps them.
const (
	dfParentID      = fixedIDPrefix + "00000000df01"
	dfOtherParentID = fixedIDPrefix + "00000000df02"
	dfUnknownID     = fixedIDPrefix + "00000000df03"
	dfCustomID      = fixedIDPrefix + "00000000df11"
	dfOtherCustomID = fixedIDPrefix + "00000000df12"
)

func init() {
	register(kind{
		name:     "dashboardsfolder",
		crdFile:  "coralogix.com_dashboardsfolders.yaml",
		resource: "dashboardsfolders",
		newFake:  newDashboardsFolderFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.DashboardsFolderReconciler{DashboardsFoldersClient: cs.DashboardFolders(), Interval: reconcileInterval}
		},
		scenarios: dashboardsFolderScenarios,
		admission: dashboardsFolderAdmission(),
		fidelity: fidelityConfig{
			captureField: "folderId",
			deletePath:   func(id string) string { return dashboardsFolderPath + "/" + id },
		},
	})
}

// dashboardsFolderManifest returns a DashboardsFolder CR named "folder". spec is YAML, indented
// by two spaces.
func dashboardsFolderManifest(spec string, labels ...string) string {
	meta := "  name: folder\n"
	if len(labels) > 0 {
		meta += "  labels:\n"
		for i := 0; i+1 < len(labels); i += 2 {
			meta += fmt.Sprintf("    %s: %q\n", labels[i], labels[i+1])
		}
	}
	return "apiVersion: coralogix.com/v1alpha1\nkind: DashboardsFolder\nmetadata:\n" + meta + "spec:\n" + spec
}

const dashboardsFolderMinimalManifest = `apiVersion: coralogix.com/v1alpha1
kind: DashboardsFolder
metadata:
  name: folder
spec:
  name: Team dashboards
`

func dashboardsFolders(h *harness) *dashboardsFolderFake {
	h.t.Helper()
	f, ok := h.API().(*dashboardsFolderFake)
	if !ok {
		h.t.Fatal("expected *dashboardsFolderFake")
	}
	return f
}

// seedParent makes a DashboardsFolder CR named "parent" in the namespace, with a status id, and
// the folder in the fake that the id names. It is the parent that a folderRef points at.
func seedParent(h *harness, namespace, id string) {
	h.t.Helper()
	dashboardsFolders(h).seed(id, "parent", "")
	createDashboardsFolder(h, namespace, "parent", id)
}

var dashboardsFolderScenarios = map[string]func(h *harness){
	// The sample file holds two CRs. The harness reads the first, the child. Its customId is not
	// sent: the Create request has no folder id.
	"create-sample": func(h *harness) {
		dashboardsFolders(h).seed(dfParentID, "parent-folder", "")
		createDashboardsFolder(h, h.namespace, "parent-folder", dfParentID)
		h.CreateFromFile("create the sample", "config/samples/v1alpha1/dashboardsfolder/dashboards-folder.yaml")
		h.Reconcile("reconcile again with no change")
	},

	"create-minimal": func(h *harness) {
		h.Create("create with a name only", dashboardsFolderMinimalManifest)
		h.Reconcile("reconcile again: the resync sends a Replace and no Get")
		h.Reconcile("reconcile again")
	},

	"create-custom-id": func(h *harness) {
		h.Create("create with a custom id: the request does not send it", dashboardsFolderManifest(
			"  name: Team dashboards\n  customId: "+dfCustomID+"\n"))
		h.Reconcile("reconcile again")
	},

	"create-parent-id": func(h *harness) {
		dashboardsFolders(h).seed(dfParentID, "parent", "")
		h.Create("create under a parent by backend id", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderId: "+dfParentID+"\n"))
		h.Reconcile("reconcile again")
	},

	"create-parent-id-empty": func(h *harness) {
		h.Create("create with an empty parent id: the request sends it and the API rejects it", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderId: \"\"\n"))
		h.Reconcile("reconcile again")
	},

	"create-parent-id-unknown": func(h *harness) {
		h.Create("create under a parent that does not exist: the API rejects it", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderId: "+dfUnknownID+"\n"))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-parent-id-not-uuid": func(h *harness) {
		h.Create("create with a parent id that is not a UUID: the API rejects it", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderId: not-a-uuid\n"))
		h.Delete("delete")
	},

	"create-parent-ref": func(h *harness) {
		seedParent(h, h.namespace, dfParentID)
		h.Create("create under a parent by DashboardsFolder CR", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderRef:\n    name: parent\n"))
		h.Reconcile("reconcile again")
	},

	"create-parent-ref-other-namespace": func(h *harness) {
		folders := createNamespace(h, namespaceName(h.namespace+"-folders"))
		seedParent(h, folders, dfOtherParentID)
		h.Create("create under a parent CR of another namespace", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderRef:\n    name: parent\n    namespace: "+folders+"\n"))
	},

	"create-parent-ref-missing": func(h *harness) {
		h.Create("do not call the API when the parent CR does not exist", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderRef:\n    name: parent\n"))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-parent-ref-without-id": func(h *harness) {
		createDashboardsFolder(h, h.namespace, "parent", "")
		h.Create("do not call the API before the parent CR has a status id", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderRef:\n    name: parent\n"))
		h.Delete("delete")
	},

	// The parent lookup reads the CR with the Kubernetes client and ignores the operator selector.
	"create-parent-ref-not-selected": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "dashboards"})})
		seedParent(h, h.namespace, dfParentID)
		h.Create("create under a parent CR that the selector does not match", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderRef:\n    name: parent\n", "team", "dashboards"))
	},

	"create-name-empty": func(h *harness) {
		h.Create("create with an empty name: the API rejects it", dashboardsFolderManifest("  name: \"\"\n"))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-name-too-long": func(h *harness) {
		h.Create("create with a name of 101 characters: the API rejects it", dashboardsFolderManifest(
			"  name: "+strings.Repeat("n", 101)+"\n"))
		h.Delete("delete")
	},

	// The API accepts a name of spaces (case "rejected names and parents").
	"create-name-blank": func(h *harness) {
		h.Create("create with a name of spaces: the API accepts it", dashboardsFolderManifest("  name: \"   \"\n"))
	},

	"create-name-longest": func(h *harness) {
		h.Create("create with a name of 100 characters", dashboardsFolderManifest(
			"  name: "+strings.Repeat("n", 100)+"\n"))
	},

	"create-rejected-duplicate-name": func(h *harness) {
		dashboardsFolders(h).seed(dfParentID, "Team dashboards", "")
		h.Create("create with a name that another folder has: the API rejects it", dashboardsFolderMinimalManifest)
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"update-rename": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		h.SetSpec("change the name", map[string]any{"name": "Team dashboards renamed"})
		h.Reconcile("reconcile again")
	},

	// Unlike Create, Replace accepts a name that another folder has (case "a cycle and a duplicate
	// name on replace"). Two folders then share a name.
	"update-rename-to-existing-name": func(h *harness) {
		dashboardsFolders(h).seed(dfParentID, "Taken", "")
		h.Create("create", dashboardsFolderMinimalManifest)
		h.SetSpec("change the name to one that another folder has: the API accepts it", map[string]any{"name": "Taken"})
		h.Reconcile("reconcile again")
	},

	"update-set-parent": func(h *harness) {
		dashboardsFolders(h).seed(dfParentID, "parent", "")
		h.Create("create at the top", dashboardsFolderMinimalManifest)
		h.SetSpec("move under a parent by backend id", map[string]any{"name": "Team dashboards", "parentFolderId": dfParentID})
	},

	"update-clear-parent": func(h *harness) {
		dashboardsFolders(h).seed(dfParentID, "parent", "")
		h.Create("create under a parent", dashboardsFolderManifest("  name: Team dashboards\n  parentFolderId: "+dfParentID+"\n"))
		h.SetSpec("remove the parent: the request leaves parentId out", map[string]any{"name": "Team dashboards"})
		h.Reconcile("reconcile again")
	},

	"update-parent-id-to-ref": func(h *harness) {
		seedParent(h, h.namespace, dfParentID)
		dashboardsFolders(h).seed(dfOtherParentID, "other", "")
		h.Create("create under a parent by backend id", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderId: "+dfOtherParentID+"\n"))
		h.SetSpec("switch to a parent CR", map[string]any{
			"name":            "Team dashboards",
			"parentFolderRef": map[string]any{"name": "parent"},
		})
	},

	// The Replace request reads the parent CR again on each reconcile.
	"update-parent-ref-follows-parent": func(h *harness) {
		seedParent(h, h.namespace, dfParentID)
		dashboardsFolders(h).seed(dfOtherParentID, "parent-2", "")
		h.Create("create under a parent CR", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderRef:\n    name: parent\n"))
		setStatusID(h, types.NamespacedName{Namespace: h.namespace, Name: "parent"}, dfOtherParentID)
		h.Reconcile("reconcile again after the parent CR got another status id")
	},

	"update-parent-ref-missing": func(h *harness) {
		seedParent(h, h.namespace, dfParentID)
		h.Create("create under a parent CR", dashboardsFolderManifest(
			"  name: Team dashboards\n  parentFolderRef:\n    name: parent\n"))
		deleteFolderCR(h, types.NamespacedName{Namespace: h.namespace, Name: "parent"})
		h.Reconcile("reconcile again after the parent CR was deleted: the API is not called")
		h.Delete("delete")
	},

	"update-parent-is-self": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		h.SetSpec("set the parent to the folder itself: the API rejects it", map[string]any{
			"name": "Team dashboards", "parentFolderId": h.StatusID(),
		})
		h.Reconcile("reconcile again")
	},

	// customId is never sent. The CRD makes it immutable once it is set.
	"custom-id-immutable": func(h *harness) {
		h.Create("create with a custom id", dashboardsFolderManifest(
			"  name: Team dashboards\n  customId: "+dfCustomID+"\n"))
		h.SetSpecRejected("change the custom id", map[string]any{"name": "Team dashboards", "customId": dfOtherCustomID})
		h.SetSpec("remove the custom id", map[string]any{"name": "Team dashboards"})
		h.SetSpec("set a custom id again", map[string]any{"name": "Team dashboards", "customId": dfOtherCustomID})
	},

	"delete": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).forget(h.StatusID())
		h.Delete("delete: the folder was deleted outside the operator")
	},

	"update-when-gone": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).forget(h.StatusID())
		h.SetSpec("change the name: the folder was deleted outside the operator", map[string]any{"name": "Team dashboards renamed"})
		h.Reconcile("reconcile again")
	},

	"resync-when-gone": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).forget(h.StatusID())
		h.Reconcile("resync: the folder was deleted outside the operator")
		h.Reconcile("reconcile again")
	},

	"delete-after-update-when-gone": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).forget(h.StatusID())
		h.SetSpec("change the name: the folder was deleted outside the operator", map[string]any{"name": "Team dashboards renamed"})
		h.Delete("delete")
		h.Reconcile("reconcile again")
	},

	// The shared reconciler creates when the status has no id, even while the CR is deleted. So the
	// controller's own "no id, nothing to delete" branch is not reached.
	"delete-without-id": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		clearStatusID(h, h.key)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"delete-without-id-and-remote-gone": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).forget(h.StatusID())
		clearStatusID(h, h.key)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	// The next answer is forced, so the scenario shows what the operator does with a failure of
	// each call.
	"update-rejected": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).answerNextError(http.MethodPut, http.StatusBadRequest, "the API rejects this request")
		h.SetSpec("change the name: the API rejects the Replace", map[string]any{"name": "Team dashboards renamed"})
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-rejected": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).answerNextError(http.MethodDelete, http.StatusBadRequest, "the API rejects this request")
		h.Delete("delete: the API rejects the Delete")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-folder-with-child": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).seed(dfOtherParentID, "child", h.StatusID())
		h.Delete("delete a folder that has a child folder")
		h.Reconcile("reconcile again")
	},

	"create-rejected-route-not-found": func(h *harness) {
		dashboardsFolders(h).answerNextRouteNotFound(http.MethodPost)
		h.Create("create: the API has no such route", dashboardsFolderMinimalManifest)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-route-not-found": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).answerNextRouteNotFound(http.MethodDelete)
		h.Delete("delete: the API has no such route")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// The next answer is replaced, as if the API sent a body that the operator cannot use.
	"create-empty-id": func(h *harness) {
		dashboardsFolders(h).answerNext(http.MethodPost, map[string]any{"folderId": ""})
		h.Create("create: the answer has an empty id", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).answerNext(http.MethodPost, map[string]any{})
		h.Reconcile("reconcile again: the answer has no id")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-malformed-response": func(h *harness) {
		dashboardsFolders(h).answerNext(http.MethodPost, "not a folder")
		h.Create("create: the answer cannot be decoded", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).answerNext(http.MethodPost, "not a folder")
		h.Reconcile("reconcile again: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).answerNext(http.MethodPut, "not a folder")
		h.Reconcile("update: the answer cannot be decoded")
		dashboardsFolders(h).answerNext(http.MethodPut, "not a folder")
		h.Reconcile("reconcile again: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-malformed-response": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		dashboardsFolders(h).answerNext(http.MethodDelete, "not a folder")
		h.Delete("delete: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"bad-status-id-update": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		setStatusID(h, h.key, "not-a-uuid")
		h.Reconcile("update with a status id that is not a UUID")
		h.Reconcile("reconcile again")
	},

	"bad-status-id-delete": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		setStatusID(h, h.key, "not-a-uuid")
		h.Delete("delete with a status id that is not a UUID")
		h.Reconcile("reconcile again")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", dashboardsFolderMinimalManifest)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "dashboards"})})
		h.Reconcile("reconcile after the selector stops matching the CR")
	},
}

// setStatusID writes status.id of a DashboardsFolder CR, as a user who edits the status could.
func setStatusID(h *harness, key types.NamespacedName, id string) {
	h.t.Helper()
	obj := getFolderCR(h, key)
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set DashboardsFolder status id: %v", err)
	}
}

// clearStatusID removes status.id of a DashboardsFolder CR, as a user who edits the status could.
func clearStatusID(h *harness, key types.NamespacedName) {
	h.t.Helper()
	obj := getFolderCR(h, key)
	unstructured.RemoveNestedField(obj.Object, "status", "id")
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("clear DashboardsFolder status id: %v", err)
	}
}

// deleteFolderCR deletes a DashboardsFolder CR that has no finalizer, since no reconciler ran for it.
func deleteFolderCR(h *harness, key types.NamespacedName) {
	h.t.Helper()
	if err := h.client.Delete(context.Background(), getFolderCR(h, key)); err != nil {
		h.t.Fatalf("delete DashboardsFolder %s: %v", key, err)
	}
}

func getFolderCR(h *harness, key types.NamespacedName) *unstructured.Unstructured {
	h.t.Helper()
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(h.gvk)
	if err := h.client.Get(context.Background(), key, obj); err != nil {
		h.t.Fatalf("get DashboardsFolder %s: %v", key, err)
	}
	return obj
}

// dashboardsFolderAdmission lists the CRs that TestAdmission creates. Each case has one reason
// that the CRD accepts or rejects it.
func dashboardsFolderAdmission() map[string]string {
	spec := func(fields string) string { return dashboardsFolderManifest(fields) }
	return map[string]string{
		"name only":                      spec("  name: Team dashboards\n"),
		"no name":                        spec("  customId: " + dfCustomID + "\n"),
		"name of spaces":                 spec("  name: \"   \"\n"),
		"empty name":                     spec("  name: \"\"\n"),
		"name of 101 characters":         spec("  name: " + strings.Repeat("n", 101) + "\n"),
		"custom id":                      spec("  name: f\n  customId: " + dfCustomID + "\n"),
		"custom id in upper case":        spec("  name: f\n  customId: " + strings.ToUpper(dfCustomID) + "\n"),
		"custom id of version 1":         spec("  name: f\n  customId: efa8b720-c5d9-1c08-bb27-e3de0837a8d2\n"),
		"custom id of version 5":         spec("  name: f\n  customId: efa8b720-c5d9-5c08-bb27-e3de0837a8d2\n"),
		"custom id of version 7":         spec("  name: f\n  customId: 0190a8b7-c5d9-7c08-bb27-e3de0837a8d2\n"),
		"custom id with variant c":       spec("  name: f\n  customId: efa8b720-c5d9-4c08-cb27-e3de0837a8d2\n"),
		"custom id that is not a UUID":   spec("  name: f\n  customId: not-a-uuid\n"),
		"empty custom id":                spec("  name: f\n  customId: \"\"\n"),
		"parent id":                      spec("  name: f\n  parentFolderId: " + dfParentID + "\n"),
		"empty parent id":                spec("  name: f\n  parentFolderId: \"\"\n"),
		"parent id that is not a UUID":   spec("  name: f\n  parentFolderId: not-a-uuid\n"),
		"parent ref":                     spec("  name: f\n  parentFolderRef:\n    name: parent\n"),
		"parent ref in a namespace":      spec("  name: f\n  parentFolderRef:\n    name: parent\n    namespace: other\n"),
		"parent ref without a name":      spec("  name: f\n  parentFolderRef: {}\n"),
		"parent ref with an empty name":  spec("  name: f\n  parentFolderRef:\n    name: \"\"\n"),
		"parent id and parent ref":       spec("  name: f\n  parentFolderId: " + dfParentID + "\n  parentFolderRef:\n    name: parent\n"),
		"empty parent id and parent ref": spec("  name: f\n  parentFolderId: \"\"\n  parentFolderRef:\n    name: parent\n"),
		"custom id and a parent ref":     spec("  name: f\n  customId: " + dfCustomID + "\n  parentFolderRef:\n    name: parent\n"),
		"unknown field is not an error":  spec("  name: f\n  description: ignored\n"),
		"empty spec":                     "apiVersion: coralogix.com/v1alpha1\nkind: DashboardsFolder\nmetadata:\n  name: folder\nspec: {}\n",
	}
}

// dashboardsFolderFake is the dashboard folders API, written from the SDK's openapi.yaml.
//
//	POST   /dashboards/folders/v1        {folder: {id?, name, parentId?}} -> {folderId}
//	PUT    /dashboards/folders/v1        {folder: {id, name, parentId?}}  -> {}
//	GET    /dashboards/folders/v1/{id}   -> {folder: {id, name, parentId?}}
//	GET    /dashboards/folders/v1        -> {folder: [...]}
//	DELETE /dashboards/folders/v1/{id}   -> {}
//
// The status codes, error texts, and stored values follow fidelity/recorded.json, which the real
// API answered. Each rule names the recorded case that shows it. A rule with no case name is not
// recorded: it is the simplest answer that the scenarios need.
type dashboardsFolderFake struct {
	mu      sync.Mutex
	folders map[string]*storedFolder
	// next holds one forced answer per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]forcedAnswer
}

type storedFolder struct {
	name     string
	parentID string
}

type forcedAnswer struct {
	status int
	body   any
}

func newDashboardsFolderFake() apiFake {
	return &dashboardsFolderFake{folders: map[string]*storedFolder{}, next: map[string]forcedAnswer{}}
}

// answerNext makes the next request with the method get a 200 answer with this body.
func (f *dashboardsFolderFake) answerNext(method string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = forcedAnswer{status: http.StatusOK, body: body}
}

// answerNextError makes the next request with the method get an error answer.
func (f *dashboardsFolderFake) answerNextError(method string, status int, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := apiError(status, reason)
	f.next[method] = forcedAnswer{status: st, body: body}
}

// answerNextRouteNotFound makes the next request with the method get the answer for a route that
// the API does not have.
func (f *dashboardsFolderFake) answerNextRouteNotFound(method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := routeNotFound()
	f.next[method] = forcedAnswer{status: st, body: body}
}

// seed stores a folder, as if another client made it.
func (f *dashboardsFolderFake) seed(id, name, parentID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.folders[id] = &storedFolder{name: name, parentID: parentID}
}

// forget deletes a folder without a request, as if it was deleted outside the operator.
func (f *dashboardsFolderFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.folders, id)
}

var folderUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// folderAnswer is an error answer of the folders API: {"code": 400, "message": "Bad Request: <text>"}.
func folderAnswer(status int, prefix, text string) (int, any) {
	return status, map[string]any{"code": status, "message": prefix + ": " + text}
}

func folderBadRequest(text string) (int, any) {
	return folderAnswer(http.StatusBadRequest, "Bad Request", text)
}

// folderNotFound is the answer for an id that no folder has. The text starts with "Not Found:" and
// names the id, so the SDK reads it as a missing resource. Get says "dashboard folder", Replace and
// Delete say "folder" (cases "create, get, replace, ..." and "ids that are unknown or malformed").
func folderNotFound(noun, id string) (int, any) {
	return folderAnswer(http.StatusNotFound, "Not Found", fmt.Sprintf("%s with id DashboardFolderId(%s) does not exist", noun, id))
}

func (f *dashboardsFolderFake) handle(method, path string, _ url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if forced, ok := f.next[method]; ok {
		delete(f.next, method)
		return forced.status, forced.body
	}
	rest, ok := strings.CutPrefix(path, dashboardsFolderPath)
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
	case method == http.MethodGet && rest == "":
		list := []any{}
		for id, stored := range f.folders {
			list = append(list, stored.answer(id))
		}
		return http.StatusOK, map[string]any{"folder": list}
	case method == http.MethodGet && id != "" && !strings.Contains(id, "/"):
		if !folderUUIDPattern.MatchString(id) {
			return folderBadRequest("invalid uuid")
		}
		stored, ok := f.folders[id]
		if !ok {
			return folderNotFound("dashboard folder", id)
		}
		return http.StatusOK, map[string]any{"folder": stored.answer(id)}
	case method == http.MethodDelete && id != "" && !strings.Contains(id, "/"):
		return f.delete(id)
	}
	return routeNotFound()
}

// create follows the cases "rejected names and parents", "a custom id", and "a parent id that is
// an empty string". The API keeps a custom folder id. A second folder with the same id or the same
// name is a conflict. A name of spaces is accepted. A folder that is not an object, or has no name,
// has its own message.
func (f *dashboardsFolderFake) create(in map[string]any) (int, any) {
	folder, ok := in["folder"].(map[string]any)
	if !ok {
		return folderBadRequest("folder is required")
	}
	name, status, errBody := checkFolderName(folder)
	if status != 0 {
		return status, errBody
	}
	parentID, hasParent := folder["parentId"].(string)
	if hasParent {
		if status, errBody := f.checkParent(parentID); status != 0 {
			return status, errBody
		}
	}
	id, _ := folder["id"].(string)
	if id == "" {
		id = newUUID()
	}
	if f.conflicts(id, name) {
		return folderAnswer(http.StatusConflict, "Already Exists",
			fmt.Sprintf("folder with id DashboardFolderId(%s) or name %s already exist in your company", id, name))
	}
	f.folders[id] = &storedFolder{name: name, parentID: parentID}
	return http.StatusOK, map[string]any{"folderId": id}
}

// replace follows the cases "create, get, replace, ...", "a parent, a child, moving the child",
// and "a cycle and a duplicate name on replace". It replaces the folder as a whole: an omitted
// parentId moves the folder to the top. It accepts a name that another folder has.
func (f *dashboardsFolderFake) replace(in map[string]any) (int, any) {
	folder, _ := in["folder"].(map[string]any)
	id, _ := folder["id"].(string)
	if !folderUUIDPattern.MatchString(id) {
		return folderBadRequest("invalid uuid")
	}
	if _, ok := f.folders[id]; !ok {
		return folderNotFound("folder", id)
	}
	name, status, errBody := checkFolderName(folder)
	if status != 0 {
		return status, errBody
	}
	parentID, hasParent := folder["parentId"].(string)
	if hasParent {
		if status, errBody := f.checkParent(parentID); status != 0 {
			return status, errBody
		}
		if parentID == id {
			return folderBadRequest(fmt.Sprintf("folder_id and parent_id can't be equal (%s)", id))
		}
		for ancestor := parentID; ancestor != ""; ancestor = f.folders[ancestor].parentID {
			if ancestor == id {
				return folderBadRequest(fmt.Sprintf("cannot move folder DashboardFolderId(%s) under its own descendant DashboardFolderId(%s)", id, parentID))
			}
		}
	}
	f.folders[id] = &storedFolder{name: name, parentID: parentID}
	return http.StatusOK, map[string]any{}
}

// delete follows the cases "delete a parent that has a child" and "create, get, replace, ...". A
// folder that has a child is not deleted: the answer is 403, not 400.
func (f *dashboardsFolderFake) delete(id string) (int, any) {
	if !folderUUIDPattern.MatchString(id) {
		return folderBadRequest("invalid uuid")
	}
	if _, ok := f.folders[id]; !ok {
		return folderNotFound("folder", id)
	}
	for _, stored := range f.folders {
		if stored.parentID == id {
			return folderAnswer(http.StatusForbidden, "Permission Denied",
				fmt.Sprintf("Trying to delete a non empty folder DashboardFolderId(%s)", id))
		}
	}
	delete(f.folders, id)
	return http.StatusOK, map[string]any{}
}

// checkFolderName checks the name of a create or replace request. A missing name and an empty name
// have different messages. The API rejects a name over 100 characters, and it does not cut it.
func checkFolderName(folder map[string]any) (string, int, any) {
	name, ok := folder["name"].(string)
	switch {
	case !ok:
		status, body := folderBadRequest("folder name not found")
		return "", status, body
	case name == "":
		status, body := folderBadRequest("folder name cannot be empty")
		return "", status, body
	case len([]rune(name)) > 100:
		status, body := folderBadRequest("folder name length is bigger than 100")
		return "", status, body
	}
	return name, 0, nil
}

// checkParent checks a parent id of a create or replace request: an empty string is not a UUID.
func (f *dashboardsFolderFake) checkParent(parentID string) (int, any) {
	if !folderUUIDPattern.MatchString(parentID) {
		return folderBadRequest("invalid uuid")
	}
	if _, ok := f.folders[parentID]; !ok {
		return folderBadRequest(fmt.Sprintf("parent folder id DashboardFolderId(%s) for company CompanyId(<redacted>) does not exist", parentID))
	}
	return 0, nil
}

// conflicts is true when a folder has the id or the name.
func (f *dashboardsFolderFake) conflicts(id, name string) bool {
	if _, ok := f.folders[id]; ok {
		return true
	}
	for _, stored := range f.folders {
		if stored.name == name {
			return true
		}
	}
	return false
}

func (s *storedFolder) answer(id string) map[string]any {
	out := map[string]any{"id": id, "name": s.name, "restrictedGroupIds": []any{}}
	if s.parentID != "" {
		out["parentId"] = s.parentID
	}
	return out
}
