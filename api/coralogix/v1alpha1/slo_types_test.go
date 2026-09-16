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
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

// The "exactly one" checks on ApmSli and ApmLatencySli are enforced in Go rather than by
// a CEL rule, because errorConfig and average are empty objects and Kubernetes 1.29 and
// earlier cannot evaluate a CEL rule against a field whose schema has no properties.
func TestExpandApmSliRequiresExactlyOneBranch(t *testing.T) {
	latency := &ApmLatencySli{
		TimeWindow: "5m",
		Quantile:   &ApmLatencyQuantile{Percentile: ptr.To(resource.MustParse("0.95"))},
	}

	for _, tc := range []struct {
		name    string
		apmSli  ApmSli
		wantErr string
	}{
		{
			name:    "neither branch",
			apmSli:  ApmSli{Services: []string{"svc"}},
			wantErr: "exactly one of errorConfig or latencyConfig must be set",
		},
		{
			name: "both branches",
			apmSli: ApmSli{
				Services:      []string{"svc"},
				ErrorConfig:   &ApmErrorSli{},
				LatencyConfig: latency,
			},
			wantErr: "exactly one of errorConfig or latencyConfig must be set",
		},
		{
			name:   "errorConfig only",
			apmSli: ApmSli{Services: []string{"svc"}, ErrorConfig: &ApmErrorSli{}},
		},
		{
			name:   "latencyConfig only",
			apmSli: ApmSli{Services: []string{"svc"}, LatencyConfig: latency},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expanded, err := tc.apmSli.ExpandApmSli()
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, []string{"svc"}, expanded.Services)
		})
	}
}

func TestExpandApmSliSerializesErrorConfigAsEmptyObject(t *testing.T) {
	apmSli := ApmSli{Services: []string{"svc"}, ErrorConfig: &ApmErrorSli{}}

	expanded, err := apmSli.ExpandApmSli()
	require.NoError(t, err)

	// The SDK drops a nil map and serializes a non-nil empty map as {}. The API rejects
	// an apmSli with no branch, so the map has to be allocated.
	require.NotNil(t, expanded.ErrorConfig)
	require.Empty(t, expanded.ErrorConfig)
}

func TestExpandApmLatencySliRequiresExactlyOneQueryType(t *testing.T) {
	for _, tc := range []struct {
		name    string
		latency ApmLatencySli
		wantErr string
	}{
		{
			name:    "neither query type",
			latency: ApmLatencySli{TimeWindow: "5m"},
			wantErr: "exactly one of quantile or average must be set",
		},
		{
			name: "both query types",
			latency: ApmLatencySli{
				TimeWindow: "5m",
				Quantile:   &ApmLatencyQuantile{},
				Average:    &ApmLatencyAverage{},
			},
			wantErr: "exactly one of quantile or average must be set",
		},
		{
			name:    "quantile only",
			latency: ApmLatencySli{TimeWindow: "5m", Quantile: &ApmLatencyQuantile{}},
		},
		{
			name:    "average only",
			latency: ApmLatencySli{TimeWindow: "5m", Average: &ApmLatencyAverage{}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expanded, err := tc.latency.ExpandApmLatencySli()
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "WINDOW_SLO_WINDOW_5_MINUTES", string(*expanded.TimeWindow))
		})
	}
}

func TestExpandApmLatencySliRejectsAnUnusableTimeWindow(t *testing.T) {
	latency := ApmLatencySli{TimeWindow: "unspecified", Average: &ApmLatencyAverage{}}

	_, err := latency.ExpandApmLatencySli()
	require.EqualError(t, err, "invalid APM latency time window: unspecified")
}

func TestExpandApmFiltersSendsAnEmptyListRatherThanNil(t *testing.T) {
	apmSli := ApmSli{
		Services:    []string{"svc"},
		ErrorConfig: &ApmErrorSli{},
		Filters:     []ApmFilter{{Key: "http.method"}},
	}

	expanded, err := apmSli.ExpandApmSli()
	require.NoError(t, err)
	require.Len(t, expanded.Filters, 1)
	require.Equal(t, "http.method", *expanded.Filters[0].Key)
	// The API accepts an empty values list. A nil slice would be dropped by the SDK,
	// which is a shape no probe covered.
	require.NotNil(t, expanded.Filters[0].Values)
	require.Empty(t, expanded.Filters[0].Values)
}
