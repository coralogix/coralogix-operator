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

	apikeys "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/aaa/apikeys/v3"
)

// CreateAPIKeyRequest is a request to create an API key.
type CreateAPIKeyRequest = apikeys.CreateApiKeyRequest

// CreateAPIKeyResponse is a response to the API keys request.
type CreateAPIKeyResponse = apikeys.CreateApiKeyResponse

// GetAPIKeyRequest is a request to get an API key.
type GetAPIKeyRequest = apikeys.GetApiKeyRequest

// GetAPIKeyResponse is a response to the API keys request.
type GetAPIKeyResponse = apikeys.GetApiKeyResponse

// UpdateAPIKeyRequest is a request to update an API key.
type UpdateAPIKeyRequest = apikeys.UpdateApiKeyRequest

// UpdateAPIKeyResponse is a response to the API keys request.
type UpdateAPIKeyResponse = apikeys.UpdateApiKeyResponse

// DeleteAPIKeyRequest is a request to delete an API key.
type DeleteAPIKeyRequest = apikeys.DeleteApiKeyRequest

// DeleteAPIKeyResponse is a response to the API keys request.
type DeleteAPIKeyResponse = apikeys.DeleteApiKeyResponse

// APIKeyPermissions is a set of permissions for an API key.
type APIKeyPermissions = apikeys.CreateApiKeyRequest_KeyPermissions

// APIKeyPermissionsUpdate is a set of permissions for an API key on update.
type APIKeyPermissionsUpdate = apikeys.UpdateApiKeyRequest_Permissions

// APIKeyPresetsUpdate is a set of presets for an API key on update.
type APIKeyPresetsUpdate = apikeys.UpdateApiKeyRequest_Presets

// Owner is an owner of an API key.
type Owner = apikeys.Owner

// OwnerUserID is an owner user ID.
type OwnerUserID = apikeys.Owner_UserId

// OwnerTeamID is an owner team ID.
type OwnerTeamID = apikeys.Owner_TeamId

// OwnerOrganisationID is an owner organisation ID.
type OwnerOrganisationID = apikeys.Owner_OrganisationId

const apiKeysFeatureGroupID = "aaa"

// RPC method names.
const (
	CreateAPIKeyRPC = apikeys.ApiKeysService_CreateApiKey_FullMethodName
	GetAPIKeyRPC    = apikeys.ApiKeysService_GetApiKey_FullMethodName
	UpdateAPIKeyRPC = apikeys.ApiKeysService_UpdateApiKey_FullMethodName
	DeleteAPIKeyRPC = apikeys.ApiKeysService_DeleteApiKey_FullMethodName
)

// ApikeysClient is a client for the Coralogix API keys API.
type ApikeysClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new API key.
func (t ApikeysClient) Create(ctx context.Context, req *apikeys.CreateApiKeyRequest) (*apikeys.CreateApiKeyResponse, error) {
	callProperties, err := t.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := apikeys.NewApiKeysServiceClient(conn)

	response, err := client.CreateApiKey(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, CreateAPIKeyRPC, apiKeysFeatureGroupID)
	}
	return response, nil
}

// Get gets an API key.
func (t ApikeysClient) Get(ctx context.Context, req *apikeys.GetApiKeyRequest) (*apikeys.GetApiKeyResponse, error) {
	callProperties, err := t.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := apikeys.NewApiKeysServiceClient(conn)

	response, err := client.GetApiKey(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetAPIKeyRPC, apiKeysFeatureGroupID)
	}
	return response, nil
}

// Update updates an API key.
func (t ApikeysClient) Update(ctx context.Context, req *apikeys.UpdateApiKeyRequest) (*apikeys.UpdateApiKeyResponse, error) {
	callProperties, err := t.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := apikeys.NewApiKeysServiceClient(conn)

	response, err := client.UpdateApiKey(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, UpdateAPIKeyRPC, apiKeysFeatureGroupID)
	}
	return response, nil
}

// Delete deletes an API key.
func (t ApikeysClient) Delete(ctx context.Context, req *apikeys.DeleteApiKeyRequest) (*apikeys.DeleteApiKeyResponse, error) {
	callProperties, err := t.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := apikeys.NewApiKeysServiceClient(conn)

	response, err := client.DeleteApiKey(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DeleteAPIKeyRPC, apiKeysFeatureGroupID)
	}
	return response, nil
}

// NewAPIKeysClient creates a new API keys client.
func NewAPIKeysClient(c *CallPropertiesCreator) *ApikeysClient {
	return &ApikeysClient{callPropertiesCreator: c}
}
