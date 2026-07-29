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

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"
	archiveretentions "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/retentions_service"
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
	// The policy priority. Required when targets is not set, or when targets do not specify their own priorities.
	// +optional
	Priority *string `json:"priority,omitempty"`

	// +optional
	// Whether the policy is disabled.
	Disabled *bool `json:"disabled,omitempty"`

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

	// Targets defines the datasets to route matched data to, each with its own priority.
	// When set, overrides or supplements the policy-level priority.
	// +optional
	Targets []TCOPolicyTarget `json:"targets,omitempty"`
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

var (
	TCOPolicySeveritySchemaToOpenAPI = map[TCOPolicySeverity]tcopolicies.QuotaV1Severity{
		"info":     tcopolicies.QUOTAV1SEVERITY_SEVERITY_INFO,
		"warning":  tcopolicies.QUOTAV1SEVERITY_SEVERITY_WARNING,
		"critical": tcopolicies.QUOTAV1SEVERITY_SEVERITY_CRITICAL,
		"error":    tcopolicies.QUOTAV1SEVERITY_SEVERITY_ERROR,
		"debug":    tcopolicies.QUOTAV1SEVERITY_SEVERITY_DEBUG,
		"verbose":  tcopolicies.QUOTAV1SEVERITY_SEVERITY_VERBOSE,
	}
	PrioritySchemaToOpenAPI = map[string]tcopolicies.QuotaV1Priority{
		"":       tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_UNSPECIFIED,
		"block":  tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_BLOCK,
		"high":   tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_HIGH,
		"medium": tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_MEDIUM,
		"low":    tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_LOW,
	}
	RuleTypeIdSchemaToOpenAPI = map[string]tcopolicies.RuleTypeId{
		"is":         tcopolicies.RULETYPEID_RULE_TYPE_ID_IS,
		"is_not":     tcopolicies.RULETYPEID_RULE_TYPE_ID_IS_NOT,
		"start_with": tcopolicies.RULETYPEID_RULE_TYPE_ID_START_WITH,
		"includes":   tcopolicies.RULETYPEID_RULE_TYPE_ID_INCLUDES,
	}
)

// +kubebuilder:validation:Enum=info;warning;critical;error;debug;verbose
// The severities to apply the policy on.
type TCOPolicySeverity string

// TCOPolicyTarget defines a dataset destination with its own priority for matched log data.
type TCOPolicyTarget struct {
	// +kubebuilder:validation:MinLength=1
	// The dataset to route data to.
	Dataset string `json:"dataset"`

	// +optional
	// The dataspace within the dataset.
	Dataspace *string `json:"dataspace,omitempty"`

	// +kubebuilder:validation:Enum=block;high;low;medium
	// +optional
	// Per-target priority. Mutually exclusive with a policy-level priority.
	Priority *string `json:"priority,omitempty"`

	// +optional
	// Dynamic quota-based priority override for this target.
	PriorityOverride *TCOPolicyPriorityOverride `json:"priorityOverride,omitempty"`

	// +optional
	// Matches the specified archive retention for this target.
	ArchiveRetention *ArchiveRetention `json:"archiveRetention,omitempty"`
}

// TCOPolicyPriorityOverride configures dynamic quota-based priority tiers for a target.
type TCOPolicyPriorityOverride struct {
	// +optional
	QuotaBased *TCOPolicyQuotaBased `json:"quotaBased,omitempty"`
}

// TCOPolicyQuotaBased maps daily quota consumption percentages to priority levels.
type TCOPolicyQuotaBased struct {
	// +kubebuilder:validation:MinItems=1
	UsageTiers []TCOPolicyUsageTier `json:"usageTiers"`
}

// TCOPolicyUsageTier maps a daily quota threshold percentage to a priority.
type TCOPolicyUsageTier struct {
	DailyQuotaPercentage resource.Quantity `json:"dailyQuotaPercentage"`

	// +kubebuilder:validation:Enum=block;high;low;medium
	Priority string `json:"priority"`
}

// A sincle TCO policy rule.
type TCOPolicyRule struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=50
	// Names to match.
	Names []string `json:"names"`

	// +kubebuilder:validation:Enum=is;is_not;start_with;includes
	// Type of matching for the name.
	RuleType string `json:"ruleType"`
}

func (s *TCOLogsPoliciesSpec) ExtractOverwriteLogPoliciesRequest(
	ctx context.Context,
	archiveRetentionsClient *archiveretentions.RetentionsServiceAPIService) (*tcopolicies.AtomicOverwriteLogPoliciesRequest, error) {
	var retentionsByName map[string]string
	if s.referencesArchiveRetention() {
		var err error
		retentionsByName, err = fetchRetentionsByName(ctx, archiveRetentionsClient)
		if err != nil {
			return nil, err
		}
	}

	var policies []tcopolicies.CreateLogPolicyRequest
	var errs error

	for _, policy := range s.Policies {
		policyReq, err := policy.extractCreateLogPolicyRequest(retentionsByName)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			policies = append(policies, *policyReq)
		}
	}

	if errs != nil {
		return nil, errs
	}

	return &tcopolicies.AtomicOverwriteLogPoliciesRequest{Policies: policies}, nil
}

func (s *TCOLogsPoliciesSpec) referencesArchiveRetention() bool {
	for _, p := range s.Policies {
		if p.ArchiveRetention != nil {
			return true
		}
		for _, t := range p.Targets {
			if t.ArchiveRetention != nil {
				return true
			}
		}
	}
	return false
}

func fetchRetentionsByName(ctx context.Context, client *archiveretentions.RetentionsServiceAPIService) (map[string]string, error) {
	resp, httpResp, err := client.RetentionsServiceGetRetentions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get archive retentions: %w", cxsdk.NewAPIError(httpResp, err))
	}
	m := make(map[string]string, len(resp.Retentions))
	for _, r := range resp.Retentions {
		if r.Name != nil && r.Id != nil {
			m[*r.Name] = *r.Id
		}
	}
	return m, nil
}

func (p *TCOLogsPolicy) extractCreateLogPolicyRequest(retentionsByName map[string]string) (*tcopolicies.CreateLogPolicyRequest, error) {
	archiveRetention, err := expandArchiveRetention(retentionsByName, p.ArchiveRetention)
	if err != nil {
		return nil, err
	}

	targets, err := expandTCOPolicyTargets(retentionsByName, p.Targets)
	if err != nil {
		return nil, err
	}

	req := &tcopolicies.CreateLogPolicyRequest{
		Policy: tcopolicies.CreateGenericPolicyRequest{
			Name:             p.Name,
			Description:      ptr.Deref(p.Description, ""),
			Priority:         PrioritySchemaToOpenAPI[ptr.Deref(p.Priority, "")],
			Disabled:         p.Disabled,
			ApplicationRule:  expandTCOPolicyRule(p.Applications),
			SubsystemRule:    expandTCOPolicyRule(p.Subsystems),
			ArchiveRetention: archiveRetention,
			Targets:          targets,
		},
		LogRules: tcopolicies.LogRules{
			Severities: expandTCOPolicySeverities(p.Severities),
		},
	}

	return req, nil
}

func expandTCOPolicyRule(rule *TCOPolicyRule) *tcopolicies.QuotaV1Rule {
	if rule == nil {
		return nil
	}

	return &tcopolicies.QuotaV1Rule{
		Name:       tcopolicies.PtrString(strings.Join(rule.Names, ",")),
		RuleTypeId: RuleTypeIdSchemaToOpenAPI[rule.RuleType].Ptr(),
	}
}

func expandTCOPolicySeverities(severities []TCOPolicySeverity) []tcopolicies.QuotaV1Severity {
	var result []tcopolicies.QuotaV1Severity
	for _, severity := range severities {
		result = append(result, TCOPolicySeveritySchemaToOpenAPI[severity])
	}

	return result
}

func expandArchiveRetention(retentionsByName map[string]string, archiveRetention *ArchiveRetention) (*tcopolicies.ArchiveRetention, error) {
	if archiveRetention == nil {
		return nil, nil
	}
	id, ok := retentionsByName[archiveRetention.BackendRef.Name]
	if !ok {
		return nil, fmt.Errorf("archive retention with name %s not found", archiveRetention.BackendRef.Name)
	}
	return &tcopolicies.ArchiveRetention{Id: &id}, nil
}

func expandTCOPolicyTargets(retentionsByName map[string]string, targets []TCOPolicyTarget) ([]tcopolicies.V1Target, error) {
	if len(targets) == 0 {
		return nil, nil
	}
	var result []tcopolicies.V1Target
	var errs error
	for _, target := range targets {
		archiveRetention, err := expandArchiveRetention(retentionsByName, target.ArchiveRetention)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		t := tcopolicies.V1Target{
			Dataset:          &target.Dataset,
			Dataspace:        target.Dataspace,
			ArchiveRetention: archiveRetention,
		}
		if target.Priority != nil {
			priority := PrioritySchemaToOpenAPI[*target.Priority]
			t.Priority = &priority
		}
		if target.PriorityOverride != nil {
			t.PriorityOverride = expandTCOPolicyPriorityOverride(target.PriorityOverride)
		}
		result = append(result, t)
	}
	return result, errs
}

func expandTCOPolicyPriorityOverride(o *TCOPolicyPriorityOverride) *tcopolicies.PriorityOverride {
	if o == nil {
		return nil
	}
	result := &tcopolicies.PriorityOverride{}
	if o.QuotaBased != nil {
		qb := &tcopolicies.QuotaBased{}
		for _, tier := range o.QuotaBased.UsageTiers {
			p := PrioritySchemaToOpenAPI[tier.Priority]
			pct := tier.DailyQuotaPercentage.AsApproximateFloat64()
			qb.UsageTiers = append(qb.UsageTiers, tcopolicies.UsageTier{
				DailyQuotaPercentage: &pct,
				Priority:             &p,
			})
		}
		result.QuotaBased = qb
	}
	return result
}

// TCOLogsPoliciesStatus defines the observed state of TCOLogsPolicies.
type TCOLogsPoliciesStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (t *TCOLogsPolicies) GetConditions() []metav1.Condition {
	return t.Status.Conditions
}

func (t *TCOLogsPolicies) SetConditions(conditions []metav1.Condition) {
	t.Status.Conditions = conditions
}

func (t *TCOLogsPolicies) GetPrintableStatus() string {
	return t.Status.PrintableStatus
}

func (t *TCOLogsPolicies) SetPrintableStatus(printableStatus string) {
	t.Status.PrintableStatus = printableStatus
}

func (t *TCOLogsPolicies) HasIDInStatus() bool {
	return true
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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
