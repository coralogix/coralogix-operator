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
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/coralogix/coralogix-operator/api/coralogix"
	"github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
)

var (
	SeveritiesV1alpha1ToV1beta1 = map[AlertSeverity]v1beta1.AlertPriority{
		AlertSeverityCritical: v1beta1.AlertPriorityP1,
		AlertSeverityError:    v1beta1.AlertPriorityP2,
		AlertSeverityWarning:  v1beta1.AlertPriorityP3,
		AlertSeverityInfo:     v1beta1.AlertPriorityP4,
		AlertSeverityLow:      v1beta1.AlertPriorityP5,
	}
	severitiesV1beta1ToV1alpha1 = coralogix.ReverseMap(SeveritiesV1alpha1ToV1beta1)
	notifyOnV1alpha1ToV1beta1   = map[NotifyOn]v1beta1.NotifyOn{
		NotifyOnTriggeredOnly:        v1beta1.NotifyOnTriggeredOnly,
		NotifyOnTriggeredAndResolved: v1beta1.NotifyOnTriggeredAndResolved,
	}
	notifyOnV1beta1ToV1alpha1  = coralogix.ReverseMap(notifyOnV1alpha1ToV1beta1)
	dayOfWeekV1alpha1ToV1beta1 = map[Day]v1beta1.DayOfWeek{
		Sunday:    v1beta1.DayOfWeekSunday,
		Monday:    v1beta1.DayOfWeekMonday,
		Tuesday:   v1beta1.DayOfWeekTuesday,
		Wednesday: v1beta1.DayOfWeekWednesday,
		Thursday:  v1beta1.DayOfWeekThursday,
		Friday:    v1beta1.DayOfWeekFriday,
		Saturday:  v1beta1.DayOfWeekSaturday,
	}
	dayOfWeekV1beta1ToV1alpha1        = coralogix.ReverseMap(dayOfWeekV1alpha1ToV1beta1)
	severitiesFilterV1alpha1ToV1beta1 = map[FiltersLogSeverity]v1beta1.LogSeverity{
		FiltersLogSeverityCritical: v1beta1.LogSeverityCritical,
		FiltersLogSeverityError:    v1beta1.LogSeverityError,
		FiltersLogSeverityWarning:  v1beta1.LogSeverityWarning,
		FiltersLogSeverityInfo:     v1beta1.LogSeverityInfo,
		FiltersLogSeverityVerbose:  v1beta1.LogSeverityVerbose,
		FiltersLogSeverityDebug:    v1beta1.LogSeverityDebug,
	}
	severitiesFilterV1beta1ToV1alpha1 = coralogix.ReverseMap(severitiesFilterV1alpha1ToV1beta1)
	autoRetireRatioV1alpha1ToV1beta1  = map[AutoRetireRatio]v1beta1.AutoRetireTimeframe{
		AutoRetireRatioNever:           v1beta1.AutoRetireTimeframeNeverOrUnspecified,
		AutoRetireRatioFiveMinutes:     v1beta1.AutoRetireTimeframe5M,
		AutoRetireRatioTenMinutes:      v1beta1.AutoRetireTimeframe10M,
		AutoRetireRatioHour:            v1beta1.AutoRetireTimeframe1H,
		AutoRetireRatioTwoHours:        v1beta1.AutoRetireTimeframe2H,
		AutoRetireRatioSixHours:        v1beta1.AutoRetireTimeframe6H,
		AutoRetireRatioTwelveHours:     v1beta1.AutoRetireTimeframe12H,
		AutoRetireRatioTwentyFourHours: v1beta1.AutoRetireTimeframe24H,
	}
	autoRetireRatioV1beta1ToV1alpha1 = coralogix.ReverseMap(autoRetireRatioV1alpha1ToV1beta1)
	logsTimeWindowV1alpha1ToV1beta1  = map[TimeWindow]v1beta1.LogsTimeWindowValue{
		TimeWindowFiveMinutes:     v1beta1.LogsTimeWindow5Minutes,
		TimeWindowTenMinutes:      v1beta1.LogsTimeWindow10Minutes,
		TimeWindowFifteenMinutes:  v1beta1.LogsTimeWindow15Minutes,
		TimeWindowThirtyMinutes:   v1beta1.LogsTimeWindow30Minutes,
		TimeWindowHour:            v1beta1.LogsTimeWindowHour,
		TimeWindowTwoHours:        v1beta1.LogsTimeWindow2Hours,
		TimeWindowSixHours:        v1beta1.LogsTimeWindow6Hours,
		TimeWindowTwelveHours:     v1beta1.LogsTimeWindow12Hours,
		TimeWindowTwentyFourHours: v1beta1.LogsTimeWindow24Hours,
		TimeWindowThirtySixHours:  v1beta1.LogsTimeWindow36Hours,
	}
	logsTimeWindowV1beta1ToV1alpha     = coralogix.ReverseMap(logsTimeWindowV1alpha1ToV1beta1)
	logsConditionTypeV1alpha1ToV1beta1 = map[StandardAlertWhen]v1beta1.LogsThresholdConditionType{
		StandardAlertWhenMoreThan: v1beta1.LogsThresholdConditionTypeMoreThan,
		StandardAlertWhenLessThan: v1beta1.LogsThresholdConditionTypeLessThan,
	}
	logsConditionTypeV1beta1ToV1alpha = coralogix.ReverseMap(logsConditionTypeV1alpha1ToV1beta1)
	flowOperationV1alpha1ToV1beta1    = map[FlowOperator]v1beta1.FlowStageGroupAlertsOp{
		FlowOperatorAnd: v1beta1.FlowStageGroupAlertsOpAnd,
		FlowOperatorOr:  v1beta1.FlowStageGroupAlertsOpOr,
	}
	flowOperationV1beta1ToV1alpha     = coralogix.ReverseMap(flowOperationV1alpha1ToV1beta1)
	metricTimeWindowV1alpha1ToV1beta1 = map[MetricTimeWindow]v1beta1.MetricTimeWindowSpecificValue{
		MetricTimeWindowMinute:          v1beta1.MetricTimeWindowValue1Minute,
		MetricTimeWindowFiveMinutes:     v1beta1.MetricTimeWindowValue5Minutes,
		MetricTimeWindowTenMinutes:      v1beta1.MetricTimeWindowValue10Minutes,
		MetricTimeWindowFifteenMinutes:  v1beta1.MetricTimeWindowValue15Minutes,
		MetricTimeWindowTwentyMinutes:   v1beta1.MetricTimeWindowValue20Minutes,
		MetricTimeWindowThirtyMinutes:   v1beta1.MetricTimeWindowValue30Minutes,
		MetricTimeWindowHour:            v1beta1.MetricTimeWindowValue1Hour,
		MetricTimeWindowTwoHours:        v1beta1.MetricTimeWindowValue2Hours,
		MetricTimeWindowFourHours:       v1beta1.MetricTimeWindowValue4Hours,
		MetricTimeWindowSixHours:        v1beta1.MetricTimeWindowValue6Hours,
		MetricTimeWindowTwelveHours:     v1beta1.MetricTimeWindowValue12Hours,
		MetricTimeWindowTwentyFourHours: v1beta1.MetricTimeWindowValue24Hours,
	}
	metricTimeWindowV1beta1ToV1alpha1    = coralogix.ReverseMap(metricTimeWindowV1alpha1ToV1beta1)
	metricConditionTypeV1alpha1ToV1beta1 = map[PromqlAlertWhen]v1beta1.MetricThresholdConditionType{
		PromqlAlertWhenLessThan:    v1beta1.MetricThresholdConditionTypeLessThan,
		PromqlAlertWhenMoreThan:    v1beta1.MetricThresholdConditionTypeMoreThan,
		PromqlAlertWhenLessOrEqual: v1beta1.MetricThresholdConditionTypeLessThanOrEquals,
		PromqlAlertWhenMoreOrEqual: v1beta1.MetricThresholdConditionTypeMoreThanOrEquals,
	}
	metricConditionTypeV1beta1ToV1alpha1        = coralogix.ReverseMap(metricConditionTypeV1alpha1ToV1beta1)
	metricAnomalyConditionTypeV1alpha1ToV1beta1 = map[PromqlAlertWhen]v1beta1.MetricAnomalyConditionType{
		PromqlAlertWhenLessThanUsual: v1beta1.MetricAnomalyConditionTypeLessThanUsual,
		PromqlAlertWhenMoreThanUsual: v1beta1.MetricAnomalyConditionTypeMoreThanUsual,
	}
	metricAnomalyConditionTypeV1beta1ToV1alpha1 = coralogix.ReverseMap(metricAnomalyConditionTypeV1alpha1ToV1beta1)
	newValueTimeWindowV1alpha1ToV1beta1         = map[NewValueTimeWindow]v1beta1.LogsNewValueTimeWindowSpecificValue{
		NewValueTimeWindowTwelveHours:     v1beta1.LogsNewValueTimeWindowValue12Hours,
		NewValueTimeWindowTwentyFourHours: v1beta1.LogsNewValueTimeWindowValue24Hours,
		NewValueTimeWindowFortyEightHours: v1beta1.LogsNewValueTimeWindowValue48Hours,
		NewValueTimeWindowSeventyTwoHours: v1beta1.LogsNewValueTimeWindowValue72Hours,
		NewValueTimeWindowWeek:            v1beta1.LogsNewValueTimeWindowValue1Week,
		NewValueTimeWindowMonth:           v1beta1.LogsNewValueTimeWindowValue1Month,
		NewValueTimeWindowTwoMonths:       v1beta1.LogsNewValueTimeWindowValue2Months,
		NewValueTimeWindowThreeMonths:     v1beta1.LogsNewValueTimeWindowValue3Months,
	}
	newValueTimeWindowV1beta1ToV1alpha = coralogix.ReverseMap(newValueTimeWindowV1alpha1ToV1beta1)
	tracingTimeWindowV1alpha1ToV1beta1 = map[TimeWindow]v1beta1.TracingTimeWindowSpecificValue{
		TimeWindowFiveMinutes:     v1beta1.TracingTimeWindowValue5Minutes,
		TimeWindowTenMinutes:      v1beta1.TracingTimeWindowValue10Minutes,
		TimeWindowFifteenMinutes:  v1beta1.TracingTimeWindowValue15Minutes,
		TimeWindowTwentyMinutes:   v1beta1.TracingTimeWindowValue20Minutes,
		TimeWindowThirtyMinutes:   v1beta1.TracingTimeWindowValue30Minutes,
		TimeWindowHour:            v1beta1.TracingTimeWindowValue1Hour,
		TimeWindowTwoHours:        v1beta1.TracingTimeWindowValue2Hours,
		TimeWindowFourHours:       v1beta1.TracingTimeWindowValue4Hours,
		TimeWindowSixHours:        v1beta1.TracingTimeWindowValue6Hours,
		TimeWindowTwelveHours:     v1beta1.TracingTimeWindowValue12Hours,
		TimeWindowTwentyFourHours: v1beta1.TracingTimeWindowValue24Hours,
		TimeWindowThirtySixHours:  v1beta1.TracingTimeWindowValue36Hours,
	}
	tracingTimeWindowV1beta1ToV1alpha1   = coralogix.ReverseMap(tracingTimeWindowV1alpha1ToV1beta1)
	logsRatioTimeWindowV1alpha1ToV1beta1 = map[TimeWindow]v1beta1.LogsRatioTimeWindowValue{
		TimeWindowFiveMinutes:     v1beta1.LogsRatioTimeWindowMinutes5,
		TimeWindowTenMinutes:      v1beta1.LogsRatioTimeWindowMinutes10,
		TimeWindowFifteenMinutes:  v1beta1.LogsRatioTimeWindowMinutes15,
		TimeWindowThirtyMinutes:   v1beta1.LogsRatioTimeWindowMinutes30,
		TimeWindowHour:            v1beta1.LogsRatioTimeWindow1Hour,
		TimeWindowTwoHours:        v1beta1.LogsRatioTimeWindowHours2,
		TimeWindowSixHours:        v1beta1.LogsRatioTimeWindowHours6,
		TimeWindowTwelveHours:     v1beta1.LogsRatioTimeWindowHours12,
		TimeWindowTwentyFourHours: v1beta1.LogsRatioTimeWindowHours24,
	}
	logsRatioTimeWindowV1beta1ToV1alpha1 = coralogix.ReverseMap(logsRatioTimeWindowV1alpha1ToV1beta1)
	ratioConditionTypeV1alpha1ToV1beta1  = map[AlertWhen]v1beta1.LogsRatioConditionType{
		AlertWhenMoreThan: v1beta1.LogsRatioConditionTypeMoreThan,
		AlertWhenLessThan: v1beta1.LogsRatioConditionTypeLessThan,
	}
	ratioConditionTypeV1beta1ToV1alpha1     = coralogix.ReverseMap(ratioConditionTypeV1alpha1ToV1beta1)
	timeRelativeTimeWindowV1alpha1ToV1beta1 = map[RelativeTimeWindow]v1beta1.LogsTimeRelativeComparedTo{
		RelativeTimeWindowPreviousHour:      v1beta1.LogsTimeRelativeComparedToPreviousHour,
		RelativeTimeWindowSameHourYesterday: v1beta1.LogsTimeRelativeComparedToSameHourYesterday,
		RelativeTimeWindowSameHourLastWeek:  v1beta1.LogsTimeRelativeComparedToSameHourLastWeek,
		RelativeTimeWindowYesterday:         v1beta1.LogsTimeRelativeComparedToYesterday,
		RelativeTimeWindowSameDayLastWeek:   v1beta1.LogsTimeRelativeComparedToSameDayLastWeek,
		RelativeTimeWindowSameDayLastMonth:  v1beta1.LogsTimeRelativeComparedToSameDayLastMonth,
	}
	timeRelativeTimeWindowV1beta1ToV1alpha1    = coralogix.ReverseMap(timeRelativeTimeWindowV1alpha1ToV1beta1)
	timeRelativeConditionTypeV1alpha1ToV1beta1 = map[AlertWhen]v1beta1.LogsTimeRelativeConditionType{
		AlertWhenMoreThan: v1beta1.LogsTimeRelativeConditionTypeMoreThan,
		AlertWhenLessThan: v1beta1.LogsTimeRelativeConditionTypeLessThan,
	}
	timeRelativeConditionTypeV1beta1ToV1alpha1 = coralogix.ReverseMap(timeRelativeConditionTypeV1alpha1ToV1beta1)
	logsUniqueCountTimeWindowV1alpha1ToV1beta1 = map[UniqueValueTimeWindow]v1beta1.LogsUniqueCountTimeWindowSpecificValue{
		UniqueValueTimeWindowMinute:          v1beta1.LogsUniqueCountTimeWindowValue1Minute,
		UniqueValueTimeWindowFiveMinutes:     v1beta1.LogsUniqueCountTimeWindowValue5Minutes,
		UniqueValueTimeWindowTenMinutes:      v1beta1.LogsUniqueCountTimeWindowValue10Minutes,
		UniqueValueTimeWindowFifteenMinutes:  v1beta1.LogsUniqueCountTimeWindowValue15Minutes,
		UniqueValueTimeWindowTwentyMinutes:   v1beta1.LogsUniqueCountTimeWindowValue20Minutes,
		UniqueValueTimeWindowThirtyMinutes:   v1beta1.LogsUniqueCountTimeWindowValue30Minutes,
		UniqueValueTimeWindowHour:            v1beta1.LogsUniqueCountTimeWindowValue1Hour,
		UniqueValueTimeWindowTwoHours:        v1beta1.LogsUniqueCountTimeWindowValue2Hours,
		UniqueValueTimeWindowFourHours:       v1beta1.LogsUniqueCountTimeWindowValue4Hours,
		UniqueValueTimeWindowSixHours:        v1beta1.LogsUniqueCountTimeWindowValue6Hours,
		UniqueValueTimeWindowTwelveHours:     v1beta1.LogsUniqueCountTimeWindowValue12Hours,
		UniqueValueTimeWindowTwentyFourHours: v1beta1.LogsUniqueCountTimeWindowValue24Hours,
		UniqueValueTimeWindowThirtySixHours:  v1beta1.LogsUniqueCountTimeWindowValue36Hours,
	}
	logsUniqueCountTimeWindowV1beta1ToV1alpha1 = coralogix.ReverseMap(logsUniqueCountTimeWindowV1alpha1ToV1beta1)
	msInHour                                   = int(time.Hour.Milliseconds())
	msInMinute                                 = int(time.Minute.Milliseconds())
	msInSecond                                 = int(time.Second.Milliseconds())
)

// ConvertFrom converts from the Hub version (v1beta1) to this version (v1alpha1)
func (dst *Alert) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.Alert)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec = AlertSpec{
		Name:        src.Spec.Name,
		Description: src.Spec.Description,
		Severity:    severitiesV1beta1ToV1alpha1[src.Spec.Priority],
		Active:      src.Spec.Enabled,
		Labels:      src.Spec.EntityLabels,
		Scheduling:  convertSchedulingV1beta1ToV1alpha1(src.Spec.Schedule),
	}

	if alertType, payloadFilters := convertAlertTypeV1beta1ToV1alpha1(src.Spec.TypeDefinition, src.Spec.GroupByKeys); alertType != nil {
		dst.Spec.AlertType = *alertType
		dst.Spec.PayloadFilters = payloadFilters
	} else {
		return fmt.Errorf("failed to convert alert type %s", src.Name)
	}

	dst.Spec.NotificationGroups = convertingNotificationGroupsV1beta1ToV1alpha1(src.Spec.NotificationGroup, src.Spec.NotificationGroupExcess)
	dst.Status.ID = src.Status.ID
	dst.Status.Conditions = src.Status.Conditions

	return nil
}

// ConvertTo converts this Alert (v1alpha1) to the Hub version (v1beta1)
func (src *Alert) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.Alert)
	dst.ObjectMeta = src.ObjectMeta
	dstSpec := v1beta1.AlertSpec{
		Name:         src.Spec.Name,
		Description:  src.Spec.Description,
		Priority:     SeveritiesV1alpha1ToV1beta1[src.Spec.Severity],
		Enabled:      src.Spec.Active,
		EntityLabels: src.Spec.Labels,
		Schedule:     convertSchedulingV1alpha1ToV1beta1(src.Spec.Scheduling),
	}
	dstSpec.TypeDefinition, dstSpec.GroupByKeys = convertAlertTypeV1alpha1ToV1beta1(src.Spec)
	if len(src.Spec.NotificationGroups) > 0 {
		dstSpec.NotificationGroup = convertNotificationGroupsV1alpha1ToV1beta1(src.Spec.NotificationGroups[0])
	}
	if len(src.Spec.NotificationGroups) > 1 {
		dstSpec.NotificationGroupExcess = convertNotificationGroupExcessV1alpha1ToV1beta1(src.Spec.NotificationGroups[1:])
	}

	dst.Spec = dstSpec
	dst.Status.ID = src.Status.ID
	dst.Status.Conditions = src.Status.Conditions

	return nil
}

func convertAlertTypeV1beta1ToV1alpha1(definition v1beta1.AlertTypeDefinition, groupBy []string) (*AlertType, []string) {
	if logsImmediate := definition.LogsImmediate; logsImmediate != nil {
		return &AlertType{
			Standard: convertLogImmediateV1beta1ToStandardV1alpha1(logsImmediate),
		}, logsImmediate.NotificationPayloadFilter
	} else if logsThreshold := definition.LogsThreshold; logsThreshold != nil {
		return &AlertType{
			Standard: convertLogsThresholdV1beta1ToStandardV1alpha1(logsThreshold, groupBy),
		}, logsThreshold.NotificationPayloadFilter
	} else if logsRatioThreshold := definition.LogsRatioThreshold; logsRatioThreshold != nil {
		return &AlertType{
			Ratio: convertRatioV1beta1ToV1alpha1(logsRatioThreshold, groupBy),
		}, nil
	} else if logsNewValue := definition.LogsNewValue; logsNewValue != nil {
		return &AlertType{
			NewValue: convertLogsNewValueV1beta1ToV1alpha1(logsNewValue),
		}, logsNewValue.NotificationPayloadFilter
	} else if metricAnomaly := definition.MetricAnomaly; metricAnomaly != nil {
		return &AlertType{
			Metric: convertMetricAnomalyV1beta1ToMetricV1alpha1(metricAnomaly),
		}, nil
	} else if metricThreshold := definition.MetricThreshold; metricThreshold != nil {
		return &AlertType{
			Metric: convertMetricThresholdV1beta1ToMetricV1alpha1(metricThreshold),
		}, nil
	} else if flow := definition.Flow; flow != nil {
		return &AlertType{
			Flow: convertFlowV1beta1ToV1alpha1(flow),
		}, nil
	} else if tracingThreshold := definition.TracingThreshold; tracingThreshold != nil {
		return &AlertType{
			Tracing: convertTracingThresholdV1beta1ToV1alpha1(tracingThreshold, groupBy),
		}, tracingThreshold.NotificationPayloadFilter
	} else if tracingImmediate := definition.TracingImmediate; tracingImmediate != nil {
		return &AlertType{
			Tracing: convertTracingImmediateV1beta1ToV1alpha1(tracingImmediate, groupBy),
		}, tracingImmediate.NotificationPayloadFilter
	} else if logsUniqueCount := definition.LogsUniqueCount; logsUniqueCount != nil {
		return &AlertType{
			UniqueCount: convertLogsUniqueCountV1beta1ToV1alpha1(logsUniqueCount, groupBy),
		}, logsUniqueCount.NotificationPayloadFilter
	} else if timeRelative := definition.LogsTimeRelativeThreshold; timeRelative != nil {
		return &AlertType{
			TimeRelative: convertTimeRelativeV1beta1ToV1alpha1(timeRelative, groupBy),
		}, timeRelative.NotificationPayloadFilter
	} else if logsAnomaly := definition.LogsAnomaly; logsAnomaly != nil {
		return &AlertType{
			Standard: convertLogsAnomalyV1beta1ToStandardV1alpha1(logsAnomaly, groupBy),
		}, logsAnomaly.NotificationPayloadFilter
	}

	return nil, nil
}

func convertLogsAnomalyV1beta1ToStandardV1alpha1(anomaly *v1beta1.LogsAnomaly, groupBy []string) *Standard {
	condition := anomaly.Rules[0].Condition
	timeWindow := logsTimeWindowV1beta1ToV1alpha[condition.TimeWindow.SpecificValue]
	return &Standard{
		Filters: convertLogsFilterV1beta1ToV1alpha1(anomaly.LogsFilter),
		Conditions: StandardConditions{
			AlertWhen:  StandardAlertWhenMoreThanUsual,
			Threshold:  pointer.Int(int(condition.MinimumThreshold.Value())),
			TimeWindow: &timeWindow,
			GroupBy:    groupBy,
		},
	}
}

func convertTimeRelativeV1beta1ToV1alpha1(timeRelative *v1beta1.LogsTimeRelativeThreshold, groupBy []string) *TimeRelative {
	return &TimeRelative{
		Filters:    convertLogsFilterV1beta1ToV1alpha1(&timeRelative.LogsFilter),
		Conditions: convertTimeRelativeConditionV1beta1ToV1alpha1(timeRelative, groupBy),
	}
}

func convertTimeRelativeConditionV1beta1ToV1alpha1(timeRelative *v1beta1.LogsTimeRelativeThreshold, groupBy []string) TimeRelativeConditions {
	condition := timeRelative.Rules[0].Condition
	return TimeRelativeConditions{
		AlertWhen:              timeRelativeConditionTypeV1beta1ToV1alpha1[condition.ConditionType],
		Threshold:              condition.Threshold.DeepCopy(),
		TimeWindow:             timeRelativeTimeWindowV1beta1ToV1alpha1[condition.ComparedTo],
		GroupBy:                groupBy,
		ManageUndetectedValues: convertUndetectedValuesManagementV1beta1ToV1alpha1(timeRelative.UndetectedValuesManagement),
	}
}

func convertLogsUniqueCountV1beta1ToV1alpha1(uniqueCount *v1beta1.LogsUniqueCount, groupBy []string) *UniqueCount {
	return &UniqueCount{
		Filters:    convertLogsFilterV1beta1ToV1alpha1(uniqueCount.LogsFilter),
		Conditions: convertLogsUniqueCountConditionV1beta1ToV1alpha1(uniqueCount, groupBy),
	}
}

func convertLogsUniqueCountConditionV1beta1ToV1alpha1(uniqueCount *v1beta1.LogsUniqueCount, groupBy []string) UniqueCountConditions {
	condition := uniqueCount.Rules[0].Condition
	parsedUniqueCount := UniqueCountConditions{
		Key:             uniqueCount.UniqueCountKeypath,
		MaxUniqueValues: int(condition.Threshold),
		TimeWindow:      logsUniqueCountTimeWindowV1beta1ToV1alpha1[condition.TimeWindow.SpecificValue],
	}
	if len(groupBy) > 0 {
		parsedUniqueCount.GroupBy = pointer.String(groupBy[0])
		parsedUniqueCount.MaxUniqueValuesForGroupBy = pointer.Int(int(*uniqueCount.MaxUniqueCountPerGroupByKey))
	}
	return parsedUniqueCount
}

func convertTracingThresholdV1beta1ToV1alpha1(tracingThreshold *v1beta1.TracingThreshold, groupBy []string) *Tracing {
	condition := tracingThreshold.Rules[0].Condition
	timeWindow := tracingTimeWindowV1beta1ToV1alpha1[condition.TimeWindow.SpecificValue]
	return &Tracing{
		Filters: convertTracingFilterV1beta1ToV1alpha1(tracingThreshold.TracingFilter),
		Conditions: TracingCondition{
			AlertWhen:  TracingAlertWhenMore,
			Threshold:  pointer.Int(int(condition.SpanAmount.Value())),
			TimeWindow: &timeWindow,
			GroupBy:    groupBy,
		},
	}
}

func convertTracingImmediateV1beta1ToV1alpha1(tracingImmediate *v1beta1.TracingImmediate, groupBy []string) *Tracing {
	return &Tracing{
		Filters: convertTracingFilterV1beta1ToV1alpha1(tracingImmediate.TracingFilter),
		Conditions: TracingCondition{
			AlertWhen: TracingAlertWhenImmediately,
			GroupBy:   groupBy,
		},
	}
}

func convertFlowV1beta1ToV1alpha1(flow *v1beta1.Flow) *Flow {
	return &Flow{
		Stages: convertFlowStagesV1beta1ToV1alpha1(flow.Stages),
	}
}

func convertMetricThresholdV1beta1ToMetricV1alpha1(metricThreshold *v1beta1.MetricThreshold) *Metric {
	condition := metricThreshold.Rules[0].Condition
	var minNonValuePct *int
	if metricThreshold.MissingValues.MinNonNullValuesPct != nil {
		minNonValuePct = pointer.Int(int(*metricThreshold.MissingValues.MinNonNullValuesPct))
	}

	return &Metric{
		Promql: &Promql{
			SearchQuery: metricThreshold.MetricFilter.Promql,
			Conditions: PromqlConditions{
				AlertWhen:                   metricConditionTypeV1beta1ToV1alpha1[condition.ConditionType],
				Threshold:                   condition.Threshold.DeepCopy(),
				SampleThresholdPercentage:   int(condition.ForOverPct),
				TimeWindow:                  metricTimeWindowV1beta1ToV1alpha1[condition.OfTheLast.SpecificValue],
				MinNonNullValuesPercentage:  minNonValuePct,
				ManageUndetectedValues:      convertUndetectedValuesManagementV1beta1ToV1alpha1(metricThreshold.UndetectedValuesManagement),
				ReplaceMissingValueWithZero: metricThreshold.MissingValues.ReplaceWithZero,
			},
		},
	}
}

func convertMetricAnomalyV1beta1ToMetricV1alpha1(metricAnomaly *v1beta1.MetricAnomaly) *Metric {
	condition := metricAnomaly.Rules[0].Condition

	return &Metric{
		Promql: &Promql{
			SearchQuery: metricAnomaly.MetricFilter.Promql,
			Conditions: PromqlConditions{
				AlertWhen:  metricAnomalyConditionTypeV1beta1ToV1alpha1[condition.ConditionType],
				Threshold:  condition.Threshold.DeepCopy(),
				TimeWindow: metricTimeWindowV1beta1ToV1alpha1[condition.OfTheLast.SpecificValue],
			},
		},
	}
}

func convertLogsNewValueV1beta1ToV1alpha1(logsNewValue *v1beta1.LogsNewValue) *NewValue {
	condition := logsNewValue.Rules[0].Condition
	return &NewValue{
		Filters: convertLogsFilterV1beta1ToV1alpha1(logsNewValue.LogsFilter),
		Conditions: NewValueConditions{
			Key:        condition.KeypathToTrack,
			TimeWindow: newValueTimeWindowV1beta1ToV1alpha[condition.TimeWindow.SpecificValue],
		},
	}
}

func convertRatioV1beta1ToV1alpha1(logsRatioThreshold *v1beta1.LogsRatioThreshold, groupBy []string) *Ratio {
	condition := logsRatioThreshold.Rules[0].Condition

	query1Filters := *convertLogsFilterV1beta1ToV1alpha1(&logsRatioThreshold.Numerator)
	query1Filters.Alias = pointer.String(logsRatioThreshold.NumeratorAlias)

	query2Filters := convertDenominatorV1beta1ToV1alpha1(logsRatioThreshold.Denominator)
	query2Filters.Alias = pointer.String(logsRatioThreshold.DenominatorAlias)

	return &Ratio{
		Query1Filters: query1Filters,
		Query2Filters: query2Filters,
		Conditions: RatioConditions{
			AlertWhen:  ratioConditionTypeV1beta1ToV1alpha1[condition.ConditionType],
			Ratio:      condition.Threshold.DeepCopy(),
			TimeWindow: logsRatioTimeWindowV1beta1ToV1alpha1[condition.TimeWindow.SpecificValue],
			GroupBy:    groupBy,
		},
	}
}

func convertLogsThresholdV1beta1ToStandardV1alpha1(logsThreshold *v1beta1.LogsThreshold, groupBy []string) *Standard {
	conditions := logsThreshold.Rules[0]
	timeWindow := logsTimeWindowV1beta1ToV1alpha[conditions.Condition.TimeWindow.SpecificValue]
	return &Standard{
		Filters: convertLogsFilterV1beta1ToV1alpha1(logsThreshold.LogsFilter),
		Conditions: StandardConditions{
			AlertWhen:              logsConditionTypeV1beta1ToV1alpha[conditions.Condition.LogsThresholdConditionType],
			Threshold:              pointer.Int(int(conditions.Condition.Threshold.Value())),
			TimeWindow:             &timeWindow,
			GroupBy:                groupBy,
			ManageUndetectedValues: convertUndetectedValuesManagementV1beta1ToV1alpha1(logsThreshold.UndetectedValuesManagement),
		},
	}
}

func convertLogImmediateV1beta1ToStandardV1alpha1(logsImmediate *v1beta1.LogsImmediate) *Standard {
	return &Standard{
		Filters: convertLogsFilterV1beta1ToV1alpha1(logsImmediate.LogsFilter),
	}
}

func convertTracingFilterV1beta1ToV1alpha1(filter *v1beta1.TracingFilter) TracingFilters {
	return TracingFilters{
		LatencyThresholdMilliseconds: *resource.NewQuantity(int64(*filter.Simple.LatencyThresholdMs), resource.DecimalSI),
		Applications:                 convertTracingLabelFilterV1beta1ToV1alpha1(filter.Simple.TracingLabelFilters.ApplicationName),
	}
}

func convertTracingLabelFilterV1beta1ToV1alpha1(filters []v1beta1.TracingFilterType) []string {
	var result []string
	for _, filter := range filters {
		switch filter.Operation {
		case v1beta1.TracingFilterOperationTypeOr:
			result = append(result, filter.Values...)
		case v1beta1.TracingFilterOperationTypeIncludes:
			for _, value := range filter.Values {
				result = append(result, "filter:contains:"+value)
			}
		case v1beta1.TracingFilterOperationTypeEndsWith:
			for _, value := range filter.Values {
				result = append(result, "filter:endsWith:"+value)
			}
		case v1beta1.TracingFilterOperationTypeStartsWith:
			for _, value := range filter.Values {
				result = append(result, "filter:startsWith:"+value)
			}
		case v1beta1.TracingFilterOperationTypeIsNot:
			for _, value := range filter.Values {
				result = append(result, "filter:isNot:"+value)
			}
		}
	}
	return result
}

func convertFlowStagesV1beta1ToV1alpha1(stages []v1beta1.FlowStage) []FlowStage {
	result := make([]FlowStage, len(stages))
	for i, stage := range stages {
		result[i] = FlowStage{
			Groups: convertFlowGroupsV1beta1ToV1alpha1(stage.FlowStagesType.Groups),
		}
	}
	return result
}

func convertFlowGroupsV1beta1ToV1alpha1(groups []v1beta1.FlowStageGroup) []FlowStageGroup {
	result := make([]FlowStageGroup, len(groups))
	for i, group := range groups {
		result[i] = FlowStageGroup{
			InnerFlowAlerts: convertFlowAlertsV1beta1ToV1alpha1(group.AlertDefs, group.AlertsOp),
			NextOperator:    flowOperationV1beta1ToV1alpha[group.NextOp],
		}
	}
	return result
}

func convertFlowAlertsV1beta1ToV1alpha1(defs []v1beta1.FlowStagesGroupsAlertDefs, op v1beta1.FlowStageGroupAlertsOp) InnerFlowAlerts {
	innerFlowAlerts := make([]InnerFlowAlert, len(defs))
	for i, def := range defs {
		innerFlowAlerts[i] = InnerFlowAlert{
			Not: def.Not,
		}
		if backendRef := def.AlertRef.BackendRef; backendRef != nil && backendRef.ID != nil {
			innerFlowAlerts[i].UserAlertId = *backendRef.ID
		}
	}
	return InnerFlowAlerts{
		Alerts:   innerFlowAlerts,
		Operator: flowOperationV1beta1ToV1alpha[op],
	}
}

func convertDenominatorV1beta1ToV1alpha1(denominator v1beta1.LogsFilter) RatioQ2Filters {
	return RatioQ2Filters{
		SearchQuery:  denominator.SimpleFilter.LuceneQuery,
		Severities:   convertSeveritiesFilterV1beta1ToV1alpha1(denominator.SimpleFilter.LabelFilters.Severity),
		Applications: convertLabelFilterV1beta1ToV1alpha1(denominator.SimpleFilter.LabelFilters.ApplicationName),
		Subsystems:   convertLabelFilterV1beta1ToV1alpha1(denominator.SimpleFilter.LabelFilters.SubsystemName),
	}
}

func convertUndetectedValuesManagementV1beta1ToV1alpha1(undetectedValues *v1beta1.UndetectedValuesManagement) *ManageUndetectedValues {
	if undetectedValues == nil {
		return nil
	}

	autoRetireRatio := autoRetireRatioV1beta1ToV1alpha1[undetectedValues.AutoRetireTimeframe]

	return &ManageUndetectedValues{
		EnableTriggeringOnUndetectedValues: undetectedValues.TriggerUndetectedValues,
		AutoRetireRatio:                    &autoRetireRatio,
	}
}

func convertLogsFilterV1beta1ToV1alpha1(filter *v1beta1.LogsFilter) *Filters {
	if filter == nil {
		return nil
	}

	filters := &Filters{}
	if filter.SimpleFilter.LuceneQuery != nil {
		filters.SearchQuery = filter.SimpleFilter.LuceneQuery
	}

	if filter.SimpleFilter.LabelFilters != nil {
		filters.Applications = convertLabelFilterV1beta1ToV1alpha1(filter.SimpleFilter.LabelFilters.ApplicationName)
		filters.Subsystems = convertLabelFilterV1beta1ToV1alpha1(filter.SimpleFilter.LabelFilters.SubsystemName)
		filters.Severities = convertSeveritiesFilterV1beta1ToV1alpha1(filter.SimpleFilter.LabelFilters.Severity)
	}

	return filters
}

func convertSeveritiesFilterV1beta1ToV1alpha1(severity []v1beta1.LogSeverity) []FiltersLogSeverity {
	result := make([]FiltersLogSeverity, len(severity))
	for i, s := range severity {
		result[i] = severitiesFilterV1beta1ToV1alpha1[s]
	}
	return result
}

func convertLabelFilterV1beta1ToV1alpha1(labelFilters []v1beta1.LabelFilterType) []string {
	result := make([]string, len(labelFilters))
	for i, labelFilter := range labelFilters {
		switch labelFilter.Operation {
		case v1beta1.LogFilterOperationTypeOr:
			result[i] = labelFilter.Value
		case v1beta1.LogFilterOperationTypeIncludes:
			result[i] = "filter:contains:" + labelFilter.Value
		case v1beta1.LogFilterOperationTypeEndWith:
			result[i] = "filter:endsWith:" + labelFilter.Value
		case v1beta1.LogFilterOperationTypeStartsWith:
			result[i] = "filter:startsWith:" + labelFilter.Value
		}
	}
	return result
}

func convertingNotificationGroupsV1beta1ToV1alpha1(group *v1beta1.NotificationGroup, excess []v1beta1.NotificationGroup) []NotificationGroup {
	if group == nil {
		return nil
	}

	notificationGroups := make([]NotificationGroup, len(excess)+1)
	notificationGroups[0] = convertNotificationGroupV1beta1ToV1alpha1(*group)
	for i, excessGroup := range excess {
		notificationGroups[i+1] = convertNotificationGroupV1beta1ToV1alpha1(excessGroup)
	}

	return notificationGroups
}

func convertNotificationGroupV1beta1ToV1alpha1(group v1beta1.NotificationGroup) NotificationGroup {
	return NotificationGroup{
		GroupByFields: group.GroupByKeys,
		Notifications: convertWebhooksV1beta1ToV1alpha1(group.Webhooks),
	}
}

func convertWebhooksV1beta1ToV1alpha1(webhooks []v1beta1.WebhookSettings) []Notification {
	notifications := make([]Notification, len(webhooks))
	for i, webhook := range webhooks {
		notifications[i] = convertWebhookV1beta1ToV1alpha1(webhook)
	}

	return notifications
}

func convertWebhookV1beta1ToV1alpha1(webhook v1beta1.WebhookSettings) Notification {
	notification := Notification{
		RetriggeringPeriodMinutes: int32(*webhook.RetriggeringPeriod.Minutes),
		NotifyOn:                  notifyOnV1beta1ToV1alpha1[webhook.NotifyOn],
	}
	notification = convertToIntegrationTypeV1beta1ToV1alpha1(webhook.Integration, &notification)
	return notification
}

func convertToIntegrationTypeV1beta1ToV1alpha1(integration v1beta1.IntegrationType, notification *Notification) Notification {
	if integration.IntegrationRef != nil {
		notification.IntegrationName = convertIntegrationRefV1beta1ToV1alpha1IntegrationName(*integration.IntegrationRef)
	} else if integration.Recipients != nil {
		notification.EmailRecipients = integration.Recipients
	}

	return *notification
}

func convertIntegrationRefV1beta1ToV1alpha1IntegrationName(integrationRef v1beta1.IntegrationRef) *string {
	if integrationRef.BackendRef != nil {
		return integrationRef.BackendRef.Name
	}
	return nil
}

func convertSchedulingV1beta1ToV1alpha1(schedule *v1beta1.AlertSchedule) *Scheduling {
	if schedule == nil {
		return nil
	}

	return &Scheduling{
		DaysEnabled: convertDaysOfWeekV1beta1ToV1alpha1(schedule.ActiveOn.DayOfWeek),
		StartTime:   convertTimeV1beta1ToV1alpha1(schedule.ActiveOn.StartTime),
		EndTime:     convertTimeV1beta1ToV1alpha1(schedule.ActiveOn.EndTime),
		TimeZone:    TimeZone(schedule.TimeZone),
	}
}

func convertTimeV1beta1ToV1alpha1(time *v1beta1.TimeOfDay) *Time {
	if time == nil {
		return nil
	}

	timeOfDay := *time
	return (*Time)(&timeOfDay)
}

func convertDaysOfWeekV1beta1ToV1alpha1(week []v1beta1.DayOfWeek) []Day {
	result := make([]Day, len(week))
	for i, day := range week {
		result[i] = dayOfWeekV1beta1ToV1alpha1[day]
	}

	return result
}

func convertSchedulingV1alpha1ToV1beta1(scheduling *Scheduling) *v1beta1.AlertSchedule {
	if scheduling == nil {
		return nil
	}

	return &v1beta1.AlertSchedule{
		TimeZone: v1beta1.TimeZone(scheduling.TimeZone),
		ActiveOn: &v1beta1.ActiveOn{
			DayOfWeek: convertDaysOfWeekV1alpha1ToV1beta1(scheduling.DaysEnabled),
			StartTime: convertTimeV1alpha1ToV1beta1(scheduling.StartTime),
			EndTime:   convertTimeV1alpha1ToV1beta1(scheduling.EndTime),
		},
	}
}

func convertAlertTypeV1alpha1ToV1beta1(srcSpec AlertSpec) (v1beta1.AlertTypeDefinition, []string) {
	if standard := srcSpec.AlertType.Standard; standard != nil {
		return convertStandardV1alpha1ToV1beta1(standard, srcSpec.PayloadFilters)
	} else if flow := srcSpec.AlertType.Flow; flow != nil {
		return convertFlowV1alpha1ToV1beta1(flow), nil
	} else if metric := srcSpec.AlertType.Metric; metric != nil {
		if promql := metric.Promql; promql != nil {
			return convertPromqlV1alpha1ToV1beta1(promql), nil
		}
	} else if newValue := srcSpec.AlertType.NewValue; newValue != nil {
		return convertNewValueV1alpha1ToV1beta1(newValue, srcSpec.PayloadFilters), nil
	} else if tracing := srcSpec.AlertType.Tracing; tracing != nil {
		return convertTracingV1alpha1ToV1beta1(tracing, srcSpec.PayloadFilters)
	} else if ratio := srcSpec.AlertType.Ratio; ratio != nil {
		return convertRatioV1alpha1ToV1beta1(ratio)
	} else if timeRelative := srcSpec.AlertType.TimeRelative; timeRelative != nil {
		return convertTimeRelativeV1alpha1ToV1beta1(timeRelative, srcSpec.PayloadFilters)
	} else if uniqueCount := srcSpec.AlertType.UniqueCount; uniqueCount != nil {
		return convertUniqueCountV1alpha1ToV1beta1(uniqueCount, srcSpec.PayloadFilters)
	}

	return v1beta1.AlertTypeDefinition{}, nil
}

func convertUniqueCountV1alpha1ToV1beta1(uniqueCount *UniqueCount, payloadFilters []string) (v1beta1.AlertTypeDefinition, []string) {
	condition := uniqueCount.Conditions

	var groupBy []string
	if condition.GroupBy != nil {
		groupBy = []string{*condition.GroupBy}
	}

	return v1beta1.AlertTypeDefinition{
		LogsUniqueCount: &v1beta1.LogsUniqueCount{
			LogsFilter:                  convertLogsFilterV1alpha1ToV1beta1(uniqueCount.Filters),
			NotificationPayloadFilter:   payloadFilters,
			MaxUniqueCountPerGroupByKey: pointer.Uint64(uint64(condition.MaxUniqueValues)),
			UniqueCountKeypath:          condition.Key,
			Rules: []v1beta1.LogsUniqueCountRule{
				convertUniqueCountConditionsV1alpha1ToV1beta1(condition),
			},
		},
	}, groupBy
}

func convertTimeRelativeV1alpha1ToV1beta1(timeRelative *TimeRelative, payloadFilters []string) (v1beta1.AlertTypeDefinition, []string) {
	return v1beta1.AlertTypeDefinition{
		LogsTimeRelativeThreshold: &v1beta1.LogsTimeRelativeThreshold{
			LogsFilter:                 *convertLogsFilterV1alpha1ToV1beta1(timeRelative.Filters),
			IgnoreInfinity:             timeRelative.Conditions.IgnoreInfinity,
			NotificationPayloadFilter:  payloadFilters,
			UndetectedValuesManagement: convertUndetectedValuesManagementV1alpha1ToV1beta1(timeRelative.Conditions.ManageUndetectedValues),
			Rules: []v1beta1.LogsTimeRelativeRule{
				convertTimeRelativeConditionsV1alpha1ToV1beta1(timeRelative.Conditions),
			},
		},
	}, timeRelative.Conditions.GroupBy
}

func convertRatioV1alpha1ToV1beta1(ratio *Ratio) (v1beta1.AlertTypeDefinition, []string) {
	return v1beta1.AlertTypeDefinition{
		LogsRatioThreshold: &v1beta1.LogsRatioThreshold{
			Numerator:        *convertLogsFilterV1alpha1ToV1beta1(&ratio.Query1Filters),
			NumeratorAlias:   *ratio.Query1Filters.Alias,
			Denominator:      *convertDenominatorV1alpha1ToV1beta1(&ratio.Query2Filters),
			DenominatorAlias: *ratio.Query2Filters.Alias,
			Rules: []v1beta1.LogsRatioThresholdRule{
				covertRatioConditionsV1alpha1ToV1beta1(ratio.Conditions),
			},
		},
	}, ratio.Conditions.GroupBy
}

func convertTracingV1alpha1ToV1beta1(tracing *Tracing, payloadFilters []string) (v1beta1.AlertTypeDefinition, []string) {
	if tracing.Conditions.AlertWhen == TracingAlertWhenImmediately {
		return v1beta1.AlertTypeDefinition{
			TracingImmediate: &v1beta1.TracingImmediate{
				NotificationPayloadFilter: payloadFilters,
				TracingFilter:             convertTracingFilterV1alpha1ToV1beta1(tracing.Filters),
			},
		}, tracing.Conditions.GroupBy
	} else {
		return v1beta1.AlertTypeDefinition{
			TracingThreshold: &v1beta1.TracingThreshold{
				NotificationPayloadFilter: payloadFilters,
				TracingFilter:             convertTracingFilterV1alpha1ToV1beta1(tracing.Filters),
				Rules: []v1beta1.TracingThresholdRule{
					convertTracingConditionsV1alpha1ToV1beta1(tracing.Filters, tracing.Conditions),
				},
			},
		}, tracing.Conditions.GroupBy
	}
}

func convertNewValueV1alpha1ToV1beta1(newValue *NewValue, payloadFilters []string) v1beta1.AlertTypeDefinition {
	return v1beta1.AlertTypeDefinition{
		LogsNewValue: &v1beta1.LogsNewValue{
			LogsFilter: convertLogsFilterV1alpha1ToV1beta1(newValue.Filters),
			Rules: []v1beta1.LogsNewValueRule{
				convertNewValueConditionsV1alpha1ToV1beta1(newValue.Conditions),
			},
			NotificationPayloadFilter: payloadFilters,
		},
	}
}

func convertFlowV1alpha1ToV1beta1(flow *Flow) v1beta1.AlertTypeDefinition {
	return v1beta1.AlertTypeDefinition{
		Flow: &v1beta1.Flow{
			Stages: convertFlowStagesV1alpha1ToV1beta1(flow.Stages),
		},
	}
}

func convertStandardV1alpha1ToV1beta1(standard *Standard, payloadFilters []string) (v1beta1.AlertTypeDefinition, []string) {
	switch standard.Conditions.AlertWhen {
	case StandardAlertWhenImmediately:
		return convertStandardImmediateV1alpha1toV1beta1(standard, payloadFilters), nil
	case StandardAlertWhenMoreThanUsual:
		return convertStandardAnomalyV1alpha1ToV1beta1(standard, payloadFilters), standard.Conditions.GroupBy
	default:
		return convertStandardThresholdV1alpha1ToV1beta1(standard, payloadFilters), standard.Conditions.GroupBy
	}
}

func convertStandardThresholdV1alpha1ToV1beta1(standard *Standard, payloadFilters []string) v1beta1.AlertTypeDefinition {
	return v1beta1.AlertTypeDefinition{
		LogsThreshold: &v1beta1.LogsThreshold{
			NotificationPayloadFilter:  payloadFilters,
			LogsFilter:                 convertLogsFilterV1alpha1ToV1beta1(standard.Filters),
			UndetectedValuesManagement: convertUndetectedValuesManagementV1alpha1ToV1beta1(standard.Conditions.ManageUndetectedValues),
			Rules: []v1beta1.LogsThresholdRule{
				convertStandardConditionsV1alpha1ToLogsThresholdRuleV1beta1(standard.Conditions),
			},
		},
	}
}

func convertStandardImmediateV1alpha1toV1beta1(standard *Standard, payloadFilters []string) v1beta1.AlertTypeDefinition {
	return v1beta1.AlertTypeDefinition{
		LogsImmediate: &v1beta1.LogsImmediate{
			NotificationPayloadFilter: payloadFilters,
			LogsFilter:                convertLogsFilterV1alpha1ToV1beta1(standard.Filters),
		},
	}
}

func convertStandardAnomalyV1alpha1ToV1beta1(standard *Standard, payloadFilters []string) v1beta1.AlertTypeDefinition {
	return v1beta1.AlertTypeDefinition{
		LogsAnomaly: &v1beta1.LogsAnomaly{
			LogsFilter: convertLogsFilterV1alpha1ToV1beta1(standard.Filters),
			Rules: []v1beta1.LogsAnomalyRule{
				convertStandardConditionsV1alpha1ToLogsAnomalyRuleV1beta1(standard.Conditions),
			},
			NotificationPayloadFilter: payloadFilters,
		},
	}
}

func convertStandardConditionsV1alpha1ToLogsAnomalyRuleV1beta1(conditions StandardConditions) v1beta1.LogsAnomalyRule {
	var minimumThreshold resource.Quantity
	if conditions.Threshold != nil {
		minimumThreshold = *resource.NewQuantity(int64(*conditions.Threshold), resource.DecimalSI)
	}
	return v1beta1.LogsAnomalyRule{
		Condition: v1beta1.LogsAnomalyCondition{
			MinimumThreshold: minimumThreshold,
			TimeWindow: v1beta1.LogsTimeWindow{
				SpecificValue: logsTimeWindowV1alpha1ToV1beta1[*conditions.TimeWindow],
			},
		},
	}
}

func convertPromqlV1alpha1ToV1beta1(promql *Promql) v1beta1.AlertTypeDefinition {
	switch promql.Conditions.AlertWhen {
	case PromqlAlertWhenMoreThanUsual, PromqlAlertWhenLessThanUsual:
		return convertPromqlMoreThanUsualV1alpha1ToV1beta1(promql)
	default:
		return convertPromqlThresholdV1alpha1ToV1beta1(promql)
	}
}

func convertPromqlThresholdV1alpha1ToV1beta1(promql *Promql) v1beta1.AlertTypeDefinition {
	return v1beta1.AlertTypeDefinition{
		MetricThreshold: &v1beta1.MetricThreshold{
			MetricFilter: v1beta1.MetricFilter{
				Promql: promql.SearchQuery,
			},
			Rules: []v1beta1.MetricThresholdRule{
				convertPromqlConditionsV1alpha1ToMetricThresholdRuleV1beta1(promql.Conditions),
			},
			MissingValues:              convertMissingValuesV1alpha1ToV1beta1(promql.Conditions),
			UndetectedValuesManagement: convertUndetectedValuesManagementV1alpha1ToV1beta1(promql.Conditions.ManageUndetectedValues),
		},
	}
}

func convertMissingValuesV1alpha1ToV1beta1(conditions PromqlConditions) v1beta1.MetricMissingValues {
	metricMissingValues := v1beta1.MetricMissingValues{}
	if conditions.ReplaceMissingValueWithZero {
		metricMissingValues.ReplaceWithZero = true
	} else if conditions.MinNonNullValuesPercentage != nil {
		metricMissingValues.MinNonNullValuesPct = pointer.Uint32(uint32(*conditions.MinNonNullValuesPercentage))
	}
	return metricMissingValues
}

func convertPromqlMoreThanUsualV1alpha1ToV1beta1(promql *Promql) v1beta1.AlertTypeDefinition {
	return v1beta1.AlertTypeDefinition{
		MetricAnomaly: &v1beta1.MetricAnomaly{
			MetricFilter: v1beta1.MetricFilter{
				Promql: promql.SearchQuery,
			},
			Rules: []v1beta1.MetricAnomalyRule{
				convertPromqlConditionsV1alpha1ToMetricAnomalyRuleV1beta1(promql.Conditions),
			},
		},
	}
}

func convertUniqueCountConditionsV1alpha1ToV1beta1(conditions UniqueCountConditions) v1beta1.LogsUniqueCountRule {
	return v1beta1.LogsUniqueCountRule{
		Condition: v1beta1.LogsUniqueCountCondition{
			Threshold: int64(conditions.MaxUniqueValues),
			TimeWindow: v1beta1.LogsUniqueCountTimeWindow{
				SpecificValue: logsUniqueCountTimeWindowV1alpha1ToV1beta1[conditions.TimeWindow],
			},
		},
	}
}

func convertTimeRelativeConditionsV1alpha1ToV1beta1(conditions TimeRelativeConditions) v1beta1.LogsTimeRelativeRule {
	return v1beta1.LogsTimeRelativeRule{
		Condition: v1beta1.LogsTimeRelativeCondition{
			Threshold:     conditions.Threshold.DeepCopy(),
			ComparedTo:    timeRelativeTimeWindowV1alpha1ToV1beta1[conditions.TimeWindow],
			ConditionType: timeRelativeConditionTypeV1alpha1ToV1beta1[conditions.AlertWhen],
		},
	}
}

func covertRatioConditionsV1alpha1ToV1beta1(conditions RatioConditions) v1beta1.LogsRatioThresholdRule {
	return v1beta1.LogsRatioThresholdRule{
		Condition: v1beta1.LogsRatioCondition{
			Threshold: conditions.Ratio.DeepCopy(),
			TimeWindow: v1beta1.LogsRatioTimeWindow{
				SpecificValue: logsRatioTimeWindowV1alpha1ToV1beta1[conditions.TimeWindow],
			},
			ConditionType: ratioConditionTypeV1alpha1ToV1beta1[conditions.AlertWhen],
		},
	}
}

func convertDenominatorV1alpha1ToV1beta1(denominator *RatioQ2Filters) *v1beta1.LogsFilter {
	if denominator == nil {
		return nil
	}

	return &v1beta1.LogsFilter{
		SimpleFilter: v1beta1.LogsSimpleFilter{
			LuceneQuery: denominator.SearchQuery,
			LabelFilters: &v1beta1.LabelFilters{
				ApplicationName: convertLabelFilterV1alpha1ToV1beta1(denominator.Applications),
				SubsystemName:   convertLabelFilterV1alpha1ToV1beta1(denominator.Subsystems),
				Severity:        convertSeveritiesFilterV1alpha1ToV1beta1(denominator.Severities),
			},
		},
	}
}

func convertTracingConditionsV1alpha1ToV1beta1(filters TracingFilters, conditions TracingCondition) v1beta1.TracingThresholdRule {
	return v1beta1.TracingThresholdRule{
		Condition: v1beta1.TracingThresholdRuleCondition{
			SpanAmount: filters.LatencyThresholdMilliseconds.DeepCopy(),
			TimeWindow: v1beta1.TracingTimeWindow{
				SpecificValue: tracingTimeWindowV1alpha1ToV1beta1[*conditions.TimeWindow],
			},
		},
	}
}

func convertTracingFilterV1alpha1ToV1beta1(filters TracingFilters) *v1beta1.TracingFilter {
	return &v1beta1.TracingFilter{
		Simple: &v1beta1.TracingSimpleFilter{
			TracingLabelFilters: &v1beta1.TracingLabelFilters{
				ApplicationName: convertTracingFilterTypeV1alpha1ToV1beta1(filters.Applications),
			},
			LatencyThresholdMs: pointer.Uint64(uint64(filters.LatencyThresholdMilliseconds.Value())),
		},
	}
}

func convertTracingFilterTypeV1alpha1ToV1beta1(labels []string) []v1beta1.TracingFilterType {
	filterTypeOperationToValues := map[v1beta1.TracingFilterOperationType][]string{
		v1beta1.TracingFilterOperationTypeOr:         {},
		v1beta1.TracingFilterOperationTypeIncludes:   {},
		v1beta1.TracingFilterOperationTypeEndsWith:   {},
		v1beta1.TracingFilterOperationTypeStartsWith: {},
		v1beta1.TracingFilterOperationTypeIsNot:      {},
	}

	for _, label := range labels {
		if value, prefixExist := strings.CutPrefix(label, "filter:contains:"); prefixExist {
			filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeIncludes] = append(filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeIncludes], value)
		} else if value, prefixExist = strings.CutPrefix(label, "filter:startsWith:"); prefixExist {
			filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeStartsWith] = append(filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeStartsWith], value)
		} else if value, prefixExist = strings.CutPrefix(label, "filter:endsWith:"); prefixExist {
			filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeEndsWith] = append(filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeEndsWith], value)
		} else {
			filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeOr] = append(filterTypeOperationToValues[v1beta1.TracingFilterOperationTypeOr], label)
		}
	}

	result := make([]v1beta1.TracingFilterType, 0)
	for operation, values := range filterTypeOperationToValues {
		if len(values) > 0 {
			result = append(result, v1beta1.TracingFilterType{
				Operation: operation,
				Values:    values,
			})
		}
	}

	return result
}

func convertNewValueConditionsV1alpha1ToV1beta1(conditions NewValueConditions) v1beta1.LogsNewValueRule {
	return v1beta1.LogsNewValueRule{
		Condition: v1beta1.LogsNewValueRuleCondition{
			KeypathToTrack: conditions.Key,
			TimeWindow: v1beta1.LogsNewValueTimeWindow{
				SpecificValue: newValueTimeWindowV1alpha1ToV1beta1[conditions.TimeWindow],
			},
		},
	}
}

func convertPromqlConditionsV1alpha1ToMetricThresholdRuleV1beta1(conditions PromqlConditions) v1beta1.MetricThresholdRule {
	return v1beta1.MetricThresholdRule{
		Condition: v1beta1.MetricThresholdRuleCondition{
			Threshold:  conditions.Threshold.DeepCopy(),
			ForOverPct: uint32(conditions.SampleThresholdPercentage),
			OfTheLast: v1beta1.MetricTimeWindow{
				SpecificValue: metricTimeWindowV1alpha1ToV1beta1[conditions.TimeWindow],
			},
			ConditionType: metricConditionTypeV1alpha1ToV1beta1[conditions.AlertWhen],
		},
	}
}

func convertPromqlConditionsV1alpha1ToMetricAnomalyRuleV1beta1(conditions PromqlConditions) v1beta1.MetricAnomalyRule {
	return v1beta1.MetricAnomalyRule{
		Condition: v1beta1.MetricAnomalyCondition{
			Threshold:  conditions.Threshold.DeepCopy(),
			ForOverPct: uint32(conditions.SampleThresholdPercentage),
			OfTheLast: v1beta1.MetricTimeWindow{
				SpecificValue: metricTimeWindowV1alpha1ToV1beta1[conditions.TimeWindow],
			},
			ConditionType: metricAnomalyConditionTypeV1alpha1ToV1beta1[conditions.AlertWhen],
		},
	}
}

func convertFlowStagesV1alpha1ToV1beta1(stages []FlowStage) []v1beta1.FlowStage {
	result := make([]v1beta1.FlowStage, len(stages))
	for i, stage := range stages {
		result[i] = v1beta1.FlowStage{
			FlowStagesType: v1beta1.FlowStagesType{
				Groups: convertFlowStageGroupsV1alpha1ToV1beta1(stage.Groups),
			},
			TimeframeType: v1beta1.TimeframeTypeUpTo,
			TimeframeMs:   convertFlowStateTimeWindowV1alpha1ToV1beta1(stage.TimeWindow),
		}
	}
	return result
}

func convertFlowStateTimeWindowV1alpha1ToV1beta1(timeWindow *FlowStageTimeFrame) int64 {
	if timeWindow == nil {
		return 0
	}

	return int64(msInHour*timeWindow.Hours + msInMinute*timeWindow.Minutes + msInSecond*timeWindow.Seconds)
}

func convertFlowStageGroupsV1alpha1ToV1beta1(groups []FlowStageGroup) []v1beta1.FlowStageGroup {
	result := make([]v1beta1.FlowStageGroup, len(groups))
	for i, group := range groups {
		result[i] = v1beta1.FlowStageGroup{
			NextOp:    flowOperationV1alpha1ToV1beta1[group.NextOperator],
			AlertsOp:  flowOperationV1alpha1ToV1beta1[group.InnerFlowAlerts.Operator],
			AlertDefs: convertFlowAlertDefsV1alpha1ToV1beta1(group.InnerFlowAlerts.Alerts),
		}
	}
	return result
}

func convertFlowAlertDefsV1alpha1ToV1beta1(alerts []InnerFlowAlert) []v1beta1.FlowStagesGroupsAlertDefs {
	result := make([]v1beta1.FlowStagesGroupsAlertDefs, len(alerts))
	for i, alert := range alerts {
		result[i] = v1beta1.FlowStagesGroupsAlertDefs{
			AlertRef: v1beta1.AlertRef{BackendRef: &v1beta1.AlertBackendRef{ID: &alert.UserAlertId}},
			Not:      alert.Not,
		}
	}
	return result
}

func convertStandardConditionsV1alpha1ToLogsThresholdRuleV1beta1(conditions StandardConditions) v1beta1.LogsThresholdRule {
	return v1beta1.LogsThresholdRule{
		Condition: v1beta1.LogsThresholdRuleCondition{
			TimeWindow: v1beta1.LogsTimeWindow{
				SpecificValue: logsTimeWindowV1alpha1ToV1beta1[*conditions.TimeWindow],
			},
			Threshold:                  *resource.NewQuantity(int64(*conditions.Threshold), resource.DecimalSI),
			LogsThresholdConditionType: logsConditionTypeV1alpha1ToV1beta1[conditions.AlertWhen],
		},
	}
}

func convertUndetectedValuesManagementV1alpha1ToV1beta1(manageUndetectedValues *ManageUndetectedValues) *v1beta1.UndetectedValuesManagement {
	if manageUndetectedValues == nil {
		return nil
	}

	return &v1beta1.UndetectedValuesManagement{
		TriggerUndetectedValues: manageUndetectedValues.EnableTriggeringOnUndetectedValues,
		AutoRetireTimeframe:     autoRetireRatioV1alpha1ToV1beta1[*manageUndetectedValues.AutoRetireRatio],
	}
}

func convertLogsFilterV1alpha1ToV1beta1(filters *Filters) *v1beta1.LogsFilter {
	if filters == nil {
		return nil
	}

	return &v1beta1.LogsFilter{
		SimpleFilter: v1beta1.LogsSimpleFilter{
			LuceneQuery: filters.SearchQuery,
			LabelFilters: &v1beta1.LabelFilters{
				ApplicationName: convertLabelFilterV1alpha1ToV1beta1(filters.Applications),
				SubsystemName:   convertLabelFilterV1alpha1ToV1beta1(filters.Subsystems),
				Severity:        convertSeveritiesFilterV1alpha1ToV1beta1(filters.Severities),
			},
		},
	}
}

func convertSeveritiesFilterV1alpha1ToV1beta1(severities []FiltersLogSeverity) []v1beta1.LogSeverity {
	result := make([]v1beta1.LogSeverity, len(severities))
	for i, severity := range severities {
		result[i] = severitiesFilterV1alpha1ToV1beta1[severity]
	}
	return result
}

func convertLabelFilterV1alpha1ToV1beta1(labels []string) []v1beta1.LabelFilterType {
	result := make([]v1beta1.LabelFilterType, len(labels))

	for i, label := range labels {
		if value, prefixExist := strings.CutPrefix(label, "filter:contains:"); prefixExist {
			result[i] = v1beta1.LabelFilterType{
				Value:     value,
				Operation: v1beta1.LogFilterOperationTypeIncludes,
			}
		} else if value, prefixExist = strings.CutPrefix(label, "filter:startsWith:"); prefixExist {
			result[i] = v1beta1.LabelFilterType{
				Value:     value,
				Operation: v1beta1.LogFilterOperationTypeStartsWith,
			}
		} else if value, prefixExist = strings.CutPrefix(label, "filter:endsWith:"); prefixExist {
			result[i] = v1beta1.LabelFilterType{
				Value:     value,
				Operation: v1beta1.LogFilterOperationTypeEndWith,
			}
		} else {
			result[i] = v1beta1.LabelFilterType{
				Value:     label,
				Operation: v1beta1.LogFilterOperationTypeOr,
			}
		}
	}
	return result
}

func convertTimeV1alpha1ToV1beta1(time *Time) *v1beta1.TimeOfDay {
	if time == nil {
		return nil
	}

	timeOfDay := *time
	return (*v1beta1.TimeOfDay)(&timeOfDay)
}

func convertDaysOfWeekV1alpha1ToV1beta1(enabled []Day) []v1beta1.DayOfWeek {
	rslt := make([]v1beta1.DayOfWeek, len(enabled))
	for i, day := range enabled {
		rslt[i] = dayOfWeekV1alpha1ToV1beta1[day]
	}
	return rslt
}

func convertNotificationGroupExcessV1alpha1ToV1beta1(groups []NotificationGroup) []v1beta1.NotificationGroup {
	rslt := make([]v1beta1.NotificationGroup, len(groups))
	for i, group := range groups {
		rslt[i] = *convertNotificationGroupsV1alpha1ToV1beta1(group)
	}
	return rslt
}

func convertNotificationGroupsV1alpha1ToV1beta1(group NotificationGroup) *v1beta1.NotificationGroup {
	return &v1beta1.NotificationGroup{
		GroupByKeys: group.GroupByFields,
		Webhooks:    convertWebhooksV1alpha1ToV1beta1(group.Notifications),
	}
}

func convertWebhooksV1alpha1ToV1beta1(notifications []Notification) []v1beta1.WebhookSettings {
	rslt := make([]v1beta1.WebhookSettings, len(notifications))
	for i, notification := range notifications {
		rslt[i] = convertWebhookV1alpha1ToV1beta1(notification)
	}
	return rslt
}

func convertWebhookV1alpha1ToV1beta1(notification Notification) v1beta1.WebhookSettings {
	return v1beta1.WebhookSettings{
		RetriggeringPeriod: v1beta1.RetriggeringPeriod{
			Minutes: pointer.Uint32((uint32)(notification.RetriggeringPeriodMinutes)),
		},
		NotifyOn:    notifyOnV1alpha1ToV1beta1[notification.NotifyOn],
		Integration: convertToIntegrationTypeV1alpha1ToV1beta1(notification),
	}
}

func convertToIntegrationTypeV1alpha1ToV1beta1(notification Notification) v1beta1.IntegrationType {
	if integrationName := notification.IntegrationName; integrationName != nil {
		return v1beta1.IntegrationType{
			IntegrationRef: &v1beta1.IntegrationRef{
				BackendRef: &v1beta1.OutboundWebhookBackendRef{
					Name: pointer.String(*integrationName),
				},
			},
		}
	}
	return v1beta1.IntegrationType{
		Recipients: notification.EmailRecipients,
	}
}
