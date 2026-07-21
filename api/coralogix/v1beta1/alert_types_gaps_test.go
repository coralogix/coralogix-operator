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

package v1beta1

import (
	"testing"

	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func extractProperties(t *testing.T, spec AlertSpec) *alerts.AlertDefProperties {
	t.Helper()
	got, err := spec.ExtractAlertDefProperties(nil)
	if err != nil {
		t.Fatalf("ExtractAlertDefProperties returned error: %v", err)
	}
	return got
}

func logsImmediateSpec() AlertSpec {
	return AlertSpec{
		Name:     "logs-immediate",
		Priority: AlertPriorityP3,
		TypeDefinition: AlertTypeDefinition{
			LogsImmediate: &LogsImmediate{},
		},
	}
}

func metricThresholdSpec() AlertSpec {
	return AlertSpec{
		Name:     "metric-threshold",
		Priority: AlertPriorityP3,
		TypeDefinition: AlertTypeDefinition{
			MetricThreshold: &MetricThreshold{
				MetricFilter: MetricFilter{Promql: "sum(up)"},
				Rules: []MetricThresholdRule{
					{
						Condition: MetricThresholdRuleCondition{
							Threshold:     resource.MustParse("5"),
							ForOverPct:    10,
							OfTheLast:     MetricTimeWindow{SpecificValue: ptr.To(MetricTimeWindowValue10Minutes)},
							ConditionType: MetricThresholdConditionTypeMoreThan,
						},
					},
				},
			},
		},
	}
}

func logsRatioThresholdSpec() AlertSpec {
	return AlertSpec{
		Name:     "logs-ratio",
		Priority: AlertPriorityP3,
		TypeDefinition: AlertTypeDefinition{
			LogsRatioThreshold: &LogsRatioThreshold{
				Numerator:        LogsFilter{SimpleFilter: LogsSimpleFilter{LuceneQuery: ptr.To("n")}},
				NumeratorAlias:   "Query 1",
				Denominator:      LogsFilter{SimpleFilter: LogsSimpleFilter{LuceneQuery: ptr.To("d")}},
				DenominatorAlias: "Query 2",
				Rules: []LogsRatioThresholdRule{
					{
						Condition: LogsRatioCondition{
							Threshold:     resource.MustParse("1"),
							TimeWindow:    LogsRatioTimeWindow{SpecificValue: LogsRatioTimeWindowMinutes5},
							ConditionType: LogsRatioConditionTypeLessThan,
						},
					},
				},
			},
		},
	}
}

func logsAnomalySpec() AlertSpec {
	return AlertSpec{
		Name:     "logs-anomaly",
		Priority: AlertPriorityP3,
		TypeDefinition: AlertTypeDefinition{
			LogsAnomaly: &LogsAnomaly{
				Rules: []LogsAnomalyRule{
					{
						Condition: LogsAnomalyCondition{
							MinimumThreshold: resource.MustParse("10"),
							TimeWindow:       LogsTimeWindow{SpecificValue: LogsTimeWindow5Minutes},
						},
					},
				},
			},
		},
	}
}

func metricAnomalySpec() AlertSpec {
	return AlertSpec{
		Name:     "metric-anomaly",
		Priority: AlertPriorityP3,
		TypeDefinition: AlertTypeDefinition{
			MetricAnomaly: &MetricAnomaly{
				MetricFilter: MetricFilter{Promql: "sum(up)"},
				Rules: []MetricAnomalyRule{
					{
						Condition: MetricAnomalyCondition{
							Threshold:           resource.MustParse("5"),
							ForOverPct:          10,
							OfTheLast:           MetricAnomalyTimeWindow{SpecificValue: MetricTimeWindowValue10Minutes},
							MinNonNullValuesPct: 10,
							ConditionType:       MetricAnomalyConditionTypeMoreThanUsual,
						},
					},
				},
			},
		},
	}
}

func tracingThresholdSpec() AlertSpec {
	return AlertSpec{
		Name:     "tracing-threshold",
		Priority: AlertPriorityP3,
		TypeDefinition: AlertTypeDefinition{
			TracingThreshold: &TracingThreshold{
				Rules: []TracingThresholdRule{
					{
						Condition: TracingThresholdRuleCondition{
							SpanAmount: resource.MustParse("5"),
							TimeWindow: TracingTimeWindow{SpecificValue: TracingTimeWindowValue10Minutes},
						},
					},
				},
			},
		},
	}
}

// Root-level dataSources maps to AlertDefProperties.DataSources on every alert type branch.
func TestExtractAlertDefPropertiesDataSources(t *testing.T) {
	for name, spec := range map[string]AlertSpec{
		"logsImmediate":   logsImmediateSpec(),
		"metricThreshold": metricThresholdSpec(),
	} {
		spec.DataSources = []AlertDataSource{
			{DataSpace: ptr.To("default"), DataSet: ptr.To("my-dataset")},
		}

		got := extractProperties(t, spec)
		if len(got.DataSources) != 1 {
			t.Fatalf("%s: DataSources length = %d, want 1", name, len(got.DataSources))
		}
		if got.DataSources[0].GetDataSpace() != "default" || got.DataSources[0].GetDataSet() != "my-dataset" {
			t.Fatalf("%s: DataSources[0] = %+v, want dataSpace=default dataSet=my-dataset", name, got.DataSources[0])
		}
	}

	// Unset dataSources => nil.
	got := extractProperties(t, logsImmediateSpec())
	if got.DataSources != nil {
		t.Fatalf("DataSources should be nil when unset, got %+v", got.DataSources)
	}
}

// groupByFor maps to the SDK enum, including denominatorOnly => DENUMERATOR_ONLY (SDK spelling).
func TestExtractLogsRatioGroupByFor(t *testing.T) {
	for groupByFor, want := range map[LogsRatioGroupByFor]alerts.LogsRatioGroupByFor{
		LogsRatioGroupByForBoth:            alerts.LOGSRATIOGROUPBYFOR_LOGS_RATIO_GROUP_BY_FOR_BOTH_OR_UNSPECIFIED,
		LogsRatioGroupByForNumeratorOnly:   alerts.LOGSRATIOGROUPBYFOR_LOGS_RATIO_GROUP_BY_FOR_NUMERATOR_ONLY,
		LogsRatioGroupByForDenominatorOnly: alerts.LOGSRATIOGROUPBYFOR_LOGS_RATIO_GROUP_BY_FOR_DENUMERATOR_ONLY,
	} {
		spec := logsRatioThresholdSpec()
		spec.TypeDefinition.LogsRatioThreshold.GroupByFor = ptr.To(groupByFor)

		got, err := spec.ExtractAlertDefProperties(nil)
		if err != nil {
			t.Fatalf("ExtractAlertDefProperties returned error: %v", err)
		}
		if got.LogsRatioThreshold.GetGroupByFor() != want {
			t.Fatalf("GroupByFor(%s) = %v, want %v", groupByFor, got.LogsRatioThreshold.GetGroupByFor(), want)
		}
	}

	// Unset groupByFor => nil (server default).
	got := extractProperties(t, logsRatioThresholdSpec())
	if got.LogsRatioThreshold.GroupByFor != nil {
		t.Fatalf("GroupByFor should be nil when unset, got %v", *got.LogsRatioThreshold.GroupByFor)
	}
}

// undetectedValuesManagement on logs-ratio alerts maps trigger and auto-retire timeframe.
func TestExtractLogsRatioUndetectedValuesManagement(t *testing.T) {
	spec := logsRatioThresholdSpec()
	spec.TypeDefinition.LogsRatioThreshold.UndetectedValuesManagement = &UndetectedValuesManagement{
		TriggerUndetectedValues: true,
		AutoRetireTimeframe:     AutoRetireTimeframe5M,
	}

	got := extractProperties(t, spec)
	uvm := got.LogsRatioThreshold.GetUndetectedValuesManagement()
	if !uvm.GetTriggerUndetectedValues() {
		t.Fatalf("TriggerUndetectedValues = false, want true")
	}
	if uvm.GetAutoRetireTimeframe() != alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_MINUTES_5 {
		t.Fatalf("AutoRetireTimeframe = %v, want MINUTES_5", uvm.GetAutoRetireTimeframe())
	}

	// Unset undetectedValuesManagement => nil.
	got = extractProperties(t, logsRatioThresholdSpec())
	if got.LogsRatioThreshold.UndetectedValuesManagement != nil {
		t.Fatalf("UndetectedValuesManagement should be nil when unset")
	}
}

// ignoreInfinity on logs-ratio alerts maps to the SDK field.
func TestExtractLogsRatioIgnoreInfinity(t *testing.T) {
	spec := logsRatioThresholdSpec()
	spec.TypeDefinition.LogsRatioThreshold.IgnoreInfinity = true

	got := extractProperties(t, spec)
	if !got.LogsRatioThreshold.GetIgnoreInfinity() {
		t.Fatalf("IgnoreInfinity = false, want true")
	}

	// Zero value => false.
	got = extractProperties(t, logsRatioThresholdSpec())
	if got.LogsRatioThreshold.GetIgnoreInfinity() {
		t.Fatalf("IgnoreInfinity = true, want false")
	}
}

// notificationPayloadFilter on logs-ratio alerts maps to the SDK field.
func TestExtractLogsRatioNotificationPayloadFilter(t *testing.T) {
	spec := logsRatioThresholdSpec()
	spec.TypeDefinition.LogsRatioThreshold.NotificationPayloadFilter = []string{"a", "b"}

	got := extractProperties(t, spec)
	filter := got.LogsRatioThreshold.GetNotificationPayloadFilter()
	if len(filter) != 2 || filter[0] != "a" || filter[1] != "b" {
		t.Fatalf("NotificationPayloadFilter = %v, want [a b]", filter)
	}

	// Unset => empty.
	got = extractProperties(t, logsRatioThresholdSpec())
	if len(got.LogsRatioThreshold.GetNotificationPayloadFilter()) != 0 {
		t.Fatalf("NotificationPayloadFilter should be empty when unset")
	}
}

// anomalyAlertSettings.percentageOfDeviation on logs-anomaly alerts maps to a float32.
func TestExtractLogsAnomalyAlertSettings(t *testing.T) {
	spec := logsAnomalySpec()
	spec.TypeDefinition.LogsAnomaly.AnomalyAlertSettings = &AnomalyAlertSettings{
		PercentageOfDeviation: resource.MustParse("12.5"),
	}

	got := extractProperties(t, spec)
	settings := got.LogsAnomaly.GetAnomalyAlertSettings()
	if settings.GetPercentageOfDeviation() != float32(12.5) {
		t.Fatalf("PercentageOfDeviation = %v, want 12.5", settings.GetPercentageOfDeviation())
	}

	// Unset anomalyAlertSettings => nil.
	got = extractProperties(t, logsAnomalySpec())
	if got.LogsAnomaly.AnomalyAlertSettings != nil {
		t.Fatalf("AnomalyAlertSettings should be nil when unset")
	}
}

// anomalyAlertSettings.percentageOfDeviation on metric-anomaly alerts maps to a float32.
func TestExtractMetricAnomalyAlertSettings(t *testing.T) {
	spec := metricAnomalySpec()
	spec.TypeDefinition.MetricAnomaly.AnomalyAlertSettings = &AnomalyAlertSettings{
		PercentageOfDeviation: resource.MustParse("12.5"),
	}

	got := extractProperties(t, spec)
	settings := got.MetricAnomaly.GetAnomalyAlertSettings()
	if settings.GetPercentageOfDeviation() != float32(12.5) {
		t.Fatalf("PercentageOfDeviation = %v, want 12.5", settings.GetPercentageOfDeviation())
	}

	// Unset anomalyAlertSettings => nil.
	got = extractProperties(t, metricAnomalySpec())
	if got.MetricAnomaly.AnomalyAlertSettings != nil {
		t.Fatalf("AnomalyAlertSettings should be nil when unset")
	}
}

// conditionType on logs-anomaly rule conditions maps to the SDK enum, falling back
// to MORE_THAN_USUAL when unset (CRs created before API-server defaulting).
func TestExtractLogsAnomalyConditionType(t *testing.T) {
	want := alerts.LOGSANOMALYCONDITIONTYPE_LOGS_ANOMALY_CONDITION_TYPE_MORE_THAN_USUAL_OR_UNSPECIFIED

	spec := logsAnomalySpec()
	spec.TypeDefinition.LogsAnomaly.Rules[0].Condition.ConditionType = LogsAnomalyConditionTypeMoreThanUsual
	got := extractProperties(t, spec)
	if got.LogsAnomaly.Rules[0].Condition.GetConditionType() != want {
		t.Fatalf("ConditionType = %v, want MORE_THAN_USUAL_OR_UNSPECIFIED", got.LogsAnomaly.Rules[0].Condition.GetConditionType())
	}

	// Unset conditionType => fallback to the same constant.
	got = extractProperties(t, logsAnomalySpec())
	if got.LogsAnomaly.Rules[0].Condition.GetConditionType() != want {
		t.Fatalf("ConditionType fallback = %v, want MORE_THAN_USUAL_OR_UNSPECIFIED", got.LogsAnomaly.Rules[0].Condition.GetConditionType())
	}
}

// conditionType on tracing-threshold rule conditions maps to the SDK enum, falling back
// to MORE_THAN when unset.
func TestExtractTracingThresholdConditionType(t *testing.T) {
	want := alerts.TRACINGTHRESHOLDCONDITIONTYPE_TRACING_THRESHOLD_CONDITION_TYPE_MORE_THAN_OR_UNSPECIFIED

	spec := tracingThresholdSpec()
	spec.TypeDefinition.TracingThreshold.Rules[0].Condition.ConditionType = TracingThresholdConditionTypeMoreThan
	got := extractProperties(t, spec)
	if got.TracingThreshold.Rules[0].Condition.GetConditionType() != want {
		t.Fatalf("ConditionType = %v, want MORE_THAN_OR_UNSPECIFIED", got.TracingThreshold.Rules[0].Condition.GetConditionType())
	}

	// Unset conditionType => fallback to the same constant.
	got = extractProperties(t, tracingThresholdSpec())
	if got.TracingThreshold.Rules[0].Condition.GetConditionType() != want {
		t.Fatalf("ConditionType fallback = %v, want MORE_THAN_OR_UNSPECIFIED", got.TracingThreshold.Rules[0].Condition.GetConditionType())
	}
}

// retriggeringPeriodMinutes on notification destinations maps to the SDK field.
func TestExtractDestinationRetriggeringPeriodMinutes(t *testing.T) {
	specWithDestination := func(retriggeringPeriodMinutes *int64) AlertSpec {
		spec := logsImmediateSpec()
		spec.NotificationGroup = &NotificationGroup{
			Destinations: []NotificationDestination{
				{
					Connector:                 NCRef{BackendRef: &NCBackendRef{ID: "connector-id"}},
					NotifyOn:                  NotifyOnTriggeredOnly,
					RetriggeringPeriodMinutes: retriggeringPeriodMinutes,
				},
			},
		}
		return spec
	}

	got := extractProperties(t, specWithDestination(ptr.To(int64(10))))
	destination := got.NotificationGroup.Destinations[0]
	if destination.GetRetriggeringPeriodMinutes() != 10 {
		t.Fatalf("RetriggeringPeriodMinutes = %v, want 10", destination.GetRetriggeringPeriodMinutes())
	}

	// Unset retriggeringPeriodMinutes => nil.
	got = extractProperties(t, specWithDestination(nil))
	if got.NotificationGroup.Destinations[0].RetriggeringPeriodMinutes != nil {
		t.Fatalf("RetriggeringPeriodMinutes should be nil when unset")
	}
}
