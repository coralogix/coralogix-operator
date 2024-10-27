/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RecordingRuleGroupSetSpec defines the desired state of RecordingRuleGroupSet
type RecordingRuleGroupSetSpec struct {
	// +kubebuilder:validation:MinItems=1
	Groups []RecordingRuleGroup `json:"groups"`
}

func (in *RecordingRuleGroupSetSpec) ExtractRecordingRuleGroups() []*cxsdk.InRuleGroup {
	result := make([]*cxsdk.InRuleGroup, 0, len(in.Groups))
	for _, ruleGroup := range in.Groups {
		rg := expandRecordingRuleGroup(ruleGroup)
		result = append(result, rg)
	}
	return result
}

func expandRecordingRuleGroup(group RecordingRuleGroup) *cxsdk.InRuleGroup {
	interval := new(uint32)
	*interval = uint32(group.IntervalSeconds)

	limit := new(uint64)
	*limit = uint64(group.Limit)

	rules := expandRecordingRules(group.Rules)

	return &cxsdk.InRuleGroup{
		Name:     group.Name,
		Interval: interval,
		Limit:    limit,
		Rules:    rules,
	}
}

func expandRecordingRules(rules []RecordingRule) []*cxsdk.InRule {
	result := make([]*cxsdk.InRule, 0, len(rules))
	for _, r := range rules {
		rule := extractRecordingRule(r)
		result = append(result, rule)
	}
	return result
}

func extractRecordingRule(rule RecordingRule) *cxsdk.InRule {
	return &cxsdk.InRule{
		Record: rule.Record,
		Expr:   rule.Expr,
		Labels: rule.Labels,
	}
}

func (in *RecordingRuleGroup) DeepEqual(actual RecordingRuleGroup) (bool, utils.Diff) {
	if limit, actualLimit := in.Limit, actual.Limit; limit != actualLimit {
		return false, utils.Diff{
			Name:    "Limit",
			Desired: limit,
			Actual:  actualLimit,
		}
	}

	if name, actualName := in.Name, actual.Name; name != actualName {
		return false, utils.Diff{
			Name:    "Name",
			Desired: name,
			Actual:  actualName,
		}
	}

	if interval, actualInterval := in.IntervalSeconds, actual.IntervalSeconds; interval != actualInterval {
		return false, utils.Diff{
			Name:    "Interval",
			Desired: interval,
			Actual:  actualInterval,
		}
	}

	if equal, diff := DeepEqualRecordingRules(in.Rules, actual.Rules); !equal {
		return false, diff
	}

	return true, utils.Diff{}
}

func DeepEqualRecordingRules(desiredRules, actualRule []RecordingRule) (bool, utils.Diff) {
	if length, actualLength := len(desiredRules), len(actualRule); length != actualLength {
		return false, utils.Diff{
			Name:    "Rules.length",
			Desired: length,
			Actual:  actualLength,
		}
	}

	for i := range desiredRules {
		if equal, diff := desiredRules[i].DeepEqual(actualRule[i]); !equal {
			return false, utils.Diff{
				Name:    fmt.Sprintf("Rules.%d.%s", i, diff.Name),
				Desired: diff.Desired,
				Actual:  diff.Actual,
			}
		}
	}

	return true, utils.Diff{}
}

func (in *RecordingRule) DeepEqual(rule RecordingRule) (bool, utils.Diff) {
	if expr, actualExpr := in.Expr, rule.Expr; expr != actualExpr {
		return false, utils.Diff{
			Name:    "Expr",
			Desired: expr,
			Actual:  actualExpr,
		}
	}

	if record, actualRecord := in.Record, rule.Record; record != actualRecord {
		return false, utils.Diff{
			Name:    "Record",
			Desired: record,
			Actual:  actualRecord,
		}
	}

	if labels, actualLabels := in.Labels, rule.Labels; !reflect.DeepEqual(labels, actualLabels) {
		return false, utils.Diff{
			Name:    "Labels",
			Desired: labels,
			Actual:  actualLabels,
		}
	}

	return true, utils.Diff{}
}

type RecordingRuleGroup struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name string `json:"name,omitempty"`

	//+kubebuilder:default=60
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`

	// +optional
	Limit int64 `json:"limit,omitempty"`

	Rules []RecordingRule `json:"rules,omitempty"`
}

type RecordingRule struct {
	Record string `json:"record,omitempty"`

	Expr string `json:"expr,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`
}

// RecordingRuleGroupSetStatus defines the observed state of RecordingRuleGroupSet
type RecordingRuleGroupSetStatus struct {
	ID *string `json:"id"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// RecordingRuleGroupSet is the Schema for the recordingrulegroupsets API
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
