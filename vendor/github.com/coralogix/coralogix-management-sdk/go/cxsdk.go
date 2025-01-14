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
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPC URLs for the Coralogix regions.
const (
	GrpcUS1 = "ng-api-grpc.coralogix.com:443"
	GrpcUS2 = "ng-api-grpc.cx498.coralogix.com:443"
	GrpcEU1 = "ng-api-grpc.coralogix.com:443"
	GrpcEU2 = "ng-api-grpc.eu2.coralogix.com:443"
	GrpcAP1 = "ng-api-grpc.app.coralogix.in:443"
	GrpcAP2 = "ng-api-grpc.coralogixsg.com:443"
	GrpcAP3 = "ng-api-grpc.ap3.coralogix.com:443"
)

// RESt URLs for the Coralogix regions.
const (
	RestUS1 = "https://ng-api-http.coralogix.com"
	RestUS2 = "https://ng-api-http.cx498.coralogix.com"
	RestEU1 = "https://ng-api-http.coralogix.com"
	RestEU2 = "https://ng-api-http.eu2.coralogix.com"
	RestAP1 = "https://ng-api-http.app.coralogix.in"
	RestAP2 = "https://ng-api-http.coralogixsg.com"
	RestAP3 = "https://ng-api-http.ap3.coralogix.com"
)

// SdkAPIError is an error that occurs in the Coralogix SDK.
type SdkAPIError struct {
	apiError       error
	endpoint       string
	featureGroupID string
}

// NewSdkAPIError creates a new SdkAPIError.
func NewSdkAPIError(
	apiError error,
	endpoint string,
	featureGroupID string,
) *SdkAPIError {
	return &SdkAPIError{
		apiError:       apiError,
		endpoint:       endpoint,
		featureGroupID: featureGroupID,
	}
}

func (e *SdkAPIError) Error() string {
	return fmt.Sprintf("SDK API error from %s for feature group %s: %s", e.endpoint, e.featureGroupID, e.apiError)
}

// Code returns the error code of the SdkAPIError.apiError.
func Code(err error) codes.Code {
	var sdkErr *SdkAPIError
	if errors.As(err, &sdkErr) {
		return status.Code(sdkErr.apiError)
	}
	return codes.Unknown
}

// ClientSet is a set of clients for the Coralogix SDK.
type ClientSet struct {
	ruleGroups          *RuleGroupsClient
	recordingRuleGroups *RecordingRuleGroupSetsClient
	alerts              *AlertsClient
	enrichments         *EnrichmentsClient
	dataSet             *DataSetClient
	dashboards          *DashboardsClient
	actions             *ActionsClient
	tcoPolicies         *TCOPoliciesClient
	webhooks            *WebhooksClient
	events2Metrics      *Events2MetricsClient
	archiveRetentions   *ArchiveRetentionsClient
	archiveMetrics      *ArchiveMetricsClient
	archiveLogs         *ArchiveLogsClient
	alertScheduler      *AlertSchedulerClient
	teams               *TeamsClient
	slos                *SLOsClient
	scopes              *ScopesClient
	apiKeys             *ApikeysClient
	users               *UsersClient
	groups              *GroupsClient
	saml                *SamlClient
	dataUsage           *DataUsageClient
}

// RuleGroups gets a RuleGroupsClient from the ClientSet.
func (c *ClientSet) RuleGroups() *RuleGroupsClient {
	return c.ruleGroups
}

// RecordingRuleGroups gets a RecordingRuleGroupSetsClient from the ClientSet.
func (c *ClientSet) RecordingRuleGroups() *RecordingRuleGroupSetsClient {
	return c.recordingRuleGroups
}

// Alerts gets an AlertsClient from the ClientSet.
func (c *ClientSet) Alerts() *AlertsClient {
	return c.alerts
}

// Enrichments gets an EnrichmentsClient from the ClientSet.
func (c *ClientSet) Enrichments() *EnrichmentsClient {
	return c.enrichments
}

// DataSet gets a DataSetClient from the ClientSet.
func (c *ClientSet) DataSet() *DataSetClient {
	return c.dataSet
}

// func (c *ClientSet) Dashboards() *DashboardsClient {
// 	return c.dashboards
// }

// Actions gets an ActionsClient from the ClientSet.
func (c *ClientSet) Actions() *ActionsClient {
	return c.actions
}

// TCOPolicies gets a TCOPoliciesClient from the ClientSet.
func (c *ClientSet) TCOPolicies() *TCOPoliciesClient {
	return c.tcoPolicies
}

// Webhooks gets a WebhooksClient from the ClientSet.
func (c *ClientSet) Webhooks() *WebhooksClient {
	return c.webhooks
}

// Events2Metrics gets an Events2MetricsClient from the ClientSet.
func (c *ClientSet) Events2Metrics() *Events2MetricsClient {
	return c.events2Metrics
}

// ArchiveRetentions gets an ArchiveRetentionsClient from the ClientSet.
func (c *ClientSet) ArchiveRetentions() *ArchiveRetentionsClient {
	return c.archiveRetentions
}

// ArchiveMetrics gets an ArchiveMetricsClient from the ClientSet.
func (c *ClientSet) ArchiveMetrics() *ArchiveMetricsClient {
	return c.archiveMetrics
}

// ArchiveLogs gets an ArchiveLogsClient from the ClientSet.
func (c *ClientSet) ArchiveLogs() *ArchiveLogsClient {
	return c.archiveLogs
}

// AlertSchedulers Gets an AlertSchedulerClient from the ClientSet.
func (c *ClientSet) AlertSchedulers() *AlertSchedulerClient {
	return c.alertScheduler
}

// Teams gets a TeamsClient from the ClientSet.
func (c *ClientSet) Teams() *TeamsClient {
	return c.teams
}

// APIKeys gets an SLOsClient from the ClientSet.
func (c *ClientSet) APIKeys() *ApikeysClient {
	return c.apiKeys
}

// SLOs gets an SLOsClient from the ClientSet.
func (c *ClientSet) SLOs() *SLOsClient {
	return c.slos
}

// Scopes gets a ScopesClient from the ClientSet.
func (c *ClientSet) Scopes() *ScopesClient {
	return c.scopes
}

// Users gets a UsersClient from the ClientSet.
func (c *ClientSet) Users() *UsersClient {
	return c.users
}

// Groups gets a GroupsClient from the ClientSet.
func (c *ClientSet) Groups() *GroupsClient {
	return c.groups
}

// Saml gets a SamlClient from the ClientSet.
func (c *ClientSet) Saml() *SamlClient {
	return c.saml
}

// DataUsage gets a DataUsageClient from the ClientSet.
func (c *ClientSet) DataUsage() *DataUsageClient {
	return c.dataUsage
}

// NewClientSet Creates a new ClientSet.
func NewClientSet(targetURL, teamsLevelAPIKey string, userLevelAPIKey string) *ClientSet {
	authContext := NewAuthContext(teamsLevelAPIKey, userLevelAPIKey)
	apikeyCPC := NewCallPropertiesCreator(targetURL, authContext)

	return &ClientSet{
		ruleGroups:          NewRuleGroupsClient(apikeyCPC),
		recordingRuleGroups: NewRecordingRuleGroupSetsClient(apikeyCPC),
		alerts:              NewAlertsClient(apikeyCPC),
		events2Metrics:      NewEvents2MetricsClient(apikeyCPC),
		enrichments:         NewEnrichmentClient(apikeyCPC),
		dataSet:             NewDataSetClient(apikeyCPC),
		dashboards:          NewDashboardsClient(apikeyCPC),
		actions:             NewActionsClient(apikeyCPC),
		tcoPolicies:         NewTCOPoliciesClient(apikeyCPC),
		webhooks:            NewWebhooksClient(apikeyCPC),
		archiveRetentions:   NewArchiveRetentionsClient(apikeyCPC),
		archiveMetrics:      NewArchiveMetricsClient(apikeyCPC),
		archiveLogs:         NewArchiveLogsClient(apikeyCPC),
		alertScheduler:      NewAlertSchedulerClient(apikeyCPC),
		teams:               NewTeamsClient(apikeyCPC),
		slos:                NewSLOsClient(apikeyCPC),
		scopes:              NewScopesClient(apikeyCPC),
		// dahboardsFolders:  NewDashboardsFoldersClient(apikeyCPC),
		apiKeys: NewAPIKeysClient(apikeyCPC),
		groups:  NewGroupsClient(apikeyCPC),
		saml:    NewSamlClient(apikeyCPC),
		// users:             NewUsersClient(apikeyCPC),
	}
}

// EnvCoralogxRegion is the name of the environment variable that contains the Coralogix region.
const EnvCoralogxRegion = "CORALOGIX_REGION"

// EnvCoralogixTeamLevelAPIKey is the name of the environment variable that contains the Coralogix Team API key.
const EnvCoralogixTeamLevelAPIKey = "CORALOGIX_TEAM_API_KEY"

// EnvCoralogixUserLevelAPIKey is the name of the environment variable that contains the Coralogix User API key.
const EnvCoralogixUserLevelAPIKey = "CORALOGIX_USER_API_KEY"

// CoralogixRegionFromEnv reads the Coralogix region from environment variables.
func CoralogixRegionFromEnv() (string, error) {
	regionIdentifier := strings.ToLower(os.Getenv(EnvCoralogxRegion))

	if regionIdentifier == "" {
		return "", fmt.Errorf("the %s environment variable is not set", EnvCoralogxRegion)
	}
	return regionIdentifier, nil
}

// CoralogixGrpcEndpointFromRegion reads the Coralogix region from environment variables.
func CoralogixGrpcEndpointFromRegion(regionIdentifier string) string {
	switch regionIdentifier {
	case "us1":
		return GrpcUS1
	case "us2":
		return GrpcUS2
	case "eu1":
		return GrpcEU1
	case "eu2":
		return GrpcEU2
	case "ap1":
		return GrpcAP1
	case "ap2":
		return GrpcAP2
	case "ap3":
		return GrpcAP3
	default:
		return regionIdentifier
	}
}

// CoralogixRestEndpointFromRegion reads the Coralogix REST endpoint from environment variables.
func CoralogixRestEndpointFromRegion(regionIdentifier string) string {

	switch regionIdentifier {
	case "us1":
		return RestUS1
	case "us2":
		return RestUS2
	case "eu1":
		return RestEU1
	case "eu2":
		return RestEU2
	case "ap1":
		return RestAP1
	case "ap2":
		return RestAP2
	case "ap3":
		return RestAP3
	default:
		return regionIdentifier
	}
}

// AuthContext is a struct that holds the API keys for the Coralogix SDK.
type AuthContext struct {
	teamLevelAPIKey string
	userLevelAPIKey string
}

// NewAuthContext creates a new AuthContext.
func NewAuthContext(teamLevelAPIKey, userLevelAPIKey string) AuthContext {
	if teamLevelAPIKey == "" {
		log.Println("Warning: teamLevelAPIKey was not provided. Some functionality will not be available.")
	}
	if userLevelAPIKey == "" {
		log.Println("Warning: userLevelAPIKey was not provided. Some functionality will not be available.")
	}
	return AuthContext{
		teamLevelAPIKey: teamLevelAPIKey,
		userLevelAPIKey: userLevelAPIKey,
	}
}

// AuthContextFromEnv reads the Coralogix API keys from environment variables.
func AuthContextFromEnv() (AuthContext, error) {
	teamLevelAPIKey, err := CoralogixTeamsLevelAPIKeyFromEnv()
	if err != nil {
		return AuthContext{}, err
	}
	userLevelAPIKey, err := CoralogixUserLevelAPIKeyFromEnv()
	if err != nil {
		return AuthContext{}, err
	}
	if teamLevelAPIKey == "" {
		log.Println("Warning: teamLevelAPIKey is empty. Some functionality will not be available.")
	}
	if userLevelAPIKey == "" {
		log.Println("Warning: userLevelAPIKey is empty. Some functionality will not be available.")
	}

	return AuthContext{teamLevelAPIKey: teamLevelAPIKey, userLevelAPIKey: userLevelAPIKey}, nil
}

// CoralogixTeamsLevelAPIKeyFromEnv reads the Coralogix Team API key from environment variables.
func CoralogixTeamsLevelAPIKeyFromEnv() (string, error) {
	apiKey := os.Getenv(EnvCoralogixTeamLevelAPIKey)
	if apiKey == "" {
		return "", fmt.Errorf("the %s environment variable is not set", EnvCoralogixTeamLevelAPIKey)
	}
	return apiKey, nil
}

// CoralogixUserLevelAPIKeyFromEnv reads the Coralogix User API key from environment variables.
func CoralogixUserLevelAPIKeyFromEnv() (string, error) {
	apiKey := os.Getenv(EnvCoralogixUserLevelAPIKey)
	if apiKey == "" {
		return "", fmt.Errorf("the %s environment variable is not set", EnvCoralogixUserLevelAPIKey)
	}
	return apiKey, nil
}
