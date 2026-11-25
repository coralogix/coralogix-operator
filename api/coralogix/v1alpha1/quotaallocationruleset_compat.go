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

package v1alpha1

import (
	"context"
	"fmt"

	quotas "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/quota_allocation_rule_set_service"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Keep the names used by Go clients of the released kind.
type QuotaAllocationRule = QuotaAllocationRuleSetQuotaAllocationEntityTypeRule
type QuotaAllocationType = string

const (
	QuotaAllocationTypePercentage  QuotaAllocationType = "percentage"
	QuotaAllocationTypeLockedUnits QuotaAllocationType = "lockedUnits"
	QuotaAllocationTypeUnspecified QuotaAllocationType = "unspecified"
)

var maxQuotaAllocationPercentage = resource.MustParse("100")

// ValidateQuotaAllocationRules keeps the released checks before a remote call.
func (s *QuotaAllocationRuleSetSpec) ValidateQuotaAllocationRules() error {
	seen := make(map[string]struct{}, len(s.Rules))
	for _, rule := range s.Rules {
		if _, exists := seen[rule.EntityType]; exists {
			return fmt.Errorf("duplicate quota allocation rule entityType %q", rule.EntityType)
		}
		if rule.Allocation.Sign() < 0 {
			return fmt.Errorf("quota allocation rule entityType %q has negative allocation", rule.EntityType)
		}
		if (rule.AllocationType == nil || *rule.AllocationType == QuotaAllocationTypePercentage || *rule.AllocationType == QuotaAllocationTypeUnspecified) &&
			rule.Allocation.Cmp(maxQuotaAllocationPercentage) > 0 {
			return fmt.Errorf("quota allocation rule entityType %q has percentage allocation greater than 100", rule.EntityType)
		}
		seen[rule.EntityType] = struct{}{}
	}
	return nil
}

// ExtractQuotaAllocationRuleSetRequest keeps the released Go API. The generated conversion
// supplies all current and future resource fields.
func (s *QuotaAllocationRuleSetSpec) ExtractQuotaAllocationRuleSetRequest() (*quotas.QuotaAllocationEntityTypeRuleSet, error) {
	if err := s.ValidateQuotaAllocationRules(); err != nil {
		return nil, err
	}
	ruleSet, err := s.toQuotaAllocationEntityTypeRuleSetUpdate(context.Background(), "")
	if err != nil {
		return nil, err
	}
	defaultQuotaAllocationTypes(ruleSet.Rules)
	return ruleSet, nil
}

// ExtractQuotaAllocationRules converts the rules and keeps their released order.
func (s *QuotaAllocationRuleSetSpec) ExtractQuotaAllocationRules() ([]quotas.QuotaAllocationEntityTypeRule, error) {
	ruleSet, err := s.ExtractQuotaAllocationRuleSetRequest()
	if err != nil {
		return nil, err
	}
	return ruleSet.Rules, nil
}

// ExtractQuotaAllocationEntityTypeRule keeps the released single-rule conversion.
func (r *QuotaAllocationRuleSetQuotaAllocationEntityTypeRule) ExtractQuotaAllocationEntityTypeRule() quotas.QuotaAllocationEntityTypeRule {
	allocationType := quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_PERCENTAGE
	if r.AllocationType != nil && *r.AllocationType == QuotaAllocationTypeLockedUnits {
		allocationType = quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_LOCKED_UNITS
	}
	return quotas.QuotaAllocationEntityTypeRule{
		EntityType:     r.EntityType,
		Allocation:     float32(r.Allocation.AsApproximateFloat64()),
		AllocationType: allocationType.Ptr(),
		Enabled:        r.Enabled,
		CanOverflow:    r.CanOverflow,
	}
}

func defaultQuotaAllocationTypes(rules []quotas.QuotaAllocationEntityTypeRule) {
	for i := range rules {
		if rules[i].AllocationType == nil {
			rules[i].AllocationType = quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_PERCENTAGE.Ptr()
		}
	}
}
