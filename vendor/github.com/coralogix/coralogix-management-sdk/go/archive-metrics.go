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

	archiveMetrics "github.com/coralogix/coralogix-management-sdk/go/internal/coralogix/metrics"

	"google.golang.org/protobuf/types/known/emptypb"
)

// ConfigureTenantRequest is a request to configure the tenant.
type ConfigureTenantRequest = archiveMetrics.ConfigureTenantRequest

// UpdateTenantRequest is a request to update the tenant.
type UpdateTenantRequest = archiveMetrics.UpdateRequest

// ValidateBucketRequest is a request to validate the bucket.
type ValidateBucketRequest = archiveMetrics.ValidateBucketRequest

// GetTenantConfigRequest is a request to get the tenant configuration.
type GetTenantConfigRequest = archiveMetrics.GetTenantConfigRequest

// ConfigureTenantRequestS3 is a request to configure the tenant with S3.
type ConfigureTenantRequestS3 = archiveMetrics.ConfigureTenantRequest_S3

// ConfigureTenantRequestIbm is a request to configure the tenant with Ibm.
type ConfigureTenantRequestIbm = archiveMetrics.ConfigureTenantRequest_Ibm

// ArchiveIbmConfigV2 is an Ibm configuration for the archive.
type ArchiveIbmConfigV2 = archiveMetrics.IbmConfigV2

// ArchiveS3Config is an S3 configuration for the archive.
type ArchiveS3Config = archiveMetrics.S3Config

// ValidateBucketRequestS3 is a request to validate the S3 bucket.
type ValidateBucketRequestS3 = archiveMetrics.ValidateBucketRequest_S3

// ValidateBucketRequestIbm is a request to validate the IBM storage.
type ValidateBucketRequestIbm = archiveMetrics.ValidateBucketRequest_Ibm

// UpdateRequestS3 is a type to update the S3 bucket.
type UpdateRequestS3 = archiveMetrics.UpdateRequest_S3

// UpdateRequestIbm is a type to update the IBM storage.
type UpdateRequestIbm = archiveMetrics.UpdateRequest_Ibm

// TenantConfigV2S3 is a type to view the S3 bucket config.
type TenantConfigV2S3 = archiveMetrics.TenantConfigV2_S3

// TenantConfigV2Ibm is a type to view the IBM storage config.
type TenantConfigV2Ibm = archiveMetrics.TenantConfigV2_Ibm

// TenantConfigV2 is a type to configure the tenant.
type TenantConfigV2 = archiveMetrics.TenantConfigV2

// RetentionPolicyRequest is a request to set the retention policy.
type RetentionPolicyRequest = archiveMetrics.RetentionPolicyRequest

const archiveMetricsFeatureGroupID = "metrics"

// RPC names.
const (
	ArchiveMetricsConfigureTenantRPC = archiveMetrics.MetricsConfiguratorPublicService_ConfigureTenant_FullMethodName
	ArchiveMetricsUpdateRPC          = archiveMetrics.MetricsConfiguratorPublicService_Update_FullMethodName
	ArchiveMetricsValidateBucketRPC  = archiveMetrics.MetricsConfiguratorPublicService_ValidateBucket_FullMethodName
	ArchiveMetricsGetTenantConfigRPC = archiveMetrics.MetricsConfiguratorPublicService_GetTenantConfig_FullMethodName
	ArchiveMetricsEnableArchiveRPC   = archiveMetrics.MetricsConfiguratorPublicService_EnableArchive_FullMethodName
	ArchiveMetricsDisableArchiveRPC  = archiveMetrics.MetricsConfiguratorPublicService_DisableArchive_FullMethodName
)

// ArchiveMetricsClient is a client for the Coralogix Archive Metrics API.
type ArchiveMetricsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Update updates the archive metrics configuration.
func (c ArchiveMetricsClient) Update(ctx context.Context, req *archiveMetrics.UpdateRequest) (*emptypb.Empty, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveMetrics.NewMetricsConfiguratorPublicServiceClient(conn)

	response, err := client.Update(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveMetricsUpdateRPC, archiveMetricsFeatureGroupID)
	}
	return response, nil
}

// Get gets the archive metrics configuration.
func (c ArchiveMetricsClient) Get(ctx context.Context) (*archiveMetrics.GetTenantConfigResponseV2, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveMetrics.NewMetricsConfiguratorPublicServiceClient(conn)

	response, err := client.GetTenantConfig(callProperties.Ctx, &emptypb.Empty{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveMetricsGetTenantConfigRPC, archiveMetricsFeatureGroupID)
	}
	return response, nil
}

// ConfigureTenant configures the archive metrics bucket.
func (c ArchiveMetricsClient) ConfigureTenant(ctx context.Context, req *archiveMetrics.ConfigureTenantRequest) (*emptypb.Empty, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveMetrics.NewMetricsConfiguratorPublicServiceClient(conn)

	response, err := client.ConfigureTenant(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveMetricsConfigureTenantRPC, archiveMetricsFeatureGroupID)
	}
	return response, nil
}

// ValidateTarget validates the archive metrics bucket.
func (c ArchiveMetricsClient) ValidateTarget(ctx context.Context, req *archiveMetrics.ValidateBucketRequest) (*emptypb.Empty, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveMetrics.NewMetricsConfiguratorPublicServiceClient(conn)

	response, err := client.ValidateBucket(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveMetricsValidateBucketRPC, archiveMetricsFeatureGroupID)
	}
	return response, nil
}

// Enable enables the metrics archive.
func (c ArchiveMetricsClient) Enable(ctx context.Context) (*emptypb.Empty, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveMetrics.NewMetricsConfiguratorPublicServiceClient(conn)

	response, err := client.EnableArchive(callProperties.Ctx, &emptypb.Empty{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveMetricsEnableArchiveRPC, archiveMetricsFeatureGroupID)
	}
	return response, nil
}

// Disable disables the metrics archive.
func (c ArchiveMetricsClient) Disable(ctx context.Context) (*emptypb.Empty, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := archiveMetrics.NewMetricsConfiguratorPublicServiceClient(conn)

	response, err := client.DisableArchive(callProperties.Ctx, &emptypb.Empty{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ArchiveMetricsDisableArchiveRPC, archiveMetricsFeatureGroupID)
	}
	return response, nil
}

// NewArchiveMetricsClient creates a new archive metrics client.
func NewArchiveMetricsClient(c *CallPropertiesCreator) *ArchiveMetricsClient {
	return &ArchiveMetricsClient{callPropertiesCreator: c}
}
