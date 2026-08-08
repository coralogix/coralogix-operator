// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"testing"

	webhooks "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/outgoing_webhooks_service"
)

func TestExtractOutgoingWebhookInputDataMicrosoftTeamsWorkflow(t *testing.T) {
	spec := OutboundWebhookSpec{
		Name: "teams-workflow-webhook",
		OutboundWebhookType: OutboundWebhookType{
			MicrosoftTeamsWorkflow: &MicrosoftTeamsWorkflow{
				Url: "https://example.com/workflow",
			},
		},
	}

	data, err := spec.ExtractOutgoingWebhookInputData()
	if err != nil {
		t.Fatalf("ExtractOutgoingWebhookInputData() error = %v", err)
	}

	if data.OutgoingWebhookInputDataMsTeamsWorkflow == nil {
		t.Fatalf("expected Teams Workflow input data, got %#v", data)
	}
	if data.OutgoingWebhookInputDataMicrosoftTeams != nil {
		t.Fatalf("expected Teams Workflow not legacy Microsoft Teams, got legacy data: %#v", data.OutgoingWebhookInputDataMicrosoftTeams)
	}

	teamsWorkflow := data.OutgoingWebhookInputDataMsTeamsWorkflow
	if teamsWorkflow.Name == nil || *teamsWorkflow.Name != spec.Name {
		t.Fatalf("Teams Workflow name = %v, want %q", teamsWorkflow.Name, spec.Name)
	}
	if teamsWorkflow.Type == nil || *teamsWorkflow.Type != webhooks.WEBHOOKTYPE_MS_TEAMS_WORKFLOW {
		t.Fatalf("Teams Workflow type = %v, want %s", teamsWorkflow.Type, webhooks.WEBHOOKTYPE_MS_TEAMS_WORKFLOW)
	}
	if teamsWorkflow.Url == nil || *teamsWorkflow.Url != spec.OutboundWebhookType.MicrosoftTeamsWorkflow.Url {
		t.Fatalf("Teams Workflow URL = %v, want %q", teamsWorkflow.Url, spec.OutboundWebhookType.MicrosoftTeamsWorkflow.Url)
	}
	if teamsWorkflow.MsTeamsWorkflow == nil {
		t.Fatal("Teams Workflow config map is nil")
	}
}

func TestExtractOutgoingWebhookInputDataMicrosoftTeamsLegacyRemainsDistinct(t *testing.T) {
	spec := OutboundWebhookSpec{
		Name: "legacy-teams-webhook",
		OutboundWebhookType: OutboundWebhookType{
			MicrosoftTeams: &MicrosoftTeams{
				Url: "https://example.com/legacy-teams",
			},
		},
	}

	data, err := spec.ExtractOutgoingWebhookInputData()
	if err != nil {
		t.Fatalf("ExtractOutgoingWebhookInputData() error = %v", err)
	}

	if data.OutgoingWebhookInputDataMicrosoftTeams == nil {
		t.Fatalf("expected legacy Microsoft Teams input data, got %#v", data)
	}
	if data.OutgoingWebhookInputDataMsTeamsWorkflow != nil {
		t.Fatalf("expected legacy Microsoft Teams not Teams Workflow, got workflow data: %#v", data.OutgoingWebhookInputDataMsTeamsWorkflow)
	}

	legacyTeams := data.OutgoingWebhookInputDataMicrosoftTeams
	if legacyTeams.Type == nil || *legacyTeams.Type != webhooks.WEBHOOKTYPE_MICROSOFT_TEAMS {
		t.Fatalf("legacy Microsoft Teams type = %v, want %s", legacyTeams.Type, webhooks.WEBHOOKTYPE_MICROSOFT_TEAMS)
	}
}
