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
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/api/coralogix"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/utils"
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
		AlertPriorityP5: cxsdk.AlertDefPriorityP5OrUnspecified,
	}
	LogSeverityToProtoSeverity = map[LogSeverity]cxsdk.LogSeverity{
		LogSeverityDebug:    cxsdk.LogSeverityDebug,
		LogSeverityInfo:     cxsdk.LogSeverityInfo,
		LogSeverityWarning:  cxsdk.LogSeverityWarning,
		LogSeverityError:    cxsdk.LogSeverityError,
		LogSeverityCritical: cxsdk.LogSeverityCritical,
		LogSeverityVerbose:  cxsdk.LogSeverityVerboseUnspecified,
	}
	LogsFiltersOperationToProtoOperation = map[LogFilterOperationType]cxsdk.LogFilterOperationType{
		LogFilterOperationTypeIs:         cxsdk.LogFilterOperationIsOrUnspecified,
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
		LogsTimeWindow5Minutes:  cxsdk.LogsTimeWindowValue5MinutesOrUnspecified,
		LogsTimeWindow10Minutes: cxsdk.LogsTimeWindow10Minutes,
		LogsTimeWindow15Minutes: cxsdk.LogsTimeWindow15Minutes,
		LogsTimeWindow30Minutes: cxsdk.LogsTimeWindow30Minutes,
		LogsTimeWindowHour:      cxsdk.LogsTimeWindow1Hour,
		LogsTimeWindow2Hours:    cxsdk.LogsTimeWindow2Hours,
		LogsTimeWindow6Hours:    cxsdk.LogsTimeWindow6Hours,
		LogsTimeWindow12Hours:   cxsdk.LogsTimeWindow12Hours,
		LogsTimeWindow24Hours:   cxsdk.LogsTimeWindow24Hours,
		LogsTimeWindow36Hours:   cxsdk.LogsTimeWindow36Hours,
	}
	LogsThresholdConditionTypeToProto = map[LogsThresholdConditionType]cxsdk.LogsThresholdConditionType{
		LogsThresholdConditionTypeMoreThan: cxsdk.LogsThresholdConditionTypeMoreThanOrUnspecified,
		LogsThresholdConditionTypeLessThan: cxsdk.LogsThresholdConditionTypeLessThan,
	}
	LogsRatioTimeWindowToProto = map[LogsRatioTimeWindowValue]cxsdk.LogsRatioTimeWindowValue{
		LogsRatioTimeWindowMinutes5:  cxsdk.LogsRatioTimeWindowValue5MinutesOrUnspecified,
		LogsRatioTimeWindowMinutes10: cxsdk.LogsRatioTimeWindowValue10Minutes,
		LogsRatioTimeWindowMinutes15: cxsdk.LogsRatioTimeWindowValue15Minutes,
		LogsRatioTimeWindowMinutes30: cxsdk.LogsRatioTimeWindowValue30Minutes,
		LogsRatioTimeWindow1Hour:     cxsdk.LogsRatioTimeWindowValue1Hour,
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
		MetricThresholdConditionTypeLessThan:         cxsdk.MetricThresholdConditionTypeLessThan,
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
		TracingFilterOperationTypeIs:         cxsdk.TracingFilterOperationTypeIsOrUnspecified,
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
		MetricAnomalyConditionTypeMoreThanUsual: cxsdk.MetricAnomalyConditionTypeMoreThanOrUnspecified,
		MetricAnomalyConditionTypeLessThanUsual: cxsdk.MetricAnomalyConditionTypeLessThan,
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

// AlertSpec defines the desired state of a Coralogix Alert. For more info check - https://coralogix.com/docs/getting-started-with-coralogix-alerts/.
//
// Note that this is only for the latest version of the alerts API. If your account has been created before March 2025, make sure that your account has been migrated before using advanced features of alerts.
// +kubebuilder:validation:XValidation:rule="(self.alertType.logsImmediate == null && self.alertType.logsImmediate == null) || !has(self.groupByKeys)",message="groupByKeys is not supported for this alert type"
type AlertSpec struct {
	// Name of the alert
	//+kubebuilder:validation:MinLength=0
	Name string `json:"name"`

	// Description of the alert
	// +optional
	Description string `json:"description,omitempty"`

	// Priority of the alert.
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
}

func (a *Alert) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *Alert) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
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
	Minutes *uint32 `json:"minutes,omitempty"`
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

	// The destinations for notifications (Notification Center feature).
	// +optional
	Destinations []NotificationDestination `json:"destinations,omitempty"`

	// The router for notifications (Notification Center feature).
	// +optional
	Router *NotificationRouter `json:"router,omitempty"`
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
	Recipients []string `json:"recipients"`
}

// Reference to the integration.
// +kubebuilder:validation:XValidation:rule="has(self.backendRef) || has(self.resourceRef)",message="Exactly one of backendRef or resourceRef is required"
type IntegrationRef struct {

	// Backend reference for the outbound webhook.
	// +optional
	BackendRef *OutboundWebhookBackendRef `json:"backendRef,omitempty"`

	// Resource reference for use with the alert notification.
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef"`
}

// Outbound webhook reference.
// +kubebuilder:validation:XValidation:rule="has(self.id) != has(self.name)",message="One of id or name is required"
type OutboundWebhookBackendRef struct {
	// Webhook Id.
	// +optional
	ID *uint32 `json:"id,omitempty"`

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
	OutputSchemaId string `json:"outputSchemaId"`

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
// +kubebuilder:validation:XValidation:rule="(has(self.logsImmediate) ? 1 : 0) + (has(self.logsThreshold) ? 1 : 0) + (has(self.logsRatioThreshold) ? 1 : 0) + (has(self.logsTimeRelativeThreshold) ? 1 : 0) + (has(self.metricThreshold) ? 1 : 0) + (has(self.tracingThreshold) ? 1 : 0) + (has(self.tracingImmediate) ? 1 : 0) + (has(self.flow) ? 1 : 0) + (has(self.logsAnomaly) ? 1 : 0) + (has(self.metricAnomaly) ? 1 : 0) + (has(self.logsNewValue) ? 1 : 0) + (has(self.logsUniqueCount) ? 1 : 0) == 1", message="Exactly one of logsImmediate, logsThreshold, logsRatioThreshold, logsTimeRelativeThreshold, metricThreshold, tracingThreshold, tracingImmediate, flow, logsAnomaly, metricAnomaly, logsNewValue or logsUniqueCount must be set"
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

// +kubebuilder:validation:Enum={"5m","10m","15m","30m","1h","2h","6h","12h","24h","36h"}
// Logs time window type
type LogsTimeWindowValue string

// Logs time window values
const (
	LogsTimeWindow5Minutes  LogsTimeWindowValue = "5m"
	LogsTimeWindow10Minutes LogsTimeWindowValue = "10m"
	LogsTimeWindow15Minutes LogsTimeWindowValue = "15m"
	LogsTimeWindow30Minutes LogsTimeWindowValue = "30m"
	LogsTimeWindowHour      LogsTimeWindowValue = "1h"
	LogsTimeWindow2Hours    LogsTimeWindowValue = "2h"
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
	UndetectedValuesManagement *UndetectedValuesManagement `json:"undetectedValuesManagement"`
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
	// +optional
	SpecificValue MetricTimeWindowSpecificValue `json:"specificValue,omitempty"`
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
	MinNonNullValuesPct *uint32 `json:"minNonNullValuesPct,omitempty"`
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
	ForOverPct uint32 `json:"forOverPct"`

	// Time window to match within
	OfTheLast MetricAnomalyTimeWindow `json:"ofTheLast"`
	// +kubebuilder:validation:Maximum:=100
	// Replace with a number
	MinNonNullValuesPct uint32 `json:"minNonNullValuesPct"`
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

func (in AlertSpec) ExtractAlertProperties(listingAlertsAndWebhooksProperties *GetResourceRefProperties) (*cxsdk.AlertDefProperties, error) {
	notificationGroup, err := expandNotificationGroup(in.NotificationGroup, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand notification group: %w", err)
	}

	notificationGroupExcess, err := expandNotificationGroupExcess(in.NotificationGroupExcess, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand notification group excess: %w", err)
	}

	alertDefProperties := &cxsdk.AlertDefProperties{
		Name:                    wrapperspb.String(in.Name),
		Description:             wrapperspb.String(in.Description),
		Enabled:                 wrapperspb.Bool(in.Enabled),
		Priority:                AlertPriorityToProtoPriority[in.Priority],
		GroupByKeys:             coralogix.StringSliceToWrappedStringSlice(in.GroupByKeys),
		IncidentsSettings:       expandIncidentsSettings(in.IncidentsSettings),
		NotificationGroup:       notificationGroup,
		NotificationGroupExcess: notificationGroupExcess,
		EntityLabels:            in.EntityLabels,
		PhantomMode:             wrapperspb.Bool(in.PhantomMode),
		Schedule:                expandAlertSchedule(in.Schedule),
	}

	alertDefProperties, err = expandAlertTypeDefinition(alertDefProperties, in.TypeDefinition, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand alert type definition: %w", err)
	}

	return alertDefProperties, nil
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

func expandNotificationGroupExcess(excess []NotificationGroup, listingAlertsAndWebhooksProperties *GetResourceRefProperties) ([]*cxsdk.AlertDefNotificationGroup, error) {
	result := make([]*cxsdk.AlertDefNotificationGroup, len(excess))
	var errs error
	for _, group := range excess {
		ng, err := expandNotificationGroup(&group, listingAlertsAndWebhooksProperties)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to expand notification group: %w", err))
			continue
		}
		result = append(result, ng)
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func expandNotificationGroup(notificationGroup *NotificationGroup, listingAlertsAndWebhooksProperties *GetResourceRefProperties) (*cxsdk.AlertDefNotificationGroup, error) {
	if notificationGroup == nil {
		return nil, nil
	}

	webhooks, err := expandWebhooksSettings(notificationGroup.Webhooks, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand webhooks settings: %w", err)
	}

	var destinations []*cxsdk.NotificationDestination
	var router *cxsdk.NotificationRouter
	if notificationGroup.Destinations != nil {
		destinations, err = expandNotificationDestinations(notificationGroup.Destinations, listingAlertsAndWebhooksProperties)
		if err != nil {
			return nil, fmt.Errorf("failed to expand notification destinations: %w", err)
		}
	} else if notificationGroup.Router != nil {
		notifyOn := NotifyOnToProtoNotifyOn[notificationGroup.Router.NotifyOn]
		router = &cxsdk.NotificationRouter{
			Id:       "router_default",
			NotifyOn: &notifyOn,
		}
	}

	return &cxsdk.AlertDefNotificationGroup{
		GroupByKeys:  coralogix.StringSliceToWrappedStringSlice(notificationGroup.GroupByKeys),
		Webhooks:     webhooks,
		Destinations: destinations,
		Router:       router,
	}, nil
}

func expandWebhooksSettings(webhooksSettings []WebhookSettings, listingAlertsAndWebhooksProperties *GetResourceRefProperties) ([]*cxsdk.AlertDefWebhooksSettings, error) {
	result := make([]*cxsdk.AlertDefWebhooksSettings, len(webhooksSettings))
	var errs error
	for i, setting := range webhooksSettings {
		expandedWebhookSetting, err := expandWebhookSetting(setting, listingAlertsAndWebhooksProperties)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to expand webhook setting: %w", err))
			continue
		}
		result[i] = expandedWebhookSetting
	}

	if errs != nil {
		return nil, errs
	}
	return result, nil
}

func expandWebhookSetting(webhooksSetting WebhookSettings, listingAlertsAndWebhooksProperties *GetResourceRefProperties) (*cxsdk.AlertDefWebhooksSettings, error) {
	notifyOn := NotifyOnToProtoNotifyOn[webhooksSetting.NotifyOn]
	integration, err := expandIntegration(webhooksSetting.Integration, listingAlertsAndWebhooksProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to expand integration: %w", err)
	}
	return &cxsdk.AlertDefWebhooksSettings{
		NotifyOn:    &notifyOn,
		Integration: integration,
		RetriggeringPeriod: &cxsdk.AlertDefWebhooksSettingsMinutes{
			Minutes: wrapperspb.UInt32(*webhooksSetting.RetriggeringPeriod.Minutes),
		},
	}, nil
}

func expandIntegration(integration IntegrationType, listingWebhooksProperties *GetResourceRefProperties) (*cxsdk.AlertDefIntegrationType, error) {
	if integrationRef := integration.IntegrationRef; integrationRef != nil {
		var integrationID *wrapperspb.UInt32Value
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
				integrationID = wrapperspb.UInt32(*id)
			} else if name := backendRef.Name; name != nil {
				integrationID, err = convertNameToIntegrationID(*name, listingWebhooksProperties)
				if err != nil {
					return nil, fmt.Errorf("failed to convert name to integration ID: %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("integration type not found")
		}

		return &cxsdk.AlertDefIntegrationType{
			IntegrationType: &cxsdk.AlertDefIntegrationTypeIntegrationID{
				IntegrationId: integrationID,
			},
		}, nil
	} else if recipients := integration.Recipients; recipients != nil {
		return &cxsdk.AlertDefIntegrationType{
			IntegrationType: &cxsdk.AlertDefIntegrationTypeRecipients{
				Recipients: &cxsdk.AlertDefRecipients{
					Emails: coralogix.StringSliceToWrappedStringSlice(recipients),
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("integration type not found")
}

func convertNameToIntegrationID(name string, properties *GetResourceRefProperties) (*wrapperspb.UInt32Value, error) {
	if properties.WebhookNameToId == nil {
		if err := fillWebhookNameToId(properties); err != nil {
			return nil, err
		}
	}

	id, ok := properties.WebhookNameToId[name]
	if !ok {
		return nil, fmt.Errorf("webhook %s not found", name)
	}

	return wrapperspb.UInt32(id), nil
}

func fillWebhookNameToId(properties *GetResourceRefProperties) error {
	log, client, ctx := properties.Log, properties.Clientset.Webhooks(), properties.Ctx
	log.V(1).Info("Listing webhooks from the backend")
	webhooks, err := client.List(ctx, &cxsdk.ListAllOutgoingWebhooksRequest{})
	if err != nil {
		return err
	}

	properties.WebhookNameToId = make(map[string]uint32)
	for _, webhook := range webhooks.Deployed {
		properties.WebhookNameToId[webhook.Name.Value] = webhook.ExternalId.Value
	}

	return nil
}

func expandNotificationDestinations(destinations []NotificationDestination, properties *GetResourceRefProperties) ([]*cxsdk.NotificationDestination, error) {
	var result []*cxsdk.NotificationDestination
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

		triggeredRoutingOverrides, err := expandRoutingOverrides(destination.TriggeredRoutingOverrides)
		if err != nil {
			return nil, fmt.Errorf("failed to expand triggered routing overrides: %w", err)
		}

		var resolvedRoutingOverrides *cxsdk.SourceOverrides
		if destination.ResolvedRoutingOverrides != nil {
			resolvedRoutingOverrides, err = expandRoutingOverrides(*destination.ResolvedRoutingOverrides)
			if err != nil {
				return nil, fmt.Errorf("failed to expand resolved routing overrides: %w", err)
			}
		}

		notificationDestination := &cxsdk.NotificationDestination{
			ConnectorId: connectorId,
			PresetId:    presetId,
			NotifyOn:    NotifyOnToProtoNotifyOn[destination.NotifyOn],
			TriggeredRoutingOverrides: &cxsdk.NotificationRouting{
				ConfigOverrides: triggeredRoutingOverrides,
			},
			ResolvedRouteOverrides: &cxsdk.NotificationRouting{
				ConfigOverrides: resolvedRoutingOverrides,
			},
		}

		result = append(result, notificationDestination)
	}

	return result, nil
}

func expandRoutingOverrides(overrides NotificationRouting) (*cxsdk.SourceOverrides, error) {
	if overrides.ConfigOverrides == nil {
		return nil, nil
	}

	connectorOverrides := extractConnectorOverrides(overrides.ConfigOverrides.ConnectorConfigFields)
	presetOverrides := extractPresetOverrides(overrides.ConfigOverrides.MessageConfigFields)

	sourceOverrides := &cxsdk.SourceOverrides{
		ConnectorConfigFields: connectorOverrides,
		MessageConfigFields:   presetOverrides,
		OutputSchemaId:        overrides.ConfigOverrides.OutputSchemaId,
	}

	return sourceOverrides, nil
}

func extractConnectorOverrides(overrides []ConfigField) []*cxsdk.AlertsConnectorConfigField {
	var result []*cxsdk.AlertsConnectorConfigField
	for _, override := range overrides {
		result = append(result, &cxsdk.AlertsConnectorConfigField{
			FieldName: override.FieldName,
			Template:  override.Template,
		})
	}

	return result
}

func extractPresetOverrides(overrides []ConfigField) []*cxsdk.AlertsMessageConfigField {
	var result []*cxsdk.AlertsMessageConfigField
	for _, override := range overrides {
		result = append(result, &cxsdk.AlertsMessageConfigField{
			FieldName: override.FieldName,
			Template:  override.Template,
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

func expandAlertSchedule(alertSchedule *AlertSchedule) *cxsdk.AlertDefPropertiesActiveOn {
	if alertSchedule == nil {
		return nil
	}

	utc := extractUTC(alertSchedule.TimeZone)
	daysOfWeek := expandDaysOfWeek(alertSchedule.ActiveOn.DayOfWeek)
	start := expandTime(alertSchedule.ActiveOn.StartTime)
	end := expandTime(alertSchedule.ActiveOn.EndTime)

	start, end, daysOfWeek = convertTimeFramesToGMT(start, end, daysOfWeek, utc)

	return &cxsdk.AlertDefPropertiesActiveOn{
		ActiveOn: &cxsdk.AlertsActivitySchedule{
			DayOfWeek: daysOfWeek,
			StartTime: start,
			EndTime:   end,
		},
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

func expandTime(time *TimeOfDay) *cxsdk.AlertTimeOfDay {
	if time == nil {
		return nil
	}

	timeArr := strings.Split(string(*time), ":")
	hours, _ := strconv.Atoi(timeArr[0])
	minutes, _ := strconv.Atoi(timeArr[1])

	return &cxsdk.AlertTimeOfDay{
		Hours:   int32(hours),
		Minutes: int32(minutes),
	}
}

func convertTimeFramesToGMT(start, end *cxsdk.AlertTimeOfDay, daysOfWeek []cxsdk.AlertDayOfWeek, utc int32) (*cxsdk.AlertTimeOfDay, *cxsdk.AlertTimeOfDay, []cxsdk.AlertDayOfWeek) {
	daysOfWeekOffset := daysOfWeekOffsetToGMT(start, utc)
	start.Hours = convertUtcToGmt(start.GetHours(), utc)
	end.Hours = convertUtcToGmt(end.GetHours(), utc)
	if daysOfWeekOffset != 0 {
		for i, d := range daysOfWeek {
			daysOfWeek[i] = cxsdk.AlertDayOfWeek((int32(d) + daysOfWeekOffset) % 7)
		}
	}

	return start, end, daysOfWeek
}

func daysOfWeekOffsetToGMT(start *cxsdk.AlertTimeOfDay, utc int32) int32 {
	daysOfWeekOffset := int32(start.Hours-utc) / 24
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

func expandDaysOfWeek(week []DayOfWeek) []cxsdk.AlertDayOfWeek {
	result := make([]cxsdk.AlertDayOfWeek, len(week))
	for i, d := range week {
		result[i] = DaysOfWeekToProtoDayOfWeek[d]
	}

	return result
}

func expandAlertTypeDefinition(properties *cxsdk.AlertDefProperties, definition AlertTypeDefinition, listingWebhooksProperties *GetResourceRefProperties) (*cxsdk.AlertDefProperties, error) {
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
	} else if tracingImmediate := definition.TracingImmediate; tracingImmediate != nil {
		properties.TypeDefinition = expandTracingImmediate(tracingImmediate)
		properties.Type = cxsdk.AlertDefTypeTracingImmediate
	} else if flow := definition.Flow; flow != nil {
		typeDefinition, err := expandFlow(listingWebhooksProperties, flow)
		if err != nil {
			return nil, err
		}
		properties.TypeDefinition = typeDefinition
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
	} else {
		return nil, fmt.Errorf("unsupported alert type definition")
	}

	return properties, nil
}

func expandLogsUniqueCount(uniqueCount *LogsUniqueCount) *cxsdk.AlertDefPropertiesLogsUniqueCount {
	return &cxsdk.AlertDefPropertiesLogsUniqueCount{
		LogsUniqueCount: &cxsdk.LogsUniqueCountType{
			LogsFilter:                  expandLogsFilter(uniqueCount.LogsFilter),
			Rules:                       expandLogsUniqueCountRules(uniqueCount.Rules),
			NotificationPayloadFilter:   coralogix.StringSliceToWrappedStringSlice(uniqueCount.NotificationPayloadFilter),
			MaxUniqueCountPerGroupByKey: wrapperspb.Int64(int64(*uniqueCount.MaxUniqueCountPerGroupByKey)),
			UniqueCountKeypath:          wrapperspb.String(uniqueCount.UniqueCountKeypath),
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
			NotificationPayloadFilter: coralogix.StringSliceToWrappedStringSlice(logsNewValue.NotificationPayloadFilter),
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
			LogsNewValueTimeWindowSpecificValue: LogsNewValueTimeWindowValueToProto[timeWindow.SpecificValue],
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

func expandMetricAnomalyRules(rules []MetricAnomalyRule) []*cxsdk.MetricAnomalyRule {
	result := make([]*cxsdk.MetricAnomalyRule, len(rules))
	for i := range rules {
		result[i] = expandMetricAnomalyRule(rules[i])
	}

	return result
}

func expandMetricAnomalyRule(rule MetricAnomalyRule) *cxsdk.MetricAnomalyRule {
	return &cxsdk.MetricAnomalyRule{
		Condition: expandMetricAnomalyCondition(rule.Condition),
	}
}

func expandMetricAnomalyCondition(condition MetricAnomalyCondition) *cxsdk.MetricAnomalyCondition {
	return &cxsdk.MetricAnomalyCondition{
		Threshold:           wrapperspb.Double(condition.Threshold.AsApproximateFloat64()),
		ForOverPct:          wrapperspb.UInt32(condition.ForOverPct),
		OfTheLast:           expandAnomalyMetricTimeWindow(condition.OfTheLast),
		MinNonNullValuesPct: wrapperspb.UInt32(condition.MinNonNullValuesPct),
		ConditionType:       MetricAnomalyConditionTypeToProto[condition.ConditionType],
	}
}

func expandLogsAnomaly(anomaly *LogsAnomaly) *cxsdk.AlertDefPropertiesLogsAnomaly {
	return &cxsdk.AlertDefPropertiesLogsAnomaly{
		LogsAnomaly: &cxsdk.LogsAnomalyType{
			LogsFilter:                expandLogsFilter(anomaly.LogsFilter),
			Rules:                     expandLogsAnomalyRules(anomaly.Rules),
			NotificationPayloadFilter: coralogix.StringSliceToWrappedStringSlice(anomaly.NotificationPayloadFilter),
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

func expandFlow(listingAlertsProperties *GetResourceRefProperties, flow *Flow) (*cxsdk.AlertDefPropertiesFlow, error) {
	stages, err := expandFlowStages(listingAlertsProperties, flow.Stages)
	if err != nil {
		return nil, err
	}
	return &cxsdk.AlertDefPropertiesFlow{
		Flow: &cxsdk.FlowType{
			Stages:             stages,
			EnforceSuppression: wrapperspb.Bool(flow.EnforceSuppression),
		},
	}, nil
}

func expandFlowStages(listingAlertsProperties *GetResourceRefProperties, stages []FlowStage) ([]*cxsdk.FlowStages, error) {
	result := make([]*cxsdk.FlowStages, len(stages))
	var errs error
	for i, stage := range stages {
		flowStage, err := expandFlowStage(listingAlertsProperties, stage)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result[i] = flowStage
	}

	return result, errs
}

func expandFlowStage(listingAlertsProperties *GetResourceRefProperties, stage FlowStage) (*cxsdk.FlowStages, error) {
	flowStages, err := expandFlowStagesType(listingAlertsProperties, stage.FlowStagesType)
	if err != nil {
		return nil, err
	}

	return &cxsdk.FlowStages{
		FlowStages:    flowStages,
		TimeframeMs:   wrapperspb.Int64(stage.TimeframeMs),
		TimeframeType: TimeframeTypeToProto[stage.TimeframeType],
	}, nil
}

func expandFlowStagesType(listingAlertsProperties *GetResourceRefProperties, stagesType FlowStagesType) (*cxsdk.FlowStagesGroups, error) {
	groups, err := expandFlowStagesGroups(listingAlertsProperties, stagesType.Groups)
	if err != nil {
		return nil, err
	}

	return &cxsdk.FlowStagesGroups{
		FlowStagesGroups: &cxsdk.FlowStagesGroupsValue{
			Groups: groups,
		},
	}, nil
}

func expandFlowStagesGroups(listingAlertsProperties *GetResourceRefProperties, groups []FlowStageGroup) ([]*cxsdk.FlowStagesGroup, error) {
	result := make([]*cxsdk.FlowStagesGroup, len(groups))
	var errs error
	for i, group := range groups {
		expandedGroup, err := expandFlowStagesGroup(listingAlertsProperties, group)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result[i] = expandedGroup
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func expandFlowStagesGroup(listingWebhooksProperties *GetResourceRefProperties, group FlowStageGroup) (*cxsdk.FlowStagesGroup, error) {
	alertDefs, err := expandFlowStagesGroupsAlertDefs(listingWebhooksProperties, group.AlertDefs)
	if err != nil {
		return nil, err
	}

	return &cxsdk.FlowStagesGroup{
		AlertDefs: alertDefs,
		NextOp:    FlowStageGroupNextOpToProto[group.NextOp],
		AlertsOp:  FlowStageGroupAlertsOpToProto[group.AlertsOp],
	}, nil
}

func expandFlowStagesGroupsAlertDefs(listingAlertsProperties *GetResourceRefProperties, alertDefs []FlowStagesGroupsAlertDefs) ([]*cxsdk.FlowStagesGroupsAlertDefs, error) {
	result := make([]*cxsdk.FlowStagesGroupsAlertDefs, len(alertDefs))
	var errs error
	for i := range alertDefs {
		expandedAlertDef, err := expandFlowStagesGroupsAlertDef(listingAlertsProperties, alertDefs[i])
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result[i] = expandedAlertDef
	}

	return result, nil
}

func expandFlowStagesGroupsAlertDef(listingAlertsProperties *GetResourceRefProperties, defs FlowStagesGroupsAlertDefs) (*cxsdk.FlowStagesGroupsAlertDefs, error) {
	id, err := expandAlertRef(listingAlertsProperties, defs.AlertRef)
	if err != nil {
		return nil, err
	}

	return &cxsdk.FlowStagesGroupsAlertDefs{
		Id:  id,
		Not: wrapperspb.Bool(defs.Not),
	}, nil
}

func expandAlertRef(listingAlertsProperties *GetResourceRefProperties, ref AlertRef) (*wrapperspb.StringValue, error) {
	if backendRef := ref.BackendRef; backendRef != nil {
		if id := backendRef.ID; id != nil {
			return wrapperspb.String(*id), nil
		} else if name := backendRef.Name; name != nil {
			return convertAlertNameToID(listingAlertsProperties, *name)
		}
	} else if resourceRef := ref.ResourceRef; resourceRef != nil {
		if namespace := resourceRef.Namespace; namespace != nil {
			listingAlertsProperties.Namespace = *namespace
		}
		return convertAlertCrNameToID(listingAlertsProperties, resourceRef.Name)
	}

	return nil, fmt.Errorf("alert ref not found")
}

func convertAlertCrNameToID(listingAlertsProperties *GetResourceRefProperties, alertCrName string) (*wrapperspb.StringValue, error) {
	ctx, namespace := listingAlertsProperties.Ctx, listingAlertsProperties.Namespace
	alertCR := &Alert{}
	err := config.GetClient().Get(ctx, client.ObjectKey{Name: alertCrName, Namespace: namespace}, alertCR)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert %w", err)
	}

	if alertCR.Status.ID == nil {
		return nil, fmt.Errorf("alert with name %s has no ID", alertCrName)
	}

	return wrapperspb.String(*alertCR.Status.ID), nil
}

func convertAlertNameToID(listingAlertsProperties *GetResourceRefProperties, alertName string) (*wrapperspb.StringValue, error) {
	if listingAlertsProperties.AlertNameToId == nil {
		listingAlertsProperties.AlertNameToId = make(map[string]string)
		log, alertsClient, ctx := listingAlertsProperties.Log, listingAlertsProperties.Clientset.Alerts(), listingAlertsProperties.Ctx
		log.V(1).Info("Listing all alerts")
		listAlertsResp, err := alertsClient.List(ctx, &cxsdk.ListAlertDefsRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to list all alerts %w", err)
		}

		for _, alert := range listAlertsResp.GetAlertDefs() {
			listingAlertsProperties.AlertNameToId[alert.AlertDefProperties.Name.Value] = alert.GetId().GetValue()
		}
	}

	alertID, ok := listingAlertsProperties.AlertNameToId[alertName]
	if !ok {
		return nil, fmt.Errorf("alert with name %s not found", alertName)
	}

	return wrapperspb.String(alertID), nil
}

func expandTracingThreshold(tracingThreshold *TracingThreshold) *cxsdk.AlertDefPropertiesTracingThreshold {
	return &cxsdk.AlertDefPropertiesTracingThreshold{
		TracingThreshold: &cxsdk.TracingThresholdType{
			TracingFilter:             expandTracingFilter(tracingThreshold.TracingFilter),
			Rules:                     expandTracingThresholdRules(tracingThreshold.Rules),
			NotificationPayloadFilter: coralogix.StringSliceToWrappedStringSlice(tracingThreshold.NotificationPayloadFilter),
		},
	}
}

func expandTracingImmediate(tracingImmediate *TracingImmediate) *cxsdk.AlertDefPropertiesTracingImmediate {
	return &cxsdk.AlertDefPropertiesTracingImmediate{
		TracingImmediate: &cxsdk.TracingImmediateType{
			TracingFilter:             expandTracingFilter(tracingImmediate.TracingFilter),
			NotificationPayloadFilter: coralogix.StringSliceToWrappedStringSlice(tracingImmediate.NotificationPayloadFilter),
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
		Values:    coralogix.StringSliceToWrappedStringSlice(filterType.Values),
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
			TracingTimeWindowValue: TracingTimeWindowSpecificValueToProto[timeWindow.SpecificValue],
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
	if specificValue := timeWindow.SpecificValue; specificValue != nil {
		return &cxsdk.MetricTimeWindow{
			Type: &cxsdk.MetricTimeWindowSpecificValue{
				MetricTimeWindowSpecificValue: MetricTimeWindowToProto[*specificValue],
			},
		}
	} else if dynamicTimeWindow := timeWindow.DynamicDuration; dynamicTimeWindow != nil {
		return &cxsdk.MetricTimeWindow{
			Type: &cxsdk.MetricTimeWindowDynamicDuration{
				MetricTimeWindowDynamicDuration: wrapperspb.String(*dynamicTimeWindow),
			},
		}
	}

	return nil
}

func expandAnomalyMetricTimeWindow(timeWindow MetricAnomalyTimeWindow) *cxsdk.MetricTimeWindow {
	return &cxsdk.MetricTimeWindow{
		Type: &cxsdk.MetricTimeWindowSpecificValue{
			MetricTimeWindowSpecificValue: MetricTimeWindowToProto[timeWindow.SpecificValue],
		},
	}
}

func expandMetricMissingValues(missingValues *MetricMissingValues) *cxsdk.MetricMissingValues {
	if missingValues == nil {
		return nil
	} else if missingValues.ReplaceWithZero {
		return &cxsdk.MetricMissingValues{
			MissingValues: &cxsdk.MetricMissingValuesReplaceWithZero{
				ReplaceWithZero: wrapperspb.Bool(true),
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
			NotificationPayloadFilter: coralogix.StringSliceToWrappedStringSlice(immediate.NotificationPayloadFilter),
		},
	}
}

func expandLogsThreshold(logsThreshold *LogsThreshold) *cxsdk.AlertDefPropertiesLogsThreshold {
	return &cxsdk.AlertDefPropertiesLogsThreshold{
		LogsThreshold: &cxsdk.LogsThresholdType{
			LogsFilter:                 expandLogsFilter(logsThreshold.LogsFilter),
			UndetectedValuesManagement: expandUndetectedValuesManagement(logsThreshold.UndetectedValuesManagement),
			Rules:                      expandLogsThresholdRules(logsThreshold.Rules),
			NotificationPayloadFilter:  coralogix.StringSliceToWrappedStringSlice(logsThreshold.NotificationPayloadFilter),
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
			NotificationPayloadFilter:  coralogix.StringSliceToWrappedStringSlice(threshold.NotificationPayloadFilter),
			UndetectedValuesManagement: expandUndetectedValuesManagement(threshold.UndetectedValuesManagement),
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
			LogsRatioTimeWindowSpecificValue: LogsRatioTimeWindowToProto[timeWindow.SpecificValue],
		},
	}
}

func expandAlertOverride(override *AlertOverride) *cxsdk.AlertDefPriorityOverride {
	if override == nil {
		return &cxsdk.AlertDefPriorityOverride{}
	}

	return &cxsdk.AlertDefPriorityOverride{
		Priority: AlertPriorityToProtoPriority[override.Priority],
	}
}

func expandLogsFilter(filter *LogsFilter) *cxsdk.LogsFilter {
	if filter == nil {
		return nil
	}

	return &cxsdk.LogsFilter{
		FilterType: expandSimpleFilter(filter.SimpleFilter),
	}
}

func expandSimpleFilter(filter LogsSimpleFilter) *cxsdk.LogsFilterSimpleFilter {
	return &cxsdk.LogsFilterSimpleFilter{
		SimpleFilter: &cxsdk.SimpleFilter{
			LuceneQuery:  coralogix.StringPointerToWrapperspbString(filter.LuceneQuery),
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

// +k8s:deepcopy-gen=false
type GetResourceRefProperties struct {
	Ctx             context.Context
	Log             logr.Logger
	AlertNameToId   map[string]string
	WebhookNameToId map[string]uint32
	Clientset       *cxsdk.ClientSet
	Namespace       string
}

func convertCRNameToIntegrationID(name string, properties *GetResourceRefProperties) (*wrapperspb.UInt32Value, error) {
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

	return wrapperspb.UInt32(uint32(externalIDInt)), nil
}
