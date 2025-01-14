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

	teams "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/aaa/organisations/v2"
)

// CreateTeamInOrgRequest is a request to create a team.
type CreateTeamInOrgRequest = teams.CreateTeamInOrgRequest

// UpdateTeamRequest is a request to update a team.
type UpdateTeamRequest = teams.UpdateTeamRequest

// GetTeamRequest is a request to get a team.
type GetTeamRequest = teams.GetTeamRequest

// DeleteTeamRequest is a request to delete a team.
type DeleteTeamRequest = teams.DeleteTeamRequest

// SetDailyQuotaRequest is a request to set the daily quota for a team.
type SetDailyQuotaRequest = teams.SetDailyQuotaRequest

// GetTeamQuotaRequest is a request to get the quota for a team.
type GetTeamQuotaRequest = teams.GetTeamQuotaRequest

// MoveQuotaRequest is a request to move the quota from one team to another.
type MoveQuotaRequest = teams.MoveQuotaRequest

// TeamID identifies a team.
type TeamID = teams.TeamId

const teamsFeatureGroupID = "aaa"

// RPC names.
const (
	CreateTeamInOrgRPC = teams.TeamService_CreateTeamInOrg_FullMethodName
	MoveQuotaRPC       = teams.TeamService_MoveQuota_FullMethodName
	GetTeamQuotaRPC    = teams.TeamService_GetTeamQuota_FullMethodName
	SetDailyQuotaRPC   = teams.TeamService_SetDailyQuota_FullMethodName
	UpdateTeamRPC      = teams.TeamService_UpdateTeam_FullMethodName
	GetTeamRPC         = teams.TeamService_GetTeam_FullMethodName
	DeleteTeamRPC      = teams.TeamService_DeleteTeam_FullMethodName
	ListTeamsRPC       = teams.TeamService_ListTeams_FullMethodName
)

// TeamsClient is a client for the Coralogix Teams API.
type TeamsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new team.
func (c TeamsClient) Create(ctx context.Context, req *CreateTeamInOrgRequest) (*teams.CreateTeamInOrgResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := teams.NewTeamServiceClient(conn)

	response, err := client.CreateTeamInOrg(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, CreateTeamInOrgRPC, teamsFeatureGroupID)
	}
	return response, nil
}

// Update updates a team.
func (c TeamsClient) Update(ctx context.Context, req *UpdateTeamRequest) (*teams.UpdateTeamResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := teams.NewTeamServiceClient(conn)

	response, err := client.UpdateTeam(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, UpdateTeamRPC, teamsFeatureGroupID)
	}
	return response, nil
}

// Get gets a team.
func (c TeamsClient) Get(ctx context.Context, req *GetTeamRequest) (*teams.GetTeamResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := teams.NewTeamServiceClient(conn)

	response, err := client.GetTeam(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetTeamRPC, teamsFeatureGroupID)
	}
	return response, nil
}

// Delete deletes a team.
func (c TeamsClient) Delete(ctx context.Context, req *DeleteTeamRequest) (*teams.DeleteTeamResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := teams.NewTeamServiceClient(conn)

	response, err := client.DeleteTeam(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DeleteTeamRPC, teamsFeatureGroupID)
	}
	return response, nil
}

// SetDailyQuota sets the daily quota for a team.
func (c TeamsClient) SetDailyQuota(ctx context.Context, req *SetDailyQuotaRequest) (*teams.SetDailyQuotaResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := teams.NewTeamServiceClient(conn)

	response, err := client.SetDailyQuota(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, SetDailyQuotaRPC, teamsFeatureGroupID)
	}
	return response, nil
}

// GetQuota gets the quota for a team.
func (c TeamsClient) GetQuota(ctx context.Context, req *GetTeamQuotaRequest) (*teams.GetTeamQuotaResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := teams.NewTeamServiceClient(conn)

	response, err := client.GetTeamQuota(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetTeamQuotaRPC, teamsFeatureGroupID)
	}
	return response, nil
}

// MoveQuota moves the quota from one team to another.
func (c TeamsClient) MoveQuota(ctx context.Context, req *MoveQuotaRequest) (*teams.MoveQuotaResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := teams.NewTeamServiceClient(conn)

	response, err := client.MoveQuota(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, MoveQuotaRPC, teamsFeatureGroupID)
	}
	return response, nil
}

// NewTeamsClient creates a new teams client.
func NewTeamsClient(c *CallPropertiesCreator) *TeamsClient {
	return &TeamsClient{callPropertiesCreator: c}
}
