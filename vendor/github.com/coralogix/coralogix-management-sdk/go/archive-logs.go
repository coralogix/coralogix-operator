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

	archiveLogs "github.com/coralogix/coralogix-management-sdk/go/internal/coralogix/archive/v2"
)

// SetTargetRequest is a request to set the archive logs target.
type SetTargetRequest = archiveLogs.SetTargetRequest

// SetTargetRequestS3 is a request to set the archive logs target to S3.
type SetTargetRequestS3 = archiveLogs.SetTargetRequest_S3

// S3TargetSpec is an S3 target spec.
type S3TargetSpec = archiveLogs.S3TargetSpec

// SetTargetRequestIbmCos is a request to set the archive logs target to IBM COS.
type SetTargetRequestIbmCos = archiveLogs.SetTargetRequest_IbmCos

// Target is a target for storing archive logs.
type Target = archiveLogs.Target

// TargetS3 is a S3 target for storing archive logs.
type TargetS3 = archiveLogs.Target_S3

// TargetIbmCos is an IBM COS target for storing archive logs.
type TargetIbmCos = archiveLogs.Target_IbmCos

const archiveLogsFeatureGroupID = "logs"

// RPC names.
const (
	ArchiveLogsGetTargetRPC      = archiveLogs.TargetService_GetTarget_FullMethodName
	ArchiveLogsSetTargetRPC      = archiveLogs.TargetService_SetTarget_FullMethodName
	ArchiveLogsValidateTargetRPC = archiveLogs.TargetService_ValidateTarget_FullMethodName
)

// ArchiveLogsClient is a client for the Coralogix Archive Logs API.
type ArchiveLogsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Update updates the archive logs target.
func (c ArchiveLogsClient) Update(ctx context.Context, req *archiveLogs.SetTargetRequest) (*archiveLogs.SetTargetResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveLogs.NewTargetServiceClient(conn)

	response, err := client.SetTarget(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveLogsSetTargetRPC, archiveLogsFeatureGroupID)
	}
	return response, nil
}

// Get gets the archive logs target.
func (c ArchiveLogsClient) Get(ctx context.Context) (*archiveLogs.GetTargetResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveLogs.NewTargetServiceClient(conn)

	response, err := client.GetTarget(callProperties.Ctx, &archiveLogs.GetTargetRequest{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveLogsGetTargetRPC, archiveLogsFeatureGroupID)
	}
	return response, nil
}

// ValidateTarget validates the archive logs target.
func (c ArchiveLogsClient) ValidateTarget(ctx context.Context, req *archiveLogs.ValidateTargetRequest) (*archiveLogs.ValidateTargetResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveLogs.NewTargetServiceClient(conn)

	response, err := client.ValidateTarget(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveLogsValidateTargetRPC, archiveLogsFeatureGroupID)
	}
	return response, nil
}

// NewArchiveLogsClient creates a new archive logs client.
func NewArchiveLogsClient(c *CallPropertiesCreator) *ArchiveLogsClient {
	return &ArchiveLogsClient{callPropertiesCreator: c}
}
