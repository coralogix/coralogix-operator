// Copyright 2024 Coralogix Ltd.
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

package cxsdk

import (
	"context"

	e2m "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/events2metrics/v2"
	l2m "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/logs2metrics/v2"
	s2m "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/spans2metrics/v2"
)

// E2M is a type for configuring the Events2Metrics service.
type E2M = e2m.E2M

// GetE2MRequest is a type for configuring the Events2Metrics service.
type GetE2MRequest = e2m.GetE2MRequest

// CreateE2MRequest is a type for configuring the Events2Metrics service.
type CreateE2MRequest = e2m.CreateE2MRequest

// ReplaceE2MRequest is a type for configuring the Events2Metrics service.
type ReplaceE2MRequest = e2m.ReplaceE2MRequest

// DeleteE2MRequest is a type for configuring the Events2Metrics service.
type DeleteE2MRequest = e2m.DeleteE2MRequest

// ListLabelsCardinalityRequest is a type for configuring the Events2Metrics service.
type ListLabelsCardinalityRequest = e2m.ListLabelsCardinalityRequest

// AtomicBatchExecuteE2MRequest is a type for configuring the Events2Metrics service.
type AtomicBatchExecuteE2MRequest = e2m.AtomicBatchExecuteE2MRequest

// E2MAggSampleType is a type for configuring the Events2Metrics service.
type E2MAggSampleType = e2m.E2MAggSamples_SampleType

// E2MAggSamples is a type for configuring the Events2Metrics service.
type E2MAggSamples = e2m.E2MAggSamples

// E2MAggHistogram is a type for configuring the Events2Metrics service.
type E2MAggHistogram = e2m.E2MAggHistogram

// E2MCreateParams is a type for configuring the Events2Metrics service.
type E2MCreateParams = e2m.E2MCreateParams

// E2MPermutations is a type for configuring the Events2Metrics service.
type E2MPermutations = e2m.E2MPermutations

// MetricLabel is a type for configuring the Events2Metrics service.
type MetricLabel = e2m.MetricLabel

// MetricField is a type for configuring the Events2Metrics service.
type MetricField = e2m.MetricField

// E2MCreateParamsSpansQuery is a type for configuring the Events2Metrics service.
type E2MCreateParamsSpansQuery = e2m.E2MCreateParams_SpansQuery

// E2MCreateParamsLogsQuery is a type for configuring the Events2Metrics service.
type E2MCreateParamsLogsQuery = e2m.E2MCreateParams_LogsQuery

// E2MSpansQuery is a type for configuring the Events2Metrics service.
type E2MSpansQuery = e2m.E2M_SpansQuery

// S2MSpansQuery is a type for configuring the Events2Metrics service.
type S2MSpansQuery = s2m.SpansQuery

// E2MLogsQuery is a type for configuring the Events2Metrics service.
type E2MLogsQuery = e2m.E2M_LogsQuery

// L2MLogsQuery is a type for configuring the Events2Metrics service.
type L2MLogsQuery = l2m.LogsQuery

// L2MSeverity is a type for configuring the Events2Metrics service.
type L2MSeverity = l2m.Severity

// L2MSeverity values.
const (
	L2MSeverityUnspecified = l2m.Severity_SEVERITY_UNSPECIFIED
	L2MSeverityDebug       = l2m.Severity_SEVERITY_DEBUG
	L2MSeverityVerbose     = l2m.Severity_SEVERITY_VERBOSE
	L2MSeverityInfo        = l2m.Severity_SEVERITY_INFO
	L2MSeverityWarning     = l2m.Severity_SEVERITY_WARNING
	L2MSeverityError       = l2m.Severity_SEVERITY_ERROR
	L2MSeverityCritical    = l2m.Severity_SEVERITY_CRITICAL
)

// E2MAggSampleType values.
const (
	E2MAggSampleTypeMin = e2m.E2MAggSamples_SAMPLE_TYPE_MIN
	E2MAggSampleTypeMax = e2m.E2MAggSamples_SAMPLE_TYPE_MAX
)

// E2MAggregationType is a type for configuring the Events2Metrics service.
type E2MAggregationType = e2m.Aggregation_AggType

// E2MAggregation is a type for configuring the Events2Metrics service.
type E2MAggregation = e2m.Aggregation

// E2MAggregationHistogram is a type for configuring the Events2Metrics service.
type E2MAggregationHistogram = e2m.Aggregation_Histogram

// E2MAggregationSamples is a type for configuring the Events2Metrics service.
type E2MAggregationSamples = e2m.Aggregation_Samples

// E2MAggregationType values.
const (
	E2MAggregationTypeMin       = e2m.Aggregation_AGG_TYPE_MIN
	E2MAggregationTypeMax       = e2m.Aggregation_AGG_TYPE_MAX
	E2MAggregationTypeCount     = e2m.Aggregation_AGG_TYPE_COUNT
	E2MAggregationTypeAvg       = e2m.Aggregation_AGG_TYPE_AVG
	E2MAggregationTypeSum       = e2m.Aggregation_AGG_TYPE_SUM
	E2MAggregationTypeHistogram = e2m.Aggregation_AGG_TYPE_HISTOGRAM
	E2MAggregationTypeSamples   = e2m.Aggregation_AGG_TYPE_SAMPLES
)

// E2MType values.
const (
	E2MTypeUnspecified   = e2m.E2MType_E2M_TYPE_UNSPECIFIED
	E2MTypeLogs2Metrics  = e2m.E2MType_E2M_TYPE_LOGS2METRICS
	E2MTypeSpans2Metrics = e2m.E2MType_E2M_TYPE_SPANS2METRICS
)

// RPC values.
const (
	E2MCreateRPC                = e2m.Events2MetricService_CreateE2M_FullMethodName
	E2MListRPC                  = e2m.Events2MetricService_ListE2M_FullMethodName
	E2MReplaceRPC               = e2m.Events2MetricService_ReplaceE2M_FullMethodName
	E2MGetRPC                   = e2m.Events2MetricService_GetE2M_FullMethodName
	E2MDeleteRPC                = e2m.Events2MetricService_DeleteE2M_FullMethodName
	E2MAtomicBatchExecuteRPC    = e2m.Events2MetricService_AtomicBatchExecuteE2M_FullMethodName
	E2MListLabelsCardinalityRPC = e2m.Events2MetricService_ListLabelsCardinality_FullMethodName
	E2MGetLimtsRPC              = e2m.Events2MetricService_GetLimits_FullMethodName
)

const events2MetricsFeatureGroupID = "events2metrics"

// Events2MetricsClient is a client for the Coralogix Events2Metrics API. Read more at https://coralogix.com/docs/events2metrics/
type Events2MetricsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create Creates a new metric.
func (e Events2MetricsClient) Create(ctx context.Context, req *CreateE2MRequest) (*e2m.CreateE2MResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.CreateE2M(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MCreateRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// Get gets a metric.
func (e Events2MetricsClient) Get(ctx context.Context, req *GetE2MRequest) (*e2m.GetE2MResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.GetE2M(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MGetRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// Replace replaces a metric.
func (e Events2MetricsClient) Replace(ctx context.Context, req *ReplaceE2MRequest) (*e2m.ReplaceE2MResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.ReplaceE2M(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MReplaceRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// Delete deletes a metric.
func (e Events2MetricsClient) Delete(ctx context.Context, req *DeleteE2MRequest) (*e2m.DeleteE2MResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.DeleteE2M(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MDeleteRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// ListLabelsCardinality lists the cardinality of labels for a given metric.
func (e Events2MetricsClient) ListLabelsCardinality(ctx context.Context, req *ListLabelsCardinalityRequest) (*e2m.ListLabelsCardinalityResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.ListLabelsCardinality(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MListLabelsCardinalityRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// List lists all metrics
func (e Events2MetricsClient) List(ctx context.Context) (*e2m.ListE2MResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.ListE2M(callProperties.Ctx, &e2m.ListE2MRequest{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MListRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// AtomicBatchExecute executes a batch of atomic operations.
func (e Events2MetricsClient) AtomicBatchExecute(ctx context.Context, req *AtomicBatchExecuteE2MRequest) (*e2m.AtomicBatchExecuteE2MResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.AtomicBatchExecuteE2M(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MAtomicBatchExecuteRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// GetLimits lists all limits.
func (e Events2MetricsClient) GetLimits(ctx context.Context) (*e2m.GetLimitsResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := e2m.NewEvents2MetricServiceClient(conn)

	response, err := client.GetLimits(callProperties.Ctx, &e2m.GetLimitsRequest{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, E2MGetLimtsRPC, events2MetricsFeatureGroupID)
	}
	return response, nil
}

// NewEvents2MetricsClient creates a new Events2MetricsClient.
func NewEvents2MetricsClient(c *CallPropertiesCreator) *Events2MetricsClient {
	return &Events2MetricsClient{callPropertiesCreator: c}
}
