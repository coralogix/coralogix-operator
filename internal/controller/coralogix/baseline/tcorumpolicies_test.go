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
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "tcorumpolicies",
		crdFile:  "coralogix.com_tcorumpolicies.yaml",
		resource: "tcorumpolicies",
		newFake:  newTCOFake(tcoRum),
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.TCORumPoliciesReconciler{
				Client:    cs.TCOPolicies(),
				ClientSet: cs,
				Interval:  reconcileInterval,
			}
		},
		scenarios: tcoRumScenarios(),
		admission: tcoRumAdmission(),
		fidelity:  fidelityConfig{overwritesTeam: true},
	})
}

const tcoRumKind = "TCORumPolicies"

const tcoRumMinimal = `  policies:
    - name: rum-severities
      priority: high
      severities: [critical, error]
`

const tcoRumUpdated = `  policies:
    - name: rum-severities
      priority: low
      severities: [warning]
    - name: rum-second
      priority: medium
      dpxlExpression: "<v1>$d.applicationname == 'prod'"
`

const tcoRumRejected = `  policies:
    - name: rum-no-version
      priority: low
      dpxlExpression: "$d.applicationname == 'prod'"
`

// tcoRumFull sets every field of a RUM policy. dpxlExpression excludes severities, so two
// policies cover every field.
const tcoRumFull = `  policies:
    - name: rum-structured
      description: Structured rules
      priority: high
      disabled: true
      severities: [debug, verbose, info, warning, error, critical]
      applications:
        names: [prod, staging]
        ruleType: is
      subsystems:
        names: [mobile, web]
        ruleType: start_with
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
    - name: rum-dpxl
      description: DPXL rules
      priority: medium
      disabled: false
      dpxlExpression: "<v1>$d.applicationname == 'prod'"
      applications:
        names: [prod]
        ruleType: includes
      subsystems:
        names: [web]
        ruleType: is_not
`

func tcoRumScenarios() map[string]func(h *harness) {
	s := tcoCommonScenarios(tcoRumKind, tcoRum, tcoRumMinimal, tcoRumUpdated, tcoRumRejected)
	maps.Copy(s, map[string]func(h *harness){
		"create-sample": func(h *harness) {
			h.CreateFromFile("create the sample", "config/samples/v1alpha1/tcorumpolicies/example-tco-rum-policies.yaml")
		},

		"create-full": func(h *harness) {
			h.Create("create with every field", tcoManifest(tcoRumKind, tcoRumFull))
		},

		"clear-optional": func(h *harness) {
			h.Create("create with every field", tcoManifest(tcoRumKind, tcoRumFull))
			h.SetSpec("remove every optional field: description is sent as an empty string", tcoSpec(h, `policies:
  - name: rum-structured
    priority: high
    severities: [info]
  - name: rum-dpxl
    priority: medium
    dpxlExpression: "<v1>$d.applicationname == 'prod'"
`))
		},

		// Admission accepts a policy with neither severities nor a DPXL expression. The API
		// rejects it.
		"create-without-rules": func(h *harness) {
			h.Create("create with neither severities nor a DPXL expression: the API rejects it", tcoManifest(tcoRumKind, `  policies:
    - name: rum-no-rules
      priority: low
`))
		},

		"create-quota-override": func(h *harness) {
			h.Create("create with a fractional quota-based override", tcoManifest(tcoRumKind, `  policies:
    - name: quota-override
      priority: low
      severities: [info]
      priorityOverride:
        quotaBased:
          usageTiers:
            - dailyQuotaPercentage: "60.5"
              priority: medium
`))
		},
	})
	return s
}

// tcoRumAdmission holds every case of the released validation tests, and a few more limits.
func tcoRumAdmission() map[string]string {
	policy := func(fields string) string {
		return tcoManifest(tcoRumKind, "  policies:\n    - name: policy\n"+fields)
	}
	const dpxl = "      dpxlExpression: \"<v1>$d.applicationname == 'prod'\"\n"
	return map[string]string{
		"more than 50 subsystem names":           policy("      priority: low\n      severities: [info]\n      subsystems:\n        names: [" + tcoNames("subsystem", 51) + "]\n        ruleType: is\n"),
		"severities and applications":            policy("      priority: high\n      severities: [critical, error]\n      applications:\n        names: [prod]\n        ruleType: is\n"),
		"dpxl expression":                        policy("      priority: medium\n" + dpxl),
		"dpxl expression and applications":       policy("      priority: medium\n" + dpxl + "      applications:\n        names: [prod]\n        ruleType: is\n"),
		"severities and dpxl expression":         policy("      priority: low\n      severities: [info]\n" + dpxl),
		"neither severities nor dpxl expression": policy("      priority: low\n"),
		"empty severities":                       policy("      priority: low\n      severities: []\n"),
		"more than 6 severities":                 policy("      priority: low\n      severities: [info, info, warning, error, critical, debug, verbose]\n"),
		"empty dpxl expression":                  policy("      priority: low\n      dpxlExpression: \"\"\n"),
		"no priority":                            policy("      severities: [info]\n"),
		"quota-based override":                   policy("      priority: low\n      severities: [info]\n      priorityOverride:\n        quotaBased:\n          usageTiers:\n            - dailyQuotaPercentage: \"60.5\"\n              priority: medium\n"),
		"retention by name":                      policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          name: Default\n"),
		"retention by id":                        policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          id: " + unknownRetentionID + "\n"),
		"retention with both name and id":        policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          name: Default\n          id: " + unknownRetentionID + "\n"),
		"retention with an empty name":           policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          name: \"\"\n"),
		"retention with neither name nor id":     policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef: {}\n"),
		"empty name":                             tcoManifest(tcoRumKind, "  policies:\n    - name: \"\"\n      priority: low\n      severities: [info]\n"),
		"name longer than 255":                   tcoManifest(tcoRumKind, "  policies:\n    - name: "+strings.Repeat("n", 256)+"\n      priority: low\n      severities: [info]\n"),
		"name of 255":                            tcoManifest(tcoRumKind, "  policies:\n    - name: "+strings.Repeat("n", 255)+"\n      priority: low\n      severities: [info]\n"),
	}
}
