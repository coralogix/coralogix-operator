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
	"net/http"
	"os"
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	v1alpha1controllers "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/v1alpha1"
)

const alertSchedulerPath = "/alerts/suppression-rules/v1"

// The alert ids that the fake knows. They are not fixed ids, so the fidelity recording, which
// writes <id> for the real alert of the team, compares equal with the fake.
const (
	asAlertID      = "a1e7a1e7-0000-4000-8000-000000000001"
	asOtherAlertID = "a1e7a1e7-0000-4000-8000-000000000002"
	// asUnknownAlertID is an alert id that the team does not have.
	asUnknownAlertID = fixedIDPrefix + "0000000000a9"
	// asUnknownRuleID is a rule id that the team does not have.
	asUnknownRuleID = fixedIDPrefix + "0000000000a1"
)

func init() {
	register(kind{
		name:     "alertscheduler",
		crdFile:  "coralogix.com_alertschedulers.yaml",
		resource: "alertschedulers",
		newFake:  newAlertSchedulerFake,
		newReconciler: func(cs *openapicxsdk.ClientSet) reconcile.Reconciler {
			return &v1alpha1controllers.AlertSchedulerReconciler{AlertSchedulerClient: cs.AlertScheduler(), Interval: reconcileInterval}
		},
		scenarios: alertSchedulerScenarios,
		admission: alertSchedulerAdmission(),
		fidelity: fidelityConfig{
			captureField: "alertSchedulerRule.uniqueIdentifier",
			deletePath:   func(id string) string { return alertSchedulerPath + "/" + id },
			recordVars:   alertSchedulerRecordVars,
			fakeVars:     map[string]string{"alert": asAlertID},
			normalize:    sortRuleMetaLabels,
		},
	})
}

// alertSchedulerRecordVars names an alert of the team, for the case that filters by alert id. Set
// CORALOGIX_ALERT_ID, or the recorder takes the first alert of the team. The cases only name the
// alert in a rule; they never change it.
func alertSchedulerRecordVars(t *testing.T, base, apiKey string) map[string]string {
	t.Helper()
	if id := os.Getenv("CORALOGIX_ALERT_ID"); id != "" {
		return map[string]string{"alert": id}
	}
	status, resp, err := sendRecordRequest(base, apiKey, http.MethodGet, "/alerts/alerts/v3", nil)
	if err != nil || status != http.StatusOK {
		t.Fatalf("list alerts: %d %v", status, err)
	}
	for _, alert := range listOf(resp, "alertDefs") {
		if id, _ := alert["id"].(string); id != "" {
			return map[string]string{"alert": id}
		}
	}
	t.Fatal("the team has no alert: create one, or set CORALOGIX_ALERT_ID")
	return nil
}

// sortRuleMetaLabels sorts the meta labels of a rule by key. The API stores them as a set and
// answers them in the order of their new random ids.
func sortRuleMetaLabels(resp any) any {
	root, _ := resp.(map[string]any)
	rule, _ := root["alertSchedulerRule"].(map[string]any)
	if labels, ok := rule["metaLabels"].([]any); ok {
		sort.SliceStable(labels, func(i, j int) bool {
			a, _ := labels[i].(map[string]any)
			b, _ := labels[j].(map[string]any)
			ka, _ := a["key"].(string)
			kb, _ := b["key"].(string)
			return ka < kb
		})
	}
	return resp
}

// alertSchedulerManifest returns an AlertScheduler CR named "scheduler". spec is YAML, indented
// by two spaces.
func alertSchedulerManifest(spec string, labels ...string) string {
	meta := "  name: scheduler\n"
	if len(labels) > 0 {
		meta += "  labels:\n"
		for i := 0; i+1 < len(labels); i += 2 {
			meta += "    " + labels[i] + ": " + labels[i+1] + "\n"
		}
	}
	return "apiVersion: coralogix.com/v1alpha1\nkind: AlertScheduler\nmetadata:\n" + meta + "spec:\n" + spec
}

const (
	asWhat         = "  name: Baseline scheduler\n"
	asLabelsFilter = "  filter:\n    whatExpression: source logs | filter true\n    metaLabels:\n      - key: app\n        value: cxo\n"
	asOneTime      = "  schedule:\n    operation: mute\n    oneTime:\n      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00.000\n"
)

// asSpec returns the YAML of a spec with the name, a meta label filter, and the schedule given.
// schedule is YAML, indented by four spaces, below `schedule:`.
func asSpec(schedule string) string {
	return asWhat + asLabelsFilter + "  schedule:\n" + schedule
}

// asDynamic returns the schedule YAML of a recurring dynamic schedule with the frequency given.
// frequency is YAML, indented by 10 spaces, below `frequency:`.
func asDynamic(frequency string) string {
	return "    operation: mute\n    recurring:\n      dynamic:\n        repeatEvery: 1\n        frequency:\n" + frequency +
		"        timeFrame:\n          startTime: 2027-01-04T08:00:00.000\n          timezone: UTC+2\n          duration:\n            forOver: 30\n            frequency: minutes\n"
}

func asAlertsFilter(refs string) string {
	return "  filter:\n    whatExpression: source logs | filter true\n    alerts:\n" + refs
}

var asMinimal = alertSchedulerManifest(asWhat + asLabelsFilter + asOneTime)

// asSpecMap is the spec of asMinimal with the fields given, as SetSpec takes it.
func asSpecMap(fields ...any) map[string]any {
	spec := map[string]any{
		"name": "Baseline scheduler",
		"filter": map[string]any{
			"whatExpression": "source logs | filter true",
			"metaLabels":     []any{map[string]any{"key": "app", "value": "cxo"}},
		},
		"schedule": map[string]any{
			"operation": "mute",
			"oneTime": map[string]any{
				"startTime": "2027-01-04T00:00:00.000",
				"timezone":  "UTC+0",
				"endTime":   "2027-01-05T00:00:00.000",
			},
		},
	}
	for i := 0; i+1 < len(fields); i += 2 {
		spec[fields[i].(string)] = fields[i+1]
	}
	return spec
}

func alertSchedulers(h *harness) *alertSchedulerFake {
	h.t.Helper()
	f, ok := h.API().(*alertSchedulerFake)
	if !ok {
		h.t.Fatal("expected *alertSchedulerFake")
	}
	return f
}

// createAlert creates a v1beta1 Alert CR without a reconciler. A non-empty id is set as its status
// id, as the Alert controller does after Create.
func createAlert(h *harness, namespace, name, id string) {
	h.t.Helper()
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "coralogix.com/v1beta1",
		"kind":       "Alert",
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"spec": map[string]any{
			"name":     name,
			"priority": "p3",
			"alertType": map[string]any{"logsImmediate": map[string]any{
				"logsFilter": map[string]any{"simpleFilter": map[string]any{"luceneQuery": "error"}},
			}},
		},
	}}
	if err := h.client.Create(context.Background(), obj); err != nil {
		h.t.Fatalf("create Alert %s: %v", name, err)
	}
	if id == "" {
		return
	}
	if err := unstructured.SetNestedField(obj.Object, id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set Alert %s status id: %v", name, err)
	}
}

// createAlertWithEmptyStatusID creates a v1beta1 Alert CR whose status id is present and empty.
func createAlertWithEmptyStatusID(h *harness, namespace, name string) {
	h.t.Helper()
	createAlert(h, namespace, name, "")
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("coralogix.com/v1beta1")
	obj.SetKind("Alert")
	if err := h.client.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, obj); err != nil {
		h.t.Fatalf("get Alert %s: %v", name, err)
	}
	if err := unstructured.SetNestedField(obj.Object, "", "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set Alert %s status id: %v", name, err)
	}
}

// setAlertSchedulerStatusID writes status.id of the CR, or removes it when id is nil, as a user
// who edits the status could.
func setAlertSchedulerStatusID(h *harness, id *string) {
	h.t.Helper()
	obj := h.get()
	if id == nil {
		unstructured.RemoveNestedField(obj.Object, "status", "id")
	} else if err := unstructured.SetNestedField(obj.Object, *id, "status", "id"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.client.Status().Update(context.Background(), obj); err != nil {
		h.t.Fatalf("set AlertScheduler status id: %v", err)
	}
}

var alertSchedulerScenarios = map[string]func(h *harness){
	// The sample names the Alert CR metric-threshold.
	"create-sample-one-time": func(h *harness) {
		createAlert(h, h.namespace, "metric-threshold", asAlertID)
		h.CreateFromFile("create the one-time sample", "config/samples/v1alpha1/alertscheduler/one-time-alert-scheduler.yaml")
		h.Reconcile("reconcile again: the resync sends a Replace and no Get")
	},

	"create-sample-recurring": func(h *harness) {
		h.CreateFromFile("create the recurring sample", "config/samples/v1alpha1/alertscheduler/recurring-alert-scheduler.yaml")
		h.Reconcile("reconcile again")
	},

	"create-minimal": func(h *harness) {
		h.Create("create with the required fields: description and enabled are sent", asMinimal)
		h.Reconcile("reconcile again: the resync sends a Replace with the status id")
		h.Reconcile("reconcile again")
	},

	// A released operator created the rule: the CR has its finalizer and status id. This is the
	// first reconcile after an upgrade.
	"upgrade-from-released": func(h *harness) {
		id := alertSchedulers(h).seed()
		h.CreateReleased("the released operator made the rule: reconcile the CR", asMinimal,
			[]string{"alert-scheduler.coralogix.com/finalizer"}, map[string]any{"id": id, "printableStatus": "RemoteSynced"})
		if got := h.StatusID(); got != id {
			h.t.Fatalf("status.id changed from %s to %s", id, got)
		}
		h.Reconcile("reconcile again")
		h.Delete("delete")
		h.RecordRemote("the remote rule is gone", alertSchedulers(h).rules)
	},

	"create-every-field": func(h *harness) {
		h.Create("create with every field", alertSchedulerManifest(
			"  name: Baseline scheduler\n  description: Every field\n  enabled: true\n"+
				"  metaLabels:\n    - key: team\n      value: cxo\n    - key: tier\n"+
				asLabelsFilter+"  schedule:\n"+
				"    operation: activate\n    recurring:\n      dynamic:\n        repeatEvery: 3\n        terminationDate: 2028-01-01T00:00:00.000\n"+
				"        frequency:\n          weekly:\n            days: [Monday, Wednesday]\n"+
				"        timeFrame:\n          startTime: 2027-01-04T08:00:00.000\n          timezone: UTC+2\n          endTime: 2027-01-04T09:00:00.000\n"))
	},

	"create-disabled": func(h *harness) {
		h.Create("create disabled: enabled false is sent", alertSchedulerManifest(
			"  name: Baseline scheduler\n  enabled: false\n"+asLabelsFilter+asOneTime))
		h.Reconcile("resync: adding the finalizer dropped enabled false, the default set it to true, and the Replace enables the rule")
	},

	"create-meta-label-without-value": func(h *harness) {
		h.Create("create with meta labels without a value", alertSchedulerManifest(
			asWhat+"  metaLabels:\n    - key: team\n"+
				"  filter:\n    whatExpression: source logs | filter true\n    metaLabels:\n      - key: env\n"+asOneTime))
	},

	"create-empty-meta-labels-filter": func(h *harness) {
		h.Create("create with an empty meta label filter: the rule is created, and adding the finalizer fails", alertSchedulerManifest(
			asWhat+"  filter:\n    whatExpression: source logs | filter true\n    metaLabels: []\n"+asOneTime))
		h.Reconcile("reconcile again")
		h.Delete("delete")
		h.RecordRemote("the remote rule stays", alertSchedulers(h).rules)
	},

	"create-one-time-duration-minutes": func(h *harness) {
		h.Create("create a one-time schedule of 90 minutes", alertSchedulerManifest(asSpec(
			"    operation: activate\n    oneTime:\n      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC-4\n      duration:\n        forOver: 90\n        frequency: minutes\n")))
	},

	"create-one-time-duration-hours": func(h *harness) {
		h.Create("create a one-time schedule of 2 hours", alertSchedulerManifest(asSpec(
			"    operation: mute\n    oneTime:\n      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+10\n      duration:\n        forOver: 2\n        frequency: hours\n")))
	},

	"create-one-time-duration-days": func(h *harness) {
		h.Create("create a one-time schedule of 7 days", alertSchedulerManifest(asSpec(
			"    operation: mute\n    oneTime:\n      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        forOver: 7\n        frequency: days\n")))
	},

	"create-daily": func(h *harness) {
		h.Create("create a daily schedule", alertSchedulerManifest(asSpec(asDynamic("          daily: {}\n"))))
	},

	"create-weekly-every-day": func(h *harness) {
		h.Create("create a weekly schedule on every day: the days are sent as 1 (Sunday) to 7", alertSchedulerManifest(asSpec(asDynamic(
			"          weekly:\n            days: [Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday]\n"))))
	},

	"create-monthly": func(h *harness) {
		h.Create("create a monthly schedule", alertSchedulerManifest(asSpec(asDynamic(
			"          monthly:\n            days: [4, 15]\n"))))
	},

	"create-always": func(h *harness) {
		h.Create("create an always-active schedule", alertSchedulerManifest(asSpec(
			"    operation: activate\n    recurring:\n      always: {}\n")))
		h.Reconcile("reconcile again")
	},

	// The API stores a recurring schedule with repeatEvery 0 as always active. The operator does not
	// read it back, so each resync sends the dynamic schedule again.
	"create-repeat-every-zero": func(h *harness) {
		h.Create("create a daily schedule with repeatEvery 0: the API stores it as always active", alertSchedulerManifest(asSpec(
			"    operation: mute\n    recurring:\n      dynamic:\n        repeatEvery: 0\n        frequency:\n          daily: {}\n"+
				"        timeFrame:\n          startTime: 2027-01-04T08:00:00.000\n          timezone: UTC+2\n          duration:\n            forOver: 30\n            frequency: minutes\n")))
		h.RecordRemote("the remote rule", alertSchedulers(h).rule(h.StatusID()))
		h.Reconcile("reconcile again: the same Replace")
	},

	// The CRD has no rule on frequency. An empty frequency fails before any request.
	"create-frequency-empty": func(h *harness) {
		h.Create("create with an empty frequency: no request", alertSchedulerManifest(asSpec(asDynamic("          {}\n"))))
		h.Delete("delete")
	},

	// With two frequencies, the operator sends the first of daily, weekly, and monthly.
	"create-frequency-daily-and-weekly": func(h *harness) {
		h.Create("create with daily and weekly: daily is sent", alertSchedulerManifest(asSpec(asDynamic(
			"          daily: {}\n          weekly:\n            days: [Monday]\n"))))
	},

	// The CRD accepts a CR without spec. Reconcile then finds no schedule and fails before any request.
	"create-without-spec": func(h *harness) {
		h.Create("create without spec: no request", "apiVersion: coralogix.com/v1alpha1\nkind: AlertScheduler\nmetadata:\n  name: scheduler\n")
		h.Delete("delete")
	},

	// The CRD accepts a forOver that int32 cannot hold. The typed Get of the reconciler cannot decode
	// the CR.
	"create-for-over-above-int32": func(h *harness) {
		h.Create("create with forOver 3000000000: the CR cannot be decoded", alertSchedulerManifest(asSpec(
			"    operation: mute\n    oneTime:\n      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        forOver: 3000000000\n        frequency: days\n")))
		h.Delete("delete")
	},

	"create-alerts": func(h *harness) {
		createAlert(h, h.namespace, "first", asAlertID)
		createAlert(h, h.namespace, "second", asOtherAlertID)
		h.Create("create with two Alert CRs: their status ids are sent", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n      - resourceRef:\n          name: second\n")+asOneTime))
		h.Reconcile("reconcile again")
	},

	"create-alerts-other-namespace": func(h *harness) {
		alerts := createNamespace(h, namespaceName(h.namespace+"-alerts"))
		createAlert(h, alerts, "first", asAlertID)
		h.Create("create with an Alert CR of another namespace", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n          namespace: "+alerts+"\n")+asOneTime))
	},

	"create-alerts-missing": func(h *harness) {
		createAlert(h, h.namespace, "first", asAlertID)
		h.Create("do not call the API when an Alert CR does not exist", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n      - resourceRef:\n          name: missing\n")+asOneTime))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-alerts-without-id": func(h *harness) {
		createAlert(h, h.namespace, "first", "")
		h.Create("do not call the API before the Alert CR has a status id", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n")+asOneTime))
		h.Delete("delete")
	},

	// A status id that is present but empty is not a missing id: the handwritten code sends the empty
	// id, and the API rejects it.
	"create-alerts-empty-id": func(h *harness) {
		createAlertWithEmptyStatusID(h, h.namespace, "first")
		h.Create("create with an Alert CR whose status id is empty: the empty id is sent", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n")+asOneTime))
		h.Delete("delete")
	},

	"create-alerts-not-selected": func(h *harness) {
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "alerts"})})
		createAlert(h, h.namespace, "first", asAlertID)
		h.Create("do not call the API when the selector does not match the Alert CR", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n")+asOneTime, "team", "alerts"))
		h.Delete("delete")
	},

	"create-alerts-unknown-id": func(h *harness) {
		createAlert(h, h.namespace, "first", asUnknownAlertID)
		h.Create("create with an Alert CR whose id the team does not have: the API rejects it", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n")+asOneTime))
		h.Delete("delete")
	},

	"create-alerts-empty": func(h *harness) {
		h.Create("create with an empty alerts list: the rule is created, and adding the finalizer fails", alertSchedulerManifest(
			asWhat+"  filter:\n    whatExpression: source logs | filter true\n    alerts: []\n"+asOneTime))
		h.Reconcile("reconcile again")
		h.Delete("delete")
		h.RecordRemote("the remote rule stays", alertSchedulers(h).rules)
	},

	"create-rejected-expression": func(h *harness) {
		h.Create("create with an expression that the API rejects", alertSchedulerManifest(
			asWhat+"  filter:\n    whatExpression: source logs\n    metaLabels:\n      - key: app\n        value: cxo\n"+asOneTime))
		h.Reconcile("reconcile again")
		h.Delete("delete")
	},

	"create-rejected-month-day": func(h *harness) {
		h.Create("create a monthly schedule with day 32: the API rejects it", alertSchedulerManifest(asSpec(asDynamic(
			"          monthly:\n            days: [32]\n"))))
	},

	"create-rejected-route-not-found": func(h *harness) {
		alertSchedulers(h).answerNextRouteNotFound(http.MethodPost)
		h.Create("create: the API has no such route", asMinimal)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"create-malformed-response": func(h *harness) {
		alertSchedulers(h).answerNext(http.MethodPost, http.StatusOK, "not a rule")
		h.Create("create: the answer cannot be decoded", asMinimal)
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// An answer without uniqueIdentifier gives the CR an empty status id. The next reconcile
	// creates again.
	"create-empty-id": func(h *harness) {
		alertSchedulers(h).answerNext(http.MethodPost, http.StatusOK, map[string]any{"alertSchedulerRule": map[string]any{
			"name": "Baseline scheduler", "filter": map[string]any{}, "schedule": map[string]any{},
		}})
		h.Create("create: the answer has no uniqueIdentifier", asMinimal)
		h.Reconcile("reconcile again")
	},

	"update-rename": func(h *harness) {
		h.Create("create", asMinimal)
		h.SetSpec("change the name and the description", asSpecMap("name", "Renamed scheduler", "description", "Renamed"))
		h.Reconcile("reconcile again")
	},

	"update-remove-description": func(h *harness) {
		h.Create("create with a description", alertSchedulerManifest("  name: Baseline scheduler\n  description: Gone soon\n"+asLabelsFilter+asOneTime))
		h.SetSpec("remove the description: an empty description is sent", asSpecMap())
	},

	"update-remove-meta-labels": func(h *harness) {
		h.Create("create with meta labels", alertSchedulerManifest(asWhat+"  metaLabels:\n    - key: team\n      value: cxo\n"+asLabelsFilter+asOneTime))
		h.SetSpec("remove the meta labels: the request leaves them out", asSpecMap())
		h.RecordRemote("the remote rule", alertSchedulers(h).rule(h.StatusID()))
	},

	"update-disable": func(h *harness) {
		h.Create("create", asMinimal)
		h.SetSpec("disable", asSpecMap("enabled", false))
	},

	// SetSpec replaces the spec, so a spec without enabled gets the default true again.
	"update-remove-enabled": func(h *harness) {
		h.Create("create disabled", alertSchedulerManifest("  name: Baseline scheduler\n  enabled: false\n"+asLabelsFilter+asOneTime))
		h.SetSpec("remove enabled: the default sets it to true", asSpecMap())
	},

	"update-to-recurring": func(h *harness) {
		h.Create("create a one-time schedule", asMinimal)
		h.SetSpec("switch to a monthly schedule", asSpecMap("schedule", map[string]any{
			"operation": "activate",
			"recurring": map[string]any{"dynamic": map[string]any{
				"repeatEvery": int64(2),
				"frequency":   map[string]any{"monthly": map[string]any{"days": []any{int64(1)}}},
				"timeFrame": map[string]any{
					"startTime": "2027-01-01T00:00:00.000",
					"timezone":  "UTC+0",
					"duration":  map[string]any{"forOver": int64(1), "frequency": "days"},
				},
			}},
		}))
	},

	"update-filter-to-alerts": func(h *harness) {
		createAlert(h, h.namespace, "first", asAlertID)
		h.Create("create with a meta label filter", asMinimal)
		h.SetSpec("filter by an Alert CR", asSpecMap("filter", map[string]any{
			"whatExpression": "source logs | filter true",
			"alerts":         []any{map[string]any{"resourceRef": map[string]any{"name": "first"}}},
		}))
	},

	// The Alert CR gets a new status id, such as after it was created again. The resync sends it.
	"update-alert-id-follows-alert": func(h *harness) {
		createAlert(h, h.namespace, "first", asAlertID)
		h.Create("create with an Alert CR", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n")+asOneTime))
		obj := &unstructured.Unstructured{}
		obj.SetAPIVersion("coralogix.com/v1beta1")
		obj.SetKind("Alert")
		if err := h.client.Get(context.Background(), client.ObjectKey{Namespace: h.namespace, Name: "first"}, obj); err != nil {
			h.t.Fatal(err)
		}
		if err := unstructured.SetNestedField(obj.Object, asOtherAlertID, "status", "id"); err != nil {
			h.t.Fatal(err)
		}
		if err := h.client.Status().Update(context.Background(), obj); err != nil {
			h.t.Fatal(err)
		}
		h.Reconcile("resync after the Alert CR got another status id")
	},

	"update-alert-deleted": func(h *harness) {
		createAlert(h, h.namespace, "first", asAlertID)
		h.Create("create with an Alert CR", alertSchedulerManifest(
			asWhat+asAlertsFilter("      - resourceRef:\n          name: first\n")+asOneTime))
		obj := &unstructured.Unstructured{}
		obj.SetAPIVersion("coralogix.com/v1beta1")
		obj.SetKind("Alert")
		obj.SetNamespace(h.namespace)
		obj.SetName("first")
		if err := h.client.Delete(context.Background(), obj); err != nil {
			h.t.Fatal(err)
		}
		h.Reconcile("resync after the Alert CR is gone: no request")
		h.Delete("delete")
	},

	"update-rejected": func(h *harness) {
		h.Create("create", asMinimal)
		h.SetSpec("change the expression to one that the API rejects", asSpecMap("filter", map[string]any{
			"whatExpression": "source logs",
			"metaLabels":     []any{map[string]any{"key": "app", "value": "cxo"}},
		}))
		h.RecordRemote("the remote rule keeps its expression", alertSchedulers(h).rule(h.StatusID()))
		h.Reconcile("reconcile again")
	},

	// A Replace of a rule that the API does not have answers 404 "No result found". The shared
	// reconciler removes the status id, and the next reconcile creates again.
	"update-when-gone": func(h *harness) {
		h.Create("create", asMinimal)
		alertSchedulers(h).remove(h.StatusID())
		h.Reconcile("resync after the rule was deleted outside the operator")
		h.Reconcile("reconcile again: create")
	},

	"update-malformed-response": func(h *harness) {
		h.Create("create", asMinimal)
		alertSchedulers(h).answerNext(http.MethodPut, http.StatusOK, "not a rule")
		h.Reconcile("resync: the answer cannot be decoded")
		h.Reconcile("reconcile again")
	},

	"update-route-not-found": func(h *harness) {
		h.Create("create", asMinimal)
		alertSchedulers(h).answerNextRouteNotFound(http.MethodPut)
		h.Reconcile("resync: the API has no such route")
		h.Reconcile("reconcile again")
	},

	"delete": func(h *harness) {
		h.Create("create", asMinimal)
		h.Delete("delete")
	},

	"delete-when-gone": func(h *harness) {
		h.Create("create", asMinimal)
		alertSchedulers(h).remove(h.StatusID())
		h.Delete("delete a rule that the API does not have: 404 is success")
	},

	"delete-rejected": func(h *harness) {
		h.Create("create", asMinimal)
		alertSchedulers(h).answerNextError(http.MethodDelete, http.StatusInternalServerError, "try again")
		h.Delete("delete: the API answers 500")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	"delete-route-not-found": func(h *harness) {
		h.Create("create", asMinimal)
		alertSchedulers(h).answerNextRouteNotFound(http.MethodDelete)
		h.Delete("delete: the API has no such route")
		h.Reconcile("reconcile again: the API answers as it should")
	},

	// The shared reconciler creates when the status has no id, even while the CR is deleted.
	"delete-without-id": func(h *harness) {
		h.Create("create", asMinimal)
		setAlertSchedulerStatusID(h, nil)
		h.Delete("delete a CR that has a finalizer and no status id")
		h.Reconcile("reconcile again")
	},

	"bad-status-id": func(h *harness) {
		h.Create("create", asMinimal)
		bad := "not-a-uuid"
		setAlertSchedulerStatusID(h, &bad)
		h.Reconcile("resync with a status id that is not a UUID")
		h.Delete("delete with a status id that is not a UUID: the API rejects it")
	},

	"unknown-status-id": func(h *harness) {
		h.Create("create", asMinimal)
		unknown := asUnknownRuleID
		setAlertSchedulerStatusID(h, &unknown)
		h.Reconcile("resync with a status id that the API does not have")
		h.Reconcile("reconcile again")
	},

	"selector-mismatch": func(h *harness) {
		h.Create("create", asMinimal)
		h.SetSelector(config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"team": "alerts"})})
		h.Reconcile("reconcile after the selector stops matching the CR: delete the rule")
		h.Reconcile("reconcile again")
	},
}

// alertSchedulerAdmission lists the CRs that TestAdmission creates. Each case has one reason that
// the CRD accepts or rejects it.
func alertSchedulerAdmission() map[string]string {
	spec := func(fields string) string { return alertSchedulerManifest(fields) }
	withSchedule := func(schedule string) string { return spec(asWhat + asLabelsFilter + "  schedule:\n" + schedule) }
	oneTime := func(timeFrame string) string {
		return withSchedule("    operation: mute\n    oneTime:\n" + timeFrame)
	}
	dynamic := func(body string) string {
		return withSchedule("    operation: mute\n    recurring:\n      dynamic:\n" + body)
	}
	daily := "        frequency:\n          daily: {}\n"
	dynTF := "        timeFrame:\n          startTime: 2027-01-04T08:00:00.000\n          timezone: UTC+2\n          endTime: 2027-01-04T09:00:00.000\n"
	return map[string]string{
		"minimal":                               asMinimal,
		"no spec":                               "apiVersion: coralogix.com/v1alpha1\nkind: AlertScheduler\nmetadata:\n  name: scheduler\n",
		"empty spec":                            spec(" {}\n"),
		"without name":                          spec(asLabelsFilter + asOneTime),
		"empty name":                            spec("  name: \"\"\n" + asLabelsFilter + asOneTime),
		"without filter":                        spec(asWhat + asOneTime),
		"without schedule":                      spec(asWhat + asLabelsFilter),
		"filter without expression":             spec(asWhat + "  filter:\n    metaLabels: []\n" + asOneTime),
		"filter without labels or alerts":       spec(asWhat + "  filter:\n    whatExpression: source logs | filter true\n" + asOneTime),
		"filter with labels and alerts":         spec(asWhat + "  filter:\n    whatExpression: source logs | filter true\n    metaLabels: []\n    alerts: []\n" + asOneTime),
		"alert without resourceRef":             spec(asWhat + asAlertsFilter("      - {}\n") + asOneTime),
		"alert ref without name":                spec(asWhat + asAlertsFilter("      - resourceRef: {}\n") + asOneTime),
		"meta label without key":                spec(asWhat + "  metaLabels:\n    - value: v\n" + asLabelsFilter + asOneTime),
		"enabled is not a boolean":              spec(asWhat + "  enabled: \"yes\"\n" + asLabelsFilter + asOneTime),
		"operation activate":                    withSchedule("    operation: activate\n    recurring:\n      always: {}\n"),
		"operation API name":                    withSchedule("    operation: SCHEDULE_OPERATION_MUTE\n    recurring:\n      always: {}\n"),
		"operation uppercase":                   withSchedule("    operation: Mute\n    recurring:\n      always: {}\n"),
		"without operation":                     withSchedule("    recurring:\n      always: {}\n"),
		"schedule without oneTime or recurring": withSchedule("    operation: mute\n"),
		"schedule with oneTime and recurring": withSchedule("    operation: mute\n    oneTime:\n      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00.000\n" +
			"    recurring:\n      always: {}\n"),
		"one-time without startTime":        oneTime("      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00.000\n"),
		"one-time without timezone":         oneTime("      startTime: 2027-01-04T00:00:00.000\n      endTime: 2027-01-05T00:00:00.000\n"),
		"one-time without end or duration":  oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n"),
		"one-time with end and duration":    oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00.000\n      duration:\n        forOver: 1\n        frequency: days\n"),
		"startTime without milliseconds":    oneTime("      startTime: 2027-01-04T00:00:00\n      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00.000\n"),
		"startTime with a zone":             oneTime("      startTime: 2027-01-04T00:00:00.000Z\n      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00.000\n"),
		"startTime date only":               oneTime("      startTime: 2027-01-04\n      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00.000\n"),
		"endTime without milliseconds":      oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      endTime: 2027-01-05T00:00:00\n"),
		"timezone UTC+10":                   oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+10\n      endTime: 2027-01-05T00:00:00.000\n"),
		"timezone UTC+100":                  oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+100\n      endTime: 2027-01-05T00:00:00.000\n"),
		"timezone UTC":                      oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC\n      endTime: 2027-01-05T00:00:00.000\n"),
		"timezone with minutes":             oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+05:30\n      endTime: 2027-01-05T00:00:00.000\n"),
		"timezone IANA name":                oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: Europe/Berlin\n      endTime: 2027-01-05T00:00:00.000\n"),
		"duration without forOver":          oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        frequency: days\n"),
		"duration without frequency":        oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        forOver: 1\n"),
		"duration frequency weeks":          oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        forOver: 1\n        frequency: weeks\n"),
		"duration frequency API name":       oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        forOver: 1\n        frequency: DURATION_FREQUENCY_DAY\n"),
		"duration forOver negative":         oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        forOver: -1\n        frequency: days\n"),
		"duration forOver above int32":      oneTime("      startTime: 2027-01-04T00:00:00.000\n      timezone: UTC+0\n      duration:\n        forOver: 3000000000\n        frequency: days\n"),
		"recurring always":                  withSchedule("    operation: mute\n    recurring:\n      always: {}\n"),
		"recurring empty":                   withSchedule("    operation: mute\n    recurring: {}\n"),
		"recurring always and dynamic":      withSchedule("    operation: mute\n    recurring:\n      always: {}\n      dynamic:\n        repeatEvery: 1\n" + daily + dynTF),
		"dynamic daily":                     dynamic("        repeatEvery: 1\n" + daily + dynTF),
		"dynamic without repeatEvery":       dynamic(daily + dynTF),
		"dynamic repeatEvery zero":          dynamic("        repeatEvery: 0\n" + daily + dynTF),
		"dynamic repeatEvery negative":      dynamic("        repeatEvery: -1\n" + daily + dynTF),
		"dynamic without frequency":         dynamic("        repeatEvery: 1\n" + dynTF),
		"dynamic empty frequency":           dynamic("        repeatEvery: 1\n        frequency: {}\n" + dynTF),
		"dynamic daily and weekly":          dynamic("        repeatEvery: 1\n        frequency:\n          daily: {}\n          weekly:\n            days: [Monday]\n" + dynTF),
		"dynamic without timeFrame":         dynamic("        repeatEvery: 1\n" + daily),
		"dynamic terminationDate":           dynamic("        repeatEvery: 1\n        terminationDate: 2028-01-01T00:00:00.000\n" + daily + dynTF),
		"dynamic terminationDate date only": dynamic("        repeatEvery: 1\n        terminationDate: 2028-01-01\n" + daily + dynTF),
		"weekly every day":                  dynamic("        repeatEvery: 1\n        frequency:\n          weekly:\n            days: [Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday]\n" + dynTF),
		"weekly lowercase day":              dynamic("        repeatEvery: 1\n        frequency:\n          weekly:\n            days: [monday]\n" + dynTF),
		"weekly day number":                 dynamic("        repeatEvery: 1\n        frequency:\n          weekly:\n            days: [2]\n" + dynTF),
		"weekly empty day":                  dynamic("        repeatEvery: 1\n        frequency:\n          weekly:\n            days: [\"\"]\n" + dynTF),
		"weekly no days":                    dynamic("        repeatEvery: 1\n        frequency:\n          weekly:\n            days: []\n" + dynTF),
		"weekly without days":               dynamic("        repeatEvery: 1\n        frequency:\n          weekly: {}\n" + dynTF),
		"weekly duplicate day":              dynamic("        repeatEvery: 1\n        frequency:\n          weekly:\n            days: [Monday, Monday]\n" + dynTF),
		"monthly days":                      dynamic("        repeatEvery: 1\n        frequency:\n          monthly:\n            days: [1, 31]\n" + dynTF),
		"monthly day 0 and 32":              dynamic("        repeatEvery: 1\n        frequency:\n          monthly:\n            days: [0, 32]\n" + dynTF),
		"monthly no days":                   dynamic("        repeatEvery: 1\n        frequency:\n          monthly:\n            days: []\n" + dynTF),
		"monthly without days":              dynamic("        repeatEvery: 1\n        frequency:\n          monthly: {}\n" + dynTF),
		"unknown field is not an error":     spec(asWhat + "  folder: x\n" + asLabelsFilter + asOneTime),
	}
}
