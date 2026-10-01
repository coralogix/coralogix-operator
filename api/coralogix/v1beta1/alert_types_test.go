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

package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"
)

func analyticsAlertSpec(typeDefinition AlertTypeDefinition) *AlertSpec {
	return &AlertSpec{
		Name:           "analytics",
		Priority:       AlertPriorityP3,
		GroupByKeys:    []string{"applicationname"},
		TypeDefinition: typeDefinition,
	}
}

func TestExtractAlertDefPropertiesAnalyticsImmediate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		immediate *AnalyticsImmediate
		want      *alerts.AnalyticsImmediateType
	}{
		{
			name: "minimal leaves optionals unset",
			immediate: &AnalyticsImmediate{
				DataprimeQuery:   DataprimeQuery{Query: "source logs | count"},
				TimeframeMinutes: 15,
			},
			want: &alerts.AnalyticsImmediateType{
				DataprimeQuery:   &alerts.DataprimeAlertQuery{Query: ptr.To("source logs | count")},
				TimeframeMinutes: ptr.To[int32](15),
			},
		},
		{
			name: "explicit zero values are sent",
			immediate: &AnalyticsImmediate{
				DataprimeQuery:        DataprimeQuery{Query: "source logs | count"},
				TimeframeMinutes:      15,
				UseRowsAsPermutations: ptr.To(false),
				EvaluationDelayMs:     ptr.To[int32](0),
			},
			want: &alerts.AnalyticsImmediateType{
				DataprimeQuery:        &alerts.DataprimeAlertQuery{Query: ptr.To("source logs | count")},
				TimeframeMinutes:      ptr.To[int32](15),
				UseRowsAsPermutations: ptr.To(false),
				EvaluationDelayMs:     ptr.To[int32](0),
			},
		},
		{
			name: "full",
			immediate: &AnalyticsImmediate{
				DataprimeQuery:        DataprimeQuery{Query: "source logs | count"},
				TimeframeMinutes:      45,
				UseRowsAsPermutations: ptr.To(true),
				EvaluationDelayMs:     ptr.To[int32](120000),
				NoDataPolicy:          &NoDataPolicy{State: NoDataPolicyStateAlerting, AutoRetireSeconds: ptr.To[int32](3600)},
			},
			want: &alerts.AnalyticsImmediateType{
				DataprimeQuery:        &alerts.DataprimeAlertQuery{Query: ptr.To("source logs | count")},
				TimeframeMinutes:      ptr.To[int32](45),
				UseRowsAsPermutations: ptr.To(true),
				EvaluationDelayMs:     ptr.To[int32](120000),
				NoDataPolicy: &alerts.NoDataPolicy{
					State:             alerts.NODATAPOLICYSTATE_NO_DATA_POLICY_STATE_ALERTING.Ptr(),
					AutoRetireSeconds: ptr.To[int32](3600),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			props, err := analyticsAlertSpec(AlertTypeDefinition{AnalyticsImmediate: tc.immediate}).
				ExtractAlertDefProperties(&GetResourceRefProperties{})
			require.NoError(t, err)
			require.Equal(t, alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_ANALYTICS_IMMEDIATE, props.GetType())
			require.Equal(t, []string{"applicationname"}, props.GroupByKeys)
			require.Nil(t, props.AnalyticsThreshold)
			require.Equal(t, tc.want, props.AnalyticsImmediate)
		})
	}
}

func TestExtractAlertDefPropertiesAnalyticsThreshold(t *testing.T) {
	p1 := &alerts.AlertDefOverride{Priority: alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P1.Ptr()}
	p3 := &alerts.AlertDefOverride{Priority: alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P3.Ptr()}

	t.Run("minimal defaults override to alert priority and leaves operator unset", func(t *testing.T) {
		props, err := analyticsAlertSpec(AlertTypeDefinition{AnalyticsThreshold: &AnalyticsThreshold{
			DataprimeQuery:   DataprimeQuery{Query: "source logs | count as c"},
			TimeframeMinutes: 30,
			Rules:            []AnalyticsThresholdRule{{Condition: AnalyticsThresholdRuleCondition{Threshold: resource.MustParse("10")}}},
		}}).ExtractAlertDefProperties(&GetResourceRefProperties{})
		require.NoError(t, err)
		require.Equal(t, alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_ANALYTICS_THRESHOLD, props.GetType())
		require.Nil(t, props.AnalyticsImmediate)
		require.Equal(t, &alerts.AnalyticsThresholdType{
			DataprimeQuery:   &alerts.DataprimeAlertQuery{Query: ptr.To("source logs | count as c")},
			TimeframeMinutes: ptr.To[int32](30),
			Rules: []alerts.AnalyticsThresholdRule{
				{Condition: &alerts.AnalyticsThresholdRuleCondition{Threshold: ptr.To(10.0)}, Override: p3},
			},
		}, props.AnalyticsThreshold)
	})

	t.Run("full keeps rule order", func(t *testing.T) {
		props, err := analyticsAlertSpec(AlertTypeDefinition{AnalyticsThreshold: &AnalyticsThreshold{
			DataprimeQuery:        DataprimeQuery{Query: "source logs | count as c"},
			TimeframeMinutes:      30,
			Operator:              ptr.To(AnalyticsThresholdOperatorLessThan),
			TargetColumn:          ptr.To("c"),
			UseRowsAsPermutations: ptr.To(false),
			EvaluationDelayMs:     ptr.To[int32](60000),
			NoDataPolicy:          &NoDataPolicy{State: NoDataPolicyStateKeepLast},
			Rules: []AnalyticsThresholdRule{
				{Condition: AnalyticsThresholdRuleCondition{Threshold: resource.MustParse("42.5")}, Override: &AlertOverride{Priority: AlertPriorityP1}},
				{Condition: AnalyticsThresholdRuleCondition{Threshold: resource.MustParse("20")}},
			},
		}}).ExtractAlertDefProperties(&GetResourceRefProperties{})
		require.NoError(t, err)
		require.Equal(t, &alerts.AnalyticsThresholdType{
			DataprimeQuery:        &alerts.DataprimeAlertQuery{Query: ptr.To("source logs | count as c")},
			TimeframeMinutes:      ptr.To[int32](30),
			Operator:              alerts.ANALYTICSTHRESHOLDOPERATOR_ANALYTICS_THRESHOLD_OPERATOR_LESS_THAN.Ptr(),
			TargetColumn:          ptr.To("c"),
			UseRowsAsPermutations: ptr.To(false),
			EvaluationDelayMs:     ptr.To[int32](60000),
			NoDataPolicy:          &alerts.NoDataPolicy{State: alerts.NODATAPOLICYSTATE_NO_DATA_POLICY_STATE_KEEP_LAST.Ptr()},
			Rules: []alerts.AnalyticsThresholdRule{
				{Condition: &alerts.AnalyticsThresholdRuleCondition{Threshold: ptr.To(42.5)}, Override: p1},
				{Condition: &alerts.AnalyticsThresholdRuleCondition{Threshold: ptr.To(20.0)}, Override: p3},
			},
		}, props.AnalyticsThreshold)
	})

	for operator, want := range map[AnalyticsThresholdOperator]alerts.AnalyticsThresholdOperator{
		AnalyticsThresholdOperatorMoreThan:         alerts.ANALYTICSTHRESHOLDOPERATOR_ANALYTICS_THRESHOLD_OPERATOR_MORE_THAN_OR_UNSPECIFIED,
		AnalyticsThresholdOperatorLessThan:         alerts.ANALYTICSTHRESHOLDOPERATOR_ANALYTICS_THRESHOLD_OPERATOR_LESS_THAN,
		AnalyticsThresholdOperatorMoreThanOrEquals: alerts.ANALYTICSTHRESHOLDOPERATOR_ANALYTICS_THRESHOLD_OPERATOR_MORE_THAN_OR_EQUALS,
		AnalyticsThresholdOperatorLessThanOrEquals: alerts.ANALYTICSTHRESHOLDOPERATOR_ANALYTICS_THRESHOLD_OPERATOR_LESS_THAN_OR_EQUALS,
		AnalyticsThresholdOperatorEquals:           alerts.ANALYTICSTHRESHOLDOPERATOR_ANALYTICS_THRESHOLD_OPERATOR_EQUALS,
		AnalyticsThresholdOperatorNotEquals:        alerts.ANALYTICSTHRESHOLDOPERATOR_ANALYTICS_THRESHOLD_OPERATOR_NOT_EQUALS,
	} {
		t.Run("operator "+string(operator), func(t *testing.T) {
			props, err := analyticsAlertSpec(AlertTypeDefinition{AnalyticsThreshold: &AnalyticsThreshold{
				DataprimeQuery:   DataprimeQuery{Query: "source logs | count as c"},
				TimeframeMinutes: 30,
				Operator:         ptr.To(operator),
				Rules:            []AnalyticsThresholdRule{{Condition: AnalyticsThresholdRuleCondition{Threshold: resource.MustParse("1")}}},
			}}).ExtractAlertDefProperties(&GetResourceRefProperties{})
			require.NoError(t, err)
			require.Equal(t, want, props.AnalyticsThreshold.GetOperator())
		})
	}
}
