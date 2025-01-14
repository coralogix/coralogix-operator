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

	dataUsage "github.com/coralogix/coralogix-management-sdk/go/internal/coralogix/datausage/v2"
)

// GetSpansCountRequest is a request to get the spans count.
type GetSpansCountRequest = dataUsage.GetSpansCountRequest

// GetLogsCountRequest is a request to get the logs count.
type GetLogsCountRequest = dataUsage.GetLogsCountRequest

// GetDataUsageMetricsExportStatusRequest is a request to get the metrics export status.
type GetDataUsageMetricsExportStatusRequest = dataUsage.GetDataUsageMetricsExportStatusRequest

// UpdateDataUsageMetricsExportStatusRequest is a request to update the metrics export status.
type UpdateDataUsageMetricsExportStatusRequest = dataUsage.UpdateDataUsageMetricsExportStatusRequest

// GetDataUsageRequest is a request to get the data usage.
type GetDataUsageRequest = dataUsage.GetDataUsageRequest

// SpansCountStream is a response to the GetSpansCountRequest.
type SpansCountStream = dataUsage.DataUsageService_GetSpansCountClient

// LogsCountStream is a response to the GetLogsCountRequest.
type LogsCountStream = dataUsage.DataUsageService_GetLogsCountClient

// GetDataUsageMetricsExportStatusResponse is a response to the GetDataUsageMetricsExportStatusRequest.
type GetDataUsageMetricsExportStatusResponse = dataUsage.GetDataUsageMetricsExportStatusResponse

// UpdateDataUsageMetricsExportStatusResponse is a response to the UpdateDataUsageMetricsExportStatusRequest.
type UpdateDataUsageMetricsExportStatusResponse = dataUsage.UpdateDataUsageMetricsExportStatusResponse

// DataUsageStream is a response to the GetDataUsageRequest.
type DataUsageStream = dataUsage.DataUsageService_GetDataUsageClient

// DateRange is a date range.
type DateRange = dataUsage.DateRange

// ScopesFilter is a filter for scopes.
type ScopesFilter = dataUsage.ScopesFilter

// Dimension is a dimension.
type Dimension = dataUsage.Dimension

// AggregateBy is a type of aggregation.
type AggregateBy = dataUsage.AggregateBy

const dataPlansFeatureGroupID = "dataplans"

// AggregateBy enum values.
const (
	AggregateByUnspecified  AggregateBy = dataUsage.AggregateBy_AGGREGATE_BY_UNSPECIFIED
	AggregateByApplicatiokn AggregateBy = dataUsage.AggregateBy_AGGREGATE_BY_APPLICATION
	AggregateBySubsystem    AggregateBy = dataUsage.AggregateBy_AGGREGATE_BY_SUBSYSTEM
	AggreateByPillar        AggregateBy = dataUsage.AggregateBy_AGGREGATE_BY_PILLAR
	AggregateByPriority     AggregateBy = dataUsage.AggregateBy_AGGREGATE_BY_PRIORITY
)

// RPC names.
const (
	GetSpansCountRPC                      = dataUsage.DataUsageService_GetSpansCount_FullMethodName
	GetLogsCountRPC                       = dataUsage.DataUsageService_GetLogsCount_FullMethodName
	GetDataUsageMetricsExportStatusRPC    = dataUsage.DataUsageService_GetDataUsageMetricsExportStatus_FullMethodName
	UpdateDataUsageMetricsExportStatusRPC = dataUsage.DataUsageService_UpdateDataUsageMetricsExportStatus_FullMethodName
	GetDataUsageRPC                       = dataUsage.DataUsageService_GetDataUsage_FullMethodName
)

// DataUsageClient is a client for the Coralogix Data Usage API.
type DataUsageClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// GetSpansCount gets the spans count as a stream.
func (c DataUsageClient) GetSpansCount(ctx context.Context, req *dataUsage.GetSpansCountRequest) (dataUsage.DataUsageService_GetSpansCountClient, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dataUsage.NewDataUsageServiceClient(conn)

	response, err := client.GetSpansCount(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetSpansCountRPC, dataPlansFeatureGroupID)
	}
	return response, nil
}

// GetLogsCount gets the logs count as a stream.
func (c DataUsageClient) GetLogsCount(ctx context.Context, req *dataUsage.GetLogsCountRequest) (dataUsage.DataUsageService_GetLogsCountClient, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dataUsage.NewDataUsageServiceClient(conn)
	response, err := client.GetLogsCount(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetLogsCountRPC, dataPlansFeatureGroupID)
	}
	return response, nil
}

// GetDataUsageMetricsExportStatus gets the metrics export status.
func (c DataUsageClient) GetDataUsageMetricsExportStatus(ctx context.Context, req *dataUsage.GetDataUsageMetricsExportStatusRequest) (*dataUsage.GetDataUsageMetricsExportStatusResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dataUsage.NewDataUsageServiceClient(conn)
	response, err := client.GetDataUsageMetricsExportStatus(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetDataUsageMetricsExportStatusRPC, dataPlansFeatureGroupID)
	}
	return response, nil
}

// UpdateDataUsageMetricsExportStatus updates the metrics export status.
func (c DataUsageClient) UpdateDataUsageMetricsExportStatus(ctx context.Context, req *dataUsage.UpdateDataUsageMetricsExportStatusRequest) (*dataUsage.UpdateDataUsageMetricsExportStatusResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dataUsage.NewDataUsageServiceClient(conn)
	response, err := client.UpdateDataUsageMetricsExportStatus(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, UpdateDataUsageMetricsExportStatusRPC, dataPlansFeatureGroupID)
	}
	return response, nil
}

// GetDataUsage gets the data usage as a stream.
func (c DataUsageClient) GetDataUsage(ctx context.Context, req *dataUsage.GetDataUsageRequest) (dataUsage.DataUsageService_GetDataUsageClient, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dataUsage.NewDataUsageServiceClient(conn)
	response, err := client.GetDataUsage(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetDataUsageRPC, dataPlansFeatureGroupID)
	}
	return response, nil
}

// NewDataUsageClient creates a new DataUsageClient.
func NewDataUsageClient(c *CallPropertiesCreator) *DataUsageClient {
	return &DataUsageClient{callPropertiesCreator: c}
}
