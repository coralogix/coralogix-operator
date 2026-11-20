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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const configurationGroupPath = "/fleet-management/configuration-groups/v1"

func init() {
	register(kind{
		name:     "configurationgroup",
		crdFile:  "coralogix.com_configurationgroups.yaml",
		resource: "configurationgroups",
		newFake:  newConfigurationGroupFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.ConfigurationGroupReconciler{Client: cs.ConfigGroups(), ClientSet: cs, Interval: reconcileInterval}
		},
		scenarios: configurationGroupScenarios,
		admission: configurationGroupAdmission(),
		fidelity: fidelityConfig{
			captureField: "id",
			// The API has no DELETE. An active group cannot be archived, so the cleanup deactivates
			// the family first.
			cleanup: func(id string) []fidelityStep {
				return []fidelityStep{
					{Method: http.MethodPatch, Path: configurationGroupPath + "/" + id + "?update_mask=family.active", Body: map[string]any{"family": map[string]any{"active": false}}},
					{Method: http.MethodPost, Path: configurationGroupPath + "/" + id + "/archive", Body: map[string]any{}},
				}
			},
			normalize: hideRenderedConfigurations,
		},
	})
}

func TestHideRenderedConfigurationsSortsByName(t *testing.T) {
	remote := func(name string) map[string]any {
		return map[string]any{"name": name, "rawConfiguration": "exporters: {}", "hash": "abc"}
	}
	answer := func(names ...string) any {
		remotes := []any{}
		for _, name := range names {
			remotes = append(remotes, remote(name))
		}
		return map[string]any{"family": map[string]any{"preset": map[string]any{"remoteConfigurations": remotes}}}
	}
	got := hideRenderedConfigurations(answer("cluster-collector", "agent"))
	want := hideRenderedConfigurations(answer("agent", "cluster-collector"))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the order of the API changes the answer:\n got %v\nwant %v", got, want)
	}
	first := want.(map[string]any)["family"].(map[string]any)["preset"].(map[string]any)["remoteConfigurations"].([]any)[0].(map[string]any)
	if first["name"] != "agent" || first["rawConfiguration"] != cgRendered || first["hash"] != cgRendered {
		t.Fatalf("first remote configuration = %v", first)
	}
}

// cgRendered stands for the collector configuration that the API renders from a preset template.
const cgRendered = "<rendered from the template>"

// hideRenderedConfigurations writes cgRendered for the text and hash of each remote configuration
// of a preset family, and sorts them by name: the API can answer them in any order. The fake does
// not render templates, and the operator never reads them.
func hideRenderedConfigurations(resp any) any {
	group, _ := resp.(map[string]any)
	family, _ := group["family"].(map[string]any)
	preset, _ := family["preset"].(map[string]any)
	remotes, _ := preset["remoteConfigurations"].([]any)
	for _, item := range remotes {
		if remote, ok := item.(map[string]any); ok {
			remote["rawConfiguration"], remote["hash"] = cgRendered, cgRendered
		}
	}
	sort.SliceStable(remotes, func(i, j int) bool { return remoteName(remotes[i]) < remoteName(remotes[j]) })
	return resp
}

func remoteName(item any) string {
	remote, _ := item.(map[string]any)
	name, _ := remote["name"].(string)
	return name
}

// cgYAML is a small collector configuration. The API stores it without the last newline.
const cgYAML = "receivers:\n  otlp:\n    protocols:\n      grpc: {}\n"

// configurationGroupManifest returns a ConfigurationGroup CR named "group". spec is YAML, indented
// by two spaces.
func configurationGroupManifest(spec string) string {
	return "apiVersion: coralogix.com/v1alpha1\nkind: ConfigurationGroup\nmetadata:\n  name: group\nspec:\n" + spec
}

// cgRawFamily is the YAML of a family with one remote configuration, indented for a spec.
func cgRawFamily(active bool) string {
	return fmt.Sprintf(`  family:
    active: %t
    raw:
      collectorVersion: "0.114.0"
      remoteConfigurations:
        - name: default
          rawConfiguration: %q
          agentSelector:
            attributes:
              service.version: "0.114.0"
`, active, cgYAML)
}

var configurationGroupMinimalManifest = configurationGroupManifest(`  name: Production collectors
  family:
    raw:
      remoteConfigurations:
        - rawConfiguration: "receivers: {}"
`)

// cgRawSpec returns a spec map with one remote configuration, for SetSpec.
func cgRawSpec(name string, family map[string]any, fields map[string]any) map[string]any {
	spec := map[string]any{"name": name, "family": family}
	for k, v := range fields {
		spec[k] = v
	}
	return spec
}

// cgRaw returns a family with one remote configuration and the family fields, for SetSpec.
func cgRaw(familyFields map[string]any) map[string]any {
	family := map[string]any{"raw": map[string]any{"remoteConfigurations": []any{map[string]any{"rawConfiguration": "receivers: {}"}}}}
	for k, v := range familyFields {
		family[k] = v
	}
	return family
}

func cgPresetSpec(name string, presetFields map[string]any) map[string]any {
	preset := map[string]any{
		"chartName":             "CHART_NAME_OTEL_INTEGRATION",
		"chartVersion":          "0.0.289",
		"metadata":              map[string]any{"ClusterName": "production", "KubernetesRunningOn": "openshift"},
		"observabilityFeatures": `{"logs":{"enabled":false}}`,
	}
	for k, v := range presetFields {
		preset[k] = v
	}
	return map[string]any{"name": name, "family": map[string]any{"preset": preset}}
}

func configurationGroups(h *harness) *configurationGroupFake {
	h.t.Helper()
	f, ok := h.API().(*configurationGroupFake)
	if !ok {
		h.t.Fatal("expected *configurationGroupFake")
	}
	return f
}

var configurationGroupScenarios = map[string]func(h *harness){
	"create-sample": func(h *harness) {
		h.CreateFromFile("create the raw sample", "config/samples/v1alpha1/configurationgroup/coralogix_v1alpha1_configurationgroup.yaml")
		h.Reconcile("reconcile again with no change")
	},

	// The fake renders no remote configurations for a preset family; the operator reads none.
	"create-sample-preset": func(h *harness) {
		h.CreateFromFile("create the preset sample", "config/samples/v1alpha1/configurationgroup/coralogix_v1alpha1_configurationgroup_preset.yaml")
		h.Reconcile("reconcile again with no change")
	},

	"create-minimal": func(h *harness) {
		h.Create("create with a name and one remote configuration", configurationGroupMinimalManifest)
		h.Reconcile("reconcile again: the resync sends an Update with the whole mask and no Get")
		h.Reconcile("reconcile again")
	},

	"create-every-field": func(h *harness) {
		h.Create("create with every field", configurationGroupManifest(`  name: Production collectors
  description: Collectors of production
  tags: [production, eu]
  priorityOrder: 100
  family:
    active: true
    description: First family
    raw:
      collectorVersion: "0.114.0"
      metadata:
        team: observability
      remoteConfigurations:
        - name: default
          rawConfiguration: "receivers: {}"
          agentSelector:
            attributes:
              service.version: "0.114.0"
              cx.agent.type: agent
        - rawConfiguration: "exporters: {}"
`))
		h.Reconcile("reconcile again")
	},

	"create-priority-order-negative": func(h *harness) {
		h.Create("create with a negative priority order", configurationGroupManifest("  name: Production collectors\n  priorityOrder: -5\n"+cgRawFamily(false)))
	},

	// The CRD refuses a number over int32 (admission case "priority order over int32"). The
	// handwritten CRD accepted it, and the typed client could not read the CR back.
	"create-priority-order-over-int32": func(h *harness) {
		h.CreateRejected("create with a priority order over int32: the CRD refuses it", configurationGroupManifest("  name: Production collectors\n  priorityOrder: 2147483648\n"+cgRawFamily(false)))
	},

	"create-empty-agent-selector": func(h *harness) {
		h.Create("create with an empty agent selector: the request sends {}", configurationGroupManifest(`  name: Production collectors
  family:
    raw:
      remoteConfigurations:
        - rawConfiguration: "receivers: {}"
          agentSelector: {}
`))
	},

	"create-rejected-duplicate-name": func(h *harness) {
		configurationGroups(h).seed("Production collectors", false, false)
		h.Create("create with a name that another group has: the API rejects it", configurationGroupMinimalManifest)
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	// The API keeps the name of an archived group (case "a name that an archived group has").
	"create-rejected-name-of-archived-group": func(h *harness) {
		configurationGroups(h).seed("Production collectors", false, true)
		h.Create("create with the name of an archived group: the API rejects it", configurationGroupMinimalManifest)
		h.Delete("delete")
	},

	"create-rejected-collector-version-mismatch": func(h *harness) {
		h.Create("create with a selector service.version that is not collectorVersion: the API rejects it", configurationGroupManifest(`  name: Production collectors
  family:
    raw:
      collectorVersion: "0.114.0"
      remoteConfigurations:
        - rawConfiguration: "receivers: {}"
          agentSelector:
            attributes:
              service.version: "0.115.0"
`))
		h.Delete("delete")
	},

	"create-rejected-opamp-extension": func(h *harness) {
		h.Create("create with an OpAMP extension in the collector configuration: the API rejects it", configurationGroupManifest(`  name: Production collectors
  family:
    raw:
      remoteConfigurations:
        - rawConfiguration: "extensions:\n  opamp: {}\nservice:\n  extensions: [opamp]\n"
`))
		h.Delete("delete")
	},

	"create-rejected-preset-metadata": func(h *harness) {
		h.Create("create a preset family without ClusterName: the API rejects it", configurationGroupManifest(`  name: Production collectors
  family:
    preset:
      chartName: CHART_NAME_OTEL_INTEGRATION
      chartVersion: "0.0.289"
      observabilityFeatures: "{}"
`))
		h.Delete("delete")
	},

	"update-description": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		h.SetSpec("set the descriptions and tags", cgRawSpec("Production collectors",
			cgRaw(map[string]any{"description": "First family"}),
			map[string]any{"description": "Collectors of production", "tags": []any{"production"}}))
		h.Reconcile("reconcile again")
	},

	// The mask lists every field, so an omitted field is cleared or reset to its default.
	"update-clear-optional-fields": func(h *harness) {
		h.Create("create with every field", configurationGroupManifest(`  name: Production collectors
  description: Collectors of production
  tags: [production]
  priorityOrder: 100
  family:
    description: First family
    raw:
      collectorVersion: "0.114.0"
      metadata:
        team: observability
      remoteConfigurations:
        - name: default
          rawConfiguration: "receivers: {}"
          agentSelector:
            attributes:
              service.version: "0.114.0"
`))
		h.SetSpec("remove every optional field", cgRawSpec("Production collectors", cgRaw(nil), nil))
		h.Reconcile("reconcile again")
	},

	"update-activate": func(h *harness) {
		h.Create("create an inactive family", configurationGroupManifest("  name: Production collectors\n"+cgRawFamily(false)))
		h.SetSpec("activate the family", cgRawSpec("Production collectors", cgRaw(map[string]any{"active": true}), nil))
		h.SetSpec("remove active: the family is deactivated", cgRawSpec("Production collectors", cgRaw(nil), nil))
	},

	"update-family-content": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		h.SetSpec("change the collector configuration: the API makes a new family version", cgRawSpec("Production collectors",
			map[string]any{"raw": map[string]any{"remoteConfigurations": []any{map[string]any{"rawConfiguration": "exporters: {}"}}}}, nil))
		h.Reconcile("reconcile again: the same content makes no new version")
	},

	"update-raw-to-preset": func(h *harness) {
		h.Create("create a raw family", configurationGroupMinimalManifest)
		h.SetSpec("switch to a preset family: the API rejects it", cgPresetSpec("Production collectors", nil))
		h.Reconcile("reconcile again")
	},

	"update-preset": func(h *harness) {
		h.Create("create a preset family", configurationGroupManifest(`  name: Production collectors
  family:
    preset:
      chartName: CHART_NAME_OTEL_INTEGRATION
      chartVersion: "0.0.289"
      integrationVersion: "0.8.0"
      metadata:
        ClusterName: production
        KubernetesRunningOn: openshift
      observabilityFeatures: '{"logs":{"enabled":false}}'
`))
		h.SetSpec("remove integrationVersion and change the features", cgPresetSpec("Production collectors",
			map[string]any{"observabilityFeatures": `{"logs":{"enabled":true}}`}))
	},

	"update-rename-to-existing-name": func(h *harness) {
		configurationGroups(h).seed("Taken", false, false)
		h.Create("create", configurationGroupMinimalManifest)
		h.SetSpec("change the name to one that another group has: the API rejects it", cgRawSpec("Taken", cgRaw(nil), nil))
		h.Reconcile("reconcile again")
	},

	"delete": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		h.Delete("delete: the family is deactivated, then the group is archived")
	},

	"delete-active": func(h *harness) {
		h.Create("create an active family", configurationGroupManifest("  name: Production collectors\n"+cgRawFamily(true)))
		h.Delete("delete: the family is deactivated, then the group is archived")
	},

	// The API keeps the name of an archived group, so a new CR with the same name fails.
	"recreate-with-same-name": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		h.Delete("delete")
		h.Create("create the CR again with the same name: the API rejects it", configurationGroupMinimalManifest)
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).forget(h.StatusID())
		h.Delete("delete: the group was archived outside the operator")
	},

	"update-when-gone": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).forget(h.StatusID())
		h.SetSpec("change the description: the group was archived outside the operator", cgRawSpec("Production collectors", cgRaw(nil), map[string]any{"description": "changed"}))
		h.Reconcile("reconcile again")
	},

	"resync-when-gone": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).forget(h.StatusID())
		h.Reconcile("resync: the group was archived outside the operator")
		h.Reconcile("reconcile again")
	},

	"delete-after-update-when-gone": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).forget(h.StatusID())
		h.SetSpec("change the description: the group was archived outside the operator", cgRawSpec("Production collectors", cgRaw(nil), map[string]any{"description": "changed"}))
		h.Delete("delete")
		h.Reconcile("reconcile again")
	},

	"delete-without-id": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		clearStatusID(h, h.key)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"delete-without-id-and-remote-gone": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).forget(h.StatusID())
		clearStatusID(h, h.key)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"update-rejected": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).answerNextError(http.MethodPatch, http.StatusBadRequest, "the API rejects this request")
		h.SetSpec("change the description: the API rejects the Update", cgRawSpec("Production collectors", cgRaw(nil), map[string]any{"description": "changed"}))
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-deactivate-rejected": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).answerNextError(http.MethodPatch, http.StatusBadRequest, "the API rejects this request")
		h.Delete("delete: the API rejects the deactivation")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-rejected": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).answerNextError(http.MethodPost, http.StatusBadRequest, "the API rejects this request")
		h.Delete("delete: the API rejects the archive")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// An archive of an active group answers 412 (case "archive an active group"). The deactivation
	// before it answers 200 and changes nothing, as when another client activates the family again.
	"delete-still-active": func(h *harness) {
		h.Create("create an active family", configurationGroupManifest("  name: Production collectors\n"+cgRawFamily(true)))
		configurationGroups(h).ignoreNextUpdate()
		h.Delete("delete: the deactivation is lost, so the API refuses the archive")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-rejected-route-not-found": func(h *harness) {
		configurationGroups(h).answerNextRouteNotFound(http.MethodPost)
		h.Create("create: the API has no such route", configurationGroupMinimalManifest)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-route-not-found": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).answerNextRouteNotFound(http.MethodPost)
		h.Delete("delete: the API has no archive route")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-empty-id": func(h *harness) {
		configurationGroups(h).answerNext(http.MethodPost, map[string]any{"id": ""})
		h.Create("create: the answer has an empty id", configurationGroupMinimalManifest)
		configurationGroups(h).answerNext(http.MethodPost, map[string]any{})
		h.Reconcile("reconcile again: the answer has no id")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// The API stores the group, but the answer has no id, as a lost response would. The SDK cannot
	// decode a group without its id, so the CR gets no status id. The retry creates again, and the
	// API refuses the name of the first group. No CR tracks the first group, so deleting the CR
	// leaves it.
	"create-stored-without-id": func(h *harness) {
		configurationGroups(h).storeNextCreateWithoutID()
		h.Create("create: the API stores the group and answers without its id", configurationGroupMinimalManifest)
		h.Reconcile("reconcile again: the create is sent again, and the API refuses the name")
		h.RecordRemote("the remote groups: the first group stays", configurationGroups(h).list())
		h.Delete("delete the CR")
		h.RecordRemote("the remote groups after the delete: the first group stays", configurationGroups(h).list())
	},

	"create-malformed-response": func(h *harness) {
		configurationGroups(h).answerNext(http.MethodPost, "not a group")
		h.Create("create: the answer cannot be decoded", configurationGroupMinimalManifest)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).answerNext(http.MethodPatch, "not a group")
		h.Reconcile("update: the answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-malformed-response": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		configurationGroups(h).answerNext(http.MethodPost, "not a group")
		h.Delete("delete: the archive answer cannot be decoded")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"bad-status-id-update": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		setStatusID(h, h.key, "not-a-uuid")
		h.Reconcile("update with a status id that is not a UUID: the API answers 404")
		h.Reconcile("reconcile again")
	},

	"bad-status-id-delete": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		setStatusID(h, h.key, "not-a-uuid")
		h.Delete("delete with a status id that is not a UUID: the API answers 404")
		h.Reconcile("reconcile again")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", configurationGroupMinimalManifest)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "fleet"})})
		h.Reconcile("reconcile after the selector stops matching the CR")
	},
}

// configurationGroupAdmission lists the CRs that TestAdmission creates. Each case has one reason
// that the CRD accepts or rejects it.
func configurationGroupAdmission() map[string]string {
	semver := "0.114.0"
	raw := func(rawFields string) string {
		return "  family:\n    raw:\n" + rawFields
	}
	remote := "      remoteConfigurations:\n        - rawConfiguration: \"receivers: {}\"\n"
	spec := func(fields string) string { return configurationGroupManifest("  name: g\n" + fields) }
	preset := func(fields string) string {
		return spec("  family:\n    preset:\n" + fields)
	}
	presetOK := func(extra string) string {
		return "      chartName: CHART_NAME_OTEL_INTEGRATION\n      chartVersion: \"0.0.289\"\n      observabilityFeatures: \"{}\"\n" + extra
	}
	many := func(prefix string, n int, line func(i int) string) string {
		var b strings.Builder
		b.WriteString(prefix)
		for i := range n {
			b.WriteString(line(i))
		}
		return b.String()
	}
	cases := map[string]string{
		"raw family":                       spec(raw(remote)),
		"preset family":                    preset(presetOK("")),
		"preset and raw":                   spec("  family:\n    preset:\n" + presetOK("") + "    raw:\n" + remote),
		"neither preset nor raw":           spec("  family:\n    active: true\n"),
		"no family":                        spec(""),
		"no name":                          configurationGroupManifest(raw(remote)),
		"empty name":                       configurationGroupManifest("  name: \"\"\n" + raw(remote)),
		"name of 256 characters":           configurationGroupManifest("  name: " + strings.Repeat("n", 256) + "\n" + raw(remote)),
		"name of 257 characters":           configurationGroupManifest("  name: " + strings.Repeat("n", 257) + "\n" + raw(remote)),
		"description of 4097 characters":   spec("  description: " + strings.Repeat("d", 4097) + "\n" + raw(remote)),
		"65 tags":                          spec(many("  tags:\n", 65, func(i int) string { return fmt.Sprintf("    - t%d\n", i) }) + raw(remote)),
		"tag of 257 characters":            spec("  tags: [" + strings.Repeat("t", 257) + "]\n" + raw(remote)),
		"empty tag":                        spec("  tags: [\"\"]\n" + raw(remote)),
		"negative priority order":          spec("  priorityOrder: -2147483648\n" + raw(remote)),
		"priority order over int32":        spec("  priorityOrder: 2147483648\n" + raw(remote)),
		"family description of 4097":       spec("  family:\n    description: " + strings.Repeat("d", 4097) + "\n    raw:\n" + remote),
		"no remote configurations":         spec(raw("      remoteConfigurations: []\n")),
		"129 remote configurations":        spec(raw(many("      remoteConfigurations:\n", 129, func(int) string { return "        - rawConfiguration: x\n" }))),
		"empty raw configuration":          spec(raw("      remoteConfigurations:\n        - rawConfiguration: \"\"\n")),
		"remote name of 257 characters":    spec(raw("      remoteConfigurations:\n        - name: " + strings.Repeat("n", 257) + "\n          rawConfiguration: x\n")),
		"empty remote name":                spec(raw("      remoteConfigurations:\n        - name: \"\"\n          rawConfiguration: x\n")),
		"empty agent selector":             spec(raw("      remoteConfigurations:\n        - rawConfiguration: x\n          agentSelector: {}\n")),
		"long agent selector value":        spec(raw("      remoteConfigurations:\n        - rawConfiguration: x\n          agentSelector:\n            attributes:\n              k: " + strings.Repeat("v", 4097) + "\n")),
		"collector version":                spec(raw("      collectorVersion: \"" + semver + "\"\n" + remote)),
		"collector version with v":         spec(raw("      collectorVersion: v0.114.0\n" + remote)),
		"collector version of two parts":   spec(raw("      collectorVersion: \"1.2\"\n" + remote)),
		"collector version with a suffix":  spec(raw("      collectorVersion: 1.2.3-rc.1+build.5\n" + remote)),
		"empty collector version":          spec(raw("      collectorVersion: \"\"\n" + remote)),
		"raw metadata":                     spec(raw("      metadata:\n        team: observability\n" + remote)),
		"raw metadata value of 4097":       spec(raw("      metadata:\n        team: " + strings.Repeat("v", 4097) + "\n" + remote)),
		"raw metadata of 65 entries":       spec(raw(many("      metadata:\n", 65, func(i int) string { return fmt.Sprintf("        k%d: v\n", i) }) + remote)),
		"chart name of each template":      preset(presetOK("")),
		"chart name linux":                 preset(strings.Replace(presetOK(""), "OTEL_INTEGRATION", "OTEL_LINUX_STANDALONE", 1)),
		"chart name windows":               preset(strings.Replace(presetOK(""), "OTEL_INTEGRATION", "OTEL_WINDOWS_STANDALONE", 1)),
		"chart name macos":                 preset(strings.Replace(presetOK(""), "OTEL_INTEGRATION", "OTEL_MACOS_STANDALONE", 1)),
		"chart name ecs":                   preset(strings.Replace(presetOK(""), "OTEL_INTEGRATION", "OTEL_ECS_EC2", 1)),
		"chart name unspecified":           preset(strings.Replace(presetOK(""), "OTEL_INTEGRATION", "UNSPECIFIED", 1)),
		"chart name opentelemetry":         preset(strings.Replace(presetOK(""), "OTEL_INTEGRATION", "OPENTELEMETRY_COLLECTOR", 1)),
		"chart name in camel case":         preset(strings.Replace(presetOK(""), "CHART_NAME_OTEL_INTEGRATION", "otelIntegration", 1)),
		"chart version that is not semver": preset(strings.Replace(presetOK(""), "0.0.289", "latest", 1)),
		"no chart version":                 preset("      chartName: CHART_NAME_OTEL_INTEGRATION\n      observabilityFeatures: \"{}\"\n"),
		"integration version":              preset(presetOK("      integrationVersion: \"0.8.0\"\n")),
		"integration version of two parts": preset(presetOK("      integrationVersion: \"0.8\"\n")),
		"observability features of 1":      preset(strings.Replace(presetOK(""), "\"{}\"", "x", 1)),
		"no observability features":        preset("      chartName: CHART_NAME_OTEL_INTEGRATION\n      chartVersion: \"0.0.289\"\n"),
		"preset metadata of 65 entries":    preset(presetOK(many("      metadata:\n", 65, func(i int) string { return fmt.Sprintf("        k%d: v\n", i) }))),
		"preset metadata value of 4097":    preset(presetOK("      metadata:\n        ClusterName: " + strings.Repeat("v", 4097) + "\n")),
		"unknown field is not an error":    spec("  unknown: ignored\n" + raw(remote)),
	}
	delete(cases, "chart name of each template")
	return cases
}

// configurationGroupFake is the Fleet Manager configuration groups API, written from the SDK's
// openapi.yaml.
//
//	POST  /fleet-management/configuration-groups/v1                     {name, family, ...} -> group
//	GET   /fleet-management/configuration-groups/v1/{id}                -> group
//	PATCH /fleet-management/configuration-groups/v1/{id}?update_mask=…  {fields of the mask} -> group
//	POST  /fleet-management/configuration-groups/v1/{id}/archive        {} -> {}
//
// The status codes, error texts, and stored values follow fidelity/recorded.json, which the real
// API answered. Each rule names the recorded case that shows it. A rule with no case name is not
// recorded: it is the simplest answer that the scenarios need. The API renders the remote
// configurations of a preset family from a template. The fake answers their names and agent
// selectors, and cgRendered for their text, which the fidelity check hides in both answers.
type configurationGroupFake struct {
	mu     sync.Mutex
	groups map[string]*storedGroup
	// next holds one forced answer per method. The next request with that method gets it, and the
	// fake does not run the request.
	next map[string]forcedAnswer
	// ignoreUpdate makes the next update answer the stored group and change nothing.
	ignoreUpdate bool
	// hideNextCreateID makes the next create store the group and answer it without its id, as a
	// lost response would.
	hideNextCreateID bool
}

type storedGroup struct {
	name, description string
	tags              []string
	priorityOrder     int64
	archived          bool
	family            storedFamily
}

type storedFamily struct {
	id, description string
	version         int
	active          bool
	activatedOnce   bool
	raw             *storedRaw
	preset          map[string]any
}

type storedRaw struct {
	collectorVersion string
	metadata         map[string]any
	remotes          []storedRemote
}

type storedRemote struct {
	id, name, rawConfiguration string
	selector                   map[string]any
}

func newConfigurationGroupFake() apiFake {
	return &configurationGroupFake{groups: map[string]*storedGroup{}, next: map[string]forcedAnswer{}}
}

func (f *configurationGroupFake) answerNext(method string, body any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next[method] = forcedAnswer{status: http.StatusOK, body: body}
}

func (f *configurationGroupFake) answerNextError(method string, status int, reason string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := apiError(status, reason)
	f.next[method] = forcedAnswer{status: st, body: body}
}

func (f *configurationGroupFake) answerNextRouteNotFound(method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, body := routeNotFound()
	f.next[method] = forcedAnswer{status: st, body: body}
}

// ignoreNextUpdate makes the next update answer 200 with the stored group, unchanged.
func (f *configurationGroupFake) ignoreNextUpdate() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ignoreUpdate = true
}

// storeNextCreateWithoutID makes the next create store the group and answer it without its id.
func (f *configurationGroupFake) storeNextCreateWithoutID() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hideNextCreateID = true
}

// list returns the id, name, and archived flag of each stored group, sorted by name and id.
func (f *configurationGroupFake) list() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []map[string]any{}
	for id, g := range f.groups {
		out = append(out, map[string]any{"id": id, "name": g.name, "archived": g.archived})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i]["name"] != out[j]["name"] {
			return out[i]["name"].(string) < out[j]["name"].(string)
		}
		return out[i]["id"].(string) < out[j]["id"].(string)
	})
	return out
}

// seed stores a group with one remote configuration, as if another client made it.
func (f *configurationGroupFake) seed(name string, active, archived bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.groups[newUUID()] = &storedGroup{name: name, archived: archived, family: storedFamily{
		id: newUUID(), version: 1, active: active, activatedOnce: active,
		raw: &storedRaw{metadata: map[string]any{}, remotes: []storedRemote{{id: newUUID(), rawConfiguration: "receivers: {}"}}},
	}}
}

// forget archives a group without a request, as if it was archived outside the operator.
func (f *configurationGroupFake) forget(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if g, ok := f.groups[id]; ok {
		g.archived = true
	}
}

func cgAnswer(status int, prefix, text string) (int, any) {
	return status, map[string]any{"code": status, "message": prefix + ": " + text}
}

func cgBadRequest(text string) (int, any) {
	return cgAnswer(http.StatusBadRequest, "Bad Request", text)
}

// cgNotFound is the answer for an unknown, malformed, or archived id (case "ids that are unknown
// or malformed"). A malformed id is not a 400.
func cgNotFound() (int, any) {
	return cgAnswer(http.StatusNotFound, "Not Found", "configuration group not found or archived")
}

func cgNameTaken() (int, any) {
	return cgAnswer(http.StatusConflict, "Already Exists", "configuration group name already exists")
}

func (f *configurationGroupFake) handle(method, path string, query url.Values, body any) (int, any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if forced, ok := f.next[method]; ok {
		delete(f.next, method)
		return forced.status, forced.body
	}
	rest, ok := strings.CutPrefix(path, configurationGroupPath)
	if !ok {
		return routeNotFound()
	}
	in, _ := body.(map[string]any)
	switch {
	case method == http.MethodPost && rest == "":
		return f.create(in)
	case method == http.MethodGet && strings.Count(rest, "/") == 1:
		g := f.live(strings.TrimPrefix(rest, "/"))
		if g == nil {
			return cgNotFound()
		}
		return http.StatusOK, g.answer(strings.TrimPrefix(rest, "/"))
	case method == http.MethodPatch && strings.Count(rest, "/") == 1:
		return f.update(strings.TrimPrefix(rest, "/"), query.Get("update_mask"), in)
	case method == http.MethodPost && strings.HasSuffix(rest, "/archive") && strings.Count(rest, "/") == 2:
		return f.archive(strings.TrimSuffix(strings.TrimPrefix(rest, "/"), "/archive"))
	}
	return routeNotFound()
}

// live returns the group with the id, or nil when it is unknown or archived.
func (f *configurationGroupFake) live(id string) *storedGroup {
	g, ok := f.groups[id]
	if !ok || g.archived {
		return nil
	}
	return g
}

// nameTaken is true when a group other than id has the name. An archived group keeps its name
// (case "a name that an archived group has").
func (f *configurationGroupFake) nameTaken(name, id string) bool {
	for other, g := range f.groups {
		if other != id && g.name == name {
			return true
		}
	}
	return false
}

// create follows the cases "creates that the API rejects or accepts at the limits", "every field,
// then ...", and "a name that another group has". The API accepts a name over the 256 characters
// of the contract.
func (f *configurationGroupFake) create(in map[string]any) (int, any) {
	name, _ := in["name"].(string)
	if name == "" {
		return cgBadRequest("name is required")
	}
	familyIn, ok := in["family"].(map[string]any)
	if !ok {
		return cgBadRequest("family is required")
	}
	family := storedFamily{id: newUUID(), version: 1}
	family.active, _ = familyIn["active"].(bool)
	family.activatedOnce = family.active
	family.description, _ = familyIn["description"].(string)
	switch {
	case familyIn["raw"] != nil:
		raw, status, errBody := parseRaw(familyIn["raw"])
		if status != 0 {
			return status, errBody
		}
		family.raw = raw
	case familyIn["preset"] != nil:
		preset, _ := familyIn["preset"].(map[string]any)
		if status, errBody := checkPreset(preset); status != 0 {
			return status, errBody
		}
		family.preset = preset
	default:
		return cgBadRequest("family.raw or family.preset is required")
	}
	if f.nameTaken(name, "") {
		return cgNameTaken()
	}
	g := &storedGroup{name: name, family: family}
	g.description, _ = in["description"].(string)
	g.tags = stringList(in["tags"])
	g.priorityOrder = int64Value(in["priorityOrder"])
	id := newUUID()
	f.groups[id] = g
	if f.hideNextCreateID {
		f.hideNextCreateID = false
		answer := g.answer(id)
		delete(answer, "id")
		return http.StatusOK, answer
	}
	return http.StatusOK, g.answer(id)
}

// cgMaskPaths are the paths that update_mask accepts. A path inside family.raw or family.preset
// is accepted too.
var cgMaskPaths = map[string]bool{
	"name": true, "description": true, "tags": true, "priorityOrder": true,
	"family.description": true, "family.active": true, "family.raw": true, "family.preset": true,
}

// update follows the cases "update masks and bodies that the API rejects", "every field, then the
// update that clears them", "an update that leaves optional fields out of the body", and "change
// the family content, the description, and active". A field of the mask that the body leaves out
// is cleared or reset to its default. Only a change of the family content makes a new version, and
// a new version has not been activated yet.
func (f *configurationGroupFake) update(id, mask string, in map[string]any) (int, any) {
	g := f.live(id)
	if g == nil {
		return cgNotFound()
	}
	if f.ignoreUpdate {
		f.ignoreUpdate = false
		return http.StatusOK, g.answer(id)
	}
	paths := map[string]bool{}
	for _, p := range strings.Split(mask, ",") {
		switch {
		case p == "*":
			return cgBadRequest(`parsing field "update_mask": field mask wildcard is not supported`)
		case cgMaskPaths[p], strings.HasPrefix(p, "family.raw."), strings.HasPrefix(p, "family.preset."):
			paths[p] = true
		default:
			return cgBadRequest(fmt.Sprintf("unsupported update_mask path %q", p))
		}
	}
	if paths["family.raw"] && paths["family.preset"] {
		return cgBadRequest("family.raw and family.preset cannot both be in update_mask")
	}
	name, _ := in["name"].(string)
	if paths["name"] && name == "" {
		return cgBadRequest("name is required when it is in update_mask")
	}
	familyIn, _ := in["family"].(map[string]any)
	next := g.family
	if paths["family.preset"] && g.family.raw != nil {
		return cgBadRequest("preset content cannot replace a raw configuration family")
	}
	if paths["family.raw"] && g.family.preset != nil {
		return cgBadRequest("raw content cannot replace a configuration family rendered from a Helm chart")
	}
	if paths["family.raw"] {
		raw, status, errBody := parseRaw(familyIn["raw"])
		if status != 0 {
			return status, errBody
		}
		if !reflect.DeepEqual(raw.content(), g.family.raw.content()) {
			next.raw, next.version, next.id, next.activatedOnce = raw, next.version+1, newUUID(), false
		}
	}
	if paths["family.preset"] {
		preset, _ := familyIn["preset"].(map[string]any)
		if status, errBody := checkPreset(preset); status != 0 {
			return status, errBody
		}
		if !reflect.DeepEqual(preset, g.family.preset) {
			next.preset, next.version, next.id, next.activatedOnce = preset, next.version+1, newUUID(), false
		}
	}
	if paths["name"] && f.nameTaken(name, id) {
		return cgNameTaken()
	}
	if paths["name"] {
		g.name = name
	}
	if paths["description"] {
		g.description, _ = in["description"].(string)
	}
	if paths["tags"] {
		g.tags = stringList(in["tags"])
	}
	if paths["priorityOrder"] {
		g.priorityOrder = int64Value(in["priorityOrder"])
	}
	if paths["family.description"] {
		next.description, _ = familyIn["description"].(string)
	}
	if paths["family.active"] {
		next.active, _ = familyIn["active"].(bool)
		next.activatedOnce = next.activatedOnce || next.active
	}
	g.family = next
	return http.StatusOK, g.answer(id)
}

// archive follows the cases "archive an active group" and "create, get, update, ...". An active
// family blocks the archive with 412. A second archive is a 404.
func (f *configurationGroupFake) archive(id string) (int, any) {
	g := f.live(id)
	if g == nil {
		return cgNotFound()
	}
	if g.family.active {
		return cgAnswer(http.StatusPreconditionFailed, "Failed Precondition", "configuration group has active families")
	}
	g.archived = true
	return http.StatusOK, map[string]any{}
}

// parseRaw reads and checks a raw family of a create or update request (case "creates that the
// API rejects ...").
func parseRaw(v any) (*storedRaw, int, any) {
	in, _ := v.(map[string]any)
	raw := &storedRaw{metadata: map[string]any{}}
	raw.collectorVersion, _ = in["collectorVersion"].(string)
	if m, ok := in["metadata"].(map[string]any); ok {
		raw.metadata = m
	}
	items, _ := in["remoteConfigurations"].([]any)
	if len(items) == 0 {
		status, body := cgBadRequest("at least one remote configuration is required")
		return nil, status, body
	}
	for _, item := range items {
		m, _ := item.(map[string]any)
		r := storedRemote{id: newUUID()}
		r.name, _ = m["name"].(string)
		r.rawConfiguration, _ = m["rawConfiguration"].(string)
		if sel, ok := m["agentSelector"].(map[string]any); ok {
			r.selector, _ = sel["attributes"].(map[string]any)
		}
		if version, ok := r.selector["service.version"].(string); ok && raw.collectorVersion != "" && version != raw.collectorVersion {
			status, body := cgBadRequest(fmt.Sprintf("service.version in agent_selector (%q) does not match collector_version (%q)", version, raw.collectorVersion))
			return nil, status, body
		}
		if definesOpAMP(r.rawConfiguration) {
			status, body := cgBadRequest("collector configuration must not define extensions.opamp")
			return nil, status, body
		}
		raw.remotes = append(raw.remotes, r)
	}
	return raw, 0, nil
}

func definesOpAMP(config string) bool {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(config), &doc); err != nil {
		return false
	}
	extensions, _ := doc["extensions"].(map[string]any)
	_, ok := extensions["opamp"]
	return ok
}

// checkPreset follows the case "creates that the API rejects ...": the API supports the OTel
// templates only, and the Kubernetes template needs ClusterName, then KubernetesRunningOn.
func checkPreset(preset map[string]any) (int, any) {
	chart, _ := preset["chartName"].(string)
	switch chart {
	case "CHART_NAME_OTEL_INTEGRATION", "CHART_NAME_OTEL_LINUX_STANDALONE", "CHART_NAME_OTEL_WINDOWS_STANDALONE",
		"CHART_NAME_OTEL_MACOS_STANDALONE", "CHART_NAME_OTEL_ECS_EC2":
	default:
		return cgBadRequest("unsupported chart_name")
	}
	if chart == "CHART_NAME_OTEL_INTEGRATION" {
		metadata, _ := preset["metadata"].(map[string]any)
		for _, key := range []string{"ClusterName", "KubernetesRunningOn"} {
			if _, ok := metadata[key]; !ok {
				return cgBadRequest("missing required metadata: " + key)
			}
		}
	}
	return 0, nil
}

// content is what decides whether an update makes a new family version.
func (r *storedRaw) content() any {
	if r == nil {
		return nil
	}
	remotes := make([]any, 0, len(r.remotes))
	for _, remote := range r.remotes {
		remotes = append(remotes, []any{remote.name, strings.TrimSpace(remote.rawConfiguration), remote.selector})
	}
	return []any{r.collectorVersion, r.metadata, remotes}
}

func stringList(v any) []string {
	items, _ := v.([]any)
	out := []string{}
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// int64Value reads a JSON number. The SDK sends priorityOrder as a number.
func int64Value(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

// cgTime is the time of every stored event. The fidelity check writes <time> for it.
const cgTime = "2026-01-01T00:00:00Z"

// answer is the group as the API answers it. An empty description is left out, tags and metadata
// are always there, a remote configuration name is "" when it was not set, and an empty agent
// selector is left out.
func (g *storedGroup) answer(id string) map[string]any {
	tags := make([]any, 0, len(g.tags))
	for _, tag := range g.tags {
		tags = append(tags, tag)
	}
	out := map[string]any{
		"id": id, "name": g.name, "tags": tags, "priorityOrder": g.priorityOrder,
		"archived": false, "family": g.family.answer(),
	}
	if g.description != "" {
		out["description"] = g.description
	}
	return out
}

func (f storedFamily) answer() map[string]any {
	out := map[string]any{"id": f.id, "active": f.active, "version": strconv.Itoa(f.version), "sources": []any{}}
	if f.description != "" {
		out["description"] = f.description
	}
	if f.activatedOnce {
		out["lastActivatedAt"] = cgTime
	}
	if f.version > 1 {
		out["basedOnConfigurationFamilyId"] = newUUID()
	}
	if f.preset != nil {
		out["preset"] = presetAnswer(f.preset)
	}
	if f.raw != nil {
		remotes := make([]any, 0, len(f.raw.remotes))
		for _, r := range f.raw.remotes {
			text := strings.TrimSpace(r.rawConfiguration)
			sum := sha256.Sum256([]byte(text))
			remote := map[string]any{
				"id": r.id, "name": r.name, "rawConfiguration": text, "hash": hex.EncodeToString(sum[:]),
				"createdAt": cgTime, "updatedAt": cgTime,
			}
			if len(r.selector) > 0 {
				remote["agentSelector"] = map[string]any{"attributes": r.selector}
			}
			remotes = append(remotes, remote)
		}
		raw := map[string]any{"metadata": f.raw.metadata, "remoteConfigurations": remotes}
		if f.raw.collectorVersion != "" {
			raw["collectorVersion"] = f.raw.collectorVersion
		}
		out["raw"] = raw
	}
	return out
}

// cgTemplate is what the API adds to a preset family of a chart (cases "a preset family: ...",
// "a preset family of a standalone template", "a preset family of the ECS template"): the default
// integrationVersion, the collectorVersion, and the remote configurations that it renders. Windows
// and macOS are not recorded: they follow the Linux template.
type cgTemplate struct {
	integrationVersion, collectorVersion string
	// remotes are the cx.agent.type of each rendered remote configuration, which is also its name.
	remotes []string
	// clusterName is true when each agent selector has cx.cluster.name, from metadata.ClusterName.
	clusterName bool
}

var cgTemplates = map[string]cgTemplate{
	"CHART_NAME_OTEL_INTEGRATION":        {"0.8.0", "0.147.0", []string{"agent", "cluster-collector"}, true},
	"CHART_NAME_OTEL_LINUX_STANDALONE":   {"0.1.0", "0.147.0", []string{"agent"}, false},
	"CHART_NAME_OTEL_WINDOWS_STANDALONE": {"0.1.0", "0.147.0", []string{"agent"}, false},
	"CHART_NAME_OTEL_MACOS_STANDALONE":   {"0.1.0", "0.147.0", []string{"agent"}, false},
	"CHART_NAME_OTEL_ECS_EC2":            {"0.5.0", "0.131.1", []string{"agent"}, true},
}

// presetAnswer is a preset family as the API answers it. An integrationVersion that the request
// leaves out is the default of the template, also on an update (case "a preset family: ...").
func presetAnswer(preset map[string]any) map[string]any {
	chart, _ := preset["chartName"].(string)
	tpl := cgTemplates[chart]
	metadata, _ := preset["metadata"].(map[string]any)
	remotes := make([]any, 0, len(tpl.remotes))
	for _, agentType := range tpl.remotes {
		attributes := map[string]any{"cx.agent.type": agentType, "service.version": tpl.collectorVersion}
		if tpl.clusterName {
			attributes["cx.cluster.name"] = metadata["ClusterName"]
		}
		remotes = append(remotes, map[string]any{
			"id": newUUID(), "name": agentType, "rawConfiguration": cgRendered, "hash": cgRendered,
			"agentSelector": map[string]any{"attributes": attributes}, "createdAt": cgTime, "updatedAt": cgTime,
		})
	}
	out := map[string]any{
		"integrationVersion": tpl.integrationVersion, "collectorVersion": tpl.collectorVersion,
		"metadata": map[string]any{}, "remoteConfigurations": remotes,
	}
	for k, v := range preset {
		out[k] = v
	}
	return out
}
