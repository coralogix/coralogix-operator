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
	"testing"

	quotas "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/quota_allocation_rule_set_service"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
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

func TestRejectManagedQuotaAllocationRuleCollisionsRejectsManagedEntityType(t *testing.T) {
	planned := []quotas.QuotaAllocationEntityTypeRule{quotaRule("logs", false)}
	current := []quotas.QuotaAllocationEntityTypeRule{
		quotaRule("logs", true),
		quotaRule("metrics", true),
	}

	err := RejectManagedQuotaAllocationRuleCollisions(planned, current)

	require.ErrorContains(t, err, `quota allocation rule entityType "logs" is managed by Coralogix and cannot be replaced`)
}

func TestRejectManagedQuotaAllocationRuleCollisionsAllowsUserManagedEntityType(t *testing.T) {
	planned := []quotas.QuotaAllocationEntityTypeRule{quotaRule("logs", false)}
	current := []quotas.QuotaAllocationEntityTypeRule{
		quotaRule("logs", false),
		quotaRule("metrics", true),
	}

	require.NoError(t, RejectManagedQuotaAllocationRuleCollisions(planned, current))
}

func TestEnsureSingleSelectedRuleSetRejectsAnotherActiveResource(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, coralogixv1alpha1.AddToScheme(scheme))

	current := &coralogixv1alpha1.QuotaAllocationRuleSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "current",
			Namespace: "team-a",
			UID:       types.UID("current"),
		},
	}
	other := &coralogixv1alpha1.QuotaAllocationRuleSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other",
			Namespace: "team-b",
			UID:       types.UID("other"),
		},
	}

	originalClient := config.GetClient()
	originalSelector := config.GetConfig().Selector
	t.Cleanup(func() {
		config.InitClient(originalClient)
		config.GetConfig().Selector = originalSelector
	})

	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(current, other).Build())
	config.GetConfig().Selector = config.Selector{}

	err := (&QuotaAllocationRuleSetReconciler{}).ensureSingleSelectedRuleSet(context.Background(), current)

	require.ErrorContains(t, err, "only one selected QuotaAllocationRuleSet can manage account-level quota allocation rules")
	require.ErrorContains(t, err, "team-b/other")
}

func TestQuotaAllocationRuleSetRemoteSynced(t *testing.T) {
	tests := []struct {
		name    string
		ruleSet *coralogixv1alpha1.QuotaAllocationRuleSet
		want    bool
	}{
		{
			name: "synced current generation",
			ruleSet: quotaAllocationRuleSetWithRemoteSyncedCondition(
				metav1.ConditionTrue,
				2,
				2,
			),
			want: true,
		},
		{
			name: "synced stale generation",
			ruleSet: quotaAllocationRuleSetWithRemoteSyncedCondition(
				metav1.ConditionTrue,
				2,
				1,
			),
			want: false,
		},
		{
			name: "not synced",
			ruleSet: quotaAllocationRuleSetWithRemoteSyncedCondition(
				metav1.ConditionFalse,
				2,
				2,
			),
			want: false,
		},
		{
			name:    "nil",
			ruleSet: nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, quotaAllocationRuleSetRemoteSynced(tt.ruleSet))
		})
	}
}

func quotaAllocationRuleSetWithRemoteSyncedCondition(
	status metav1.ConditionStatus,
	generation int64,
	observedGeneration int64,
) *coralogixv1alpha1.QuotaAllocationRuleSet {
	return &coralogixv1alpha1.QuotaAllocationRuleSet{
		ObjectMeta: metav1.ObjectMeta{
			Generation: generation,
		},
		Status: coralogixv1alpha1.QuotaAllocationRuleSetStatus{
			Conditions: []metav1.Condition{
				{
					Type:               utils.ConditionTypeRemoteSynced,
					Status:             status,
					ObservedGeneration: observedGeneration,
				},
			},
		},
	}
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
