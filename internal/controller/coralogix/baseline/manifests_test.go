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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
)

// rbacFiles are the cluster roles that list the operator's permissions.
var rbacFiles = []string{
	"config/rbac/role.yaml",
	"charts/coralogix-operator/templates/cluster_role.yaml",
}

// TestCRD records the CRD with every description removed. It covers validation, required
// fields, enums, defaults, print columns, and the status shape. Description text may change.
func TestCRD(t *testing.T) {
	for _, k := range sortedKinds() {
		t.Run(k.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(repoRoot, "config", "crd", "bases", k.crdFile))
			if err != nil {
				t.Fatal(err)
			}
			var crd map[string]any
			if err := yaml.Unmarshal(raw, &crd); err != nil {
				t.Fatal(err)
			}
			// The controller-gen version is not behavior.
			if meta, ok := crd["metadata"].(map[string]any); ok {
				if annotations, ok := meta["annotations"].(map[string]any); ok {
					delete(annotations, "controller-gen.kubebuilder.io/version")
					if len(annotations) == 0 {
						delete(meta, "annotations")
					}
				}
			}
			out, err := yaml.Marshal(stripDescriptions(crd, false))
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, k.goldenDir(), "crd.yaml", out)
		})
	}
}

// stripDescriptions removes every "description" key of a schema. Under "properties" the keys are
// field names, so a field named "description" stays.
func stripDescriptions(v any, fieldNames bool) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, e := range x {
			if k == "description" && !fieldNames {
				continue
			}
			out[k] = stripDescriptions(e, !fieldNames && k == "properties")
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = stripDescriptions(e, false)
		}
		return out
	default:
		return v
	}
}

type rbacRule struct {
	APIGroups []string `json:"apiGroups"`
	Resource  string   `json:"resource"`
	Verbs     []string `json:"verbs"`
}

// TestRBAC records the cluster role rules for the kind's resource and its subresources.
func TestRBAC(t *testing.T) {
	for _, k := range sortedKinds() {
		t.Run(k.name, func(t *testing.T) {
			got := map[string][]rbacRule{}
			for _, file := range rbacFiles {
				got[file] = rulesFor(t, file, k.resource)
			}
			checkGolden(t, k.goldenDir(), "rbac.json", marshalGolden(t, got))
		})
	}
}

func rulesFor(t *testing.T, file, resource string) []rbacRule {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, file))
	if err != nil {
		t.Fatal(err)
	}
	// The Helm template has template actions only in metadata, so those lines are dropped.
	var kept []string
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.Contains(line, "{{") {
			kept = append(kept, line)
		}
	}
	var role rbacv1.ClusterRole
	if err := yaml.Unmarshal([]byte(strings.Join(kept, "\n")), &role); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	out := []rbacRule{}
	for _, rule := range role.Rules {
		for _, r := range rule.Resources {
			if r == resource || strings.HasPrefix(r, resource+"/") {
				verbs := append([]string(nil), rule.Verbs...)
				sort.Strings(verbs)
				out = append(out, rbacRule{APIGroups: rule.APIGroups, Resource: r, Verbs: verbs})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Resource < out[j].Resource })
	return out
}

// TestFinalizer records the finalizer name. A changed name leaves old CRs with a finalizer
// that no controller removes.
func TestFinalizer(t *testing.T) {
	cs := openapicxsdk.NewClientSet(openapicxsdk.NewConfigBuilder().WithURL("http://127.0.0.1" + apiBasePath).Build())
	for _, k := range sortedKinds() {
		t.Run(k.name, func(t *testing.T) {
			r, ok := k.newReconciler(cs).(interface{ FinalizerName() string })
			if !ok {
				t.Fatalf("the %s reconciler has no FinalizerName method", k.name)
			}
			checkGolden(t, k.goldenDir(), "finalizer.txt", []byte(r.FinalizerName()+"\n"))
		})
	}
}
