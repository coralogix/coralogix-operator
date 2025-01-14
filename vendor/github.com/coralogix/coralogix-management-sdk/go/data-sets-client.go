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

// CreateDataSetRequest is a request to create a data set.
type CreateDataSetRequest = enrichment.CreateCustomEnrichmentRequest

// GetDataSetRequest is a request to get a data set.
type GetDataSetRequest = enrichment.GetCustomEnrichmentRequest

// UpdateDataSetRequest is a request to update a data set.
type UpdateDataSetRequest = enrichment.UpdateCustomEnrichmentRequest

// DeleteDataSetRequest is a request to delete a data set.
type DeleteDataSetRequest = enrichment.DeleteCustomEnrichmentRequest

// ListDataSetsRequest is a request to list data sets.
type ListDataSetsRequest = enrichment.GetCustomEnrichmentsRequest

// DataSet is a custom data set (enrichment).
type DataSet = enrichment.CustomEnrichment

// File is a file. Can be either FileBinary or FileTextual.
type File = enrichment.File

// FileBinary is a binary file.
type FileBinary = enrichment.File_Binary

// FileTextual is a textual file.
type FileTextual = enrichment.File_Textual

const dataSetsFeatureGroupID = "logs"

// RPC names.
const (
	CreateDataSetRPC = enrichment.CustomEnrichmentService_CreateCustomEnrichment_FullMethodName
	GetDataSetRPC    = enrichment.CustomEnrichmentService_GetCustomEnrichment_FullMethodName
	UpdateDataSetRPC = enrichment.CustomEnrichmentService_UpdateCustomEnrichment_FullMethodName
	DeleteDataSetRPC = enrichment.CustomEnrichmentService_DeleteCustomEnrichment_FullMethodName
	GetCDataSetRPC   = enrichment.CustomEnrichmentService_GetCustomEnrichments_FullMethodName
)

// DataSetClient is a client for the Coralogix Data Sets API.
type DataSetClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new data set.
func (d DataSetClient) Create(ctx context.Context, req *CreateDataSetRequest) (*enrichment.CreateCustomEnrichmentResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewCustomEnrichmentServiceClient(conn)

	response, err := client.CreateCustomEnrichment(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, CreateDataSetRPC, dataSetsFeatureGroupID)
	}
	return response, nil
}

// Get gets a data set.
func (d DataSetClient) Get(ctx context.Context, req *GetDataSetRequest) (*enrichment.GetCustomEnrichmentResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewCustomEnrichmentServiceClient(conn)

	response, err := client.GetCustomEnrichment(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetDataSetRPC, dataSetsFeatureGroupID)
	}
	return response, nil
}

// Update updates a data set.
func (d DataSetClient) Update(ctx context.Context, req *UpdateDataSetRequest) (*enrichment.UpdateCustomEnrichmentResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewCustomEnrichmentServiceClient(conn)

	response, err := client.UpdateCustomEnrichment(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, UpdateDataSetRPC, dataSetsFeatureGroupID)
	}
	return response, nil
}

// Delete deletes a data set.
func (d DataSetClient) Delete(ctx context.Context, req *DeleteDataSetRequest) (*enrichment.DeleteCustomEnrichmentResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewCustomEnrichmentServiceClient(conn)

	return client.DeleteCustomEnrichment(callProperties.Ctx, req, callProperties.CallOptions...)
}

// List retrieves all data sets.
func (d DataSetClient) List(ctx context.Context, req *ListDataSetsRequest) (*enrichment.GetCustomEnrichmentsResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := enrichment.NewCustomEnrichmentServiceClient(conn)

	response, err := client.GetCustomEnrichments(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetCDataSetRPC, dataSetsFeatureGroupID)
	}
	return response, nil
}

// NewDataSetClient creates a new data set client.
func NewDataSetClient(c *CallPropertiesCreator) *DataSetClient {
	return &DataSetClient{callPropertiesCreator: c}
}
