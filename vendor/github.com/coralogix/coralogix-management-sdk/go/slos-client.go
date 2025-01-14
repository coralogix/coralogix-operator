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

	slos "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/apm/services/v1"
)

// ServiceSlo is an SLO.
type ServiceSlo = slos.ServiceSlo

// CreateServiceSloRequest is a request to create an SLO.
type CreateServiceSloRequest = slos.CreateServiceSloRequest

// ServiceSloErrorSli is an SLO error SLI.
type ServiceSloErrorSli = slos.ServiceSlo_ErrorSli

// ErrorSli is an SLO error SLI.
type ErrorSli = slos.ErrorSli

// LatencySli is an SLO latency SLI.
type LatencySli = slos.LatencySli

// ServiceSloLatencySli is a something.
type ServiceSloLatencySli = slos.ServiceSlo_LatencySli

// GetServiceSloRequest is a request to get an SLO.
type GetServiceSloRequest = slos.GetServiceSloRequest

// ReplaceServiceSloRequest is a request to replace an SLO.
type ReplaceServiceSloRequest = slos.ReplaceServiceSloRequest

// DeleteServiceSloRequest is a request to delete an SLO.
type DeleteServiceSloRequest = slos.DeleteServiceSloRequest

// SliFilter is an SLO filter.
type SliFilter = slos.SliFilter

// ThresholdSymbol is an SLO threshold symbol.
type ThresholdSymbol = slos.ThresholdSymbol

const infraMonitoringFeatureGroupID = "infra-monitoring"

// SloThresholdSymbol values.
const (
	SloThresholdSymbolGreater        = slos.ThresholdSymbol_THRESHOLD_SYMBOL_GREATER
	SloThresholdSymbolGreaterOrEqual = slos.ThresholdSymbol_THRESHOLD_SYMBOL_GREATER_OR_EQUAL
	SloThresholdSymbolLess           = slos.ThresholdSymbol_THRESHOLD_SYMBOL_LESS
	SloThresholdSymbolLessOrEqual    = slos.ThresholdSymbol_THRESHOLD_SYMBOL_LESS_OR_EQUAL
	SloThresholdSymbolEqual          = slos.ThresholdSymbol_THRESHOLD_SYMBOL_EQUAL
)

// SloPeriod is a SLO period.
type SloPeriod = slos.SloPeriod

// SloStatus is a SLO status.
type SloStatus = slos.SloStatus

// SLO period values.
const (
	SloPeriodUnspecified = slos.SloPeriod_SLO_PERIOD_UNSPECIFIED
	SloPeriod7Days       = slos.SloPeriod_SLO_PERIOD_7_DAYS
	SloPeriod14Days      = slos.SloPeriod_SLO_PERIOD_14_DAYS
	SloPeriod30Days      = slos.SloPeriod_SLO_PERIOD_30_DAYS
)

// CompareType is an SLO compare type.
type CompareType = slos.CompareType

// SloCompareType values.
const (
	SloCompareTypeUnspecified = slos.CompareType_COMPARE_TYPE_UNSPECIFIED
	SloCompareTypeIs          = slos.CompareType_COMPARE_TYPE_IS
	SloCompareTypeStartsWith  = slos.CompareType_COMPARE_TYPE_START_WITH
	SloCompareTypeEndsWith    = slos.CompareType_COMPARE_TYPE_ENDS_WITH
	SloCompareTypeIncludes    = slos.CompareType_COMPARE_TYPE_INCLUDES
)

// SloStatus values.
const (
	SloStatusUnspecified = slos.SloStatus_SLO_STATUS_UNSPECIFIED
	SloStatusOk          = slos.SloStatus_SLO_STATUS_OK
	SloStatusBreached    = slos.SloStatus_SLO_STATUS_BREACHED
)

// RPC names.
const (
	SloGetRPC      = slos.ServiceSloService_GetServiceSlo_FullMethodName
	SloCreateRPC   = slos.ServiceSloService_CreateServiceSlo_FullMethodName
	SloReplaceRPC  = slos.ServiceSloService_ReplaceServiceSlo_FullMethodName
	SloDeleteRPC   = slos.ServiceSloService_DeleteServiceSlo_FullMethodName
	SloListRPC     = slos.ServiceSloService_ListServiceSlos_FullMethodName
	SloBatchGetRPC = slos.ServiceSloService_BatchGetServiceSlos_FullMethodName
)

// SLOsClient is a client for the Coralogix SLOs API.
type SLOsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new SLO.
func (c SLOsClient) Create(ctx context.Context, req *slos.CreateServiceSloRequest) (*slos.CreateServiceSloResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := slos.NewServiceSloServiceClient(conn)

	response, err := client.CreateServiceSlo(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, SloCreateRPC, infraMonitoringFeatureGroupID)
	}
	return response, nil
}

// Get gets the specified SLO.
func (c SLOsClient) Get(ctx context.Context, req *slos.GetServiceSloRequest) (*slos.GetServiceSloResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := slos.NewServiceSloServiceClient(conn)

	response, err := client.GetServiceSlo(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, SloGetRPC, infraMonitoringFeatureGroupID)
	}
	return response, nil
}

// Update updates the specified SLO.
func (c SLOsClient) Update(ctx context.Context, req *slos.ReplaceServiceSloRequest) (*slos.ReplaceServiceSloResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := slos.NewServiceSloServiceClient(conn)

	response, err := client.ReplaceServiceSlo(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, SloReplaceRPC, infraMonitoringFeatureGroupID)
	}
	return response, nil
}

// Delete deletes the specified SLO.
func (c SLOsClient) Delete(ctx context.Context, req *slos.DeleteServiceSloRequest) (*slos.DeleteServiceSloResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := slos.NewServiceSloServiceClient(conn)

	response, err := client.DeleteServiceSlo(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, SloDeleteRPC, infraMonitoringFeatureGroupID)
	}
	return response, nil
}

// GetBulk gets multiple SLOs in a single call.
func (c SLOsClient) GetBulk(ctx context.Context, req *slos.BatchGetServiceSlosRequest) (*slos.BatchGetServiceSlosResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := slos.NewServiceSloServiceClient(conn)

	response, err := client.BatchGetServiceSlos(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, SloBatchGetRPC, infraMonitoringFeatureGroupID)
	}
	return response, nil
}

// List lists all service SLOs.
func (c SLOsClient) List(ctx context.Context, req *slos.ListServiceSlosRequest) (*slos.ListServiceSlosResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := slos.NewServiceSloServiceClient(conn)

	response, err := client.ListServiceSlos(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, SloListRPC, infraMonitoringFeatureGroupID)
	}
	return response, nil
}

// NewSLOsClient creates a new SLOs client.
func NewSLOsClient(c *CallPropertiesCreator) *SLOsClient {
	return &SLOsClient{callPropertiesCreator: c}
}
