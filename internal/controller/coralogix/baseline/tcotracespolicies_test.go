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
	"maps"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "tcotracespolicies",
		crdFile:  "coralogix.com_tcotracespolicies.yaml",
		resource: "tcotracespolicies",
		newFake:  newTCOFake(tcoSpans),
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.TCOTracesPoliciesReconciler{
				TCOPoliciesClient:       cs.TCOPolicies(),
				ArchiveRetentionsClient: cs.ArchiveRetentions(),
				Interval:                reconcileInterval,
			}
		},
		scenarios: tcoTracesScenarios(),
		admission: tcoTracesAdmission(),
		fidelity:  fidelityConfig{overwritesTeam: true},
	})
}

const tcoTracesKind = "TCOTracesPolicies"

const tcoTracesMinimal = `  policies:
    - name: traces-minimal
      priority: high
`

const tcoTracesUpdated = `  policies:
    - name: traces-minimal
      priority: low
      services:
        names: [checkout]
        ruleType: is
    - name: traces-second
      priority: medium
      actions:
        names: [GET]
        ruleType: start_with
`

const tcoTracesRejected = `  policies:
    - name: traces-no-version
      priority: low
      dpxlExpression: "$d.status == 'ERROR'"
`

// tcoTracesFull sets every field of a traces policy. dpxlExpression excludes every structured
// rule, so two policies cover every field.
const tcoTracesFull = `  policies:
    - name: traces-structured
      description: Structured rules
      priority: high
      disabled: true
      applications:
        names: [prod, staging]
        ruleType: is
      subsystems:
        names: [mobile]
        ruleType: is_not
      services:
        names: [service, system]
        ruleType: includes
      actions:
        names: [action1, action2]
        ruleType: start_with
      tags:
        - name: tags.http
          ruleType: is
          values: [GET, POST]
        - name: tags.env
          ruleType: is_not
          values: [dev]
      archiveRetention:
        backendRef:
          id: ` + longTermRetentionID + `
      priorityOverride:
        quotaBased:
          usageTiers:
            - dailyQuotaPercentage: 30
              priority: medium
            - dailyQuotaPercentage: "60.5"
              priority: low
    - name: traces-dpxl
      description: DPXL rules
      priority: medium
      disabled: false
      dpxlExpression: "<v1> $d.status == 'ERROR'"
`

func tcoTracesScenarios() map[string]func(h *harness) {
	s := tcoCommonScenarios(tcoTracesKind, tcoSpans, tcoTracesMinimal, tcoTracesUpdated, tcoTracesRejected)
	maps.Copy(s, map[string]func(h *harness){
		"create-sample": func(h *harness) {
			h.CreateFromFile("create the sample", "config/samples/v1alpha1/tcotracespolicies/example-tco-traces-policies.yaml")
		},

		"create-full": func(h *harness) {
			h.Create("create with every field", tcoManifest(tcoTracesKind, tcoTracesFull))
		},

		"clear-optional": func(h *harness) {
			h.Create("create with every field", tcoManifest(tcoTracesKind, tcoTracesFull))
			h.SetSpec("remove every optional field: description is sent as an empty string", tcoSpec(h, `policies:
  - name: traces-structured
    priority: high
  - name: traces-dpxl
    priority: medium
`))
		},

		"create-structured-rules": func(h *harness) {
			h.Create("create with a service rule only", tcoManifest(tcoTracesKind, `  policies:
    - name: traces-structured
      priority: high
      services:
        names: [service]
        ruleType: is
`))
		},

		"create-dpxl-and-override": func(h *harness) {
			h.Create("create with a DPXL expression and a policy override", tcoManifest(tcoTracesKind, `  policies:
    - name: traces-advanced
      priority: low
      dpxlExpression: "<v1> $d.status == 'ERROR'"
      priorityOverride:
        quotaBased:
          usageTiers:
            - dailyQuotaPercentage: 60
              priority: medium
`))
		},

		"tag-values-joined": func(h *harness) {
			h.Create("create with 50 tag values: they are sent as one comma-separated value", tcoManifest(tcoTracesKind, `  policies:
    - name: tag-limit
      priority: low
      tags:
        - name: tags.http
          ruleType: is
          values: [`+tcoNames("value", 50)+`]
`))
		},
	})
	return s
}

// tcoTracesAdmission holds every case of the released validation tests, and a few more limits.
func tcoTracesAdmission() map[string]string {
	policy := func(fields string) string {
		return tcoManifest(tcoTracesKind, "  policies:\n    - name: policy\n"+fields)
	}
	const dpxl = "      dpxlExpression: \"<v1> $d.status == 'ERROR'\"\n"
	rule := func(field string) string { return "      " + field + ":\n        names: [x]\n        ruleType: is\n" }
	return map[string]string{
		"dpxl expression":                   policy("      priority: medium\n" + dpxl),
		"dpxl expression and services":      policy("      priority: low\n" + dpxl + rule("services")),
		"dpxl expression and actions":       policy("      priority: low\n" + dpxl + rule("actions")),
		"dpxl expression and applications":  policy("      priority: low\n" + dpxl + rule("applications")),
		"dpxl expression and subsystems":    policy("      priority: low\n" + dpxl + rule("subsystems")),
		"dpxl expression and tags":          policy("      priority: low\n" + dpxl + "      tags:\n        - name: tags.env\n          ruleType: is\n          values: [prod]\n"),
		"dpxl expression and an empty tags": policy("      priority: low\n" + dpxl + "      tags: []\n"),
		"empty dpxl expression":             policy("      priority: low\n      dpxlExpression: \"\"\n"),
		"policy override":                   policy("      priority: low\n      priorityOverride:\n        quotaBased:\n          usageTiers:\n            - dailyQuotaPercentage: 30\n              priority: high\n            - dailyQuotaPercentage: 60\n              priority: medium\n"),
		"no priority":                       policy("      services:\n        names: [x]\n        ruleType: is\n"),
		"tag name without the tags prefix":  policy("      priority: low\n      tags:\n        - name: http\n          ruleType: is\n          values: [GET]\n"),
		"tag without values":                policy("      priority: low\n      tags:\n        - name: tags.http\n          ruleType: is\n          values: []\n"),
		"tag with more than 50 values":      policy("      priority: low\n      tags:\n        - name: tags.http\n          ruleType: is\n          values: [" + tcoNames("value", 51) + "]\n"),
		"retention with both name and id":   policy("      priority: low\n      archiveRetention:\n        backendRef:\n          name: Default\n          id: " + unknownRetentionID + "\n"),
		"more than 200 policies":            tcoManifest(tcoTracesKind, tcoManyPolicies(201, "      priority: low\n")),
	}
}
