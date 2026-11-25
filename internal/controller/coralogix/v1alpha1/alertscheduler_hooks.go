// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/go-logr/logr"

	alertschedulerruleservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_scheduler_rule_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// validate checks what the CRD cannot.
//
// A recurring schedule sets exactly one of always and dynamic. always is an empty object, and
// Kubernetes 1.28 and earlier cannot evaluate a CEL rule that reads such a field, so the CRD has no
// rule for it.
//
// The filter list that the CR sets has at least one item. The CRD admits an empty list, but the
// operator cannot keep it: the list is omitempty, so the write that adds the finalizer drops it, and
// the CRD rule that one of metaLabels and alerts is set then rejects that write. A rule created for
// such a CR would stay behind when the CR is deleted.
func (r *AlertSchedulerReconciler) validate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.AlertScheduler) error {
	if recurring := cr.Spec.Schedule.Recurring; recurring != nil && (recurring.Always == nil) == (recurring.Dynamic == nil) {
		return errors.New("schedule.recurring: exactly one of always or dynamic must be set")
	}
	if filter := cr.Spec.Filter; len(filter.MetaLabels) == 0 && len(filter.Alerts) == 0 {
		return errors.New("filter: metaLabels or alerts must have at least one item")
	}
	return nil
}

// beforeCreate sends one frequency, as the released operator did.
func (r *AlertSchedulerReconciler) beforeCreate(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.AlertScheduler, body *alertschedulerruleservice.CreateAlertSchedulerRuleRequestDataStructure) error { //nolint:unparam // the generated lifecycle fixes the signature
	keepFirstFrequency(&body.AlertSchedulerRule.Schedule)
	return nil
}

// beforeUpdate sends one frequency, as the released operator did.
func (r *AlertSchedulerReconciler) beforeUpdate(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.AlertScheduler, body *alertschedulerruleservice.UpdateAlertSchedulerRuleRequestDataStructure) error { //nolint:unparam // the generated lifecycle fixes the signature
	keepFirstFrequency(&body.AlertSchedulerRule.Schedule)
	return nil
}

// keepFirstFrequency keeps the first frequency of daily, weekly, and monthly. The released CRD has
// no rule on frequency, so a stored CR can set two; the released operator sent the first, and the
// API rejects a schedule with two.
func keepFirstFrequency(schedule *alertschedulerruleservice.Schedule) {
	if schedule.Recurring == nil || schedule.Recurring.Schedule == nil {
		return
	}
	dynamic := schedule.Recurring.Schedule
	switch {
	case dynamic.Daily != nil:
		dynamic.Weekly, dynamic.Monthly = nil, nil
	case dynamic.Weekly != nil:
		dynamic.Monthly = nil
	}
}
