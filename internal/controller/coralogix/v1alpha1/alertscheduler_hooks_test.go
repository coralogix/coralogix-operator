// Copyright 2026 Coralogix Ltd.
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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"

	alertschedulerruleservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_scheduler_rule_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var labelsFilter = coralogixv1alpha1.AlertSchedulerRuleProtobufV1Filter{
	MetaLabels: []coralogixv1alpha1.AlertSchedulerMetaLabelsProtobufV1MetaLabel{{Key: "app"}},
}

// The CRD cannot check the always arm of a recurring schedule, an empty object, so validate does.
// Each case runs twice: the hook is safe to repeat.
func TestAlertSchedulerValidateChecksTheRecurringArms(t *testing.T) {
	recurring := func(r *coralogixv1alpha1.AlertSchedulerRecurring) *coralogixv1alpha1.AlertScheduler {
		return &coralogixv1alpha1.AlertScheduler{Spec: coralogixv1alpha1.AlertSchedulerSpec{
			Filter:   labelsFilter,
			Schedule: coralogixv1alpha1.AlertSchedulerSchedule{Recurring: r},
		}}
	}
	dynamic := &coralogixv1alpha1.AlertSchedulerRecurringDynamic{RepeatEvery: 1}
	for _, tc := range []struct {
		name    string
		cr      *coralogixv1alpha1.AlertScheduler
		wantErr bool
	}{
		{"a one-time schedule", &coralogixv1alpha1.AlertScheduler{Spec: coralogixv1alpha1.AlertSchedulerSpec{
			Filter:   labelsFilter,
			Schedule: coralogixv1alpha1.AlertSchedulerSchedule{OneTime: &coralogixv1alpha1.AlertSchedulerTimeframe{}},
		}}, false},
		{"always", recurring(&coralogixv1alpha1.AlertSchedulerRecurring{Always: &coralogixv1alpha1.AlertSchedulerAlways{}}), false},
		{"dynamic", recurring(&coralogixv1alpha1.AlertSchedulerRecurring{Dynamic: dynamic}), false},
		{"neither", recurring(&coralogixv1alpha1.AlertSchedulerRecurring{}), true},
		{"both", recurring(&coralogixv1alpha1.AlertSchedulerRecurring{Always: &coralogixv1alpha1.AlertSchedulerAlways{}, Dynamic: dynamic}), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &AlertSchedulerReconciler{}
			for range 2 {
				err := r.validate(context.Background(), logr.Discard(), tc.cr)
				if tc.wantErr {
					require.ErrorContains(t, err, "exactly one of always or dynamic")
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

// The operator cannot keep an empty filter list through its own writes, so validate rejects it.
func TestAlertSchedulerValidateRejectsAnEmptyFilter(t *testing.T) {
	schedule := coralogixv1alpha1.AlertSchedulerSchedule{OneTime: &coralogixv1alpha1.AlertSchedulerTimeframe{}}
	for _, tc := range []struct {
		name    string
		filter  coralogixv1alpha1.AlertSchedulerRuleProtobufV1Filter
		wantErr bool
	}{
		{"meta labels", labelsFilter, false},
		{"alerts", coralogixv1alpha1.AlertSchedulerRuleProtobufV1Filter{
			Alerts: []coralogixv1alpha1.ResourceOnlyRef{{ResourceRef: &coralogixv1alpha1.ResourceRef{Name: "first"}}},
		}, false},
		{"empty meta labels", coralogixv1alpha1.AlertSchedulerRuleProtobufV1Filter{
			MetaLabels: []coralogixv1alpha1.AlertSchedulerMetaLabelsProtobufV1MetaLabel{},
		}, true},
		{"empty alerts", coralogixv1alpha1.AlertSchedulerRuleProtobufV1Filter{Alerts: []coralogixv1alpha1.ResourceOnlyRef{}}, true},
		{"no list", coralogixv1alpha1.AlertSchedulerRuleProtobufV1Filter{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &AlertSchedulerReconciler{}
			cr := &coralogixv1alpha1.AlertScheduler{Spec: coralogixv1alpha1.AlertSchedulerSpec{Filter: tc.filter, Schedule: schedule}}
			for range 2 {
				err := r.validate(context.Background(), logr.Discard(), cr)
				if tc.wantErr {
					require.ErrorContains(t, err, "metaLabels or alerts must have at least one item")
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}

// A stored CR can set two frequencies. Create and Update send the first of daily, weekly, and
// monthly, as the released operator did. Each hook runs twice: it is safe to repeat.
func TestAlertSchedulerSendsTheFirstFrequency(t *testing.T) {
	weekly := func() *alertschedulerruleservice.Weekly { return &alertschedulerruleservice.Weekly{} }
	monthly := func() *alertschedulerruleservice.Monthly { return &alertschedulerruleservice.Monthly{} }
	for _, tc := range []struct {
		name                             string
		in                               alertschedulerruleservice.RecurringDynamic
		wantDaily, wantWeekly, wantMonth bool
	}{
		{"daily and weekly", alertschedulerruleservice.RecurringDynamic{Daily: map[string]interface{}{}, Weekly: weekly()}, true, false, false},
		{"daily and monthly", alertschedulerruleservice.RecurringDynamic{Daily: map[string]interface{}{}, Monthly: monthly()}, true, false, false},
		{"weekly and monthly", alertschedulerruleservice.RecurringDynamic{Weekly: weekly(), Monthly: monthly()}, false, true, false},
		{"all three", alertschedulerruleservice.RecurringDynamic{Daily: map[string]interface{}{}, Weekly: weekly(), Monthly: monthly()}, true, false, false},
		{"monthly", alertschedulerruleservice.RecurringDynamic{Monthly: monthly()}, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := func(d *alertschedulerruleservice.RecurringDynamic) {
				require.Equal(t, tc.wantDaily, d.Daily != nil, "daily")
				require.Equal(t, tc.wantWeekly, d.Weekly != nil, "weekly")
				require.Equal(t, tc.wantMonth, d.Monthly != nil, "monthly")
			}
			r := &AlertSchedulerReconciler{}
			createDynamic := tc.in
			create := &alertschedulerruleservice.CreateAlertSchedulerRuleRequestDataStructure{}
			create.AlertSchedulerRule.Schedule.Recurring = &alertschedulerruleservice.Recurring{Schedule: &createDynamic}
			updateDynamic := tc.in
			update := &alertschedulerruleservice.UpdateAlertSchedulerRuleRequestDataStructure{}
			update.AlertSchedulerRule.Schedule.Recurring = &alertschedulerruleservice.Recurring{Schedule: &updateDynamic}
			for range 2 {
				require.NoError(t, r.beforeCreate(context.Background(), logr.Discard(), nil, create))
				check(create.AlertSchedulerRule.Schedule.Recurring.Schedule)
				require.NoError(t, r.beforeUpdate(context.Background(), logr.Discard(), nil, update))
				check(update.AlertSchedulerRule.Schedule.Recurring.Schedule)
			}
		})
	}
	// A one-time schedule has no frequency.
	body := &alertschedulerruleservice.CreateAlertSchedulerRuleRequestDataStructure{}
	require.NoError(t, (&AlertSchedulerReconciler{}).beforeCreate(context.Background(), logr.Discard(), nil, body))
}
