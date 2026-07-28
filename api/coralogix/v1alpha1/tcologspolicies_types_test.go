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

	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

// A TCO logs policy with dataset-routing targets maps each target through to
// CreateGenericPolicyRequest.Targets, including per-target priority and the
// quota-based priority override.
func TestTCOLogsPolicyExtractTargets(t *testing.T) {
	policy := &TCOLogsPolicy{
		Name:       "policy-with-targets",
		Priority:   "medium",
		Severities: []TCOPolicySeverity{"info"},
		Targets: []TCOPolicyTarget{
			{
				Dataspace: "default",
				Dataset:   "logs",
				Priority:  ptr.To("high"),
			},
			{
				Dataspace: "default",
				Dataset:   "audit_logs",
				Priority:  ptr.To("block"),
				PriorityOverride: &TCOPriorityOverride{
					QuotaBased: &TCOQuotaBased{
						UsageTiers: []TCOUsageTier{
							{
								DailyQuotaPercentage: resource.MustParse("80"),
								Priority:             "low",
							},
						},
					},
				},
			},
		},
	}

	// nil archive-retentions client: none of the targets reference an
	// archiveRetention, so expandArchiveRetention returns early without a call.
	req, err := policy.ExtractCreateLogPolicyRequest(context.Background(), nil)
	if err != nil {
		t.Fatalf("ExtractCreateLogPolicyRequest returned error: %v", err)
	}

	targets := req.Policy.Targets
	if len(targets) != 2 {
		t.Fatalf("Targets length = %d, want 2", len(targets))
	}

	if ptr.Deref(targets[0].Dataspace, "") != "default" || ptr.Deref(targets[0].Dataset, "") != "logs" {
		t.Fatalf("Targets[0] dataspace/dataset = %q/%q, want default/logs",
			ptr.Deref(targets[0].Dataspace, ""), ptr.Deref(targets[0].Dataset, ""))
	}
	if targets[0].Priority == nil || *targets[0].Priority != tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_HIGH {
		t.Fatalf("Targets[0].Priority = %v, want PRIORITY_TYPE_HIGH", targets[0].Priority)
	}

	if ptr.Deref(targets[1].Dataspace, "") != "default" || ptr.Deref(targets[1].Dataset, "") != "audit_logs" {
		t.Fatalf("Targets[1] dataspace/dataset = %q/%q, want default/audit_logs",
			ptr.Deref(targets[1].Dataspace, ""), ptr.Deref(targets[1].Dataset, ""))
	}
	if targets[1].Priority == nil || *targets[1].Priority != tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_BLOCK {
		t.Fatalf("Targets[1].Priority = %v, want PRIORITY_TYPE_BLOCK", targets[1].Priority)
	}

	if targets[1].PriorityOverride == nil || targets[1].PriorityOverride.QuotaBased == nil {
		t.Fatalf("Targets[1].PriorityOverride.QuotaBased not set")
	}
	tiers := targets[1].PriorityOverride.QuotaBased.UsageTiers
	if len(tiers) != 1 {
		t.Fatalf("Targets[1] usage tiers = %d, want 1", len(tiers))
	}
	if ptr.Deref(tiers[0].DailyQuotaPercentage, 0) != 80 {
		t.Fatalf("UsageTiers[0].DailyQuotaPercentage = %v, want 80", tiers[0].DailyQuotaPercentage)
	}
	if tiers[0].Priority == nil || *tiers[0].Priority != tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_LOW {
		t.Fatalf("UsageTiers[0].Priority = %v, want PRIORITY_TYPE_LOW", tiers[0].Priority)
	}
}

// A policy-level priorityOverride maps through to CreateGenericPolicyRequest,
// and a policy without targets leaves the Targets list unset (backward compatible).
func TestTCOLogsPolicyExtractPolicyPriorityOverrideAndNoTargets(t *testing.T) {
	policy := &TCOLogsPolicy{
		Name:       "policy-override-no-targets",
		Priority:   "high",
		Severities: []TCOPolicySeverity{"info"},
		PriorityOverride: &TCOPriorityOverride{
			QuotaBased: &TCOQuotaBased{
				UsageTiers: []TCOUsageTier{
					{
						DailyQuotaPercentage: resource.MustParse("50"),
						Priority:             "medium",
					},
				},
			},
		},
	}

	req, err := policy.ExtractCreateLogPolicyRequest(context.Background(), nil)
	if err != nil {
		t.Fatalf("ExtractCreateLogPolicyRequest returned error: %v", err)
	}

	if req.Policy.Targets != nil {
		t.Fatalf("Targets should be nil when unset, got %v", req.Policy.Targets)
	}
	if req.Policy.PriorityOverride == nil || req.Policy.PriorityOverride.QuotaBased == nil {
		t.Fatalf("policy PriorityOverride.QuotaBased not set")
	}
	tiers := req.Policy.PriorityOverride.QuotaBased.UsageTiers
	if len(tiers) != 1 || tiers[0].Priority == nil ||
		*tiers[0].Priority != tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_MEDIUM {
		t.Fatalf("policy usage tier not mapped correctly: %+v", tiers)
	}
}
