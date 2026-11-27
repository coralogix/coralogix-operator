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
	"os"
	"path/filepath"
	"strconv"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "group",
		crdFile:  "coralogix.com_groups.yaml",
		resource: "groups",
		// The controller reads the CustomRole and Scope that the spec names.
		rbacExtra: []string{"customroles", "scopes"},
		newFake:   newGroupFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.GroupReconciler{GroupsClient: cs.Groups(), UsersClient: cs.Users(), Interval: reconcileInterval}
		},
		scenarios: groupScenarios,
		admission: groupAdmission(),
		fidelity: fidelityConfig{
			captureField: "group.groupId",
			numericID:    true,
			deletePath:   func(id string) string { return groupPath + "/" + id },
			recordVars:   groupRecordVars,
			fakeVars: map[string]string{
				"userA": groupFakeUserA, "userB": groupFakeUserB,
				"scopeId": groupFakeScope, "roleId": strconv.FormatInt(groupFakeRole, 10),
			},
		},
	})
}

// groupRecordVars reads two users of the team, and makes a scope and a custom role for the
// cases: a group can name only things that exist. It removes the scope and the role when the
// recording ends. The users are the team's own: the recording hides their ids.
func groupRecordVars(t *testing.T, base, apiKey string) map[string]string {
	t.Helper()
	status, resp, err := sendRecordRequest(base, apiKey, http.MethodGet, usersPath+"?page_size=100", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("search users: %d %v", status, err)
	}
	var ids []string
	for _, user := range listOf(resp, "users") {
		if id, _ := user["userId"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 {
		t.Fatal("the team needs two users to record the member cases")
	}

	status, resp, err = sendRecordRequest(base, apiKey, http.MethodPost, scopePath,
		map[string]any{"displayName": "cxo-fidelity-group-scope", "defaultExpression": "<v1>true"})
	if err != nil || !succeeded(status) {
		t.Fatalf("create the scope: %d %v", status, err)
	}
	scopeID := stringAt(resp, "scope.id")
	t.Cleanup(func() { removeGroupFixture(t, base, apiKey, scopePath+"/"+scopeID) })

	status, resp, err = sendRecordRequest(base, apiKey, http.MethodPut, customRolePath, map[string]any{
		"name": "cxo-fidelity-group-role", "description": "Fidelity role",
		"parentRoleName": "Standard User", "permissions": []string{"team-actions:UpdateConfig"},
	})
	if err != nil || !succeeded(status) {
		t.Fatalf("create the custom role (delete a role named cxo-fidelity-group-role if one was left): %d %s %v", status, compact(resp), err)
	}
	roleID, _ := valueAt(resp, "id").(float64)
	role := strconv.FormatInt(int64(roleID), 10)
	t.Cleanup(func() { removeGroupFixture(t, base, apiKey, customRolePath+"/"+role) })

	return map[string]string{"userA": ids[0], "userB": ids[1], "scopeId": scopeID, "roleId": role}
}

func removeGroupFixture(t *testing.T, base, apiKey, path string) {
	t.Helper()
	status, resp, err := sendRecordRequest(base, apiKey, http.MethodDelete, path, nil)
	if problem := cleanupProblem(status, resp, err); problem != "" {
		t.Errorf("cleanup: DELETE %s %s; delete it by hand", path, problem)
	}
}

const (
	groupSpec = `  name: baseline-group
  description: Baseline group
  groupType: closed
  members:
    - userName: member-a@example.com
  customRole:
    resourceRef:
      name: role
  scope:
    resourceRef:
      name: scope
`
	groupMinimalSpec = "  name: baseline-group\n"
)

func groupManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: Group\nmetadata:\n  name: group\nspec:\n" + spec
}

func groups(h *harness) *groupFake {
	h.t.Helper()
	f, ok := h.API().(*groupFake)
	if !ok {
		h.t.Fatal("expected *groupFake")
	}
	return f
}

// groupSpecMap is a whole spec. role and scope name the CR that the spec references, "" for none.
func groupSpecMap(name string, description, groupType *string, members []string, role, scope string) map[string]any {
	spec := map[string]any{"name": name}
	if description != nil {
		spec["description"] = *description
	}
	if groupType != nil {
		spec["groupType"] = *groupType
	}
	if len(members) > 0 {
		items := make([]any, 0, len(members))
		for _, m := range members {
			items = append(items, map[string]any{"userName": m})
		}
		spec["members"] = items
	}
	if role != "" {
		spec["customRole"] = map[string]any{"resourceRef": map[string]any{"name": role}}
	}
	if scope != "" {
		spec["scope"] = map[string]any{"resourceRef": map[string]any{"name": scope}}
	}
	return spec
}

// groupDependency creates the CustomRole or Scope that a Group names, from the published
// sample, with the status id that its controller would have written. An empty id leaves the
// status empty.
func groupDependency(h *harness, sample, name, namespace string, labels map[string]string, id string) {
	h.t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, sample))
	if err != nil {
		h.t.Fatal(err)
	}
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(raw, &obj.Object); err != nil {
		h.t.Fatal(err)
	}
	obj.SetName(name)
	obj.SetNamespace(namespace)
	obj.SetLabels(labels)
	if err := h.client.Create(context.Background(), obj); err != nil {
		h.t.Fatalf("create %s: %v", name, err)
	}
	if id == "" {
		return
	}
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set the status id of %s: %v", name, err)
	}
}

const (
	groupRoleSample  = "config/samples/v1alpha1/customroles/example-custom-role.yaml"
	groupScopeSample = "config/samples/v1alpha1/scopes/example-scope.yaml"
)

// groupDeps creates the CustomRole "role" and the Scope "scope" that the baseline spec names.
func groupDeps(h *harness) {
	h.t.Helper()
	groupDependency(h, groupRoleSample, "role", h.namespace, nil, strconv.FormatInt(groupFakeRole, 10))
	groupDependency(h, groupScopeSample, "scope", h.namespace, nil, groupFakeScope)
}

func groupNamespace(h *harness, suffix string) string {
	h.t.Helper()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName(h.namespace + "-" + suffix)}}
	if err := h.client.Create(context.Background(), ns); err != nil {
		h.t.Fatalf("create namespace: %v", err)
	}
	return ns.Name
}

func setGroupStatusID(h *harness, id string) {
	h.t.Helper()
	obj := h.get()
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatal(err)
	}
}

func groupStatusInt(h *harness) int64 {
	h.t.Helper()
	id, err := strconv.ParseInt(h.StatusID(), 10, 64)
	if err != nil {
		h.t.Fatal(err)
	}
	return id
}

// manyUsers returns n users, with the given username at the position at.
func manyUsers(n, at int, username string) []groupUser {
	users := make([]groupUser, n)
	for i := range users {
		users[i] = groupUser{id: fmt.Sprintf("00000000-0000-4000-8000-%012d", i), username: fmt.Sprintf("user-%03d@example.com", i)}
	}
	users[at].username = username
	return users
}

var groupScenarios = map[string]func(h *harness){
	// The published sample names a Scope, a CustomRole, and two users.
	"create-sample": func(h *harness) {
		groupDependency(h, groupRoleSample, "custom-role-sample", h.namespace, nil, strconv.FormatInt(groupFakeRole, 10))
		groupDependency(h, groupScopeSample, "scope-sample", h.namespace, nil, groupFakeScope)
		groups(h).setUsers(groupUser{id: groupFakeUserA, username: "example@coralogix.com"}, groupUser{id: groupFakeUserB, username: "example2@coralogix.com"})
		h.CreateFromFile("create the released sample", "config/samples/v1alpha1/groups/example-group.yaml")
		h.Reconcile("resync: update sends the whole spec")
		h.RecordRemote("the remote group", groups(h).snapshot())
	},
	"create-minimal": func(h *harness) {
		h.Create("create with a name only: no users are searched", groupManifest(groupMinimalSpec))
		h.Reconcile("resync: update sets no role and clears the scope")
		h.Reconcile("resync again")
		h.RecordRemote("the remote group", groups(h).snapshot())
	},
	"create-full": func(h *harness) {
		groupDeps(h)
		h.Create("create with a role, a scope, and a member", groupManifest(groupSpec))
		h.Reconcile("resync")
		h.RecordRemote("the remote group", groups(h).snapshot())
	},
	"create-without-spec": func(h *harness) {
		h.Create("create without spec", "apiVersion: coralogix.com/v1alpha1\nkind: Group\nmetadata:\n  name: group\n")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	// The user name matches without case, and a user listed twice is sent twice.
	"create-members-case-and-duplicates": func(h *harness) {
		h.Create("members by name without case, one twice", groupManifest("  name: baseline-group\n  members:\n    - userName: MEMBER-A@Example.com\n    - userName: member-b@example.com\n    - userName: member-a@example.com\n"))
		h.RecordRemote("the remote group stores each user once", groups(h).snapshot())
	},
	"create-user-not-found": func(h *harness) {
		h.Create("a member that no user has: no group is created", groupManifest("  name: baseline-group\n  members:\n    - userName: member-a@example.com\n    - userName: nobody@example.com\n    - userName: nobody-either@example.com\n"))
		h.Reconcile("retry")
		h.RecordRemote("no remote group", groups(h).snapshot())
	},
	// The search reads every page of 100 users before it creates the group.
	"create-users-paginated": func(h *harness) {
		groups(h).setUsers(manyUsers(250, 230, "member-a@example.com")...)
		h.Create("the member is on the third page", groupManifest("  name: baseline-group\n  members:\n    - userName: member-a@example.com\n"))
		h.RecordRemote("the remote group", groups(h).snapshot())
	},
	"create-user-search-rejected": func(h *harness) {
		groups(h).answerNext(http.MethodGet, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Create("the users search fails: no group is created", groupManifest("  name: baseline-group\n  members:\n    - userName: member-a@example.com\n"))
		h.Reconcile("retry")
	},
	"create-role-namespace": func(h *harness) {
		other := groupNamespace(h, "other")
		groupDependency(h, groupRoleSample, "role", other, nil, strconv.FormatInt(groupFakeRole, 10))
		groupDependency(h, groupScopeSample, "scope", other, nil, groupFakeScope)
		h.Create("the role and the scope are in another namespace", groupManifest(fmt.Sprintf(`  name: baseline-group
  customRole:
    resourceRef:
      name: role
      namespace: %s
  scope:
    resourceRef:
      name: scope
      namespace: %s
`, other, other)))
	},
	// Helm chart 1.0 wrote spec.customRoles. The first entry is the role when customRole is not set.
	"create-deprecated-custom-roles": func(h *harness) {
		groupDeps(h)
		groupDependency(h, groupRoleSample, "second-role", h.namespace, nil, "7002")
		groups(h).addRole(7002, groupRole{name: "second", description: "Second role"})
		h.Create("customRoles: the first entry is used", groupManifest("  name: baseline-group\n  customRoles:\n    - resourceRef:\n        name: role\n    - resourceRef:\n        name: second-role\n"))
		h.RecordRemote("the remote group has the first role", groups(h).snapshot())
	},
	"create-custom-role-wins": func(h *harness) {
		groupDeps(h)
		groupDependency(h, groupRoleSample, "second-role", h.namespace, nil, "7002")
		groups(h).addRole(7002, groupRole{name: "second", description: "Second role"})
		h.Create("customRole and customRoles: customRole is used", groupManifest("  name: baseline-group\n  customRole:\n    resourceRef:\n      name: second-role\n  customRoles:\n    - resourceRef:\n        name: role\n"))
		h.RecordRemote("the remote group has the role of customRole", groups(h).snapshot())
	},
	"create-missing-custom-role": func(h *harness) {
		groupDependency(h, groupScopeSample, "scope", h.namespace, nil, groupFakeScope)
		h.Create("the CustomRole does not exist: no request", groupManifest(groupSpec))
		h.Reconcile("retry")
	},
	"create-missing-scope": func(h *harness) {
		groupDependency(h, groupRoleSample, "role", h.namespace, nil, strconv.FormatInt(groupFakeRole, 10))
		h.Create("the Scope does not exist: no request", groupManifest(groupSpec))
		h.Reconcile("retry")
	},
	"create-custom-role-without-id": func(h *harness) {
		groupDependency(h, groupRoleSample, "role", h.namespace, nil, "")
		groupDependency(h, groupScopeSample, "scope", h.namespace, nil, groupFakeScope)
		h.Create("the CustomRole has no status id: no request", groupManifest(groupSpec))
	},
	"create-scope-without-id": func(h *harness) {
		groupDependency(h, groupRoleSample, "role", h.namespace, nil, strconv.FormatInt(groupFakeRole, 10))
		groupDependency(h, groupScopeSample, "scope", h.namespace, nil, "")
		h.Create("the Scope has no status id: no request", groupManifest(groupSpec))
	},
	"create-custom-role-bad-id": func(h *harness) {
		groupDependency(h, groupRoleSample, "role", h.namespace, nil, "not-a-number")
		groupDependency(h, groupScopeSample, "scope", h.namespace, nil, groupFakeScope)
		h.Create("the CustomRole status id is not a number: no request", groupManifest(groupSpec))
	},
	// The label selector applies to the Group by an event predicate that a direct reconcile does
	// not run. A role or a scope that the selector does not match is refused.
	"create-custom-role-selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "a"})})
		groupDependency(h, groupRoleSample, "role", h.namespace, nil, strconv.FormatInt(groupFakeRole, 10))
		groupDependency(h, groupScopeSample, "scope", h.namespace, map[string]string{"team": "a"}, groupFakeScope)
		h.Create("the CustomRole does not match the selector: no request",
			"apiVersion: coralogix.com/v1alpha1\nkind: Group\nmetadata:\n  name: group\n  labels:\n    team: a\nspec:\n"+groupSpec)
	},
	"create-scope-selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "a"})})
		groupDependency(h, groupRoleSample, "role", h.namespace, map[string]string{"team": "a"}, strconv.FormatInt(groupFakeRole, 10))
		groupDependency(h, groupScopeSample, "scope", h.namespace, nil, groupFakeScope)
		h.Create("the Scope does not match the selector: no request",
			"apiVersion: coralogix.com/v1alpha1\nkind: Group\nmetadata:\n  name: group\n  labels:\n    team: a\nspec:\n"+groupSpec)
	},
	"create-unknown-scope": func(h *harness) {
		groupDependency(h, groupScopeSample, "scope", h.namespace, nil, "11111111-1111-4111-8111-111111111111")
		h.Create("the API does not know the scope", groupManifest("  name: baseline-group\n  scope:\n    resourceRef:\n      name: scope\n"))
		h.Reconcile("retry")
	},
	"create-unknown-role": func(h *harness) {
		groupDependency(h, groupRoleSample, "role", h.namespace, nil, "99999")
		h.Create("the API does not know the role", groupManifest("  name: baseline-group\n  customRole:\n    resourceRef:\n      name: role\n"))
		h.Reconcile("retry")
	},
	"create-name-conflict": func(h *harness) {
		groups(h).put(900, groupRecord{name: "baseline-group", groupType: "GROUP_TYPE_OPEN"})
		h.Create("the name is taken: 409", groupManifest(groupMinimalSpec))
		h.Reconcile("retry")
		h.Delete("delete the CR without a remote id")
		h.RecordRemote("the other group remains", groups(h).snapshot())
	},
	"create-empty-name": func(h *harness) {
		h.Create("admission accepts an empty name, the API rejects it", groupManifest("  name: \"\"\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-group-types": func(h *harness) {
		h.Create("restricted", groupManifest("  name: baseline-group\n  groupType: restricted\n"))
		h.RecordRemote("the remote type", groups(h).snapshot())
		h.SetSpec("unspecified: the API stores open", map[string]any{"name": "baseline-group", "groupType": "unspecified"})
		h.RecordRemote("the remote type is open", groups(h).snapshot())
	},
	"create-rejected": func(h *harness) {
		groups(h).answerNext(http.MethodPost, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Create("create returns an API error", groupManifest(groupMinimalSpec))
		h.Reconcile("retry creates the group")
	},
	"update-fields": func(h *harness) {
		groupDeps(h)
		h.Create("create", groupManifest(groupSpec))
		groups(h).setUsers(groupUser{id: groupFakeUserA, username: "member-a@example.com"}, groupUser{id: groupFakeUserB, username: "member-b@example.com"})
		h.SetSpec("rename, change the description, the type, and the members", groupSpecMap("renamed-group", ptr.To("Changed"), ptr.To("open"), []string{"member-b@example.com"}, "role", "scope"))
		h.RecordRemote("the remote group", groups(h).snapshot())
	},
	"update-clear-description": func(h *harness) {
		h.Create("create", groupManifest(groupSpec))
		h.SetSpec("remove the description: update omits it and the API clears it", groupSpecMap("baseline-group", nil, ptr.To("closed"), nil, "", ""))
		h.RecordRemote("the remote description is cleared", groups(h).snapshot())
	},
	"update-empty-description": func(h *harness) {
		h.Create("create", groupManifest(groupSpec))
		h.SetSpec("an empty description is sent and stored", groupSpecMap("baseline-group", ptr.To(""), ptr.To("closed"), nil, "", ""))
		h.RecordRemote("the remote description is empty", groups(h).snapshot())
	},
	"update-remove-members": func(h *harness) {
		groupDeps(h)
		h.Create("create with a member", groupManifest(groupSpec))
		h.SetSpec("remove the members: update sets an empty list", groupSpecMap("baseline-group", ptr.To("Baseline group"), ptr.To("closed"), nil, "role", "scope"))
		h.RecordRemote("the remote group has no users", groups(h).snapshot())
	},
	// Without a CustomRole reference, update sends no role update. The remote role stays.
	"update-remove-custom-role": func(h *harness) {
		groupDeps(h)
		h.Create("create with a role", groupManifest(groupSpec))
		h.SetSpec("remove the custom role: the remote role stays", groupSpecMap("baseline-group", ptr.To("Baseline group"), ptr.To("closed"), []string{"member-a@example.com"}, "", "scope"))
		h.RecordRemote("the remote group keeps its role", groups(h).snapshot())
	},
	// Without a Scope reference, update sends a scope update with no value. The API clears the scope.
	"update-remove-scope": func(h *harness) {
		groupDeps(h)
		h.Create("create with a scope", groupManifest(groupSpec))
		h.SetSpec("remove the scope: the remote scope is cleared", groupSpecMap("baseline-group", ptr.To("Baseline group"), ptr.To("closed"), []string{"member-a@example.com"}, "role", ""))
		h.RecordRemote("the remote group has no scope", groups(h).snapshot())
	},
	"update-change-custom-role": func(h *harness) {
		groupDeps(h)
		groupDependency(h, groupRoleSample, "second-role", h.namespace, nil, "7002")
		groups(h).addRole(7002, groupRole{name: "second", description: "Second role"})
		h.Create("create", groupManifest(groupSpec))
		h.SetSpec("another custom role", groupSpecMap("baseline-group", ptr.To("Baseline group"), ptr.To("closed"), []string{"member-a@example.com"}, "second-role", "scope"))
		h.RecordRemote("the remote role", groups(h).snapshot())
	},
	// An omitted groupType is not sent, and the API keeps the stored type.
	"update-remove-group-type": func(h *harness) {
		h.Create("create a closed group", groupManifest("  name: baseline-group\n  groupType: closed\n"))
		h.SetSpec("remove the type: the remote type stays", map[string]any{"name": "baseline-group"})
		h.RecordRemote("the remote type", groups(h).snapshot())
	},
	"update-rename-conflict": func(h *harness) {
		groups(h).put(902, groupRecord{name: "taken-group", groupType: "GROUP_TYPE_OPEN"})
		h.Create("create", groupManifest(groupMinimalSpec))
		h.SetSpec("rename to a name that another group has: the API answers 500", map[string]any{"name": "taken-group"})
		h.Reconcile("retry")
	},
	"update-rejected": func(h *harness) {
		h.Create("create", groupManifest(groupMinimalSpec))
		groups(h).answerNext(http.MethodPut, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.SetSpec("update returns an API error", map[string]any{"name": "renamed-group"})
		h.Reconcile("retry")
		h.RecordRemote("the remote group", groups(h).snapshot())
	},
	"update-user-not-found": func(h *harness) {
		h.Create("create", groupManifest(groupMinimalSpec))
		h.SetSpec("a member that no user has: no update request", groupSpecMap("baseline-group", nil, nil, []string{"nobody@example.com"}, "", ""))
		h.RecordRemote("the remote group is unchanged", groups(h).snapshot())
	},
	"update-when-gone": func(h *harness) {
		h.Create("create", groupManifest(groupMinimalSpec))
		groups(h).forget(groupStatusInt(h))
		h.SetSpec("update after remote deletion: the 404 removes status.id", map[string]any{"name": "renamed-group"})
		h.Reconcile("recreate")
	},
	// The update 404 removes status.id. A delete before the retry finds a CR with a finalizer and
	// no status id: it names no remote group, and no group is created.
	"delete-after-update-gone": func(h *harness) {
		h.Create("create", groupManifest(groupMinimalSpec))
		groups(h).forget(groupStatusInt(h))
		h.SetSpec("update after remote deletion: the 404 removes status.id", map[string]any{"name": "renamed-group"})
		h.Delete("delete a CR that has a finalizer and no status id")
		h.RecordRemote("no remote group exists", groups(h).snapshot())
	},
	"bad-status-id": func(h *harness) {
		h.Create("create", groupManifest(groupMinimalSpec))
		setGroupStatusID(h, "not-an-integer")
		h.Reconcile("update cannot parse status id")
		h.Delete("delete cannot parse status id")
	},
	"delete": func(h *harness) {
		h.Create("create", groupManifest(groupSpec))
		h.Delete("delete")
		h.Reconcile("delete again")
		h.RecordRemote("no remote group exists", groups(h).snapshot())
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", groupManifest(groupMinimalSpec))
		groups(h).forget(groupStatusInt(h))
		h.Delete("delete after remote deletion: the API answers 404")
	},
	"delete-rejected": func(h *harness) {
		h.Create("create", groupManifest(groupMinimalSpec))
		groups(h).answerNext(http.MethodDelete, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Delete("delete returns an API error")
		h.Reconcile("retry deletes the group")
	},
	"selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
		h.Create("create: a direct reconcile does not run the event predicate", groupManifest(groupMinimalSpec))
		h.Reconcile("the selector does not match: delete the remote group and clear the status")
		h.Delete("delete")
	},
	// A chart upgrade: the released operator created the group and wrote status.id and its
	// finalizer. The operator updates that group, creates none, and deletes it on delete.
	"upgrade-from-released": func(h *harness) {
		groupDeps(h)
		groups(h).put(4242, groupRecord{
			name: "baseline-group", description: ptr.To("Baseline group"), groupType: "GROUP_TYPE_CLOSED",
			userIDs: []string{groupFakeUserA},
		})
		h.CreateReleased("the released operator made the group: reconcile the CR", groupManifest(groupSpec),
			[]string{"group.coralogix.com/finalizer"}, map[string]any{"id": "4242", "printableStatus": "RemoteSynced"})
		if got := h.StatusID(); got != "4242" {
			h.t.Fatalf("status.id changed from 4242 to %s", got)
		}
		h.Reconcile("reconcile again")
		h.Delete("delete: the group is deleted by its released id")
		h.RecordRemote("no group remains", groups(h).snapshot())
	},
	// A CR that Helm chart 1.0 stored holds customRoles, and no status.printableStatus.
	"upgrade-from-chart-1-0": func(h *harness) {
		groupDeps(h)
		groups(h).put(4243, groupRecord{name: "baseline-group", groupType: "GROUP_TYPE_OPEN"})
		h.CreateReleased("the released operator made the group from customRoles", groupManifest("  name: baseline-group\n  customRoles:\n    - resourceRef:\n        name: role\n"),
			[]string{"group.coralogix.com/finalizer"}, map[string]any{"id": "4243"})
		h.RecordRemote("the remote group has the role", groups(h).snapshot())
	},
}

func groupAdmission() map[string]string {
	return map[string]string{
		"sample":                       groupManifest(groupSpec),
		"without spec":                 "apiVersion: coralogix.com/v1alpha1\nkind: Group\nmetadata:\n  name: group\n",
		"empty spec":                   groupManifest("  {}\n"),
		"name absent":                  groupManifest("  description: d\n"),
		"name empty":                   groupManifest("  name: \"\"\n"),
		"description empty":            groupManifest("  name: g\n  description: \"\"\n"),
		"group type open":              groupManifest("  name: g\n  groupType: open\n"),
		"group type closed":            groupManifest("  name: g\n  groupType: closed\n"),
		"group type restricted":        groupManifest("  name: g\n  groupType: restricted\n"),
		"group type unspecified":       groupManifest("  name: g\n  groupType: unspecified\n"),
		"group type upper case":        groupManifest("  name: g\n  groupType: OPEN\n"),
		"group type unknown":           groupManifest("  name: g\n  groupType: public\n"),
		"members empty":                groupManifest("  name: g\n  members: []\n"),
		"member user name absent":      groupManifest("  name: g\n  members:\n    - {}\n"),
		"member user name empty":       groupManifest("  name: g\n  members:\n    - userName: \"\"\n"),
		"member duplicate":             groupManifest("  name: g\n  members:\n    - userName: a@example.com\n    - userName: a@example.com\n"),
		"custom role":                  groupManifest("  name: g\n  customRole:\n    resourceRef:\n      name: role\n"),
		"custom role namespace":        groupManifest("  name: g\n  customRole:\n    resourceRef:\n      name: role\n      namespace: other\n"),
		"custom role empty":            groupManifest("  name: g\n  customRole: {}\n"),
		"custom role name empty":       groupManifest("  name: g\n  customRole:\n    resourceRef:\n      name: \"\"\n"),
		"custom role name absent":      groupManifest("  name: g\n  customRole:\n    resourceRef: {}\n"),
		"custom roles":                 groupManifest("  name: g\n  customRoles:\n    - resourceRef:\n        name: role\n"),
		"custom roles empty":           groupManifest("  name: g\n  customRoles: []\n"),
		"custom role and custom roles": groupManifest("  name: g\n  customRole:\n    resourceRef:\n      name: a\n  customRoles:\n    - resourceRef:\n        name: b\n"),
		"scope":                        groupManifest("  name: g\n  scope:\n    resourceRef:\n      name: scope\n"),
		"scope namespace":              groupManifest("  name: g\n  scope:\n    resourceRef:\n      name: scope\n      namespace: other\n"),
		"scope empty":                  groupManifest("  name: g\n  scope: {}\n"),
		"scope name empty":             groupManifest("  name: g\n  scope:\n    resourceRef:\n      name: \"\"\n"),
		"unknown spec field":           groupManifest("  name: g\n  extra: true\n"),
	}
}

// The handwritten controller reads the id from the Create answer without a check. An answer
// without the group is a nil dereference, which the manager recovers as a failed reconcile. A
// generated controller returns an error instead, so this test records a known bug of the
// released code, and the migration changes it.
func TestGroupCreateAnswerWithoutGroupPanics(t *testing.T) {
	h := newHarness(t, kinds["group"], "create-without-group")
	groups(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{})
	defer func() {
		if recover() == nil {
			t.Fatal("the handwritten controller no longer panics on a Create answer without a group")
		}
		if len(h.server.takeRequests()) != 1 {
			t.Fatal("the controller sent more than the Create request")
		}
	}()
	h.Create("create succeeds but answers without the group", groupManifest(groupMinimalSpec))
}
