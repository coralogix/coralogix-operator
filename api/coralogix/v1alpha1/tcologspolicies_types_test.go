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
	"strings"
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
				// A dataset other than default/logs cannot use high or block, and
				// this fallback has to be at least as restrictive as the tier below.
				Priority: ptr.To("low"),
				PriorityOverride: &TCOPriorityOverride{
					QuotaBased: &TCOQuotaBased{
						UsageTiers: []TCOUsageTier{
							{
								DailyQuotaPercentage: resource.MustParse("80"),
								Priority:             "medium",
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
	if targets[1].Priority == nil || *targets[1].Priority != tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_LOW {
		t.Fatalf("Targets[1].Priority = %v, want PRIORITY_TYPE_LOW", targets[1].Priority)
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
	if tiers[0].Priority == nil || *tiers[0].Priority != tcopolicies.QUOTAV1PRIORITY_PRIORITY_TYPE_MEDIUM {
		t.Fatalf("UsageTiers[0].Priority = %v, want PRIORITY_TYPE_MEDIUM", tiers[0].Priority)
	}
}

// A policy-level priorityOverride maps through to CreateGenericPolicyRequest,
// and a policy without targets leaves the Targets list unset (backward compatible).
func TestTCOLogsPolicyExtractPolicyPriorityOverrideAndNoTargets(t *testing.T) {
	policy := &TCOLogsPolicy{
		Name: "policy-override-no-targets",
		// Least restrictive priority the "medium" tier below is allowed to fall back to.
		Priority:   "low",
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

// dailyQuotaPercentage outside 0-100 is rejected during extraction. This can't be
// caught by CRD admission: the Kubernetes CEL "quantity" library needed to bound a
// resource.Quantity field isn't available before Kubernetes 1.29, and this repo's
// CRDs must install cleanly on 1.28.
func TestTCOLogsPolicyExtractRejectsOutOfRangeQuotaPercentage(t *testing.T) {
	policy := &TCOLogsPolicy{
		Name:       "policy-invalid-quota-percentage",
		Priority:   "medium",
		Severities: []TCOPolicySeverity{"info"},
		PriorityOverride: &TCOPriorityOverride{
			QuotaBased: &TCOQuotaBased{
				UsageTiers: []TCOUsageTier{
					{
						DailyQuotaPercentage: resource.MustParse("120"),
						Priority:             "low",
					},
				},
			},
		},
	}

	_, err := policy.ExtractCreateLogPolicyRequest(context.Background(), nil)
	if err == nil {
		t.Fatal("ExtractCreateLogPolicyRequest should have returned an error for a dailyQuotaPercentage of 120")
	}
	if !strings.Contains(err.Error(), "dailyQuotaPercentage must be between 0 and 100") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Coralogix requires the priority a quota-based override falls back to, once every
// tier is consumed, to be at least as restrictive as the last tier. A policy whose
// own priority is less restrictive than its last tier is rejected during extraction.
func TestTCOLogsPolicyExtractRejectsLessRestrictiveQuotaFallback(t *testing.T) {
	policy := &TCOLogsPolicy{
		Name:       "policy-lenient-fallback",
		Priority:   "medium",
		Severities: []TCOPolicySeverity{"info"},
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
	}

	_, err := policy.ExtractCreateLogPolicyRequest(context.Background(), nil)
	if err == nil {
		t.Fatal("ExtractCreateLogPolicyRequest should have returned an error for a medium fallback behind a low tier")
	}
	if !strings.Contains(err.Error(), "must be at least as restrictive as the last usage tier priority") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// A target without its own priority inherits the policy's, so the inherited value is
// what its quota-based override's fallback is measured against.
func TestTCOLogsPolicyExtractRejectsLessRestrictiveInheritedQuotaFallback(t *testing.T) {
	policy := &TCOLogsPolicy{
		Name:       "policy-lenient-inherited-fallback",
		Priority:   "medium",
		Severities: []TCOPolicySeverity{"info"},
		Targets: []TCOPolicyTarget{
			{
				Dataspace: "default",
				Dataset:   "audit_logs",
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

	_, err := policy.ExtractCreateLogPolicyRequest(context.Background(), nil)
	if err == nil {
		t.Fatal("ExtractCreateLogPolicyRequest should have returned an error for an inherited medium fallback behind a low tier")
	}
	if !strings.Contains(err.Error(), "must be at least as restrictive as the last usage tier priority") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Usage tiers are consumed in order, so a descending dailyQuotaPercentage is rejected.
func TestTCOLogsPolicyExtractRejectsUnorderedUsageTiers(t *testing.T) {
	policy := &TCOLogsPolicy{
		Name:       "policy-unordered-tiers",
		Priority:   "block",
		Severities: []TCOPolicySeverity{"info"},
		PriorityOverride: &TCOPriorityOverride{
			QuotaBased: &TCOQuotaBased{
				UsageTiers: []TCOUsageTier{
					{
						DailyQuotaPercentage: resource.MustParse("80"),
						Priority:             "medium",
					},
					{
						DailyQuotaPercentage: resource.MustParse("50"),
						Priority:             "low",
					},
				},
			},
		},
	}

	_, err := policy.ExtractCreateLogPolicyRequest(context.Background(), nil)
	if err == nil {
		t.Fatal("ExtractCreateLogPolicyRequest should have returned an error for descending usage tiers")
	}
	if !strings.Contains(err.Error(), "must be ordered by ascending dailyQuotaPercentage") {
		t.Fatalf("unexpected error: %v", err)
	}
}
