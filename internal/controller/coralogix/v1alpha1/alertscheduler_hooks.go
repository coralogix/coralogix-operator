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

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// validate checks that a recurring schedule sets exactly one of always and dynamic. always is an
// empty object, and Kubernetes 1.28 and earlier cannot evaluate a CEL rule that reads such a field,
// so the CRD has no rule for it.
func (r *AlertSchedulerReconciler) validate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.AlertScheduler) error {
	if recurring := cr.Spec.Schedule.Recurring; recurring != nil && (recurring.Always == nil) == (recurring.Dynamic == nil) {
		return errors.New("schedule.recurring: exactly one of always or dynamic must be set")
	}
	return nil
}
