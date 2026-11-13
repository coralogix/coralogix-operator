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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	slosservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

func sloWithAPM(apm *coralogixv1alpha1.SLOApmSli) *coralogixv1alpha1.SLO {
	return &coralogixv1alpha1.SLO{Spec: coralogixv1alpha1.SLOSpec{SliType: coralogixv1alpha1.SLOSpecSliType{ApmSli: apm}}}
}

// The CRD cannot check the empty-object arms of apmSli, so validate does. Each case runs twice:
// the hook is safe to repeat.
func TestSLOValidateChecksTheAPMArms(t *testing.T) {
	latency := func(quantile bool, average bool) *coralogixv1alpha1.SLOApmLatencySli {
		l := &coralogixv1alpha1.SLOApmLatencySli{TimeWindow: "5m"}
		if quantile {
			l.Quantile = &coralogixv1alpha1.SLOApmLatencyQuantile{Percentile: ptr.To(resource.MustParse("0.95"))}
		}
		if average {
			l.Average = &coralogixv1alpha1.SLOApmLatencyAverage{}
		}
		return l
	}
	for _, tc := range []struct {
		name    string
		slo     *coralogixv1alpha1.SLO
		wantErr string
	}{
		{"a metric SLI", &coralogixv1alpha1.SLO{}, ""},
		{"errorConfig only", sloWithAPM(&coralogixv1alpha1.SLOApmSli{ErrorConfig: &coralogixv1alpha1.SLOApmErrorSli{}}), ""},
		{"latencyConfig with quantile", sloWithAPM(&coralogixv1alpha1.SLOApmSli{LatencyConfig: latency(true, false)}), ""},
		{"latencyConfig with average", sloWithAPM(&coralogixv1alpha1.SLOApmSli{LatencyConfig: latency(false, true)}), ""},
		{"neither arm", sloWithAPM(&coralogixv1alpha1.SLOApmSli{}), "exactly one of errorConfig or latencyConfig"},
		{"both arms", sloWithAPM(&coralogixv1alpha1.SLOApmSli{ErrorConfig: &coralogixv1alpha1.SLOApmErrorSli{}, LatencyConfig: latency(true, false)}), "exactly one of errorConfig or latencyConfig"},
		{"neither query type", sloWithAPM(&coralogixv1alpha1.SLOApmSli{LatencyConfig: latency(false, false)}), "exactly one of quantile or average"},
		{"both query types", sloWithAPM(&coralogixv1alpha1.SLOApmSli{LatencyConfig: latency(true, true)}), "exactly one of quantile or average"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &SLOReconciler{}
			for range 2 {
				err := r.validate(context.Background(), logr.Discard(), tc.slo)
				if tc.wantErr == "" {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, tc.wantErr)
				}
			}
		})
	}
}

func sloResponseRevision(revision int32) slosservice.Slo {
	return slosservice.Slo{Revision: &slosservice.V1Revision{Revision: ptr.To(revision)}}
}

// afterCreate keeps the revision of the created SLO. A second run leaves the same status.
func TestSLOAfterCreateIsSafeToRepeat(t *testing.T) {
	r := &SLOReconciler{}
	cr := &coralogixv1alpha1.SLO{}
	resp := &slosservice.CreateSloResponse{Slo: sloResponseRevision(3)}
	for range 2 {
		require.NoError(t, r.afterCreate(context.Background(), logr.Discard(), cr, resp))
		require.Equal(t, ptr.To(int32(3)), cr.Status.Revision)
	}
}

// afterUpdate sets a missing revision, after a failed afterCreate, and leaves a set one as it is,
// as the released kind did. A second run leaves the same status.
func TestSLOAfterUpdateSetsOnlyAMissingRevision(t *testing.T) {
	r := &SLOReconciler{}
	resp := &slosservice.ReplaceSloResponse{Slo: sloResponseRevision(5)}

	missing := &coralogixv1alpha1.SLO{}
	for range 2 {
		require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), missing, resp))
		require.Equal(t, ptr.To(int32(5)), missing.Status.Revision)
	}

	set := &coralogixv1alpha1.SLO{Status: coralogixv1alpha1.SLOStatus{Revision: ptr.To(int32(1))}}
	for range 2 {
		require.NoError(t, r.afterUpdate(context.Background(), logr.Discard(), set, resp))
		require.Equal(t, ptr.To(int32(1)), set.Status.Revision)
	}
}
