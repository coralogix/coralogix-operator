/*
Copyright 2023.

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

package path

import (
	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	utils "github.com/coralogix/coralogix-operator/apis"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

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
		LogFilterOperationTypeOr:               cxsdk.LogFilterOperationIsOrUnspecified,
		LogFilterOperationTypeIncludes:         cxsdk.LogFilterOperationIncludes,
		LogFilterOperationTypeEndWith:          cxsdk.LogFilterOperationStartsWith,
		LogFilterOperationTypeEndWithStartWith: cxsdk.LogFilterOperationEndsWith,
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
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
// +kubebuilder:resource:path=alerts,scope=Namespaced

// Alert is the Schema for the alerts API
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

// AlertSpec defines the desired state of Alert
type AlertSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`
	// +optional
	Description string `json:"description,omitempty"`
	//+kubebuilder:validation:Enum=P1;P2;P3;P4
	Priority AlertPriority `json:"priority"`
	//+kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	GroupBy []string `json:"groupBy,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
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

type ActiveOn struct {
	DayOfWeek []DayOfWeek `json:"dayOfWeek,omitempty"`
	StartTime *TimeOfDay  `protobuf:"bytes,2,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	EndTime   *TimeOfDay  `protobuf:"bytes,3,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
}

type TimeOfDay struct {
	Hours   int32 `json:"hours,omitempty"`
	Minutes int32 `json:"minutes,omitempty"`
}

// +kubebuilder:validation:Enum=Sunday;Monday;Tuesday;Wednesday;Thursday;Friday;Saturday;
type DayOfWeek string

const (
	DayOfWeekSunday    DayOfWeek = "Sunday"
	DayOfWeekMonday    DayOfWeek = "Monday"
	DayOfWeekTuesday   DayOfWeek = "Tuesday"
	DayOfWeekWednesday DayOfWeek = "Wednesday"
	DayOfWeekThursday  DayOfWeek = "Thursday"
	DayOfWeekFriday    DayOfWeek = "Friday"
	DayOfWeekSaturday  DayOfWeek = "Saturday"
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
	LogsUnusual *LogsUnusual `json:"logsUnusual,omitempty"`
	// +optional
	MetricUnusual *MetricUnusual `json:"metricUnusual,omitempty"`
	// +optional
	LogsNewValue *LogsNewValue `json:"logsNewValue,omitempty"`
	// +optional
	LogsUniqueCount *LogsUniqueCount `json:"logsUniqueCount,omitempty"`
}

type LogsImmediate struct {
	LogsFilter                *LogsFilter `json:"logsFilter,omitempty"`
	NotificationPayloadFilter []string    `json:"notificationPayloadFilter,omitempty"`
}

type LogsThreshold struct {
}

type LogsRatioThreshold struct {
}

type LogsTimeRelativeThreshold struct {
}

type MetricThreshold struct {
}

type TracingThreshold struct {
}

type Flow struct {
}

type LogsUnusual struct {
}

type MetricUnusual struct {
}

type LogsNewValue struct {
}

type LogsUniqueCount struct {
}

type LogsFilter struct {
	FilterType FilterType `json:"filterType,omitempty"`
}

type FilterType struct {
	// +optional
	SimpleFilter *SimpleFilter `json:"simpleFilter,omitempty"`
}

type SimpleFilter struct {
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

// +kubebuilder:validation:Enum=Or;Includes;EndsWith;StartsWith
type LogFilterOperationType string

const (
	LogFilterOperationTypeOr               LogFilterOperationType = "Or"
	LogFilterOperationTypeIncludes         LogFilterOperationType = "Includes"
	LogFilterOperationTypeEndWith          LogFilterOperationType = "EndsWith"
	LogFilterOperationTypeEndWithStartWith LogFilterOperationType = "StartsWith"
)

// +kubebuilder:validation:Enum=Debug;Info;Warning;Error;Critical
type LogSeverity string

const (
	LogSeverityDebug    LogSeverity = "Debug"
	LogSeverityInfo     LogSeverity = "Info"
	LogSeverityWarning  LogSeverity = "Warning"
	LogSeverityError    LogSeverity = "Error"
	LogSeverityCritical LogSeverity = "Critical"
)

// +kubebuilder:validation:Enum=P1;P2;P3;P4
type AlertPriority string

const (
	AlertPriorityP1 AlertPriority = "P1"
	AlertPriorityP2 AlertPriority = "P2"
	AlertPriorityP3 AlertPriority = "P3"
	AlertPriorityP4 AlertPriority = "P4"
)

func init() {
	SchemeBuilder.Register(&Alert{}, &AlertList{})
}

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
		Priority:          AlertPriorityToProtoPriority[AlertPriorityP1],
		GroupBy:           utils.StringSliceToWrappedStringSlice(in.GroupBy),
		IncidentsSettings: &cxsdk.AlertDefIncidentSettings{},
		NotificationGroup: &cxsdk.AlertDefNotificationGroup{},
		Labels:            in.Labels,
		PhantomMode:       wrapperspb.Bool(in.PhantomMode),
	}
	alertDefProperties = expandAlertSchedule(alertDefProperties, in.Schedule)
	alertDefProperties = expandAlertTypeDefinition(alertDefProperties, in.TypeDefinition)

	return alertDefProperties
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
	}
	//} else if logsThreshold := definition.LogsThreshold; logsThreshold != nil {
	//	properties.TypeDefinition = expandLogsThreshold(logsThreshold)
	//} else if logsRatioThreshold := definition.LogsRatioThreshold; logsRatioThreshold != nil {
	//	properties.TypeDefinition = expandLogsRatioThreshold(logsRatioThreshold)
	//} else if logsTimeRelativeThreshold := definition.LogsTimeRelativeThreshold; logsTimeRelativeThreshold != nil {
	//	properties.TypeDefinition = expandLogsTimeRelativeThreshold(logsTimeRelativeThreshold)
	//} else if metricThreshold := definition.MetricThreshold; metricThreshold != nil {
	//	properties.TypeDefinition = expandMetricThreshold(metricThreshold)
	//} else if tracingThreshold := definition.TracingThreshold; tracingThreshold != nil {
	//	properties.TypeDefinition = expandTracingThreshold(tracingThreshold)
	//} else if flow := definition.Flow; flow != nil {
	//	properties.TypeDefinition = expandFlow(flow)
	//} else if logsUnusual := definition.LogsUnusual; logsUnusual != nil {
	//	properties.TypeDefinition = expandLogsUnusual(logsUnusual)
	//} else if metricUnusual := definition.MetricUnusual; metricUnusual != nil {
	//	properties.TypeDefinition = expandMetricUnusual(metricUnusual)
	//} else if logsNewValue := definition.LogsNewValue; logsNewValue != nil {
	//	properties.TypeDefinition = expandLogsNewValue(logsNewValue)
	//} else if logsUniqueCount := definition.LogsUniqueCount; logsUniqueCount != nil {
	//	properties.TypeDefinition = expandLogsUniqueCount(logsUniqueCount)
	//}

	return properties
}

func expandLogsImmediate(immediate *LogsImmediate) *cxsdk.AlertDefPropertiesLogsImmediate {
	return &cxsdk.AlertDefPropertiesLogsImmediate{
		LogsImmediate: &cxsdk.LogsImmediateType{
			LogsFilter:                expandLogsFilter(immediate.LogsFilter),
			NotificationPayloadFilter: utils.StringSliceToWrappedStringSlice(immediate.NotificationPayloadFilter),
		},
	}
}

func expandLogsFilter(filter *LogsFilter) *cxsdk.LogsFilter {
	if filter == nil {
		return nil
	}

	return expandFilterType(&cxsdk.LogsFilter{}, filter.FilterType)
}

func expandFilterType(filter *cxsdk.LogsFilter, filterType FilterType) *cxsdk.LogsFilter {
	if simpleFilter := filterType.SimpleFilter; simpleFilter != nil {
		filter.FilterType = expandSimpleFilter(simpleFilter)
	}

	return filter
}

func expandSimpleFilter(filter *SimpleFilter) *cxsdk.LogsFilterSimpleFilter {
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
