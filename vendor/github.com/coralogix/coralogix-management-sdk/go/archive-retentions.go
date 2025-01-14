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

	archiveRetention "github.com/coralogix/coralogix-management-sdk/go/internal/coralogix/archive/v1"
)

// UpdateRetentionsRequest is a type for archive retention.
type UpdateRetentionsRequest = archiveRetention.UpdateRetentionsRequest

// GetRetentionsRequest is a type for archive retention.
type GetRetentionsRequest = archiveRetention.GetRetentionsRequest

// RetentionUpdateElement is a type for archive retention.
type RetentionUpdateElement = archiveRetention.RetentionUpdateElement

// GetRetentionsEnabledRequest is a request type.
type GetRetentionsEnabledRequest = archiveRetention.GetRetentionsEnabledRequest

// ActivateRetentionsRequest is a request type.
type ActivateRetentionsRequest = archiveRetention.ActivateRetentionsRequest

// Retention is a type for archive retention.
type Retention = archiveRetention.Retention

const archiveFeatureGroupID = "archive"

// RPC values.
const (
	ArchiveRetentionGetRetentionsRPC        = archiveRetention.RetentionsService_GetRetentions_FullMethodName
	ArchiveRetentionUpdateRetentionsRPC     = archiveRetention.RetentionsService_UpdateRetentions_FullMethodName
	ArchiveRetentionActivateRetentionsRPC   = archiveRetention.RetentionsService_ActivateRetentions_FullMethodName
	ArchiveRetentionGetRetentionsEnabledRPC = archiveRetention.RetentionsService_GetRetentionsEnabled_FullMethodName
)

// ArchiveRetentionsClient is a client for the Coralogix Archive Retentions API.
type ArchiveRetentionsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Get gets the archive retentions.
func (c ArchiveRetentionsClient) Get(ctx context.Context, req *archiveRetention.GetRetentionsRequest) (*archiveRetention.GetRetentionsResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveRetention.NewRetentionsServiceClient(conn)

	response, err := client.GetRetentions(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveRetentionGetRetentionsRPC, archiveFeatureGroupID)
	}
	return response, nil
}

// Update updates the archive retentions.
func (c ArchiveRetentionsClient) Update(ctx context.Context, req *UpdateRetentionsRequest) (*archiveRetention.UpdateRetentionsResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveRetention.NewRetentionsServiceClient(conn)

	response, err := client.UpdateRetentions(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveRetentionUpdateRetentionsRPC, archiveFeatureGroupID)
	}
	return response, nil
}

// Activate activates the archive retentions.
func (c ArchiveRetentionsClient) Activate(ctx context.Context, req *ActivateRetentionsRequest) (*archiveRetention.ActivateRetentionsResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveRetention.NewRetentionsServiceClient(conn)

	response, err := client.ActivateRetentions(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveRetentionActivateRetentionsRPC, archiveFeatureGroupID)
	}
	return response, nil
}

// GetEnabled returns a boolean that signals whether archive retentions are enabled.
func (c ArchiveRetentionsClient) GetEnabled(ctx context.Context, req *GetRetentionsEnabledRequest) (*archiveRetention.GetRetentionsEnabledResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveRetention.NewRetentionsServiceClient(conn)

	response, err := client.GetRetentionsEnabled(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveRetentionGetRetentionsEnabledRPC, archiveFeatureGroupID)
	}
	return response, nil

}

// NewArchiveRetentionsClient Creates a new archive retentions client.
func NewArchiveRetentionsClient(c *CallPropertiesCreator) *ArchiveRetentionsClient {
	return &ArchiveRetentionsClient{callPropertiesCreator: c}
}
