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
	"context"
	"testing"

	connectors "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/connectors_service"
	presets "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/presets_service"
)

// Preset attachmentConfig maps to the AttachmentConfig policy.
func TestPresetExtractAttachmentConfig(t *testing.T) {
	policy := "ENABLED"
	preset := &Preset{
		Spec: PresetSpec{
			Name:             "p",
			Description:      "d",
			ConnectorType:    "genericHttps",
			EntityType:       "alerts",
			AttachmentConfig: &policy,
		},
	}

	got, err := preset.ExtractPreset()
	if err != nil {
		t.Fatalf("ExtractPreset returned error: %v", err)
	}
	if got.AttachmentConfig == nil || got.AttachmentConfig.Policy == nil {
		t.Fatalf("AttachmentConfig not set")
	}
	if *got.AttachmentConfig.Policy != presets.ATTACHMENTCONFIGPOLICY_ENABLED {
		t.Fatalf("AttachmentConfig.Policy = %q, want ENABLED", *got.AttachmentConfig.Policy)
	}

	// Unset attachmentConfig => nil (server default AUTO).
	preset.Spec.AttachmentConfig = nil
	got, err = preset.ExtractPreset()
	if err != nil {
		t.Fatalf("ExtractPreset returned error: %v", err)
	}
	if got.AttachmentConfig != nil {
		t.Fatalf("AttachmentConfig should be nil when unset")
	}
}

// Preset supports the pagerDutyIncidents connector type and the cases entity type,
// so PagerDuty-Incidents case presets can be managed as code.
func TestPresetExtractPagerdutyIncidentsAndCases(t *testing.T) {
	preset := &Preset{
		Spec: PresetSpec{
			Name:          "p",
			Description:   "d",
			ConnectorType: "pagerDutyIncidents",
			EntityType:    "cases",
		},
	}

	got, err := preset.ExtractPreset()
	if err != nil {
		t.Fatalf("ExtractPreset returned error: %v", err)
	}
	if got.ConnectorType == nil || *got.ConnectorType != presets.NOTIFICATIONCENTERCONNECTORTYPE_PAGERDUTY_INCIDENTS {
		t.Fatalf("ConnectorType = %v, want PAGERDUTY_INCIDENTS", got.ConnectorType)
	}
	if got.EntityType == nil || *got.EntityType != presets.NOTIFICATIONCENTERENTITYTYPE_CASES {
		t.Fatalf("EntityType = %v, want CASES", got.EntityType)
	}
}

// Connector supports the PAGERDUTY_INCIDENTS type and CASES config overrides.
func TestConnectorExtractPagerdutyIncidentsAndCases(t *testing.T) {
	value := "v"
	connector := &Connector{
		Spec: ConnectorSpec{
			Name:        "c",
			Description: "d",
			Type:        "pagerDutyIncidents",
			ConnectorConfig: ConnectorConfig{
				Fields: []ConnectorConfigField{{FieldName: "integrationId", Value: &value}},
			},
			ConfigOverrides: []EntityTypeConfigOverrides{
				{EntityType: "cases", Fields: []TemplatedConnectorConfigField{{FieldName: "service", Template: "PXXXXXX"}}},
			},
		},
	}

	got, err := connector.ExtractConnector(context.Background())
	if err != nil {
		t.Fatalf("ExtractConnector returned error: %v", err)
	}
	if got.Type == nil || *got.Type != connectors.NOTIFICATIONCENTERCONNECTORTYPE_PAGERDUTY_INCIDENTS {
		t.Fatalf("Type = %v, want PAGERDUTY_INCIDENTS", got.Type)
	}
	if len(got.ConfigOverrides) != 1 || got.ConfigOverrides[0].EntityType == nil ||
		*got.ConfigOverrides[0].EntityType != connectors.NOTIFICATIONCENTERENTITYTYPE_CASES {
		t.Fatalf("config override entity type = %v, want CASES", got.ConfigOverrides)
	}
}

// GlobalRouter supports disabled, fallbackTargets, and CASES routing-rule entity type.
func TestGlobalRouterExtractDisabledFallbackTargetsAndCases(t *testing.T) {
	disabled := true
	entityType := "cases"
	router := &GlobalRouter{
		Spec: GlobalRouterSpec{
			Name:        "r",
			Description: "d",
			Disabled:    &disabled,
			Rules: []RoutingRule{
				{
					Name:          "rule",
					EntityType:    &entityType,
					Condition:     "true",
					CustomDetails: map[string]string{"ruleKey": "ruleVal"},
					Targets: []RoutingTarget{
						{
							Connector:     NCRef{BackendRef: &NCBackendRef{ID: "conn-1"}},
							CustomDetails: map[string]string{"targetKey": "targetVal"},
						},
					},
				},
			},
			FallbackTargets: []FallbackTarget{
				{
					EntityType: "alerts",
					Target:     RoutingTarget{Connector: NCRef{BackendRef: &NCBackendRef{ID: "conn-1"}}},
				},
			},
		},
	}

	got, err := router.ExtractGlobalRouter(context.Background())
	if err != nil {
		t.Fatalf("ExtractGlobalRouter returned error: %v", err)
	}
	if got.Disabled == nil || !*got.Disabled {
		t.Fatalf("Disabled = %v, want true", got.Disabled)
	}
	if len(got.Rules) != 1 || got.Rules[0].EntityType == nil ||
		string(*got.Rules[0].EntityType) != "CASES" {
		t.Fatalf("rule entity type = %v, want CASES", got.Rules)
	}
	if len(got.FallbackTargets) != 1 || got.FallbackTargets[0].EntityType == nil ||
		string(*got.FallbackTargets[0].EntityType) != "ALERTS" {
		t.Fatalf("fallbackTargets = %v, want one ALERTS target", got.FallbackTargets)
	}
	if got.FallbackTargets[0].Target == nil || got.FallbackTargets[0].Target.ConnectorId == nil ||
		*got.FallbackTargets[0].Target.ConnectorId != "conn-1" {
		t.Fatalf("fallbackTargets[0].Target connector = %v, want conn-1", got.FallbackTargets[0].Target)
	}
	if got.Rules[0].CustomDetails == nil || (*got.Rules[0].CustomDetails)["ruleKey"] != "ruleVal" {
		t.Fatalf("rule custom details = %v, want ruleKey=ruleVal", got.Rules[0].CustomDetails)
	}
	if len(got.Rules[0].Targets) != 1 || got.Rules[0].Targets[0].CustomDetails == nil ||
		(*got.Rules[0].Targets[0].CustomDetails)["targetKey"] != "targetVal" {
		t.Fatalf("target custom details = %v, want targetKey=targetVal", got.Rules[0].Targets)
	}
}
