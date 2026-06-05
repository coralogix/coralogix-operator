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

func TestPreserveManagedQuotaAllocationRulesAppendsManagedRules(t *testing.T) {
	planned := []quotas.QuotaAllocationEntityTypeRule{quotaRule("logs", false)}
	current := []quotas.QuotaAllocationEntityTypeRule{
		quotaRule("metrics", true),
		quotaRule("spans", false),
	}

	result := PreserveManagedQuotaAllocationRules(planned, current)

	require.Len(t, result, 2)
	require.Equal(t, "logs", result[0].EntityType)
	require.Equal(t, "metrics", result[1].EntityType)
	require.Nil(t, result[1].CxManaged)
}

func TestPreserveManagedQuotaAllocationRulesFiltersManagedDuplicates(t *testing.T) {
	planned := []quotas.QuotaAllocationEntityTypeRule{quotaRule("logs", false)}
	current := []quotas.QuotaAllocationEntityTypeRule{
		quotaRule("logs", true),
		quotaRule("metrics", true),
	}

	result := PreserveManagedQuotaAllocationRules(planned, current)

	require.Len(t, result, 2)
	require.Equal(t, "logs", result[0].EntityType)
	require.False(t, result[0].GetCxManaged())
	require.Equal(t, "metrics", result[1].EntityType)
	require.Nil(t, result[1].CxManaged)
}

func TestPreserveManagedQuotaAllocationRulesKeepsManagedRulesOnDelete(t *testing.T) {
	current := []quotas.QuotaAllocationEntityTypeRule{
		quotaRule("logs", false),
		quotaRule("metrics", true),
	}

	result := PreserveManagedQuotaAllocationRules(nil, current)

	require.Len(t, result, 1)
	require.Equal(t, "metrics", result[0].EntityType)
	require.Nil(t, result[0].CxManaged)
}

func quotaRule(entityType string, cxManaged bool) quotas.QuotaAllocationEntityTypeRule {
	rule := quotas.QuotaAllocationEntityTypeRule{
		EntityType:  entityType,
		Allocation:  50,
		Enabled:     true,
		CanOverflow: false,
	}
	rule.SetCxManaged(cxManaged)
	return rule
}
