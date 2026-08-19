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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"
	archiveretentions "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/retentions_service"
)

// TCORumPoliciesSpec defines the desired state of Coralogix TCO RUM policies.
type TCORumPoliciesSpec struct {
	// Coralogix TCO-Policies-List.
	// +kubebuilder:validation:MaxItems=10000
	Policies []TCORumPolicy `json:"policies"`
}

// A TCO policy for RUM (browser/mobile) events.
// +kubebuilder:validation:XValidation:rule="!(has(self.severities) && has(self.dpxlExpression))",message="severities and dpxlExpression are mutually exclusive"
type TCORumPolicy struct {
	// Name of the policy.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
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

	// The severities to apply the policy on. Mutually exclusive with dpxlExpression; exactly one of the two is required.
	// +optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=6
	Severities []TCOPolicySeverity `json:"severities,omitempty"`

	// A DPXL expression to match RUM events on. Mutually exclusive with severities; exactly one of the two is required.
	// The expression must carry a <v1> version prefix and use the canonical $d.* schema (NOT $d.cx_rum.*),
	// otherwise it fails to compile.
	// +optional
	// +kubebuilder:validation:MinLength=1
	DpxlExpression *string `json:"dpxlExpression,omitempty"`

	// Dynamic quota-based priority override for the policy.
	// +optional
	PriorityOverride *TCOPolicyPriorityOverride `json:"priorityOverride,omitempty"`

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

func (s *TCORumPoliciesSpec) ExtractOverwriteRumPoliciesRequest(
	ctx context.Context,
	archiveRetentionsClient *archiveretentions.RetentionsServiceAPIService) (*tcopolicies.AtomicOverwriteRumPoliciesRequest, error) {
	var retentionsByName map[string]string
	if s.referencesArchiveRetention() {
		var err error
		retentionsByName, err = fetchRetentionsByName(ctx, archiveRetentionsClient)
		if err != nil {
			return nil, err
		}
	}

	var policies []tcopolicies.CreateRumPolicyRequest
	var errs error

	for _, policy := range s.Policies {
		policyReq, err := policy.extractCreateRumPolicyRequest(retentionsByName)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			policies = append(policies, *policyReq)
		}
	}

	if errs != nil {
		return nil, errs
	}

	return &tcopolicies.AtomicOverwriteRumPoliciesRequest{Policies: policies}, nil
}

func (s *TCORumPoliciesSpec) referencesArchiveRetention() bool {
	for _, p := range s.Policies {
		if p.ArchiveRetention != nil {
			return true
		}
	}
	return false
}

func (p *TCORumPolicy) extractCreateRumPolicyRequest(retentionsByName map[string]string) (*tcopolicies.CreateRumPolicyRequest, error) {
	archiveRetention, err := expandArchiveRetention(retentionsByName, p.ArchiveRetention)
	if err != nil {
		return nil, err
	}

	req := &tcopolicies.CreateRumPolicyRequest{
		Policy: tcopolicies.CreateGenericPolicyRequest{
			Name:             p.Name,
			Description:      ptr.Deref(p.Description, ""),
			Priority:         PrioritySchemaToOpenAPI[p.Priority],
			Disabled:         p.Disabled,
			ApplicationRule:  expandTCOPolicyRule(p.Applications),
			SubsystemRule:    expandTCOPolicyRule(p.Subsystems),
			ArchiveRetention: archiveRetention,
			PriorityOverride: expandTCOPolicyPriorityOverride(p.PriorityOverride),
		},
		RumRules: tcopolicies.LogRules{
			Severities:     expandTCOPolicySeverities(p.Severities),
			DpxlExpression: p.DpxlExpression,
		},
	}

	return req, nil
}

// TCORumPoliciesStatus defines the observed state of TCORumPolicies.
type TCORumPoliciesStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (t *TCORumPolicies) GetConditions() []metav1.Condition {
	return t.Status.Conditions
}

func (t *TCORumPolicies) SetConditions(conditions []metav1.Condition) {
	t.Status.Conditions = conditions
}

func (t *TCORumPolicies) GetPrintableStatus() string {
	return t.Status.PrintableStatus
}

func (t *TCORumPolicies) SetPrintableStatus(printableStatus string) {
	t.Status.PrintableStatus = printableStatus
}

func (t *TCORumPolicies) HasIDInStatus() bool {
	return true
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// TCORumPolicies is the Schema for the TCORumPolicies API.
// NOTE: This resource performs an atomic overwrite of all existing TCO RUM policies
// in the backend. Any existing policies not defined in this resource will be
// removed. Use with caution as this operation is destructive.
//
// See also https://coralogix.com/docs/tco-optimizer-api
//
// **Added in v0.5.0**
type TCORumPolicies struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TCORumPoliciesSpec   `json:"spec,omitempty"`
	Status TCORumPoliciesStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// TCORumPoliciesList contains a list of TCORumPolicies.
type TCORumPoliciesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TCORumPolicies `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TCORumPolicies{}, &TCORumPoliciesList{})
}
