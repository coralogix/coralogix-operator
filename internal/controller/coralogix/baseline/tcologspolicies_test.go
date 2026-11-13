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
	"maps"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

func init() {
	register(kind{
		name:     "tcologspolicies",
		crdFile:  "coralogix.com_tcologspolicies.yaml",
		resource: "tcologspolicies",
		newFake:  newTCOFake(tcoLogs),
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.TCOLogsPoliciesReconciler{
				Client:    cs.TCOPolicies(),
				ClientSet: cs,
				Interval:  reconcileInterval,
			}
		},
		scenarios: tcoLogsScenarios(),
		admission: tcoLogsAdmission(),
		fidelity:  fidelityConfig{overwritesTeam: true},
	})
}

const tcoLogsKind = "TCOLogsPolicies"

const tcoLogsMinimal = `  policies:
    - name: logs-severities
      priority: high
      severities: [info, error]
`

const tcoLogsUpdated = `  policies:
    - name: logs-severities
      priority: low
      severities: [warning]
    - name: logs-second
      priority: medium
      severities: [critical]
`

const tcoLogsRejected = `  policies:
    - name: logs-no-version
      priority: low
      dpxlExpression: "$d.applicationname == 'prod'"
`

// tcoLogsFull sets every field of a logs policy. dpxlExpression excludes severities,
// applications, and subsystems, and targets exclude the policy priority, so three policies
// cover every field.
const tcoLogsFull = `  policies:
    - name: logs-structured
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
    - name: logs-dpxl
      description: DPXL rules
      priority: medium
      disabled: false
      dpxlExpression: "<v1>$d.applicationname == 'prod'"
    - name: logs-targets
      severities: [info]
      applications:
        names: [dev]
        ruleType: includes
      subsystems:
        names: [test]
        ruleType: is_not
      targets:
        - dataset: myDataset
          dataspace: default
          priority: low
        - dataset: archiveDataset
          priority: block
          priorityOverride:
            quotaBased:
              usageTiers:
                - dailyQuotaPercentage: 60
                  priority: medium
          archiveRetention:
            backendRef:
              id: ` + defaultRetentionID + `
`

func tcoLogsScenarios() map[string]func(h *harness) {
	s := tcoCommonScenarios(tcoLogsKind, tcoLogs, tcoLogsMinimal, tcoLogsUpdated, tcoLogsRejected)
	maps.Copy(s, map[string]func(h *harness){
		"create-sample": func(h *harness) {
			h.CreateFromFile("create the sample", "config/samples/v1alpha1/tcologspolicies/example-tco-logs-policies.yaml")
		},

		"create-full": func(h *harness) {
			h.Create("create with every field", tcoManifest(tcoLogsKind, tcoLogsFull))
		},

		"clear-optional": func(h *harness) {
			h.Create("create with every field", tcoManifest(tcoLogsKind, tcoLogsFull))
			h.SetSpec("remove every optional field: description is sent as an empty string", tcoSpec(h, `policies:
  - name: logs-structured
    priority: high
    severities: [info]
  - name: logs-dpxl
    priority: medium
    dpxlExpression: "<v1>$d.applicationname == 'prod'"
  - name: logs-targets
    severities: [info]
    targets:
      - dataset: myDataset
        priority: low
`))
		},

		// The policy priority is optional. When it is not set, the operator sends
		// PRIORITY_TYPE_UNSPECIFIED, and the API rejects a policy without targets.
		"create-without-priority": func(h *harness) {
			h.Create("create without a priority or targets: the API rejects it", tcoManifest(tcoLogsKind, `  policies:
    - name: logs-no-priority
      severities: [info]
`))
		},

		"create-targets-without-priority": func(h *harness) {
			h.Create("create with targets that each have a priority, and no policy priority", tcoManifest(tcoLogsKind, `  policies:
    - name: targets-own-priority
      severities: [info]
      targets:
        - dataset: ds1
          priority: low
        - dataset: ds2
          priority: medium
`))
		},

		"create-override-without-priority": func(h *harness) {
			h.Create("create with a policy override and no priority", tcoManifest(tcoLogsKind, `  policies:
    - name: override-only
      severities: [info]
      priorityOverride:
        quotaBased:
          usageTiers:
            - dailyQuotaPercentage: 60
              priority: medium
`))
		},

		"create-dpxl-and-override": func(h *harness) {
			h.Create("create with a DPXL expression and a policy override", tcoManifest(tcoLogsKind, `  policies:
    - name: logs-advanced
      priority: low
      dpxlExpression: "<v1>$d.applicationname == 'prod'"
      priorityOverride:
        quotaBased:
          usageTiers:
            - dailyQuotaPercentage: 30
              priority: high
            - dailyQuotaPercentage: 60
              priority: medium
`))
		},

		"create-fractional-quota": func(h *harness) {
			h.Create("create with a fractional daily quota percentage on a target", tcoManifest(tcoLogsKind, `  policies:
    - name: quota-override
      severities: [info]
      targets:
        - dataset: myDataset
          priority: low
          priorityOverride:
            quotaBased:
              usageTiers:
                - dailyQuotaPercentage: "60.5"
                  priority: medium
`))
		},

		"archive-retention-by-name-on-target": func(h *harness) {
			h.Create("create with a retention name on a target only: one retentions read", tcoManifest(tcoLogsKind, `  policies:
    - name: target-retention
      severities: [info]
      targets:
        - dataset: myDataset
          priority: low
          archiveRetention:
            backendRef:
              name: Default
`))
		},

		"archive-retention-names-not-found": func(h *harness) {
			h.Create("create with two retention names that the team does not have: both are reported, no overwrite", tcoManifest(tcoLogsKind, `  policies:
    - name: first
      severities: [info]
      archiveRetention:
        backendRef:
          name: Missing
      targets:
        - dataset: myDataset
          priority: low
          archiveRetention:
            backendRef:
              name: AlsoMissing
`))
		},

		"rule-names-joined": func(h *harness) {
			h.Create("create with 50 subsystem names: they are sent as one comma-separated name", tcoManifest(tcoLogsKind, `  policies:
    - name: at-limit
      priority: low
      severities: [info]
      subsystems:
        names: [`+tcoNames("subsystem", 50)+`]
        ruleType: is
`))
		},
	})
	return s
}

// tcoNames returns n names with the prefix, as a YAML flow list body.
func tcoNames(prefix string, n int) string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("%s-%d", prefix, i)
	}
	return strings.Join(names, ", ")
}

// tcoLogsAdmission holds every case of the released validation tests, and a few more limits.
func tcoLogsAdmission() map[string]string {
	policy := func(fields string) string {
		return tcoManifest(tcoLogsKind, "  policies:\n    - name: policy\n"+fields)
	}
	return map[string]string{
		"more than 50 subsystem names":                             policy("      priority: low\n      severities: [info]\n      subsystems:\n        names: [" + tcoNames("subsystem", 51) + "]\n        ruleType: is\n"),
		"exactly 50 subsystem names":                               policy("      priority: low\n      severities: [info]\n      subsystems:\n        names: [" + tcoNames("subsystem", 50) + "]\n        ruleType: is\n"),
		"no subsystem names":                                       policy("      priority: low\n      severities: [info]\n      subsystems:\n        names: []\n        ruleType: is\n"),
		"target with an empty dataset":                             policy("      severities: [info]\n      targets:\n        - dataset: \"\"\n          priority: low\n"),
		"valid target":                                             policy("      severities: [info]\n      targets:\n        - dataset: myDataset\n          priority: low\n"),
		"targets that each have a priority and no policy priority": policy("      severities: [info]\n      targets:\n        - dataset: ds1\n          priority: low\n        - dataset: ds2\n          priority: medium\n"),
		"target without a priority":                                policy("      severities: [info]\n      targets:\n        - dataset: myDataset\n"),
		"target with a dataspace longer than 50":                   policy("      severities: [info]\n      targets:\n        - dataset: myDataset\n          dataspace: " + strings.Repeat("d", 51) + "\n          priority: low\n"),
		"target override with a fractional percentage":             policy("      severities: [info]\n      targets:\n        - dataset: myDataset\n          priority: low\n          priorityOverride:\n            quotaBased:\n              usageTiers:\n                - dailyQuotaPercentage: \"60.5\"\n                  priority: medium\n"),
		"override without usage tiers":                             policy("      priority: low\n      severities: [info]\n      priorityOverride:\n        quotaBased:\n          usageTiers: []\n"),
		"retention by name":                                        policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          name: Default\n"),
		"retention by id":                                          policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          id: " + unknownRetentionID + "\n"),
		"retention id that is not a UUID":                          policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          id: not-a-uuid\n"),
		"retention with both name and id":                          policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          name: Default\n          id: " + unknownRetentionID + "\n"),
		"retention with an empty name":                             policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef:\n          name: \"\"\n"),
		"retention with neither name nor id":                       policy("      priority: low\n      severities: [info]\n      archiveRetention:\n        backendRef: {}\n"),
		"target retention with both name and id":                   policy("      severities: [info]\n      targets:\n        - dataset: myDataset\n          priority: low\n          archiveRetention:\n            backendRef:\n              name: Default\n              id: " + unknownRetentionID + "\n"),
		"dpxl expression":                                          policy("      priority: medium\n      dpxlExpression: \"<v1>$d.applicationname == 'prod'\"\n"),
		"empty dpxl expression":                                    policy("      priority: medium\n      dpxlExpression: \"\"\n"),
		"severities and dpxl expression":                           policy("      priority: low\n      severities: [info]\n      dpxlExpression: \"<v1>$d.applicationname == 'prod'\"\n"),
		"empty severities and dpxl expression":                     policy("      priority: low\n      severities: []\n      dpxlExpression: \"<v1>$d.applicationname == 'prod'\"\n"),
		"dpxl expression and applications":                         policy("      priority: low\n      dpxlExpression: \"<v1>$d.applicationname == 'prod'\"\n      applications:\n        names: [prod]\n        ruleType: is\n"),
		"dpxl expression and subsystems":                           policy("      priority: low\n      dpxlExpression: \"<v1>$d.applicationname == 'prod'\"\n      subsystems:\n        names: [mobile]\n        ruleType: is\n"),
		"policy override":                                          policy("      priority: low\n      severities: [info]\n      priorityOverride:\n        quotaBased:\n          usageTiers:\n            - dailyQuotaPercentage: 60\n              priority: medium\n"),
		"no priority, no targets":                                  policy("      severities: [info]\n"),
		"unknown priority":                                         policy("      priority: urgent\n      severities: [info]\n"),
		"unknown severity":                                         policy("      priority: low\n      severities: [fatal]\n"),
		"unknown rule type":                                        policy("      priority: low\n      severities: [info]\n      applications:\n        names: [prod]\n        ruleType: equals\n"),
		"no name":                                                  tcoManifest(tcoLogsKind, "  policies:\n    - priority: low\n      severities: [info]\n"),
		"no policies field":                                        tcoManifest(tcoLogsKind, "  {}\n"),
		"more than 200 policies":                                   tcoManifest(tcoLogsKind, tcoManyPolicies(201, "      priority: low\n      severities: [info]\n")),
	}
}

// tcoManyPolicies is a spec with n policies that have the fields.
func tcoManyPolicies(n int, fields string) string {
	var b strings.Builder
	b.WriteString("  policies:\n")
	for i := range n {
		fmt.Fprintf(&b, "    - name: policy-%d\n%s", i, fields)
	}
	return b.String()
}
