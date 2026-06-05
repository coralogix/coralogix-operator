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
	"fmt"

	quotas "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/quota_allocation_rule_set_service"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QuotaAllocationRuleSetSpec defines the desired state of Coralogix quota allocation rules.
type QuotaAllocationRuleSetSpec struct {
	// Coralogix quota allocation rules.
	// +kubebuilder:validation:MinItems=1
	// +listType=map
	// +listMapKey=entityType
	Rules []QuotaAllocationRule `json:"rules"`
}

// QuotaAllocationRule defines quota allocation for a single entity type.
type QuotaAllocationRule struct {
	// Entity type to allocate quota for.
	// +kubebuilder:validation:MinLength=1
	EntityType string `json:"entityType"`

	// Allocation value. Percent allocations are 0-100; locked unit allocations are absolute units.
	// Fractional values must be supplied as quoted quantities, for example "12.5".
	Allocation resource.Quantity `json:"allocation"`

	// Interprets allocation as a percentage, locked units, or unspecified.
	// Defaults to percentage when omitted.
	// +optional
	AllocationType *QuotaAllocationType `json:"allocationType,omitempty"`

	// Whether this quota allocation rule is enabled.
	Enabled bool `json:"enabled"`

	// Whether this quota allocation rule can overflow.
	CanOverflow bool `json:"canOverflow"`
}

// +kubebuilder:validation:Enum=percentage;lockedUnits;unspecified
// QuotaAllocationType specifies how allocation values are interpreted.
type QuotaAllocationType string

const (
	QuotaAllocationTypePercentage  QuotaAllocationType = "percentage"
	QuotaAllocationTypeLockedUnits QuotaAllocationType = "lockedUnits"
	QuotaAllocationTypeUnspecified QuotaAllocationType = "unspecified"
)

var quotaAllocationTypeSchemaToOpenAPI = map[QuotaAllocationType]quotas.QuotaAllocationType{
	QuotaAllocationTypePercentage:  quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_PERCENTAGE,
	QuotaAllocationTypeLockedUnits: quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_LOCKED_UNITS,
	QuotaAllocationTypeUnspecified: quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_UNSPECIFIED,
}

var maxQuotaAllocationPercentage = resource.MustParse("100")

// ExtractQuotaAllocationRuleSetRequest converts the Kubernetes spec to the OpenAPI quota rule set model.
func (s *QuotaAllocationRuleSetSpec) ExtractQuotaAllocationRuleSetRequest() (*quotas.QuotaAllocationEntityTypeRuleSet, error) {
	rules, err := s.ExtractQuotaAllocationRules()
	if err != nil {
		return nil, err
	}

	return &quotas.QuotaAllocationEntityTypeRuleSet{Rules: rules}, nil
}

// ExtractQuotaAllocationRules converts Kubernetes quota rules to the OpenAPI quota rule model.
func (s *QuotaAllocationRuleSetSpec) ExtractQuotaAllocationRules() ([]quotas.QuotaAllocationEntityTypeRule, error) {
	seenEntityTypes := make(map[string]struct{}, len(s.Rules))
	rules := make([]quotas.QuotaAllocationEntityTypeRule, 0, len(s.Rules))

	for _, rule := range s.Rules {
		if _, exists := seenEntityTypes[rule.EntityType]; exists {
			return nil, fmt.Errorf("duplicate quota allocation rule entityType %q", rule.EntityType)
		}
		if rule.Allocation.Sign() < 0 {
			return nil, fmt.Errorf("quota allocation rule entityType %q has negative allocation", rule.EntityType)
		}
		if (rule.AllocationType == nil || *rule.AllocationType == QuotaAllocationTypePercentage || *rule.AllocationType == QuotaAllocationTypeUnspecified) &&
			rule.Allocation.Cmp(maxQuotaAllocationPercentage) > 0 {
			return nil, fmt.Errorf("quota allocation rule entityType %q has percentage allocation greater than 100", rule.EntityType)
		}
		seenEntityTypes[rule.EntityType] = struct{}{}
		rules = append(rules, rule.ExtractQuotaAllocationEntityTypeRule())
	}

	return rules, nil
}

// ExtractQuotaAllocationEntityTypeRule converts a Kubernetes quota rule to the OpenAPI quota rule model.
func (r *QuotaAllocationRule) ExtractQuotaAllocationEntityTypeRule() quotas.QuotaAllocationEntityTypeRule {
	allocationType := QuotaAllocationTypePercentage
	if r.AllocationType != nil {
		allocationType = *r.AllocationType
	}
	openAPIAllocationType := quotaAllocationTypeSchemaToOpenAPI[allocationType]

	return quotas.QuotaAllocationEntityTypeRule{
		EntityType:     r.EntityType,
		Allocation:     float32(r.Allocation.AsApproximateFloat64()),
		AllocationType: openAPIAllocationType.Ptr(),
		Enabled:        r.Enabled,
		CanOverflow:    r.CanOverflow,
		CxManaged:      nil,
	}
}

// QuotaAllocationRuleSetStatus defines the observed state of QuotaAllocationRuleSet.
type QuotaAllocationRuleSetStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (q *QuotaAllocationRuleSet) GetConditions() []metav1.Condition {
	return q.Status.Conditions
}

func (q *QuotaAllocationRuleSet) SetConditions(conditions []metav1.Condition) {
	q.Status.Conditions = conditions
}

func (q *QuotaAllocationRuleSet) GetPrintableStatus() string {
	return q.Status.PrintableStatus
}

func (q *QuotaAllocationRuleSet) SetPrintableStatus(printableStatus string) {
	q.Status.PrintableStatus = printableStatus
}

func (q *QuotaAllocationRuleSet) HasIDInStatus() bool {
	return true
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// QuotaAllocationRuleSet is the Schema for the QuotaAllocationRuleSet API.
// NOTE: This account-level singleton resource replaces all user-managed backend
// quota allocation rules. Coralogix-managed rules returned by the backend are
// preserved by the controller.
//
// **Added in v0.4.0**
type QuotaAllocationRuleSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuotaAllocationRuleSetSpec   `json:"spec,omitempty"`
	Status QuotaAllocationRuleSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// QuotaAllocationRuleSetList contains a list of QuotaAllocationRuleSet.
type QuotaAllocationRuleSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuotaAllocationRuleSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QuotaAllocationRuleSet{}, &QuotaAllocationRuleSetList{})
}
