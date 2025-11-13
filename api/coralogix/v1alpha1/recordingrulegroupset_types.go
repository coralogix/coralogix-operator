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

package v1alpha1

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	recordingrules "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/recording_rules_service"
)

// RecordingRuleGroupSetSpec defines the desired state of a set of Coralogix recording rule groups.
type RecordingRuleGroupSetSpec struct {
	// +kubebuilder:validation:MinItems=1
	// Recording rule groups.
	Groups []RecordingRuleGroup `json:"groups"`
}

func (in *RecordingRuleGroupSetSpec) ExtractRecordingRuleGroups() []recordingrules.InRuleGroup {
	result := make([]recordingrules.InRuleGroup, 0, len(in.Groups))
	for _, ruleGroup := range in.Groups {
		rg := expandRecordingRuleGroup(ruleGroup)
		result = append(result, *rg)
	}
	return result
}

func expandRecordingRuleGroup(group RecordingRuleGroup) *recordingrules.InRuleGroup {
	interval := new(int64)
	*interval = int64(group.IntervalSeconds)

	limit := new(string)
	*limit = strconv.FormatInt(group.Limit, 10)

	rules := expandRecordingRules(group.Rules)

	return &recordingrules.InRuleGroup{
		Name:     recordingrules.PtrString(group.Name),
		Interval: interval,
		Limit:    limit,
		Rules:    rules,
	}
}

func expandRecordingRules(rules []RecordingRule) []recordingrules.InRule {
	result := make([]recordingrules.InRule, 0, len(rules))
	for _, r := range rules {
		rule := extractRecordingRule(r)
		result = append(result, *rule)
	}
	return result
}

func extractRecordingRule(rule RecordingRule) *recordingrules.InRule {
	return &recordingrules.InRule{
		Record: recordingrules.PtrString(rule.Record),
		Expr:   recordingrules.PtrString(rule.Expr),
		Labels: ptr.To(rule.Labels),
	}
}

// A Coralogix recording rule group.
type RecordingRuleGroup struct {

	// The (unique) rule group name.
	Name string `json:"name,omitempty"`

	// How often rules in the group are evaluated (in seconds).
	//+kubebuilder:default=60
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`

	// Limits the number of alerts an alerting rule and series a recording-rule can produce. 0 is no limit.
	// +optional
	Limit int64 `json:"limit,omitempty"`

	// Rules of this group.
	Rules []RecordingRule `json:"rules,omitempty"`
}

// A recording rule.
type RecordingRule struct {

	// The name of the time series to output to. Must be a valid metric name.
	Record string `json:"record,omitempty"`

	// The PromQL expression to evaluate.
	// Every evaluation cycle this is evaluated at the current time, and the result recorded as a new set of time series with the metric name as given by 'record'.
	Expr string `json:"expr,omitempty"`

	// Labels to add or overwrite before storing the result.
	Labels map[string]string `json:"labels,omitempty"`
}

// RecordingRuleGroupSetStatus defines the observed state of RecordingRuleGroupSet
type RecordingRuleGroupSetStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (r *RecordingRuleGroupSet) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *RecordingRuleGroupSet) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

func (r *RecordingRuleGroupSet) GetPrintableStatus() string {
	return r.Status.PrintableStatus
}

func (r *RecordingRuleGroupSet) SetPrintableStatus(printableStatus string) {
	r.Status.PrintableStatus = printableStatus
}

func (r *RecordingRuleGroupSet) HasIDInStatus() bool {
	return r.Status.ID != nil && *r.Status.ID != ""
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:storageversion

// RecordingRuleGroupSet is the Schema for the RecordingRuleGroupSets API
// See also https://coralogix.com/docs/user-guides/data-transformation/metric-rules/recording-rules/
//
// **Added in v0.4.0**
type RecordingRuleGroupSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RecordingRuleGroupSetSpec   `json:"spec,omitempty"`
	Status RecordingRuleGroupSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RecordingRuleGroupSetList contains a list of RecordingRuleGroupSet
type RecordingRuleGroupSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RecordingRuleGroupSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RecordingRuleGroupSet{}, &RecordingRuleGroupSetList{})
}
