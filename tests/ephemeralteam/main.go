// Copyright 2026 Coralogix Ltd.
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

// Command ephemeralteam provisions a disposable Coralogix team for an e2e CI
// job so concurrent workflow runs stop clobbering each other's team-wide
// singleton state (TCO policies, archive targets, IP access, quota allocation
// rules, global router, enrichments).
//
// The operator is deployed with a single API key, so the isolation unit is the
// whole CI job, not a single spec: the workflow creates a team plus a
// team-scoped API key before deploying the operator, runs the suite against
// that team, and deletes the team only when everything passed. A failed run
// keeps the team so its state can be inspected; leaked teams are recognizable
// by the TeamNamePrefix and can be swept by a scheduled cleanup.
//
// Usage:
//
//	go run ./tests/ephemeralteam create
//	go run ./tests/ephemeralteam delete <team-id>
//
// `create` reads CORALOGIX_ORG_API_KEY (an org-scoped key allowed to manage
// teams) and CORALOGIX_REGION or CORALOGIX_DOMAIN. It prints KEY=VALUE lines
// on stdout, ready to be appended to $GITHUB_ENV:
//
//	CORALOGIX_API_KEY=<team-scoped api key>
//	EPHEMERAL_TEAM_ID=<team id>
//	EPHEMERAL_TEAM_NAME=<team name>
//
// Everything else (progress, errors) goes to stderr. When
// CORALOGIX_ORG_API_KEY is unset, `create` prints nothing and exits 0, so the
// harness is a no-op until the secret exists. `delete` reads the same env
// vars and removes the team given on the command line.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	apikeys "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/api_keys_service"
	teams "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/teams_service"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
)

// orgAPIKeyEnvVar holds the org-scoped API key that is allowed to create and
// delete teams. The create subcommand is a no-op when it is unset.
const orgAPIKeyEnvVar = "CORALOGIX_ORG_API_KEY"

// teamNamePrefix marks teams created by this command so leaked teams (kept
// after a failed run, or orphaned by a killed one) can be found and swept.
const teamNamePrefix = "cx-operator-e2e-ephemeral"

// defaultTeamKeyPermissions is granted to the minted team key. It is the
// documented "Legacy Api Key" permission set, extended with the surfaces the
// live e2e suites turned out to need on top of it (AI evaluations,
// notification center, SLO, alert scheduler suppression rules, roles, groups,
// members, scopes, IP access, saved views, integrations, and the api-key and
// dashboard access policies). All names are verified against the SDK's
// permission catalog (permission_definitions).
var defaultTeamKeyPermissions = []string{
	"ai-app-catalog:Read",
	"ai-app-discovery:Manage",
	"ai-app-discovery:Read",
	"ai-app-evaluators:Manage",
	"ai-app-evaluators:Read",
	"ai-overview:Read",
	"alerts:ReadConfig",
	"alerts:UpdateConfig",
	"cloud-metadata-enrichment:ReadConfig",
	"cloud-metadata-enrichment:UpdateConfig",
	"data-usage:Read",
	"geo-enrichment:ReadConfig",
	"geo-enrichment:UpdateConfig",
	"grafana:Read",
	"grafana:Update",
	"integrations:Deploy",
	"integrations:ReadConfig",
	"livetail:Read",
	"logs.alerts:ReadConfig",
	"logs.alerts:UpdateConfig",
	"logs.data-setup#low:ReadConfig",
	"logs.data-setup#low:UpdateConfig",
	"logs.events2metrics:ReadConfig",
	"logs.events2metrics:UpdateConfig",
	"logs.tco:ReadPolicies",
	"logs.tco:UpdatePolicies",
	"metrics.alerts:ReadConfig",
	"metrics.alerts:UpdateConfig",
	"metrics.data-analytics#high:Read",
	"metrics.data-analytics#low:Read",
	"metrics.data-setup#high:ReadConfig",
	"metrics.data-setup#high:UpdateConfig",
	"metrics.data-setup#low:ReadConfig",
	"metrics.data-setup#low:UpdateConfig",
	"metrics.recording-rules:ReadConfig",
	"metrics.recording-rules:UpdateConfig",
	"metrics.tco:ReadPolicies",
	"metrics.tco:UpdatePolicies",
	"notification-center-connectors:ReadConfig",
	"notification-center-connectors:ReadSummary",
	"notification-center-connectors:UpdateConfig",
	"notification-center-presets:ReadConfig",
	"notification-center-presets:ReadSummary",
	"notification-center-presets:UpdateConfig",
	"notification-center-routers:ReadConfig",
	"notification-center-routers:ReadSummary",
	"notification-center-routers:UpdateConfig",
	"outbound-webhooks:ReadConfig",
	"outbound-webhooks:UpdateConfig",
	"parsing-rules:ReadConfig",
	"parsing-rules:UpdateConfig",
	"security-enrichment:ReadConfig",
	"security-enrichment:UpdateConfig",
	"serverless:Read",
	"service-catalog:Read",
	"service-catalog:ReadApdexConfig",
	"service-catalog:ReadDimensionsConfig",
	"service-catalog:ReadSLIConfig",
	"service-catalog:Update",
	"service-catalog:UpdateApdexConfig",
	"service-catalog:UpdateDimensionsConfig",
	"service-catalog:UpdateSLIConfig",
	"service-map:Read",
	"slo:ReadConfig",
	"slo:UpdateConfig",
	"source-mapping:UploadMapping",
	"spans.alerts:ReadConfig",
	"spans.alerts:UpdateConfig",
	"spans.data-api#high:ReadData",
	"spans.data-api#low:ReadData",
	"spans.data-setup#low:ReadConfig",
	"spans.data-setup#low:UpdateConfig",
	"spans.events2metrics:ReadConfig",
	"spans.events2metrics:UpdateConfig",
	"spans.tco:ReadPolicies",
	"spans.tco:UpdatePolicies",
	"suppression-rules:ReadConfig",
	"suppression-rules:UpdateConfig",
	"team-actions:ReadConfig",
	"team-actions:UpdateConfig",
	"team-ai-settings:Manage",
	"team-ai-settings:ReadConfig",
	"team-api-keys-security-settings:Manage",
	"team-api-keys-security-settings:ReadConfig",
	"team-api-keys:Manage",
	"team-api-keys:ReadAccessPolicy",
	"team-api-keys:ReadConfig",
	"team-api-keys:UpdateAccessPolicy",
	"team-custom-api-keys:Manage",
	"team-custom-api-keys:ReadAccessPolicy",
	"team-custom-api-keys:ReadConfig",
	"team-custom-api-keys:UpdateAccessPolicy",
	"team-custom-enrichment:ReadConfig",
	"team-custom-enrichment:ReadData",
	"team-custom-enrichment:UpdateConfig",
	"team-custom-enrichment:UpdateData",
	"team-dashboards:Read",
	"team-dashboards:ReadAccessPolicy",
	"team-dashboards:Update",
	"team-dashboards:UpdateAccessPolicy",
	"team-groups:Manage",
	"team-groups:ReadConfig",
	"team-ip-access:Manage",
	"team-ip-access:ReadConfig",
	"team-members:Manage",
	"team-members:ReadConfig",
	"team-quota-rules:Manage",
	"team-quota-rules:Read",
	"team-quota:Manage",
	"team-quota:Read",
	"team-roles:Manage",
	"team-roles:ReadConfig",
	"team-roles:ReadSummary",
	"team-saved-views:Read",
	"team-saved-views:Update",
	"team-scopes:Manage",
	"team-scopes:ReadConfig",
	"user-actions:ReadConfig",
	"user-actions:UpdateConfig",
	"user-dashboards:Read",
	"user-dashboards:Update",
	"user-saved-views:Read",
	"user-saved-views:Update",
	"version-benchmark-tags:Read",
	"version-benchmark-tags:Update",
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ephemeralteam: %s\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ephemeralteam create | ephemeralteam delete <team-id>")
	}
	switch args[0] {
	case "create":
		return create()
	case "delete":
		if len(args) != 2 {
			return fmt.Errorf("usage: ephemeralteam delete <team-id>")
		}
		teamID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid team id %q: %w", args[1], err)
		}
		return deleteTeam(teamID)
	default:
		return fmt.Errorf("unknown subcommand %q (want create or delete)", args[0])
	}
}

// create provisions a team plus a team-scoped API key and prints the
// KEY=VALUE lines for $GITHUB_ENV on stdout.
func create() error {
	orgKey := os.Getenv(orgAPIKeyEnvVar)
	if orgKey == "" {
		// Opt-in: without the org key the workflow keeps using the shared
		// team from the CORALOGIX_API_KEY secret.
		fmt.Fprintf(os.Stderr, "ephemeralteam: %s is not set; nothing to do\n", orgAPIKeyEnvVar)
		return nil
	}
	cs, err := newOrgClientSet(orgKey)
	if err != nil {
		return err
	}
	ctx := context.Background()

	name := fmt.Sprintf("%s-%d", teamNamePrefix, time.Now().UnixNano())
	createResp, httpResp, err := cs.Teams().
		TeamServiceCreateTeamInOrg(ctx).
		TeamServiceCreateTeamInOrgRequest(teams.TeamServiceCreateTeamInOrgRequest{
			TeamName: name,
		}).
		Execute()
	if err != nil {
		return fmt.Errorf("creating team %q: %w", name, openapicxsdk.NewAPIError(httpResp, err))
	}
	teamID := createResp.TeamId.GetId()
	if teamID == 0 {
		return fmt.Errorf("backend returned no team id for team %q", name)
	}
	fmt.Fprintf(os.Stderr, "ephemeralteam: created team %d (%s)\n", teamID, name)

	keyName := name + "-key"
	hashed := false // so the response carries the plain key value
	keyResp, httpResp, err := cs.APIKeys().
		ApiKeysServiceCreateApiKey(ctx).
		CreateApiKeyRequest(apikeys.CreateApiKeyRequest{
			Name:   &keyName,
			Hashed: &hashed,
			Owner:  &apikeys.Owner{TeamId: &teamID},
			KeyPermissions: &apikeys.CreateApiKeyRequestKeyPermissions{
				Permissions: defaultTeamKeyPermissions,
			},
		}).
		Execute()
	if err != nil {
		return fmt.Errorf("creating API key for team %d: %w", teamID, openapicxsdk.NewAPIError(httpResp, err))
	}
	if keyResp.GetValue() == "" {
		return fmt.Errorf("backend returned an empty API key value for team %d", teamID)
	}
	fmt.Fprintf(os.Stderr, "ephemeralteam: minted API key %q for team %d\n", keyName, teamID)

	// Machine-readable output for the workflow. The caller must
	// ::add-mask:: the key before appending these lines to $GITHUB_ENV.
	fmt.Printf("CORALOGIX_API_KEY=%s\n", keyResp.GetValue())
	fmt.Printf("EPHEMERAL_TEAM_ID=%d\n", teamID)
	fmt.Printf("EPHEMERAL_TEAM_NAME=%s\n", name)
	return nil
}

func deleteTeam(teamID int64) error {
	orgKey := os.Getenv(orgAPIKeyEnvVar)
	if orgKey == "" {
		return fmt.Errorf("%s is not set", orgAPIKeyEnvVar)
	}
	cs, err := newOrgClientSet(orgKey)
	if err != nil {
		return err
	}
	_, httpResp, err := cs.Teams().TeamServiceDeleteTeam(context.Background(), teamID).Execute()
	if err != nil {
		return fmt.Errorf("deleting team %d: %w", teamID, openapicxsdk.NewAPIError(httpResp, err))
	}
	fmt.Fprintf(os.Stderr, "ephemeralteam: deleted team %d\n", teamID)
	return nil
}

// newOrgClientSet builds an OpenAPI SDK client authenticated with the
// org-scoped key, targeting CORALOGIX_DOMAIN or CORALOGIX_REGION — the same
// resolution tests/e2e/client.go uses for the team-scoped clients.
func newOrgClientSet(orgKey string) (*openapicxsdk.ClientSet, error) {
	builder := openapicxsdk.NewConfigBuilder().WithAPIKey(orgKey)
	if domain := os.Getenv("CORALOGIX_DOMAIN"); domain != "" {
		builder = builder.WithDomain(domain)
	} else if region := os.Getenv("CORALOGIX_REGION"); region != "" {
		builder = builder.WithRegion(region)
	} else {
		return nil, fmt.Errorf("CORALOGIX_REGION or CORALOGIX_DOMAIN must be set")
	}
	return openapicxsdk.NewClientSet(builder.Build()), nil
}
