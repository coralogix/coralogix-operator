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
)

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

	req := &tcopolicies.CreateLogPolicyRequest{
		Policy: tcopolicies.CreateGenericPolicyRequest{
			Name:             p.Name,
			Description:      ptr.Deref(p.Description, ""),
			Priority:         PrioritySchemaToOpenAPI[p.Priority],
			Disabled:         p.Disabled,
			ApplicationRule:  expandTCOPolicyRule(p.Applications),
			SubsystemRule:    expandTCOPolicyRule(p.Subsystems),
			ArchiveRetention: archiveRetention,
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
