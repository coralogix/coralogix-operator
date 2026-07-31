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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	events2metrics "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/events2metrics_service"
)

func TestExtractCreateE2MRequestLogsWireShape(t *testing.T) {
	spec := Events2MetricSpec{
		Name:              "logs-e2m",
		Description:       ptr.To("desc"),
		DataSource:        ptr.To("default/logs"),
		PermutationsLimit: ptr.To(int32(100)),
		MetricLabels: []MetricLabel{
			{TargetLabel: "status", SourceField: "status"},
		},
		MetricFields: []MetricField{
			{
				TargetBaseMetricName: "request_count",
				SourceField:          "request_count",
				Aggregations: []MetricFieldAggregation{
					{
						Enabled:          true,
						AggType:          AggregationTypeMin,
						TargetMetricName: "min_request_count",
					},
					{
						Enabled:          false,
						AggType:          AggregationTypeMax,
						TargetMetricName: "max_request_count",
					},
					{
						Enabled:          true,
						AggType:          AggregationTypeCount,
						TargetMetricName: "count_request_count",
					},
					{
						Enabled:          true,
						AggType:          AggregationTypeAvg,
						TargetMetricName: "avg_request_count",
					},
					{
						Enabled:          true,
						AggType:          AggregationTypeSum,
						TargetMetricName: "sum_request_count",
					},
				},
			},
		},
		Query: E2MQuery{
			Logs: &E2MQueryLogs{
				Lucene:                 ptr.To("status:200"),
				Alias:                  ptr.To("e2m-logs"),
				ApplicationNameFilters: []string{"app"},
				SubsystemNameFilters:   []string{"sub"},
				SeverityFilters:        []L2MSeverity{L2MSeverityError, L2MSeverityCritical},
			},
		},
	}

	got, err := spec.ExtractCreateE2MRequest()
	require.NoError(t, err)

	require.Equal(t, "logs-e2m", got.Name)
	require.Equal(t, ptr.To("desc"), got.Description)
	require.Equal(t, ptr.To("default/logs"), got.DataSource)
	require.Equal(t, ptr.To(int32(100)), got.PermutationsLimit)
	require.Equal(t, events2metrics.E2MTYPE_E2_M_TYPE_LOGS2_METRICS.Ptr(), got.Type)
	require.Nil(t, got.SpansQuery)
	require.NotNil(t, got.LogsQuery)
	require.Equal(t, ptr.To("status:200"), got.LogsQuery.Lucene)
	require.Equal(t, ptr.To("e2m-logs"), got.LogsQuery.Alias)
	require.Equal(t, []string{"app"}, got.LogsQuery.ApplicationnameFilters)
	require.Equal(t, []string{"sub"}, got.LogsQuery.SubsystemnameFilters)
	require.Equal(t, []events2metrics.Logs2metricsV2Severity{
		events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_ERROR,
		events2metrics.LOGS2METRICSV2SEVERITY_SEVERITY_CRITICAL,
	}, got.LogsQuery.SeverityFilters)

	require.Len(t, got.MetricLabels, 1)
	require.Equal(t, "status", got.MetricLabels[0].TargetLabel)
	require.Equal(t, "status", got.MetricLabels[0].SourceField)

	require.Len(t, got.MetricFields, 1)
	require.Equal(t, "request_count", got.MetricFields[0].TargetBaseMetricName)
	require.Len(t, got.MetricFields[0].Aggregations, 5)

	for _, agg := range got.MetricFields[0].Aggregations {
		require.NotNil(t, agg.Enabled)
		require.NotNil(t, agg.AggType)
		require.NotNil(t, agg.TargetMetricName)
		require.Nil(t, agg.Samples)
		require.Nil(t, agg.Histogram)
		require.Nil(t, agg.None)
	}
	require.True(t, *got.MetricFields[0].Aggregations[0].Enabled)
	require.False(t, *got.MetricFields[0].Aggregations[1].Enabled)
	require.Equal(t, events2metrics.AGGTYPE_AGG_TYPE_MIN, *got.MetricFields[0].Aggregations[0].AggType)
	require.Equal(t, events2metrics.AGGTYPE_AGG_TYPE_MAX, *got.MetricFields[0].Aggregations[1].AggType)
}

func TestExtractCreateE2MRequestSpansWireShape(t *testing.T) {
	spec := Events2MetricSpec{
		Name: "spans-e2m",
		MetricFields: []MetricField{
			{
				TargetBaseMetricName: "duration",
				SourceField:          "duration",
				Aggregations: []MetricFieldAggregation{
					{
						Enabled:          true,
						AggType:          AggregationTypeMin,
						TargetMetricName: "min_duration",
					},
				},
			},
		},
		Query: E2MQuery{
			Spans: &E2MQuerySpans{
				Lucene:                 ptr.To("service:nginx"),
				ApplicationNameFilters: []string{"app"},
				SubsystemNameFilters:   []string{"sub"},
				ActionFilters:          []string{"GET"},
				ServiceFilters:         []string{"nginx"},
			},
		},
	}

	got, err := spec.ExtractCreateE2MRequest()
	require.NoError(t, err)
	require.Equal(t, events2metrics.E2MTYPE_E2_M_TYPE_SPANS2_METRICS.Ptr(), got.Type)
	require.Nil(t, got.LogsQuery)
	require.NotNil(t, got.SpansQuery)
	require.Equal(t, ptr.To("service:nginx"), got.SpansQuery.Lucene)
	require.Equal(t, []string{"app"}, got.SpansQuery.ApplicationnameFilters)
	require.Equal(t, []string{"sub"}, got.SpansQuery.SubsystemnameFilters)
	require.Equal(t, []string{"GET"}, got.SpansQuery.ActionFilters)
	require.Equal(t, []string{"nginx"}, got.SpansQuery.ServiceFilters)
}

func TestExtractCreateE2MRequestSamplesAndHistogram(t *testing.T) {
	spec := Events2MetricSpec{
		Name: "agg-meta-e2m",
		MetricFields: []MetricField{
			{
				TargetBaseMetricName: "latency",
				SourceField:          "latency",
				Aggregations: []MetricFieldAggregation{
					{
						Enabled:          true,
						AggType:          AggregationTypeSamples,
						TargetMetricName: "samples_latency",
						AggMetadata: AggregationMetadata{
							Samples: &SamplesMetadata{SampleType: E2MAggSamplesSampleTypeMax},
						},
					},
					{
						Enabled:          true,
						AggType:          AggregationTypeHistogram,
						TargetMetricName: "histogram_latency",
						AggMetadata: AggregationMetadata{
							Histogram: &HistogramMetadata{
								Buckets: []resource.Quantity{
									resource.MustParse("0.1"),
									resource.MustParse("5.5"),
									resource.MustParse("100"),
								},
							},
						},
					},
				},
			},
		},
		Query: E2MQuery{
			Logs: &E2MQueryLogs{},
		},
	}

	got, err := spec.ExtractCreateE2MRequest()
	require.NoError(t, err)
	require.Len(t, got.MetricFields[0].Aggregations, 2)

	samples := got.MetricFields[0].Aggregations[0]
	require.Equal(t, events2metrics.AGGTYPE_AGG_TYPE_SAMPLES, *samples.AggType)
	require.NotNil(t, samples.Samples)
	require.Equal(t, events2metrics.SAMPLETYPE_SAMPLE_TYPE_MAX, *samples.Samples.SampleType)
	require.Nil(t, samples.Histogram)
	require.Nil(t, samples.None)

	histogram := got.MetricFields[0].Aggregations[1]
	require.Equal(t, events2metrics.AGGTYPE_AGG_TYPE_HISTOGRAM, *histogram.AggType)
	require.NotNil(t, histogram.Histogram)
	require.Equal(t, []float32{0.1, 5.5, 100}, histogram.Histogram.Buckets)
	require.Nil(t, histogram.Samples)
	require.Nil(t, histogram.None)
}

func TestExtractCreateE2MRequestEmptyAggregationsIsEmptySlice(t *testing.T) {
	spec := Events2MetricSpec{
		Name: "empty-aggs",
		MetricFields: []MetricField{
			{
				TargetBaseMetricName: "field",
				SourceField:          "field",
			},
		},
		Query: E2MQuery{Logs: &E2MQueryLogs{}},
	}

	got, err := spec.ExtractCreateE2MRequest()
	require.NoError(t, err)
	require.NotNil(t, got.MetricFields[0].Aggregations)
	require.Empty(t, got.MetricFields[0].Aggregations)

	payload, err := json.Marshal(got.MetricFields[0])
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(payload, &raw))
	aggs, ok := raw["aggregations"].([]any)
	require.True(t, ok, "aggregations must be present as a JSON array, got %#v", raw["aggregations"])
	require.Empty(t, aggs)
}

func TestExtractCreateAndReplaceOmitOptionalFields(t *testing.T) {
	spec := Events2MetricSpec{
		Name: "omit-optionals",
		Query: E2MQuery{
			Logs: &E2MQueryLogs{},
		},
	}

	create, err := spec.ExtractCreateE2MRequest()
	require.NoError(t, err)
	require.Nil(t, create.Description)
	require.Nil(t, create.DataSource)
	require.Nil(t, create.PermutationsLimit)
	require.Nil(t, create.LogsQuery.Lucene)
	require.Nil(t, create.LogsQuery.Alias)

	createPayload, err := json.Marshal(create)
	require.NoError(t, err)
	var createRaw map[string]any
	require.NoError(t, json.Unmarshal(createPayload, &createRaw))
	_, hasPermutationsLimit := createRaw["permutationsLimit"]
	require.False(t, hasPermutationsLimit)
	_, hasDescription := createRaw["description"]
	require.False(t, hasDescription)
	_, hasDataSource := createRaw["dataSource"]
	require.False(t, hasDataSource)

	replace, err := spec.ExtractReplaceE2MRequest("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	require.Equal(t, ptr.To("11111111-1111-1111-1111-111111111111"), replace.Id)
	require.Equal(t, ptr.To(""), replace.Description)
	require.Nil(t, replace.DataSource)
	require.Nil(t, replace.LogsQuery.Lucene)
	require.Nil(t, replace.LogsQuery.Alias)
	require.Nil(t, replace.Permutations)

	replacePayload, err := json.Marshal(replace)
	require.NoError(t, err)
	var replaceRaw map[string]any
	require.NoError(t, json.Unmarshal(replacePayload, &replaceRaw))
	_, hasPermutations := replaceRaw["permutations"]
	require.False(t, hasPermutations)
	require.Equal(t, "", replaceRaw["description"])
	_, hasDataSource = replaceRaw["dataSource"]
	require.False(t, hasDataSource)
	logsQuery, ok := replaceRaw["logsQuery"].(map[string]any)
	require.True(t, ok)
	_, hasLucene := logsQuery["lucene"]
	require.False(t, hasLucene)
	_, hasAlias := logsQuery["alias"]
	require.False(t, hasAlias)

	spansSpec := Events2MetricSpec{
		Name: "omit-optionals-spans",
		Query: E2MQuery{
			Spans: &E2MQuerySpans{},
		},
	}
	spansReplace, err := spansSpec.ExtractReplaceE2MRequest("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	require.Nil(t, spansReplace.SpansQuery.Lucene)
}

func TestExtractReplaceE2MRequestPresenceRules(t *testing.T) {
	spec := Events2MetricSpec{
		Name:              "replace-e2m",
		Description:       ptr.To("updated"),
		DataSource:        ptr.To("default/logs"),
		PermutationsLimit: ptr.To(int32(42)),
		MetricFields: []MetricField{
			{
				TargetBaseMetricName: "field",
				SourceField:          "field",
				Aggregations: []MetricFieldAggregation{
					{
						Enabled:          false,
						AggType:          AggregationTypeSum,
						TargetMetricName: "sum_field",
					},
				},
			},
		},
		Query: E2MQuery{
			Logs: &E2MQueryLogs{
				Lucene: ptr.To("status:500"),
				Alias:  ptr.To("alias"),
			},
		},
	}

	got, err := spec.ExtractReplaceE2MRequest("22222222-2222-2222-2222-222222222222")
	require.NoError(t, err)
	require.Equal(t, "replace-e2m", got.Name)
	require.Equal(t, events2metrics.E2MTYPE_E2_M_TYPE_LOGS2_METRICS, got.Type)
	require.Equal(t, ptr.To("updated"), got.Description)
	require.Equal(t, ptr.To("default/logs"), got.DataSource)
	require.NotNil(t, got.Permutations)
	require.Equal(t, int32(42), got.Permutations.Limit)
	require.False(t, got.Permutations.HasExceededLimit)
	require.False(t, *got.MetricFields[0].Aggregations[0].Enabled)
	require.Equal(t, ptr.To("status:500"), got.LogsQuery.Lucene)
	require.Equal(t, ptr.To("alias"), got.LogsQuery.Alias)
}
