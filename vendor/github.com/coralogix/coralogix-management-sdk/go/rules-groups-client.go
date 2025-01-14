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

	rulegroups "github.com/coralogix/coralogix-management-sdk/go/internal/coralogix/rules/v1"
)

// CreateRuleGroupRequest represents a request to create a rule group.
type CreateRuleGroupRequest = rulegroups.CreateRuleGroupRequest

// CreateRuleGroupResponse represents the response after creating a rule group.
type CreateRuleGroupResponse = rulegroups.CreateRuleGroupResponse

// GetRuleGroupRequest represents a request to get a rule group.
type GetRuleGroupRequest = rulegroups.GetRuleGroupRequest

// GetRuleGroupResponse represents the response after getting a rule group.
type GetRuleGroupResponse = rulegroups.GetRuleGroupResponse

// UpdateRuleGroupRequest represents a request to update a rule group.
type UpdateRuleGroupRequest = rulegroups.UpdateRuleGroupRequest

// UpdateRuleGroupResponse represents the response after updating a rule group.
type UpdateRuleGroupResponse = rulegroups.UpdateRuleGroupResponse

// DeleteRuleGroupRequest represents a request to delete a rule group.
type DeleteRuleGroupRequest = rulegroups.DeleteRuleGroupRequest

// DeleteRuleGroupResponse represents the response after deleting a rule group.
type DeleteRuleGroupResponse = rulegroups.DeleteRuleGroupResponse

// CreateRuleGroupRequestCreateRuleSubgroup represents a subgroup in the create rule group request.
type CreateRuleGroupRequestCreateRuleSubgroup = rulegroups.CreateRuleGroupRequest_CreateRuleSubgroup

// CreateRuleGroupRequestCreateRuleSubgroupCreateRule represents a rule in the create rule group subgroup request.
type CreateRuleGroupRequestCreateRuleSubgroupCreateRule = rulegroups.CreateRuleGroupRequest_CreateRuleSubgroup_CreateRule

// RuleParameters represents rule parameters.
type RuleParameters = rulegroups.RuleParameters

// RuleParametersParseParameters represents parse parameters for a rule.
type RuleParametersParseParameters = rulegroups.RuleParameters_ParseParameters

// ParseParameters represents parse parameters.
type ParseParameters = rulegroups.ParseParameters

// RuleParametersJSONParseParameters represents JSON parse parameters for a rule.
type RuleParametersJSONParseParameters = rulegroups.RuleParameters_JsonParseParameters

// JSONParseParameters represents JSON parse parameters.
type JSONParseParameters = rulegroups.JsonParseParameters

// RuleParametersJSONStringifyParameters represents JSON stringify parameters for a rule.
type RuleParametersJSONStringifyParameters = rulegroups.RuleParameters_JsonStringifyParameters

// JSONStringifyParameters represents JSON stringify parameters.
type JSONStringifyParameters = rulegroups.JsonStringifyParameters

// RuleParametersJSONExtractParameters represents JSON extract parameters for a rule.
type RuleParametersJSONExtractParameters = rulegroups.RuleParameters_JsonExtractParameters

// JSONExtractParameters represents JSON extract parameters.
type JSONExtractParameters = rulegroups.JsonExtractParameters

// RuleParametersRemoveFieldsParameters represents parameters to remove fields in a rule.
type RuleParametersRemoveFieldsParameters = rulegroups.RuleParameters_RemoveFieldsParameters

// RemoveFieldsParameters represents parameters to remove fields.
type RemoveFieldsParameters = rulegroups.RemoveFieldsParameters

// RuleParametersExtractTimestampParameters represents parameters to extract timestamp in a rule.
type RuleParametersExtractTimestampParameters = rulegroups.RuleParameters_ExtractTimestampParameters

// ExtractTimestampParameters represents parameters to extract timestamp.
type ExtractTimestampParameters = rulegroups.ExtractTimestampParameters

// RuleParametersBlockParameters represents parameters to block in a rule.
type RuleParametersBlockParameters = rulegroups.RuleParameters_BlockParameters

// BlockParameters represents block parameters.
type BlockParameters = rulegroups.BlockParameters

// RuleParametersAllowParameters represents allow parameters in a rule.
type RuleParametersAllowParameters = rulegroups.RuleParameters_AllowParameters

// AllowParameters represents allow parameters.
type AllowParameters = rulegroups.AllowParameters

// RuleParametersReplaceParameters represents replace parameters in a rule.
type RuleParametersReplaceParameters = rulegroups.RuleParameters_ReplaceParameters

// ReplaceParameters represents replace parameters.
type ReplaceParameters = rulegroups.ReplaceParameters

// RuleParametersExtractParameters represents extract parameters in a rule.
type RuleParametersExtractParameters = rulegroups.RuleParameters_ExtractParameters

// ExtractParameters represents extract parameters.
type ExtractParameters = rulegroups.ExtractParameters

// RuleMatcher represents a rule matcher.
type RuleMatcher = rulegroups.RuleMatcher

// ExtractTimestampParametersFormatStandard represents the standard timestamp format.
type ExtractTimestampParametersFormatStandard = rulegroups.ExtractTimestampParameters_FormatStandard

// ApplicationNameConstraint represents an application name constraint.
type ApplicationNameConstraint = rulegroups.ApplicationNameConstraint

// RuleMatcherApplicationName represents an application name matcher.
type RuleMatcherApplicationName = rulegroups.RuleMatcher_ApplicationName

// SubsystemNameConstraint represents a subsystem name constraint.
type SubsystemNameConstraint = rulegroups.SubsystemNameConstraint

// RuleMatcherSubsystemName represents a subsystem name matcher.
type RuleMatcherSubsystemName = rulegroups.RuleMatcher_SubsystemName

// SeverityConstraint represents a severity constraint.
type SeverityConstraint = rulegroups.SeverityConstraint

// SeverityConstraintValue represents a severity constraint value.
type SeverityConstraintValue = rulegroups.SeverityConstraint_Value

// JSONExtractParametersDestinationField represents a JSON extract destination field.
type JSONExtractParametersDestinationField = rulegroups.JsonExtractParameters_DestinationField

// RuleMatcherSeverity represents a severity matcher.
type RuleMatcherSeverity = rulegroups.RuleMatcher_Severity

// RuleGroup represents a rule group.
type RuleGroup = rulegroups.RuleGroup

// RuleSubgroup represents a subgroup of a rule.
type RuleSubgroup = rulegroups.RuleSubgroup

// Rule represents a rule.
type Rule = rulegroups.Rule

const rulesFeatureGroupID = "rules"

// SeverityConstraint values.
const (
	SeverityConstraintValueDebugOrUnspecified = rulegroups.SeverityConstraint_VALUE_DEBUG_OR_UNSPECIFIED
	SeverityConstraintValueVerbose            = rulegroups.SeverityConstraint_VALUE_VERBOSE
	SeverityConstraintValueInfo               = rulegroups.SeverityConstraint_VALUE_INFO
	SeverityConstraintValueWarning            = rulegroups.SeverityConstraint_VALUE_WARNING
	SeverityConstraintValueError              = rulegroups.SeverityConstraint_VALUE_ERROR
	SeverityConstraintValueCritical           = rulegroups.SeverityConstraint_VALUE_CRITICAL
)

// JSONExtractParameters values.
const (
	JSONExtractParametersDestinationFieldCategoryOrUnspecified = rulegroups.JsonExtractParameters_DESTINATION_FIELD_CATEGORY_OR_UNSPECIFIED
	JSONExtractParametersDestinationFieldClassName             = rulegroups.JsonExtractParameters_DESTINATION_FIELD_CLASSNAME
	JSONExtractParametersDestinationFieldMethodName            = rulegroups.JsonExtractParameters_DESTINATION_FIELD_METHODNAME
	JSONExtractParametersDestinationFieldThreadID              = rulegroups.JsonExtractParameters_DESTINATION_FIELD_THREADID
	JSONExtractParametersDestinationFieldSeverity              = rulegroups.JsonExtractParameters_DESTINATION_FIELD_SEVERITY
	JSONExtractParametersDestinationFieldText                  = rulegroups.JsonExtractParameters_DESTINATION_FIELD_TEXT
)

// ExtractTimestampParameters values.
const (
	ExtractTimestampParametersFormatStandardStrftimeOrUnspecified = rulegroups.ExtractTimestampParameters_FORMAT_STANDARD_STRFTIME_OR_UNSPECIFIED
	ExtractTimestampParametersFormatStandardJavasdf               = rulegroups.ExtractTimestampParameters_FORMAT_STANDARD_JAVASDF
	ExtractTimestampParametersFormatStandardGolang                = rulegroups.ExtractTimestampParameters_FORMAT_STANDARD_GOLANG
	ExtractTimestampParametersFormatStandardSecondsTS             = rulegroups.ExtractTimestampParameters_FORMAT_STANDARD_SECONDSTS
	ExtractTimestampParametersFormatStandardMilliTS               = rulegroups.ExtractTimestampParameters_FORMAT_STANDARD_MILLITS
	ExtractTimestampParametersFormatStandardMicroTS               = rulegroups.ExtractTimestampParameters_FORMAT_STANDARD_MICROTS
	ExtractTimestampParametersFormatStandardNanoTS                = rulegroups.ExtractTimestampParameters_FORMAT_STANDARD_NANOTS
)

// RPC names.
const (
	RuleGroupsGetRuleGroupRPC             = rulegroups.RuleGroupsService_GetRuleGroup_FullMethodName
	RuleGroupsListRuleGroupsRPC           = rulegroups.RuleGroupsService_ListRuleGroups_FullMethodName
	RuleGroupsCreateRuleGroupRPC          = rulegroups.RuleGroupsService_CreateRuleGroup_FullMethodName
	RuleGroupsUpdateRuleGroupRPC          = rulegroups.RuleGroupsService_UpdateRuleGroup_FullMethodName
	RuleGroupsDeleteRuleGroupRPC          = rulegroups.RuleGroupsService_DeleteRuleGroup_FullMethodName
	RuleGroupsBulkDeleteRuleGroupRPC      = rulegroups.RuleGroupsService_BulkDeleteRuleGroup_FullMethodName
	RuleGroupsGetRuleGroupModelMappingRPC = rulegroups.RuleGroupsService_GetRuleGroupModelMapping_FullMethodName
	RuleGroupsGetCompanyUsageLimitsRPC    = rulegroups.RuleGroupsService_GetCompanyUsageLimits_FullMethodName
)

// RuleGroupsClient is a client for the Coralogix Rules Groups API.
type RuleGroupsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// Create creates a new rule group.
func (r RuleGroupsClient) Create(ctx context.Context, req *CreateRuleGroupRequest) (*rulegroups.CreateRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rulegroups.NewRuleGroupsServiceClient(conn)

	response, err := client.CreateRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, RuleGroupsCreateRuleGroupRPC, rulesFeatureGroupID)
	}
	return response, nil
}

// Get gets a rule group.
func (r RuleGroupsClient) Get(ctx context.Context, req *GetRuleGroupRequest) (*rulegroups.GetRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rulegroups.NewRuleGroupsServiceClient(conn)

	response, err := client.GetRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, RuleGroupsGetRuleGroupRPC, rulesFeatureGroupID)
	}
	return response, nil
}

// Update updates a rule group.
func (r RuleGroupsClient) Update(ctx context.Context, req *UpdateRuleGroupRequest) (*rulegroups.UpdateRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := rulegroups.NewRuleGroupsServiceClient(conn)

	response, err := client.UpdateRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, RuleGroupsUpdateRuleGroupRPC, rulesFeatureGroupID)
	}
	return response, nil
}

// Delete deletes a rule group.
func (r RuleGroupsClient) Delete(ctx context.Context, req *DeleteRuleGroupRequest) (*rulegroups.DeleteRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := rulegroups.NewRuleGroupsServiceClient(conn)

	response, err := client.DeleteRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, RuleGroupsDeleteRuleGroupRPC, rulesFeatureGroupID)
	}
	return response, nil
}

// NewRuleGroupsClient creates a new rule groups client.
func NewRuleGroupsClient(c *CallPropertiesCreator) *RuleGroupsClient {
	return &RuleGroupsClient{callPropertiesCreator: c}
}
