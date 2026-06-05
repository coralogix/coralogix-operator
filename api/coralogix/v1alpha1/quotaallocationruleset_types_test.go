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
	"testing"

	quotas "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/quota_allocation_rule_set_service"
	"github.com/stretchr/testify/require"
)

func TestExtractQuotaAllocationRuleSetRequestDefaultsAllocationType(t *testing.T) {
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType:  "logs",
			Allocation:  60,
			Enabled:     true,
			CanOverflow: true,
		}},
	}

	ruleSet, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.NoError(t, err)
	require.Len(t, ruleSet.Rules, 1)
	require.Equal(t, quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_PERCENTAGE, ruleSet.Rules[0].GetAllocationType())
	require.Nil(t, ruleSet.Rules[0].CxManaged)
}

func TestExtractQuotaAllocationRuleSetRequestMapsLockedUnits(t *testing.T) {
	allocationType := QuotaAllocationTypeLockedUnits
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType:     "metrics",
			Allocation:     1000,
			AllocationType: &allocationType,
			Enabled:        true,
			CanOverflow:    false,
		}},
	}

	ruleSet, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.NoError(t, err)
	require.Len(t, ruleSet.Rules, 1)
	require.Equal(t, quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_LOCKED_UNITS, ruleSet.Rules[0].GetAllocationType())
}

func TestExtractQuotaAllocationRuleSetRequestRejectsDuplicateEntityTypes(t *testing.T) {
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{
			{EntityType: "logs", Allocation: 60, Enabled: true},
			{EntityType: "logs", Allocation: 40, Enabled: true},
		},
	}

	_, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.ErrorContains(t, err, `duplicate quota allocation rule entityType "logs"`)
}
