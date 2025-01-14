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

	enrichment "github.com/coralogix/coralogix-management-sdk/go/internal/coralogix/enrichment/v1"
)

// Enrichment is a column mapping for enrichments.
type Enrichment = enrichment.Enrichment

// GetEnrichmentsRequest is a request for getting enrichments.
type GetEnrichmentsRequest = enrichment.GetEnrichmentsRequest

// GetEnrichmentLimitRequest is a request for getting the enrichment limits.
type GetEnrichmentLimitRequest = enrichment.GetEnrichmentLimitRequest

// AddEnrichmentsRequest is a request to add enrichments.
type AddEnrichmentsRequest = enrichment.AddEnrichmentsRequest

// DeleteEnrichmentsRequest is a request to remove enrichments.
type DeleteEnrichmentsRequest = enrichment.RemoveEnrichmentsRequest

// EnrichmentTypeAws is a type of enrichment.
type EnrichmentTypeAws = enrichment.EnrichmentType_Aws

// EnrichmentTypeGeoIP is a type of enrichment.
type EnrichmentTypeGeoIP = enrichment.EnrichmentType_GeoIp

// EnrichmentTypeSuspiciousIP is a type of enrichment.
type EnrichmentTypeSuspiciousIP = enrichment.EnrichmentType_SuspiciousIp

// EnrichmentTypeCustomEnrichment is a type of enrichment.
type EnrichmentTypeCustomEnrichment = enrichment.EnrichmentType_CustomEnrichment

// EnrichmentType is a type of enrichment.
type EnrichmentType = enrichment.EnrichmentType

// AwsType is a type of enrichment.
type AwsType = enrichment.AwsType

// SuspiciousIPType is a type of enrichment.
type SuspiciousIPType = enrichment.SuspiciousIpType

// GeoIPType is a type of enrichment.
type GeoIPType = enrichment.GeoIpType

// CustomEnrichmentType is a type of enrichment.
type CustomEnrichmentType = enrichment.CustomEnrichmentType

// EnrichmentRequestModel is a inner type requests.
type EnrichmentRequestModel = enrichment.EnrichmentRequestModel

const enrichmentsFeatureGroupID = "enrichments"

// RPC names.
const (
	GetEnrichmentsRPC               = enrichment.EnrichmentService_GetEnrichments_FullMethodName
	AddEnrichmentsRPC               = enrichment.EnrichmentService_AddEnrichments_FullMethodName
	DeleteEnrichmentsRPC            = enrichment.EnrichmentService_RemoveEnrichments_FullMethodName
	GetEnrichmentLimitRPC           = enrichment.EnrichmentService_GetEnrichmentLimit_FullMethodName
	AtomicOverwriteEnrichmentsRPC   = enrichment.EnrichmentService_AtomicOverwriteEnrichments_FullMethodName
	GetCompanyEnrichmentSettingsRPC = enrichment.EnrichmentService_GetCompanyEnrichmentSettings_FullMethodName
)

// EnrichmentsClient is a client for the Coralogix Enrichments API.
type EnrichmentsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Add creates a new enrichment.
func (e EnrichmentsClient) Add(ctx context.Context, req *AddEnrichmentsRequest) (*enrichment.AddEnrichmentsResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := enrichment.NewEnrichmentServiceClient(conn)

	response, err := client.AddEnrichments(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, AddEnrichmentsRPC, enrichmentsFeatureGroupID)
	}
	return response, nil
}

// Delete deletes the specified enrichments.
func (e EnrichmentsClient) Delete(ctx context.Context, req *DeleteEnrichmentsRequest) error {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewEnrichmentServiceClient(conn)

	_, err = client.RemoveEnrichments(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return NewSdkAPIError(err, DeleteEnrichmentsRPC, "enrichments")
	}
	return nil
}

// List returns all enrichments.
func (e EnrichmentsClient) List(ctx context.Context, req *GetEnrichmentsRequest) (*enrichment.GetEnrichmentsResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewEnrichmentServiceClient(conn)

	response, err := client.GetEnrichments(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetEnrichmentsRPC, enrichmentsFeatureGroupID)
	}
	return response, nil
}

// GetLimits returns the enrichment limits.
func (e EnrichmentsClient) GetLimits(ctx context.Context) (*enrichment.GetEnrichmentLimitResponse, error) {
	callProperties, err := e.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewEnrichmentServiceClient(conn)

	response, err := client.GetEnrichmentLimit(callProperties.Ctx, &GetEnrichmentLimitRequest{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetEnrichmentLimitRPC, enrichmentsFeatureGroupID)
	}
	return response, nil
}

// NewEnrichmentClient creates a new enrichments client.
func NewEnrichmentClient(c *CallPropertiesCreator) *EnrichmentsClient {
	return &EnrichmentsClient{callPropertiesCreator: c}
}
