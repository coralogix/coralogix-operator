// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cxsdk

import (
	"context"

	recordingRuleGroups "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/metrics-rule-manager/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateRuleGroupSetRequest is a request to create a new recording rule group set.
type CreateRuleGroupSetRequest = recordingRuleGroups.CreateRuleGroupSet

// CreateRuleGroupSetResponse is a result of creating a new recording rule group set.
type CreateRuleGroupSetResponse = recordingRuleGroups.CreateRuleGroupSetResult

// UpdateRuleGroupSetRequest is a request to update a recording rule group set.
type UpdateRuleGroupSetRequest = recordingRuleGroups.UpdateRuleGroupSet

// DeleteRuleGroupSetRequest is a request to delete a recording rule group set.
type DeleteRuleGroupSetRequest = recordingRuleGroups.DeleteRuleGroupSet

// GetRuleGroupSetRequest is a request to retrieve a recording rule group set.
type GetRuleGroupSetRequest = recordingRuleGroups.FetchRuleGroupSet

// GetRuleGroupSetResponse is the response from retrieving a recording rule group set.
type GetRuleGroupSetResponse = recordingRuleGroups.OutRuleGroupSet

// ListRuleGroupSetResponse is the response from listing recording rule group sets.
type ListRuleGroupSetResponse = recordingRuleGroups.RuleGroupSetListing

// InRuleGroup is a recording rule group.
type InRuleGroup = recordingRuleGroups.InRuleGroup

// InRule is a recording rule.
type InRule = recordingRuleGroups.InRule

// OutRuleGroup is a recording rule group.
type OutRuleGroup = recordingRuleGroups.OutRuleGroup

// OutRule is a recording rule.
type OutRule = recordingRuleGroups.OutRule

const recordingRulesFeatureGroupID = "recording-rules"

// RPC Name values
const (
	CreateRuleGroupSetRPC = recordingRuleGroups.RuleGroupSets_Create_FullMethodName
	UpdateRuleGroupSetRPC = recordingRuleGroups.RuleGroupSets_Update_FullMethodName
	DeleteRuleGroupSetRPC = recordingRuleGroups.RuleGroupSets_Delete_FullMethodName
	GetRuleGroupSetRPC    = recordingRuleGroups.RuleGroupSets_Fetch_FullMethodName
	ListRuleGroupSetRPC   = recordingRuleGroups.RuleGroupSets_List_FullMethodName
)

// RecordingRuleGroupSetsClient is a client for managing rule groups.
type RecordingRuleGroupSetsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new recording rule group set.
func (r RecordingRuleGroupSetsClient) Create(ctx context.Context, req *CreateRuleGroupSetRequest) (*CreateRuleGroupSetResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := recordingRuleGroups.NewRuleGroupSetsClient(conn)

	response, err := client.Create(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, CreateRuleGroupSetRPC, recordingRulesFeatureGroupID)
	}
	return response, nil
}

// Update updates an existing recording rule group set.
func (r RecordingRuleGroupSetsClient) Update(ctx context.Context, req *UpdateRuleGroupSetRequest) (*emptypb.Empty, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := recordingRuleGroups.NewRuleGroupSetsClient(conn)

	response, err := client.Update(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, UpdateRuleGroupSetRPC, recordingRulesFeatureGroupID)
	}
	return response, nil
}

// Delete deletes a recording rule group set.
func (r RecordingRuleGroupSetsClient) Delete(ctx context.Context, req *DeleteRuleGroupSetRequest) (*emptypb.Empty, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := recordingRuleGroups.NewRuleGroupSetsClient(conn)

	response, err := client.Delete(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DeleteRuleGroupSetRPC, recordingRulesFeatureGroupID)
	}
	return response, nil
}

// Get retrieves a recording rule group set by ID.
func (r RecordingRuleGroupSetsClient) Get(ctx context.Context, req *GetRuleGroupSetRequest) (*GetRuleGroupSetResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := recordingRuleGroups.NewRuleGroupSetsClient(conn)

	response, err := client.Fetch(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetRuleGroupSetRPC, recordingRulesFeatureGroupID)
	}
	return response, nil
}

// List retrieves all recording rule group sets.
func (r RecordingRuleGroupSetsClient) List(ctx context.Context) (*ListRuleGroupSetResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := recordingRuleGroups.NewRuleGroupSetsClient(conn)

	response, err := client.List(callProperties.Ctx, &emptypb.Empty{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, ListRuleGroupSetRPC, recordingRulesFeatureGroupID)
	}
	return response, nil
}

// NewRecordingRuleGroupSetsClient creates a new rule groups client.
func NewRecordingRuleGroupSetsClient(c *CallPropertiesCreator) *RecordingRuleGroupSetsClient {
	return &RecordingRuleGroupSetsClient{callPropertiesCreator: c}
}
