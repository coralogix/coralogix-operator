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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
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

func newRefResource(kind, name, id string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: utils.CoralogixAPIGroup, Version: utils.V1alpha1APIVersion, Kind: kind})
	u.SetName(name)
	u.SetNamespace("default")
	if id != "" {
		_ = unstructured.SetNestedField(u.Object, id, "status", "id")
	}
	return u
}

func TestExtractAlertDefPropertiesCaseSettings(t *testing.T) {
	config.InitClient(fake.NewClientBuilder().WithObjects(
		newRefResource(utils.ConnectorKind, "cases-connector", "connector-from-ref"),
		newRefResource(utils.PresetKind, "cases-preset", "preset-from-ref"),
		newRefResource(utils.ConnectorKind, "unsynced-connector", ""),
	).Build())
	refProperties := &GetResourceRefProperties{Ctx: context.Background(), Namespace: "default"}
	backendRef := func(id string) NCRef { return NCRef{BackendRef: &NCBackendRef{ID: id}} }
	resourceRef := func(name string) NCRef { return NCRef{ResourceRef: &ResourceRef{Name: name}} }

	for _, tc := range []struct {
		name         string
		caseSettings *AlertCaseSettings
		want         *alerts.AlertDefCaseSettings
		wantErr      string
	}{
		{
			name: "omitted is not sent",
		},
		{
			name:         "empty object is sent empty",
			caseSettings: &AlertCaseSettings{},
			want:         &alerts.AlertDefCaseSettings{},
		},
		{
			name: "explicit empty lists are kept",
			caseSettings: &AlertCaseSettings{
				EnrichmentQueries: []CaseEnrichmentQuery{},
				Destinations:      []CaseDestination{},
			},
			want: &alerts.AlertDefCaseSettings{
				EnrichmentQueries: []alerts.AlertDefCaseEnrichmentQuery{},
				Destinations:      []alerts.AlertDefCaseDestination{},
			},
		},
		{
			name:         "auto-resolve enabled",
			caseSettings: &AlertCaseSettings{AutoResolveMode: ptr.To(CaseAutoResolveModeEnabled)},
			want: &alerts.AlertDefCaseSettings{
				AutoResolveMode: alerts.ALERTDEFCASEAUTORESOLVEMODE_ALERT_DEF_CASE_AUTO_RESOLVE_MODE_ENABLED.Ptr(),
			},
		},
		{
			name: "full with backend refs",
			caseSettings: &AlertCaseSettings{
				AutoResolveMode: ptr.To(CaseAutoResolveModeDisabled),
				EnrichmentQueries: []CaseEnrichmentQuery{
					{Query: "source logs | limit 1", Type: ptr.To(CaseEnrichmentQueryTypeDataprime)},
				},
				Destinations: []CaseDestination{
					{Connector: backendRef("connector-1"), Preset: ptr.To(backendRef("preset-1")), Condition: "true"},
					{Connector: backendRef("connector-2"), Condition: "caseMetadata.notificationReason == 'caseResolved'"},
				},
			},
			want: &alerts.AlertDefCaseSettings{
				AutoResolveMode: alerts.ALERTDEFCASEAUTORESOLVEMODE_ALERT_DEF_CASE_AUTO_RESOLVE_MODE_DISABLED.Ptr(),
				EnrichmentQueries: []alerts.AlertDefCaseEnrichmentQuery{
					{Query: "source logs | limit 1", Type: alerts.ALERTDEFCASEENRICHMENTQUERYTYPE_ALERT_DEF_CASE_ENRICHMENT_QUERY_TYPE_DATAPRIME.Ptr()},
				},
				Destinations: []alerts.AlertDefCaseDestination{
					{ConnectorId: "connector-1", PresetId: ptr.To("preset-1"), Condition: "true"},
					{ConnectorId: "connector-2", Condition: "caseMetadata.notificationReason == 'caseResolved'"},
				},
			},
		},
		{
			name: "query type is left unset when omitted",
			caseSettings: &AlertCaseSettings{
				EnrichmentQueries: []CaseEnrichmentQuery{{Query: "source logs | limit 1"}},
			},
			want: &alerts.AlertDefCaseSettings{
				EnrichmentQueries: []alerts.AlertDefCaseEnrichmentQuery{{Query: "source logs | limit 1"}},
			},
		},
		{
			name: "resource refs are resolved to IDs",
			caseSettings: &AlertCaseSettings{
				Destinations: []CaseDestination{
					{Connector: resourceRef("cases-connector"), Preset: ptr.To(resourceRef("cases-preset")), Condition: "true"},
				},
			},
			want: &alerts.AlertDefCaseSettings{
				Destinations: []alerts.AlertDefCaseDestination{
					{ConnectorId: "connector-from-ref", PresetId: ptr.To("preset-from-ref"), Condition: "true"},
				},
			},
		},
		{
			name: "missing connector resource fails",
			caseSettings: &AlertCaseSettings{
				Destinations: []CaseDestination{{Connector: resourceRef("missing-connector"), Condition: "true"}},
			},
			wantErr: "failed to expand case settings: failed to expand case destination connector ID",
		},
		{
			name: "connector without an ID fails",
			caseSettings: &AlertCaseSettings{
				Destinations: []CaseDestination{{Connector: resourceRef("unsynced-connector"), Condition: "true"}},
			},
			wantErr: "does not have an ID populated",
		},
		{
			name: "missing preset resource fails",
			caseSettings: &AlertCaseSettings{
				Destinations: []CaseDestination{
					{Connector: backendRef("connector-1"), Preset: ptr.To(resourceRef("missing-preset")), Condition: "true"},
				},
			},
			wantErr: "failed to expand case destination preset ID",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := analyticsAlertSpec(AlertTypeDefinition{LogsImmediate: &LogsImmediate{}})
			spec.GroupByKeys = nil
			spec.CaseSettings = tc.caseSettings
			props, err := spec.ExtractAlertDefProperties(refProperties)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, props.CaseSettings)
		})
	}
}

func TestExtractAlertDefPropertiesCaseSettingsForEveryAlertType(t *testing.T) {
	want := &alerts.AlertDefCaseSettings{
		AutoResolveMode: alerts.ALERTDEFCASEAUTORESOLVEMODE_ALERT_DEF_CASE_AUTO_RESOLVE_MODE_DISABLED.Ptr(),
	}

	for name, typeDefinition := range map[string]AlertTypeDefinition{
		"logsImmediate":             {LogsImmediate: &LogsImmediate{}},
		"logsThreshold":             {LogsThreshold: &LogsThreshold{}},
		"logsRatioThreshold":        {LogsRatioThreshold: &LogsRatioThreshold{}},
		"logsTimeRelativeThreshold": {LogsTimeRelativeThreshold: &LogsTimeRelativeThreshold{}},
		"metricThreshold":           {MetricThreshold: &MetricThreshold{}},
		"tracingThreshold":          {TracingThreshold: &TracingThreshold{}},
		"tracingImmediate":          {TracingImmediate: &TracingImmediate{}},
		"flow":                      {Flow: &Flow{}},
		"logsAnomaly":               {LogsAnomaly: &LogsAnomaly{}},
		"metricAnomaly":             {MetricAnomaly: &MetricAnomaly{}},
		"logsNewValue":              {LogsNewValue: &LogsNewValue{}},
		"logsUniqueCount":           {LogsUniqueCount: &LogsUniqueCount{}},
		"sloThreshold": {SloThreshold: &SloThreshold{
			SloDefinition: SloDefinition{SloRef: SloRef{BackendRef: &SloBackendRef{ID: ptr.To("slo-1")}}},
			ErrorBudget:   &ErrorBudget{},
		}},
		"analyticsImmediate": {AnalyticsImmediate: &AnalyticsImmediate{}},
		"analyticsThreshold": {AnalyticsThreshold: &AnalyticsThreshold{}},
	} {
		t.Run(name, func(t *testing.T) {
			spec := analyticsAlertSpec(typeDefinition)
			spec.CaseSettings = &AlertCaseSettings{AutoResolveMode: ptr.To(CaseAutoResolveModeDisabled)}
			props, err := spec.ExtractAlertDefProperties(&GetResourceRefProperties{})
			require.NoError(t, err)
			require.Equal(t, want, props.CaseSettings)
		})
	}
}

func TestExpandLogsUniqueCountMaxUniqueCountPerGroupByKey(t *testing.T) {
	t.Run("unset is not sent", func(t *testing.T) {
		got := expandLogsUniqueCount(&LogsUniqueCount{UniqueCountKeypath: "remote_addr"})
		require.Nil(t, got.MaxUniqueCountPerGroupByKey)
		require.Equal(t, "remote_addr", got.GetUniqueCountKeypath())
	})

	t.Run("set is sent", func(t *testing.T) {
		got := expandLogsUniqueCount(&LogsUniqueCount{MaxUniqueCountPerGroupByKey: ptr.To[uint64](10)})
		require.Equal(t, ptr.To("10"), got.MaxUniqueCountPerGroupByKey)
	})
}

func TestExtractAlertDefPropertiesUngroupedLogsUniqueCount(t *testing.T) {
	spec := analyticsAlertSpec(AlertTypeDefinition{LogsUniqueCount: &LogsUniqueCount{}})
	spec.GroupByKeys = nil
	require.NotPanics(t, func() {
		_, err := spec.ExtractAlertDefProperties(&GetResourceRefProperties{})
		require.NoError(t, err)
	})
}
