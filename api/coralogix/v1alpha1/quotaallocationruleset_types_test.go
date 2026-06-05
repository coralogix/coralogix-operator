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
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

func TestExtractQuotaAllocationRuleSetRequestDefaultsAllocationType(t *testing.T) {
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType:  "logs",
			Allocation:  resource.MustParse("60"),
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
			Allocation:     resource.MustParse("1000"),
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

func TestExtractQuotaAllocationRuleSetRequestAllowsLockedUnitsAboveOneHundred(t *testing.T) {
	allocationType := QuotaAllocationTypeLockedUnits
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType:     "metrics",
			Allocation:     resource.MustParse("1000"),
			AllocationType: &allocationType,
			Enabled:        true,
			CanOverflow:    false,
		}},
	}

	ruleSet, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.NoError(t, err)
	require.Len(t, ruleSet.Rules, 1)
	require.Equal(t, float32(1000), ruleSet.Rules[0].Allocation)
}

func TestExtractQuotaAllocationRuleSetRequestPreservesFractionalAllocation(t *testing.T) {
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType:  "logs",
			Allocation:  resource.MustParse("12.5"),
			Enabled:     true,
			CanOverflow: true,
		}},
	}

	ruleSet, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.NoError(t, err)
	require.Len(t, ruleSet.Rules, 1)
	require.Equal(t, float32(12.5), ruleSet.Rules[0].Allocation)
}

func TestExtractQuotaAllocationRuleSetRequestRejectsDuplicateEntityTypes(t *testing.T) {
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{
			{EntityType: "logs", Allocation: resource.MustParse("60"), Enabled: true},
			{EntityType: "logs", Allocation: resource.MustParse("40"), Enabled: true},
		},
	}

	_, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.ErrorContains(t, err, `duplicate quota allocation rule entityType "logs"`)
}

func TestExtractQuotaAllocationRuleSetRequestRejectsNegativeAllocation(t *testing.T) {
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType: "logs",
			Allocation: resource.MustParse("-1"),
			Enabled:    true,
		}},
	}

	_, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.ErrorContains(t, err, `quota allocation rule entityType "logs" has negative allocation`)
}

func TestExtractQuotaAllocationRuleSetRequestRejectsDefaultPercentageAboveOneHundred(t *testing.T) {
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType: "logs",
			Allocation: resource.MustParse("101"),
			Enabled:    true,
		}},
	}

	_, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.ErrorContains(t, err, `quota allocation rule entityType "logs" has percentage allocation greater than 100`)
}

func TestExtractQuotaAllocationRuleSetRequestRejectsPercentageAboveOneHundred(t *testing.T) {
	allocationType := QuotaAllocationTypePercentage
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType:     "logs",
			Allocation:     resource.MustParse("101"),
			AllocationType: &allocationType,
			Enabled:        true,
		}},
	}

	_, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.ErrorContains(t, err, `quota allocation rule entityType "logs" has percentage allocation greater than 100`)
}

func TestExtractQuotaAllocationRuleSetRequestRejectsUnspecifiedAboveOneHundred(t *testing.T) {
	allocationType := QuotaAllocationTypeUnspecified
	spec := &QuotaAllocationRuleSetSpec{
		Rules: []QuotaAllocationRule{{
			EntityType:     "logs",
			Allocation:     resource.MustParse("101"),
			AllocationType: &allocationType,
			Enabled:        true,
		}},
	}

	_, err := spec.ExtractQuotaAllocationRuleSetRequest()
	require.ErrorContains(t, err, `quota allocation rule entityType "logs" has percentage allocation greater than 100`)
}

func TestQuotaAllocationRuleSetYAMLDecodesIntegerAndFractionalAllocations(t *testing.T) {
	manifest := []byte(`
apiVersion: coralogix.com/v1alpha1
kind: QuotaAllocationRuleSet
metadata:
  name: example
spec:
  rules:
    - entityType: logs
      allocation: 60
      enabled: true
      canOverflow: true
    - entityType: metrics
      allocation: "12.5"
      enabled: true
      canOverflow: false
`)

	var ruleSet QuotaAllocationRuleSet
	require.NoError(t, yaml.Unmarshal(manifest, &ruleSet))

	request, err := ruleSet.Spec.ExtractQuotaAllocationRuleSetRequest()
	require.NoError(t, err)
	require.Len(t, request.Rules, 2)
	require.Equal(t, float32(60), request.Rules[0].Allocation)
	require.Equal(t, float32(12.5), request.Rules[1].Allocation)
}
