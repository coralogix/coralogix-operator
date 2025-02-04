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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// AlertSpec defines the desired state of Alert.
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

// +kubebuilder:validation:Enum=More;Less;MoreOrEqual;LessOrEqual;MoreThanUsual;LessThanUsual
type PromqlAlertWhen string

const (
	PromqlAlertWhenLessThan      PromqlAlertWhen = "Less"
	PromqlAlertWhenMoreThan      PromqlAlertWhen = "More"
	PromqlAlertWhenMoreOrEqual   PromqlAlertWhen = "MoreOrEqual"
	PromqlAlertWhenLessOrEqual   PromqlAlertWhen = "LessOrEqual"
	PromqlAlertWhenMoreThanUsual PromqlAlertWhen = "MoreThanUsual"
	PromqlAlertWhenLessThanUsual PromqlAlertWhen = "LessThanUsual"
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

const (
	FlowOperatorAnd FlowOperator = "And"
	FlowOperatorOr  FlowOperator = "Or"
)

// AlertStatus defines the observed state of Alert
type AlertStatus struct {
	// +optional
	ID *string `json:"id"`
}

func NewDefaultAlertStatus() *AlertStatus {
	return &AlertStatus{
		ID: ptr.To(""),
	}
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Alert is the v1alpha1 version Schema for the alerts API. v1alpha1 Alert is going to be deprecated, consider using v1beta1.Alert instead.
type Alert struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertSpec   `json:"spec,omitempty"`
	Status AlertStatus `json:"status,omitempty"`
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
