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
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
)

// TCOLogsPoliciesSpec defines the desired state of Coralogix TCO logs policies.
type TCOLogsPoliciesSpec struct {
	// Coralogix TCO-Policies-List.
	Policies []TCOLogsPolicy `json:"policies"`
}

// A TCO policy for logs.
type TCOLogsPolicy struct {
	// Name of the policy.
	Name string `json:"name"`

	// Description of the policy.
	// +optional
	Description *string `json:"description,omitempty"`

	// +kubebuilder:validation:Enum=block;high;medium;low
	// The policy priority.
	Priority string `json:"priority"`

	// The severities to apply the policy on.
	Severities []TCOPolicySeverity `json:"severities"`

	// Matches the specified retention.
	// +optional
	ArchiveRetention *ArchiveRetention `json:"archiveRetention,omitempty"`

	// The applications to apply the policy on. Applies the policy on all the applications by default.
	// +optional
	Applications *TCOPolicyRule `json:"applications,omitempty"`

	// The subsystems to apply the policy on. Applies the policy on all the subsystems by default.
	// +optional
	Subsystems *TCOPolicyRule `json:"subsystems,omitempty"`
}

// Matches the specified retention.
type ArchiveRetention struct {
	// Reference to the retention policy
	BackendRef ArchiveRetentionBackendRef `json:"backendRef"`
}

// Backend reference to the policy.
type ArchiveRetentionBackendRef struct {
	// Name of the policy.
	Name string `json:"name"`
}

// +kubebuilder:validation:Enum=info;warning;critical;error;debug;verbose
// The severities to apply the policy on.
type TCOPolicySeverity string

// A sincle TCO policy rule.
type TCOPolicyRule struct {
	// Names to match.
	Names []string `json:"names"`

	// +kubebuilder:validation:Enum=is;is_not;start_with;includes
	// Type of matching for the name.
	RuleType string `json:"ruleType"`
}

func (s *TCOLogsPoliciesSpec) ExtractOverwriteLogPoliciesRequest(ctx context.Context, coralogixClientSet *cxsdk.ClientSet) (*cxsdk.AtomicOverwriteLogPoliciesRequest, error) {
	var policies []*cxsdk.CreateLogPolicyRequest
	var errs error

	for _, policy := range s.Policies {
		policyReq, err := policy.ExtractCreateLogPolicyRequest(ctx, coralogixClientSet)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			policies = append(policies, policyReq)
		}
	}

	if errs != nil {
		return nil, errs
	}

	return &cxsdk.AtomicOverwriteLogPoliciesRequest{Policies: policies}, nil
}

func (p *TCOLogsPolicy) ExtractCreateLogPolicyRequest(ctx context.Context, coralogixClientSet *cxsdk.ClientSet) (*cxsdk.CreateLogPolicyRequest, error) {
	var errs error
	priority, err := expandTCOPolicyPriority(p.Priority)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	applicationRule, err := expandTCOPolicyRule(p.Applications)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	subsystemRule, err := expandTCOPolicyRule(p.Subsystems)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	severities, err := expandTCOPolicySeverities(p.Severities)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	archiveRetentionID, err := expandArchiveRetention(ctx, coralogixClientSet, p.ArchiveRetention)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return nil, errs
	}

	req := &cxsdk.CreateLogPolicyRequest{
		Policy: &cxsdk.CreateGenericPolicyRequest{
			Name:             wrapperspb.String(p.Name),
			Description:      utils.StringPointerToWrapperspbString(p.Description),
			Priority:         priority,
			ApplicationRule:  applicationRule,
			SubsystemRule:    subsystemRule,
			ArchiveRetention: archiveRetentionID,
		},
		LogRules: &cxsdk.TCOLogRules{
			Severities: severities,
		},
	}

	return req, nil
}

func expandTCOPolicyPriority(priority string) (cxsdk.TCOPolicyPriority, error) {
	priorityValue, ok := cxsdk.TCOPolicyPriorityValueLookup["PRIORITY_TYPE_"+strings.ToUpper(priority)]
	if !ok {
		return 0, fmt.Errorf("invalid priority for TCO policy: %s", priority)
	}
	return cxsdk.TCOPolicyPriority(priorityValue), nil
}

func expandTCOPolicyRule(rule *TCOPolicyRule) (*cxsdk.TCOPolicyRule, error) {
	if rule == nil {
		return nil, nil
	}

	ruleType, ok := cxsdk.TCOPolicyRuleTypeValueLookup["RULE_TYPE_ID_"+strings.ToUpper(rule.RuleType)]
	if !ok {
		return nil, fmt.Errorf("invalid rule type for TCO policy: %s", rule.RuleType)
	}

	return &cxsdk.TCOPolicyRule{
		Name:       wrapperspb.String(strings.Join(rule.Names, ",")),
		RuleTypeId: cxsdk.TCOPolicyRuleTypeID(ruleType),
	}, nil
}

func expandTCOPolicySeverities(severities []TCOPolicySeverity) ([]cxsdk.TCOPolicySeverity, error) {
	var result []cxsdk.TCOPolicySeverity
	for _, severity := range severities {
		severityValue, ok := cxsdk.TCOPolicySeverityValueLookup["SEVERITY_"+strings.ToUpper(string(severity))]
		if !ok {
			return nil, fmt.Errorf("invalid severity for TCO policy: %s", severity)
		}
		result = append(result, cxsdk.TCOPolicySeverity(severityValue))
	}

	return result, nil
}

func expandArchiveRetention(ctx context.Context, coralogixClientSet *cxsdk.ClientSet, archiveRetention *ArchiveRetention) (*cxsdk.ArchiveRetention, error) {
	if archiveRetention == nil {
		return nil, nil
	}

	archiveRetentions, err := coralogixClientSet.ArchiveRetentions().Get(ctx, &cxsdk.GetRetentionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get archive retentions: %w", err)
	}

	for _, retention := range archiveRetentions.Retentions {
		if *utils.WrapperspbStringToStringPointer(retention.Name) == archiveRetention.BackendRef.Name {
			return &cxsdk.ArchiveRetention{Id: retention.Id}, nil
		}
	}

	return nil, fmt.Errorf("archive retention with name %s not found", archiveRetention.BackendRef.Name)
}

// TCOLogsPoliciesStatus defines the observed state of TCOLogsPolicies.
type TCOLogsPoliciesStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (t *TCOLogsPolicies) GetConditions() []metav1.Condition {
	return t.Status.Conditions
}

func (t *TCOLogsPolicies) SetConditions(conditions []metav1.Condition) {
	t.Status.Conditions = conditions
}

func (t *TCOLogsPolicies) HasIDInStatus() bool {
	return true
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// TCOLogsPolicies is the Schema for the TCOLogsPolicies API.
// NOTE: This resource performs an atomic overwrite of all existing TCO logs policies
// in the backend. Any existing policies not defined in this resource will be
// removed. Use with caution as this operation is destructive.
//
// See also https://coralogix.com/docs/tco-optimizer-api
//
// **Added in v0.4.0**
type TCOLogsPolicies struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TCOLogsPoliciesSpec   `json:"spec,omitempty"`
	Status TCOLogsPoliciesStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// TCOLogsPoliciesList contains a list of TCOLogsPolicies.
type TCOLogsPoliciesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TCOLogsPolicies `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TCOLogsPolicies{}, &TCOLogsPoliciesList{})
}
