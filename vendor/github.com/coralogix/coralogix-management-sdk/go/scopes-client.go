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

	scopes "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/scopes/v1"
)

// CreateScopeRequest is a request to create a new scope
type CreateScopeRequest = scopes.CreateScopeRequest

// GetTeamScopesByIDsRequest is a request to get a scope by its ID
type GetTeamScopesByIDsRequest = scopes.GetTeamScopesByIdsRequest

// GetTeamScopesRequest is a request to list all scopes for the current team
type GetTeamScopesRequest = scopes.GetTeamScopesRequest

// GetScopesResponse is a response containing a list of scopes
type GetScopesResponse = scopes.GetScopesResponse

// UpdateScopeRequest is a request to update a scope
type UpdateScopeRequest = scopes.UpdateScopeRequest

// DeleteScopeRequest is a request to delete a scope
type DeleteScopeRequest = scopes.DeleteScopeRequest

// ScopeFilter is a filter for a scope
type ScopeFilter = scopes.Filter

// EntityTypeValueLookup is an entity type value lookup.
var EntityTypeValueLookup = scopes.EntityType_value

// EntityTypeNameLookup is an entity type name lookup.
var EntityTypeNameLookup = scopes.EntityType_name

const scopesFeatureGroupID = "aaa"

// EntityType is an entity type.
type EntityType = scopes.EntityType

// EntityType values
const (
	EntityTypeUnspecified = scopes.EntityType_ENTITY_TYPE_UNSPECIFIED
	EntityTypeLogs        = scopes.EntityType_ENTITY_TYPE_LOGS
	EntityTypeSpans       = scopes.EntityType_ENTITY_TYPE_SPANS
)

// RPC Name values
const (
	CreateScopeRPC        = scopes.ScopesService_CreateScope_FullMethodName
	DeleteScopeRPC        = scopes.ScopesService_DeleteScope_FullMethodName
	GetTeamScopesRPC      = scopes.ScopesService_GetTeamScopes_FullMethodName
	GetTeamScopesByIDsRPC = scopes.ScopesService_GetTeamScopesByIds_FullMethodName
	UpdateScopeRPC        = scopes.ScopesService_UpdateScope_FullMethodName
)

// ScopesClient is a client for the scopes service
type ScopesClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new scope
func (c ScopesClient) Create(ctx context.Context, req *CreateScopeRequest) (*scopes.CreateScopeResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := scopes.NewScopesServiceClient(conn)

	response, err := client.CreateScope(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, CreateScopeRPC, scopesFeatureGroupID)
	}
	return response, nil
}

// Get gets a scope by its ID
func (c ScopesClient) Get(ctx context.Context, req *GetTeamScopesByIDsRequest) (*scopes.GetScopesResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := scopes.NewScopesServiceClient(conn)

	response, err := client.GetTeamScopesByIds(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetTeamScopesByIDsRPC, scopesFeatureGroupID)
	}
	return response, nil
}

// List lists all scopes for the current team
func (c ScopesClient) List(ctx context.Context, req *GetTeamScopesRequest) (*scopes.GetScopesResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := scopes.NewScopesServiceClient(conn)

	response, err := client.GetTeamScopes(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetTeamScopesRPC, scopesFeatureGroupID)
	}
	return response, nil
}

// Update updates a scope
func (c ScopesClient) Update(ctx context.Context, req *UpdateScopeRequest) (*scopes.UpdateScopeResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := scopes.NewScopesServiceClient(conn)

	response, err := client.UpdateScope(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, UpdateScopeRPC, scopesFeatureGroupID)
	}
	return response, nil
}

// Delete deletes a scope
func (c ScopesClient) Delete(ctx context.Context, req *DeleteScopeRequest) (*scopes.DeleteScopeResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := scopes.NewScopesServiceClient(conn)

	response, err := client.DeleteScope(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DeleteScopeRPC, scopesFeatureGroupID)
	}
	return response, nil
}

// NewScopesClient creates a new ScopesClient
func NewScopesClient(c *CallPropertiesCreator) *ScopesClient {
	return &ScopesClient{callPropertiesCreator: c}
}
