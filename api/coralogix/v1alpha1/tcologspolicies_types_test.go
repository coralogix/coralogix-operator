// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"
)

func TestExtractCreateLogPolicyRequestDpxlAndPriorityOverride(t *testing.T) {
	policy := TCOLogsPolicy{
		Name:           "logs-advanced",
		Priority:       ptr.To("low"),
		DpxlExpression: ptr.To("<v1>$d.applicationname == 'prod'"),
		PriorityOverride: &TCOPolicyPriorityOverride{
			QuotaBased: &TCOPolicyQuotaBased{
				UsageTiers: []TCOPolicyUsageTier{
					{DailyQuotaPercentage: resource.MustParse("30"), Priority: "high"},
					{DailyQuotaPercentage: resource.MustParse("60"), Priority: "medium"},
				},
			},
		},
	}

	got, err := policy.extractCreateLogPolicyRequest(map[string]string{})
	require.NoError(t, err)

	// dpxl maps onto LogRules and severities stays empty.
	require.Equal(t, ptr.To("<v1>$d.applicationname == 'prod'"), got.LogRules.DpxlExpression)
	require.Empty(t, got.LogRules.Severities)

	// policy-level priority override maps onto the generic policy.
	require.NotNil(t, got.Policy.PriorityOverride)
	require.NotNil(t, got.Policy.PriorityOverride.QuotaBased)
	tiers := got.Policy.PriorityOverride.QuotaBased.UsageTiers
	require.Len(t, tiers, 2)
	require.Equal(t, float64(30), *tiers[0].DailyQuotaPercentage)
	require.Equal(t, tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_HIGH, *tiers[0].Priority)
	require.Equal(t, float64(60), *tiers[1].DailyQuotaPercentage)
	require.Equal(t, tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_MEDIUM, *tiers[1].Priority)
}

func TestExtractCreateLogPolicyRequestSeveritiesNoDpxl(t *testing.T) {
	policy := TCOLogsPolicy{
		Name:       "logs-severities",
		Priority:   ptr.To("high"),
		Severities: []TCOPolicySeverity{"info", "error"},
	}

	got, err := policy.extractCreateLogPolicyRequest(map[string]string{})
	require.NoError(t, err)
	require.Nil(t, got.LogRules.DpxlExpression)
	require.Equal(t, []tcopolicies.QuotaV1Severity{
		tcopolicies.QUOTAV1SEVERITY_SEVERITY_INFO,
		tcopolicies.QUOTAV1SEVERITY_SEVERITY_ERROR,
	}, got.LogRules.Severities)
	require.Nil(t, got.Policy.PriorityOverride)
}
