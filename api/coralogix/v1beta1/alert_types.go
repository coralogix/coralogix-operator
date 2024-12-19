/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	utils "github.com/coralogix/coralogix-operator/api"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// Alert is the Schema for the alerts API.
type Alert struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertSpec   `json:"spec,omitempty"`
	Status AlertStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertList contains a list of Alert.
type AlertList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Alert `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Alert{}, &AlertList{})
}

var (
	AlertPriorityToProtoPriority = map[AlertPriority]cxsdk.AlertDefPriority{
		AlertPriorityP1: cxsdk.AlertDefPriorityP1,
		AlertPriorityP2: cxsdk.AlertDefPriorityP2,
		AlertPriorityP3: cxsdk.AlertDefPriorityP3,
		AlertPriorityP4: cxsdk.AlertDefPriorityP4,
	}
	LogSeverityToProtoSeverity = map[LogSeverity]cxsdk.LogSeverity{
		LogSeverityDebug:    cxsdk.LogSeverityDebug,
		LogSeverityInfo:     cxsdk.LogSeverityInfo,
		LogSeverityWarning:  cxsdk.LogSeverityWarning,
		LogSeverityError:    cxsdk.LogSeverityError,
		LogSeverityCritical: cxsdk.LogSeverityCritical,
	}
	LogsFiltersOperationToProtoOperation = map[LogFilterOperationType]cxsdk.LogFilterOperationType{
		LogFilterOperationTypeOr:         cxsdk.LogFilterOperationIsOrUnspecified,
		LogFilterOperationTypeIncludes:   cxsdk.LogFilterOperationIncludes,
		LogFilterOperationTypeEndWith:    cxsdk.LogFilterOperationEndsWith,
		LogFilterOperationTypeStartsWith: cxsdk.LogFilterOperationStartsWith,
	}
	DaysOfWeekToProtoDayOfWeek = map[DayOfWeek]cxsdk.AlertDayOfWeek{
		DayOfWeekSunday:    cxsdk.AlertDayOfWeekSunday,
		DayOfWeekMonday:    cxsdk.AlertDayOfWeekMonday,
		DayOfWeekTuesday:   cxsdk.AlertDayOfWeekTuesday,
		DayOfWeekWednesday: cxsdk.AlertDayOfWeekWednesday,
		DayOfWeekThursday:  cxsdk.AlertDayOfWeekThursday,
		DayOfWeekFriday:    cxsdk.AlertDayOfWeekFriday,
		DayOfWeekSaturday:  cxsdk.AlertDayOfWeekSaturday,
	}
	NotifyOnToProtoNotifyOn = map[NotifyOn]cxsdk.AlertNotifyOn{
		NotifyOnTriggeredOnly:        cxsdk.AlertNotifyOnTriggeredOnlyUnspecified,
		NotifyOnTriggeredAndResolved: cxsdk.AlertNotifyOnTriggeredAndResolved,
	}
	AutoRetireTimeframeToProtoAutoRetireTimeframe = map[AutoRetireTimeframe]cxsdk.AutoRetireTimeframe{
		AutoRetireTimeframeNeverOrUnspecified: cxsdk.AutoRetireTimeframeNeverOrUnspecified,
		AutoRetireTimeframe5M:                 cxsdk.AutoRetireTimeframe5Minutes,
		AutoRetireTimeframe10M:                cxsdk.AutoRetireTimeframe10Minutes,
		AutoRetireTimeframe1H:                 cxsdk.AutoRetireTimeframe1Hour,
		AutoRetireTimeframe2H:                 cxsdk.AutoRetireTimeframe2Hours,
		AutoRetireTimeframe6H:                 cxsdk.AutoRetireTimeframe6Hours,
		AutoRetireTimeframe12H:                cxsdk.AutoRetireTimeframe12Hours,
		AutoRetireTimeframe24H:                cxsdk.AutoRetireTimeframe24Hours,
	}
	LogsTimeWindowToProto = map[LogsTimeWindowValue]cxsdk.LogsTimeWindowValue{
		LogsTimeWindowLast5Minutes:  cxsdk.LogsTimeWindowValue5MinutesOrUnspecified,
		LogsTimeWindowLast10Minutes: cxsdk.LogsTimeWindow10Minutes,
		LogsTimeWindowLast15Minutes: cxsdk.LogsTimeWindow15Minutes,
		LogsTimeWindowLast30Minutes: cxsdk.LogsTimeWindow30Minutes,
		LogsTimeWindowLastHour:      cxsdk.LogsTimeWindow1Hour,
		LogsTimeWindowLast2Hours:    cxsdk.LogsTimeWindow2Hours,
		LogsTimeWindowLast6Hours:    cxsdk.LogsTimeWindow6Hours,
		LogsTimeWindowLast12Hours:   cxsdk.LogsTimeWindow12Hours,
		LogsTimeWindowLast24Hours:   cxsdk.LogsTimeWindow24Hours,
		LogsTimeWindowLast36Hours:   cxsdk.LogsTimeWindow36Hours,
	}
	LogsThresholdConditionTypeToProto = map[LogsThresholdConditionType]cxsdk.LogsThresholdConditionType{
		LogsThresholdConditionTypeMoreThan: cxsdk.LogsThresholdConditionTypeMoreThanOrUnspecified,
		LogsThresholdConditionTypeLessThan: cxsdk.LogsThresholdConditionTypeLessThan,
	}
	LogsRatioTimeWindowToProto = map[LogsRatioTimeWindow]cxsdk.LogsRatioTimeWindowValue{
		LogsRatioTimeWindowMinutes5:  cxsdk.LogsRatioTimeWindowValue5MinutesOrUnspecified,
		LogsRatioTimeWindowMinutes10: cxsdk.LogsRatioTimeWindowValue10Minutes,
		LogsRatioTimeWindowMinutes15: cxsdk.LogsRatioTimeWindowValue15Minutes,
		LogsRatioTimeWindowMinutes30: cxsdk.LogsRatioTimeWindowValue30Minutes,
		LogsRatioTimeWindowHour1:     cxsdk.LogsRatioTimeWindowValue1Hour,
		LogsRatioTimeWindowHours2:    cxsdk.LogsRatioTimeWindowValue2Hours,
		LogsRatioTimeWindowHours4:    cxsdk.LogsRatioTimeWindowValue4Hours,
		LogsRatioTimeWindowHours6:    cxsdk.LogsRatioTimeWindowValue6Hours,
		LogsRatioTimeWindowHours12:   cxsdk.LogsRatioTimeWindowValue12Hours,
		LogsRatioTimeWindowHours24:   cxsdk.LogsRatioTimeWindowValue24Hours,
		LogsRatioTimeWindowHours36:   cxsdk.LogsRatioTimeWindowValue36Hours,
	}
	LogsRatioConditionTypeToProto = map[LogsRatioConditionType]cxsdk.LogsRatioConditionType{
		LogsRatioConditionTypeMoreThan: cxsdk.LogsRatioConditionTypeMoreThanOrUnspecified,
		LogsRatioConditionTypeLessThan: cxsdk.LogsRatioConditionTypeLessThan,
	}
	LogsTimeRelativeComparedToToProto = map[LogsTimeRelativeComparedTo]cxsdk.LogsTimeRelativeComparedTo{
		LogsTimeRelativeComparedToPreviousHour:      cxsdk.LogsTimeRelativeComparedToPreviousHourOrUnspecified,
		LogsTimeRelativeComparedToSameHourYesterday: cxsdk.LogsTimeRelativeComparedToSameHourYesterday,
		LogsTimeRelativeComparedToSameHourLastWeek:  cxsdk.LogsTimeRelativeComparedToSameHourLastWeek,
		LogsTimeRelativeComparedToYesterday:         cxsdk.LogsTimeRelativeComparedToYesterday,
		LogsTimeRelativeComparedToSameDayLastWeek:   cxsdk.LogsTimeRelativeComparedToSameDayLastWeek,
		LogsTimeRelativeComparedToSameDayLastMonth:  cxsdk.LogsTimeRelativeComparedToSameDayLastMonth,
	}
	LogsTimeRelativeConditionTypeToProto = map[LogsTimeRelativeConditionType]cxsdk.LogsTimeRelativeConditionType{
		LogsTimeRelativeConditionTypeMoreThan: cxsdk.LogsTimeRelativeConditionTypeMoreThanOrUnspecified,
		LogsTimeRelativeConditionTypeLessThan: cxsdk.LogsTimeRelativeConditionTypeLessThan,
	}
	MetricThresholdConditionTypeToProto = map[MetricThresholdConditionType]cxsdk.MetricThresholdConditionType{
		MetricThresholdConditionTypeMoreThan:         cxsdk.MetricThresholdConditionTypeMoreThanOrUnspecified,
		MetricThresholdConditionTypeLessThan:         cxsdk.MetricThresholdConditionTypeLessThanOrEquals,
		MetricThresholdConditionTypeMoreThanOrEquals: cxsdk.MetricThresholdConditionTypeMoreThanOrEquals,
		MetricThresholdConditionTypeLessThanOrEquals: cxsdk.MetricThresholdConditionTypeLessThanOrEquals,
	}
	MetricTimeWindowToProto = map[MetricTimeWindowSpecificValue]cxsdk.MetricTimeWindowValue{
		MetricTimeWindowValue1Minute:   cxsdk.MetricTimeWindowValue1MinuteOrUnspecified,
		MetricTimeWindowValue5Minutes:  cxsdk.MetricTimeWindowValue5Minutes,
		MetricTimeWindowValue10Minutes: cxsdk.MetricTimeWindowValue10Minutes,
		MetricTimeWindowValue15Minutes: cxsdk.MetricTimeWindowValue15Minutes,
		MetricTimeWindowValue20Minutes: cxsdk.MetricTimeWindowValue20Minutes,
		MetricTimeWindowValue30Minutes: cxsdk.MetricTimeWindowValue30Minutes,
		MetricTimeWindowValue1Hour:     cxsdk.MetricTimeWindowValue1Hour,
		MetricTimeWindowValue2Hours:    cxsdk.MetricTimeWindowValue2Hours,
		MetricTimeWindowValue4Hours:    cxsdk.MetricTimeWindowValue4Hours,
		MetricTimeWindowValue6Hours:    cxsdk.MetricTimeWindowValue6Hours,
		MetricTimeWindowValue12Hours:   cxsdk.MetricTimeWindowValue12Hours,
		MetricTimeWindowValue24Hours:   cxsdk.MetricTimeWindowValue24Hours,
		MetricTimeWindowValue36Hours:   cxsdk.MetricTimeWindowValue36Hours,
	}
	TracingTimeWindowSpecificValueToProto = map[TracingTimeWindowSpecificValue]cxsdk.TracingTimeWindowValue{
		TracingTimeWindowValue5Minutes:  cxsdk.TracingTimeWindowValue5MinutesOrUnspecified,
		TracingTimeWindowValue10Minutes: cxsdk.TracingTimeWindowValue10Minutes,
		TracingTimeWindowValue15Minutes: cxsdk.TracingTimeWindowValue15Minutes,
		TracingTimeWindowValue20Minutes: cxsdk.TracingTimeWindowValue20Minutes,
		TracingTimeWindowValue30Minutes: cxsdk.TracingTimeWindowValue30Minutes,
		TracingTimeWindowValue1Hour:     cxsdk.TracingTimeWindowValue1Hour,
		TracingTimeWindowValue2Hours:    cxsdk.TracingTimeWindowValue2Hours,
		TracingTimeWindowValue4Hours:    cxsdk.TracingTimeWindowValue4Hours,
		TracingTimeWindowValue6Hours:    cxsdk.TracingTimeWindowValue6Hours,
		TracingTimeWindowValue12Hours:   cxsdk.TracingTimeWindowValue12Hours,
		TracingTimeWindowValue24Hours:   cxsdk.TracingTimeWindowValue24Hours,
		TracingTimeWindowValue36Hours:   cxsdk.TracingTimeWindowValue36Hours,
	}
	TracingFilterOperationTypeToProto = map[TracingFilterOperationType]cxsdk.TracingFilterOperationType{
		TracingFilterOperationTypeOr:         cxsdk.TracingFilterOperationTypeIsOrUnspecified,
		TracingFilterOperationTypeIncludes:   cxsdk.TracingFilterOperationTypeIncludes,
		TracingFilterOperationTypeEndsWith:   cxsdk.TracingFilterOperationTypeEndsWith,
		TracingFilterOperationTypeStartsWith: cxsdk.TracingFilterOperationTypeStartsWith,
		TracingFilterOperationTypeIsNot:      cxsdk.TracingFilterOperationTypeIsNot,
	}
	TimeframeTypeToProto = map[FlowTimeframeType]cxsdk.TimeframeType{
		TimeframeTypeUnspecified: cxsdk.TimeframeTypeUnspecified,
		TimeframeTypeUpTo:        cxsdk.TimeframeTypeUpTo,
	}
	FlowStageGroupAlertsOpToProto = map[FlowStageGroupAlertsOp]cxsdk.AlertsOp{
		FlowStageGroupAlertsOpAnd: cxsdk.AlertsOpAndOrUnspecified,
		FlowStageGroupAlertsOpOr:  cxsdk.AlertsOpOr,
	}
	FlowStageGroupNextOpToProto = map[FlowStageGroupAlertsOp]cxsdk.NextOp{
		FlowStageGroupAlertsOpAnd: cxsdk.NextOpAndOrUnspecified,
		FlowStageGroupAlertsOpOr:  cxsdk.NextOpOr,
	}
	MetricAnomalyConditionTypeToProto = map[MetricAnomalyConditionType]cxsdk.MetricAnomalyConditionType{
		MetricAnomalyConditionTypeMoreThan: cxsdk.MetricAnomalyConditionTypeMoreThanOrUnspecified,
		MetricAnomalyConditionTypeLessThan: cxsdk.MetricAnomalyConditionTypeLessThan,
	}
	LogsNewValueTimeWindowValueToProto = map[LogsNewValueTimeWindowSpecificValue]cxsdk.LogsNewValueTimeWindowValue{
		LogsNewValueTimeWindowValue12Hours: cxsdk.LogsNewValueTimeWindowValue12HoursOrUnspecified,
		LogsNewValueTimeWindowValue24Hours: cxsdk.LogsNewValueTimeWindowValue24Hours,
		LogsNewValueTimeWindowValue48Hours: cxsdk.LogsNewValueTimeWindowValue48Hours,
		LogsNewValueTimeWindowValue72Hours: cxsdk.LogsNewValueTimeWindowValue72Hours,
		LogsNewValueTimeWindowValue1Week:   cxsdk.LogsNewValueTimeWindowValue1Week,
		LogsNewValueTimeWindowValue1Month:  cxsdk.LogsNewValueTimeWindowValue1Month,
		LogsNewValueTimeWindowValue2Months: cxsdk.LogsNewValueTimeWindowValue2Months,
		LogsNewValueTimeWindowValue3Months: cxsdk.LogsNewValueTimeWindowValue3Months,
	}
	LogsUniqueCountTimeWindowValueToProto = map[LogsUniqueCountTimeWindowSpecificValue]cxsdk.LogsUniqueValueTimeWindowValue{
		LogsUniqueCountTimeWindowValue1Minute:   cxsdk.LogsUniqueValueTimeWindowValue1MinuteOrUnspecified,
		LogsUniqueCountTimeWindowValue5Minutes:  cxsdk.LogsUniqueValueTimeWindowValue5Minutes,
		LogsUniqueCountTimeWindowValue10Minutes: cxsdk.LogsUniqueValueTimeWindowValue10Minutes,
		LogsUniqueCountTimeWindowValue15Minutes: cxsdk.LogsUniqueValueTimeWindowValue15Minutes,
		LogsUniqueCountTimeWindowValue20Minutes: cxsdk.LogsUniqueValueTimeWindowValue20Minutes,
		LogsUniqueCountTimeWindowValue30Minutes: cxsdk.LogsUniqueValueTimeWindowValue30Minutes,
		LogsUniqueCountTimeWindowValue1Hour:     cxsdk.LogsUniqueValueTimeWindowValue1Hour,
		LogsUniqueCountTimeWindowValue2Hours:    cxsdk.LogsUniqueValueTimeWindowValue2Hours,
		LogsUniqueCountTimeWindowValue4Hours:    cxsdk.LogsUniqueValueTimeWindowValue4Hours,
		LogsUniqueCountTimeWindowValue6Hours:    cxsdk.LogsUniqueValueTimeWindowValue6Hours,
		LogsUniqueCountTimeWindowValue12Hours:   cxsdk.LogsUniqueValueTimeWindowValue12Hours,
		LogsUniqueCountTimeWindowValue24Hours:   cxsdk.LogsUniqueValueTimeWindowValue24Hours,
		LogsUniqueCountTimeWindowValue36Hours:   cxsdk.LogsUniqueValueTimeWindowValue36Hours,
	}
)

// AlertSpec defines the desired state of Alert
type AlertSpec struct {
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`
	// +optional
	Description string        `json:"description,omitempty"`
	Priority    AlertPriority `json:"priority"`
	//+kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	GroupByKeys []string `json:"groupByKeys,omitempty"`
	// +optional
	IncidentsSettings *IncidentsSettings `json:"incidentsSettings,omitempty"`
	// +optional
	NotificationGroup *NotificationGroup `json:"notificationGroup,omitempty"`
	// +optional
	EntityLabels map[string]string `json:"entityLabels,omitempty"`
	//+kubebuilder:default=false
	PhantomMode bool `json:"phantomMode,omitempty"`
	// +optional
	Schedule       *AlertSchedule      `json:"schedule,omitempty"`
	TypeDefinition AlertTypeDefinition `json:"alertType"`
}

// AlertStatus defines the observed state of Alert
type AlertStatus struct {
	ID *string `json:"id,omitempty"`
}

type AlertSchedule struct {
	// +optional
	ActiveOn *ActiveOn `json:"activeOn,omitempty"`
}

type IncidentsSettings struct {
	//+kubebuilder:default=triggeredOnly
	NotifyOn           NotifyOn           `json:"notifyOn,omitempty"`
	RetriggeringPeriod RetriggeringPeriod `json:"retriggeringPeriod,omitempty"`
}

// +kubebuilder:validation:Enum=triggeredOnly;triggeredAndResolved
type NotifyOn string

const (
	NotifyOnTriggeredOnly        NotifyOn = "triggeredOnly"
	NotifyOnTriggeredAndResolved NotifyOn = "triggeredAndResolved"
)

// +kubebuilder:validation:Enum={"never","5m","10m","1h","2h","6h","12h","24h"}
type AutoRetireTimeframe string

const (
	AutoRetireTimeframeNeverOrUnspecified AutoRetireTimeframe = "never"
	AutoRetireTimeframe5M                 AutoRetireTimeframe = "5m"
	AutoRetireTimeframe10M                AutoRetireTimeframe = "10m"
	AutoRetireTimeframe1H                 AutoRetireTimeframe = "1h"
	AutoRetireTimeframe2H                 AutoRetireTimeframe = "2h"
	AutoRetireTimeframe6H                 AutoRetireTimeframe = "6h"
	AutoRetireTimeframe12H                AutoRetireTimeframe = "12h"
	AutoRetireTimeframe24H                AutoRetireTimeframe = "24h"
)

type RetriggeringPeriod struct {
	// +optional
	Minutes *uint32 `json:"minutes,omitempty"`
}

type NotificationGroup struct {
	// +optional
	GroupByKeys []string          `json:"groupByKeys,omitempty"`
	Webhooks    []WebhookSettings `json:"webhooks"`
}

type WebhookSettings struct {
	RetriggeringPeriod RetriggeringPeriod `json:"retriggeringPeriod"`
	// +kubebuilder:default=triggeredOnly
	NotifyOn    NotifyOn        `json:"notifyOn,omitempty"`
	Integration IntegrationType `json:"integration"`
}

type IntegrationType struct {
	// +optional
	IntegrationId *uint32 `json:"integrationId,omitempty"`
	// +optional
	Recipients []string `json:"recipients,omitempty"`
}

type ActiveOn struct {
	DayOfWeek []DayOfWeek `json:"dayOfWeek,omitempty"`
	StartTime *TimeOfDay  `json:"startTime,omitempty"`
	EndTime   *TimeOfDay  `json:"endTime,omitempty"`
}

type TimeOfDay struct {
	Hours   int32 `json:"hours,omitempty"`
	Minutes int32 `json:"minutes,omitempty"`
}

// +kubebuilder:validation:Enum=sunday;monday;tuesday;wednesday;thursday;friday;saturday
type DayOfWeek string

const (
	DayOfWeekSunday    DayOfWeek = "sunday"
	DayOfWeekMonday    DayOfWeek = "monday"
	DayOfWeekTuesday   DayOfWeek = "tuesday"
	DayOfWeekWednesday DayOfWeek = "wednesday"
	DayOfWeekThursday  DayOfWeek = "thursday"
	DayOfWeekFriday    DayOfWeek = "friday"
	DayOfWeekSaturday  DayOfWeek = "saturday"
)

type AlertTypeDefinition struct {
	// +optional
	LogsImmediate *LogsImmediate `json:"logsImmediate,omitempty"`
	// +optional
	LogsThreshold *LogsThreshold `json:"logsThreshold,omitempty"`
	// +optional
	LogsRatioThreshold *LogsRatioThreshold `json:"logsRatioThreshold,omitempty"`
	// +optional
	LogsTimeRelativeThreshold *LogsTimeRelativeThreshold `json:"logsTimeRelativeThreshold,omitempty"`
	// +optional
	MetricThreshold *MetricThreshold `json:"metricThreshold,omitempty"`
	// +optional
	TracingThreshold *TracingThreshold `json:"tracingThreshold,omitempty"`
	// +optional
	Flow *Flow `json:"flow,omitempty"`
	// +optional
	LogsAnomaly *LogsAnomaly `json:"logsAnomaly,omitempty"`
	// +optional
	MetricAnomaly *MetricAnomaly `json:"metricAnomaly,omitempty"`
	// +optional
	LogsNewValue *LogsNewValue `json:"logsNewValue,omitempty"`
	// +optional
	LogsUniqueCount *LogsUniqueCount `json:"logsUniqueCount,omitempty"`
}

type LogsImmediate struct {
	// +optional
	LogsFilter *LogsFilter `json:"logsFilter,omitempty"`
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter,omitempty"`
}

type LogsThreshold struct {
	// +optional
	LogsFilter *LogsFilter `json:"logsFilter,omitempty"`
	// +optional
	UndetectedValuesManagement *UndetectedValuesManagement `json:"undetectedValuesManagement,omitempty"`
	Rules                      []LogsThresholdRule         `json:"rules,omitempty"`
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter,omitempty"`
}

type LogsThresholdRule struct {
	Condition LogsThresholdRuleCondition `json:"condition"`
	// +optional
	Override AlertOverride `json:"override"`
}

type LogsThresholdRuleCondition struct {
	TimeWindow                 LogsTimeWindow             `json:"timeWindow"`
	Threshold                  resource.Quantity          `json:"threshold"`
	LogsThresholdConditionType LogsThresholdConditionType `json:"logsThresholdConditionType"`
}

type LogsTimeWindow struct {
	SpecificValue LogsTimeWindowValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"5m","10m","15m","30m","1h","2h","6h","12h","24h","36h"}
type LogsTimeWindowValue string

const (
	LogsTimeWindowLast5Minutes  LogsTimeWindowValue = "5m"
	LogsTimeWindowLast10Minutes LogsTimeWindowValue = "10m"
	LogsTimeWindowLast15Minutes LogsTimeWindowValue = "15m"
	LogsTimeWindowLast30Minutes LogsTimeWindowValue = "30m"
	LogsTimeWindowLastHour      LogsTimeWindowValue = "1h"
	LogsTimeWindowLast2Hours    LogsTimeWindowValue = "2h"
	LogsTimeWindowLast6Hours    LogsTimeWindowValue = "6h"
	LogsTimeWindowLast12Hours   LogsTimeWindowValue = "12h"
	LogsTimeWindowLast24Hours   LogsTimeWindowValue = "24h"
	LogsTimeWindowLast36Hours   LogsTimeWindowValue = "36h"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
type LogsThresholdConditionType string

const (
	LogsThresholdConditionTypeMoreThan LogsThresholdConditionType = "moreThan"
	LogsThresholdConditionTypeLessThan LogsThresholdConditionType = "lessThan"
)

type AlertOverride struct {
	Priority AlertPriority `json:"priority"`
}

type LogsRatioThreshold struct {
	Numerator        LogsFilter               `json:"numerator"`
	NumeratorAlias   string                   `json:"numeratorAlias"`
	Denominator      LogsFilter               `json:"denominator"`
	DenominatorAlias string                   `json:"denominatorAlias"`
	Rules            []LogsRatioThresholdRule `json:"rules"`
}

type LogsRatioThresholdRule struct {
	Condition LogsRatioCondition `json:"condition"`
	Override  AlertOverride      `json:"override"`
}

type LogsRatioCondition struct {
	Threshold     resource.Quantity      `json:"threshold"`
	TimeWindow    LogsRatioTimeWindow    `json:"timeWindow"`
	ConditionType LogsRatioConditionType `json:"conditionType"`
}

// +kubebuilder:validation:Enum={"5m","10m","15m","30m","1h","2h","4h","6h","12h","24h","36h"}
type LogsRatioTimeWindow string

const (
	LogsRatioTimeWindowMinutes5  LogsRatioTimeWindow = "5m"
	LogsRatioTimeWindowMinutes10 LogsRatioTimeWindow = "10m"
	LogsRatioTimeWindowMinutes15 LogsRatioTimeWindow = "15m"
	LogsRatioTimeWindowMinutes30 LogsRatioTimeWindow = "30m"
	LogsRatioTimeWindowHour1     LogsRatioTimeWindow = "1h"
	LogsRatioTimeWindowHours2    LogsRatioTimeWindow = "2h"
	LogsRatioTimeWindowHours4    LogsRatioTimeWindow = "4h"
	LogsRatioTimeWindowHours6    LogsRatioTimeWindow = "6h"
	LogsRatioTimeWindowHours12   LogsRatioTimeWindow = "12h"
	LogsRatioTimeWindowHours24   LogsRatioTimeWindow = "24h"
	LogsRatioTimeWindowHours36   LogsRatioTimeWindow = "36h"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
type LogsRatioConditionType string

const (
	LogsRatioConditionTypeMoreThan LogsRatioConditionType = "moreThan"
	LogsRatioConditionTypeLessThan LogsRatioConditionType = "lessThan"
)

type LogsTimeRelativeRule struct {
	Condition LogsTimeRelativeCondition `json:"condition"`
	// +optional
	Override AlertOverride `json:"override"`
}

type LogsTimeRelativeCondition struct {
	Threshold     resource.Quantity             `json:"threshold"`
	ComparedTo    LogsTimeRelativeComparedTo    `json:"comparedTo"`
	ConditionType LogsTimeRelativeConditionType `json:"conditionType"`
}

// +kubebuilder:validation:Enum=previousHour;sameHourYesterday;sameHourLastWeek;yesterday;sameDayLastWeek;sameDayLastMonth
type LogsTimeRelativeComparedTo string

const (
	LogsTimeRelativeComparedToPreviousHour      LogsTimeRelativeComparedTo = "previousHour"
	LogsTimeRelativeComparedToSameHourYesterday LogsTimeRelativeComparedTo = "sameHourYesterday"
	LogsTimeRelativeComparedToSameHourLastWeek  LogsTimeRelativeComparedTo = "sameHourLastWeek"
	LogsTimeRelativeComparedToYesterday         LogsTimeRelativeComparedTo = "yesterday"
	LogsTimeRelativeComparedToSameDayLastWeek   LogsTimeRelativeComparedTo = "sameDayLastWeek"
	LogsTimeRelativeComparedToSameDayLastMonth  LogsTimeRelativeComparedTo = "sameDayLastMonth"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
type LogsTimeRelativeConditionType string

const (
	LogsTimeRelativeConditionTypeMoreThan LogsTimeRelativeConditionType = "moreThan"
	LogsTimeRelativeConditionTypeLessThan LogsTimeRelativeConditionType = "lessThan"
)

type LogsTimeRelativeThreshold struct {
	LogsFilter LogsFilter             `json:"logsFilter"`
	Rules      []LogsTimeRelativeRule `json:"rules"`
	//+kubebuilder:default=false
	IgnoreInfinity            bool     `json:"ignoreInfinity"`
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
	// +optional
	UndetectedValuesManagement UndetectedValuesManagement `json:"undetectedValuesManagement"`
}

type MetricThreshold struct {
	MetricFilter  MetricFilter          `json:"metricFilter"`
	Rules         []MetricThresholdRule `json:"rules"`
	MissingValues MetricMissingValues   `json:"missingValues"`
	// +optional
	UndetectedValuesManagement *UndetectedValuesManagement `json:"undetectedValuesManagement"`
}

type MetricFilter struct {
	Promql string `json:"promql,omitempty"`
}

type MetricThresholdRule struct {
	Condition MetricThresholdRuleCondition `json:"condition"`
	// +optional
	Override AlertOverride `json:"override"`
}

type MetricThresholdRuleCondition struct {
	Threshold     resource.Quantity            `json:"threshold"`
	ForOverPct    uint32                       `json:"forOverPct"`
	OfTheLast     MetricTimeWindow             `json:"ofTheLast"`
	ConditionType MetricThresholdConditionType `json:"conditionType"`
}

type MetricTimeWindow struct {
	SpecificValue MetricTimeWindowSpecificValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"1m","5m","10m","15m","20m","30m","1h","2h","4h","6h","12h","24h","36h"}
type MetricTimeWindowSpecificValue string

const (
	MetricTimeWindowValue1Minute   MetricTimeWindowSpecificValue = "1m"
	MetricTimeWindowValue5Minutes  MetricTimeWindowSpecificValue = "5m"
	MetricTimeWindowValue10Minutes MetricTimeWindowSpecificValue = "10m"
	MetricTimeWindowValue15Minutes MetricTimeWindowSpecificValue = "15m"
	MetricTimeWindowValue20Minutes MetricTimeWindowSpecificValue = "20m"
	MetricTimeWindowValue30Minutes MetricTimeWindowSpecificValue = "30m"
	MetricTimeWindowValue1Hour     MetricTimeWindowSpecificValue = "1h"
	MetricTimeWindowValue2Hours    MetricTimeWindowSpecificValue = "2h"
	MetricTimeWindowValue4Hours    MetricTimeWindowSpecificValue = "4h"
	MetricTimeWindowValue6Hours    MetricTimeWindowSpecificValue = "6h"
	MetricTimeWindowValue12Hours   MetricTimeWindowSpecificValue = "12h"
	MetricTimeWindowValue24Hours   MetricTimeWindowSpecificValue = "24h"
	MetricTimeWindowValue36Hours   MetricTimeWindowSpecificValue = "36h"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
type MetricThresholdConditionType string

const (
	MetricThresholdConditionTypeMoreThan         MetricThresholdConditionType = "moreThan"
	MetricThresholdConditionTypeLessThan         MetricThresholdConditionType = "lessThan"
	MetricThresholdConditionTypeMoreThanOrEquals MetricThresholdConditionType = "moreThanOrEquals"
	MetricThresholdConditionTypeLessThanOrEquals MetricThresholdConditionType = "lessThanOrEquals"
)

type MetricMissingValues struct {
	// +optional
	ReplaceWithZero *bool `json:"replaceWithZero,omitempty"`
	// +optional
	MinNonNullValuesPct *uint32 `json:"minNonNullValuesPct,omitempty"`
}

type TracingThreshold struct {
	// +optional
	TracingFilter *TracingFilter         `json:"tracingFilter,omitempty"`
	Rules         []TracingThresholdRule `json:"rules,omitempty"`
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter,omitempty"`
}

type TracingFilter struct {
	Simple *TracingSimpleFilter `json:"simple,omitempty"`
}

type TracingFilterType struct {
	Values    []string                   `json:"values"`
	Operation TracingFilterOperationType `json:"operation"`
}

// +kubebuilder:validation:Enum=or;includes;endsWith;startsWith;isNot
type TracingFilterOperationType string

const (
	TracingFilterOperationTypeOr         TracingFilterOperationType = "or"
	TracingFilterOperationTypeIncludes   TracingFilterOperationType = "includes"
	TracingFilterOperationTypeEndsWith   TracingFilterOperationType = "endsWith"
	TracingFilterOperationTypeStartsWith TracingFilterOperationType = "startsWith"
	TracingFilterOperationTypeIsNot      TracingFilterOperationType = "isNot"
)

type TracingSimpleFilter struct {
	TracingLabelFilters *TracingLabelFilters `json:"tracingLabelFilters,omitempty"`
	LatencyThresholdMs  *uint64              `json:"latencyThresholdMs,omitempty"`
}

type TracingLabelFilters struct {
	// +optional
	ApplicationName []TracingFilterType `json:"applicationName,omitempty"`
	// +optional
	SubsystemName []TracingFilterType `json:"subsystemName,omitempty"`
	// +optional
	ServiceName []TracingFilterType `json:"serviceName,omitempty"`
	// +optional
	OperationName []TracingFilterType `json:"operationName,omitempty"`
	// +optional
	SpanFields []TracingSpanFieldsFilterType `json:"spanFields,omitempty"`
}

type TracingSpanFieldsFilterType struct {
	Key        string            `json:"key"`
	FilterType TracingFilterType `json:"filterType"`
}

type TracingThresholdRule struct {
	Condition TracingThresholdRuleCondition `json:"condition"`
}

type TracingThresholdRuleCondition struct {
	SpanAmount resource.Quantity `json:"spanAmount"`
	TimeWindow TracingTimeWindow `json:"timeWindow"`
}

type TracingTimeWindow struct {
	SpecificValue *TracingTimeWindowSpecificValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"5m","10m","15m","20m","30m","1h","2h","4h","6h","12h","24h","36h"}
type TracingTimeWindowSpecificValue string

const (
	TracingTimeWindowValue5Minutes  TracingTimeWindowSpecificValue = "5m"
	TracingTimeWindowValue10Minutes TracingTimeWindowSpecificValue = "10m"
	TracingTimeWindowValue15Minutes TracingTimeWindowSpecificValue = "15m"
	TracingTimeWindowValue20Minutes TracingTimeWindowSpecificValue = "20m"
	TracingTimeWindowValue30Minutes TracingTimeWindowSpecificValue = "30m"
	TracingTimeWindowValue1Hour     TracingTimeWindowSpecificValue = "1h"
	TracingTimeWindowValue2Hours    TracingTimeWindowSpecificValue = "2h"
	TracingTimeWindowValue4Hours    TracingTimeWindowSpecificValue = "4h"
	TracingTimeWindowValue6Hours    TracingTimeWindowSpecificValue = "6h"
	TracingTimeWindowValue12Hours   TracingTimeWindowSpecificValue = "12h"
	TracingTimeWindowValue24Hours   TracingTimeWindowSpecificValue = "24h"
	TracingTimeWindowValue36Hours   TracingTimeWindowSpecificValue = "36h"
)

type Flow struct {
	Stages []FlowStage `json:"stages"`
	// +kubebuilder:default=false
	EnforceSuppression bool `json:"enforceSuppression"`
}

type FlowStage struct {
	FlowStagesType FlowStagesType    `json:"flowStagesType"`
	TimeframeMs    int64             `json:"timeframeMs"`
	TimeframeType  FlowTimeframeType `json:"timeframeType"`
}

type FlowStagesType struct {
	Groups []FlowStageGroup `json:"groups"`
}

type FlowStageGroup struct {
	AlertDefs []FlowStagesGroupsAlertDefs `json:"alertDefs"`
	NextOp    FlowStageGroupAlertsOp      `json:"nextOp"`
	AlertsOp  FlowStageGroupAlertsOp      `json:"alertsOp"`
}

type FlowStagesGroupsAlertDefs struct {
	Id string `json:"id"`
	// +kubebuilder:default=false
	Not bool `json:"not"`
}

// +kubebuilder:validation:Enum=and;or
type FlowStageGroupAlertsOp string

const (
	FlowStageGroupAlertsOpAnd FlowStageGroupAlertsOp = "and"
	FlowStageGroupAlertsOpOr  FlowStageGroupAlertsOp = "or"
)

// +kubebuilder:validation:Enum=unspecified;upTo
type FlowTimeframeType string

const (
	TimeframeTypeUnspecified FlowTimeframeType = "unspecified"
	TimeframeTypeUpTo        FlowTimeframeType = "upTo"
)

type LogsAnomaly struct {
	// +optional
	LogsFilter *LogsFilter       `json:"logsFilter,omitempty"`
	Rules      []LogsAnomalyRule `json:"rules,omitempty"`
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter,omitempty"`
}

type LogsAnomalyRule struct {
	Condition LogsAnomalyCondition `json:"condition"`
}

type LogsAnomalyCondition struct {
	MinimumThreshold resource.Quantity `json:"minimumThreshold"`
	TimeWindow       LogsTimeWindow    `json:"timeWindow"`
}

type MetricAnomaly struct {
	MetricFilter *MetricFilter        `json:"metricFilter"`
	Rules        []*MetricAnomalyRule `json:"rules"`
}

type MetricAnomalyCondition struct {
	Threshold           resource.Quantity          `json:"threshold"`
	ForOverPct          uint32                     `json:"forOverPct"`
	OfTheLast           MetricTimeWindow           `json:"ofTheLast"`
	MinNonNullValuesPct uint32                     `json:"minNonNullValuesPct"`
	ConditionType       MetricAnomalyConditionType `json:"conditionType"`
}

// +kubebuilder:validation:Enum=moreThan;lessThan
type MetricAnomalyConditionType string

const (
	MetricAnomalyConditionTypeMoreThan MetricAnomalyConditionType = "moreThan"
	MetricAnomalyConditionTypeLessThan MetricAnomalyConditionType = "lessThan"
)

type MetricAnomalyRule struct {
	Condition MetricAnomalyCondition `json:"condition"`
}

type LogsNewValue struct {
	LogsFilter                *LogsFilter        `json:"logsFilter"`
	Rules                     []LogsNewValueRule `json:"rules"`
	NotificationPayloadFilter []string           `json:"notificationPayloadFilter"`
}

type LogsNewValueRule struct {
	Condition LogsNewValueRuleCondition `json:"condition"`
}

type LogsNewValueRuleCondition struct {
	KeypathToTrack string                 `json:"keypathToTrack"`
	TimeWindow     LogsNewValueTimeWindow `json:"timeWindow"`
}

type LogsNewValueTimeWindow struct {
	SpecificValue *LogsNewValueTimeWindowSpecificValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"12h","24h","48h","72h","1w","1mo","2mo","3mo"}
type LogsNewValueTimeWindowSpecificValue string

const (
	LogsNewValueTimeWindowValue12Hours LogsNewValueTimeWindowSpecificValue = "12h"
	LogsNewValueTimeWindowValue24Hours LogsNewValueTimeWindowSpecificValue = "24h"
	LogsNewValueTimeWindowValue48Hours LogsNewValueTimeWindowSpecificValue = "48h"
	LogsNewValueTimeWindowValue72Hours LogsNewValueTimeWindowSpecificValue = "72h"
	LogsNewValueTimeWindowValue1Week   LogsNewValueTimeWindowSpecificValue = "1w"
	LogsNewValueTimeWindowValue1Month  LogsNewValueTimeWindowSpecificValue = "1mo"
	LogsNewValueTimeWindowValue2Months LogsNewValueTimeWindowSpecificValue = "2mo"
	LogsNewValueTimeWindowValue3Months LogsNewValueTimeWindowSpecificValue = "3mo"
)

type LogsUniqueCount struct {
	LogsFilter                  *LogsFilter           `json:"logsFilter"`
	Rules                       []LogsUniqueCountRule `json:"rules"`
	NotificationPayloadFilter   []string              `json:"notificationPayloadFilter"`
	MaxUniqueCountPerGroupByKey *uint64               `json:"maxUniqueCountPerGroupByKey"`
	UniqueCountKeypath          *string               `json:"uniqueCountKeypath"`
}

type LogsUniqueCountCondition struct {
	Threshold  int64                     `json:"threshold"`
	TimeWindow LogsUniqueCountTimeWindow `json:"timeWindow"`
}

type LogsUniqueCountTimeWindow struct {
	SpecificValue LogsUniqueCountTimeWindowSpecificValue `json:"specificValue"`
}

// +kubebuilder:validation:Enum={"1m","5m","10m","15m","20m","30m","1h","2h","4h","6h","12h","24h","36h"}
type LogsUniqueCountTimeWindowSpecificValue string

const (
	LogsUniqueCountTimeWindowValue1Minute   LogsUniqueCountTimeWindowSpecificValue = "1m"
	LogsUniqueCountTimeWindowValue5Minutes  LogsUniqueCountTimeWindowSpecificValue = "5m"
	LogsUniqueCountTimeWindowValue10Minutes LogsUniqueCountTimeWindowSpecificValue = "10m"
	LogsUniqueCountTimeWindowValue15Minutes LogsUniqueCountTimeWindowSpecificValue = "15m"
	LogsUniqueCountTimeWindowValue20Minutes LogsUniqueCountTimeWindowSpecificValue = "20m"
	LogsUniqueCountTimeWindowValue30Minutes LogsUniqueCountTimeWindowSpecificValue = "30m"
	LogsUniqueCountTimeWindowValue1Hour     LogsUniqueCountTimeWindowSpecificValue = "1h"
	LogsUniqueCountTimeWindowValue2Hours    LogsUniqueCountTimeWindowSpecificValue = "2h"
	LogsUniqueCountTimeWindowValue4Hours    LogsUniqueCountTimeWindowSpecificValue = "4h"
	LogsUniqueCountTimeWindowValue6Hours    LogsUniqueCountTimeWindowSpecificValue = "6h"
	LogsUniqueCountTimeWindowValue12Hours   LogsUniqueCountTimeWindowSpecificValue = "12h"
	LogsUniqueCountTimeWindowValue24Hours   LogsUniqueCountTimeWindowSpecificValue = "24h"
	LogsUniqueCountTimeWindowValue36Hours   LogsUniqueCountTimeWindowSpecificValue = "36h"
)

type LogsUniqueCountRule struct {
	Condition LogsUniqueCountCondition `json:"condition"`
}

type LogsFilter struct {
	FilterType LogsFilterType `json:"filterType,omitempty"`
}

type LogsFilterType struct {
	// +optional
	SimpleFilter *LogsSimpleFilter `json:"simpleFilter,omitempty"`
}

type LogsSimpleFilter struct {
	// +optional
	LuceneQuery  *string       `json:"luceneQuery,omitempty"`
	LabelFilters *LabelFilters `json:"labelFilters,omitempty"`
}

type LabelFilters struct {
	// +optional
	ApplicationName []LabelFilterType `json:"applicationName,omitempty"`
	// +optional
	SubsystemName []LabelFilterType `json:"subsystemName,omitempty"`
	// +optional
	Severity []LogSeverity `json:"severity,omitempty"`
}

type LabelFilterType struct {
	//+kubebuilder:validation:MinLength=0
	Value string `json:"value"`

	Operation LogFilterOperationType `json:"operation"`
}

type UndetectedValuesManagement struct {
	//+kubebuilder:default=false
	TriggerUndetectedValues bool `json:"triggerUndetectedValues"`
	//+kubebuilder:default=never
	AutoRetireTimeframe AutoRetireTimeframe `json:"autoRetireTimeframe"`
}

// +kubebuilder:validation:Enum=or;includes;endsWith;startsWith
type LogFilterOperationType string

const (
	LogFilterOperationTypeOr         LogFilterOperationType = "or"
	LogFilterOperationTypeIncludes   LogFilterOperationType = "includes"
	LogFilterOperationTypeEndWith    LogFilterOperationType = "endsWith"
	LogFilterOperationTypeStartsWith LogFilterOperationType = "startsWith"
)

// +kubebuilder:validation:Enum=debug;info;warning;error;critical
type LogSeverity string

const (
	LogSeverityDebug    LogSeverity = "debug"
	LogSeverityInfo     LogSeverity = "info"
	LogSeverityWarning  LogSeverity = "warning"
	LogSeverityError    LogSeverity = "error"
	LogSeverityCritical LogSeverity = "critical"
)

// +kubebuilder:validation:Enum=p1;p2;p3;p4
type AlertPriority string

const (
	AlertPriorityP1 AlertPriority = "p1"
	AlertPriorityP2 AlertPriority = "p2"
	AlertPriorityP3 AlertPriority = "p3"
	AlertPriorityP4 AlertPriority = "p4"
)

func NewDefaultAlertStatus() *AlertStatus {
	return &AlertStatus{
		ID: ptr.To(""),
	}
}

func (in AlertSpec) ExtractAlertProperties() *cxsdk.AlertDefProperties {
	alertDefProperties := &cxsdk.AlertDefProperties{
		Name:              wrapperspb.String(in.Name),
		Description:       wrapperspb.String(in.Description),
		Enabled:           wrapperspb.Bool(in.Enabled),
		Priority:          AlertPriorityToProtoPriority[in.Priority],
		GroupByKeys:       utils.StringSliceToWrappedStringSlice(in.GroupByKeys),
		IncidentsSettings: expandIncidentsSettings(in.IncidentsSettings),
		NotificationGroup: expandNotificationGroup(in.NotificationGroup),
		EntityLabels:      in.EntityLabels,
		PhantomMode:       wrapperspb.Bool(in.PhantomMode),
	}
	alertDefProperties = expandAlertSchedule(alertDefProperties, in.Schedule)
	alertDefProperties = expandAlertTypeDefinition(alertDefProperties, in.TypeDefinition)

	return alertDefProperties
}

func expandIncidentsSettings(incidentsSettings *IncidentsSettings) *cxsdk.AlertDefIncidentSettings {
	if incidentsSettings == nil {
		return nil
	}

	alertDefIncidentSettings := &cxsdk.AlertDefIncidentSettings{
		NotifyOn: NotifyOnToProtoNotifyOn[incidentsSettings.NotifyOn],
	}
	alertDefIncidentSettings = expandRetriggeringPeriod(alertDefIncidentSettings, incidentsSettings.RetriggeringPeriod)
	return alertDefIncidentSettings
}

func expandRetriggeringPeriod(alertDefIncidentSettings *cxsdk.AlertDefIncidentSettings, retriggeringPeriod RetriggeringPeriod) *cxsdk.AlertDefIncidentSettings {
	if retriggeringPeriod.Minutes != nil {
		alertDefIncidentSettings.RetriggeringPeriod = &cxsdk.AlertDefIncidentSettingsMinutes{
			Minutes: wrapperspb.UInt32(*retriggeringPeriod.Minutes),
		}
	}

	return alertDefIncidentSettings
}

func expandNotificationGroup(notificationGroup *NotificationGroup) *cxsdk.AlertDefNotificationGroup {
	if notificationGroup == nil {
		return nil
	}

	return &cxsdk.AlertDefNotificationGroup{
		GroupByKeys: utils.StringSliceToWrappedStringSlice(notificationGroup.GroupByKeys),
		Webhooks:    expandWebhooksSettings(notificationGroup.Webhooks),
	}
}

func expandWebhooksSettings(webhooksSettings []WebhookSettings) []*cxsdk.AlertDefWebhooksSettings {
	result := make([]*cxsdk.AlertDefWebhooksSettings, len(webhooksSettings))
	for _, setting := range webhooksSettings {
		result = append(result, expandWebhookSetting(setting))
	}
	return result[1:]
}

func expandWebhookSetting(webhooksSetting WebhookSettings) *cxsdk.AlertDefWebhooksSettings {
	notifyOn := NotifyOnToProtoNotifyOn[webhooksSetting.NotifyOn]
	return &cxsdk.AlertDefWebhooksSettings{
		NotifyOn:    &notifyOn,
		Integration: expandIntegration(webhooksSetting.Integration),
		RetriggeringPeriod: &cxsdk.AlertDefWebhooksSettingsMinutes{
			Minutes: wrapperspb.UInt32(*webhooksSetting.RetriggeringPeriod.Minutes),
		},
	}
}

func expandIntegration(integration IntegrationType) *cxsdk.AlertDefIntegrationType {
	if integrationID := integration.IntegrationId; integrationID != nil {
		return &cxsdk.AlertDefIntegrationType{
			IntegrationType: &cxsdk.AlertDefIntegrationTypeIntegrationID{
				IntegrationId: wrapperspb.UInt32(*integrationID),
			},
		}
	} else if recipients := integration.Recipients; recipients != nil {
		return &cxsdk.AlertDefIntegrationType{
			IntegrationType: &cxsdk.AlertDefIntegrationTypeRecipients{
				Recipients: &cxsdk.AlertDefRecipients{
					Emails: utils.StringSliceToWrappedStringSlice(recipients),
				},
			},
		}
	}

	return nil
}

func expandAlertSchedule(alertProperties *cxsdk.AlertDefProperties, alertSchedule *AlertSchedule) *cxsdk.AlertDefProperties {
	if alertSchedule == nil {
		return alertProperties
	}

	if activeOn := alertSchedule.ActiveOn; activeOn != nil {
		alertProperties.Schedule = &cxsdk.AlertDefPropertiesActiveOn{
			ActiveOn: expandActivitySchedule(activeOn),
		}
	}

	return alertProperties
}

func expandActivitySchedule(activeOn *ActiveOn) *cxsdk.AlertsActivitySchedule {
	return &cxsdk.AlertsActivitySchedule{
		DayOfWeek: expandDaysOfWeek(activeOn.DayOfWeek),
		StartTime: expandTimeOfDay(activeOn.StartTime),
		EndTime:   &cxsdk.AlertTimeOfDay{},
	}
}

func expandDaysOfWeek(week []DayOfWeek) []cxsdk.AlertDayOfWeek {
	result := make([]cxsdk.AlertDayOfWeek, len(week))
	for i, d := range week {
		result[i] = DaysOfWeekToProtoDayOfWeek[d]
	}

	return result
}

func expandTimeOfDay(time *TimeOfDay) *cxsdk.AlertTimeOfDay {
	if time == nil {
		return nil
	}

	return &cxsdk.AlertTimeOfDay{
		Hours:   time.Hours,
		Minutes: time.Minutes,
	}
}

func expandAlertTypeDefinition(properties *cxsdk.AlertDefProperties, definition AlertTypeDefinition) *cxsdk.AlertDefProperties {
	if logsImmediate := definition.LogsImmediate; logsImmediate != nil {
		properties.TypeDefinition = expandLogsImmediate(logsImmediate)
		properties.Type = cxsdk.AlertDefTypeLogsImmediateOrUnspecified
	} else if logsThreshold := definition.LogsThreshold; logsThreshold != nil {
		properties.TypeDefinition = expandLogsThreshold(logsThreshold)
		properties.Type = cxsdk.AlertDefTypeLogsThreshold
	} else if logsRatioThreshold := definition.LogsRatioThreshold; logsRatioThreshold != nil {
		properties.TypeDefinition = expandLogsRatioThreshold(logsRatioThreshold)
		properties.Type = cxsdk.AlertDefTypeLogsRatioThreshold
	} else if logsTimeRelativeThreshold := definition.LogsTimeRelativeThreshold; logsTimeRelativeThreshold != nil {
		properties.TypeDefinition = expandLogsTimeRelativeThreshold(logsTimeRelativeThreshold)
		properties.Type = cxsdk.AlertDefTypeLogsTimeRelativeThreshold
	} else if metricThreshold := definition.MetricThreshold; metricThreshold != nil {
		properties.TypeDefinition = expandMetricThreshold(metricThreshold)
		properties.Type = cxsdk.AlertDefTypeMetricThreshold
	} else if tracingThreshold := definition.TracingThreshold; tracingThreshold != nil {
		properties.TypeDefinition = expandTracingThreshold(tracingThreshold)
		properties.Type = cxsdk.AlertDefTypeTracingThreshold
	} else if flow := definition.Flow; flow != nil {
		properties.TypeDefinition = expandFlow(flow)
		properties.Type = cxsdk.AlertDefTypeFlow
	} else if logsAnomaly := definition.LogsAnomaly; logsAnomaly != nil {
		properties.TypeDefinition = expandLogsAnomaly(logsAnomaly)
		properties.Type = cxsdk.AlertDefTypeLogsAnomaly
	} else if metricAnomaly := definition.MetricAnomaly; metricAnomaly != nil {
		properties.TypeDefinition = expandMetricAnomaly(metricAnomaly)
		properties.Type = cxsdk.AlertDefTypeMetricAnomaly
	} else if logsNewValue := definition.LogsNewValue; logsNewValue != nil {
		properties.TypeDefinition = expandLogsNewValue(logsNewValue)
		properties.Type = cxsdk.AlertDefTypeLogsNewValue
	} else if logsUniqueCount := definition.LogsUniqueCount; logsUniqueCount != nil {
		properties.TypeDefinition = expandLogsUniqueCount(logsUniqueCount)
		properties.Type = cxsdk.AlertDefTypeLogsUniqueCount
	}

	return properties
}

func expandLogsUniqueCount(uniqueCount *LogsUniqueCount) *cxsdk.AlertDefPropertiesLogsUniqueCount {
	return &cxsdk.AlertDefPropertiesLogsUniqueCount{
		LogsUniqueCount: &cxsdk.LogsUniqueCountType{
			LogsFilter:                  expandLogsFilter(uniqueCount.LogsFilter),
			Rules:                       expandLogsUniqueCountRules(uniqueCount.Rules),
			NotificationPayloadFilter:   utils.StringSliceToWrappedStringSlice(uniqueCount.NotificationPayloadFilter),
			MaxUniqueCountPerGroupByKey: wrapperspb.Int64(int64(*uniqueCount.MaxUniqueCountPerGroupByKey)),
			UniqueCountKeypath:          wrapperspb.String(*uniqueCount.UniqueCountKeypath),
		},
	}
}

func expandLogsUniqueCountRules(rules []LogsUniqueCountRule) []*cxsdk.LogsUniqueCountRule {
	result := make([]*cxsdk.LogsUniqueCountRule, len(rules))
	for i := range rules {
		result[i] = expandLogsUniqueCountRule(rules[i])
	}

	return result
}

func expandLogsUniqueCountRule(rule LogsUniqueCountRule) *cxsdk.LogsUniqueCountRule {
	return &cxsdk.LogsUniqueCountRule{
		Condition: expandLogsUniqueCountCondition(rule.Condition),
	}
}

func expandLogsUniqueCountCondition(condition LogsUniqueCountCondition) *cxsdk.LogsUniqueCountCondition {
	return &cxsdk.LogsUniqueCountCondition{
		MaxUniqueCount: wrapperspb.Int64(condition.Threshold),
		TimeWindow:     expandLogsUniqueCountTimeWindow(condition.TimeWindow),
	}
}

func expandLogsUniqueCountTimeWindow(timeWindow LogsUniqueCountTimeWindow) *cxsdk.LogsUniqueValueTimeWindow {
	return &cxsdk.LogsUniqueValueTimeWindow{
		Type: &cxsdk.LogsUniqueValueTimeWindowSpecificValue{
			LogsUniqueValueTimeWindowSpecificValue: LogsUniqueCountTimeWindowValueToProto[timeWindow.SpecificValue],
		},
	}
}

func expandLogsNewValue(logsNewValue *LogsNewValue) *cxsdk.AlertDefPropertiesLogsNewValue {
	return &cxsdk.AlertDefPropertiesLogsNewValue{
		LogsNewValue: &cxsdk.LogsNewValueType{
			LogsFilter:                expandLogsFilter(logsNewValue.LogsFilter),
			Rules:                     expandLogsNewValueRules(logsNewValue.Rules),
			NotificationPayloadFilter: utils.StringSliceToWrappedStringSlice(logsNewValue.NotificationPayloadFilter),
		},
	}
}

func expandLogsNewValueRules(rules []LogsNewValueRule) []*cxsdk.LogsNewValueRule {
	result := make([]*cxsdk.LogsNewValueRule, len(rules))
	for i := range rules {
		result[i] = expandLogsNewValueRule(rules[i])
	}

	return result
}

func expandLogsNewValueRule(rule LogsNewValueRule) *cxsdk.LogsNewValueRule {
	return &cxsdk.LogsNewValueRule{
		Condition: expandLogsNewValueRuleCondition(rule.Condition),
	}
}

func expandLogsNewValueRuleCondition(condition LogsNewValueRuleCondition) *cxsdk.LogsNewValueCondition {
	return &cxsdk.LogsNewValueCondition{
		KeypathToTrack: wrapperspb.String(condition.KeypathToTrack),
		TimeWindow:     expandLogsNewValueTimeWindow(condition.TimeWindow),
	}
}

func expandLogsNewValueTimeWindow(timeWindow LogsNewValueTimeWindow) *cxsdk.LogsNewValueTimeWindow {
	return &cxsdk.LogsNewValueTimeWindow{
		Type: &cxsdk.LogsNewValueTimeWindowSpecificValue{
			LogsNewValueTimeWindowSpecificValue: LogsNewValueTimeWindowValueToProto[*timeWindow.SpecificValue],
		},
	}
}

func expandMetricAnomaly(metricAnomaly *MetricAnomaly) *cxsdk.AlertDefPropertiesMetricAnomaly {
	return &cxsdk.AlertDefPropertiesMetricAnomaly{
		MetricAnomaly: &cxsdk.MetricAnomalyType{
			MetricFilter: &cxsdk.MetricFilter{
				Type: &cxsdk.MetricFilterPromql{
					Promql: wrapperspb.String(metricAnomaly.MetricFilter.Promql),
				},
			},
			Rules: expandMetricAnomalyRules(metricAnomaly.Rules),
		},
	}

}

func expandMetricAnomalyRules(rules []*MetricAnomalyRule) []*cxsdk.MetricAnomalyRule {
	result := make([]*cxsdk.MetricAnomalyRule, len(rules))
	for i := range rules {
		result[i] = expandMetricAnomalyRule(rules[i])
	}

	return result
}

func expandMetricAnomalyRule(rule *MetricAnomalyRule) *cxsdk.MetricAnomalyRule {
	if rule == nil {
		return nil
	}

	return &cxsdk.MetricAnomalyRule{
		Condition: expandMetricAnomalyCondition(rule.Condition),
	}
}

func expandMetricAnomalyCondition(condition MetricAnomalyCondition) *cxsdk.MetricAnomalyCondition {
	return &cxsdk.MetricAnomalyCondition{
		Threshold:           wrapperspb.Double(condition.Threshold.AsApproximateFloat64()),
		ForOverPct:          wrapperspb.UInt32(condition.ForOverPct),
		OfTheLast:           expandMetricTimeWindow(condition.OfTheLast),
		MinNonNullValuesPct: wrapperspb.UInt32(condition.MinNonNullValuesPct),
		ConditionType:       MetricAnomalyConditionTypeToProto[condition.ConditionType],
	}
}

func expandLogsAnomaly(anomaly *LogsAnomaly) *cxsdk.AlertDefPropertiesLogsAnomaly {
	return &cxsdk.AlertDefPropertiesLogsAnomaly{
		LogsAnomaly: &cxsdk.LogsAnomalyType{
			LogsFilter:                expandLogsFilter(anomaly.LogsFilter),
			Rules:                     expandLogsAnomalyRules(anomaly.Rules),
			NotificationPayloadFilter: utils.StringSliceToWrappedStringSlice(anomaly.NotificationPayloadFilter),
		},
	}
}

func expandLogsAnomalyRules(rules []LogsAnomalyRule) []*cxsdk.LogsAnomalyRule {
	result := make([]*cxsdk.LogsAnomalyRule, len(rules))
	for i := range rules {
		result[i] = expandLogsAnomalyRule(rules[i])
	}

	return result
}

func expandLogsAnomalyRule(rule LogsAnomalyRule) *cxsdk.LogsAnomalyRule {
	return &cxsdk.LogsAnomalyRule{
		Condition: expandLogsAnomalyRuleCondition(rule.Condition),
	}
}

func expandLogsAnomalyRuleCondition(condition LogsAnomalyCondition) *cxsdk.LogsAnomalyCondition {
	return &cxsdk.LogsAnomalyCondition{
		MinimumThreshold: wrapperspb.Double(condition.MinimumThreshold.AsApproximateFloat64()),
		TimeWindow:       expandLogsTimeWindow(condition.TimeWindow),
		ConditionType:    cxsdk.LogsAnomalyConditionTypeMoreThanOrUnspecified,
	}
}

func expandFlow(flow *Flow) *cxsdk.AlertDefPropertiesFlow {
	return &cxsdk.AlertDefPropertiesFlow{
		Flow: &cxsdk.FlowType{
			Stages:             expandFlowStages(flow.Stages),
			EnforceSuppression: wrapperspb.Bool(flow.EnforceSuppression),
		},
	}
}

func expandFlowStages(stages []FlowStage) []*cxsdk.FlowStages {
	result := make([]*cxsdk.FlowStages, len(stages))
	for i, stage := range stages {
		result[i] = expandFlowStage(stage)
	}

	return result
}

func expandFlowStage(stage FlowStage) *cxsdk.FlowStages {
	return &cxsdk.FlowStages{
		FlowStages:    expandFlowStagesType(stage.FlowStagesType),
		TimeframeMs:   wrapperspb.Int64(stage.TimeframeMs),
		TimeframeType: TimeframeTypeToProto[stage.TimeframeType],
	}
}

func expandFlowStagesType(stagesType FlowStagesType) *cxsdk.FlowStagesGroups {
	return &cxsdk.FlowStagesGroups{
		FlowStagesGroups: &cxsdk.FlowStagesGroupsValue{
			Groups: expandFlowStagesGroups(stagesType.Groups),
		},
	}
}

func expandFlowStagesGroups(groups []FlowStageGroup) []*cxsdk.FlowStagesGroup {
	result := make([]*cxsdk.FlowStagesGroup, len(groups))
	for i, group := range groups {
		result[i] = expandFlowStagesGroup(group)
	}

	return result
}

func expandFlowStagesGroup(group FlowStageGroup) *cxsdk.FlowStagesGroup {
	return &cxsdk.FlowStagesGroup{
		AlertDefs: expandFlowStagesGroupsAlertDefs(group.AlertDefs),
		NextOp:    FlowStageGroupNextOpToProto[group.NextOp],
		AlertsOp:  FlowStageGroupAlertsOpToProto[group.AlertsOp],
	}
}

func expandFlowStagesGroupsAlertDefs(alertDefs []FlowStagesGroupsAlertDefs) []*cxsdk.FlowStagesGroupsAlertDefs {
	result := make([]*cxsdk.FlowStagesGroupsAlertDefs, len(alertDefs))
	for i := range alertDefs {
		result[i] = expandFlowStagesGroupsAlertDef(alertDefs[i])
	}

	return result
}

func expandFlowStagesGroupsAlertDef(defs FlowStagesGroupsAlertDefs) *cxsdk.FlowStagesGroupsAlertDefs {
	return &cxsdk.FlowStagesGroupsAlertDefs{
		Id:  wrapperspb.String(defs.Id),
		Not: wrapperspb.Bool(defs.Not),
	}
}

func expandTracingThreshold(tracingThreshold *TracingThreshold) *cxsdk.AlertDefPropertiesTracingThreshold {
	return &cxsdk.AlertDefPropertiesTracingThreshold{
		TracingThreshold: &cxsdk.TracingThresholdType{
			TracingFilter:             expandTracingFilter(tracingThreshold.TracingFilter),
			Rules:                     expandTracingThresholdRules(tracingThreshold.Rules),
			NotificationPayloadFilter: utils.StringSliceToWrappedStringSlice(tracingThreshold.NotificationPayloadFilter),
		},
	}
}

func expandTracingFilter(filter *TracingFilter) *cxsdk.TracingFilter {
	if filter == nil {
		return nil
	}

	return &cxsdk.TracingFilter{
		FilterType: expandTracingSimpleFilter(filter.Simple),
	}
}

func expandTracingSimpleFilter(filter *TracingSimpleFilter) *cxsdk.TracingFilterSimpleFilter {
	return &cxsdk.TracingFilterSimpleFilter{
		SimpleFilter: &cxsdk.TracingSimpleFilter{
			TracingLabelFilters: expandTracingLabelFilters(filter.TracingLabelFilters),
			LatencyThresholdMs:  wrapperspb.UInt64(*filter.LatencyThresholdMs),
		},
	}
}

func expandTracingLabelFilters(filters *TracingLabelFilters) *cxsdk.TracingLabelFilters {
	if filters == nil {
		return nil
	}

	return &cxsdk.TracingLabelFilters{
		ApplicationName: expandTracingFilterTypes(filters.ApplicationName),
		SubsystemName:   expandTracingFilterTypes(filters.SubsystemName),
		ServiceName:     expandTracingFilterTypes(filters.ServiceName),
		OperationName:   expandTracingFilterTypes(filters.OperationName),
		SpanFields:      expandTracingSpanFieldsFilterTypes(filters.SpanFields),
	}
}

func expandTracingFilterTypes(filters []TracingFilterType) []*cxsdk.TracingFilterType {
	result := make([]*cxsdk.TracingFilterType, len(filters))
	for i := range filters {
		result[i] = expandTracingFilterType(filters[i])
	}

	return result
}

func expandTracingFilterType(filterType TracingFilterType) *cxsdk.TracingFilterType {
	return &cxsdk.TracingFilterType{
		Values:    utils.StringSliceToWrappedStringSlice(filterType.Values),
		Operation: TracingFilterOperationTypeToProto[filterType.Operation],
	}
}

func expandTracingSpanFieldsFilterTypes(fields []TracingSpanFieldsFilterType) []*cxsdk.TracingSpanFieldsFilterType {
	result := make([]*cxsdk.TracingSpanFieldsFilterType, len(fields))
	for i := range fields {
		result[i] = expandTracingSpanFieldsFilterType(fields[i])
	}

	return result
}

func expandTracingSpanFieldsFilterType(filterType TracingSpanFieldsFilterType) *cxsdk.TracingSpanFieldsFilterType {
	return &cxsdk.TracingSpanFieldsFilterType{
		Key:        wrapperspb.String(filterType.Key),
		FilterType: expandTracingFilterType(filterType.FilterType),
	}
}

func expandTracingThresholdRules(rules []TracingThresholdRule) []*cxsdk.TracingThresholdRule {
	result := make([]*cxsdk.TracingThresholdRule, len(rules))
	for i := range rules {
		result[i] = expandTracingThresholdRule(rules[i])
	}

	return result
}

func expandTracingThresholdRule(rule TracingThresholdRule) *cxsdk.TracingThresholdRule {
	return &cxsdk.TracingThresholdRule{
		Condition: expandTracingThresholdCondition(rule.Condition),
	}
}

func expandTracingThresholdCondition(condition TracingThresholdRuleCondition) *cxsdk.TracingThresholdCondition {
	return &cxsdk.TracingThresholdCondition{
		SpanAmount:    wrapperspb.Double(condition.SpanAmount.AsApproximateFloat64()),
		TimeWindow:    expandTracingTimeWindow(condition.TimeWindow),
		ConditionType: cxsdk.TracingThresholdConditionTypeMoreThanOrUnspecified,
	}
}

func expandTracingTimeWindow(timeWindow TracingTimeWindow) *cxsdk.TracingTimeWindow {
	return &cxsdk.TracingTimeWindow{
		Type: &cxsdk.TracingTimeWindowSpecificValue{
			TracingTimeWindowValue: TracingTimeWindowSpecificValueToProto[*timeWindow.SpecificValue],
		},
	}
}

func expandMetricThreshold(threshold *MetricThreshold) *cxsdk.AlertDefPropertiesMetricThreshold {
	return &cxsdk.AlertDefPropertiesMetricThreshold{
		MetricThreshold: &cxsdk.MetricThresholdType{
			MetricFilter:               expandMetricFilter(threshold.MetricFilter),
			MissingValues:              expandMetricMissingValues(&threshold.MissingValues),
			Rules:                      expandMetricThresholdRules(threshold.Rules),
			UndetectedValuesManagement: expandUndetectedValuesManagement(threshold.UndetectedValuesManagement),
		},
	}
}

func expandMetricFilter(metricFilter MetricFilter) *cxsdk.MetricFilter {
	return &cxsdk.MetricFilter{
		Type: &cxsdk.MetricFilterPromql{
			Promql: wrapperspb.String(metricFilter.Promql),
		},
	}
}

func expandMetricThresholdRules(rules []MetricThresholdRule) []*cxsdk.MetricThresholdRule {
	result := make([]*cxsdk.MetricThresholdRule, len(rules))
	for i := range rules {
		result[i] = expandMetricThresholdRule(rules[i])
	}

	return result
}

func expandMetricThresholdRule(rule MetricThresholdRule) *cxsdk.MetricThresholdRule {
	return &cxsdk.MetricThresholdRule{
		Condition: expandMetricThresholdCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override),
	}
}

func expandMetricThresholdCondition(condition MetricThresholdRuleCondition) *cxsdk.MetricThresholdCondition {
	return &cxsdk.MetricThresholdCondition{
		Threshold:     wrapperspb.Double(condition.Threshold.AsApproximateFloat64()),
		ForOverPct:    wrapperspb.UInt32(condition.ForOverPct),
		OfTheLast:     expandMetricTimeWindow(condition.OfTheLast),
		ConditionType: MetricThresholdConditionTypeToProto[condition.ConditionType],
	}
}

func expandMetricTimeWindow(timeWindow MetricTimeWindow) *cxsdk.MetricTimeWindow {
	return &cxsdk.MetricTimeWindow{
		Type: &cxsdk.MetricTimeWindowSpecificValue{
			MetricTimeWindowSpecificValue: MetricTimeWindowToProto[timeWindow.SpecificValue],
		},
	}
}

func expandMetricMissingValues(missingValues *MetricMissingValues) *cxsdk.MetricMissingValues {
	if missingValues == nil {
		return nil
	} else if missingValues.ReplaceWithZero != nil {
		return &cxsdk.MetricMissingValues{
			MissingValues: &cxsdk.MetricMissingValuesReplaceWithZero{
				ReplaceWithZero: wrapperspb.Bool(*missingValues.ReplaceWithZero),
			},
		}
	} else if missingValues.MinNonNullValuesPct != nil {
		return &cxsdk.MetricMissingValues{
			MissingValues: &cxsdk.MetricMissingValuesMinNonNullValuesPct{
				MinNonNullValuesPct: wrapperspb.UInt32(*missingValues.MinNonNullValuesPct),
			},
		}
	}

	return nil
}

func expandLogsImmediate(immediate *LogsImmediate) *cxsdk.AlertDefPropertiesLogsImmediate {
	return &cxsdk.AlertDefPropertiesLogsImmediate{
		LogsImmediate: &cxsdk.LogsImmediateType{
			LogsFilter:                expandLogsFilter(immediate.LogsFilter),
			NotificationPayloadFilter: utils.StringSliceToWrappedStringSlice(immediate.NotificationPayloadFilter),
		},
	}
}

func expandLogsThreshold(logsThreshold *LogsThreshold) *cxsdk.AlertDefPropertiesLogsThreshold {
	return &cxsdk.AlertDefPropertiesLogsThreshold{
		LogsThreshold: &cxsdk.LogsThresholdType{
			LogsFilter:                 expandLogsFilter(logsThreshold.LogsFilter),
			UndetectedValuesManagement: expandUndetectedValuesManagement(logsThreshold.UndetectedValuesManagement),
			Rules:                      expandLogsThresholdRules(logsThreshold.Rules),
			NotificationPayloadFilter:  utils.StringSliceToWrappedStringSlice(logsThreshold.NotificationPayloadFilter),
		},
	}
}

func expandLogsRatioThreshold(logsRatioThreshold *LogsRatioThreshold) *cxsdk.AlertDefPropertiesLogsRatioThreshold {
	return &cxsdk.AlertDefPropertiesLogsRatioThreshold{
		LogsRatioThreshold: &cxsdk.LogsRatioThresholdType{
			Numerator:        expandLogsFilter(&logsRatioThreshold.Numerator),
			NumeratorAlias:   wrapperspb.String(logsRatioThreshold.NumeratorAlias),
			Denominator:      expandLogsFilter(&logsRatioThreshold.Denominator),
			DenominatorAlias: wrapperspb.String(logsRatioThreshold.DenominatorAlias),
			Rules:            expandLogsRatioThresholdRules(logsRatioThreshold.Rules),
		},
	}
}

func expandLogsTimeRelativeThreshold(threshold *LogsTimeRelativeThreshold) *cxsdk.AlertDefPropertiesLogsTimeRelativeThreshold {
	return &cxsdk.AlertDefPropertiesLogsTimeRelativeThreshold{
		LogsTimeRelativeThreshold: &cxsdk.LogsTimeRelativeThresholdType{
			LogsFilter:                 expandLogsFilter(&threshold.LogsFilter),
			Rules:                      expandLogsTimeRelativeRules(threshold.Rules),
			IgnoreInfinity:             wrapperspb.Bool(threshold.IgnoreInfinity),
			NotificationPayloadFilter:  utils.StringSliceToWrappedStringSlice(threshold.NotificationPayloadFilter),
			UndetectedValuesManagement: expandUndetectedValuesManagement(&threshold.UndetectedValuesManagement),
		},
	}
}

func expandLogsTimeRelativeRules(rules []LogsTimeRelativeRule) []*cxsdk.LogsTimeRelativeRule {
	result := make([]*cxsdk.LogsTimeRelativeRule, len(rules))
	for i := range rules {
		result[i] = expandLogsTimeRelativeRule(rules[i])
	}

	return result
}

func expandLogsTimeRelativeRule(rule LogsTimeRelativeRule) *cxsdk.LogsTimeRelativeRule {
	return &cxsdk.LogsTimeRelativeRule{
		Condition: expandLogsTimeRelativeCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override),
	}
}

func expandLogsTimeRelativeCondition(condition LogsTimeRelativeCondition) *cxsdk.LogsTimeRelativeCondition {
	return &cxsdk.LogsTimeRelativeCondition{
		Threshold:     wrapperspb.Double(condition.Threshold.AsApproximateFloat64()),
		ComparedTo:    LogsTimeRelativeComparedToToProto[condition.ComparedTo],
		ConditionType: LogsTimeRelativeConditionTypeToProto[condition.ConditionType],
	}
}

func expandLogsRatioThresholdRules(rules []LogsRatioThresholdRule) []*cxsdk.LogsRatioRules {
	result := make([]*cxsdk.LogsRatioRules, len(rules))
	for i := range rules {
		result[i] = expandLogsRatioThresholdRule(rules[i])
	}
	return result
}

func expandLogsRatioThresholdRule(rule LogsRatioThresholdRule) *cxsdk.LogsRatioRules {
	return &cxsdk.LogsRatioRules{
		Condition: expandLogsRatioCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override),
	}
}

func expandLogsRatioCondition(condition LogsRatioCondition) *cxsdk.LogsRatioCondition {
	return &cxsdk.LogsRatioCondition{
		Threshold:     wrapperspb.Double(condition.Threshold.AsApproximateFloat64()),
		TimeWindow:    expandLogsRatioTimeWindow(condition.TimeWindow),
		ConditionType: LogsRatioConditionTypeToProto[condition.ConditionType],
	}
}

func expandLogsRatioTimeWindow(timeWindow LogsRatioTimeWindow) *cxsdk.LogsRatioTimeWindow {
	return &cxsdk.LogsRatioTimeWindow{
		Type: &cxsdk.LogsRatioTimeWindowSpecificValue{
			LogsRatioTimeWindowSpecificValue: LogsRatioTimeWindowToProto[timeWindow],
		},
	}
}

func expandAlertOverride(override AlertOverride) *cxsdk.AlertDefPriorityOverride {
	return &cxsdk.AlertDefPriorityOverride{
		Priority: AlertPriorityToProtoPriority[override.Priority],
	}
}

func expandLogsFilter(filter *LogsFilter) *cxsdk.LogsFilter {
	if filter == nil {
		return nil
	}

	return expandLogsFilterType(&cxsdk.LogsFilter{}, filter.FilterType)
}

func expandLogsFilterType(filter *cxsdk.LogsFilter, filterType LogsFilterType) *cxsdk.LogsFilter {
	if simpleFilter := filterType.SimpleFilter; simpleFilter != nil {
		filter.FilterType = expandSimpleFilter(simpleFilter)
	}

	return filter
}

func expandSimpleFilter(filter *LogsSimpleFilter) *cxsdk.LogsFilterSimpleFilter {
	return &cxsdk.LogsFilterSimpleFilter{
		SimpleFilter: &cxsdk.SimpleFilter{
			LuceneQuery:  utils.StringPointerToWrapperspbString(filter.LuceneQuery),
			LabelFilters: expandLabelFilters(filter.LabelFilters),
		},
	}
}

func expandLabelFilters(filters *LabelFilters) *cxsdk.LabelFilters {
	return &cxsdk.LabelFilters{
		ApplicationName: expandLabelFilterTypes(filters.ApplicationName),
		SubsystemName:   expandLabelFilterTypes(filters.SubsystemName),
		Severities:      expandLogSeverities(filters.Severity),
	}
}

func expandLogSeverities(severity []LogSeverity) []cxsdk.LogSeverity {
	result := make([]cxsdk.LogSeverity, len(severity))
	for i, s := range severity {
		result[i] = LogSeverityToProtoSeverity[s]
	}

	return result
}

func expandLabelFilterTypes(name []LabelFilterType) []*cxsdk.LabelFilterType {
	result := make([]*cxsdk.LabelFilterType, len(name))
	for i, n := range name {
		result[i] = &cxsdk.LabelFilterType{
			Value:     wrapperspb.String(n.Value),
			Operation: LogsFiltersOperationToProtoOperation[n.Operation],
		}
	}

	return result
}

func expandUndetectedValuesManagement(management *UndetectedValuesManagement) *cxsdk.UndetectedValuesManagement {
	if management == nil {
		return nil
	}
	autoRetireTimeframe := AutoRetireTimeframeToProtoAutoRetireTimeframe[management.AutoRetireTimeframe]
	return &cxsdk.UndetectedValuesManagement{
		TriggerUndetectedValues: wrapperspb.Bool(management.TriggerUndetectedValues),
		AutoRetireTimeframe:     &autoRetireTimeframe,
	}
}

func expandLogsThresholdRules(rules []LogsThresholdRule) []*cxsdk.LogsThresholdRule {
	result := make([]*cxsdk.LogsThresholdRule, len(rules))
	for i := range rules {
		result[i] = expandLogsThresholdRule(rules[i])
	}

	return result
}

func expandLogsThresholdRule(rule LogsThresholdRule) *cxsdk.LogsThresholdRule {
	return &cxsdk.LogsThresholdRule{
		Condition: expandLogsThresholdRuleCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override),
	}
}

func expandLogsThresholdRuleCondition(condition LogsThresholdRuleCondition) *cxsdk.LogsThresholdCondition {
	return &cxsdk.LogsThresholdCondition{
		Threshold:     wrapperspb.Double(condition.Threshold.AsApproximateFloat64()),
		TimeWindow:    expandLogsTimeWindow(condition.TimeWindow),
		ConditionType: LogsThresholdConditionTypeToProto[condition.LogsThresholdConditionType],
	}
}

func expandLogsTimeWindow(timeWindow LogsTimeWindow) *cxsdk.LogsTimeWindow {
	return &cxsdk.LogsTimeWindow{
		Type: &cxsdk.LogsTimeWindowSpecificValue{
			LogsTimeWindowSpecificValue: LogsTimeWindowToProto[timeWindow.SpecificValue],
		},
	}
}

func NewAlert() *Alert {
	return &Alert{
		Spec: AlertSpec{
			EntityLabels: make(map[string]string),
		},
	}
}
