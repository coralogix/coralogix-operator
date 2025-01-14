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

	dashboards "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/dashboards/v1"
)

// ListDashboardFolderRequest is a request to get a dashboard folders.
type ListDashboardFolderRequest = dashboards.ListDashboardFoldersRequest

// CreateDashboardFolderRequest is a request to create a dashboards folder.
type CreateDashboardFolderRequest = dashboards.CreateDashboardFolderRequest

// DashboardFolder is a dashboard folder.
type DashboardFolder = dashboards.DashboardFolder

// GetDashboardFolderRequest is a request to get a dashboard folders.
type GetDashboardFolderRequest = dashboards.GetDashboardFolderRequest

// ReplaceDashboardFolderRequest is a request to replace a dashboard folder.
type ReplaceDashboardFolderRequest = dashboards.ReplaceDashboardFolderRequest

// DeleteDashboardFolderRequest is a request to delete a dashboard folder.
type DeleteDashboardFolderRequest = dashboards.DeleteDashboardFolderRequest

const dashboardFoldersFeatureGroupID = "dashboards"

// RPC names.
const (
	DashboardFoldersListDashboardFoldersRPC   = dashboards.DashboardFoldersService_ListDashboardFolders_FullMethodName
	DashboardFoldersGetDashboardFolderRPC     = dashboards.DashboardFoldersService_GetDashboardFolder_FullMethodName
	DashboardFoldersCreateDashboardFolderRPC  = dashboards.DashboardFoldersService_CreateDashboardFolder_FullMethodName
	DashboardFoldersReplaceDashboardFolderRPC = dashboards.DashboardFoldersService_ReplaceDashboardFolder_FullMethodName
	DashboardFoldersDeleteDashboardFolderRPC  = dashboards.DashboardFoldersService_DeleteDashboardFolder_FullMethodName
)

// DashboardsFoldersClient is a client for the Coralogix Dashboards Folders API.
type DashboardsFoldersClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new dashboard folder.
func (c DashboardsFoldersClient) Create(ctx context.Context, req *dashboards.CreateDashboardFolderRequest) (*dashboards.CreateDashboardFolderResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardFoldersServiceClient(conn)

	response, err := client.CreateDashboardFolder(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DashboardFoldersCreateDashboardFolderRPC, dashboardFoldersFeatureGroupID)
	}
	return response, nil
}

// Get dashboard folder details.
func (c DashboardsFoldersClient) Get(ctx context.Context, req *dashboards.GetDashboardFolderRequest) (*dashboards.GetDashboardFolderResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardFoldersServiceClient(conn)

	response, err := client.GetDashboardFolder(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DashboardFoldersGetDashboardFolderRPC, dashboardFoldersFeatureGroupID)
	}
	return response, nil
}

// List gets all dashboard folders.
func (c DashboardsFoldersClient) List(ctx context.Context) (*dashboards.ListDashboardFoldersResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetUserLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardFoldersServiceClient(conn)

	response, err := client.ListDashboardFolders(callProperties.Ctx, &dashboards.ListDashboardFoldersRequest{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DashboardFoldersListDashboardFoldersRPC, dashboardFoldersFeatureGroupID)
	}
	return response, nil
}

// Replace updates a dashboard folder.
func (c DashboardsFoldersClient) Replace(ctx context.Context, req *dashboards.ReplaceDashboardFolderRequest) (*dashboards.ReplaceDashboardFolderResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardFoldersServiceClient(conn)

	response, err := client.ReplaceDashboardFolder(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DashboardFoldersReplaceDashboardFolderRPC, dashboardFoldersFeatureGroupID)
	}
	return response, nil
}

// Delete deletes a dashboard folder.
func (c DashboardsFoldersClient) Delete(ctx context.Context, req *dashboards.DeleteDashboardFolderRequest) (*dashboards.DeleteDashboardFolderResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardFoldersServiceClient(conn)

	response, err := client.DeleteDashboardFolder(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DashboardFoldersDeleteDashboardFolderRPC, dashboardFoldersFeatureGroupID)
	}
	return response, nil
}

// NewDashboardsFoldersClient Creates a new DashboardsFoldersClient.
func NewDashboardsFoldersClient(c *CallPropertiesCreator) *DashboardsFoldersClient {
	return &DashboardsFoldersClient{callPropertiesCreator: c}
}
