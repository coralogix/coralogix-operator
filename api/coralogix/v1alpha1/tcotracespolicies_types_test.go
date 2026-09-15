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

func TestExtractCreateSpanPolicyRequestDpxlAndPriorityOverride(t *testing.T) {
	policy := TCOTracesPolicy{
		Name:           "traces-advanced",
		Priority:       "low",
		DpxlExpression: ptr.To("<v1> $d.status == 'ERROR'"),
		PriorityOverride: &TCOPolicyPriorityOverride{
			QuotaBased: &TCOPolicyQuotaBased{
				UsageTiers: []TCOPolicyUsageTier{
					{DailyQuotaPercentage: resource.MustParse("60"), Priority: "medium"},
				},
			},
		},
	}

	got, err := policy.extractCreateSpanPolicyRequest(map[string]string{})
	require.NoError(t, err)

	// dpxl maps onto SpanRules alongside the (empty) structured rules.
	require.Equal(t, ptr.To("<v1> $d.status == 'ERROR'"), got.SpanRules.DpxlExpression)
	require.Nil(t, got.SpanRules.ServiceRule)
	require.Nil(t, got.SpanRules.ActionRule)

	require.NotNil(t, got.Policy.PriorityOverride)
	tiers := got.Policy.PriorityOverride.QuotaBased.UsageTiers
	require.Len(t, tiers, 1)
	require.Equal(t, float64(60), *tiers[0].DailyQuotaPercentage)
	require.Equal(t, tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_MEDIUM, *tiers[0].Priority)
}

func TestExtractCreateSpanPolicyRequestStructuredRulesNoDpxl(t *testing.T) {
	policy := TCOTracesPolicy{
		Name:     "traces-structured",
		Priority: "high",
		Services: &TCOPolicyRule{Names: []string{"service"}, RuleType: "is"},
	}

	got, err := policy.extractCreateSpanPolicyRequest(map[string]string{})
	require.NoError(t, err)
	require.Nil(t, got.SpanRules.DpxlExpression)
	require.NotNil(t, got.SpanRules.ServiceRule)
	require.Nil(t, got.Policy.PriorityOverride)
}
