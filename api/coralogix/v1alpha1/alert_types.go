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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	alerts "github.com/coralogix/coralogix-operator/internal/controller/clientset/grpc/alerts/v2"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

var (
	AlertSchemaSeverityToProtoSeverity = map[AlertSeverity]alerts.AlertSeverity{
		AlertSeverityInfo:     alerts.AlertSeverity_ALERT_SEVERITY_INFO_OR_UNSPECIFIED,
		AlertSeverityWarning:  alerts.AlertSeverity_ALERT_SEVERITY_WARNING,
		AlertSeverityCritical: alerts.AlertSeverity_ALERT_SEVERITY_CRITICAL,
		AlertSeverityError:    alerts.AlertSeverity_ALERT_SEVERITY_ERROR,
		AlertSeverityLow:      alerts.AlertSeverity_ALERT_SEVERITY_LOW,
	}
	AlertSchemaDayToProtoDay = map[Day]alerts.DayOfWeek{
		Sunday:    alerts.DayOfWeek_DAY_OF_WEEK_SUNDAY,
		Monday:    alerts.DayOfWeek_DAY_OF_WEEK_MONDAY_OR_UNSPECIFIED,
		Tuesday:   alerts.DayOfWeek_DAY_OF_WEEK_TUESDAY,
		Wednesday: alerts.DayOfWeek_DAY_OF_WEEK_WEDNESDAY,
		Thursday:  alerts.DayOfWeek_DAY_OF_WEEK_THURSDAY,
		Friday:    alerts.DayOfWeek_DAY_OF_WEEK_FRIDAY,
		Saturday:  alerts.DayOfWeek_DAY_OF_WEEK_SATURDAY,
	}
	AlertSchemaTimeWindowToProtoTimeWindow = map[string]alerts.Timeframe{
		"Minute":          alerts.Timeframe_TIMEFRAME_1_MIN,
		"FiveMinutes":     alerts.Timeframe_TIMEFRAME_5_MIN_OR_UNSPECIFIED,
		"TenMinutes":      alerts.Timeframe_TIMEFRAME_10_MIN,
		"FifteenMinutes":  alerts.Timeframe_TIMEFRAME_15_MIN,
		"TwentyMinutes":   alerts.Timeframe_TIMEFRAME_20_MIN,
		"ThirtyMinutes":   alerts.Timeframe_TIMEFRAME_30_MIN,
		"Hour":            alerts.Timeframe_TIMEFRAME_1_H,
		"TwoHours":        alerts.Timeframe_TIMEFRAME_2_H,
		"FourHours":       alerts.Timeframe_TIMEFRAME_4_H,
		"SixHours":        alerts.Timeframe_TIMEFRAME_6_H,
		"TwelveHours":     alerts.Timeframe_TIMEFRAME_12_H,
		"TwentyFourHours": alerts.Timeframe_TIMEFRAME_24_H,
		"ThirtySixHours":  alerts.Timeframe_TIMEFRAME_36_H,
	}
	AlertSchemaAutoRetireRatioToProtoAutoRetireRatio = map[AutoRetireRatio]alerts.CleanupDeadmanDuration{
		AutoRetireRatioNever:           alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_NEVER_OR_UNSPECIFIED,
		AutoRetireRatioFiveMinutes:     alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_5MIN,
		AutoRetireRatioTenMinutes:      alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_10MIN,
		AutoRetireRatioHour:            alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_1H,
		AutoRetireRatioTwoHours:        alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_2H,
		AutoRetireRatioSixHours:        alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_6H,
		AutoRetireRatioTwelveHours:     alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_12H,
		AutoRetireRatioTwentyFourHours: alerts.CleanupDeadmanDuration_CLEANUP_DEADMAN_DURATION_24H,
	}
	AlertSchemaFiltersLogSeverityToProtoFiltersLogSeverity = map[FiltersLogSeverity]alerts.AlertFilters_LogSeverity{
		FiltersLogSeverityDebug:    alerts.AlertFilters_LOG_SEVERITY_DEBUG_OR_UNSPECIFIED,
		FiltersLogSeverityVerbose:  alerts.AlertFilters_LOG_SEVERITY_VERBOSE,
		FiltersLogSeverityInfo:     alerts.AlertFilters_LOG_SEVERITY_INFO,
		FiltersLogSeverityWarning:  alerts.AlertFilters_LOG_SEVERITY_WARNING,
		FiltersLogSeverityCritical: alerts.AlertFilters_LOG_SEVERITY_CRITICAL,
		FiltersLogSeverityError:    alerts.AlertFilters_LOG_SEVERITY_ERROR,
	}
	AlertSchemaRelativeTimeFrameToProtoTimeFrameAndRelativeTimeFrame = map[RelativeTimeWindow]ProtoTimeFrameAndRelativeTimeFrame{
		RelativeTimeWindowPreviousHour:      {TimeFrame: alerts.Timeframe_TIMEFRAME_1_H, RelativeTimeFrame: alerts.RelativeTimeframe_RELATIVE_TIMEFRAME_HOUR_OR_UNSPECIFIED},
		RelativeTimeWindowSameHourYesterday: {TimeFrame: alerts.Timeframe_TIMEFRAME_1_H, RelativeTimeFrame: alerts.RelativeTimeframe_RELATIVE_TIMEFRAME_DAY},
		RelativeTimeWindowSameHourLastWeek:  {TimeFrame: alerts.Timeframe_TIMEFRAME_1_H, RelativeTimeFrame: alerts.RelativeTimeframe_RELATIVE_TIMEFRAME_WEEK},
		RelativeTimeWindowYesterday:         {TimeFrame: alerts.Timeframe_TIMEFRAME_24_H, RelativeTimeFrame: alerts.RelativeTimeframe_RELATIVE_TIMEFRAME_DAY},
		RelativeTimeWindowSameDayLastWeek:   {TimeFrame: alerts.Timeframe_TIMEFRAME_24_H, RelativeTimeFrame: alerts.RelativeTimeframe_RELATIVE_TIMEFRAME_WEEK},
		RelativeTimeWindowSameDayLastMonth:  {TimeFrame: alerts.Timeframe_TIMEFRAME_24_H, RelativeTimeFrame: alerts.RelativeTimeframe_RELATIVE_TIMEFRAME_MONTH},
	}
	AlertSchemaArithmeticOperatorToProtoArithmeticOperator = map[ArithmeticOperator]alerts.MetricAlertConditionParameters_ArithmeticOperator{
		ArithmeticOperatorAvg:        alerts.MetricAlertConditionParameters_ARITHMETIC_OPERATOR_AVG_OR_UNSPECIFIED,
		ArithmeticOperatorMin:        alerts.MetricAlertConditionParameters_ARITHMETIC_OPERATOR_MIN,
		ArithmeticOperatorMax:        alerts.MetricAlertConditionParameters_ARITHMETIC_OPERATOR_MAX,
		ArithmeticOperatorSum:        alerts.MetricAlertConditionParameters_ARITHMETIC_OPERATOR_SUM,
		ArithmeticOperatorCount:      alerts.MetricAlertConditionParameters_ARITHMETIC_OPERATOR_COUNT,
		ArithmeticOperatorPercentile: alerts.MetricAlertConditionParameters_ARITHMETIC_OPERATOR_PERCENTILE,
	}
	AlertSchemaFlowOperatorToProtoFlowOperator = map[FlowOperator]alerts.FlowOperator{
		"And": alerts.FlowOperator_AND,
		"Or":  alerts.FlowOperator_OR,
	}
	AlertSchemaNotifyOnToProtoNotifyOn = map[NotifyOn]alerts.NotifyOn{
		NotifyOnTriggeredOnly:        alerts.NotifyOn_TRIGGERED_ONLY,
		NotifyOnTriggeredAndResolved: alerts.NotifyOn_TRIGGERED_AND_RESOLVED,
	}
	msInHour       = int(time.Hour.Milliseconds())
	msInMinute     = int(time.Minute.Milliseconds())
	WebhooksClient clientset.OutboundWebhooksClientInterface
)

type ProtoTimeFrameAndRelativeTimeFrame struct {
	TimeFrame         alerts.Timeframe
	RelativeTimeFrame alerts.RelativeTimeframe
}

// AlertSpec defines the desired state of Alert
type AlertSpec struct {
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// +optional
	Description string `json:"description,omitempty"`

	//+kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	Severity AlertSeverity `json:"severity"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +optional
	ExpirationDate *ExpirationDate `json:"expirationDate,omitempty"`

	// +optional
	NotificationGroups []NotificationGroup `json:"notificationGroups,omitempty"`

	// +optional
	ShowInInsight *ShowInInsight `json:"showInInsight,omitempty"`

	// +optional
	PayloadFilters []string `json:"payloadFilters,omitempty"`

	// +optional
	Scheduling *Scheduling `json:"scheduling,omitempty"`

	AlertType AlertType `json:"alertType"`
}

func (a *Alert) ExtractCreateAlertRequest(ctx context.Context, log logr.Logger) (*alerts.CreateAlertRequest, error) {
	notificationGroups, err := expandNotificationGroups(ctx, log, a.Spec.NotificationGroups)
	if err != nil {
		return nil, err
	}

	return &alerts.CreateAlertRequest{
		IsActive:                   wrapperspb.Bool(a.Spec.Active),
		Name:                       wrapperspb.String(a.Spec.Name),
		Description:                wrapperspb.String(a.Spec.Description),
		Severity:                   AlertSchemaSeverityToProtoSeverity[a.Spec.Severity],
		MetaLabels:                 expandMetaLabels(a.Spec.Labels),
		Expiration:                 expandExpirationDate(a.Spec.ExpirationDate),
		ShowInInsight:              expandShowInInsight(a.Spec.ShowInInsight),
		NotificationGroups:         notificationGroups,
		NotificationPayloadFilters: utils.StringSliceToWrappedStringSlice(a.Spec.PayloadFilters),
		ActiveWhen:                 expandActiveWhen(a.Spec.Scheduling),
		Filters:                    expandAlertType(a.Spec.AlertType).filters,
		Condition:                  expandAlertType(a.Spec.AlertType).condition,
		TracingAlert:               expandAlertType(a.Spec.AlertType).tracingAlert,
	}, nil
}

type alertTypeParams struct {
	filters      *alerts.AlertFilters
	condition    *alerts.AlertCondition
	tracingAlert *alerts.TracingAlert
}

func expandAlertType(alertType AlertType) alertTypeParams {
	if standard := alertType.Standard; standard != nil {
		return expandStandard(standard)
	} else if ratio := alertType.Ratio; ratio != nil {
		return expandRatio(ratio)
	} else if newValue := alertType.NewValue; newValue != nil {
		return expandNewValue(newValue)
	} else if uniqueCount := alertType.UniqueCount; uniqueCount != nil {
		return expandUniqueCount(uniqueCount)
	} else if timeRelative := alertType.TimeRelative; timeRelative != nil {
		return expandTimeRelative(timeRelative)
	} else if metric := alertType.Metric; metric != nil {
		return expandMetric(metric)
	} else if tracing := alertType.Tracing; tracing != nil {
		return expandTracing(tracing)
	} else if flow := alertType.Flow; flow != nil {
		return expandFlow(flow)
	}

	return alertTypeParams{}
}

func expandStandard(standard *Standard) alertTypeParams {
	condition := expandStandardCondition(standard.Conditions)
	filters := expandCommonFilters(standard.Filters)
	filters.FilterType = alerts.AlertFilters_FILTER_TYPE_TEXT_OR_UNSPECIFIED
	return alertTypeParams{
		condition: condition,
		filters:   filters,
	}
}

func expandRatio(ratio *Ratio) alertTypeParams {
	groupBy := utils.StringSliceToWrappedStringSlice(ratio.Conditions.GroupBy)
	var groupByQ1, groupByQ2 []*wrapperspb.StringValue
	if groupByFor := ratio.Conditions.GroupByFor; groupByFor != nil {
		switch *groupByFor {
		case GroupByForQ1:
			groupByQ1 = groupBy
		case GroupByForQ2:
			groupByQ2 = groupBy
		case GroupByForBoth:
			groupByQ1 = groupBy
			groupByQ2 = groupBy
		}
	}

	condition := expandRatioCondition(ratio.Conditions, groupByQ1)
	filters := expandRatioFilters(&ratio.Query1Filters, &ratio.Query2Filters, groupByQ2)

	return alertTypeParams{
		condition: condition,
		filters:   filters,
	}
}

func expandRatioCondition(conditions RatioConditions, q1GroupBy []*wrapperspb.StringValue) *alerts.AlertCondition {
	threshold := wrapperspb.Double(conditions.Ratio.AsApproximateFloat64())
	timeFrame := AlertSchemaTimeWindowToProtoTimeWindow[string(conditions.TimeWindow)]
	ignoreInfinity := wrapperspb.Bool(conditions.IgnoreInfinity)
	relatedExtendedData := expandRelatedData(conditions.ManageUndetectedValues)

	parameters := &alerts.ConditionParameters{
		Threshold:           threshold,
		Timeframe:           timeFrame,
		GroupBy:             q1GroupBy,
		IgnoreInfinity:      ignoreInfinity,
		RelatedExtendedData: relatedExtendedData,
	}

	switch conditions.AlertWhen {
	case "More":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThan{
				MoreThan: &alerts.MoreThanCondition{Parameters: parameters},
			},
		}
	case "Less":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_LessThan{
				LessThan: &alerts.LessThanCondition{Parameters: parameters},
			},
		}
	}

	return nil
}

func expandRatioFilters(q1Filters *Filters, q2Filters *RatioQ2Filters, groupByQ2 []*wrapperspb.StringValue) *alerts.AlertFilters {
	filters := expandCommonFilters(q1Filters)
	if q1Alias := q1Filters.Alias; q1Alias != nil {
		filters.Alias = wrapperspb.String(*q1Alias)
	}
	q2 := expandQ2Filters(q2Filters, groupByQ2)
	filters.RatioAlerts = []*alerts.AlertFilters_RatioAlert{q2}
	filters.FilterType = alerts.AlertFilters_FILTER_TYPE_RATIO
	return filters
}

func expandQ2Filters(q2Filters *RatioQ2Filters, q2GroupBy []*wrapperspb.StringValue) *alerts.AlertFilters_RatioAlert {
	var text *wrapperspb.StringValue
	if searchQuery := q2Filters.SearchQuery; searchQuery != nil {
		text = wrapperspb.String(*searchQuery)
	}

	var alias *wrapperspb.StringValue
	if desiredAlias := q2Filters.Alias; desiredAlias != nil {
		alias = wrapperspb.String(*desiredAlias)
	}
	severities := expandAlertFiltersSeverities(q2Filters.Severities)
	applications := utils.StringSliceToWrappedStringSlice(q2Filters.Applications)
	subsystems := utils.StringSliceToWrappedStringSlice(q2Filters.Subsystems)

	return &alerts.AlertFilters_RatioAlert{
		Alias:        alias,
		Text:         text,
		Severities:   severities,
		Applications: applications,
		Subsystems:   subsystems,
		GroupBy:      q2GroupBy,
	}
}

func expandNewValue(newValue *NewValue) alertTypeParams {
	condition := expandNewValueCondition(&newValue.Conditions)
	filters := expandCommonFilters(newValue.Filters)
	filters.FilterType = alerts.AlertFilters_FILTER_TYPE_TEXT_OR_UNSPECIFIED
	return alertTypeParams{
		condition: condition,
		filters:   filters,
	}
}

func expandNewValueCondition(conditions *NewValueConditions) *alerts.AlertCondition {
	timeFrame := AlertSchemaTimeWindowToProtoTimeWindow[string(conditions.TimeWindow)]
	groupBy := []*wrapperspb.StringValue{wrapperspb.String(conditions.Key)}
	parameters := &alerts.ConditionParameters{
		Timeframe: timeFrame,
		GroupBy:   groupBy,
	}

	return &alerts.AlertCondition{
		Condition: &alerts.AlertCondition_NewValue{
			NewValue: &alerts.NewValueCondition{
				Parameters: parameters,
			},
		},
	}
}

func expandUniqueCount(uniqueCount *UniqueCount) alertTypeParams {
	condition := expandUniqueCountCondition(&uniqueCount.Conditions)
	filters := expandCommonFilters(uniqueCount.Filters)
	filters.FilterType = alerts.AlertFilters_FILTER_TYPE_UNIQUE_COUNT
	return alertTypeParams{
		condition: condition,
		filters:   filters,
	}
}

func expandUniqueCountCondition(conditions *UniqueCountConditions) *alerts.AlertCondition {
	uniqueCountKey := []*wrapperspb.StringValue{wrapperspb.String(conditions.Key)}
	threshold := wrapperspb.Double(float64(conditions.MaxUniqueValues))
	timeFrame := AlertSchemaTimeWindowToProtoTimeWindow[string(conditions.TimeWindow)]
	var groupBy []*wrapperspb.StringValue
	var maxUniqueValuesForGroupBy *wrapperspb.UInt32Value
	if groupByKey := conditions.GroupBy; groupByKey != nil {
		groupBy = []*wrapperspb.StringValue{wrapperspb.String(*groupByKey)}
		maxUniqueValuesForGroupBy = wrapperspb.UInt32(uint32(*conditions.MaxUniqueValuesForGroupBy))
	}

	parameters := &alerts.ConditionParameters{
		CardinalityFields:                 uniqueCountKey,
		Threshold:                         threshold,
		Timeframe:                         timeFrame,
		GroupBy:                           groupBy,
		MaxUniqueCountValuesForGroupByKey: maxUniqueValuesForGroupBy,
	}

	return &alerts.AlertCondition{
		Condition: &alerts.AlertCondition_UniqueCount{
			UniqueCount: &alerts.UniqueCountCondition{
				Parameters: parameters,
			},
		},
	}
}

func expandTimeRelative(timeRelative *TimeRelative) alertTypeParams {
	condition := expandTimeRelativeCondition(&timeRelative.Conditions)
	filters := expandCommonFilters(timeRelative.Filters)
	filters.FilterType = alerts.AlertFilters_FILTER_TYPE_TIME_RELATIVE
	return alertTypeParams{
		condition: condition,
		filters:   filters,
	}
}

func expandTimeRelativeCondition(condition *TimeRelativeConditions) *alerts.AlertCondition {
	threshold := wrapperspb.Double(condition.Threshold.AsApproximateFloat64())
	timeFrameAndRelativeTimeFrame := AlertSchemaRelativeTimeFrameToProtoTimeFrameAndRelativeTimeFrame[condition.TimeWindow]
	groupBy := utils.StringSliceToWrappedStringSlice(condition.GroupBy)
	ignoreInf := wrapperspb.Bool(condition.IgnoreInfinity)
	relatedExtendedData := expandRelatedData(condition.ManageUndetectedValues)

	parameters := &alerts.ConditionParameters{
		Timeframe:           timeFrameAndRelativeTimeFrame.TimeFrame,
		RelativeTimeframe:   timeFrameAndRelativeTimeFrame.RelativeTimeFrame,
		GroupBy:             groupBy,
		Threshold:           threshold,
		IgnoreInfinity:      ignoreInf,
		RelatedExtendedData: relatedExtendedData,
	}

	switch condition.AlertWhen {
	case "More":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThan{
				MoreThan: &alerts.MoreThanCondition{Parameters: parameters},
			},
		}
	case "Less":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_LessThan{
				LessThan: &alerts.LessThanCondition{Parameters: parameters},
			},
		}
	}

	return nil
}

func expandMetric(metric *Metric) alertTypeParams {
	if promql := metric.Promql; promql != nil {
		return expandPromql(promql)
	} else if lucene := metric.Lucene; lucene != nil {
		return expandLucene(lucene)
	}

	return alertTypeParams{}
}

func expandPromql(promql *Promql) alertTypeParams {
	condition := expandPromqlCondition(&promql.Conditions, promql.SearchQuery)
	filters := &alerts.AlertFilters{
		FilterType: alerts.AlertFilters_FILTER_TYPE_METRIC,
	}

	return alertTypeParams{
		condition: condition,
		filters:   filters,
	}
}

func expandPromqlCondition(conditions *PromqlConditions, searchQuery string) *alerts.AlertCondition {
	text := wrapperspb.String(searchQuery)
	sampleThresholdPercentage := wrapperspb.UInt32(uint32(conditions.SampleThresholdPercentage))
	var nonNullPercentage *wrapperspb.UInt32Value
	if minNonNullValuesPercentage := conditions.MinNonNullValuesPercentage; minNonNullValuesPercentage != nil {
		nonNullPercentage = wrapperspb.UInt32(uint32(*minNonNullValuesPercentage))
	}
	swapNullValues := wrapperspb.Bool(conditions.ReplaceMissingValueWithZero)
	promqlParams := &alerts.MetricAlertPromqlConditionParameters{
		PromqlText:                text,
		SampleThresholdPercentage: sampleThresholdPercentage,
		NonNullPercentage:         nonNullPercentage,
		SwapNullValues:            swapNullValues,
	}
	threshold := wrapperspb.Double(conditions.Threshold.AsApproximateFloat64())
	timeWindow := AlertSchemaTimeWindowToProtoTimeWindow[string(conditions.TimeWindow)]
	relatedExtendedData := expandRelatedData(conditions.ManageUndetectedValues)

	parameters := &alerts.ConditionParameters{
		Threshold:                   threshold,
		Timeframe:                   timeWindow,
		RelatedExtendedData:         relatedExtendedData,
		MetricAlertPromqlParameters: promqlParams,
	}

	switch conditions.AlertWhen {
	case PromqlAlertWhenMoreThan:
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThan{
				MoreThan: &alerts.MoreThanCondition{Parameters: parameters},
			},
		}
	case PromqlAlertWhenLessThan:
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_LessThan{
				LessThan: &alerts.LessThanCondition{Parameters: parameters},
			},
		}
	case PromqlAlertWhenMoreThanUsual:
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThanUsual{
				MoreThanUsual: &alerts.MoreThanUsualCondition{Parameters: parameters},
			},
		}
	}

	return nil
}

func expandLucene(lucene *Lucene) alertTypeParams {
	condition := expandLuceneCondition(&lucene.Conditions)
	var text *wrapperspb.StringValue
	if searchQuery := lucene.SearchQuery; searchQuery != nil {
		text = wrapperspb.String(*searchQuery)
	}

	filters := &alerts.AlertFilters{
		FilterType: alerts.AlertFilters_FILTER_TYPE_METRIC,
		Text:       text,
	}

	return alertTypeParams{
		condition: condition,
		filters:   filters,
	}
}

func expandLuceneCondition(conditions *LuceneConditions) *alerts.AlertCondition {
	metricField := wrapperspb.String(conditions.MetricField)
	arithmeticOperator := AlertSchemaArithmeticOperatorToProtoArithmeticOperator[conditions.ArithmeticOperator]
	var arithmeticOperatorModifier *wrapperspb.UInt32Value
	if modifier := conditions.ArithmeticOperatorModifier; modifier != nil {
		arithmeticOperatorModifier = wrapperspb.UInt32(uint32(*modifier))
	}
	sampleThresholdPercentage := wrapperspb.UInt32(uint32(conditions.SampleThresholdPercentage))
	swapNullValues := wrapperspb.Bool(conditions.ReplaceMissingValueWithZero)
	nonNullPercentage := wrapperspb.UInt32(uint32(conditions.MinNonNullValuesPercentage))

	luceneParams := &alerts.MetricAlertConditionParameters{
		MetricSource:               alerts.MetricAlertConditionParameters_METRIC_SOURCE_LOGS2METRICS_OR_UNSPECIFIED,
		MetricField:                metricField,
		ArithmeticOperator:         arithmeticOperator,
		ArithmeticOperatorModifier: arithmeticOperatorModifier,
		SampleThresholdPercentage:  sampleThresholdPercentage,
		NonNullPercentage:          nonNullPercentage,
		SwapNullValues:             swapNullValues,
	}

	groupBy := utils.StringSliceToWrappedStringSlice(conditions.GroupBy)
	threshold := wrapperspb.Double(conditions.Threshold.AsApproximateFloat64())
	timeWindow := AlertSchemaTimeWindowToProtoTimeWindow[string(conditions.TimeWindow)]
	relatedExtendedData := expandRelatedData(conditions.ManageUndetectedValues)

	parameters := &alerts.ConditionParameters{
		GroupBy:               groupBy,
		Threshold:             threshold,
		Timeframe:             timeWindow,
		RelatedExtendedData:   relatedExtendedData,
		MetricAlertParameters: luceneParams,
	}

	switch conditions.AlertWhen {
	case "More":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThan{
				MoreThan: &alerts.MoreThanCondition{Parameters: parameters},
			},
		}
	case "Less":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_LessThan{
				LessThan: &alerts.LessThanCondition{Parameters: parameters},
			},
		}
	}

	return nil
}

func expandTracing(tracing *Tracing) alertTypeParams {
	filters := &alerts.AlertFilters{
		FilterType: alerts.AlertFilters_FILTER_TYPE_TRACING,
	}
	condition := expandTracingCondition(&tracing.Conditions)
	tracingAlert := expandTracingAlert(&tracing.Filters)
	return alertTypeParams{
		filters:      filters,
		condition:    condition,
		tracingAlert: tracingAlert,
	}
}

func expandTracingCondition(conditions *TracingCondition) *alerts.AlertCondition {
	switch conditions.AlertWhen {
	case "More":
		var timeFrame alerts.Timeframe
		if timeWindow := conditions.TimeWindow; timeWindow != nil {
			timeFrame = AlertSchemaTimeWindowToProtoTimeWindow[string(*timeWindow)]
		}
		groupBy := utils.StringSliceToWrappedStringSlice(conditions.GroupBy)
		threshold := wrapperspb.Double(float64(*conditions.Threshold))
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThan{
				MoreThan: &alerts.MoreThanCondition{
					Parameters: &alerts.ConditionParameters{
						Timeframe: timeFrame,
						Threshold: threshold,
						GroupBy:   groupBy,
					},
				},
			},
		}
	case "Immediately":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_Immediate{},
		}
	}

	return nil
}

func expandTracingAlert(tracingFilters *TracingFilters) *alerts.TracingAlert {
	conditionLatency := uint32(tracingFilters.LatencyThresholdMilliseconds.AsApproximateFloat64() * float64(time.Millisecond.Microseconds()))
	fieldFilters := expandFiltersData(tracingFilters.Applications, tracingFilters.Subsystems, tracingFilters.Services)
	tagFilters := expandTagFilters(tracingFilters.TagFilters)
	return &alerts.TracingAlert{
		ConditionLatency: conditionLatency,
		FieldFilters:     fieldFilters,
		TagFilters:       tagFilters,
	}
}

func expandFiltersData(applications, subsystems, services []string) []*alerts.FilterData {
	result := make([]*alerts.FilterData, 0)
	if len(applications) != 0 {
		result = append(result, expandSpecificFilter("applicationName", applications))
	}
	if len(subsystems) != 0 {
		result = append(result, expandSpecificFilter("subsystemName", subsystems))
	}
	if len(services) != 0 {
		result = append(result, expandSpecificFilter("serviceName", services))
	}

	return result
}

func expandTagFilters(tagFilters []TagFilter) []*alerts.FilterData {
	result := make([]*alerts.FilterData, 0, len(tagFilters))
	for _, tagFilter := range tagFilters {
		result = append(result, expandSpecificFilter(tagFilter.Field, tagFilter.Values))
	}
	return result
}

func expandSpecificFilter(filterName string, values []string) *alerts.FilterData {
	operatorToFilterValues := make(map[string]*alerts.Filters)
	for _, val := range values {
		operator, filterValue := expandFilter(val)
		if _, ok := operatorToFilterValues[operator]; !ok {
			operatorToFilterValues[operator] = new(alerts.Filters)
			operatorToFilterValues[operator].Operator = operator
			operatorToFilterValues[operator].Values = make([]string, 0)
		}
		operatorToFilterValues[operator].Values = append(operatorToFilterValues[operator].Values, filterValue)
	}

	filterResult := make([]*alerts.Filters, 0, len(operatorToFilterValues))
	for _, filters := range operatorToFilterValues {
		filterResult = append(filterResult, filters)
	}

	return &alerts.FilterData{
		Field:   filterName,
		Filters: filterResult,
	}
}

func expandFilter(filterString string) (operator, filterValue string) {
	operator, filterValue = "equals", filterString
	if strings.HasPrefix(filterValue, "filter:") {
		arr := strings.SplitN(filterValue, ":", 3)
		operator, filterValue = arr[1], arr[2]
	}

	return
}

func expandFlow(flow *Flow) alertTypeParams {
	stages := expandFlowStages(flow.Stages)
	return alertTypeParams{
		condition: &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_Flow{
				Flow: &alerts.FlowCondition{
					Stages: stages,
				},
			},
		},
		filters: &alerts.AlertFilters{
			FilterType: alerts.AlertFilters_FILTER_TYPE_FLOW,
		},
	}
}

func expandFlowStages(stages []FlowStage) []*alerts.FlowStage {
	result := make([]*alerts.FlowStage, 0, len(stages))
	for _, s := range stages {
		stage := expandFlowStage(s)
		result = append(result, stage)
	}
	return result
}

func expandFlowStage(stage FlowStage) *alerts.FlowStage {
	groups := expandFlowStageGroups(stage.Groups)
	var timeFrame *alerts.FlowTimeframe
	if timeWindow := stage.TimeWindow; timeWindow != nil {
		timeFrame = new(alerts.FlowTimeframe)
		timeFrame.Ms = wrapperspb.UInt32(uint32(expandTimeToMS(*timeWindow)))
	}

	return &alerts.FlowStage{
		Groups:    groups,
		Timeframe: timeFrame,
	}
}

func expandTimeToMS(t FlowStageTimeFrame) int {
	timeMS := msInHour * t.Hours
	timeMS += msInMinute * t.Minutes

	return timeMS
}

func expandFlowStageGroups(groups []FlowStageGroup) []*alerts.FlowGroup {
	result := make([]*alerts.FlowGroup, 0, len(groups))
	for _, g := range groups {
		group := expandFlowStageGroup(g)
		result = append(result, group)
	}
	return result
}

func expandFlowStageGroup(group FlowStageGroup) *alerts.FlowGroup {
	subAlerts := expandFlowSubgroupAlerts(group.InnerFlowAlerts)
	nextOp := AlertSchemaFlowOperatorToProtoFlowOperator[group.NextOperator]
	return &alerts.FlowGroup{
		Alerts: subAlerts,
		NextOp: nextOp,
	}
}

func expandFlowSubgroupAlerts(subgroup InnerFlowAlerts) *alerts.FlowAlerts {
	return &alerts.FlowAlerts{
		Op:     AlertSchemaFlowOperatorToProtoFlowOperator[subgroup.Operator],
		Values: expandFlowInnerAlerts(subgroup.Alerts),
	}
}

func expandFlowInnerAlerts(innerAlerts []InnerFlowAlert) []*alerts.FlowAlert {
	result := make([]*alerts.FlowAlert, 0, len(innerAlerts))
	for _, a := range innerAlerts {
		alert := expandFlowInnerAlert(a)
		result = append(result, alert)
	}
	return result
}

func expandFlowInnerAlert(alert InnerFlowAlert) *alerts.FlowAlert {
	return &alerts.FlowAlert{
		Id:  wrapperspb.String(alert.UserAlertId),
		Not: wrapperspb.Bool(alert.Not),
	}
}

func expandCommonFilters(filters *Filters) *alerts.AlertFilters {
	severities := expandAlertFiltersSeverities(filters.Severities)
	metadata := expandMetadata(filters)
	var text *wrapperspb.StringValue
	if searchQuery := filters.SearchQuery; searchQuery != nil {
		text = wrapperspb.String(*searchQuery)
	}
	return &alerts.AlertFilters{
		Severities: severities,
		Metadata:   metadata,
		Text:       text,
	}
}

func expandAlertFiltersSeverities(severities []FiltersLogSeverity) []alerts.AlertFilters_LogSeverity {
	result := make([]alerts.AlertFilters_LogSeverity, 0, len(severities))
	for _, s := range severities {
		severity := AlertSchemaFiltersLogSeverityToProtoFiltersLogSeverity[s]
		result = append(result, severity)
	}
	return result
}

func expandMetadata(filters *Filters) *alerts.AlertFilters_MetadataFilters {
	categories := utils.StringSliceToWrappedStringSlice(filters.Categories)
	applications := utils.StringSliceToWrappedStringSlice(filters.Applications)
	subsystems := utils.StringSliceToWrappedStringSlice(filters.Subsystems)
	ips := utils.StringSliceToWrappedStringSlice(filters.IPs)
	classes := utils.StringSliceToWrappedStringSlice(filters.Classes)
	methods := utils.StringSliceToWrappedStringSlice(filters.Methods)
	computers := utils.StringSliceToWrappedStringSlice(filters.Computers)
	return &alerts.AlertFilters_MetadataFilters{
		Categories:   categories,
		Applications: applications,
		Subsystems:   subsystems,
		IpAddresses:  ips,
		Classes:      classes,
		Methods:      methods,
		Computers:    computers,
	}
}

func expandStandardCondition(condition StandardConditions) *alerts.AlertCondition {
	var threshold *wrapperspb.DoubleValue
	if condition.Threshold != nil {
		threshold = wrapperspb.Double(float64(*condition.Threshold))
	}
	var timeFrame alerts.Timeframe
	if condition.TimeWindow != nil {
		timeFrame = AlertSchemaTimeWindowToProtoTimeWindow[string(*condition.TimeWindow)]
	}
	groupBy := utils.StringSliceToWrappedStringSlice(condition.GroupBy)
	relatedExtendedData := expandRelatedData(condition.ManageUndetectedValues)

	parameters := &alerts.ConditionParameters{
		Threshold:           threshold,
		Timeframe:           timeFrame,
		GroupBy:             groupBy,
		RelatedExtendedData: relatedExtendedData,
	}

	switch condition.AlertWhen {
	case "More":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThan{
				MoreThan: &alerts.MoreThanCondition{Parameters: parameters},
			},
		}
	case "Less":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_LessThan{
				LessThan: &alerts.LessThanCondition{Parameters: parameters},
			},
		}
	case "Immediately":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_Immediate{},
		}
	case "MoreThanUsual":
		return &alerts.AlertCondition{
			Condition: &alerts.AlertCondition_MoreThanUsual{
				MoreThanUsual: &alerts.MoreThanUsualCondition{Parameters: parameters},
			},
		}
	}

	return nil
}

func expandRelatedData(manageUndetectedValues *ManageUndetectedValues) *alerts.RelatedExtendedData {
	if manageUndetectedValues != nil {
		shouldTriggerDeadman := wrapperspb.Bool(manageUndetectedValues.EnableTriggeringOnUndetectedValues)
		cleanupDeadmanDuration := AlertSchemaAutoRetireRatioToProtoAutoRetireRatio[*manageUndetectedValues.AutoRetireRatio]
		return &alerts.RelatedExtendedData{
			ShouldTriggerDeadman:   shouldTriggerDeadman,
			CleanupDeadmanDuration: &cleanupDeadmanDuration,
		}
	}
	return nil
}

func expandActiveWhen(scheduling *Scheduling) *alerts.AlertActiveWhen {
	if scheduling == nil {
		return nil
	}

	timeFrames := expandTimeFrames(scheduling)

	return &alerts.AlertActiveWhen{
		Timeframes: timeFrames,
	}
}

func expandTimeFrames(scheduling *Scheduling) []*alerts.AlertActiveTimeframe {
	utc := ExtractUTC(scheduling.TimeZone)
	daysOfWeek := expandDaysOfWeek(scheduling.DaysEnabled)
	start := expandTime(scheduling.StartTime)
	end := expandTime(scheduling.EndTime)
	timeRange := &alerts.TimeRange{
		Start: start,
		End:   end,
	}
	timeRange, daysOfWeek = convertTimeFramesToGMT(timeRange, daysOfWeek, utc)

	alertActiveTimeframe := &alerts.AlertActiveTimeframe{
		DaysOfWeek: daysOfWeek,
		Range:      timeRange,
	}

	return []*alerts.AlertActiveTimeframe{
		alertActiveTimeframe,
	}
}

func ExtractUTC(timeZone TimeZone) int32 {
	parts := strings.Split(string(timeZone), "UTC")
	if len(parts) < 2 {
		return 0
	}
	utcStr := parts[1]
	utc, err := strconv.Atoi(utcStr)
	if err != nil {
		return 0
	}
	return int32(utc)
}

func expandTime(time *Time) *alerts.Time {
	if time == nil {
		return nil
	}

	timeArr := strings.Split(string(*time), ":")
	hours, _ := strconv.Atoi(timeArr[0])
	minutes, _ := strconv.Atoi(timeArr[1])

	return &alerts.Time{
		Hours:   int32(hours),
		Minutes: int32(minutes),
	}
}

func expandDaysOfWeek(days []Day) []alerts.DayOfWeek {
	daysOfWeek := make([]alerts.DayOfWeek, 0, len(days))
	for _, d := range days {
		daysOfWeek = append(daysOfWeek, AlertSchemaDayToProtoDay[d])
	}
	return daysOfWeek
}

func convertTimeFramesToGMT(frameRange *alerts.TimeRange, daysOfWeek []alerts.DayOfWeek, utc int32) (*alerts.TimeRange, []alerts.DayOfWeek) {
	daysOfWeekOffset := daysOfWeekOffsetToGMT(frameRange, utc)
	frameRange.Start.Hours = convertUtcToGmt(frameRange.GetStart().GetHours(), utc)
	frameRange.End.Hours = convertUtcToGmt(frameRange.GetEnd().GetHours(), utc)
	if daysOfWeekOffset != 0 {
		for i, d := range daysOfWeek {
			daysOfWeek[i] = alerts.DayOfWeek((int32(d) + daysOfWeekOffset) % 7)
		}
	}

	return frameRange, daysOfWeek
}

func daysOfWeekOffsetToGMT(frameRange *alerts.TimeRange, utc int32) int32 {
	daysOfWeekOffset := int32(frameRange.Start.Hours-utc) / 24
	if daysOfWeekOffset < 0 {
		daysOfWeekOffset += 7
	}
	return daysOfWeekOffset
}

func convertUtcToGmt(hours, utc int32) int32 {
	hours -= utc
	if hours < 0 {
		hours += 24
	} else if hours >= 24 {
		hours -= 24
	}

	return hours
}

func expandMetaLabels(labels map[string]string) []*alerts.MetaLabel {
	result := make([]*alerts.MetaLabel, 0)
	for k, v := range labels {
		result = append(result, &alerts.MetaLabel{
			Key:   wrapperspb.String(k),
			Value: wrapperspb.String(v),
		})
	}
	return result
}

func expandExpirationDate(date *ExpirationDate) *alerts.Date {
	if date == nil {
		return nil
	}

	return &alerts.Date{
		Year:  date.Year,
		Month: date.Month,
		Day:   date.Day,
	}
}

func expandShowInInsight(showInInsight *ShowInInsight) *alerts.ShowInInsight {
	if showInInsight == nil {
		return nil
	}

	retriggeringPeriodSeconds := wrapperspb.UInt32(uint32(showInInsight.RetriggeringPeriodMinutes) * 60)
	notifyOn := AlertSchemaNotifyOnToProtoNotifyOn[showInInsight.NotifyOn]

	return &alerts.ShowInInsight{
		RetriggeringPeriodSeconds: retriggeringPeriodSeconds,
		NotifyOn:                  &notifyOn,
	}
}

func expandNotificationGroups(ctx context.Context, log logr.Logger, notificationGroups []NotificationGroup) ([]*alerts.AlertNotificationGroups, error) {
	webhooksNamesToIds, err := getWebhooksNamesToIds(ctx, log)
	if err != nil {
		return nil, err
	}
	result := make([]*alerts.AlertNotificationGroups, 0, len(notificationGroups))
	for i, ng := range notificationGroups {
		notificationGroup, err := expandNotificationGroup(ng, webhooksNamesToIds)
		if err != nil {
			return nil, fmt.Errorf("error on notificationGroups[%d] - %s", i, err.Error())
		}
		result = append(result, notificationGroup)
	}
	return result, nil
}

func getWebhooksNamesToIds(ctx context.Context, log logr.Logger) (map[string]uint32, error) {
	webhooksNamesToIds := make(map[string]uint32)
	log.V(1).Info("Listing all outgoing webhooks")
	listWebhooksResp, err := WebhooksClient.List(ctx, &cxsdk.ListAllOutgoingWebhooksRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list all outgoing webhooks %w", err)
	}
	for _, webhook := range listWebhooksResp.GetDeployed() {
		webhooksNamesToIds[webhook.GetName().GetValue()] = webhook.GetExternalId().GetValue()
	}
	return webhooksNamesToIds, nil
}

func expandNotificationGroup(notificationGroup NotificationGroup, webhooksNameToIds map[string]uint32) (*alerts.AlertNotificationGroups, error) {
	groupFields := utils.StringSliceToWrappedStringSlice(notificationGroup.GroupByFields)
	notifications, err := expandNotifications(notificationGroup.Notifications, webhooksNameToIds)
	if err != nil {
		return nil, err
	}

	return &alerts.AlertNotificationGroups{
		GroupByFields: groupFields,
		Notifications: notifications,
	}, nil
}

func expandNotifications(notifications []Notification, webhooksNameToIds map[string]uint32) ([]*alerts.AlertNotification, error) {
	result := make([]*alerts.AlertNotification, 0, len(notifications))
	for i, notification := range notifications {
		expandedNotification, err := expandNotification(notification, webhooksNameToIds)
		if err != nil {
			return nil, fmt.Errorf("error on notifications[%d] - %s", i, err.Error())
		}
		result = append(result, expandedNotification)
	}
	return result, nil
}

func expandNotification(notification Notification, webhooksNameToIds map[string]uint32) (*alerts.AlertNotification, error) {
	retriggeringPeriodSeconds := wrapperspb.UInt32(uint32(60 * notification.RetriggeringPeriodMinutes))
	notifyOn := AlertSchemaNotifyOnToProtoNotifyOn[notification.NotifyOn]

	result := &alerts.AlertNotification{
		RetriggeringPeriodSeconds: retriggeringPeriodSeconds,
		NotifyOn:                  &notifyOn,
	}

	if integrationName := notification.IntegrationName; integrationName != nil {
		integrationID, _ := webhooksNameToIds[*integrationName]
		result.IntegrationType = &alerts.AlertNotification_IntegrationId{
			IntegrationId: wrapperspb.UInt32(integrationID),
		}
	}

	emails := notification.EmailRecipients
	{
		if result.IntegrationType != nil && len(emails) != 0 {
			return nil, fmt.Errorf("required exactly on of 'integrationName' or 'emailRecipients'")
		}

		if result.IntegrationType == nil {
			result.IntegrationType = &alerts.AlertNotification_Recipients{
				Recipients: &alerts.Recipients{
					Emails: utils.StringSliceToWrappedStringSlice(emails),
				},
			}
		}
	}

	return result, nil
}

func (in *AlertSpec) ExtractUpdateAlertRequest(ctx context.Context, log logr.Logger, id string) (*alerts.UpdateAlertByUniqueIdRequest, error) {
	uniqueIdentifier := wrapperspb.String(id)
	enabled := wrapperspb.Bool(in.Active)
	name := wrapperspb.String(in.Name)
	description := wrapperspb.String(in.Description)
	severity := AlertSchemaSeverityToProtoSeverity[in.Severity]
	metaLabels := expandMetaLabels(in.Labels)
	expirationDate := expandExpirationDate(in.ExpirationDate)
	showInInsight := expandShowInInsight(in.ShowInInsight)
	notificationGroups, err := expandNotificationGroups(ctx, log, in.NotificationGroups)
	if err != nil {
		return nil, err
	}
	payloadFilters := utils.StringSliceToWrappedStringSlice(in.PayloadFilters)
	activeWhen := expandActiveWhen(in.Scheduling)
	alertTypeParams := expandAlertType(in.AlertType)

	return &alerts.UpdateAlertByUniqueIdRequest{
		Alert: &alerts.Alert{
			UniqueIdentifier:           uniqueIdentifier,
			Name:                       name,
			Description:                description,
			IsActive:                   enabled,
			Severity:                   severity,
			MetaLabels:                 metaLabels,
			Expiration:                 expirationDate,
			ShowInInsight:              showInInsight,
			NotificationGroups:         notificationGroups,
			NotificationPayloadFilters: payloadFilters,
			ActiveWhen:                 activeWhen,
			Filters:                    alertTypeParams.filters,
			Condition:                  alertTypeParams.condition,
			TracingAlert:               alertTypeParams.tracingAlert,
		},
	}, nil
}

// +kubebuilder:validation:Enum=Info;Warning;Critical;Error;Low
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "Info"
	AlertSeverityWarning  AlertSeverity = "Warning"
	AlertSeverityCritical AlertSeverity = "Critical"
	AlertSeverityError    AlertSeverity = "Error"
	AlertSeverityLow      AlertSeverity = "Low"
)

type ExpirationDate struct {
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=31
	Day int32 `json:"day,omitempty"`

	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=12
	Month int32 `json:"month,omitempty"`

	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=9999
	Year int32 `json:"year,omitempty"`
}

func (in *ExpirationDate) DeepEqual(date *alerts.Date) bool {
	return in.Year != date.Year || in.Month != date.Month || in.Day != date.Day
}

type NotificationGroup struct {
	// +optional
	GroupByFields []string `json:"groupByFields,omitempty"`

	Notifications []Notification `json:"notifications,omitempty"`
}

type Notification struct {
	RetriggeringPeriodMinutes int32 `json:"retriggeringPeriodMinutes,omitempty"`

	NotifyOn NotifyOn `json:"notifyOn,omitempty"`

	// +optional
	IntegrationName *string `json:"integrationName,omitempty"`

	// +optional
	EmailRecipients []string `json:"emailRecipients,omitempty"`
}

type ShowInInsight struct {
	// +kubebuilder:validation:Minimum:=1
	RetriggeringPeriodMinutes int32 `json:"retriggeringPeriodMinutes,omitempty"`

	//+kubebuilder:default=TriggeredOnly
	NotifyOn NotifyOn `json:"notifyOn,omitempty"`
}

// +kubebuilder:validation:Enum=TriggeredOnly;TriggeredAndResolved;
type NotifyOn string

const (
	NotifyOnTriggeredOnly        = "TriggeredOnly"
	NotifyOnTriggeredAndResolved = "TriggeredAndResolved"
)

type Recipients struct {
	// +optional
	Emails []string `json:"emails,omitempty"`

	// +optional
	Webhooks []string `json:"webhooks,omitempty"`
}

type Scheduling struct {
	//+kubebuilder:default=UTC+00
	TimeZone TimeZone `json:"timeZone,omitempty"`

	DaysEnabled []Day `json:"daysEnabled,omitempty"`

	StartTime *Time `json:"startTime,omitempty"`

	EndTime *Time `json:"endTime,omitempty"`
}

// +kubebuilder:validation:Pattern=`^UTC[+-]\d{2}$`
// +kubebuilder:default=UTC+00
type TimeZone string

// +kubebuilder:validation:Enum=Sunday;Monday;Tuesday;Wednesday;Thursday;Friday;Saturday;
type Day string

const (
	Sunday    Day = "Sunday"
	Monday    Day = "Monday"
	Tuesday   Day = "Tuesday"
	Wednesday Day = "Wednesday"
	Thursday  Day = "Thursday"
	Friday    Day = "Friday"
	Saturday  Day = "Saturday"
)

// +kubebuilder:validation:Pattern=`^(0\d|1\d|2[0-3]):[0-5]\d$`
type Time string

type AlertType struct {
	// +optional
	Standard *Standard `json:"standard,omitempty"`

	// +optional
	Ratio *Ratio `json:"ratio,omitempty"`

	// +optional
	NewValue *NewValue `json:"newValue,omitempty"`

	// +optional
	UniqueCount *UniqueCount `json:"uniqueCount,omitempty"`

	// +optional
	TimeRelative *TimeRelative `json:"timeRelative,omitempty"`

	// +optional
	Metric *Metric `json:"metric,omitempty"`

	// +optional
	Tracing *Tracing `json:"tracing,omitempty"`

	// +optional
	Flow *Flow `json:"flow,omitempty"`
}

type Standard struct {
	// +optional
	Filters *Filters `json:"filters,omitempty"`

	Conditions StandardConditions `json:"conditions"`
}

type Ratio struct {
	Query1Filters Filters `json:"q1Filters,omitempty"`

	Query2Filters RatioQ2Filters `json:"q2Filters,omitempty"`

	Conditions RatioConditions `json:"conditions"`
}

type RatioQ2Filters struct {
	// +optional
	Alias *string `json:"alias,omitempty"`

	// +optional
	SearchQuery *string `json:"searchQuery,omitempty"`

	// +optional
	Severities []FiltersLogSeverity `json:"severities,omitempty"`

	// +optional
	Applications []string `json:"applications,omitempty"`

	// +optional
	Subsystems []string `json:"subsystems,omitempty"`
}

type NewValue struct {
	// +optional
	Filters *Filters `json:"filters,omitempty"`

	Conditions NewValueConditions `json:"conditions"`
}

type UniqueCount struct {
	// +optional
	Filters *Filters `json:"filters,omitempty"`

	Conditions UniqueCountConditions `json:"conditions"`
}

type TimeRelative struct {
	// +optional
	Filters *Filters `json:"filters,omitempty"`

	Conditions TimeRelativeConditions `json:"conditions"`
}

type Metric struct {
	// +optional
	Lucene *Lucene `json:"lucene,omitempty"`

	// +optional
	Promql *Promql `json:"promql,omitempty"`
}

type Lucene struct {
	// +optional
	SearchQuery *string `json:"searchQuery,omitempty"`

	Conditions LuceneConditions `json:"conditions"`
}

type Promql struct {
	SearchQuery string `json:"searchQuery,omitempty"`

	Conditions PromqlConditions `json:"conditions"`
}

type Tracing struct {
	Filters TracingFilters `json:"filters,omitempty"`

	Conditions TracingCondition `json:"conditions"`
}

type Flow struct {
	Stages []FlowStage `json:"stages"`
}

type StandardConditions struct {
	AlertWhen StandardAlertWhen `json:"alertWhen"`

	// +optional
	Threshold *int `json:"threshold,omitempty"`

	// +optional
	TimeWindow *TimeWindow `json:"timeWindow,omitempty"`

	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	// +optional
	ManageUndetectedValues *ManageUndetectedValues `json:"manageUndetectedValues,omitempty"`
}

type RatioConditions struct {
	AlertWhen AlertWhen `json:"alertWhen"`

	Ratio resource.Quantity `json:"ratio"`

	//+kubebuilder:default=false
	IgnoreInfinity bool `json:"ignoreInfinity,omitempty"`

	TimeWindow TimeWindow `json:"timeWindow"`

	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	// +optional
	GroupByFor *GroupByFor `json:"groupByFor,omitempty"`

	// +optional
	ManageUndetectedValues *ManageUndetectedValues `json:"manageUndetectedValues,omitempty"`
}

type NewValueConditions struct {
	Key string `json:"key"`

	TimeWindow NewValueTimeWindow `json:"timeWindow"`
}

type UniqueCountConditions struct {
	Key string `json:"key"`

	// +kubebuilder:validation:Minimum:=1
	MaxUniqueValues int `json:"maxUniqueValues"`

	TimeWindow UniqueValueTimeWindow `json:"timeWindow"`

	GroupBy *string `json:"groupBy,omitempty"`

	// +kubebuilder:validation:Minimum:=1
	MaxUniqueValuesForGroupBy *int `json:"maxUniqueValuesForGroupBy,omitempty"`
}

type TimeRelativeConditions struct {
	AlertWhen AlertWhen `json:"alertWhen"`

	Threshold resource.Quantity `json:"threshold"`

	//+kubebuilder:default=false
	IgnoreInfinity bool `json:"ignoreInfinity,omitempty"`

	TimeWindow RelativeTimeWindow `json:"timeWindow"`

	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	// +optional
	ManageUndetectedValues *ManageUndetectedValues `json:"manageUndetectedValues,omitempty"`
}

// +kubebuilder:validation:Enum=Avg;Min;Max;Sum;Count;Percentile;
type ArithmeticOperator string

const (
	ArithmeticOperatorAvg        ArithmeticOperator = "Avg"
	ArithmeticOperatorMin        ArithmeticOperator = "Min"
	ArithmeticOperatorMax        ArithmeticOperator = "Max"
	ArithmeticOperatorSum        ArithmeticOperator = "Sum"
	ArithmeticOperatorCount      ArithmeticOperator = "Count"
	ArithmeticOperatorPercentile ArithmeticOperator = "Percentile"
)

type LuceneConditions struct {
	MetricField string `json:"metricField"`

	ArithmeticOperator ArithmeticOperator `json:"arithmeticOperator"`

	// +optional
	ArithmeticOperatorModifier *int `json:"arithmeticOperatorModifier,omitempty"`

	AlertWhen AlertWhen `json:"alertWhen"`

	Threshold resource.Quantity `json:"threshold"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:MultipleOf:=10
	SampleThresholdPercentage int `json:"sampleThresholdPercentage,omitempty"`

	TimeWindow MetricTimeWindow `json:"timeWindow"`

	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	//+kubebuilder:default=false
	ReplaceMissingValueWithZero bool `json:"replaceMissingValueWithZero,omitempty"`

	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:MultipleOf:=10
	MinNonNullValuesPercentage int `json:"minNonNullValuesPercentage,omitempty"`

	// +optional
	ManageUndetectedValues *ManageUndetectedValues `json:"manageUndetectedValues,omitempty"`
}

type PromqlConditions struct {
	AlertWhen PromqlAlertWhen `json:"alertWhen"`

	Threshold resource.Quantity `json:"threshold"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:MultipleOf:=10
	SampleThresholdPercentage int `json:"sampleThresholdPercentage,omitempty"`

	TimeWindow MetricTimeWindow `json:"timeWindow"`

	// +optional
	ReplaceMissingValueWithZero bool `json:"replaceMissingValueWithZero,omitempty"`

	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:MultipleOf:=10
	MinNonNullValuesPercentage *int `json:"minNonNullValuesPercentage,omitempty"`

	// +optional
	ManageUndetectedValues *ManageUndetectedValues `json:"manageUndetectedValues,omitempty"`
}

type TracingCondition struct {
	AlertWhen TracingAlertWhen `json:"alertWhen"`

	// +optional
	Threshold *int `json:"threshold,omitempty"`

	// +optional
	TimeWindow *TimeWindow `json:"timeWindow,omitempty"`

	// +optional
	GroupBy []string `json:"groupBy,omitempty"`
}

// +kubebuilder:validation:Enum=Never;FiveMinutes;TenMinutes;Hour;TwoHours;SixHours;TwelveHours;TwentyFourHours
type AutoRetireRatio string

const (
	AutoRetireRatioNever           AutoRetireRatio = "Never"
	AutoRetireRatioFiveMinutes     AutoRetireRatio = "FiveMinutes"
	AutoRetireRatioTenMinutes      AutoRetireRatio = "TenMinutes"
	AutoRetireRatioHour            AutoRetireRatio = "Hour"
	AutoRetireRatioTwoHours        AutoRetireRatio = "TwoHours"
	AutoRetireRatioSixHours        AutoRetireRatio = "SixHours"
	AutoRetireRatioTwelveHours     AutoRetireRatio = "TwelveHours"
	AutoRetireRatioTwentyFourHours AutoRetireRatio = "TwentyFourHours"
)

// +kubebuilder:validation:Enum=More;Less
type AlertWhen string

const (
	AlertWhenLessThan AlertWhen = "Less"
	AlertWhenMoreThan AlertWhen = "More"
)

// +kubebuilder:validation:Enum=More;Less;MoreThanUsual
type PromqlAlertWhen string

const (
	PromqlAlertWhenLessThan      PromqlAlertWhen = "Less"
	PromqlAlertWhenMoreThan      PromqlAlertWhen = "More"
	PromqlAlertWhenMoreOrEqual   PromqlAlertWhen = "MoreOrEqual"
	PromqlAlertWhenLessOrEqual   PromqlAlertWhen = "LessOrEqual"
	PromqlAlertWhenMoreThanUsual PromqlAlertWhen = "MoreThanUsual"
)

// +kubebuilder:validation:Enum=More;Less;Immediately;MoreThanUsual
type StandardAlertWhen string

const (
	StandardAlertWhenLessThan      StandardAlertWhen = "Less"
	StandardAlertWhenMoreThan      StandardAlertWhen = "More"
	StandardAlertWhenMoreThanUsual StandardAlertWhen = "MoreThanUsual"
	StandardAlertWhenImmediately   StandardAlertWhen = "Immediately"
)

// +kubebuilder:validation:Enum=More;Immediately
type TracingAlertWhen string

const (
	TracingAlertWhenMore        TracingAlertWhen = "More"
	TracingAlertWhenImmediately TracingAlertWhen = "Immediately"
)

// +kubebuilder:validation:Enum=Q1;Q2;Both
type GroupByFor string

const (
	GroupByForQ1   GroupByFor = "Q1"
	GroupByForQ2   GroupByFor = "Q2"
	GroupByForBoth GroupByFor = "Both"
)

// +kubebuilder:validation:Enum=FiveMinutes;TenMinutes;FifteenMinutes;TwentyMinutes;ThirtyMinutes;Hour;TwoHours;FourHours;SixHours;TwelveHours;TwentyFourHours;ThirtySixHours
type TimeWindow string

const (
	TimeWindowMinute          TimeWindow = "Minute"
	TimeWindowFiveMinutes     TimeWindow = "FiveMinutes"
	TimeWindowTenMinutes      TimeWindow = "TenMinutes"
	TimeWindowFifteenMinutes  TimeWindow = "FifteenMinutes"
	TimeWindowTwentyMinutes   TimeWindow = "TwentyMinutes"
	TimeWindowThirtyMinutes   TimeWindow = "ThirtyMinutes"
	TimeWindowHour            TimeWindow = "Hour"
	TimeWindowTwoHours        TimeWindow = "TwoHours"
	TimeWindowFourHours       TimeWindow = "FourHours"
	TimeWindowSixHours        TimeWindow = "SixHours"
	TimeWindowTwelveHours     TimeWindow = "TwelveHours"
	TimeWindowTwentyFourHours TimeWindow = "TwentyFourHours"
	TimeWindowThirtySixHours  TimeWindow = "ThirtySixHours"
)

// +kubebuilder:validation:Enum=TwelveHours;TwentyFourHours;FortyEightHours;SeventyTwoHours;Week;Month;TwoMonths;ThreeMonths;
type NewValueTimeWindow string

const (
	NewValueTimeWindowTwelveHours     NewValueTimeWindow = "TwelveHours"
	NewValueTimeWindowTwentyFourHours NewValueTimeWindow = "TwentyFourHours"
	NewValueTimeWindowFortyEightHours NewValueTimeWindow = "FortyEightHours"
	NewValueTimeWindowSeventyTwoHours NewValueTimeWindow = "SeventyTwoHours"
	NewValueTimeWindowWeek            NewValueTimeWindow = "Week"
	NewValueTimeWindowMonth           NewValueTimeWindow = "Month"
	NewValueTimeWindowTwoMonths       NewValueTimeWindow = "TwoMonths"
	NewValueTimeWindowThreeMonths     NewValueTimeWindow = "ThreeMonths"
)

// +kubebuilder:validation:Enum=Minute;FiveMinutes;TenMinutes;FifteenMinutes;TwentyMinutes;ThirtyMinutes;Hour;TwoHours;FourHours;SixHours;TwelveHours;TwentyFourHours;ThirtySixHours
type UniqueValueTimeWindow string

const (
	UniqueValueTimeWindowMinute          UniqueValueTimeWindow = "Minute"
	UniqueValueTimeWindowFiveMinutes     UniqueValueTimeWindow = "FiveMinutes"
	UniqueValueTimeWindowTenMinutes      UniqueValueTimeWindow = "TenMinutes"
	UniqueValueTimeWindowFifteenMinutes  UniqueValueTimeWindow = "FifteenMinutes"
	UniqueValueTimeWindowTwentyMinutes   UniqueValueTimeWindow = "TwentyMinutes"
	UniqueValueTimeWindowThirtyMinutes   UniqueValueTimeWindow = "ThirtyMinutes"
	UniqueValueTimeWindowHour            UniqueValueTimeWindow = "Hour"
	UniqueValueTimeWindowTwoHours        UniqueValueTimeWindow = "TwoHours"
	UniqueValueTimeWindowFourHours       UniqueValueTimeWindow = "FourHours"
	UniqueValueTimeWindowSixHours        UniqueValueTimeWindow = "SixHours"
	UniqueValueTimeWindowTwelveHours     UniqueValueTimeWindow = "TwelveHours"
	UniqueValueTimeWindowTwentyFourHours UniqueValueTimeWindow = "TwentyFourHours"
	UniqueValueTimeWindowThirtySixHours  UniqueValueTimeWindow = "ThirtySixHours"
)

// +kubebuilder:validation:Enum=Minute;FiveMinutes;TenMinutes;FifteenMinutes;TwentyMinutes;ThirtyMinutes;Hour;TwoHours;FourHours;SixHours;TwelveHours;TwentyFourHours;ThirtySixHours
type MetricTimeWindow string

const (
	MetricTimeWindowMinute          MetricTimeWindow = "Minute"
	MetricTimeWindowFiveMinutes     MetricTimeWindow = "FiveMinutes"
	MetricTimeWindowTenMinutes      MetricTimeWindow = "TenMinutes"
	MetricTimeWindowFifteenMinutes  MetricTimeWindow = "FifteenMinutes"
	MetricTimeWindowTwentyMinutes   MetricTimeWindow = "TwentyMinutes"
	MetricTimeWindowThirtyMinutes   MetricTimeWindow = "ThirtyMinutes"
	MetricTimeWindowHour            MetricTimeWindow = "Hour"
	MetricTimeWindowTwoHours        MetricTimeWindow = "TwoHours"
	MetricTimeWindowFourHours       MetricTimeWindow = "FourHours"
	MetricTimeWindowSixHours        MetricTimeWindow = "SixHours"
	MetricTimeWindowTwelveHours     MetricTimeWindow = "TwelveHours"
	MetricTimeWindowTwentyFourHours MetricTimeWindow = "TwentyFourHours"
	MetricTimeWindowThirtySixHours  MetricTimeWindow = "ThirtySixHours"
)

// +kubebuilder:validation:Enum=PreviousHour;SameHourYesterday;SameHourLastWeek;Yesterday;SameDayLastWeek;SameDayLastMonth;
type RelativeTimeWindow string

const (
	RelativeTimeWindowPreviousHour      RelativeTimeWindow = "PreviousHour"
	RelativeTimeWindowSameHourYesterday RelativeTimeWindow = "SameHourYesterday"
	RelativeTimeWindowSameHourLastWeek  RelativeTimeWindow = "SameHourLastWeek"
	RelativeTimeWindowYesterday         RelativeTimeWindow = "Yesterday"
	RelativeTimeWindowSameDayLastWeek   RelativeTimeWindow = "SameDayLastWeek"
	RelativeTimeWindowSameDayLastMonth  RelativeTimeWindow = "SameDayLastMonth"
)

type Filters struct {
	// +optional
	SearchQuery *string `json:"searchQuery,omitempty"`

	// +optional
	Severities []FiltersLogSeverity `json:"severities,omitempty"`

	// +optional
	Applications []string `json:"applications,omitempty"`

	// +optional
	Subsystems []string `json:"subsystems,omitempty"`

	// +optional
	Categories []string `json:"categories,omitempty"`

	// +optional
	Computers []string `json:"computers,omitempty"`

	// +optional
	Classes []string `json:"classes,omitempty"`

	// +optional
	Methods []string `json:"methods,omitempty"`

	// +optional
	IPs []string `json:"ips,omitempty"`

	// +optional
	Alias *string `json:"alias,omitempty"`
}

// +kubebuilder:validation:Enum=Debug;Verbose;Info;Warning;Critical;Error;
type FiltersLogSeverity string

const (
	FiltersLogSeverityDebug    FiltersLogSeverity = "Debug"
	FiltersLogSeverityVerbose  FiltersLogSeverity = "Verbose"
	FiltersLogSeverityInfo     FiltersLogSeverity = "Info"
	FiltersLogSeverityWarning  FiltersLogSeverity = "Warning"
	FiltersLogSeverityCritical FiltersLogSeverity = "Critical"
	FiltersLogSeverityError    FiltersLogSeverity = "Error"
)

type TracingFilters struct {
	LatencyThresholdMilliseconds resource.Quantity `json:"latencyThresholdMilliseconds,omitempty"`

	// +optional
	TagFilters []TagFilter `json:"tagFilters,omitempty"`

	// +optional
	Applications []string `json:"applications,omitempty"`

	// +optional
	Subsystems []string `json:"subsystems,omitempty"`

	// +optional
	Services []string `json:"services,omitempty"`
}

type TagFilter struct {
	Field  string   `json:"field,omitempty"`
	Values []string `json:"values,omitempty"`
}

// +kubebuilder:validation:Enum=Equals;Contains;StartWith;EndWith;
type FilterOperator string

const (
	FilterOperatorEquals    = "Equals"
	FilterOperatorContains  = "Contains"
	FilterOperatorStartWith = "StartWith"
	FilterOperatorEndWith   = "EndWith"
)

// +kubebuilder:validation:Enum=Application;Subsystem;Service;
type FieldFilterType string

const (
	FieldFilterTypeApplication = "Application"
	FieldFilterTypeSubsystem   = "Subsystem"
	FieldFilterTypeService     = "Service"
)

type ManageUndetectedValues struct {
	//+kubebuilder:default=true
	EnableTriggeringOnUndetectedValues bool `json:"enableTriggeringOnUndetectedValues,omitempty"`

	//+kubebuilder:default=Never
	AutoRetireRatio *AutoRetireRatio `json:"autoRetireRatio,omitempty"`
}

type FlowStage struct {
	// +optional
	TimeWindow *FlowStageTimeFrame `json:"timeWindow,omitempty"`

	Groups []FlowStageGroup `json:"groups"`
}

type FlowStageTimeFrame struct {
	// +optional
	Hours int `json:"hours,omitempty"`

	// +optional
	Minutes int `json:"minutes,omitempty"`

	// +optional
	Seconds int `json:"seconds,omitempty"`
}

type FlowStageGroup struct {
	InnerFlowAlerts InnerFlowAlerts `json:"innerFlowAlerts"`

	NextOperator FlowOperator `json:"nextOperator"`
}

type InnerFlowAlerts struct {
	Operator FlowOperator `json:"operator"`

	Alerts []InnerFlowAlert `json:"alerts"`
}

type InnerFlowAlert struct {
	// +kubebuilder:default=false
	Not bool `json:"not,omitempty"`

	// +optional
	UserAlertId string `json:"userAlertId,omitempty"`
}

// +kubebuilder:validation:Enum=And;Or
type FlowOperator string

// AlertStatus defines the observed state of Alert
type AlertStatus struct {
	ID *string `json:"id"`
}

func NewDefaultAlertStatus() *AlertStatus {
	return &AlertStatus{
		ID: ptr.To(""),
	}
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Alert is the Schema for the alerts API
type Alert struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertSpec   `json:"spec,omitempty"`
	Status AlertStatus `json:"status,omitempty"`
}

func NewAlert() *Alert {
	return &Alert{
		Spec: AlertSpec{
			Labels: make(map[string]string),
		},
	}
}

//+kubebuilder:object:root=true

// AlertList contains a list of Alert
type AlertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Alert `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Alert{}, &AlertList{})
}
