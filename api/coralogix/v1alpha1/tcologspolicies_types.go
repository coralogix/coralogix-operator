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
	// +kubebuilder:validation:MaxItems=1000
	Policies []TCOLogsPolicy `json:"policies"`
}

// A TCO policy for logs.
// +kubebuilder:validation:XValidation:rule="!has(self.targets) || self.targets.all(t, (t.dataspace == 'default' && t.dataset == 'logs') || ((has(t.priority) ? !(t.priority in ['high','block']) : !(self.priority in ['high','block'])) && (!has(t.priorityOverride) || !has(t.priorityOverride.quotaBased) || !has(t.priorityOverride.quotaBased.usageTiers) || t.priorityOverride.quotaBased.usageTiers.all(u, !(u.priority in ['high','block'])))))",message="targets other than default/logs cannot use, inherit, or override to, 'high' or 'block' priority"
type TCOLogsPolicy struct {
	// Name of the policy.
	Name string `json:"name"`

	// Description of the policy.
	// +optional
	Description *string `json:"description,omitempty"`

	// +kubebuilder:validation:Enum=block;high;medium;low
	// The policy priority.
	Priority string `json:"priority"`

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

	// Routes matching logs to one or more datasets, each with its own priority and quota configuration. Policies without targets keep their single-priority behavior.
	// +kubebuilder:validation:MaxItems=20
	// +optional
	Targets []TCOPolicyTarget `json:"targets,omitempty"`

	// Overrides the policy priority based on quota consumption.
	// +optional
	PriorityOverride *TCOPriorityOverride `json:"priorityOverride,omitempty"`
}

// A dataset-routing target for a TCO logs policy. Routes matching logs to a dataset within a dataspace, with its own priority and quota configuration.
type TCOPolicyTarget struct {
	// The dataset to route matching logs to.
	Dataset string `json:"dataset"`

	// +kubebuilder:validation:Pattern=`^[A-Za-z](?:[A-Za-z0-9_]|\.[A-Za-z0-9_])*$`
	// The dataspace to route matching logs to. Currently always "default".
	Dataspace string `json:"dataspace"`

	// +kubebuilder:validation:Enum=block;high;medium;low
	// The priority for logs routed to this target.
	// +optional
	Priority *string `json:"priority,omitempty"`

	// Matches the specified retention for logs routed to this target.
	// +optional
	ArchiveRetention *ArchiveRetention `json:"archiveRetention,omitempty"`

	// Overrides this target's priority based on quota consumption.
	// +optional
	PriorityOverride *TCOPriorityOverride `json:"priorityOverride,omitempty"`
}

// A priority override for a TCO logs policy or routing target.
type TCOPriorityOverride struct {
	// Overrides the priority based on daily quota consumption.
	// +optional
	QuotaBased *TCOQuotaBased `json:"quotaBased,omitempty"`
}

// A quota-based priority override.
type TCOQuotaBased struct {
	// Ordered list of usage tiers mapping daily quota consumption percentages to priorities.
	// +kubebuilder:validation:MaxItems=10
	// +optional
	UsageTiers []TCOUsageTier `json:"usageTiers,omitempty"`
}

// A usage tier mapping a daily quota consumption percentage to a priority.
type TCOUsageTier struct {
	// The daily quota consumption percentage threshold for this tier.
	DailyQuotaPercentage resource.Quantity `json:"dailyQuotaPercentage"`

	// +kubebuilder:validation:Enum=block;high;medium;low
	// The priority to apply for this usage tier.
	Priority string `json:"priority"`
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
	// Priorities ordered from least to most restrictive, for comparing a
	// quota-based override's fallback priority against its last usage tier.
	priorityRestrictiveness = map[string]int{
		"high":   0,
		"medium": 1,
		"low":    2,
		"block":  3,
	}
)

// +kubebuilder:validation:Enum=info;warning;critical;error;debug;verbose
// The severities to apply the policy on.
type TCOPolicySeverity string

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
	var policies []tcopolicies.CreateLogPolicyRequest
	var errs error

	for _, policy := range s.Policies {
		policyReq, err := policy.ExtractCreateLogPolicyRequest(ctx, archiveRetentionsClient)
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

func (p *TCOLogsPolicy) ExtractCreateLogPolicyRequest(
	ctx context.Context,
	archiveRetentionsClient *archiveretentions.RetentionsServiceAPIService) (*tcopolicies.CreateLogPolicyRequest, error) {
	archiveRetention, err := expandArchiveRetention(ctx, archiveRetentionsClient, p.ArchiveRetention)
	if err != nil {
		return nil, err
	}

	targets, err := expandTCOPolicyTargets(ctx, archiveRetentionsClient, p.Targets, p.Priority)
	if err != nil {
		return nil, err
	}

	priorityOverride, err := expandPriorityOverride(p.PriorityOverride, p.Priority)
	if err != nil {
		return nil, err
	}

	req := &tcopolicies.CreateLogPolicyRequest{
		Policy: tcopolicies.CreateGenericPolicyRequest{
			Name:             p.Name,
			Description:      ptr.Deref(p.Description, ""),
			Priority:         PrioritySchemaToOpenAPI[p.Priority],
			Disabled:         p.Disabled,
			ApplicationRule:  expandTCOPolicyRule(p.Applications),
			SubsystemRule:    expandTCOPolicyRule(p.Subsystems),
			ArchiveRetention: archiveRetention,
			Targets:          targets,
			PriorityOverride: priorityOverride,
		},
		LogRules: tcopolicies.LogRules{
			Severities: expandTCOPolicySeverities(p.Severities),
		},
	}

	return req, nil
}

func expandTCOPolicyTargets(
	ctx context.Context,
	archiveRetentionsClient *archiveretentions.RetentionsServiceAPIService,
	targets []TCOPolicyTarget,
	policyPriority string) ([]tcopolicies.V1Target, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	result := make([]tcopolicies.V1Target, 0, len(targets))
	for _, target := range targets {
		archiveRetention, err := expandArchiveRetention(ctx, archiveRetentionsClient, target.ArchiveRetention)
		if err != nil {
			return nil, err
		}

		// A target without its own priority inherits the policy's, so that is
		// the fallback its quota-based override has to be measured against.
		priorityOverride, err := expandPriorityOverride(target.PriorityOverride, ptr.Deref(target.Priority, policyPriority))
		if err != nil {
			return nil, err
		}

		v1Target := tcopolicies.V1Target{
			Dataset:          tcopolicies.PtrString(target.Dataset),
			Dataspace:        tcopolicies.PtrString(target.Dataspace),
			ArchiveRetention: archiveRetention,
			PriorityOverride: priorityOverride,
		}
		if target.Priority != nil {
			v1Target.Priority = PrioritySchemaToOpenAPI[*target.Priority].Ptr()
		}

		result = append(result, v1Target)
	}

	return result, nil
}

func expandPriorityOverride(override *TCOPriorityOverride, fallbackPriority string) (*tcopolicies.PriorityOverride, error) {
	if override == nil {
		return nil, nil
	}

	quotaBased, err := expandQuotaBased(override.QuotaBased, fallbackPriority)
	if err != nil {
		return nil, err
	}

	return &tcopolicies.PriorityOverride{
		QuotaBased: quotaBased,
	}, nil
}

// expandQuotaBased maps the usage tiers and enforces the invariants Coralogix
// documents for them: percentages stay within 0-100 and ascend, and the priority
// they fall back to once every tier is consumed is at least as restrictive as the
// last tier. These are enforced here rather than as CRD validation rules because
// bounding a resource.Quantity needs the CEL quantity library, which is
// unavailable on the oldest Kubernetes version this operator supports, and
// ranking priorities in CEL would push the schema past its rule cost budget.
func expandQuotaBased(quotaBased *TCOQuotaBased, fallbackPriority string) (*tcopolicies.QuotaBased, error) {
	if quotaBased == nil {
		return nil, nil
	}

	var usageTiers []tcopolicies.UsageTier
	for i, tier := range quotaBased.UsageTiers {
		percentage := tier.DailyQuotaPercentage.AsApproximateFloat64()
		if percentage < 0 || percentage > 100 {
			return nil, fmt.Errorf("dailyQuotaPercentage must be between 0 and 100, got %s", tier.DailyQuotaPercentage.String())
		}

		if i > 0 {
			previous := quotaBased.UsageTiers[i-1].DailyQuotaPercentage
			if percentage <= previous.AsApproximateFloat64() {
				return nil, fmt.Errorf("usageTiers must be ordered by ascending dailyQuotaPercentage, got %s after %s",
					tier.DailyQuotaPercentage.String(), previous.String())
			}
		}

		usageTiers = append(usageTiers, tcopolicies.UsageTier{
			DailyQuotaPercentage: tcopolicies.PtrFloat64(percentage),
			Priority:             PrioritySchemaToOpenAPI[tier.Priority].Ptr(),
		})
	}

	if len(quotaBased.UsageTiers) > 0 {
		lastPriority := quotaBased.UsageTiers[len(quotaBased.UsageTiers)-1].Priority
		if priorityRestrictiveness[fallbackPriority] < priorityRestrictiveness[lastPriority] {
			return nil, fmt.Errorf("priority %q must be at least as restrictive as the last usage tier priority %q, since it is the fallback once all tiers are consumed",
				fallbackPriority, lastPriority)
		}
	}

	return &tcopolicies.QuotaBased{
		UsageTiers: usageTiers,
	}, nil
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

func expandArchiveRetention(
	ctx context.Context,
	archiveRetentionsClient *archiveretentions.RetentionsServiceAPIService,
	archiveRetention *ArchiveRetention) (*tcopolicies.ArchiveRetention, error) {
	if archiveRetention == nil {
		return nil, nil
	}

	resp, httpResp, err := archiveRetentionsClient.
		RetentionsServiceGetRetentions(ctx).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get archive retentions: %w", cxsdk.NewAPIError(httpResp, err))
	}

	for _, retention := range resp.Retentions {
		if retention.Name != nil && *retention.Name == archiveRetention.BackendRef.Name {
			return &tcopolicies.ArchiveRetention{Id: retention.Id}, nil
		}
	}

	return nil, fmt.Errorf("archive retention with name %s not found", archiveRetention.BackendRef.Name)
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
