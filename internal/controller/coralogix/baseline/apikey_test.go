// Copyright 2024 Coralogix Ltd.
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
	"net/http"
	"sort"
	"strconv"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "apikey",
		crdFile:  "coralogix.com_apikeys.yaml",
		resource: "apikeys",
		newFake:  newAPIKeyFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.ApiKeyReconciler{ApiKeysClient: cs.APIKeys(), Interval: reconcileInterval}
		},
		scenarios: apiKeyScenarios,
		admission: apiKeyAdmission(),
		fidelity: fidelityConfig{
			captureField: "keyId",
			deletePath:   func(id string) string { return apiKeyPath + "/" + id },
			normalize:    normalizeAPIKeyAnswer,
			recordVars:   apiKeyRecordVars,
			fakeVars:     map[string]string{"teamId": strconv.Itoa(apiKeyFakeTeam)},
		},
	})
}

// apiKeyRecordVars reads the id of the team that owns the recording key: a key that the team
// owns can be created only for the team of the caller. It takes the team of the first key that a
// team owns.
func apiKeyRecordVars(t *testing.T, base, apiKey string) map[string]string {
	t.Helper()
	status, resp, err := sendRecordRequest(base, apiKey, http.MethodGet, apiKeyPath+"/list/all", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list api keys: %d %v", status, err)
	}
	for _, key := range listOf(resp, "keys") {
		owner, _ := key["owner"].(map[string]any)
		if team, ok := owner["teamId"].(float64); ok && team > 0 {
			return map[string]string{"teamId": strconv.FormatInt(int64(team), 10)}
		}
	}
	t.Fatal("the team has no key that a team owns")
	return nil
}

// normalizeAPIKeyAnswer hides the secret value of a key, and sorts the presets and permissions.
// The API answers both lists in no stable order.
func normalizeAPIKeyAnswer(resp any) any {
	root, _ := resp.(map[string]any)
	for _, key := range []map[string]any{root, asMap(root["keyInfo"])} {
		if key == nil {
			continue
		}
		if value, ok := key["value"].(string); ok && value != "" {
			key["value"] = "<secret>"
		}
		permissions := asMap(key["keyPermissions"])
		if permissions == nil {
			continue
		}
		sortStrings(permissions["permissions"])
		if presets, ok := permissions["presets"].([]any); ok {
			for _, preset := range presets {
				sortStrings(asMap(preset)["permissions"])
			}
			sort.SliceStable(presets, func(i, j int) bool {
				a, _ := asMap(presets[i])["name"].(string)
				b, _ := asMap(presets[j])["name"].(string)
				return a < b
			})
		}
	}
	return resp
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func sortStrings(v any) {
	list, ok := v.([]any)
	if !ok {
		return
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, _ := list[i].(string)
		b, _ := list[j].(string)
		return a < b
	})
}

const apiKeySpec = `  name: baseline-key
  owner:
    teamId: 4013254
  presets:
    - APM
  permissions:
    - livetail:Read
`

func apiKeyManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: ApiKey\nmetadata:\n  name: apikey\nspec:\n" + spec
}

func apiKeys(h *harness) *apiKeyFake {
	h.t.Helper()
	f, ok := h.API().(*apiKeyFake)
	if !ok {
		h.t.Fatal("expected *apiKeyFake")
	}
	return f
}

// apiKeySpecMap is a whole spec of a key that the team of the fake owns.
func apiKeySpecMap(name string, active bool, presets, permissions []any, accessPolicy *string) map[string]any {
	spec := map[string]any{"name": name, "active": active, "owner": map[string]any{"teamId": int64(apiKeyFakeTeam)}}
	if presets != nil {
		spec["presets"] = presets
	}
	if permissions != nil {
		spec["permissions"] = permissions
	}
	if accessPolicy != nil {
		spec["accessPolicy"] = *accessPolicy
	}
	return spec
}

// apiKeySecretName is the Secret that the controller writes the key value to. Before the CR
// exists, it is the Secret of the CR that apiKeyManifest names.
func apiKeySecretName(h *harness) string {
	if h.key.Name == "" {
		return "apikey-secret"
	}
	return h.key.Name + "-secret"
}

// apiKeySecret returns the Secret of the CR as the golden shows it, or nil when it does not exist.
// The values are those of the fake, never a real key.
func apiKeySecret(h *harness) any {
	h.t.Helper()
	secret := &corev1.Secret{}
	err := h.client.Get(context.Background(), client.ObjectKey{Namespace: h.namespace, Name: apiKeySecretName(h)}, secret)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		h.t.Fatalf("get the secret: %v", err)
	}
	owners := []any{}
	for _, o := range secret.OwnerReferences {
		owners = append(owners, map[string]any{"apiVersion": o.APIVersion, "kind": o.Kind, "name": o.Name, "controller": o.Controller != nil && *o.Controller})
	}
	data := map[string]string{}
	for k, v := range secret.Data {
		data[k] = string(v)
	}
	return map[string]any{"ownerReferences": owners, "data": data}
}

// recordAPIKeyRemote adds a step that records the keys of the fake and the Secret of the CR.
func recordAPIKeyRemote(h *harness, name string) {
	h.t.Helper()
	h.RecordRemote(name, map[string]any{"keys": apiKeys(h).snapshot(), "secret": apiKeySecret(h)})
}

// putAPIKeySecret writes the Secret of the CR with a value, as another client or an earlier
// operator did.
func putAPIKeySecret(h *harness, value string) {
	h.t.Helper()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: apiKeySecretName(h), Namespace: h.namespace},
		Data:       map[string][]byte{"key-value": []byte(value)},
	}
	existing := &corev1.Secret{}
	err := h.client.Get(context.Background(), client.ObjectKeyFromObject(secret), existing)
	switch {
	case apierrors.IsNotFound(err):
		err = h.client.Create(context.Background(), secret)
	case err == nil:
		existing.Data = secret.Data
		err = h.client.Update(context.Background(), existing)
	}
	if err != nil {
		h.t.Fatalf("write the secret: %v", err)
	}
}

func deleteAPIKeySecret(h *harness) {
	h.t.Helper()
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: apiKeySecretName(h), Namespace: h.namespace}}
	if err := h.client.Delete(context.Background(), secret); err != nil {
		h.t.Fatalf("delete the secret: %v", err)
	}
}

// releasedAPIKey is the key of upgrade-from-released, as the released operator created it.
func releasedAPIKey() apiKeyRecord {
	return apiKeyRecord{
		name: "baseline-key", teamID: apiKeyFakeTeam, active: true,
		presets: []string{"APM"}, permissions: []string{"livetail:Read"}, value: "cxo-fake-released-value",
	}
}

const apiKeyReleasedID = "22222222-2222-4222-8222-222222222222"

var apiKeyScenarios = map[string]func(h *harness){
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the released team sample", "config/samples/v1alpha1/apikeys/team_key.yaml")
		recordAPIKeyRemote(h, "the key and its secret")
		h.Reconcile("resync: update sends every field, then get reads the value")
		recordAPIKeyRemote(h, "the remote policy is normalized, the secret is unchanged")
	},
	"create-user-sample": func(h *harness) {
		h.CreateFromFile("create the released user sample", "config/samples/v1alpha1/apikeys/user_key.yaml")
		h.Reconcile("resync")
		recordAPIKeyRemote(h, "a key that the user owns")
	},
	// The released sample names a team that is not the team of the caller. The API answers 403.
	"create-other-team": func(h *harness) {
		h.CreateFromFile("create the released send-data sample: another team", "config/samples/v1alpha1/apikeys/send_your_data_key.yaml")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-minimal": func(h *harness) {
		h.Create("create with presets only: active defaults to true", apiKeyManifest("  name: baseline-key\n  owner:\n    teamId: 4013254\n  presets:\n    - SendData\n"))
		h.Reconcile("resync")
		h.Reconcile("resync again")
	},
	// Create cannot set active: the key is active until the first resync sends isActive false.
	"create-inactive": func(h *harness) {
		h.Create("create with active false", apiKeyManifest(apiKeySpec+"  active: false\n"))
		recordAPIKeyRemote(h, "the new key is active")
		h.Reconcile("resync sends isActive false")
		recordAPIKeyRemote(h, "the key is inactive")
	},
	"create-empty-name": func(h *harness) {
		h.Create("admission accepts an empty name, the API rejects it", apiKeyManifest("  name: \"\"\n  owner:\n    teamId: 4013254\n  presets:\n    - APM\n"))
		h.Reconcile("retry")
		h.Delete("delete")
	},
	"create-lower-case-preset": func(h *harness) {
		h.Create("a preset in lower case", apiKeyManifest("  name: baseline-key\n  owner:\n    teamId: 4013254\n  presets:\n    - apm\n"))
		h.Reconcile("resync sends the preset as the CR has it")
		recordAPIKeyRemote(h, "the remote preset is APM")
	},
	"create-without-spec": func(h *harness) {
		h.Create("create without spec", "apiVersion: coralogix.com/v1alpha1\nkind: ApiKey\nmetadata:\n  name: apikey\n")
		h.Reconcile("retry")
		h.Delete("delete")
	},
	// Admission accepts a team id that is negative or above uint32, but the Go type is uint32. The
	// controller cannot read the CR, and makes no request. In a cluster the typed list of every
	// ApiKey fails too.
	"create-negative-team-id": func(h *harness) {
		h.Create("admission accepts -1, the controller cannot read the CR", apiKeyManifest("  name: baseline-key\n  owner:\n    teamId: -1\n  presets:\n    - APM\n"))
		h.Reconcile("retry")
	},
	"create-rejected": func(h *harness) {
		apiKeys(h).answerNext(http.MethodPost, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Create("create returns an API error", apiKeyManifest(apiKeySpec))
		h.Reconcile("retry creates the key")
	},
	// A 200 with an empty key id: the controller writes the secret and saves the empty id, which
	// reads as no id. The retry creates a second key, and its secret write fails because the secret
	// exists. The error saves the new id, so the next reconcile updates that key and its secret.
	"create-empty-id": func(h *harness) {
		apiKeys(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{"keyId": "", "name": "baseline-key", "value": "cxo-fake-orphan-value"})
		h.Create("create succeeds without an id", apiKeyManifest(apiKeySpec))
		h.Reconcile("retry creates a key, the secret exists")
		h.Reconcile("update writes the value of the new key")
		recordAPIKeyRemote(h, "the key and the secret")
	},
	// A Secret with the name exists: the key is created and the secret write fails. The error
	// saves status.id, but no finalizer is added. The next reconcile updates the key and
	// overwrites the secret, which keeps no owner reference. A delete then removes the CR without
	// deleting the key.
	"create-secret-exists": func(h *harness) {
		putAPIKeySecret(h, "not-from-the-operator")
		h.Create("create: the key is made, the secret exists", apiKeyManifest(apiKeySpec))
		h.Reconcile("update writes the value of the key")
		recordAPIKeyRemote(h, "one key, the secret holds its value")
		h.Delete("delete: no finalizer, so no remote delete")
		recordAPIKeyRemote(h, "the key remains")
	},
	// A chart upgrade: the released operator created the key and wrote status.id, its finalizer,
	// and the secret. The operator updates that key, creates none, and deletes it on delete.
	"upgrade-from-released": func(h *harness) {
		apiKeys(h).put(apiKeyReleasedID, releasedAPIKey())
		putAPIKeySecret(h, "cxo-fake-released-value")
		h.CreateReleased("the released operator made the key: reconcile the CR", apiKeyManifest(apiKeySpec),
			[]string{"api-key.coralogix.com/finalizer"}, map[string]any{"id": apiKeyReleasedID, "printableStatus": "RemoteSynced"})
		if got := h.StatusID(); got != apiKeyReleasedID {
			h.t.Fatalf("status.id changed from %s to %s", apiKeyReleasedID, got)
		}
		h.Reconcile("reconcile again")
		h.Delete("delete: the key is deleted by its released id")
		recordAPIKeyRemote(h, "no key remains")
	},
	// The released operator made the key, but its secret is gone: the update writes it again.
	"upgrade-without-secret": func(h *harness) {
		apiKeys(h).put(apiKeyReleasedID, releasedAPIKey())
		h.CreateReleased("reconcile a released CR without its secret", apiKeyManifest(apiKeySpec),
			[]string{"api-key.coralogix.com/finalizer"}, map[string]any{"id": apiKeyReleasedID})
		recordAPIKeyRemote(h, "the secret holds the value that get returns")
	},
	// A key that is hashed has no value in get. The update writes an empty value over the
	// secret, which held the value of the create answer.
	"hashed-key": func(h *harness) {
		key := releasedAPIKey()
		key.hashed = true
		apiKeys(h).put(apiKeyReleasedID, key)
		putAPIKeySecret(h, "cxo-fake-released-value")
		h.CreateReleased("reconcile a CR of a hashed key", apiKeyManifest(apiKeySpec),
			[]string{"api-key.coralogix.com/finalizer"}, map[string]any{"id": apiKeyReleasedID})
		recordAPIKeyRemote(h, "the secret value is empty")
	},
	"update-fields": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		h.SetSpec("rename, deactivate, replace presets and permissions", apiKeySpecMap("renamed-key", false,
			[]any{"SendData"}, []any{"serverless:Read", "alerts:MapRead"}, nil))
		recordAPIKeyRemote(h, "the remote key")
	},
	"update-clear-presets": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		h.SetSpec("remove the presets: update sends an empty presets object", apiKeySpecMap("baseline-key", true, nil, []any{"livetail:Read"}, nil))
		recordAPIKeyRemote(h, "the remote key has no preset")
	},
	"update-set-access-policy": func(h *harness) {
		h.Create("create without a policy", apiKeyManifest(apiKeySpec))
		policy := `{"version":"2025-01-01","default":{"permissions":{"team-custom-api-keys:ReadConfig":"grant"}},"rules":[]}`
		h.SetSpec("set an incomplete policy", apiKeySpecMap("baseline-key", true, []any{"APM"}, []any{"livetail:Read"}, &policy))
		recordAPIKeyRemote(h, "the API adds the missing permissions as deny")
		h.Reconcile("resync sends the policy of the CR again")
	},
	"update-remove-access-policy": func(h *harness) {
		policy := `{"version":"2025-01-01","default":{"permissions":{"team-custom-api-keys:ReadConfig":"grant"}},"rules":[]}`
		h.Create("create with a policy", apiKeyManifest(apiKeySpec+"  accessPolicy: '"+policy+"'\n"))
		h.SetSpec("remove the policy: update omits it", apiKeySpecMap("baseline-key", true, []any{"APM"}, []any{"livetail:Read"}, nil))
		recordAPIKeyRemote(h, "the remote policy stays")
		empty := ""
		h.SetSpec("an empty policy clears it", apiKeySpecMap("baseline-key", true, []any{"APM"}, []any{"livetail:Read"}, &empty))
		recordAPIKeyRemote(h, "the remote key has no policy")
	},
	"update-rejected": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		h.SetSpec("an unknown permission", apiKeySpecMap("baseline-key", true, []any{"APM"}, []any{"no-such:Permission"}, nil))
		h.Reconcile("retry")
		recordAPIKeyRemote(h, "the remote key is unchanged")
	},
	// The update 404 removes status.id. The retry creates a new key, and its secret write fails:
	// the secret of the old key exists. The error saves the new id, so the next reconcile updates
	// the new key and writes its value.
	"update-when-gone": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		apiKeys(h).forget(h.StatusID())
		h.SetSpec("update after remote deletion", apiKeySpecMap("renamed-key", true, []any{"APM"}, nil, nil))
		h.Reconcile("recreate: the key is made, the secret exists")
		h.Reconcile("update writes the value of the new key")
		recordAPIKeyRemote(h, "the new key and its value")
	},
	"update-get-fails": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		apiKeys(h).answerNext(http.MethodGet, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Reconcile("update succeeds, get fails")
		h.Reconcile("retry")
	},
	"secret-deleted": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		deleteAPIKeySecret(h)
		h.Reconcile("resync writes the secret again")
		recordAPIKeyRemote(h, "the secret holds the value of the key")
	},
	"secret-changed": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		putAPIKeySecret(h, "changed-by-hand")
		h.Reconcile("resync writes the value of the key")
		recordAPIKeyRemote(h, "the secret holds the value of the key")
	},
	// An id that is not a UUID is a 400 on update, not a 404: the status id stays, and each
	// reconcile fails.
	"bad-status-id": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		setScopeStatusID(h, "not-a-uuid")
		h.Reconcile("update of an id that is not a UUID is a 400")
		h.Reconcile("retry")
		h.Delete("delete of the id is a 400 too")
	},
	// The update 404 removes status.id. A delete before the retry finds a CR with a finalizer
	// and no status id: the shared reconciler removes the finalizer with no API call.
	"delete-after-update-gone": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		apiKeys(h).forget(h.StatusID())
		h.SetSpec("update after remote deletion: the 404 removes status.id", apiKeySpecMap("renamed-key", true, []any{"APM"}, nil, nil))
		h.Delete("delete a CR that has a finalizer and no status id: no key is created")
		recordAPIKeyRemote(h, "no remote key exists")
	},
	// envtest runs no garbage collector, so the secret stays after the CR is gone. In a cluster
	// the owner reference deletes it. The controller does not delete it.
	"delete": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		h.Delete("delete")
		h.Reconcile("delete again")
		recordAPIKeyRemote(h, "no key remains")
	},
	"delete-when-gone": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		apiKeys(h).forget(h.StatusID())
		h.Delete("delete after remote deletion: the API answers 404")
	},
	"delete-rejected": func(h *harness) {
		h.Create("create", apiKeyManifest(apiKeySpec))
		apiKeys(h).answerNext(http.MethodDelete, http.StatusInternalServerError, map[string]any{"code": 500, "message": "Internal Server Error: rejected"})
		h.Delete("delete returns an API error")
		h.Reconcile("retry deletes the key")
	},
	"selector-mismatch": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "other"})})
		h.Create("create: a direct reconcile does not run the event predicate", apiKeyManifest(apiKeySpec))
		h.Reconcile("the selector does not match: delete the remote key and clear the status")
		recordAPIKeyRemote(h, "the secret stays")
		h.Delete("delete")
	},
}

func apiKeyAdmission() map[string]string {
	const owner = "  owner:\n    teamId: 4013254\n"
	const presets = "  presets:\n    - APM\n"
	return map[string]string{
		"sample":                    apiKeyManifest(apiKeySpec),
		"without spec":              "apiVersion: coralogix.com/v1alpha1\nkind: ApiKey\nmetadata:\n  name: apikey\n",
		"empty spec":                apiKeyManifest("  {}\n"),
		"name absent":               apiKeyManifest(owner + presets),
		"name empty":                apiKeyManifest("  name: \"\"\n" + owner + presets),
		"active false":              apiKeyManifest("  name: k\n  active: false\n" + owner + presets),
		"owner absent":              apiKeyManifest("  name: k\n" + presets),
		"owner empty":               apiKeyManifest("  name: k\n  owner: {}\n" + presets),
		"owner team and user":       apiKeyManifest("  name: k\n  owner:\n    teamId: 1\n    userId: u\n" + presets),
		"owner user":                apiKeyManifest("  name: k\n  owner:\n    userId: u\n" + presets),
		"owner user empty":          apiKeyManifest("  name: k\n  owner:\n    userId: \"\"\n" + presets),
		"owner organisation":        apiKeyManifest("  name: k\n  owner:\n    organisationId: o\n" + presets),
		"team id zero":              apiKeyManifest("  name: k\n  owner:\n    teamId: 0\n" + presets),
		"team id negative":          apiKeyManifest("  name: k\n  owner:\n    teamId: -1\n" + presets),
		"team id above uint32":      apiKeyManifest("  name: k\n  owner:\n    teamId: 4294967296\n" + presets),
		"team id as text":           apiKeyManifest("  name: k\n  owner:\n    teamId: \"4013254\"\n" + presets),
		"presets and permissions":   apiKeyManifest("  name: k\n" + owner + presets + "  permissions:\n    - livetail:Read\n"),
		"permissions only":          apiKeyManifest("  name: k\n" + owner + "  permissions:\n    - livetail:Read\n"),
		"no presets or permissions": apiKeyManifest("  name: k\n" + owner),
		"presets empty":             apiKeyManifest("  name: k\n" + owner + "  presets: []\n"),
		"preset empty string":       apiKeyManifest("  name: k\n" + owner + "  presets:\n    - \"\"\n"),
		"access policy empty":       apiKeyManifest("  name: k\n" + owner + presets + "  accessPolicy: \"\"\n"),
		"access policy not json":    apiKeyManifest("  name: k\n" + owner + presets + "  accessPolicy: not json\n"),
		"hashed":                    apiKeyManifest("  name: k\n" + owner + presets + "  hashed: true\n"),
	}
}
