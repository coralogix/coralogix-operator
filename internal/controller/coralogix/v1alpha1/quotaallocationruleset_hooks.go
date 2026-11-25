// Copyright 2026 Coralogix Ltd.
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
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	quotas "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/quota_allocation_rule_set_service"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

const quotaAllocationRuleSetImportID = "quota-allocation-rule-set"

// validate keeps the released single-owner and allocation checks before any API call.
func (r *QuotaAllocationRuleSetReconciler) validate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.QuotaAllocationRuleSet) error {
	if err := r.ensureSingleSelectedRuleSet(ctx, cr); err != nil {
		return err
	}
	return cr.Spec.ValidateQuotaAllocationRules()
}

// beforeUpdate preserves managed rules and the released explicit percentage default.
// It starts from the CR each time, so a retry on the same body does not append rules twice.
func (r *QuotaAllocationRuleSetReconciler) beforeUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.QuotaAllocationRuleSet, body *quotas.ReplaceQuotaAllocationRuleSetRequest) error {
	fresh, err := cr.UpdateRequest(ctx)
	if err != nil {
		return err
	}
	plannedRules := fresh.RuleSet.Rules
	for i := range plannedRules {
		if plannedRules[i].AllocationType == nil {
			plannedRules[i].AllocationType = quotas.QUOTAALLOCATIONTYPE_QUOTA_ALLOCATION_TYPE_PERCENTAGE.Ptr()
		}
	}
	current, err := r.getCurrentRuleSet(ctx)
	if err != nil {
		return fmt.Errorf("error on preserving managed quota allocation rules: %w", err)
	}
	if err := RejectManagedQuotaAllocationRuleCollisions(plannedRules, current.Rules); err != nil {
		return fmt.Errorf("error on preserving managed quota allocation rules: %w", err)
	}
	body.RuleSet.Rules = PreserveManagedQuotaAllocationRules(plannedRules, current.Rules)
	return nil
}

func (r *QuotaAllocationRuleSetReconciler) getCurrentRuleSet(ctx context.Context) (*quotas.QuotaAllocationEntityTypeRuleSet, error) {
	response, httpResp, err := r.Client.
		QuotaAllocationRuleSetServiceGetQuotaAllocationRuleSet(ctx).
		Id(quotaAllocationRuleSetImportID).
		Execute()
	if err != nil {
		apiErr := cxsdk.NewAPIError(httpResp, err)
		if cxsdk.IsNotFound(apiErr) {
			return &quotas.QuotaAllocationEntityTypeRuleSet{}, nil
		}
		return nil, apiErr
	}
	if response == nil || response.RuleSet == nil {
		return &quotas.QuotaAllocationEntityTypeRuleSet{}, nil
	}
	return response.RuleSet, nil
}

func (r *QuotaAllocationRuleSetReconciler) ensureSingleSelectedRuleSet(ctx context.Context, cr *coralogixv1alpha1.QuotaAllocationRuleSet) error {
	other, err := r.findOtherSelectedRuleSet(ctx, cr)
	if err != nil {
		return err
	}
	if other != nil {
		return fmt.Errorf("only one selected QuotaAllocationRuleSet can manage account-level quota allocation rules; %s/%s already exists", other.Namespace, other.Name)
	}
	return nil
}

func (r *QuotaAllocationRuleSetReconciler) findOtherSelectedRuleSet(ctx context.Context, cr *coralogixv1alpha1.QuotaAllocationRuleSet) (*coralogixv1alpha1.QuotaAllocationRuleSet, error) {
	ruleSets := &coralogixv1alpha1.QuotaAllocationRuleSetList{}
	if err := config.GetClient().List(ctx, ruleSets); err != nil {
		return nil, fmt.Errorf("error on listing quota allocation rule sets: %w", err)
	}
	for i := range ruleSets.Items {
		ruleSet := &ruleSets.Items[i]
		if ruleSet.UID == cr.UID || !ruleSet.DeletionTimestamp.IsZero() || !config.GetConfig().Selector.Matches(ruleSet.Labels, ruleSet.Namespace) {
			continue
		}
		return ruleSet, nil
	}
	return nil, nil
}

// PreserveManagedQuotaAllocationRules appends backend-managed rules that are not replaced by the planned entity types.
func PreserveManagedQuotaAllocationRules(plannedRules, currentRules []quotas.QuotaAllocationEntityTypeRule) []quotas.QuotaAllocationEntityTypeRule {
	plannedEntityTypes := make(map[string]struct{}, len(plannedRules))
	for _, rule := range plannedRules {
		plannedEntityTypes[rule.EntityType] = struct{}{}
	}
	result := make([]quotas.QuotaAllocationEntityTypeRule, 0, len(plannedRules)+len(currentRules))
	result = append(result, plannedRules...)
	for _, rule := range currentRules {
		if rule.GetCxManaged() {
			if _, replaces := plannedEntityTypes[rule.EntityType]; !replaces {
				rule.CxManaged = nil
				result = append(result, rule)
			}
		}
	}
	return result
}

// RejectManagedQuotaAllocationRuleCollisions rejects user rules that would replace backend-managed entity types.
func RejectManagedQuotaAllocationRuleCollisions(plannedRules, currentRules []quotas.QuotaAllocationEntityTypeRule) error {
	managedEntityTypes := make(map[string]struct{}, len(currentRules))
	for _, rule := range currentRules {
		if rule.GetCxManaged() {
			managedEntityTypes[rule.EntityType] = struct{}{}
		}
	}
	for _, rule := range plannedRules {
		if _, exists := managedEntityTypes[rule.EntityType]; exists {
			return fmt.Errorf("quota allocation rule entityType %q is managed by Coralogix and cannot be replaced", rule.EntityType)
		}
	}
	return nil
}

// delete keeps managed rules and avoids removing a rule set owned by another synced CR.
func (r *QuotaAllocationRuleSetReconciler) delete(ctx context.Context, log logr.Logger, cr *coralogixv1alpha1.QuotaAllocationRuleSet) error {
	other, err := r.findOtherSelectedRuleSet(ctx, cr)
	if err != nil {
		return err
	}
	if quotaAllocationRuleSetRemoteSynced(other) {
		log.Info("Skipping remote QuotaAllocationRuleSet deletion because another selected resource is synced", "otherNamespace", other.Namespace, "otherName", other.Name)
		return nil
	}
	if other != nil {
		log.Info("Continuing remote QuotaAllocationRuleSet deletion because another selected resource has not synced", "otherNamespace", other.Namespace, "otherName", other.Name)
	}
	log.Info("Deleting QuotaAllocationRuleSet")
	ruleSet, err := r.getCurrentRuleSet(ctx)
	if err != nil {
		return fmt.Errorf("error on fetching quota-allocation-rule-set before deletion: %w", err)
	}
	preservedRules := PreserveManagedQuotaAllocationRules(nil, ruleSet.Rules)
	if len(preservedRules) == 0 {
		_, httpResp, err := r.Client.QuotaAllocationRuleSetServiceDeleteQuotaAllocationRuleSet(ctx).Execute()
		if err != nil {
			apiErr := cxsdk.NewAPIError(httpResp, err)
			if !cxsdk.IsNotFound(apiErr) {
				return apiErr
			}
		}
		log.Info("quota-allocation-rule-set was deleted from remote")
		return nil
	}
	replaceRequest := quotas.ReplaceQuotaAllocationRuleSetRequest{RuleSet: quotas.QuotaAllocationEntityTypeRuleSet{Rules: preservedRules}}
	log.Info("Preserving managed quota-allocation-rule-set rules")
	_, httpResp, err := r.Client.QuotaAllocationRuleSetServiceReplaceQuotaAllocationRuleSet(ctx).
		ReplaceQuotaAllocationRuleSetRequest(replaceRequest).Execute()
	if err != nil {
		return fmt.Errorf("error on preserving managed quota-allocation-rule-set rules: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("quota-allocation-rule-set user-managed rules were deleted from remote")
	return nil
}

func quotaAllocationRuleSetRemoteSynced(ruleSet *coralogixv1alpha1.QuotaAllocationRuleSet) bool {
	if ruleSet == nil {
		return false
	}
	condition := meta.FindStatusCondition(ruleSet.Status.Conditions, utils.ConditionTypeRemoteSynced)
	return condition != nil && condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == ruleSet.Generation
}
