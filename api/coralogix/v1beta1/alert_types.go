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
	"strconv"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	utils "github.com/coralogix/coralogix-operator/api"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	LogsTimeWindowToProtoLogsTimeWindow = map[LogsTimeWindow]cxsdk.LogsTimeWindowValue{
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
	GroupBy []string `json:"groupBy,omitempty"`
	// +optional
	IncidentsSettings *IncidentsSettings `json:"incidentsSettings,omitempty"`
	// +optional
	NotificationGroup *NotificationGroup `json:"notificationGroup,omitempty"`
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
	GroupByFields []string                 `json:"groupByFields,omitempty"`
	Targets       NotificationGroupTargets `json:"targets"`
}

type NotificationGroupTargets struct {
	// +optional
	Simple *SimpleNotificationGroupTargets `json:"simple,omitempty"`
	// +optional
	Advanced *AdvancedNotificationGroupTargets `json:"advanced,omitempty"`
}

type SimpleNotificationGroupTargets struct {
	Integrations []IntegrationType `json:"integrations,omitempty"`
}

type IntegrationType struct {
	// +optional
	IntegrationId *uint32 `json:"integrationId,omitempty"`
	// +optional
	Recipients []string `json:"recipients,omitempty"`
}

type AdvancedNotificationGroupTargets struct {
	AdvancedTargetsSettings []AdvancedTargetsSettings `json:"advancedTargetsSettings,omitempty"`
}

type AdvancedTargetsSettings struct {
	// +optional
	IntegrationId *uint32 `json:"integrationId,omitempty"`
	// +optional
	Recipients []string `json:"recipients,omitempty"`
	//+kubebuilder:default=triggeredOnly
	NotifyOn                  NotifyOn `json:"notifyOn"`
	RetriggeringPeriodMinutes uint32   `json:"retriggeringPeriodMinutes"`
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
	LogsUnusual *LogsUnusual `json:"logsUnusual,omitempty"`
	// +optional
	MetricUnusual *MetricUnusual `json:"metricUnusual,omitempty"`
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
}

type LogsThresholdRuleCondition struct {
	LogsTimeWindow LogsTimeWindow `json:"logsTimeWindow"`
	// +kubebuilder:validation:Pattern=`^-?\d+(\.\d+)?$`
	Threshold                  string                     `json:"threshold"`
	LogsThresholdConditionType LogsThresholdConditionType `json:"logsThresholdConditionType"`
}

// +kubebuilder:validation:Enum={"5m","10m","15m","30m","1h","2h","6h","12h","24h","36h"}
type LogsTimeWindow string

const (
	LogsTimeWindowLast5Minutes  LogsTimeWindow = "5m"
	LogsTimeWindowLast10Minutes LogsTimeWindow = "10m"
	LogsTimeWindowLast15Minutes LogsTimeWindow = "15m"
	LogsTimeWindowLast30Minutes LogsTimeWindow = "30m"
	LogsTimeWindowLastHour      LogsTimeWindow = "1h"
	LogsTimeWindowLast2Hours    LogsTimeWindow = "2h"
	LogsTimeWindowLast6Hours    LogsTimeWindow = "6h"
	LogsTimeWindowLast12Hours   LogsTimeWindow = "12h"
	LogsTimeWindowLast24Hours   LogsTimeWindow = "24h"
	LogsTimeWindowLast36Hours   LogsTimeWindow = "36h"
)

// +kubebuilder:validation:Enum=moreThan;lessThan
type LogsThresholdConditionType string

const (
	LogsThresholdConditionTypeMoreThan LogsThresholdConditionType = "moreThan"
	LogsThresholdConditionTypeLessThan LogsThresholdConditionType = "lessThan"
)

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
		GroupBy:           utils.StringSliceToWrappedStringSlice(in.GroupBy),
		IncidentsSettings: expandIncidentsSettings(in.IncidentsSettings),
		NotificationGroup: expandNotificationGroup(in.NotificationGroup),
		Labels:            in.Labels,
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

	alertDefNotificationGroup := &cxsdk.AlertDefNotificationGroup{
		GroupByFields: utils.StringSliceToWrappedStringSlice(notificationGroup.GroupByFields),
	}
	alertDefNotificationGroup = expandNotificationGroupTargets(alertDefNotificationGroup, notificationGroup.Targets)

	return alertDefNotificationGroup
}

func expandNotificationGroupTargets(group *cxsdk.AlertDefNotificationGroup, targets NotificationGroupTargets) *cxsdk.AlertDefNotificationGroup {
	if simple := targets.Simple; simple != nil {
		group.Targets = &cxsdk.AlertDefNotificationGroupSimple{
			Simple: &cxsdk.AlertDefTargetSimple{
				Integrations: expandIntegrations(simple.Integrations),
			},
		}
	} else if advanced := targets.Advanced; advanced != nil {
		group.Targets = &cxsdk.AlertDefNotificationGroupAdvanced{
			Advanced: &cxsdk.AlertDefAdvancedTargets{
				AdvancedTargetsSettings: expandAdvancedTargetsSettings(advanced.AdvancedTargetsSettings),
			},
		}
	}

	return group
}

func expandAdvancedTargetsSettings(advancedTargetsSettings []AdvancedTargetsSettings) []*cxsdk.AlertDefAdvancedTargetSettings {
	result := make([]*cxsdk.AlertDefAdvancedTargetSettings, len(advancedTargetsSettings))
	for i, settings := range advancedTargetsSettings {
		result[i] = expandAdvancedTargetsSetting(settings)
	}
	return result
}

func expandAdvancedTargetsSetting(settings AdvancedTargetsSettings) *cxsdk.AlertDefAdvancedTargetSettings {
	notifyOn := NotifyOnToProtoNotifyOn[settings.NotifyOn]
	advancedTargetSettings := &cxsdk.AlertDefAdvancedTargetSettings{
		NotifyOn: &notifyOn,
	}

	if integrationID := settings.IntegrationId; integrationID != nil {
		advancedTargetSettings.Integration = &cxsdk.AlertDefIntegrationType{
			IntegrationType: &cxsdk.AlertDefIntegrationTypeIntegrationID{
				IntegrationId: wrapperspb.UInt32(*integrationID),
			},
		}
		wrapperspb.UInt32(*integrationID)
	} else if recipients := settings.Recipients; recipients != nil {
		advancedTargetSettings.Integration = &cxsdk.AlertDefIntegrationType{
			IntegrationType: &cxsdk.AlertDefIntegrationTypeRecipients{
				Recipients: &cxsdk.AlertDefRecipients{
					Emails: utils.StringSliceToWrappedStringSlice(recipients),
				},
			},
		}
	}

	advancedTargetSettings.RetriggeringPeriod = &cxsdk.AlertDefAdvancedTargetSettingsMinutes{
		Minutes: wrapperspb.UInt32(settings.RetriggeringPeriodMinutes),
	}

	return advancedTargetSettings
}

func expandIntegrations(integrations []IntegrationType) []*cxsdk.AlertDefIntegrationType {
	result := make([]*cxsdk.AlertDefIntegrationType, len(integrations))
	for i, integration := range integrations {
		result[i] = expandIntegration(integration)
	}
	return result
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
	} else if logsThreshold := definition.LogsThreshold; logsThreshold != nil {
		properties.TypeDefinition = expandLogsThreshold(logsThreshold)
	} // else if logsRatioThreshold := definition.LogsRatioThreshold; logsRatioThreshold != nil {
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
	}
}

func expandLogsThresholdRuleCondition(condition LogsThresholdRuleCondition) *cxsdk.LogsThresholdCondition {
	threshold, _ := strconv.ParseFloat(condition.Threshold, 64)
	return &cxsdk.LogsThresholdCondition{
		Threshold: wrapperspb.Double(threshold),
		TimeWindow: &cxsdk.LogsTimeWindow{
			Type: &cxsdk.LogsTimeWindowSpecificValue{
				LogsTimeWindowSpecificValue: LogsTimeWindowToProtoLogsTimeWindow[condition.LogsTimeWindow],
			},
		},
		ConditionType: LogsThresholdConditionTypeToProto[condition.LogsThresholdConditionType],
	}
}

func NewAlert() *Alert {
	return &Alert{
		Spec: AlertSpec{
			Labels: make(map[string]string),
		},
	}
}
