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

package v1beta1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// ReasonEvaluationPeriodExceedsTimeframe is the reason of the Warning event that the
// operator records when an analytics alert is evaluated less often than its time frame.
const ReasonEvaluationPeriodExceedsTimeframe = "EvaluationPeriodExceedsTimeframe"

// evaluationPeriodWarning returns a warning message when an analytics alert sets an
// evaluation period longer than its time frame. The API accepts such an alert, but data
// between evaluations is not covered by any evaluation.
func evaluationPeriodWarning(spec *coralogixv1beta1.AlertSpec) (string, bool) {
	var timeframe int32
	var period *int32
	switch {
	case spec.TypeDefinition.AnalyticsResultCount != nil:
		timeframe = spec.TypeDefinition.AnalyticsResultCount.TimeframeMinutes
		period = spec.TypeDefinition.AnalyticsResultCount.EvaluationPeriodMinutes
	case spec.TypeDefinition.AnalyticsThreshold != nil:
		timeframe = spec.TypeDefinition.AnalyticsThreshold.TimeframeMinutes
		period = spec.TypeDefinition.AnalyticsThreshold.EvaluationPeriodMinutes
	default:
		return "", false
	}
	if period == nil || *period <= timeframe {
		return "", false
	}
	return fmt.Sprintf(
		"evaluationPeriodMinutes (%d) is longer than timeframeMinutes (%d): "+
			"data between evaluations is not covered by any evaluation, so events can be missed",
		*period, timeframe,
	), true
}

// warnEvaluationPeriod records a Warning event when the alert's evaluation period exceeds
// its time frame. It never fails the reconcile. It records the event once per generation:
// a generation that is already synced does not record it again on every reconcile interval.
func (r *AlertReconciler) warnEvaluationPeriod(alert *coralogixv1beta1.Alert) {
	if r.Recorder == nil {
		return
	}
	message, ok := evaluationPeriodWarning(&alert.Spec)
	if !ok || generationSynced(alert.Status.Conditions, alert.Generation) {
		return
	}
	r.Recorder.Event(alert, corev1.EventTypeWarning, ReasonEvaluationPeriodExceedsTimeframe, message)
}

func generationSynced(conditions []metav1.Condition, generation int64) bool {
	cond := meta.FindStatusCondition(conditions, utils.ConditionTypeRemoteSynced)
	return cond != nil && cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == generation
}
