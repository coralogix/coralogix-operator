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
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/config"
)

// AlertSchedulerSpec defines the desired state Coralogix AlertScheduler.
type AlertSchedulerSpec struct {
	// Alert Scheduler name.
	Name string `json:"name"`

	// Alert Scheduler description.
	// +optional
	Description string `json:"description,omitempty"`

	//+kubebuilder:default=true
	// Alert Scheduler enabled. If set to `false`, the alert scheduler will be disabled. True by default.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Alert Scheduler meta labels.
	// +optional
	MetaLabels []MetaLabel `json:"metaLabels,omitempty"`

	// Alert Scheduler filter. Exactly one of `metaLabels` or `alerts` can be set.
	// If none of them set, all alerts will be affected.
	Filter Filter `json:"filter"`

	// Alert Scheduler schedule. Exactly one of `oneTime` or `recurring` must be set.
	Schedule Schedule `json:"schedule"`
}

// +kubebuilder:validation:XValidation:rule="has(self.metaLabels) != has(self.alerts)",message="Exactly one of metaLabels or alerts must be set"
type Filter struct {
	// DataPrime query expression - https://coralogix.com/docs/dataprime-query-language.
	WhatExpression string `json:"whatExpression"`

	// Alert Scheduler meta labels. Conflicts with `alerts`.
	// +optional
	MetaLabels []MetaLabel `json:"metaLabels,omitempty"`

	// Alert references. Conflicts with `metaLabels`.
	// +optional
	Alerts []AlertRef `json:"alerts,omitempty"`
}

type AlertRef struct {
	// Alert custom resource name and namespace. If namespace is not set, the AlertScheduler namespace will be used.
	ResourceRef *ResourceRef `json:"resourceRef"`
}

type MetaLabel struct {
	Key string `json:"key"`

	// +optional
	Value *string `json:"value,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.oneTime) != has(self.recurring)",message="Exactly one of oneTime or recurring must be set"
type Schedule struct {
	// The operation to perform. Can be `mute` or `activate`.
	// +kubebuilder:validation:Enum=mute;activate
	Operation string `json:"operation"`

	// One-time schedule. Conflicts with `recurring`.
	// +optional
	OneTime *TimeFrame `json:"oneTime,omitempty"`

	// Recurring schedule. Conflicts with `oneTime`.
	// +optional
	Recurring *Recurring `json:"recurring,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.endTime) != has(self.duration)",message="Exactly one of endTime or duration must be set"
type TimeFrame struct {
	// The start time of the time frame. In isodate format. For example, `2021-01-01T00:00:00.000`.
	// +kubebuilder:validation:Pattern=`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}$`
	StartTime string `json:"startTime"`

	// The end time of the time frame. In isodate format. For example, `2021-01-01T00:00:00.000`.
	// Conflicts with `duration`.
	// +optional
	// +kubebuilder:validation:Pattern=`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}$`
	EndTime *string `json:"endTime,omitempty"`

	// The duration from the start time to wait before the operation is performed.
	// Conflicts with `endTime`.
	// +optional
	Duration *Duration `json:"duration,omitempty"`

	// The timezone of the time frame. For example, `UTC-4` or `UTC+10`.
	// +kubebuilder:validation:Pattern=`^UTC[+-]\d{1,2}$`
	Timezone string `json:"timezone"`
}

// +kubebuilder:validation:XValidation:rule="has(self.always) != has(self.dynamic)",message="Exactly one of always or dynamic must be set"
type Recurring struct {
	// Recurring always.
	// +optional
	Always *Always `json:"always,omitempty"`

	// Dynamic schedule.
	// +optional
	Dynamic *Dynamic `json:"dynamic,omitempty"`
}

type Always struct{}

type Dynamic struct {
	// The rule will be activated in a recurring mode according to the interval.
	RepeatEvery int32 `json:"repeatEvery"`

	// The rule will be activated in a recurring mode (daily, weekly or monthly).
	Frequency *Frequency `json:"frequency"`

	// The time frame of the rule.
	TimeFrame *TimeFrame `json:"timeFrame"`

	// The termination date of the rule.
	// +optional
	// +kubebuilder:validation:Pattern=`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}$`
	TerminationDate *string `json:"terminationDate,omitempty"`
}

type Frequency struct {
	// +optional
	Daily *Daily `json:"daily,omitempty"`

	// +optional
	Weekly *Weekly `json:"weekly,omitempty"`

	// +optional
	Monthly *Monthly `json:"monthly,omitempty"`
}

type Daily struct{}

type Weekly struct {
	// The days of the week to activate the rule.
	Days []Day `json:"days"`
}

// +kubebuilder:validation:Enum=Sunday;Monday;Tuesday;Wednesday;Thursday;Friday;Saturday;
type Day string

type Monthly struct {
	// The days of the month to activate the rule.
	Days []int32 `json:"days"`
}

type Duration struct {
	// The number of time units to wait before the alert is triggered. For example,
	// if the frequency is set to `hours` and the value is set to `2`, the alert will be triggered after 2 hours.
	ForOver int32 `json:"forOver"`

	// The time unit to wait before the alert is triggered. Can be `minutes`, `hours` or `days`.
	// +kubebuilder:validation:Enum=minutes;hours;days
	Frequency string `json:"frequency"`
}

// AlertSchedulerStatus defines the observed state of AlertScheduler.
type AlertSchedulerStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	PrintableStatus string `json:"printableStatus,omitempty"`
}

var (
	schemaToProtoScheduleOperation = map[string]cxsdk.ScheduleOperation{
		"activate": cxsdk.ScheduleOperationActivate,
		"mute":     cxsdk.ScheduleOperationMute,
	}
	schemaToProtoDurationFrequency = map[string]cxsdk.DurationFrequency{
		"minutes": cxsdk.DurationFrequencyMinute,
		"hours":   cxsdk.DurationFrequencyHour,
		"days":    cxsdk.DurationFrequencyDay,
	}
	daysToProtoValue = map[Day]int32{
		"Sunday":    1,
		"Monday":    2,
		"Tuesday":   3,
		"Wednesday": 4,
		"Thursday":  5,
		"Friday":    6,
		"Saturday":  7,
	}
)

func (a *AlertScheduler) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *AlertScheduler) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
}

func (a *AlertScheduler) GetPrintableStatus() string {
	return a.Status.PrintableStatus
}

func (a *AlertScheduler) SetPrintableStatus(printableStatus string) {
	a.Status.PrintableStatus = printableStatus
}

func (a *AlertScheduler) HasIDInStatus() bool {
	return a.Status.ID != nil && *a.Status.ID != ""
}

func (a *AlertScheduler) ExtractCreateAlertSchedulerRequest() (*cxsdk.CreateAlertSchedulerRuleRequest, error) {
	alertScheduler, err := a.extractAlertScheduler()
	if err != nil {
		return nil, fmt.Errorf("error on extracting alert scheduler: %w", err)
	}

	return &cxsdk.CreateAlertSchedulerRuleRequest{
		AlertSchedulerRule: alertScheduler,
	}, nil
}

func (a *AlertScheduler) ExtractUpdateAlertSchedulerRequest() (*cxsdk.UpdateAlertSchedulerRuleRequest, error) {
	alertScheduler, err := a.extractAlertScheduler()
	if err != nil {
		return nil, fmt.Errorf("error on extracting alert scheduler: %w", err)
	}

	alertScheduler.UniqueIdentifier = a.Status.ID
	return &cxsdk.UpdateAlertSchedulerRuleRequest{
		AlertSchedulerRule: alertScheduler,
	}, nil
}

func (a *AlertScheduler) extractAlertScheduler() (*cxsdk.AlertSchedulerRule, error) {
	metaLabels := extractMetaLabels(a.Spec.MetaLabels)
	filter, err := a.extractFilter()
	if err != nil {
		return nil, fmt.Errorf("error on extracting filter: %w", err)
	}

	schedule, err := a.extractSchedule()
	if err != nil {
		return nil, fmt.Errorf("error on extracting schedule: %w", err)
	}

	return &cxsdk.AlertSchedulerRule{
		Name:        a.Spec.Name,
		Description: ptr.To(a.Spec.Description),
		MetaLabels:  metaLabels,
		Filter:      filter,
		Schedule:    schedule,
		Enabled:     a.Spec.Enabled,
	}, nil
}

func extractMetaLabels(metaLabels []MetaLabel) []*cxsdk.MetaLabel {
	var result []*cxsdk.MetaLabel
	for _, ml := range metaLabels {
		result = append(result, &cxsdk.MetaLabel{
			Key:   ml.Key,
			Value: ml.Value,
		})
	}
	return result
}

func (a *AlertScheduler) extractFilter() (*cxsdk.AlertSchedulerFilter, error) {
	if a.Spec.Filter.MetaLabels != nil {
		metaLabels := extractMetaLabels(a.Spec.Filter.MetaLabels)
		return &cxsdk.AlertSchedulerFilter{
			WhatExpression: a.Spec.Filter.WhatExpression,
			WhichAlerts: &cxsdk.AlertSchedulerFilterMetaLabels{
				AlertMetaLabels: &cxsdk.MetaLabels{
					Value: metaLabels,
				},
			},
		}, nil
	} else if a.Spec.Filter.Alerts != nil {
		alertsIds, err := a.extractAlertsIds()
		if err != nil {
			return nil, fmt.Errorf("error on extracting alerts ids: %w", err)
		}

		return &cxsdk.AlertSchedulerFilter{
			WhatExpression: a.Spec.Filter.WhatExpression,
			WhichAlerts: &cxsdk.AlertSchedulerFilterUniqueIDs{
				AlertUniqueIds: &cxsdk.AlertUniqueIDs{
					Value: alertsIds,
				},
			},
		}, nil
	}

	return nil, nil
}

func (a *AlertScheduler) extractAlertsIds() ([]string, error) {
	var result []string
	var errs error

	for _, alert := range a.Spec.Filter.Alerts {
		id, err := extractAlertId(alert, a.Namespace)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result = append(result, id)
	}

	if errs != nil {
		return nil, errs
	}

	return result, nil
}

func extractAlertId(alert AlertRef, schedulerNamespace string) (string, error) {
	var namespace string
	if alert.ResourceRef != nil && alert.ResourceRef.Namespace != nil {
		namespace = *alert.ResourceRef.Namespace
	} else {
		namespace = schedulerNamespace
	}

	a := &v1beta1.Alert{}
	err := config.GetClient().Get(context.Background(),
		client.ObjectKey{Name: alert.ResourceRef.Name, Namespace: namespace}, a)
	if err != nil {
		return "", err
	}

	if !config.GetConfig().Selector.Matches(a.Labels, a.Namespace) {
		return "", fmt.Errorf("alert %s does not match selector", a.Name)
	}

	if a.Status.ID == nil {
		return "", fmt.Errorf("ID is not populated for alert %s", a.Name)
	}

	return *a.Status.ID, nil
}

func (a *AlertScheduler) extractSchedule() (*cxsdk.Schedule, error) {
	schedule := &cxsdk.Schedule{
		ScheduleOperation: schemaToProtoScheduleOperation[a.Spec.Schedule.Operation],
	}

	if a.Spec.Schedule.OneTime != nil {
		oneTime, err := a.extractOneTime()
		if err != nil {
			return nil, fmt.Errorf("error on extracting one time schedule: %w", err)
		}
		schedule.Scheduler = oneTime
		return schedule, nil
	} else if a.Spec.Schedule.Recurring != nil {
		recurring, err := a.extractRecurring()
		if err != nil {
			return nil, fmt.Errorf("error on extracting recurring schedule: %w", err)
		}
		schedule.Scheduler = recurring
		return schedule, nil
	}

	return nil, fmt.Errorf("exactly one of `oneTime` or `recurring` must be set")
}

func (a *AlertScheduler) extractOneTime() (*cxsdk.ScheduleOneTime, error) {
	timeFrame, err := extractTimeFrame(a.Spec.Schedule.OneTime)
	if err != nil {
		return nil, fmt.Errorf("error on extracting time frame: %w", err)
	}

	return &cxsdk.ScheduleOneTime{
		OneTime: &cxsdk.OneTime{
			Timeframe: timeFrame,
		},
	}, nil
}

func (a *AlertScheduler) extractRecurring() (*cxsdk.ScheduleRecurring, error) {
	if a.Spec.Schedule.Recurring.Dynamic != nil {
		dynamic, err := a.extractDynamic()
		if err != nil {
			return nil, fmt.Errorf("error on extracting dynamic schedule: %w", err)
		}

		return &cxsdk.ScheduleRecurring{
			Recurring: &cxsdk.Recurring{
				Condition: dynamic,
			},
		}, nil
	} else if a.Spec.Schedule.Recurring.Always != nil {
		return &cxsdk.ScheduleRecurring{
			Recurring: &cxsdk.Recurring{
				Condition: &cxsdk.RecurringAlways{},
			},
		}, nil
	}

	return nil, fmt.Errorf("exactly one of `dynamic` or `always` must be set")
}

func (a *AlertScheduler) extractDynamic() (*cxsdk.RecurringDynamic, error) {
	timeFrame, err := extractTimeFrame(a.Spec.Schedule.Recurring.Dynamic.TimeFrame)
	if err != nil {
		return nil, fmt.Errorf("error on extracting time frame: %w", err)
	}

	recurringDynamic := &cxsdk.RecurringDynamic{
		Dynamic: &cxsdk.RecurringDynamicInner{
			RepeatEvery:     a.Spec.Schedule.Recurring.Dynamic.RepeatEvery,
			Timeframe:       timeFrame,
			TerminationDate: a.Spec.Schedule.Recurring.Dynamic.TerminationDate,
		},
	}

	if a.Spec.Schedule.Recurring.Dynamic.Frequency.Daily != nil {
		recurringDynamic.Dynamic.Frequency = &cxsdk.RecurringDynamicDaily{}
	} else if weekly := a.Spec.Schedule.Recurring.Dynamic.Frequency.Weekly; weekly != nil {
		var daysOfWeek []int32
		for _, day := range weekly.Days {
			daysOfWeek = append(daysOfWeek, daysToProtoValue[day])
		}
		recurringDynamic.Dynamic.Frequency = &cxsdk.RecurringDynamicWeekly{
			Weekly: &cxsdk.Weekly{
				DaysOfWeek: daysOfWeek,
			},
		}
	} else if monthly := a.Spec.Schedule.Recurring.Dynamic.Frequency.Monthly; monthly != nil {
		recurringDynamic.Dynamic.Frequency = &cxsdk.RecurringDynamicMonthly{
			Monthly: &cxsdk.Monthly{
				DaysOfMonth: monthly.Days,
			},
		}
	} else {
		return nil, fmt.Errorf("exactly one of `daily`, `weekly` or `monthly` must be set")
	}

	return recurringDynamic, nil
}

func extractTimeFrame(timeFrame *TimeFrame) (*cxsdk.Timeframe, error) {
	result := &cxsdk.Timeframe{
		StartTime: timeFrame.StartTime,
		Timezone:  timeFrame.Timezone,
	}

	if timeFrame.EndTime != nil {
		result.Until = &cxsdk.TimeframeEndTime{
			EndTime: *timeFrame.EndTime,
		}
	} else if timeFrame.Duration != nil {
		result.Until = &cxsdk.TimeframeDuration{
			Duration: &cxsdk.AlertSchedulerDuration{
				ForOver:   timeFrame.Duration.ForOver,
				Frequency: schemaToProtoDurationFrequency[timeFrame.Duration.Frequency],
			},
		}
	} else {
		return nil, fmt.Errorf("exactly one of `endTime` or `duration` must be set")
	}

	return result, nil
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.printableStatus"
// AlertScheduler is the Schema for the AlertSchedulers API.
// It is used to suppress or activate alerts based on a schedule.
// See also https://coralogix.com/docs/user-guides/alerting/alert-suppression-rules/
//
// **Added in v0.4.0**
type AlertScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertSchedulerSpec   `json:"spec,omitempty"`
	Status AlertSchedulerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertSchedulerList contains a list of AlertScheduler.
type AlertSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertScheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertScheduler{}, &AlertSchedulerList{})
}
