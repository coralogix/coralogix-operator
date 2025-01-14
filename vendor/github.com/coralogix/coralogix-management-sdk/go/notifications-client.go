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

	"github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/notification_center/common"
	commonv1 "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/notification_center/common/v1"
	connectores "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/notification_center/connectors/v1"
	notifications "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/notification_center/notifications/v1"
	presets "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/notification_center/presets/v1"
)

// CreateConnectorRequest represents a request to create a connector.
type CreateConnectorRequest = connectores.CreateConnectorRequest

// CreateConnectorResponse represents a response to create a connector.
type CreateConnectorResponse = connectores.CreateConnectorResponse

// ReplaceConnectorRequest represents a request to replace a connector.
type ReplaceConnectorRequest = connectores.ReplaceConnectorRequest

// ReplaceConnectorResponse represents a response to replace a connector.
type ReplaceConnectorResponse = connectores.ReplaceConnectorResponse

// DeleteConnectorRequest represents a request to delete a connector.
type DeleteConnectorRequest = connectores.DeleteConnectorRequest

// DeleteConnectorResponse represents a response to delete a connector.
type DeleteConnectorResponse = connectores.DeleteConnectorResponse

// GetConnectorRequest represents a request to get a connector.
type GetConnectorRequest = connectores.GetConnectorRequest

// GetConnectorResponse represents a response to get a connector.
type GetConnectorResponse = connectores.GetConnectorResponse

// ListConnectorsRequest represents a request to list connectors.
type ListConnectorsRequest = connectores.ListConnectorsRequest

// ListConnectorsResponse represents a response to list connectors.
type ListConnectorsResponse = connectores.ListConnectorsResponse

// BatchGetConnectorsRequest represents a request to batch get connectors.
type BatchGetConnectorsRequest = connectores.BatchGetConnectorsRequest

// BatchGetConnectorsResponse represents a response to batch get connectors.
type BatchGetConnectorsResponse = connectores.BatchGetConnectorsResponse

// GetConnectorTypeSummariesRequest represents a request to get connector type summaries.
type GetConnectorTypeSummariesRequest = connectores.GetConnectorTypeSummariesRequest

// GetConnectorTypeSummariesResponse represents a response to get connector type summaries.
type GetConnectorTypeSummariesResponse = connectores.GetConnectorTypeSummariesResponse

// Connector represents a connector.
type Connector = connectores.Connector

// ConnectorConfig represents a connector configuration.
type ConnectorConfig = connectores.ConnectorConfig

// ConnectorConfigField represents a connector configuration field.
type ConnectorConfigField = commonv1.ConnectorConfigField

// ConnectorType is a connector type.
type ConnectorType = common.ConnectorType

// ConnectorType values.
const (
	ConnectorTypeUnSpecified  = common.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED
	ConnectorTypeSlack        = common.ConnectorType_SLACK
	ConnectorTypeGenericHTTPS = common.ConnectorType_GENERIC_HTTPS
	ConnectorTypePagerDuty    = common.ConnectorType_PAGERDUTY
)

// CreateCustomPresetRequest is a request to create a custom preset.
type CreateCustomPresetRequest = presets.CreateCustomPresetRequest

// CreateCustomPresetResponse is a response to create a custom preset.
type CreateCustomPresetResponse = presets.CreateCustomPresetResponse

// ReplaceCustomPresetRequest is a request to replace a custom preset.
type ReplaceCustomPresetRequest = presets.ReplaceCustomPresetRequest

// ReplaceCustomPresetResponse is a response to replace a custom preset.
type ReplaceCustomPresetResponse = presets.ReplaceCustomPresetResponse

// DeleteCustomPresetRequest is a request to delete a custom preset.
type DeleteCustomPresetRequest = presets.DeleteCustomPresetRequest

// DeleteCustomPresetResponse is a response to delete a custom preset.
type DeleteCustomPresetResponse = presets.DeleteCustomPresetResponse

// SetCustomPresetAsDefaultRequest is a request to set a custom preset as default.
type SetCustomPresetAsDefaultRequest = presets.SetCustomPresetAsDefaultRequest

// SetCustomPresetAsDefaultResponse is a response to set a custom preset as default.
type SetCustomPresetAsDefaultResponse = presets.SetCustomPresetAsDefaultResponse

// GetPresetRequest is a request to get a preset.
type GetPresetRequest = presets.GetPresetRequest

// GetPresetResponse is a response to get a preset.
type GetPresetResponse = presets.GetPresetResponse

// ListPresetSummariesRequest is a request to list preset summaries.
type ListPresetSummariesRequest = presets.ListPresetSummariesRequest

// ListPresetSummariesResponse is a response to list preset summaries.
type ListPresetSummariesResponse = presets.ListPresetSummariesResponse

// BatchGetPresetsRequest is a request to batch get presets.
type BatchGetPresetsRequest = presets.BatchGetPresetsRequest

// BatchGetPresetsResponse is a response to batch get presets.
type BatchGetPresetsResponse = presets.BatchGetPresetsResponse

// GetDefaultPresetSummaryRequest is a request to get the default preset summary.
type GetDefaultPresetSummaryRequest = presets.GetDefaultPresetSummaryRequest

// GetDefaultPresetSummaryResponse is a response to get the default preset summary.
type GetDefaultPresetSummaryResponse = presets.GetDefaultPresetSummaryResponse

// GetSystemDefaultPresetSummaryRequest is a request to get the system default preset summary.
type GetSystemDefaultPresetSummaryRequest = presets.GetSystemDefaultPresetSummaryRequest

// GetSystemDefaultPresetSummaryResponse is a response to get the system default preset summary.
type GetSystemDefaultPresetSummaryResponse = presets.GetSystemDefaultPresetSummaryResponse

// Preset represents a preset.
type Preset = presets.Preset

// ConfigOverrides is a configuration override.
type ConfigOverrides = common.ConfigOverrides

// ConditionType is a condition type.
type ConditionType = common.ConditionType

// ConditionTypeMatchEntityType is a match entity type condition type.
type ConditionTypeMatchEntityType = common.ConditionType_MatchEntityType

// MatchEntityTypeCondition is a match entity type condition.
type MatchEntityTypeCondition = common.MatchEntityTypeCondition

// PresetType is a preset type.
type PresetType = presets.PresetType

// PresetType values.
const (
	PresetTypeUnspecified = presets.PresetType_PRESET_TYPE_UNSPECIFIED
	PresetTypeSystem      = presets.PresetType_SYSTEM
	PresetTypeCustom      = presets.PresetType_CUSTOM
)

// MessageConfig is a message configuration.
type MessageConfig = common.MessageConfig

// MessageConfigField is a message configuration field.
type MessageConfigField = commonv1.MessageConfigField

// TestConnectorConfigRequest is a request to test a connector configuration.
type TestConnectorConfigRequest = notifications.TestConnectorConfigRequest

// TestConnectorConfigResponse is a response to test a connector configuration.
type TestConnectorConfigResponse = notifications.TestConnectorConfigResponse

// TestExistingConnectorRequest is a request to test an existing connector.
type TestExistingConnectorRequest = notifications.TestExistingConnectorRequest

// TestExistingConnectorResponse is a response to test an existing connector.
type TestExistingConnectorResponse = notifications.TestExistingConnectorResponse

// TestPresetConfigRequest is a request to test a preset configuration.
type TestPresetConfigRequest = notifications.TestPresetConfigRequest

// TestPresetConfigResponse is a response to test a preset configuration.
type TestPresetConfigResponse = notifications.TestPresetConfigResponse

// TestTemplateRenderRequest is a request to test a template rendering.
type TestTemplateRenderRequest = notifications.TestTemplateRenderRequest

// TestTemplateRenderResponse is a response to test a template rendering.
type TestTemplateRenderResponse = notifications.TestTemplateRenderResponse

const notificationsFeatureGroupID = "notifications"

// RPC names.
const (
	ConnectorsCreateRPC                    = connectores.ConnectorsService_CreateConnector_FullMethodName
	ConnectorsReplaceRPC                   = connectores.ConnectorsService_ReplaceConnector_FullMethodName
	ConnectorsDeleteRPC                    = connectores.ConnectorsService_DeleteConnector_FullMethodName
	ConnectorsGetRPC                       = connectores.ConnectorsService_GetConnector_FullMethodName
	ConnectorsListRPC                      = connectores.ConnectorsService_ListConnectors_FullMethodName
	ConnectorsBatchGetRPC                  = connectores.ConnectorsService_BatchGetConnectors_FullMethodName
	ConnectorsGetConnectorTypeSummariesRPC = connectores.ConnectorsService_GetConnectorTypeSummaries_FullMethodName
	PresetsCreateRPC                       = presets.PresetsService_CreateCustomPreset_FullMethodName
	PresetsReplaceRPC                      = presets.PresetsService_ReplaceCustomPreset_FullMethodName
	PresetsDeleteRPC                       = presets.PresetsService_DeleteCustomPreset_FullMethodName
	PresetsSetAsDefaultRPC                 = presets.PresetsService_SetCustomPresetAsDefault_FullMethodName
	PresetsGetRPC                          = presets.PresetsService_GetPreset_FullMethodName
	PresetsListRPC                         = presets.PresetsService_ListPresetSummaries_FullMethodName
	PresetsBatchGetRPC                     = presets.PresetsService_BatchGetPresets_FullMethodName
	PresetsGetDefaultRPC                   = presets.PresetsService_GetDefaultPresetSummary_FullMethodName
	PresetsGetSystemDefaultRPC             = presets.PresetsService_GetSystemDefaultPresetSummary_FullMethodName
	TestingTestConnectorConfigRPC          = notifications.TestingService_TestConnectorConfig_FullMethodName
	TestingTestExistingConnectorRPC        = notifications.TestingService_TestExistingConnector_FullMethodName
	TestingTestPresetConfigRPC             = notifications.TestingService_TestPresetConfig_FullMethodName
	TestingTestTemplateRenderRPC           = notifications.TestingService_TestTemplateRender_FullMethodName
)

// NotificationsClient is a client for the Coralogix Notifications API.
type NotificationsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// CreateConnector creates a new connector.
func (c NotificationsClient) CreateConnector(ctx context.Context, req *CreateConnectorRequest) (*CreateConnectorResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := connectores.NewConnectorsServiceClient(conn)

	response, err := client.CreateConnector(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ConnectorsCreateRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// ReplaceConnector replaces a connector.
func (c NotificationsClient) ReplaceConnector(ctx context.Context, req *ReplaceConnectorRequest) (*ReplaceConnectorResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := connectores.NewConnectorsServiceClient(conn)

	response, err := client.ReplaceConnector(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ConnectorsReplaceRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// DeleteConnector deletes a connector.
func (c NotificationsClient) DeleteConnector(ctx context.Context, req *DeleteConnectorRequest) (*DeleteConnectorResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := connectores.NewConnectorsServiceClient(conn)

	response, err := client.DeleteConnector(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ConnectorsDeleteRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// GetConnector retrieves a connector by ID.
func (c NotificationsClient) GetConnector(ctx context.Context, req *GetConnectorRequest) (*GetConnectorResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := connectores.NewConnectorsServiceClient(conn)

	response, err := client.GetConnector(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ConnectorsGetRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// ListConnectors lists connectors.
func (c NotificationsClient) ListConnectors(ctx context.Context, req *ListConnectorsRequest) (*ListConnectorsResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := connectores.NewConnectorsServiceClient(conn)

	response, err := client.ListConnectors(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ConnectorsListRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// BatchGetConnectors retrieves connectors by IDs.
func (c NotificationsClient) BatchGetConnectors(ctx context.Context, req *BatchGetConnectorsRequest) (*BatchGetConnectorsResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := connectores.NewConnectorsServiceClient(conn)

	response, err := client.BatchGetConnectors(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ConnectorsBatchGetRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// GetConnectorTypeSummaries retrieves connector type summaries.
func (c NotificationsClient) GetConnectorTypeSummaries(ctx context.Context, req *GetConnectorTypeSummariesRequest) (*GetConnectorTypeSummariesResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := connectores.NewConnectorsServiceClient(conn)

	response, err := client.GetConnectorTypeSummaries(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ConnectorsGetConnectorTypeSummariesRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// CreateCustomPreset creates a new custom preset.
func (c NotificationsClient) CreateCustomPreset(ctx context.Context, req *CreateCustomPresetRequest) (*CreateCustomPresetResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.CreateCustomPreset(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsCreateRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// ReplaceCustomPreset replaces a custom preset.
func (c NotificationsClient) ReplaceCustomPreset(ctx context.Context, req *ReplaceCustomPresetRequest) (*ReplaceCustomPresetResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.ReplaceCustomPreset(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsReplaceRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// DeleteCustomPreset deletes a custom preset.
func (c NotificationsClient) DeleteCustomPreset(ctx context.Context, req *DeleteCustomPresetRequest) (*DeleteCustomPresetResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.DeleteCustomPreset(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsDeleteRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// SetCustomPresetAsDefault sets a custom preset as default.
func (c NotificationsClient) SetCustomPresetAsDefault(ctx context.Context, req *SetCustomPresetAsDefaultRequest) (*SetCustomPresetAsDefaultResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.SetCustomPresetAsDefault(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsSetAsDefaultRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// GetPreset retrieves a preset.
func (c NotificationsClient) GetPreset(ctx context.Context, req *GetPresetRequest) (*GetPresetResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.GetPreset(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsGetRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// ListPresetSummaries lists preset summaries.
func (c NotificationsClient) ListPresetSummaries(ctx context.Context, req *ListPresetSummariesRequest) (*ListPresetSummariesResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.ListPresetSummaries(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsListRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// BatchGetPresets retrieves a group of presets.
func (c NotificationsClient) BatchGetPresets(ctx context.Context, req *BatchGetPresetsRequest) (*BatchGetPresetsResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.BatchGetPresets(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsBatchGetRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// GetDefaultPresetSummary retrieves the default preset summary.
func (c NotificationsClient) GetDefaultPresetSummary(ctx context.Context, req *GetDefaultPresetSummaryRequest) (*GetDefaultPresetSummaryResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.GetDefaultPresetSummary(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsGetDefaultRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// GetSystemDefaultPresetSummary retrieves the system default preset summary.
func (c NotificationsClient) GetSystemDefaultPresetSummary(ctx context.Context, req *GetSystemDefaultPresetSummaryRequest) (*GetSystemDefaultPresetSummaryResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}
	conn := callProperties.Connection
	defer conn.Close()
	client := presets.NewPresetsServiceClient(conn)

	response, err := client.GetSystemDefaultPresetSummary(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PresetsGetSystemDefaultRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// TestConnectorConfig tests a connector configuration.
func (c NotificationsClient) TestConnectorConfig(ctx context.Context, req *TestConnectorConfigRequest) (*TestConnectorConfigResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := notifications.NewTestingServiceClient(conn)

	response, err := client.TestConnectorConfig(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, TestingTestConnectorConfigRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// TestExistingConnector tests an existing connector.
func (c NotificationsClient) TestExistingConnector(ctx context.Context, req *TestExistingConnectorRequest) (*TestExistingConnectorResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := notifications.NewTestingServiceClient(conn)

	response, err := client.TestExistingConnector(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, TestingTestExistingConnectorRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// TestPresetConfig tests a preset configuration.
func (c NotificationsClient) TestPresetConfig(ctx context.Context, req *TestPresetConfigRequest) (*TestPresetConfigResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := notifications.NewTestingServiceClient(conn)

	response, err := client.TestPresetConfig(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, TestingTestPresetConfigRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// TestTemplateRender tests a template rendering.
func (c NotificationsClient) TestTemplateRender(ctx context.Context, req *TestTemplateRenderRequest) (*TestTemplateRenderResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := notifications.NewTestingServiceClient(conn)

	response, err := client.TestTemplateRender(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, TestingTestTemplateRenderRPC, notificationsFeatureGroupID)
	}
	return response, nil
}

// NewNotificationsClient creates a new notifications' client.
func NewNotificationsClient(c *CallPropertiesCreator) *NotificationsClient {
	return &NotificationsClient{callPropertiesCreator: c}
}
