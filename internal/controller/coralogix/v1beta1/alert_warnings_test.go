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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

func analyticsTypeDefinitions(timeframe int32, period *int32) map[string]coralogixv1beta1.AlertTypeDefinition {
	return map[string]coralogixv1beta1.AlertTypeDefinition{
		"immediate": {AnalyticsImmediate: &coralogixv1beta1.AnalyticsImmediate{
			DataprimeQuery:          coralogixv1beta1.DataprimeQuery{Query: "source logs | count"},
			TimeframeMinutes:        timeframe,
			EvaluationPeriodMinutes: period,
		}},
		"threshold": {AnalyticsThreshold: &coralogixv1beta1.AnalyticsThreshold{
			DataprimeQuery: coralogixv1beta1.DataprimeQuery{Query: "source logs | count as c"},
			Rules: []coralogixv1beta1.AnalyticsThresholdRule{{
				Condition: coralogixv1beta1.AnalyticsThresholdRuleCondition{Threshold: resource.MustParse("1")},
				Override:  &coralogixv1beta1.AlertOverride{Priority: coralogixv1beta1.AlertPriorityP5},
			}},
			TimeframeMinutes:        timeframe,
			EvaluationPeriodMinutes: period,
		}},
	}
}

func TestEvaluationPeriodWarning(t *testing.T) {
	for _, tc := range []struct {
		name      string
		timeframe int32
		period    *int32
		want      string
	}{
		{
			name:      "greater than the time frame warns",
			timeframe: 5,
			period:    ptr.To[int32](60),
			want: "evaluationPeriodMinutes (60) is longer than timeframeMinutes (5): " +
				"data between evaluations is not covered by any evaluation, so events can be missed",
		},
		{name: "equal to the time frame does not warn", timeframe: 15, period: ptr.To[int32](15)},
		{name: "less than the time frame does not warn", timeframe: 15, period: ptr.To[int32](5)},
		{name: "omitted period does not warn", timeframe: 5, period: nil},
	} {
		for typeName, def := range analyticsTypeDefinitions(tc.timeframe, tc.period) {
			t.Run(tc.name+"/"+typeName, func(t *testing.T) {
				msg, ok := evaluationPeriodWarning(&coralogixv1beta1.AlertSpec{TypeDefinition: def})
				require.Equal(t, tc.want != "", ok)
				require.Equal(t, tc.want, msg)
			})
		}
	}

	t.Run("non-analytics type does not warn", func(t *testing.T) {
		_, ok := evaluationPeriodWarning(&testAlert(nil).Spec)
		require.False(t, ok)
	})
}

func drainEvents(recorder *record.FakeRecorder) []string {
	var events []string
	for {
		select {
		case e := <-recorder.Events:
			events = append(events, e)
		default:
			return events
		}
	}
}

func analyticsAlertServer(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		mu.Lock()
		calls++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"alertDef":{"id":"alert-id"}}`))
	}))
	t.Cleanup(server.Close)
	return server, func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}
}

func TestHandlersRecordEvaluationPeriodWarningAndStillSync(t *testing.T) {
	const want = "Warning EvaluationPeriodExceedsTimeframe evaluationPeriodMinutes (60) is longer than timeframeMinutes (5): " +
		"data between evaluations is not covered by any evaluation, so events can be missed"

	for _, handler := range []string{"create", "update"} {
		for _, tc := range []struct {
			name   string
			period *int32
			want   []string
		}{
			{name: "greater", period: ptr.To[int32](60), want: []string{want}},
			{name: "equal", period: ptr.To[int32](5)},
			{name: "omitted", period: nil},
		} {
			t.Run(handler+"/"+tc.name, func(t *testing.T) {
				server, calls := analyticsAlertServer(t)
				alert := testAlert(nil)
				alert.Spec.TypeDefinition = analyticsTypeDefinitions(5, tc.period)["immediate"]
				initAlertClient(t, alert)

				recorder := record.NewFakeRecorder(10)
				reconciler := testAlertReconciler(t, server)
				reconciler.Recorder = recorder

				var err error
				if handler == "create" {
					alert.Status = coralogixv1beta1.AlertStatus{}
					err = reconciler.HandleCreation(context.Background(), logr.Discard(), alert)
				} else {
					err = reconciler.HandleUpdate(context.Background(), logr.Discard(), alert)
				}
				require.NoError(t, err)
				require.Equal(t, 1, calls(), "the alert must still be sent to the API")
				require.Equal(t, tc.want, drainEvents(recorder))
			})
		}
	}
}

func TestHandleUpdateSkipsEvaluationPeriodWarningForSyncedGeneration(t *testing.T) {
	server, calls := analyticsAlertServer(t)
	alert := testAlert(nil)
	alert.Generation = 3
	alert.Spec.TypeDefinition = analyticsTypeDefinitions(5, ptr.To[int32](60))["threshold"]
	utils.SetSyncedConditionTrue(&alert.Status.Conditions, 3, utils.ReasonRemoteSyncedSuccessfully)
	initAlertClient(t, alert)

	recorder := record.NewFakeRecorder(10)
	reconciler := testAlertReconciler(t, server)
	reconciler.Recorder = recorder

	require.NoError(t, reconciler.HandleUpdate(context.Background(), logr.Discard(), alert))
	require.Equal(t, 1, calls())
	require.Empty(t, drainEvents(recorder), "a synced generation must not record the warning again")

	// A new generation that has not been synced yet records it again.
	alert.Generation = 4
	require.NoError(t, reconciler.HandleUpdate(context.Background(), logr.Discard(), alert))
	require.Len(t, drainEvents(recorder), 1)
}

func TestWarnEvaluationPeriodWithoutRecorder(t *testing.T) {
	alert := testAlert(nil)
	alert.ObjectMeta = metav1.ObjectMeta{Name: "a", Namespace: "default"}
	alert.Spec.TypeDefinition = analyticsTypeDefinitions(5, ptr.To[int32](60))["immediate"]
	require.NotPanics(t, func() { (&AlertReconciler{}).warnEvaluationPeriod(alert) })
}
