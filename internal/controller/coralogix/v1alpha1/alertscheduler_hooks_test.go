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

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The CRD cannot check the always arm of a recurring schedule, an empty object, so validate does.
// Each case runs twice: the hook is safe to repeat.
func TestAlertSchedulerValidateChecksTheRecurringArms(t *testing.T) {
	recurring := func(r *coralogixv1alpha1.AlertSchedulerRecurring) *coralogixv1alpha1.AlertScheduler {
		return &coralogixv1alpha1.AlertScheduler{Spec: coralogixv1alpha1.AlertSchedulerSpec{
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
