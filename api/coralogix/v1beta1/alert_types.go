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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	alerts "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_definitions_service"
	slos "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// Alert is the Schema for the Alerts API.
//
// Note that this is only for the latest version of the Alerts API. If your account has been created before March 2025, make sure that your account has been migrated before using advanced features of alerts.
//
// **Added in v0.4.0**
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
	AlertPriorityToOpenAPIPriority = map[AlertPriority]alerts.AlertDefPriority{
		AlertPriorityP1: alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P1,
		AlertPriorityP2: alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P2,
		AlertPriorityP3: alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P3,
		AlertPriorityP4: alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P4,
		AlertPriorityP5: alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P5_OR_UNSPECIFIED,
	}
	LogSeverityToOpenAPISeverity = map[LogSeverity]alerts.LogSeverity{
		LogSeverityDebug:    alerts.LOGSEVERITY_LOG_SEVERITY_DEBUG,
		LogSeverityInfo:     alerts.LOGSEVERITY_LOG_SEVERITY_INFO,
		LogSeverityWarning:  alerts.LOGSEVERITY_LOG_SEVERITY_WARNING,
		LogSeverityError:    alerts.LOGSEVERITY_LOG_SEVERITY_ERROR,
		LogSeverityCritical: alerts.LOGSEVERITY_LOG_SEVERITY_CRITICAL,
		LogSeverityVerbose:  alerts.LOGSEVERITY_LOG_SEVERITY_VERBOSE_UNSPECIFIED,
	}
	LogsFiltersOperationToOpenAPIOperation = map[LogFilterOperationType]alerts.LogFilterOperationType{
		LogFilterOperationTypeIs:         alerts.LOGFILTEROPERATIONTYPE_LOG_FILTER_OPERATION_TYPE_IS_OR_UNSPECIFIED,
		LogFilterOperationTypeIncludes:   alerts.LOGFILTEROPERATIONTYPE_LOG_FILTER_OPERATION_TYPE_INCLUDES,
		LogFilterOperationTypeEndWith:    alerts.LOGFILTEROPERATIONTYPE_LOG_FILTER_OPERATION_TYPE_ENDS_WITH,
		LogFilterOperationTypeStartsWith: alerts.LOGFILTEROPERATIONTYPE_LOG_FILTER_OPERATION_TYPE_STARTS_WITH,
	}
	DaysOfWeekToOpenAPIDayOfWeek = map[DayOfWeek]alerts.DayOfWeek{
		DayOfWeekSunday:    alerts.DAYOFWEEK_DAY_OF_WEEK_SUNDAY,
		DayOfWeekMonday:    alerts.DAYOFWEEK_DAY_OF_WEEK_MONDAY_OR_UNSPECIFIED,
		DayOfWeekTuesday:   alerts.DAYOFWEEK_DAY_OF_WEEK_TUESDAY,
		DayOfWeekWednesday: alerts.DAYOFWEEK_DAY_OF_WEEK_WEDNESDAY,
		DayOfWeekThursday:  alerts.DAYOFWEEK_DAY_OF_WEEK_THURSDAY,
		DayOfWeekFriday:    alerts.DAYOFWEEK_DAY_OF_WEEK_FRIDAY,
		DayOfWeekSaturday:  alerts.DAYOFWEEK_DAY_OF_WEEK_SATURDAY,
	}
	NotifyOnToOpenAPINotifyOn = map[NotifyOn]alerts.NotifyOn{
		NotifyOnTriggeredOnly:        alerts.NOTIFYON_NOTIFY_ON_TRIGGERED_ONLY_UNSPECIFIED,
		NotifyOnTriggeredAndResolved: alerts.NOTIFYON_NOTIFY_ON_TRIGGERED_AND_RESOLVED,
	}
	AutoRetireTimeframeToOpenAPIAutoRetireTimeframe = map[AutoRetireTimeframe]alerts.V3AutoRetireTimeframe{
		AutoRetireTimeframeNeverOrUnspecified: alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_NEVER_OR_UNSPECIFIED,
		AutoRetireTimeframe5M:                 alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_MINUTES_5,
		AutoRetireTimeframe10M:                alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_MINUTES_10,
		AutoRetireTimeframe1H:                 alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_HOUR_1,
		AutoRetireTimeframe2H:                 alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_HOURS_2,
		AutoRetireTimeframe6H:                 alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_HOURS_6,
		AutoRetireTimeframe12H:                alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_HOURS_12,
		AutoRetireTimeframe24H:                alerts.V3AUTORETIRETIMEFRAME_AUTO_RETIRE_TIMEFRAME_HOURS_24,
	}
	LogsTimeWindowToOpenAPI = map[LogsTimeWindowValue]alerts.LogsTimeWindowValue{
		LogsTimeWindow5Minutes:  alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_MINUTES_5_OR_UNSPECIFIED,
		LogsTimeWindow10Minutes: alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_MINUTES_10,
		LogsTimeWindow15Minutes: alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_MINUTES_15,
		LogsTimeWindow20Minutes: alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_MINUTES_20,
		LogsTimeWindow30Minutes: alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_MINUTES_30,
		LogsTimeWindowHour:      alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_HOUR_1,
		LogsTimeWindow2Hours:    alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_HOURS_2,
		LogsTimeWindow4Hours:    alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_HOURS_4,
		LogsTimeWindow6Hours:    alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_HOURS_6,
		LogsTimeWindow12Hours:   alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_HOURS_12,
		LogsTimeWindow24Hours:   alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_HOURS_24,
		LogsTimeWindow36Hours:   alerts.LOGSTIMEWINDOWVALUE_LOGS_TIME_WINDOW_VALUE_HOURS_36,
	}
	LogsThresholdConditionTypeToOpenAPI = map[LogsThresholdConditionType]alerts.LogsThresholdConditionType{
		LogsThresholdConditionTypeMoreThan: alerts.LOGSTHRESHOLDCONDITIONTYPE_LOGS_THRESHOLD_CONDITION_TYPE_MORE_THAN_OR_UNSPECIFIED,
		LogsThresholdConditionTypeLessThan: alerts.LOGSTHRESHOLDCONDITIONTYPE_LOGS_THRESHOLD_CONDITION_TYPE_LESS_THAN,
	}
	LogsRatioTimeWindowToOpenAPI = map[LogsRatioTimeWindowValue]alerts.LogsRatioTimeWindowValue{
		LogsRatioTimeWindowMinutes5:  alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_MINUTES_5_OR_UNSPECIFIED,
		LogsRatioTimeWindowMinutes10: alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_MINUTES_10,
		LogsRatioTimeWindowMinutes15: alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_MINUTES_15,
		LogsRatioTimeWindowMinutes30: alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_MINUTES_30,
		LogsRatioTimeWindow1Hour:     alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_HOUR_1,
		LogsRatioTimeWindowHours2:    alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_HOURS_2,
		LogsRatioTimeWindowHours4:    alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_HOURS_4,
		LogsRatioTimeWindowHours6:    alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_HOURS_6,
		LogsRatioTimeWindowHours12:   alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_HOURS_12,
		LogsRatioTimeWindowHours24:   alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_HOURS_24,
		LogsRatioTimeWindowHours36:   alerts.LOGSRATIOTIMEWINDOWVALUE_LOGS_RATIO_TIME_WINDOW_VALUE_HOURS_36,
	}
	LogsRatioConditionTypeToOpenAPI = map[LogsRatioConditionType]alerts.LogsRatioConditionType{
		LogsRatioConditionTypeMoreThan: alerts.LOGSRATIOCONDITIONTYPE_LOGS_RATIO_CONDITION_TYPE_MORE_THAN_OR_UNSPECIFIED,
		LogsRatioConditionTypeLessThan: alerts.LOGSRATIOCONDITIONTYPE_LOGS_RATIO_CONDITION_TYPE_LESS_THAN,
	}
	LogsTimeRelativeComparedToOpenAPI = map[LogsTimeRelativeComparedTo]alerts.LogsTimeRelativeComparedTo{
		LogsTimeRelativeComparedToPreviousHour:      alerts.LOGSTIMERELATIVECOMPAREDTO_LOGS_TIME_RELATIVE_COMPARED_TO_PREVIOUS_HOUR_OR_UNSPECIFIED,
		LogsTimeRelativeComparedToSameHourYesterday: alerts.LOGSTIMERELATIVECOMPAREDTO_LOGS_TIME_RELATIVE_COMPARED_TO_SAME_HOUR_YESTERDAY,
		LogsTimeRelativeComparedToSameHourLastWeek:  alerts.LOGSTIMERELATIVECOMPAREDTO_LOGS_TIME_RELATIVE_COMPARED_TO_SAME_HOUR_LAST_WEEK,
		LogsTimeRelativeComparedToYesterday:         alerts.LOGSTIMERELATIVECOMPAREDTO_LOGS_TIME_RELATIVE_COMPARED_TO_YESTERDAY,
		LogsTimeRelativeComparedToSameDayLastWeek:   alerts.LOGSTIMERELATIVECOMPAREDTO_LOGS_TIME_RELATIVE_COMPARED_TO_SAME_DAY_LAST_WEEK,
		LogsTimeRelativeComparedToSameDayLastMonth:  alerts.LOGSTIMERELATIVECOMPAREDTO_LOGS_TIME_RELATIVE_COMPARED_TO_SAME_DAY_LAST_MONTH,
	}
	LogsTimeRelativeConditionTypeToOpenAPI = map[LogsTimeRelativeConditionType]alerts.LogsTimeRelativeConditionType{
		LogsTimeRelativeConditionTypeMoreThan: alerts.LOGSTIMERELATIVECONDITIONTYPE_LOGS_TIME_RELATIVE_CONDITION_TYPE_MORE_THAN_OR_UNSPECIFIED,
		LogsTimeRelativeConditionTypeLessThan: alerts.LOGSTIMERELATIVECONDITIONTYPE_LOGS_TIME_RELATIVE_CONDITION_TYPE_LESS_THAN,
	}
	MetricThresholdConditionTypeToOpenAPI = map[MetricThresholdConditionType]alerts.MetricThresholdConditionType{
		MetricThresholdConditionTypeMoreThan:         alerts.METRICTHRESHOLDCONDITIONTYPE_METRIC_THRESHOLD_CONDITION_TYPE_MORE_THAN_OR_UNSPECIFIED,
		MetricThresholdConditionTypeLessThan:         alerts.METRICTHRESHOLDCONDITIONTYPE_METRIC_THRESHOLD_CONDITION_TYPE_LESS_THAN,
		MetricThresholdConditionTypeMoreThanOrEquals: alerts.METRICTHRESHOLDCONDITIONTYPE_METRIC_THRESHOLD_CONDITION_TYPE_MORE_THAN_OR_EQUALS,
		MetricThresholdConditionTypeLessThanOrEquals: alerts.METRICTHRESHOLDCONDITIONTYPE_METRIC_THRESHOLD_CONDITION_TYPE_LESS_THAN_OR_EQUALS,
	}
	MetricTimeWindowToOpenAPI = map[MetricTimeWindowSpecificValue]alerts.MetricTimeWindowValue{
		MetricTimeWindowValue1Minute:   alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_MINUTES_1_OR_UNSPECIFIED,
		MetricTimeWindowValue5Minutes:  alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_MINUTES_5,
		MetricTimeWindowValue10Minutes: alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_MINUTES_10,
		MetricTimeWindowValue15Minutes: alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_MINUTES_15,
		MetricTimeWindowValue20Minutes: alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_MINUTES_20,
		MetricTimeWindowValue30Minutes: alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_MINUTES_30,
		MetricTimeWindowValue1Hour:     alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_HOUR_1,
		MetricTimeWindowValue2Hours:    alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_HOURS_2,
		MetricTimeWindowValue4Hours:    alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_HOURS_4,
		MetricTimeWindowValue6Hours:    alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_HOURS_6,
		MetricTimeWindowValue12Hours:   alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_HOURS_12,
		MetricTimeWindowValue24Hours:   alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_HOURS_24,
		MetricTimeWindowValue36Hours:   alerts.METRICTIMEWINDOWVALUE_METRIC_TIME_WINDOW_VALUE_HOURS_36,
	}
	TracingTimeWindowSpecificValueToOpenAPI = map[TracingTimeWindowSpecificValue]alerts.TracingTimeWindowValue{
		TracingTimeWindowValue5Minutes:  alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_MINUTES_5_OR_UNSPECIFIED,
		TracingTimeWindowValue10Minutes: alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_MINUTES_10,
		TracingTimeWindowValue15Minutes: alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_MINUTES_15,
		TracingTimeWindowValue20Minutes: alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_MINUTES_20,
		TracingTimeWindowValue30Minutes: alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_MINUTES_30,
		TracingTimeWindowValue1Hour:     alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_HOUR_1,
		TracingTimeWindowValue2Hours:    alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_HOURS_2,
		TracingTimeWindowValue4Hours:    alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_HOURS_4,
		TracingTimeWindowValue6Hours:    alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_HOURS_6,
		TracingTimeWindowValue12Hours:   alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_HOURS_12,
		TracingTimeWindowValue24Hours:   alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_HOURS_24,
		TracingTimeWindowValue36Hours:   alerts.TRACINGTIMEWINDOWVALUE_TRACING_TIME_WINDOW_VALUE_HOURS_36,
	}
	TracingFilterOperationTypeToOpenAPI = map[TracingFilterOperationType]alerts.TracingFilterOperationType{
		TracingFilterOperationTypeIs:         alerts.TRACINGFILTEROPERATIONTYPE_TRACING_FILTER_OPERATION_TYPE_IS_OR_UNSPECIFIED,
		TracingFilterOperationTypeIncludes:   alerts.TRACINGFILTEROPERATIONTYPE_TRACING_FILTER_OPERATION_TYPE_INCLUDES,
		TracingFilterOperationTypeEndsWith:   alerts.TRACINGFILTEROPERATIONTYPE_TRACING_FILTER_OPERATION_TYPE_ENDS_WITH,
		TracingFilterOperationTypeStartsWith: alerts.TRACINGFILTEROPERATIONTYPE_TRACING_FILTER_OPERATION_TYPE_STARTS_WITH,
		TracingFilterOperationTypeIsNot:      alerts.TRACINGFILTEROPERATIONTYPE_TRACING_FILTER_OPERATION_TYPE_IS_NOT,
	}
	TimeframeTypeToOpenAPI = map[FlowTimeframeType]alerts.TimeframeType{
		TimeframeTypeUnspecified: alerts.TIMEFRAMETYPE_TIMEFRAME_TYPE_UNSPECIFIED,
		TimeframeTypeUpTo:        alerts.TIMEFRAMETYPE_TIMEFRAME_TYPE_UP_TO,
	}
	FlowStageGroupAlertsOpToOpenAPI = map[FlowStageGroupAlertsOp]alerts.AlertsOp{
		FlowStageGroupAlertsOpAnd: alerts.ALERTSOP_ALERTS_OP_AND_OR_UNSPECIFIED,
		FlowStageGroupAlertsOpOr:  alerts.ALERTSOP_ALERTS_OP_OR,
	}
	FlowStageGroupNextOpToOpenAPI = map[FlowStageGroupAlertsOp]alerts.NextOp{
		FlowStageGroupAlertsOpAnd: alerts.NEXTOP_NEXT_OP_AND_OR_UNSPECIFIED,
		FlowStageGroupAlertsOpOr:  alerts.NEXTOP_NEXT_OP_OR,
	}
	MetricAnomalyConditionTypeToOpenAPI = map[MetricAnomalyConditionType]alerts.MetricAnomalyConditionType{
		MetricAnomalyConditionTypeMoreThanUsual: alerts.METRICANOMALYCONDITIONTYPE_METRIC_ANOMALY_CONDITION_TYPE_MORE_THAN_USUAL_OR_UNSPECIFIED,
		MetricAnomalyConditionTypeLessThanUsual: alerts.METRICANOMALYCONDITIONTYPE_METRIC_ANOMALY_CONDITION_TYPE_LESS_THAN_USUAL,
	}
	LogsNewValueTimeWindowValueToOpenAPI = map[LogsNewValueTimeWindowSpecificValue]alerts.LogsNewValueTimeWindowValue{
		LogsNewValueTimeWindowValue12Hours: alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_HOURS_12_OR_UNSPECIFIED,
		LogsNewValueTimeWindowValue24Hours: alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_HOURS_24,
		LogsNewValueTimeWindowValue48Hours: alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_HOURS_48,
		LogsNewValueTimeWindowValue72Hours: alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_HOURS_72,
		LogsNewValueTimeWindowValue1Week:   alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_WEEK_1,
		LogsNewValueTimeWindowValue1Month:  alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_MONTH_1,
		LogsNewValueTimeWindowValue2Months: alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_MONTHS_2,
		LogsNewValueTimeWindowValue3Months: alerts.LOGSNEWVALUETIMEWINDOWVALUE_LOGS_NEW_VALUE_TIME_WINDOW_VALUE_MONTHS_3,
	}
	LogsUniqueCountTimeWindowValueToOpenAPI = map[LogsUniqueCountTimeWindowSpecificValue]alerts.LogsUniqueValueTimeWindowValue{
		LogsUniqueCountTimeWindowValue1Minute:   alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_MINUTE_1_OR_UNSPECIFIED,
		LogsUniqueCountTimeWindowValue5Minutes:  alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_MINUTES_5,
		LogsUniqueCountTimeWindowValue10Minutes: alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_MINUTES_10,
		LogsUniqueCountTimeWindowValue15Minutes: alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_MINUTES_15,
		LogsUniqueCountTimeWindowValue20Minutes: alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_MINUTES_20,
		LogsUniqueCountTimeWindowValue30Minutes: alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_MINUTES_30,
		LogsUniqueCountTimeWindowValue1Hour:     alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_HOURS_1,
		LogsUniqueCountTimeWindowValue2Hours:    alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_HOURS_2,
		LogsUniqueCountTimeWindowValue4Hours:    alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_HOURS_4,
		LogsUniqueCountTimeWindowValue6Hours:    alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_HOURS_6,
		LogsUniqueCountTimeWindowValue12Hours:   alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_HOURS_12,
		LogsUniqueCountTimeWindowValue24Hours:   alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_HOURS_24,
		LogsUniqueCountTimeWindowValue36Hours:   alerts.LOGSUNIQUEVALUETIMEWINDOWVALUE_LOGS_UNIQUE_VALUE_TIME_WINDOW_VALUE_HOURS_36,
	}
)

// AlertSpec defines the desired state of a Coralogix Alert. For more info check - https://coralogix.com/docs/getting-started-with-coralogix-alerts/.
// +kubebuilder:validation:XValidation:rule="!has(self.alertType.logsImmediate) || !has(self.groupByKeys)",message="groupByKeys is not supported for this alert type"
type AlertSpec struct {
	// Name of the alert
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// Description of the alert
	// +optional
	Description string `json:"description,omitempty"`

	// Priority of the alert.
	// +kubebuilder:default=p5
	Priority AlertPriority `json:"priority"`

	// Enable/disable the alert.
	//+kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Grouping fields for multiple alerts.
	// +optional
	GroupByKeys []string `json:"groupByKeys"`

	// Settings for the attached incidents.
	// +optional
	IncidentsSettings *IncidentsSettings `json:"incidentsSettings,omitempty"`

	// Where notifications should be sent to.
	// +optional
	NotificationGroup *NotificationGroup `json:"notificationGroup,omitempty"`

	// Do not use.
	// Deprecated: Legacy field for when multiple notification groups were attached.
	// +optional
	NotificationGroupExcess []NotificationGroup `json:"notificationGroupExcess,omitempty"`

	// Labels attached to the alert.
	// +optional
	EntityLabels map[string]string `json:"entityLabels,omitempty"`
	//+kubebuilder:default=false
	PhantomMode bool `json:"phantomMode,omitempty"`

	// Alert activity schedule. Will be activated all the time if not specified.
	// +optional
	Schedule *AlertSchedule `json:"schedule,omitempty"`

	// Type of alert.
	TypeDefinition AlertTypeDefinition `json:"alertType"`
}

// AlertStatus defines the observed state of Alert
type AlertStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

func (a *Alert) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *Alert) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
}

func (a *Alert) HasIDInStatus() bool {
	return a.Status.ID != nil && *a.Status.ID != ""
}

func (a *Alert) GetPrintableStatus() string {
	return a.Status.PrintableStatus
}

func (a *Alert) SetPrintableStatus(printableStatus string) {
	a.Status.PrintableStatus = printableStatus
}

// +kubebuilder:validation:Pattern=`^UTC[+-]\d{2}$`
// +kubebuilder:default=UTC+00
// A time zone expressed in UTC offsets.
type TimeZone string

// The schedule for when the alert is active.
type AlertSchedule struct {
	//+kubebuilder:default=UTC+00
	// Time zone.
	TimeZone TimeZone `json:"timeZone"`

	// Schedule to have the alert active.
	// +optional
	ActiveOn *ActiveOn `json:"activeOn,omitempty"`
}

// Settings for attached incidents.
type IncidentsSettings struct {

	// When to notify.
	//+kubebuilder:default=triggeredOnly
	NotifyOn NotifyOn `json:"notifyOn,omitempty"`

	// When to re-notify.
	RetriggeringPeriod RetriggeringPeriod `json:"retriggeringPeriod,omitempty"`
}

// +kubebuilder:validation:Enum=triggeredOnly;triggeredAndResolved
// When to notify.
type NotifyOn string

const (
	NotifyOnTriggeredOnly        NotifyOn = "triggeredOnly"
	NotifyOnTriggeredAndResolved NotifyOn = "triggeredAndResolved"
)

// +kubebuilder:validation:Enum={"never","5m","10m","1h","2h","6h","12h","24h"}
// Automatically retire the alert after...
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

// When to re-trigger the alert.
type RetriggeringPeriod struct {
	// Delay between re-triggered alerts.
	// +optional
	Minutes *int64 `json:"minutes,omitempty"`
}

// Notification group to use for alert notifications.
// +kubebuilder:validation:XValidation:rule="!(has(self.destinations) && has(self.router))", message="At most one of Destinations or Router can be set."
type NotificationGroup struct {
	// Group notification by these keys.
	// +optional
	GroupByKeys []string `json:"groupByKeys"`

	// Webhooks to trigger for notifications.
	// +optional
	Webhooks []WebhookSettings `json:"webhooks"`

	// Do not use.
	// Deprecated: This field is deprecated and will be removed in a future version.
	// +optional
	Destinations []NotificationDestination `json:"destinations,omitempty"`
}

// Settings for a notification webhook.
type WebhookSettings struct {

	// When to re-trigger.
	RetriggeringPeriod RetriggeringPeriod `json:"retriggeringPeriod"`

	// +kubebuilder:default=triggeredOnly
	// When to notify.
	NotifyOn NotifyOn `json:"notifyOn"`

	// Type and spec of webhook.
	Integration IntegrationType `json:"integration"`
}

// Type and spec of the webhook.
// +kubebuilder:validation:XValidation:rule="has(self.integrationRef) || has(self.recipients)",message="Exactly one of integrationRef or recipients is required"
type IntegrationType struct {

	// Reference to the webhook.
	// +optional
	IntegrationRef *IntegrationRef `json:"integrationRef,omitempty"`

	// Recipients for the notification.
	// +optional
	Recipients []string `json:"recipients,omitempty"`
}

// Reference to the integration.
// +kubebuilder:validation:XValidation:rule="has(self.backendRef) || has(self.resourceRef)",message="Exactly one of backendRef or resourceRef is required"
type IntegrationRef struct {

	// Backend reference for the outbound webhook.
	// +optional
	BackendRef *OutboundWebhookBackendRef `json:"backendRef,omitempty"`

	// Resource reference for use with the alert notification.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

// Outbound webhook reference.
// +kubebuilder:validation:XValidation:rule="has(self.id) != has(self.name)",message="One of id or name is required"
type OutboundWebhookBackendRef struct {
	// Webhook ID.
	// +optional
	ID *int64 `json:"id,omitempty"`

	// Name of the webhook.
	// +optional
	Name *string `json:"name,omitempty"`
}

// Reference to the alert on Coralogix.
// +kubebuilder:validation:XValidation:rule="has(self.id) != has(self.name)",message="One of id or name is required"
type AlertBackendRef struct {

	// Alert ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// Name of the alert.
	// +optional
	Name *string `json:"name,omitempty"`
}

// Reference to a resource within the cluster.
type ResourceRef struct {
	// Name of the resource.
	Name string `json:"name"`

	// Kubernetes namespace.
	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

type NotificationDestination struct {
	// Connector is the connector for the destination. Should be one of backendRef or resourceRef.
	Connector NCRef `json:"connector"`

	// Preset is the preset for the destination. Should be one of backendRef or resourceRef.
	// +optional
	Preset *NCRef `json:"preset,omitempty"`

	// +kubebuilder:default=triggeredOnly
	// When to notify.
	NotifyOn NotifyOn `json:"notifyOn"`

	// The routing configuration to override from the connector/preset for triggered notifications.
	TriggeredRoutingOverrides NotificationRouting `json:"triggeredRoutingOverrides"`

	// Optional routing configuration to override from the connector/preset for resolved notifications.
	// +optional
	ResolvedRoutingOverrides *NotificationRouting `json:"resolvedRoutingOverrides,omitempty"`
}

type NotificationRouter struct {
	// +kubebuilder:default=triggeredOnly
	// When to notify.
	NotifyOn NotifyOn `json:"notifyOn"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type NCRef struct {
	// BackendRef is a reference to a backend resource.
	// +optional
	BackendRef *NCBackendRef `json:"backendRef,omitempty"`

	// ResourceRef is a reference to a Kubernetes resource.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

type NCBackendRef struct {
	ID string `json:"id"`
}

type NotificationRouting struct {
	// +optional
	ConfigOverrides *SourceOverrides `json:"configOverrides,omitempty"`
}

type SourceOverrides struct {
	// The ID of the output schema to use for routing notifications
	PayloadType string `json:"payloadType"`

	// Notification message configuration fields.
	// +optional
	MessageConfigFields []ConfigField `json:"messageConfigFields,omitempty"`

	// Connector configuration fields.
	// +optional
	ConnectorConfigFields []ConfigField `json:"connectorConfigFields,omitempty"`
}

type ConfigField struct {
	// The name of the configuration field.
	FieldName string `json:"fieldName"`

	// The template for the configuration field.
	Template string `json:"template"`
}

type ActiveOn struct {
	DayOfWeek []DayOfWeek `json:"dayOfWeek"`
	// +kubebuilder:default="00:00"
	StartTime *TimeOfDay `json:"startTime,omitempty"`
	// +kubebuilder:default="23:59"
	EndTime *TimeOfDay `json:"endTime,omitempty"`
}

// +kubebuilder:validation:Pattern=`^(0\d|1\d|2[0-3]):[0-5]\d$`
// Time of day.
type TimeOfDay string

// +kubebuilder:validation:Enum=sunday;monday;tuesday;wednesday;thursday;friday;saturday
// Day of the week.
type DayOfWeek string

// Day of the week values.
const (
	DayOfWeekSunday    DayOfWeek = "sunday"
	DayOfWeekMonday    DayOfWeek = "monday"
	DayOfWeekTuesday   DayOfWeek = "tuesday"
	DayOfWeekWednesday DayOfWeek = "wednesday"
	DayOfWeekThursday  DayOfWeek = "thursday"
	DayOfWeekFriday    DayOfWeek = "friday"
	DayOfWeekSaturday  DayOfWeek = "saturday"
)

// Alert type definitions.
// +kubebuilder:validation:XValidation:rule="(has(self.logsImmediate) ? 1 : 0) + (has(self.logsThreshold) ? 1 : 0) + (has(self.logsRatioThreshold) ? 1 : 0) + (has(self.logsTimeRelativeThreshold) ? 1 : 0) + (has(self.metricThreshold) ? 1 : 0) + (has(self.tracingThreshold) ? 1 : 0) + (has(self.tracingImmediate) ? 1 : 0) + (has(self.flow) ? 1 : 0) + (has(self.logsAnomaly) ? 1 : 0) + (has(self.metricAnomaly) ? 1 : 0) + (has(self.logsNewValue) ? 1 : 0) + (has(self.logsUniqueCount) ? 1 : 0) + (has(self.sloThreshold) ? 1 : 0) == 1", message="Exactly one of logsImmediate, logsThreshold, logsRatioThreshold, logsTimeRelativeThreshold, metricThreshold, tracingThreshold, tracingImmediate, flow, logsAnomaly, metricAnomaly, logsNewValue, logsUniqueCount, sloThreshold must be set"
type AlertTypeDefinition struct {

	// Immediate alerts for logs.
	// +optional
	LogsImmediate *LogsImmediate `json:"logsImmediate,omitempty"`

	// Alerts for when a log crosses a threshold.
	// +optional
	LogsThreshold *LogsThreshold `json:"logsThreshold,omitempty"`

	// Alerts for when a log exceeds a defined ratio.
	// +optional
	LogsRatioThreshold *LogsRatioThreshold `json:"logsRatioThreshold,omitempty"`

	// Alerts are sent when the number of logs matching a filter is more than or less than a threshold over a specific time window.
	// +optional
	LogsTimeRelativeThreshold *LogsTimeRelativeThreshold `json:"logsTimeRelativeThreshold,omitempty"`

	// Alerts for when a metric crosses a threshold.
	// +optional
	MetricThreshold *MetricThreshold `json:"metricThreshold,omitempty"`

	// Alerts for when traces crosses a threshold.
	// +optional
	TracingThreshold *TracingThreshold `json:"tracingThreshold,omitempty"`

	// Immediate alerts for traces.
	// +optional
	TracingImmediate *TracingImmediate `json:"tracingImmediate,omitempty"`

	// Flow alerts chaining multiple alerts together.
	// +optional
	Flow *Flow `json:"flow,omitempty"`

	// Anomaly alerts for logs.
	// +optional
	LogsAnomaly *LogsAnomaly `json:"logsAnomaly,omitempty"`

	// Anomaly alerts for metrics.
	// +optional
	MetricAnomaly *MetricAnomaly `json:"metricAnomaly,omitempty"`

	// Alerts when a new log value appears.
	// +optional
	LogsNewValue *LogsNewValue `json:"logsNewValue,omitempty"`

	// Alerts for unique count changes.
	// +optional
	LogsUniqueCount *LogsUniqueCount `json:"logsUniqueCount,omitempty"`

	// Alerts for SLO thresholds.
	// +optional
	SloThreshold *SloThreshold `json:"sloThreshold,omitempty"`
}

// Immediate alerts for logs.
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/logs/immediate-notifications/
type LogsImmediate struct {
	// Filter to filter the logs with.
	// +optional
	LogsFilter *LogsFilter `json:"logsFilter,omitempty"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
}

// Alerts for when a log crosses a threshold.
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/logs/threshold-alerts/
type LogsThreshold struct {
	// Filter to filter the logs with.
	// +optional
	LogsFilter *LogsFilter `json:"logsFilter,omitempty"`

	// How to work with undetected values.
	// +optional
	UndetectedValuesManagement *UndetectedValuesManagement `json:"undetectedValuesManagement,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []LogsThresholdRule `json:"rules"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
}

// The rule to match the alert's conditions.
type LogsThresholdRule struct {
	// Condition to match
	Condition LogsThresholdRuleCondition `json:"condition"`

	// Alert overrides.
	// +optional
	Override *AlertOverride `json:"override"`
}

// Threshold rules for logs.
type LogsThresholdRuleCondition struct {
	// Time window in which the condition is checked.
	TimeWindow LogsTimeWindow `json:"timeWindow"`
	// Threshold to match to.
	Threshold resource.Quantity `json:"threshold"`
	// Condition type.
	LogsThresholdConditionType LogsThresholdConditionType `json:"logsThresholdConditionType"`
}

// Time window in which the condition is checked.
type LogsTimeWindow struct {
	SpecificValue LogsTimeWindowValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"5m","10m","15m", "20m","30m","1h","2h","4h","6h","12h","24h","36h"}
// Logs time window type
type LogsTimeWindowValue string

// Logs time window values
const (
	LogsTimeWindow5Minutes  LogsTimeWindowValue = "5m"
	LogsTimeWindow10Minutes LogsTimeWindowValue = "10m"
	LogsTimeWindow15Minutes LogsTimeWindowValue = "15m"
	LogsTimeWindow20Minutes LogsTimeWindowValue = "20m"
	LogsTimeWindow30Minutes LogsTimeWindowValue = "30m"
	LogsTimeWindowHour      LogsTimeWindowValue = "1h"
	LogsTimeWindow2Hours    LogsTimeWindowValue = "2h"
	LogsTimeWindow4Hours    LogsTimeWindowValue = "4h"
	LogsTimeWindow6Hours    LogsTimeWindowValue = "6h"
	LogsTimeWindow12Hours   LogsTimeWindowValue = "12h"
	LogsTimeWindow24Hours   LogsTimeWindowValue = "24h"
	LogsTimeWindow36Hours   LogsTimeWindowValue = "36h"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
// ConditionType type.
type LogsThresholdConditionType string

// Condition type values.
const (
	LogsThresholdConditionTypeMoreThan LogsThresholdConditionType = "moreThan"
	LogsThresholdConditionTypeLessThan LogsThresholdConditionType = "lessThan"
)

// Override alert properties
type AlertOverride struct {
	// Priority to override it
	Priority AlertPriority `json:"priority"`
}

// Logs ratio alerts.
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/logs/ratio-alerts/
type LogsRatioThreshold struct {
	Numerator        LogsFilter `json:"numerator"`
	NumeratorAlias   string     `json:"numeratorAlias"`
	Denominator      LogsFilter `json:"denominator"`
	DenominatorAlias string     `json:"denominatorAlias"`
	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []LogsRatioThresholdRule `json:"rules"`
}

// The rule to match the alert's conditions.
type LogsRatioThresholdRule struct {
	// Condition to match
	Condition LogsRatioCondition `json:"condition"`
	// +optional
	Override *AlertOverride `json:"override"`
}

// Logs ratio condition for matching alerts.
type LogsRatioCondition struct {
	// Threshold to pass.
	Threshold resource.Quantity `json:"threshold"`

	// Time window to evaluate.
	TimeWindow LogsRatioTimeWindow `json:"timeWindow"`

	// Condition to evaluate with.
	ConditionType LogsRatioConditionType `json:"conditionType"`
}

type LogsRatioTimeWindow struct {
	SpecificValue LogsRatioTimeWindowValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"5m","10m","15m","30m","1h","2h","4h","6h","12h","24h","36h"}
// Time window type.
type LogsRatioTimeWindowValue string

// Time window values.
const (
	LogsRatioTimeWindowMinutes5  LogsRatioTimeWindowValue = "5m"
	LogsRatioTimeWindowMinutes10 LogsRatioTimeWindowValue = "10m"
	LogsRatioTimeWindowMinutes15 LogsRatioTimeWindowValue = "15m"
	LogsRatioTimeWindowMinutes30 LogsRatioTimeWindowValue = "30m"
	LogsRatioTimeWindow1Hour     LogsRatioTimeWindowValue = "1h"
	LogsRatioTimeWindowHours2    LogsRatioTimeWindowValue = "2h"
	LogsRatioTimeWindowHours4    LogsRatioTimeWindowValue = "4h"
	LogsRatioTimeWindowHours6    LogsRatioTimeWindowValue = "6h"
	LogsRatioTimeWindowHours12   LogsRatioTimeWindowValue = "12h"
	LogsRatioTimeWindowHours24   LogsRatioTimeWindowValue = "24h"
	LogsRatioTimeWindowHours36   LogsRatioTimeWindowValue = "36h"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
// ConditionType type.
type LogsRatioConditionType string

// Condition type values.
const (
	LogsRatioConditionTypeMoreThan LogsRatioConditionType = "moreThan"
	LogsRatioConditionTypeLessThan LogsRatioConditionType = "lessThan"
)

// The rule to match the alert's conditions.
type LogsTimeRelativeRule struct {
	// The condition to match to.
	Condition LogsTimeRelativeCondition `json:"condition"`
	// +optional
	Override *AlertOverride `json:"override"`
}

// Logs time relative condition to match.
type LogsTimeRelativeCondition struct {
	// Threshold to match.
	Threshold resource.Quantity `json:"threshold"`

	// Comparison window.
	ComparedTo LogsTimeRelativeComparedTo `json:"comparedTo"`

	// How to compare.
	ConditionType LogsTimeRelativeConditionType `json:"conditionType"`
}

// +kubebuilder:validation:Enum=previousHour;sameHourYesterday;sameHourLastWeek;yesterday;sameDayLastWeek;sameDayLastMonth
// Comparison window type.
type LogsTimeRelativeComparedTo string

// Comparison window values.
const (
	LogsTimeRelativeComparedToPreviousHour      LogsTimeRelativeComparedTo = "previousHour"
	LogsTimeRelativeComparedToSameHourYesterday LogsTimeRelativeComparedTo = "sameHourYesterday"
	LogsTimeRelativeComparedToSameHourLastWeek  LogsTimeRelativeComparedTo = "sameHourLastWeek"
	LogsTimeRelativeComparedToYesterday         LogsTimeRelativeComparedTo = "yesterday"
	LogsTimeRelativeComparedToSameDayLastWeek   LogsTimeRelativeComparedTo = "sameDayLastWeek"
	LogsTimeRelativeComparedToSameDayLastMonth  LogsTimeRelativeComparedTo = "sameDayLastMonth"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
// ConditionType type.
type LogsTimeRelativeConditionType string

// Condition type values.
const (
	LogsTimeRelativeConditionTypeMoreThan LogsTimeRelativeConditionType = "moreThan"
	LogsTimeRelativeConditionTypeLessThan LogsTimeRelativeConditionType = "lessThan"
)

// Alerts are sent when the number of logs matching a filter is more than or less than a threshold over a specific time window.
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/logs/time-relative-alerts/
type LogsTimeRelativeThreshold struct {
	LogsFilter LogsFilter `json:"logsFilter"`
	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []LogsTimeRelativeRule `json:"rules"`

	//+kubebuilder:default=false
	// Ignore infinity on the threshold value.
	IgnoreInfinity bool `json:"ignoreInfinity"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`

	// How to work with undetected values.
	// +optional
	UndetectedValuesManagement *UndetectedValuesManagement `json:"undetectedValuesManagement"`
}

// Alerts for when a metric crosses a threshold.
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/metrics/threshold-alerts/
type MetricThreshold struct {
	// Filter for metrics
	MetricFilter MetricFilter `json:"metricFilter"`
	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []MetricThresholdRule `json:"rules"`

	MissingValues MetricMissingValues `json:"missingValues"`

	// How to work with undetected values.
	// +optional
	UndetectedValuesManagement *UndetectedValuesManagement `json:"undetectedValuesManagement,omitempty"`
}

// Filter for metrics
type MetricFilter struct {
	// PromQL query: https://coralogix.com/academy/mastering-metrics-in-coralogix/promql-fundamentals/
	Promql string `json:"promql,omitempty"`
}

// Rules that match the alert to the data.
type MetricThresholdRule struct {
	// Conditions to match for the rule.
	Condition MetricThresholdRuleCondition `json:"condition"`
	// Alert property overrides
	// +optional
	Override *AlertOverride `json:"override"`
}

// Conditions to match for the rule.
type MetricThresholdRuleCondition struct {
	Threshold resource.Quantity `json:"threshold"`
	// +kubebuilder:validation:Maximum:=100
	ForOverPct    uint32                       `json:"forOverPct"`
	OfTheLast     MetricTimeWindow             `json:"ofTheLast"`
	ConditionType MetricThresholdConditionType `json:"conditionType"`
}

// Time window type.
// +kubebuilder:validation:XValidation:rule="has(self.specificValue) != has(self.dynamicDuration)",message="Exactly one of specificValue or dynamicDuration is required"
type MetricTimeWindow struct {
	// +optional
	SpecificValue *MetricTimeWindowSpecificValue `json:"specificValue,omitempty"`
	// +optional
	// +kubebuilder:validation:Pattern:="^(0|(([0-9]+)y)?(([0-9]+)w)?(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?)$"
	DynamicDuration *string `json:"dynamicDuration,omitempty"`
}

// Time window type.
type MetricAnomalyTimeWindow struct {
	SpecificValue MetricTimeWindowSpecificValue `json:"specificValue"`
}

// +kubebuilder:validation:Enum={"1m","5m","10m","15m","20m","30m","1h","2h","4h","6h","12h","24h","36h"}
// Time window type.
type MetricTimeWindowSpecificValue string

// Time window values.
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

// +kubebuilder:validation:Enum=moreThan;lessThan;moreThanOrEquals;lessThanOrEquals
// ConditionType type.
type MetricThresholdConditionType string

// ConditionType type value.
const (
	MetricThresholdConditionTypeMoreThan         MetricThresholdConditionType = "moreThan"
	MetricThresholdConditionTypeLessThan         MetricThresholdConditionType = "lessThan"
	MetricThresholdConditionTypeMoreThanOrEquals MetricThresholdConditionType = "moreThanOrEquals"
	MetricThresholdConditionTypeLessThanOrEquals MetricThresholdConditionType = "lessThanOrEquals"
)

// Missing values strategies.
type MetricMissingValues struct {
	// +kubebuilder:default=false
	// Replace missing values with 0s
	ReplaceWithZero bool `json:"replaceWithZero,omitempty"`
	// +kubebuilder:validation:Maximum:=100
	// Replace with a number
	// +optional
	MinNonNullValuesPct *int64 `json:"minNonNullValuesPct,omitempty"`
}

// Tracing threshold alert
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/traces/tracing-alerts/
type TracingThreshold struct {
	// Filter the base collection.
	// +optional
	TracingFilter *TracingFilter `json:"tracingFilter,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []TracingThresholdRule `json:"rules"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
}

// Tracing immediate alert
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/traces/tracing-alerts/
type TracingImmediate struct {
	// +optional
	TracingFilter *TracingFilter `json:"tracingFilter,omitempty"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
}

// A simple tracing filter.
type TracingFilter struct {
	Simple *TracingSimpleFilter `json:"simple,omitempty"`
}

// Filter - values and operation.
type TracingFilterType struct {
	Values    []string                   `json:"values"`
	Operation TracingFilterOperationType `json:"operation"`
}

// +kubebuilder:validation:Enum=includes;endsWith;startsWith;isNot;is
// Tracing filter operations.
type TracingFilterOperationType string

// Tracing filter operation values.
const (
	TracingFilterOperationTypeIs         TracingFilterOperationType = "is"
	TracingFilterOperationTypeIncludes   TracingFilterOperationType = "includes"
	TracingFilterOperationTypeEndsWith   TracingFilterOperationType = "endsWith"
	TracingFilterOperationTypeStartsWith TracingFilterOperationType = "startsWith"
	TracingFilterOperationTypeIsNot      TracingFilterOperationType = "isNot"
)

// Simple tracing filter paired with a latency.
type TracingSimpleFilter struct {
	TracingLabelFilters *TracingLabelFilters `json:"tracingLabelFilters,omitempty"`
	LatencyThresholdMs  *uint64              `json:"latencyThresholdMs,omitempty"`
}

// Filter for traces.
type TracingLabelFilters struct {
	// +optional
	ApplicationName []TracingFilterType `json:"applicationName"`
	// +optional
	SubsystemName []TracingFilterType `json:"subsystemName"`
	// +optional
	ServiceName []TracingFilterType `json:"serviceName"`
	// +optional
	OperationName []TracingFilterType `json:"operationName"`
	// +optional
	SpanFields []TracingSpanFieldsFilterType `json:"spanFields"`
}

// Filter for spans
type TracingSpanFieldsFilterType struct {
	Key        string            `json:"key"`
	FilterType TracingFilterType `json:"filterType"`
}

// The rule to match the alert's conditions.
type TracingThresholdRule struct {
	// The condition to match to.
	Condition TracingThresholdRuleCondition `json:"condition"`
}

// Tracing Threshold condition.
type TracingThresholdRuleCondition struct {
	// Threshold amount.
	SpanAmount resource.Quantity `json:"spanAmount"`

	// Time window to evaluate.
	TimeWindow TracingTimeWindow `json:"timeWindow"`
}

// Tracing time window.
type TracingTimeWindow struct {
	SpecificValue TracingTimeWindowSpecificValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"5m","10m","15m","20m","30m","1h","2h","4h","6h","12h","24h","36h"}
// Time window type for tracing.
type TracingTimeWindowSpecificValue string

// Time window values.
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

// Alert to chain multiple alerts together.
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/flow-alerts/
type Flow struct {
	Stages []FlowStage `json:"stages"`
	// +kubebuilder:default=false
	EnforceSuppression bool `json:"enforceSuppression"`
}

// Stages to go through.
type FlowStage struct {
	// Type of stage.
	FlowStagesType FlowStagesType `json:"flowStagesType"`

	TimeframeMs int64 `json:"timeframeMs"`
	// Type of timeframe.
	TimeframeType FlowTimeframeType `json:"timeframeType"`
}

// Flow stage for the flow alert.
type FlowStagesType struct {
	Groups []FlowStageGroup `json:"groups"`
}

// Flow stage grouping.
type FlowStageGroup struct {
	// Alerts to group.
	AlertDefs []FlowStagesGroupsAlertDefs `json:"alertDefs"`

	// Link to the next alert.
	NextOp FlowStageGroupAlertsOp `json:"nextOp"`

	// Operation for the alert.
	AlertsOp FlowStageGroupAlertsOp `json:"alertsOp"`
}

// Alert references.
type FlowStagesGroupsAlertDefs struct {
	AlertRef AlertRef `json:"alertRef"`
	// +kubebuilder:default=false
	// Inversion.
	Not bool `json:"not"`
}

// Reference for an alert, backend or Kubernetes resource
// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type AlertRef struct {
	// Coralogix id reference.
	// +optional
	BackendRef *AlertBackendRef `json:"backendRef"`

	// Kubernetes resource reference.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef"`
}

// +kubebuilder:validation:Enum=and;or
// Flow stage operation type
type FlowStageGroupAlertsOp string

// Flow stage operation links.
const (
	FlowStageGroupAlertsOpAnd FlowStageGroupAlertsOp = "and"
	FlowStageGroupAlertsOpOr  FlowStageGroupAlertsOp = "or"
)

// +kubebuilder:validation:Enum=unspecified;upTo
// Type of timeframe
type FlowTimeframeType string

// Timeframe Type values.
const (
	TimeframeTypeUnspecified FlowTimeframeType = "unspecified"
	TimeframeTypeUpTo        FlowTimeframeType = "upTo"
)

// Logs anomaly alert
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/logs/anomaly-detection-alerts/
type LogsAnomaly struct {
	// Filter to filter the logs with.
	// +optional
	LogsFilter *LogsFilter `json:"logsFilter,omitempty"`

	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []LogsAnomalyRule `json:"rules"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
}

// The rule to match the alert's conditions.
type LogsAnomalyRule struct {
	// Condition to match to.
	Condition LogsAnomalyCondition `json:"condition"`
}

// Condition for the logs anomaly alert.
type LogsAnomalyCondition struct {
	//+kubebuilder:default=0
	// Minimum value
	MinimumThreshold resource.Quantity `json:"minimumThreshold"`
	// Time window to evaluate.
	TimeWindow LogsTimeWindow `json:"timeWindow"`
}

// Metric anomaly alert
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/metrics/anomaly-detection-alerts/
type MetricAnomaly struct {
	// PromQL filter for metrics
	MetricFilter MetricFilter `json:"metricFilter"`

	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []MetricAnomalyRule `json:"rules"`
}

// Condition to match to.
type MetricAnomalyCondition struct {
	// Threshold to clear.
	Threshold resource.Quantity `json:"threshold"`

	// +kubebuilder:validation:Maximum:=100
	// Percentage for the threshold
	ForOverPct int64 `json:"forOverPct"`

	// Time window to match within
	OfTheLast MetricAnomalyTimeWindow `json:"ofTheLast"`
	// +kubebuilder:validation:Maximum:=100
	// Replace with a number
	MinNonNullValuesPct int64 `json:"minNonNullValuesPct"`
	// Condition type.
	ConditionType MetricAnomalyConditionType `json:"conditionType"`
}

// +kubebuilder:validation:Enum=moreThanUsual;lessThanUsual
// ConditionType type.
type MetricAnomalyConditionType string

// Condition type values.
const (
	MetricAnomalyConditionTypeMoreThanUsual MetricAnomalyConditionType = "moreThanUsual"
	MetricAnomalyConditionTypeLessThanUsual MetricAnomalyConditionType = "lessThanUsual"
)

// The rule to match the alert's conditions.
type MetricAnomalyRule struct {
	// Condition to match to.
	Condition MetricAnomalyCondition `json:"condition"`
}

// Alert for when a new value is logged
// Read more at https://coralogix.com/docs/user-guides/alerting/create-an-alert/logs/new-value-alerts/
type LogsNewValue struct {
	// Filter to filter the logs with.
	LogsFilter *LogsFilter `json:"logsFilter"`

	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []LogsNewValueRule `json:"rules"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
}

// The rule to match the alert's conditions.
type LogsNewValueRule struct {
	// Condition to match to
	Condition LogsNewValueRuleCondition `json:"condition"`
}

// Condition to match.
type LogsNewValueRuleCondition struct {
	// Where to look
	KeypathToTrack string `json:"keypathToTrack"`
	// Which time window.
	TimeWindow LogsNewValueTimeWindow `json:"timeWindow"`
}

// New values time window.
type LogsNewValueTimeWindow struct {
	SpecificValue LogsNewValueTimeWindowSpecificValue `json:"specificValue,omitempty"`
}

// +kubebuilder:validation:Enum={"12h","24h","48h","72h","1w","1mo","2mo","3mo"}
// Time windows.
type LogsNewValueTimeWindowSpecificValue string

// Time windows values.
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
	// Filter to filter the logs with.
	LogsFilter *LogsFilter `json:"logsFilter"`

	// +kubebuilder:validation:MinItems=1
	// Rules that match the alert to the data.
	Rules []LogsUniqueCountRule `json:"rules"`

	// Filter for the notification payload.
	// +optional
	NotificationPayloadFilter []string `json:"notificationPayloadFilter"`
	// +optional
	MaxUniqueCountPerGroupByKey *uint64 `json:"maxUniqueCountPerGroupByKey"`
	UniqueCountKeypath          string  `json:"uniqueCountKeypath"`
}

// Condition for the logs unique count alerts.
type LogsUniqueCountCondition struct {
	// Threshold to cross
	Threshold int64 `json:"threshold"`

	// Time window to evaluate.
	TimeWindow LogsUniqueCountTimeWindow `json:"timeWindow"`
}

// Time window.
type LogsUniqueCountTimeWindow struct {
	SpecificValue LogsUniqueCountTimeWindowSpecificValue `json:"specificValue"`
}

// +kubebuilder:validation:Enum={"1m","5m","10m","15m","20m","30m","1h","2h","4h","6h","12h","24h","36h"}
// Time windows for Logs Unique Count
type LogsUniqueCountTimeWindowSpecificValue string

// Time window values.
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

// The rule to match the alert's conditions.
type LogsUniqueCountRule struct {
	// Condition to match to.
	Condition LogsUniqueCountCondition `json:"condition"`
}

// A filter for logs.
type LogsFilter struct {
	// Simple lucene filter.
	SimpleFilter LogsSimpleFilter `json:"simpleFilter,omitempty"`
}

// Simple lucene filter.
type LogsSimpleFilter struct {
	// The query.
	// +optional
	LuceneQuery *string `json:"luceneQuery,omitempty"`

	// Filter for labels.
	// +optional
	LabelFilters *LabelFilters `json:"labelFilters,omitempty"`
}

// Filters for labels.
type LabelFilters struct {
	// Application name to filter for.
	// +optional
	ApplicationName []LabelFilterType `json:"applicationName"`
	// Subsystem name to filter for.
	// +optional
	SubsystemName []LabelFilterType `json:"subsystemName"`
	// Severity to filter for.
	// +optional
	Severity []LogSeverity `json:"severity"`
}

// Label filter specifications
type LabelFilterType struct {
	// The value
	//+kubebuilder:validation:MinLength=0
	Value string `json:"value"`

	//+kubebuilder:default=is
	// Operation to apply.
	Operation LogFilterOperationType `json:"operation"`
}

// How to work with undetected values.
// Read more here: https://coralogix.com/docs/user-guides/alerting/create-an-alert/metrics/threshold-alerts/#manage-undetected-values
type UndetectedValuesManagement struct {

	//+kubebuilder:default=false
	// Deactivate triggering the alert on undetected values.
	TriggerUndetectedValues bool `json:"triggerUndetectedValues"`

	//+kubebuilder:default=never
	// Automatically retire the alerts after this time.
	AutoRetireTimeframe AutoRetireTimeframe `json:"autoRetireTimeframe"`
}

// +kubebuilder:validation:XValidation:rule="has(self.errorBudget) != has(self.burnRate)",message="Exactly one of errorBudget or burnRate is required"
type SloThreshold struct {
	SloDefinition SloDefinition `json:"sloDefinition"`

	// +optional
	ErrorBudget *ErrorBudget `json:"errorBudget,omitempty"`

	// +optional
	BurnRate *BurnRate `json:"burnRate,omitempty"`
}

type SloDefinition struct {
	SloRef SloRef `json:"sloRef"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type SloRef struct {
	// +optional
	BackendRef *SloBackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.id) != has(self.name)",message="Exactly one of id or name must be set"
type SloBackendRef struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Name *string `json:"name,omitempty"`
}

type ErrorBudget struct {
	Rules []SloThresholdRule `json:"rules"`
}

type SloThresholdRule struct {
	// Condition to match
	Condition SloThresholdRuleCondition `json:"condition"`
	// Alert overrides.
	// +optional
	Override *AlertOverride `json:"override"`
}

type SloThresholdRuleCondition struct {
	// Threshold to match to.
	Threshold resource.Quantity `json:"threshold"`
}

type BurnRate struct {
	Rules        []BurnRateRule  `json:"rules"`
	BurnRateType SloBurnRateType `json:"type"`
}

// +kubebuilder:validation:XValidation:rule="has(self.single) != has(self.dual)",message="Exactly one of single or dual must be set"
type SloBurnRateType struct {
	// +optional
	Single *SloBurnRateTypeSingle `json:"single"`
	// +optional
	Dual *SloBurnRateTypeDual `json:"dual"`
}

type SloBurnRateTypeSingle struct {
	TimeDuration TimeDuration `json:"timeDuration"`
}

type SloBurnRateTypeDual struct {
	TimeDuration TimeDuration `json:"timeDuration"`
}

type TimeDuration struct {
	Duration int              `json:"duration"`
	Unit     TimeDurationUnit `json:"unit"`
}

type BurnRateRule struct {
	// Condition to match
	Condition BurnRateRuleCondition `json:"condition"`
	// Alert overrides.
	// +optional
	Override *AlertOverride `json:"override"`
}

type BurnRateRuleCondition struct {
	Threshold resource.Quantity `json:"threshold"`
}

// +kubebuilder:validation:Enum=is;includes;endsWith;startsWith
// Operation type for log filters.
type LogFilterOperationType string

// Operation type for log filter values.
const (
	LogFilterOperationTypeIs         LogFilterOperationType = "is"
	LogFilterOperationTypeIncludes   LogFilterOperationType = "includes"
	LogFilterOperationTypeEndWith    LogFilterOperationType = "endsWith"
	LogFilterOperationTypeStartsWith LogFilterOperationType = "startsWith"
)

// +kubebuilder:validation:Enum=unspecified;hours
// Time duration unit for a Burn Rate Slo.
type TimeDurationUnit string

// Operation type for log filter values.
const (
	TimeDurationUnitUnspecified TimeDurationUnit = "unspecified"
	TimeDurationUnitHours       TimeDurationUnit = "hours"
)

// +kubebuilder:validation:Enum=debug;info;warning;error;critical;verbose
// How severe a log is.
type LogSeverity string

// Severity values.
const (
	LogSeverityDebug    LogSeverity = "debug"
	LogSeverityInfo     LogSeverity = "info"
	LogSeverityWarning  LogSeverity = "warning"
	LogSeverityError    LogSeverity = "error"
	LogSeverityCritical LogSeverity = "critical"
	LogSeverityVerbose  LogSeverity = "verbose"
)

// +kubebuilder:validation:Enum=p1;p2;p3;p4;p5
// Alert priorities.
type AlertPriority string

// Priority values.
const (
	AlertPriorityP1 AlertPriority = "p1"
	AlertPriorityP2 AlertPriority = "p2"
	AlertPriorityP3 AlertPriority = "p3"
	AlertPriorityP4 AlertPriority = "p4"
	AlertPriorityP5 AlertPriority = "p5"
)

func NewDefaultAlertStatus() *AlertStatus {
	return &AlertStatus{
		ID: ptr.To(""),
	}
}

func (in *AlertSpec) ExtractAlertDefProperties(listingAlertsAndWebhooksProperties *GetResourceRefProperties) (*alerts.AlertDefProperties, error) {
	notificationGroup, err := expandNotificationGroup(in.NotificationGroup, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand notification group: %w", err)
	}

	notificationGroupExcess, err := expandNotificationGroupExcess(in.NotificationGroupExcess, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand notification group excess: %w", err)
	}

	priority := AlertPriorityToOpenAPIPriority[in.Priority]

	if logsImmediate := in.TypeDefinition.LogsImmediate; logsImmediate != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesLogsImmediate: &alerts.AlertDefPropertiesLogsImmediate{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_LOGS_IMMEDIATE_OR_UNSPECIFIED.Ptr(),
				LogsImmediate:           expandLogsImmediate(logsImmediate),
			},
		}, nil
	} else if logsThreshold := in.TypeDefinition.LogsThreshold; logsThreshold != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesLogsThreshold: &alerts.AlertDefPropertiesLogsThreshold{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_LOGS_THRESHOLD.Ptr(),
				LogsThreshold:           expandLogsThreshold(logsThreshold, priority),
			},
		}, nil
	} else if logsRatioThreshold := in.TypeDefinition.LogsRatioThreshold; logsRatioThreshold != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesLogsRatioThreshold: &alerts.AlertDefPropertiesLogsRatioThreshold{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_LOGS_RATIO_THRESHOLD.Ptr(),
				LogsRatioThreshold:      expandLogsRatioThreshold(logsRatioThreshold, priority),
			},
		}, nil
	} else if logsTimeRelativeThreshold := in.TypeDefinition.LogsTimeRelativeThreshold; logsTimeRelativeThreshold != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesLogsTimeRelativeThreshold: &alerts.AlertDefPropertiesLogsTimeRelativeThreshold{
				Name:                      alerts.PtrString(in.Name),
				Description:               alerts.PtrString(in.Description),
				Enabled:                   alerts.PtrBool(in.Enabled),
				Priority:                  priority.Ptr(),
				GroupByKeys:               in.GroupByKeys,
				IncidentsSettings:         expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:         notificationGroup,
				NotificationGroupExcess:   notificationGroupExcess,
				EntityLabels:              ptr.To(in.EntityLabels),
				PhantomMode:               alerts.PtrBool(in.PhantomMode),
				ActiveOn:                  expandAlertSchedule(in.Schedule),
				Type:                      alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_LOGS_TIME_RELATIVE_THRESHOLD.Ptr(),
				LogsTimeRelativeThreshold: expandLogsTimeRelativeThreshold(logsTimeRelativeThreshold, priority),
			},
		}, nil
	} else if metricThreshold := in.TypeDefinition.MetricThreshold; metricThreshold != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesMetricThreshold: &alerts.AlertDefPropertiesMetricThreshold{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_METRIC_THRESHOLD.Ptr(),
				MetricThreshold:         expandMetricThreshold(metricThreshold, priority),
			},
		}, nil
	} else if tracingThreshold := in.TypeDefinition.TracingThreshold; tracingThreshold != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesTracingThreshold: &alerts.AlertDefPropertiesTracingThreshold{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_TRACING_THRESHOLD.Ptr(),
				TracingThreshold:        expandTracingThreshold(tracingThreshold),
			},
		}, nil
	} else if tracingImmediate := in.TypeDefinition.TracingImmediate; tracingImmediate != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesTracingImmediate: &alerts.AlertDefPropertiesTracingImmediate{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_TRACING_IMMEDIATE.Ptr(),
				TracingImmediate:        expandTracingImmediate(tracingImmediate),
			},
		}, nil
	} else if flow := in.TypeDefinition.Flow; flow != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesFlow: &alerts.AlertDefPropertiesFlow{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_FLOW.Ptr(),
				Flow:                    expandFlow(listingAlertsAndWebhooksProperties, flow),
			},
		}, nil
	} else if logsAnomaly := in.TypeDefinition.LogsAnomaly; logsAnomaly != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesLogsAnomaly: &alerts.AlertDefPropertiesLogsAnomaly{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_LOGS_ANOMALY.Ptr(),
				LogsAnomaly:             expandLogsAnomaly(logsAnomaly),
			},
		}, nil
	} else if metricAnomaly := in.TypeDefinition.MetricAnomaly; metricAnomaly != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesMetricAnomaly: &alerts.AlertDefPropertiesMetricAnomaly{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_METRIC_ANOMALY.Ptr(),
				MetricAnomaly:           expandMetricAnomaly(metricAnomaly),
			},
		}, nil
	} else if logsNewValue := in.TypeDefinition.LogsNewValue; logsNewValue != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesLogsNewValue: &alerts.AlertDefPropertiesLogsNewValue{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_LOGS_NEW_VALUE.Ptr(),
				LogsNewValue:            expandLogsNewValue(logsNewValue),
			},
		}, nil
	} else if logsUniqueCount := in.TypeDefinition.LogsUniqueCount; logsUniqueCount != nil {
		return &alerts.AlertDefProperties{
			AlertDefPropertiesLogsUniqueCount: &alerts.AlertDefPropertiesLogsUniqueCount{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_LOGS_UNIQUE_COUNT.Ptr(),
				LogsUniqueCount:         expandLogsUniqueCount(logsUniqueCount),
			},
		}, nil
	} else if sloThreshold := in.TypeDefinition.SloThreshold; sloThreshold != nil {
		sloThresholdType, err := expandSloThreshold(listingAlertsAndWebhooksProperties, sloThreshold)
		if err != nil {
			return nil, fmt.Errorf("failed to expand SLO threshold: %w", err)
		}
		return &alerts.AlertDefProperties{
			AlertDefPropertiesSloThreshold: &alerts.AlertDefPropertiesSloThreshold{
				Name:                    alerts.PtrString(in.Name),
				Description:             alerts.PtrString(in.Description),
				Enabled:                 alerts.PtrBool(in.Enabled),
				Priority:                priority.Ptr(),
				GroupByKeys:             in.GroupByKeys,
				IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
				NotificationGroup:       notificationGroup,
				NotificationGroupExcess: notificationGroupExcess,
				EntityLabels:            ptr.To(in.EntityLabels),
				PhantomMode:             alerts.PtrBool(in.PhantomMode),
				ActiveOn:                expandAlertSchedule(in.Schedule),
				Type:                    alerts.ALERTDEFTYPE_ALERT_DEF_TYPE_SLO_THRESHOLD.Ptr(),
				SloThreshold:            sloThresholdType,
			},
		}, nil
	}

	return nil, fmt.Errorf("unsupported alert type definition")
}

func expandIncidentsSettings(incidentsSettings *IncidentsSettings) *alerts.AlertDefIncidentSettings {
	if incidentsSettings == nil {
		return nil
	}

	alertDefIncidentSettings := &alerts.AlertDefIncidentSettings{
		NotifyOn: NotifyOnToOpenAPINotifyOn[incidentsSettings.NotifyOn].Ptr(),
	}

	if incidentsSettings.RetriggeringPeriod.Minutes != nil {
		alertDefIncidentSettings.Minutes = incidentsSettings.RetriggeringPeriod.Minutes
	}

	return alertDefIncidentSettings
}

func expandNotificationGroupExcess(excess []NotificationGroup, listingAlertsAndWebhooksProperties *GetResourceRefProperties) ([]alerts.AlertDefNotificationGroup, error) {
	result := make([]alerts.AlertDefNotificationGroup, 0, len(excess))
	var errs error
	for _, group := range excess {
		ng, err := expandNotificationGroup(&group, listingAlertsAndWebhooksProperties)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to expand notification group: %w", err))
			continue
		}
		result = append(result, *ng)
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func expandNotificationGroup(notificationGroup *NotificationGroup, listingAlertsAndWebhooksProperties *GetResourceRefProperties) (*alerts.AlertDefNotificationGroup, error) {
	if notificationGroup == nil {
		return nil, nil
	}

	webhooks, err := expandWebhooksSettings(notificationGroup.Webhooks, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand webhooks settings: %w", err)
	}

	var destinations []alerts.NotificationDestination
	var router *alerts.NotificationRouter
	if notificationGroup.Destinations != nil {
		destinations, err = expandNotificationDestinations(notificationGroup.Destinations, listingAlertsAndWebhooksProperties)
		if err != nil {
			return nil, fmt.Errorf("failed to expand notification destinations: %w", err)
		}
	} else if notificationGroup.Router != nil {
		notifyOn := NotifyOnToOpenAPINotifyOn[notificationGroup.Router.NotifyOn]
		router = &alerts.NotificationRouter{
			Id:       alerts.PtrString("router_default"),
			NotifyOn: notifyOn.Ptr(),
		}
	}

	return &alerts.AlertDefNotificationGroup{
		GroupByKeys:  notificationGroup.GroupByKeys,
		Webhooks:     webhooks,
		Destinations: destinations,
		Router:       router,
	}, nil
}

func expandWebhooksSettings(webhooksSettings []WebhookSettings, listingAlertsAndWebhooksProperties *GetResourceRefProperties) ([]alerts.AlertDefWebhooksSettings, error) {
	result := make([]alerts.AlertDefWebhooksSettings, len(webhooksSettings))
	var errs error
	for i, setting := range webhooksSettings {
		expandedWebhookSetting, err := expandWebhookSetting(setting, listingAlertsAndWebhooksProperties)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to expand webhook setting: %w", err))
			continue
		}
		result[i] = *expandedWebhookSetting
	}

	if errs != nil {
		return nil, errs
	}
	return result, nil
}

func expandWebhookSetting(webhooksSetting WebhookSettings, listingAlertsAndWebhooksProperties *GetResourceRefProperties) (*alerts.AlertDefWebhooksSettings, error) {
	notifyOn := NotifyOnToOpenAPINotifyOn[webhooksSetting.NotifyOn]
	integration, err := expandIntegration(webhooksSetting.Integration, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand integration: %w", err)
	}
	return &alerts.AlertDefWebhooksSettings{
		NotifyOn:    notifyOn.Ptr(),
		Integration: integration,
		Minutes:     webhooksSetting.RetriggeringPeriod.Minutes,
	}, nil
}

func expandIntegration(integration IntegrationType, listingWebhooksProperties *GetResourceRefProperties) (*alerts.V3IntegrationType, error) {
	if integrationRef := integration.IntegrationRef; integrationRef != nil {
		var integrationID *int64
		var err error

		if resourceRef := integrationRef.ResourceRef; resourceRef != nil {
			if namespace := resourceRef.Namespace; namespace != nil {
				listingWebhooksProperties.Namespace = *namespace
			}
			integrationID, err = convertCRNameToIntegrationID(resourceRef.Name, listingWebhooksProperties)
			if err != nil {
				return nil, fmt.Errorf("failed to convert CR name to integration ID: %w", err)
			}
		} else if backendRef := integrationRef.BackendRef; backendRef != nil {
			if id := backendRef.ID; id != nil {
				integrationID = id
			} else if name := backendRef.Name; name != nil {
				integrationID, err = convertNameToIntegrationID(*name, listingWebhooksProperties)
				if err != nil {
					return nil, fmt.Errorf("failed to convert name to integration ID: %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("integration type not found")
		}

		return &alerts.V3IntegrationType{
			V3IntegrationTypeIntegrationId: &alerts.V3IntegrationTypeIntegrationId{
				IntegrationId: integrationID,
			},
		}, nil
	} else if recipients := integration.Recipients; recipients != nil {
		return &alerts.V3IntegrationType{
			V3IntegrationTypeRecipients: &alerts.V3IntegrationTypeRecipients{
				Recipients: &alerts.Recipients{
					Emails: recipients,
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("integration type not found")
}

func convertNameToIntegrationID(name string, properties *GetResourceRefProperties) (*int64, error) {
	if properties.WebhookNameToId == nil {
		if err := fillWebhookNameToId(properties); err != nil {
			return nil, err
		}
	}

	id, ok := properties.WebhookNameToId[name]
	if !ok {
		return nil, fmt.Errorf("webhook %s not found", name)
	}

	return &id, nil
}

func fillWebhookNameToId(properties *GetResourceRefProperties) error {
	log, client, ctx := properties.Log, properties.ClientSet.Webhooks(), properties.Ctx
	log.V(1).Info("Listing webhooks from the backend")
	webhooks, httpResp, err := client.OutgoingWebhooksServiceListAllOutgoingWebhooks(ctx).Execute()
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", oapicxsdk.NewAPIError(httpResp, err))
	}

	properties.WebhookNameToId = make(map[string]int64)
	for _, webhook := range webhooks.Deployed {
		if webhook.Name == nil || webhook.ExternalId == nil {
			continue
		}
		properties.WebhookNameToId[*webhook.Name] = *webhook.ExternalId
	}

	return nil
}

func expandNotificationDestinations(destinations []NotificationDestination, properties *GetResourceRefProperties) ([]alerts.NotificationDestination, error) {
	var result []alerts.NotificationDestination
	for _, destination := range destinations {
		connectorId, err := getResourceID(destination.Connector, properties, utils.ConnectorKind)
		if err != nil {
			return nil, fmt.Errorf("failed to expand connector ID: %w", err)
		}

		var presetId *string
		if destination.Preset != nil {
			id, err := getResourceID(*destination.Preset, properties, utils.PresetKind)
			if err != nil {
				return nil, fmt.Errorf("failed to expand preset ID: %w", err)
			}

			presetId = &id
		}

		triggeredRoutingOverrides := expandRoutingOverrides(destination.TriggeredRoutingOverrides)
		var resolvedRoutingOverrides *alerts.V3SourceOverrides
		if destination.ResolvedRoutingOverrides != nil {
			resolvedRoutingOverrides = expandRoutingOverrides(*destination.ResolvedRoutingOverrides)
		}

		notificationDestination := alerts.NotificationDestination{
			ConnectorId: alerts.PtrString(connectorId),
			PresetId:    presetId,
			NotifyOn:    NotifyOnToOpenAPINotifyOn[destination.NotifyOn].Ptr(),
			TriggeredRoutingOverrides: &alerts.NotificationRouting{
				ConfigOverrides: triggeredRoutingOverrides,
			},
			ResolvedRouteOverrides: &alerts.NotificationRouting{
				ConfigOverrides: resolvedRoutingOverrides,
			},
		}

		result = append(result, notificationDestination)
	}

	return result, nil
}

func expandRoutingOverrides(overrides NotificationRouting) *alerts.V3SourceOverrides {
	if overrides.ConfigOverrides == nil {
		return nil
	}

	connectorOverrides := extractConnectorOverrides(overrides.ConfigOverrides.ConnectorConfigFields)
	presetOverrides := extractPresetOverrides(overrides.ConfigOverrides.MessageConfigFields)

	sourceOverrides := &alerts.V3SourceOverrides{
		ConnectorConfigFields: connectorOverrides,
		MessageConfigFields:   presetOverrides,
		PayloadType:           alerts.PtrString(overrides.ConfigOverrides.PayloadType),
	}

	return sourceOverrides
}

func extractConnectorOverrides(overrides []ConfigField) []alerts.V3ConnectorConfigField {
	var result []alerts.V3ConnectorConfigField
	for _, override := range overrides {
		result = append(result, alerts.V3ConnectorConfigField{
			FieldName: alerts.PtrString(override.FieldName),
			Template:  alerts.PtrString(override.Template),
		})
	}

	return result
}

func extractPresetOverrides(overrides []ConfigField) []alerts.V3MessageConfigField {
	var result []alerts.V3MessageConfigField
	for _, override := range overrides {
		result = append(result, alerts.V3MessageConfigField{
			FieldName: alerts.PtrString(override.FieldName),
			Template:  alerts.PtrString(override.Template),
		})
	}

	return result
}

func getResourceID(ref NCRef, properties *GetResourceRefProperties, kind string) (string, error) {
	if ref.BackendRef != nil {
		return ref.BackendRef.ID, nil
	}
	if ref.ResourceRef != nil {
		return extractIdFromResourceRef(ref.ResourceRef, properties, kind)
	}

	return "", fmt.Errorf("resource reference should have either backendRef or resourceRef")
}

func extractIdFromResourceRef(ref *ResourceRef, properties *GetResourceRefProperties, kind string) (string, error) {
	ctx, namespace := properties.Ctx, properties.Namespace
	if ref.Namespace != nil {
		namespace = *ref.Namespace
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   utils.CoralogixAPIGroup,
		Kind:    kind,
		Version: utils.V1alpha1APIVersion,
	})

	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: namespace}, u); err != nil {
		return "", fmt.Errorf("failed to get resource: %w", err)
	}

	if !config.GetConfig().Selector.Matches(u.GetLabels(), u.GetNamespace()) {
		return "", fmt.Errorf("resource %s does not match selector", u.GetName())
	}

	id, found, err := unstructured.NestedString(u.Object, "status", "id")
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("resource %s does not have an ID populated", u.GetName())
	}

	return id, nil
}

func expandAlertSchedule(alertSchedule *AlertSchedule) *alerts.ActivitySchedule {
	if alertSchedule == nil {
		return nil
	}

	utc := extractUTC(alertSchedule.TimeZone)
	daysOfWeek := expandDaysOfWeek(alertSchedule.ActiveOn.DayOfWeek)
	start := expandTime(alertSchedule.ActiveOn.StartTime)
	end := expandTime(alertSchedule.ActiveOn.EndTime)

	start, end, daysOfWeek = convertTimeFramesToGMT(start, end, daysOfWeek, utc)

	return &alerts.ActivitySchedule{
		DayOfWeek: daysOfWeek,
		StartTime: start,
		EndTime:   end,
	}
}

func extractUTC(timeZone TimeZone) int32 {
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

func expandTime(time *TimeOfDay) *alerts.TimeOfDay {
	if time == nil {
		return nil
	}

	timeArr := strings.Split(string(*time), ":")
	hours, _ := strconv.Atoi(timeArr[0])
	minutes, _ := strconv.Atoi(timeArr[1])

	return &alerts.TimeOfDay{
		Hours:   alerts.PtrInt32(int32(hours)),
		Minutes: alerts.PtrInt32(int32(minutes)),
	}
}

func convertTimeFramesToGMT(start, end *alerts.TimeOfDay, daysOfWeek []alerts.DayOfWeek, utc int32) (*alerts.TimeOfDay, *alerts.TimeOfDay, []alerts.DayOfWeek) {
	var dayToIndex = map[alerts.DayOfWeek]int{
		alerts.DAYOFWEEK_DAY_OF_WEEK_MONDAY_OR_UNSPECIFIED: 0,
		alerts.DAYOFWEEK_DAY_OF_WEEK_TUESDAY:               1,
		alerts.DAYOFWEEK_DAY_OF_WEEK_WEDNESDAY:             2,
		alerts.DAYOFWEEK_DAY_OF_WEEK_THURSDAY:              3,
		alerts.DAYOFWEEK_DAY_OF_WEEK_FRIDAY:                4,
		alerts.DAYOFWEEK_DAY_OF_WEEK_SATURDAY:              5,
		alerts.DAYOFWEEK_DAY_OF_WEEK_SUNDAY:                6,
	}

	var indexToDay = []alerts.DayOfWeek{
		alerts.DAYOFWEEK_DAY_OF_WEEK_MONDAY_OR_UNSPECIFIED,
		alerts.DAYOFWEEK_DAY_OF_WEEK_TUESDAY,
		alerts.DAYOFWEEK_DAY_OF_WEEK_WEDNESDAY,
		alerts.DAYOFWEEK_DAY_OF_WEEK_THURSDAY,
		alerts.DAYOFWEEK_DAY_OF_WEEK_FRIDAY,
		alerts.DAYOFWEEK_DAY_OF_WEEK_SATURDAY,
		alerts.DAYOFWEEK_DAY_OF_WEEK_SUNDAY,
	}
	daysOfWeekOffset := daysOfWeekOffsetToGMT(start, utc)
	start.Hours = alerts.PtrInt32(convertUtcToGmt(start.GetHours(), utc))
	end.Hours = alerts.PtrInt32(convertUtcToGmt(end.GetHours(), utc))
	if daysOfWeekOffset != 0 {
		for i, d := range daysOfWeek {
			idx := (dayToIndex[d] + int(daysOfWeekOffset) + 7) % 7
			daysOfWeek[i] = indexToDay[idx]
		}
	}

	return start, end, daysOfWeek
}

func daysOfWeekOffsetToGMT(start *alerts.TimeOfDay, utc int32) int32 {
	daysOfWeekOffset := (*start.Hours - utc) / 24
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

func expandDaysOfWeek(week []DayOfWeek) []alerts.DayOfWeek {
	result := make([]alerts.DayOfWeek, len(week))
	for i, d := range week {
		result[i] = DaysOfWeekToOpenAPIDayOfWeek[d]
	}

	return result
}

func expandSloThreshold(listingSloProperties *GetResourceRefProperties, sloThreshold *SloThreshold) (*alerts.SloThresholdType, error) {
	sloId, err := getSloId(listingSloProperties, sloThreshold.SloDefinition.SloRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get SLO ID: %w", err)
	}
	return expandSloThresholdType(sloId, sloThreshold)
}

func getSloId(listingSloProperties *GetResourceRefProperties, sloRef SloRef) (string, error) {
	if backendRef := sloRef.BackendRef; backendRef != nil {
		if backendRef.ID != nil {
			return *backendRef.ID, nil
		} else if name := backendRef.Name; name != nil {
			return convertSloBackendNameToId(listingSloProperties, name)
		}
		return "", fmt.Errorf("SLO backend reference must have either ID or Name")
	} else if resourceRef := sloRef.ResourceRef; resourceRef != nil {
		if namespace := resourceRef.Namespace; namespace != nil {
			listingSloProperties.Namespace = *namespace
		}
		return convertSloCrNameToID(listingSloProperties, resourceRef.Name)
	}
	return "", fmt.Errorf("SLO reference must have either backendRef or resourceRef")
}

func convertSloBackendNameToId(listingSloProperties *GetResourceRefProperties, name *string) (string, error) {
	listingSloProperties.Log.V(1).Info("Listing SLOs from the backend")
	filters := slos.SloFilters{
		Filters: []slos.SloFilter{
			{
				Field: slos.SloFilterField{
					SloFilterFieldConstFilter: &slos.SloFilterFieldConstFilter{
						ConstFilter: slos.SLOCONSTANTFILTERFIELD_SLO_CONST_FILTER_FIELD_SLO_NAME.Ptr(),
					},
				},
				Predicate: slos.SloFilterPredicate{
					Is: &slos.IsFilterPredicate{
						Is: []string{*name},
					},
				},
			},
		},
	}
	listResp, httpResp, err := listingSloProperties.ClientSet.SLOs().
		SlosServiceListSlos(listingSloProperties.Ctx).
		Filters(filters).
		Execute()
	if err != nil {
		return "", fmt.Errorf("failed to list SLOs: %w", oapicxsdk.NewAPIError(httpResp, err))
	}
	for _, slo := range listResp.Slos {
		switch {
		case slo.SloWindowBasedMetricSli != nil:
			if slo.SloWindowBasedMetricSli.Name == *name {
				if slo.SloWindowBasedMetricSli.Id != nil {
					return *slo.SloWindowBasedMetricSli.Id, nil
				}
			}
		case slo.SloRequestBasedMetricSli != nil:
			if slo.SloRequestBasedMetricSli.Name == *name {
				if slo.SloRequestBasedMetricSli.Id != nil {
					return *slo.SloRequestBasedMetricSli.Id, nil
				}
			}
		}
	}
	return "", fmt.Errorf("SLO with name %s not found", *name)
}

func convertSloCrNameToID(listingSloProperties *GetResourceRefProperties, sloCrName string) (string, error) {
	ctx, namespace := listingSloProperties.Ctx, listingSloProperties.Namespace
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   utils.CoralogixAPIGroup,
		Kind:    "SLO",
		Version: utils.V1alpha1APIVersion,
	})

	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: sloCrName, Namespace: namespace}, u); err != nil {
		return "", fmt.Errorf("failed to get slo, name: %s, namespace: %s, error: %w", sloCrName, namespace, err)
	}

	if !config.GetConfig().Selector.Matches(u.GetLabels(), u.GetNamespace()) {
		return "", fmt.Errorf("slo %s does not match selector", u.GetName())
	}

	id, found, err := unstructured.NestedString(u.Object, "status", "id")
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("status.id not found")
	}

	return id, nil
}

func expandSloThresholdType(sloId string, sloThreshold *SloThreshold) (*alerts.SloThresholdType, error) {
	if errorBudget := sloThreshold.ErrorBudget; errorBudget != nil {
		return &alerts.SloThresholdType{
			SloThresholdTypeErrorBudget: &alerts.SloThresholdTypeErrorBudget{
				ErrorBudget: &alerts.ErrorBudgetThreshold{
					Rules: expandSloErrorBudgetRules(errorBudget.Rules),
				},
				SloDefinition: &alerts.V3SloDefinition{
					SloId: alerts.PtrString(sloId),
				},
			},
		}, nil
	} else if burnRate := sloThreshold.BurnRate; burnRate != nil {
		return &alerts.SloThresholdType{
			SloThresholdTypeBurnRate: &alerts.SloThresholdTypeBurnRate{
				BurnRate: expandSloBurnRate(*burnRate),
				SloDefinition: &alerts.V3SloDefinition{
					SloId: alerts.PtrString(sloId),
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("unsupported SLO threshold type")
}

func expandSloErrorBudgetRules(rules []SloThresholdRule) []alerts.SloThresholdRule {
	result := make([]alerts.SloThresholdRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, *expandSloThresholdRule(rule))
	}
	return result
}

func expandSloThresholdRule(rule SloThresholdRule) *alerts.SloThresholdRule {
	return &alerts.SloThresholdRule{
		Condition: expandSloThresholdRuleCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override, alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P1),
	}
}

func expandSloThresholdRuleCondition(condition SloThresholdRuleCondition) *alerts.SloThresholdCondition {
	return &alerts.SloThresholdCondition{
		Threshold: alerts.PtrFloat64(condition.Threshold.AsApproximateFloat64()),
	}
}

func expandSloBurnRate(burnRate BurnRate) *alerts.BurnRateThreshold {
	if burnRate.BurnRateType.Single != nil {
		duration := strconv.Itoa(burnRate.BurnRateType.Single.TimeDuration.Duration)
		return &alerts.BurnRateThreshold{
			BurnRateThresholdSingle: &alerts.BurnRateThresholdSingle{
				Rules: expandSloBurnRateRules(burnRate.Rules),
				Single: &alerts.BurnRateTypeSingle{
					TimeDuration: &alerts.TimeDuration{
						Unit:     expandDurationUnit(burnRate.BurnRateType.Single.TimeDuration.Unit),
						Duration: &duration,
					},
				},
			},
		}
	}
	duration := strconv.Itoa(burnRate.BurnRateType.Dual.TimeDuration.Duration)
	return &alerts.BurnRateThreshold{
		BurnRateThresholdDual: &alerts.BurnRateThresholdDual{
			Rules: expandSloBurnRateRules(burnRate.Rules),
			Dual: &alerts.BurnRateTypeDual{
				TimeDuration: &alerts.TimeDuration{
					Duration: &duration,
					Unit:     expandDurationUnit(burnRate.BurnRateType.Dual.TimeDuration.Unit),
				},
			},
		},
	}
}

func expandDurationUnit(durationUnit TimeDurationUnit) *alerts.DurationUnit {
	if durationUnit == TimeDurationUnitHours {
		return alerts.DURATIONUNIT_DURATION_UNIT_HOURS.Ptr()
	}
	return alerts.DURATIONUNIT_DURATION_UNIT_UNSPECIFIED.Ptr()
}

func expandSloBurnRateRules(rules []BurnRateRule) []alerts.SloThresholdRule {
	result := make([]alerts.SloThresholdRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, *expandSloBurnRateRule(rule))
	}
	return result
}

func expandSloBurnRateRule(rule BurnRateRule) *alerts.SloThresholdRule {
	return &alerts.SloThresholdRule{
		Condition: expandSloBurnRateRuleCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override, alerts.ALERTDEFPRIORITY_ALERT_DEF_PRIORITY_P1),
	}
}

func expandSloBurnRateRuleCondition(condition BurnRateRuleCondition) *alerts.SloThresholdCondition {
	return &alerts.SloThresholdCondition{
		Threshold: alerts.PtrFloat64(condition.Threshold.AsApproximateFloat64()),
	}
}

func expandLogsUniqueCount(uniqueCount *LogsUniqueCount) *alerts.LogsUniqueCountType {
	return &alerts.LogsUniqueCountType{
		LogsFilter:                  expandLogsFilter(uniqueCount.LogsFilter),
		Rules:                       expandLogsUniqueCountRules(uniqueCount.Rules),
		NotificationPayloadFilter:   uniqueCount.NotificationPayloadFilter,
		MaxUniqueCountPerGroupByKey: alerts.PtrString(strconv.FormatUint(*uniqueCount.MaxUniqueCountPerGroupByKey, 10)),
		UniqueCountKeypath:          alerts.PtrString(uniqueCount.UniqueCountKeypath),
	}
}

func expandLogsUniqueCountRules(rules []LogsUniqueCountRule) []alerts.LogsUniqueCountRule {
	result := make([]alerts.LogsUniqueCountRule, len(rules))
	for i := range rules {
		result[i] = *expandLogsUniqueCountRule(rules[i])
	}

	return result
}

func expandLogsUniqueCountRule(rule LogsUniqueCountRule) *alerts.LogsUniqueCountRule {
	return &alerts.LogsUniqueCountRule{
		Condition: expandLogsUniqueCountCondition(rule.Condition),
	}
}

func expandLogsUniqueCountCondition(condition LogsUniqueCountCondition) *alerts.LogsUniqueCountCondition {
	return &alerts.LogsUniqueCountCondition{
		MaxUniqueCount: alerts.PtrString(strconv.FormatInt(condition.Threshold, 10)),
		TimeWindow:     expandLogsUniqueCountTimeWindow(condition.TimeWindow),
	}
}

func expandLogsUniqueCountTimeWindow(timeWindow LogsUniqueCountTimeWindow) *alerts.LogsUniqueValueTimeWindow {
	return &alerts.LogsUniqueValueTimeWindow{
		LogsUniqueValueTimeWindowSpecificValue: LogsUniqueCountTimeWindowValueToOpenAPI[timeWindow.SpecificValue].Ptr(),
	}
}

func expandLogsNewValue(logsNewValue *LogsNewValue) *alerts.LogsNewValueType {
	return &alerts.LogsNewValueType{
		LogsFilter:                expandLogsFilter(logsNewValue.LogsFilter),
		Rules:                     expandLogsNewValueRules(logsNewValue.Rules),
		NotificationPayloadFilter: logsNewValue.NotificationPayloadFilter,
	}
}

func expandLogsNewValueRules(rules []LogsNewValueRule) []alerts.LogsNewValueRule {
	result := make([]alerts.LogsNewValueRule, len(rules))
	for i := range rules {
		result[i] = *expandLogsNewValueRule(rules[i])
	}

	return result
}

func expandLogsNewValueRule(rule LogsNewValueRule) *alerts.LogsNewValueRule {
	return &alerts.LogsNewValueRule{
		Condition: expandLogsNewValueRuleCondition(rule.Condition),
	}
}

func expandLogsNewValueRuleCondition(condition LogsNewValueRuleCondition) *alerts.LogsNewValueCondition {
	return &alerts.LogsNewValueCondition{
		KeypathToTrack: alerts.PtrString(condition.KeypathToTrack),
		TimeWindow:     expandLogsNewValueTimeWindow(condition.TimeWindow),
	}
}

func expandLogsNewValueTimeWindow(timeWindow LogsNewValueTimeWindow) *alerts.LogsNewValueTimeWindow {
	return &alerts.LogsNewValueTimeWindow{
		LogsNewValueTimeWindowSpecificValue: LogsNewValueTimeWindowValueToOpenAPI[timeWindow.SpecificValue].Ptr(),
	}
}

func expandMetricAnomaly(metricAnomaly *MetricAnomaly) *alerts.MetricAnomalyType {
	return &alerts.MetricAnomalyType{
		MetricFilter: &alerts.MetricFilter{
			Promql: alerts.PtrString(metricAnomaly.MetricFilter.Promql),
		},
		Rules: expandMetricAnomalyRules(metricAnomaly.Rules),
	}
}

func expandMetricAnomalyRules(rules []MetricAnomalyRule) []alerts.MetricAnomalyRule {
	result := make([]alerts.MetricAnomalyRule, len(rules))
	for i := range rules {
		result[i] = *expandMetricAnomalyRule(rules[i])
	}
	return result
}

func expandMetricAnomalyRule(rule MetricAnomalyRule) *alerts.MetricAnomalyRule {
	return &alerts.MetricAnomalyRule{
		Condition: expandMetricAnomalyCondition(rule.Condition),
	}
}

func expandMetricAnomalyCondition(condition MetricAnomalyCondition) *alerts.MetricAnomalyCondition {
	return &alerts.MetricAnomalyCondition{
		Threshold:           alerts.PtrFloat64(condition.Threshold.AsApproximateFloat64()),
		ForOverPct:          alerts.PtrInt64(condition.ForOverPct),
		OfTheLast:           expandAnomalyMetricTimeWindow(condition.OfTheLast),
		MinNonNullValuesPct: alerts.PtrInt64(condition.MinNonNullValuesPct),
		ConditionType:       MetricAnomalyConditionTypeToOpenAPI[condition.ConditionType].Ptr(),
	}
}

func expandLogsAnomaly(anomaly *LogsAnomaly) *alerts.LogsAnomalyType {
	return &alerts.LogsAnomalyType{
		LogsFilter:                expandLogsFilter(anomaly.LogsFilter),
		Rules:                     expandLogsAnomalyRules(anomaly.Rules),
		NotificationPayloadFilter: anomaly.NotificationPayloadFilter,
	}
}

func expandLogsAnomalyRules(rules []LogsAnomalyRule) []alerts.LogsAnomalyRule {
	result := make([]alerts.LogsAnomalyRule, len(rules))
	for i := range rules {
		result[i] = *expandLogsAnomalyRule(rules[i])
	}

	return result
}

func expandLogsAnomalyRule(rule LogsAnomalyRule) *alerts.LogsAnomalyRule {
	return &alerts.LogsAnomalyRule{
		Condition: expandLogsAnomalyRuleCondition(rule.Condition),
	}
}

func expandLogsAnomalyRuleCondition(condition LogsAnomalyCondition) *alerts.LogsAnomalyCondition {
	return &alerts.LogsAnomalyCondition{
		MinimumThreshold: alerts.PtrFloat64(condition.MinimumThreshold.AsApproximateFloat64()),
		TimeWindow:       expandLogsTimeWindow(condition.TimeWindow),
		ConditionType:    alerts.LOGSANOMALYCONDITIONTYPE_LOGS_ANOMALY_CONDITION_TYPE_MORE_THAN_USUAL_OR_UNSPECIFIED.Ptr(),
	}
}

func expandFlow(listingAlertsProperties *GetResourceRefProperties, flow *Flow) *alerts.FlowType {
	return &alerts.FlowType{
		Stages:             expandFlowStages(listingAlertsProperties, flow.Stages),
		EnforceSuppression: alerts.PtrBool(flow.EnforceSuppression),
	}
}

func expandFlowStages(listingAlertsProperties *GetResourceRefProperties, stages []FlowStage) []alerts.FlowStages {
	result := make([]alerts.FlowStages, len(stages))
	for i, stage := range stages {
		result[i] = *expandFlowStage(listingAlertsProperties, stage)
	}

	return result
}

func expandFlowStage(listingAlertsProperties *GetResourceRefProperties, stage FlowStage) *alerts.FlowStages {
	return &alerts.FlowStages{
		FlowStagesGroups: expandFlowStagesType(listingAlertsProperties, stage.FlowStagesType),
		TimeframeMs:      alerts.PtrString(strconv.FormatInt(stage.TimeframeMs, 10)),
		TimeframeType:    TimeframeTypeToOpenAPI[stage.TimeframeType].Ptr(),
	}
}

func expandFlowStagesType(listingAlertsProperties *GetResourceRefProperties, stagesType FlowStagesType) *alerts.FlowStagesGroups {
	return &alerts.FlowStagesGroups{
		Groups: expandFlowStagesGroups(listingAlertsProperties, stagesType.Groups),
	}
}

func expandFlowStagesGroups(listingAlertsProperties *GetResourceRefProperties, groups []FlowStageGroup) []alerts.FlowStagesGroup {
	result := make([]alerts.FlowStagesGroup, len(groups))
	for i, group := range groups {
		result[i] = *expandFlowStagesGroup(listingAlertsProperties, group)
	}

	return result
}

func expandFlowStagesGroup(listingWebhooksProperties *GetResourceRefProperties, group FlowStageGroup) *alerts.FlowStagesGroup {
	return &alerts.FlowStagesGroup{
		AlertDefs: expandFlowStagesGroupsAlertDefs(listingWebhooksProperties, group.AlertDefs),
		NextOp:    FlowStageGroupNextOpToOpenAPI[group.NextOp].Ptr(),
		AlertsOp:  FlowStageGroupAlertsOpToOpenAPI[group.AlertsOp].Ptr(),
	}
}

func expandFlowStagesGroupsAlertDefs(listingAlertsProperties *GetResourceRefProperties, alertDefs []FlowStagesGroupsAlertDefs) []alerts.FlowStagesGroupsAlertDefs {
	result := make([]alerts.FlowStagesGroupsAlertDefs, len(alertDefs))
	var errs error
	for i := range alertDefs {
		expandedAlertDef, err := expandFlowStagesGroupsAlertDef(listingAlertsProperties, alertDefs[i])
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result[i] = *expandedAlertDef
	}

	return result
}

func expandFlowStagesGroupsAlertDef(listingAlertsProperties *GetResourceRefProperties, defs FlowStagesGroupsAlertDefs) (*alerts.FlowStagesGroupsAlertDefs, error) {
	id, err := expandAlertRef(listingAlertsProperties, defs.AlertRef)
	if err != nil {
		return nil, err
	}

	return &alerts.FlowStagesGroupsAlertDefs{
		Id:  alerts.PtrString(id),
		Not: alerts.PtrBool(defs.Not),
	}, nil
}

func expandAlertRef(listingAlertsProperties *GetResourceRefProperties, ref AlertRef) (string, error) {
	if backendRef := ref.BackendRef; backendRef != nil {
		if id := backendRef.ID; id != nil {
			return *id, nil
		} else if name := backendRef.Name; name != nil {
			return convertAlertNameToID(listingAlertsProperties, *name)
		}
	} else if resourceRef := ref.ResourceRef; resourceRef != nil {
		if namespace := resourceRef.Namespace; namespace != nil {
			listingAlertsProperties.Namespace = *namespace
		}
		return convertAlertCrNameToID(listingAlertsProperties, resourceRef.Name)
	}

	return "", fmt.Errorf("alert ref not found")
}

func convertAlertCrNameToID(listingAlertsProperties *GetResourceRefProperties, alertCrName string) (string, error) {
	ctx, namespace := listingAlertsProperties.Ctx, listingAlertsProperties.Namespace
	alertCR := &Alert{}
	err := config.GetClient().Get(ctx, client.ObjectKey{Name: alertCrName, Namespace: namespace}, alertCR)
	if err != nil {
		return "", fmt.Errorf("failed to get alert %w", err)
	}

	if alertCR.Status.ID == nil {
		return "", fmt.Errorf("alert with name %s has no ID", alertCrName)
	}

	return *alertCR.Status.ID, nil
}

func convertAlertNameToID(listingAlertsProperties *GetResourceRefProperties, alertName string) (string, error) {
	if listingAlertsProperties.AlertNameToId == nil {
		listingAlertsProperties.AlertNameToId = make(map[string]string)
		log, alertsClient, ctx := listingAlertsProperties.Log, listingAlertsProperties.ClientSet.Alerts(), listingAlertsProperties.Ctx
		log.V(1).Info("Listing all alerts")
		listAlertsResp, httpResp, err := alertsClient.AlertDefsServiceListAlertDefs(ctx).Execute()
		if err != nil {
			return "", fmt.Errorf("failed to list all alerts %w", oapicxsdk.NewAPIError(httpResp, err))
		}

		for _, alert := range listAlertsResp.GetAlertDefs() {
			var name string

			props := alert.GetAlertDefProperties()
			switch {
			case props.AlertDefPropertiesFlow != nil && props.AlertDefPropertiesFlow.Name != nil:
				name = *props.AlertDefPropertiesFlow.Name
			case props.AlertDefPropertiesLogsAnomaly != nil && props.AlertDefPropertiesLogsAnomaly.Name != nil:
				name = *props.AlertDefPropertiesLogsAnomaly.Name
			case props.AlertDefPropertiesLogsImmediate != nil && props.AlertDefPropertiesLogsImmediate.Name != nil:
				name = *props.AlertDefPropertiesLogsImmediate.Name
			case props.AlertDefPropertiesLogsNewValue != nil && props.AlertDefPropertiesLogsNewValue.Name != nil:
				name = *props.AlertDefPropertiesLogsNewValue.Name
			case props.AlertDefPropertiesLogsRatioThreshold != nil && props.AlertDefPropertiesLogsRatioThreshold.Name != nil:
				name = *props.AlertDefPropertiesLogsRatioThreshold.Name
			case props.AlertDefPropertiesLogsThreshold != nil && props.AlertDefPropertiesLogsThreshold.Name != nil:
				name = *props.AlertDefPropertiesLogsThreshold.Name
			case props.AlertDefPropertiesLogsTimeRelativeThreshold != nil && props.AlertDefPropertiesLogsTimeRelativeThreshold.Name != nil:
				name = *props.AlertDefPropertiesLogsTimeRelativeThreshold.Name
			case props.AlertDefPropertiesLogsUniqueCount != nil && props.AlertDefPropertiesLogsUniqueCount.Name != nil:
				name = *props.AlertDefPropertiesLogsUniqueCount.Name
			case props.AlertDefPropertiesMetricAnomaly != nil && props.AlertDefPropertiesMetricAnomaly.Name != nil:
				name = *props.AlertDefPropertiesMetricAnomaly.Name
			case props.AlertDefPropertiesMetricThreshold != nil && props.AlertDefPropertiesMetricThreshold.Name != nil:
				name = *props.AlertDefPropertiesMetricThreshold.Name
			case props.AlertDefPropertiesSloThreshold != nil && props.AlertDefPropertiesSloThreshold.Name != nil:
				name = *props.AlertDefPropertiesSloThreshold.Name
			case props.AlertDefPropertiesTracingImmediate != nil && props.AlertDefPropertiesTracingImmediate.Name != nil:
				name = *props.AlertDefPropertiesTracingImmediate.Name
			case props.AlertDefPropertiesTracingThreshold != nil && props.AlertDefPropertiesTracingThreshold.Name != nil:
				name = *props.AlertDefPropertiesTracingThreshold.Name
			default:
				log.V(1).Info("Skipping alert with missing name", "alertID", alert.GetId())
				continue
			}

			listingAlertsProperties.AlertNameToId[name] = alert.GetId()
		}
	}

	alertID, ok := listingAlertsProperties.AlertNameToId[alertName]
	if !ok {
		return "", fmt.Errorf("alert with name %s not found", alertName)
	}

	return alertID, nil
}

func expandTracingThreshold(tracingThreshold *TracingThreshold) *alerts.TracingThresholdType {
	tracingThresholdType := &alerts.TracingThresholdType{
		Rules:                     expandTracingThresholdRules(tracingThreshold.Rules),
		NotificationPayloadFilter: tracingThreshold.NotificationPayloadFilter,
	}

	if tracingFilter := tracingThreshold.TracingFilter; tracingFilter != nil {
		tracingThresholdType.TracingFilter = expandTracingFilter(tracingFilter)
	}

	return tracingThresholdType
}

func expandTracingImmediate(tracingImmediate *TracingImmediate) *alerts.TracingImmediateType {
	result := &alerts.TracingImmediateType{
		NotificationPayloadFilter: tracingImmediate.NotificationPayloadFilter,
	}

	if tracingFilter := tracingImmediate.TracingFilter; tracingFilter != nil {
		result.TracingFilter = expandTracingFilter(tracingFilter)
	}

	return result
}

func expandTracingFilter(filter *TracingFilter) *alerts.TracingFilter {
	if filter == nil {
		return nil
	}

	return &alerts.TracingFilter{
		SimpleFilter: expandTracingSimpleFilter(filter.Simple),
	}
}

func expandTracingSimpleFilter(filter *TracingSimpleFilter) *alerts.TracingSimpleFilter {
	var latencyThresholdStr string
	if filter.LatencyThresholdMs != nil {
		latencyThresholdStr = strconv.FormatUint(*filter.LatencyThresholdMs, 10)
	}

	return &alerts.TracingSimpleFilter{
		TracingLabelFilters: expandTracingLabelFilters(filter.TracingLabelFilters),
		LatencyThresholdMs:  alerts.PtrString(latencyThresholdStr),
	}
}

func expandTracingLabelFilters(filters *TracingLabelFilters) *alerts.TracingLabelFilters {
	if filters == nil {
		return nil
	}

	return &alerts.TracingLabelFilters{
		ApplicationName: expandTracingFilterTypes(filters.ApplicationName),
		SubsystemName:   expandTracingFilterTypes(filters.SubsystemName),
		ServiceName:     expandTracingFilterTypes(filters.ServiceName),
		OperationName:   expandTracingFilterTypes(filters.OperationName),
		SpanFields:      expandTracingSpanFieldsFilterTypes(filters.SpanFields),
	}
}

func expandTracingFilterTypes(filters []TracingFilterType) []alerts.TracingFilterType {
	result := make([]alerts.TracingFilterType, len(filters))
	for i := range filters {
		result[i] = *expandTracingFilterType(filters[i])
	}

	return result
}

func expandTracingFilterType(filterType TracingFilterType) *alerts.TracingFilterType {
	return &alerts.TracingFilterType{
		Values:    filterType.Values,
		Operation: TracingFilterOperationTypeToOpenAPI[filterType.Operation].Ptr(),
	}
}

func expandTracingSpanFieldsFilterTypes(fields []TracingSpanFieldsFilterType) []alerts.TracingSpanFieldsFilterType {
	result := make([]alerts.TracingSpanFieldsFilterType, len(fields))
	for i := range fields {
		result[i] = *expandTracingSpanFieldsFilterType(fields[i])
	}

	return result
}

func expandTracingSpanFieldsFilterType(filterType TracingSpanFieldsFilterType) *alerts.TracingSpanFieldsFilterType {
	return &alerts.TracingSpanFieldsFilterType{
		Key:        alerts.PtrString(filterType.Key),
		FilterType: expandTracingFilterType(filterType.FilterType),
	}
}

func expandTracingThresholdRules(rules []TracingThresholdRule) []alerts.TracingThresholdRule {
	result := make([]alerts.TracingThresholdRule, len(rules))
	for i := range rules {
		result[i] = *expandTracingThresholdRule(rules[i])
	}

	return result
}

func expandTracingThresholdRule(rule TracingThresholdRule) *alerts.TracingThresholdRule {
	return &alerts.TracingThresholdRule{
		Condition: expandTracingThresholdCondition(rule.Condition),
	}
}

func expandTracingThresholdCondition(condition TracingThresholdRuleCondition) *alerts.TracingThresholdCondition {
	return &alerts.TracingThresholdCondition{
		SpanAmount:    alerts.PtrFloat64(condition.SpanAmount.AsApproximateFloat64()),
		TimeWindow:    expandTracingTimeWindow(condition.TimeWindow),
		ConditionType: alerts.TRACINGTHRESHOLDCONDITIONTYPE_TRACING_THRESHOLD_CONDITION_TYPE_MORE_THAN_OR_UNSPECIFIED.Ptr(),
	}
}

func expandTracingTimeWindow(timeWindow TracingTimeWindow) *alerts.TracingTimeWindow {
	return &alerts.TracingTimeWindow{
		TracingTimeWindowValue: TracingTimeWindowSpecificValueToOpenAPI[timeWindow.SpecificValue].Ptr(),
	}
}

func expandMetricThreshold(threshold *MetricThreshold, priority alerts.AlertDefPriority) *alerts.MetricThresholdType {
	thresholdType := &alerts.MetricThresholdType{
		MetricFilter:               expandMetricFilter(threshold.MetricFilter),
		Rules:                      expandMetricThresholdRules(threshold.Rules, priority),
		UndetectedValuesManagement: expandUndetectedValuesManagement(threshold.UndetectedValuesManagement),
	}

	missingValues := expandMetricMissingValues(&threshold.MissingValues)
	if missingValues != nil {
		thresholdType.MissingValues = missingValues
	}

	return thresholdType
}

func expandMetricFilter(metricFilter MetricFilter) *alerts.MetricFilter {
	return &alerts.MetricFilter{
		Promql: alerts.PtrString(metricFilter.Promql),
	}
}

func expandMetricThresholdRules(rules []MetricThresholdRule, priority alerts.AlertDefPriority) []alerts.MetricThresholdRule {
	result := make([]alerts.MetricThresholdRule, len(rules))
	for i := range rules {
		result[i] = *expandMetricThresholdRule(rules[i], priority)
	}

	return result
}

func expandMetricThresholdRule(rule MetricThresholdRule, priority alerts.AlertDefPriority) *alerts.MetricThresholdRule {
	return &alerts.MetricThresholdRule{
		Condition: expandMetricThresholdCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override, priority),
	}
}

func expandMetricThresholdCondition(condition MetricThresholdRuleCondition) *alerts.MetricThresholdCondition {
	metricThresholdCondition := &alerts.MetricThresholdCondition{
		Threshold:     alerts.PtrFloat64(condition.Threshold.AsApproximateFloat64()),
		ForOverPct:    alerts.PtrInt64(int64(condition.ForOverPct)),
		ConditionType: MetricThresholdConditionTypeToOpenAPI[condition.ConditionType].Ptr(),
	}

	ofTheLast := expandMetricTimeWindow(condition.OfTheLast)
	if ofTheLast != nil {
		metricThresholdCondition.OfTheLast = ofTheLast
	}

	return metricThresholdCondition
}

func expandMetricTimeWindow(timeWindow MetricTimeWindow) *alerts.MetricTimeWindow {
	if specificValue := timeWindow.SpecificValue; specificValue != nil {
		return &alerts.MetricTimeWindow{
			MetricTimeWindowMetricTimeWindowSpecificValue: &alerts.MetricTimeWindowMetricTimeWindowSpecificValue{
				MetricTimeWindowSpecificValue: MetricTimeWindowToOpenAPI[*specificValue].Ptr(),
			},
		}
	} else if dynamicTimeWindow := timeWindow.DynamicDuration; dynamicTimeWindow != nil {
		return &alerts.MetricTimeWindow{
			MetricTimeWindowMetricTimeWindowDynamicDuration: &alerts.MetricTimeWindowMetricTimeWindowDynamicDuration{
				MetricTimeWindowDynamicDuration: dynamicTimeWindow,
			},
		}
	}

	return nil
}

func expandAnomalyMetricTimeWindow(timeWindow MetricAnomalyTimeWindow) *alerts.MetricTimeWindow {
	return &alerts.MetricTimeWindow{
		MetricTimeWindowMetricTimeWindowSpecificValue: &alerts.MetricTimeWindowMetricTimeWindowSpecificValue{
			MetricTimeWindowSpecificValue: MetricTimeWindowToOpenAPI[timeWindow.SpecificValue].Ptr(),
		},
	}
}

func expandMetricMissingValues(missingValues *MetricMissingValues) *alerts.MetricMissingValues {
	if missingValues == nil {
		return nil
	} else if missingValues.ReplaceWithZero {
		return &alerts.MetricMissingValues{
			MetricMissingValuesReplaceWithZero: &alerts.MetricMissingValuesReplaceWithZero{
				ReplaceWithZero: alerts.PtrBool(true),
			},
		}
	} else if missingValues.MinNonNullValuesPct != nil {
		return &alerts.MetricMissingValues{
			MetricMissingValuesMinNonNullValuesPct: &alerts.MetricMissingValuesMinNonNullValuesPct{
				MinNonNullValuesPct: missingValues.MinNonNullValuesPct,
			},
		}
	}

	return nil
}

func expandLogsImmediate(immediate *LogsImmediate) *alerts.LogsImmediateType {
	logsFilter := expandLogsFilter(immediate.LogsFilter)
	if logsFilter == nil {
		return nil
	}

	return &alerts.LogsImmediateType{
		LogsFilter:                logsFilter,
		NotificationPayloadFilter: immediate.NotificationPayloadFilter,
	}
}

func expandLogsThreshold(logsThreshold *LogsThreshold, priority alerts.AlertDefPriority) *alerts.LogsThresholdType {
	return &alerts.LogsThresholdType{
		LogsFilter:                 expandLogsFilter(logsThreshold.LogsFilter),
		UndetectedValuesManagement: expandUndetectedValuesManagement(logsThreshold.UndetectedValuesManagement),
		Rules:                      expandLogsThresholdRules(logsThreshold.Rules, priority),
		NotificationPayloadFilter:  logsThreshold.NotificationPayloadFilter,
	}
}

func expandLogsRatioThreshold(logsRatioThreshold *LogsRatioThreshold, priority alerts.AlertDefPriority) *alerts.LogsRatioThresholdType {
	if logsRatioThreshold == nil {
		return nil
	}

	thresholdType := &alerts.LogsRatioThresholdType{
		NumeratorAlias:   alerts.PtrString(logsRatioThreshold.NumeratorAlias),
		DenominatorAlias: alerts.PtrString(logsRatioThreshold.DenominatorAlias),
		Rules:            expandLogsRatioThresholdRules(logsRatioThreshold.Rules, priority),
	}

	Numerator := expandLogsFilter(&logsRatioThreshold.Numerator)
	if Numerator != nil {
		thresholdType.Numerator = Numerator
	}

	Denominator := expandLogsFilter(&logsRatioThreshold.Denominator)
	if Denominator != nil {
		thresholdType.Denominator = Denominator
	}

	return thresholdType
}

func expandLogsTimeRelativeThreshold(threshold *LogsTimeRelativeThreshold, priority alerts.AlertDefPriority) *alerts.LogsTimeRelativeThresholdType {
	return &alerts.LogsTimeRelativeThresholdType{
		LogsFilter:                 expandLogsFilter(&threshold.LogsFilter),
		Rules:                      expandLogsTimeRelativeRules(threshold.Rules, priority),
		IgnoreInfinity:             alerts.PtrBool(threshold.IgnoreInfinity),
		NotificationPayloadFilter:  threshold.NotificationPayloadFilter,
		UndetectedValuesManagement: expandUndetectedValuesManagement(threshold.UndetectedValuesManagement),
	}
}

func expandLogsTimeRelativeRules(rules []LogsTimeRelativeRule, priority alerts.AlertDefPriority) []alerts.LogsTimeRelativeRule {
	result := make([]alerts.LogsTimeRelativeRule, len(rules))
	for i := range rules {
		result[i] = *expandLogsTimeRelativeRule(rules[i], priority)
	}

	return result
}

func expandLogsTimeRelativeRule(rule LogsTimeRelativeRule, priority alerts.AlertDefPriority) *alerts.LogsTimeRelativeRule {
	return &alerts.LogsTimeRelativeRule{
		Condition: expandLogsTimeRelativeCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override, priority),
	}
}

func expandLogsTimeRelativeCondition(condition LogsTimeRelativeCondition) *alerts.LogsTimeRelativeCondition {
	return &alerts.LogsTimeRelativeCondition{
		Threshold:     alerts.PtrFloat64(condition.Threshold.AsApproximateFloat64()),
		ComparedTo:    LogsTimeRelativeComparedToOpenAPI[condition.ComparedTo].Ptr(),
		ConditionType: LogsTimeRelativeConditionTypeToOpenAPI[condition.ConditionType].Ptr(),
	}
}

func expandLogsRatioThresholdRules(rules []LogsRatioThresholdRule, priority alerts.AlertDefPriority) []alerts.LogsRatioRules {
	result := make([]alerts.LogsRatioRules, len(rules))
	for i := range rules {
		result[i] = *expandLogsRatioThresholdRule(rules[i], priority)
	}
	return result
}

func expandLogsRatioThresholdRule(rule LogsRatioThresholdRule, priority alerts.AlertDefPriority) *alerts.LogsRatioRules {
	return &alerts.LogsRatioRules{
		Condition: expandLogsRatioCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override, priority),
	}
}

func expandLogsRatioCondition(condition LogsRatioCondition) *alerts.LogsRatioCondition {
	return &alerts.LogsRatioCondition{
		Threshold:     alerts.PtrFloat64(condition.Threshold.AsApproximateFloat64()),
		TimeWindow:    expandLogsRatioTimeWindow(condition.TimeWindow),
		ConditionType: LogsRatioConditionTypeToOpenAPI[condition.ConditionType].Ptr(),
	}
}

func expandLogsRatioTimeWindow(timeWindow LogsRatioTimeWindow) *alerts.LogsRatioTimeWindow {
	return &alerts.LogsRatioTimeWindow{
		LogsRatioTimeWindowSpecificValue: LogsRatioTimeWindowToOpenAPI[timeWindow.SpecificValue].Ptr(),
	}
}

func expandAlertOverride(override *AlertOverride, priority alerts.AlertDefPriority) *alerts.AlertDefOverride {
	if override == nil {
		return &alerts.AlertDefOverride{
			Priority: priority.Ptr(),
		}
	}

	return &alerts.AlertDefOverride{
		Priority: AlertPriorityToOpenAPIPriority[override.Priority].Ptr(),
	}
}

func expandLogsFilter(filter *LogsFilter) *alerts.V3LogsFilter {
	if filter == nil {
		return nil
	}

	return &alerts.V3LogsFilter{
		SimpleFilter: expandSimpleFilter(filter.SimpleFilter),
	}
}

func expandSimpleFilter(filter LogsSimpleFilter) *alerts.LogsSimpleFilter {
	return &alerts.LogsSimpleFilter{
		LuceneQuery:  filter.LuceneQuery,
		LabelFilters: expandLabelFilters(filter.LabelFilters),
	}
}

func expandLabelFilters(filters *LabelFilters) *alerts.LabelFilters {
	return &alerts.LabelFilters{
		ApplicationName: expandLabelFilterTypes(filters.ApplicationName),
		SubsystemName:   expandLabelFilterTypes(filters.SubsystemName),
		Severities:      expandLogSeverities(filters.Severity),
	}
}

func expandLogSeverities(severity []LogSeverity) []alerts.LogSeverity {
	result := make([]alerts.LogSeverity, len(severity))
	for i, s := range severity {
		result[i] = LogSeverityToOpenAPISeverity[s]
	}

	return result
}

func expandLabelFilterTypes(name []LabelFilterType) []alerts.LabelFilterType {
	result := make([]alerts.LabelFilterType, len(name))
	for i, n := range name {
		result[i] = alerts.LabelFilterType{
			Value:     alerts.PtrString(n.Value),
			Operation: LogsFiltersOperationToOpenAPIOperation[n.Operation].Ptr(),
		}
	}

	return result
}

func expandUndetectedValuesManagement(management *UndetectedValuesManagement) *alerts.V3UndetectedValuesManagement {
	if management == nil {
		return nil
	}
	autoRetireTimeframe := AutoRetireTimeframeToOpenAPIAutoRetireTimeframe[management.AutoRetireTimeframe]
	return &alerts.V3UndetectedValuesManagement{
		TriggerUndetectedValues: alerts.PtrBool(management.TriggerUndetectedValues),
		AutoRetireTimeframe:     &autoRetireTimeframe,
	}
}

func expandLogsThresholdRules(rules []LogsThresholdRule, priority alerts.AlertDefPriority) []alerts.LogsThresholdRule {
	result := make([]alerts.LogsThresholdRule, len(rules))
	for i := range rules {
		result[i] = *expandLogsThresholdRule(rules[i], priority)
	}

	return result
}

func expandLogsThresholdRule(rule LogsThresholdRule, priority alerts.AlertDefPriority) *alerts.LogsThresholdRule {
	return &alerts.LogsThresholdRule{
		Condition: expandLogsThresholdRuleCondition(rule.Condition),
		Override:  expandAlertOverride(rule.Override, priority),
	}
}

func expandLogsThresholdRuleCondition(condition LogsThresholdRuleCondition) *alerts.LogsThresholdCondition {
	return &alerts.LogsThresholdCondition{
		Threshold:     alerts.PtrFloat64(condition.Threshold.AsApproximateFloat64()),
		TimeWindow:    expandLogsTimeWindow(condition.TimeWindow),
		ConditionType: LogsThresholdConditionTypeToOpenAPI[condition.LogsThresholdConditionType].Ptr(),
	}
}

func expandLogsTimeWindow(timeWindow LogsTimeWindow) *alerts.LogsTimeWindow {
	return &alerts.LogsTimeWindow{
		LogsTimeWindowSpecificValue: LogsTimeWindowToOpenAPI[timeWindow.SpecificValue].Ptr(),
	}
}

// +k8s:deepcopy-gen=false
type GetResourceRefProperties struct {
	Ctx             context.Context
	Log             logr.Logger
	AlertNameToId   map[string]string
	WebhookNameToId map[string]int64
	ClientSet       *oapicxsdk.ClientSet
	Namespace       string
}

func convertCRNameToIntegrationID(name string, properties *GetResourceRefProperties) (*int64, error) {
	ctx, namespace := properties.Ctx, properties.Namespace

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   utils.CoralogixAPIGroup,
		Kind:    "OutboundWebhook",
		Version: utils.V1alpha1APIVersion,
	})

	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, u); err != nil {
		return nil, fmt.Errorf("failed to get webhook, name: %s, namespace: %s, error: %w", name, namespace, err)
	}

	if !config.GetConfig().Selector.Matches(u.GetLabels(), u.GetNamespace()) {
		return nil, fmt.Errorf("outbound webhook %s does not match selector", u.GetName())
	}

	externalID, found, err := unstructured.NestedString(u.Object, "status", "externalId")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("status.externalID not found")
	}

	externalIDInt, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert externalID to int, externalID: %s, error: %w", externalID, err)
	}

	return ptr.To(int64(externalIDInt)), nil
}
