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
	"fmt"
	"time"

	"github.com/coralogix/coralogix-operator/v2/internal/utils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	quotas "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/quota_allocation_rule_set_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
)

const quotaAllocationRuleSetImportID = "quota-allocation-rule-set"

// QuotaAllocationRuleSetReconciler reconciles a QuotaAllocationRuleSet object.
type QuotaAllocationRuleSetReconciler struct {
	QuotaAllocationRulesClient *quotas.QuotaAllocationRuleSetServiceAPIService
	Interval                   time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=quotaallocationrulesets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=quotaallocationrulesets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=quotaallocationrulesets/finalizers,verbs=update

func (r *QuotaAllocationRuleSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.QuotaAllocationRuleSet{}, r)
}

func (r *QuotaAllocationRuleSetReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *QuotaAllocationRuleSetReconciler) replace(ctx context.Context, log logr.Logger, quotaAllocationRuleSet *coralogixv1alpha1.QuotaAllocationRuleSet) error {
	if err := r.ensureSingleSelectedRuleSet(ctx, quotaAllocationRuleSet); err != nil {
		return err
	}

	ruleSet, err := quotaAllocationRuleSet.Spec.ExtractQuotaAllocationRuleSetRequest()
	if err != nil {
		return fmt.Errorf("error on extracting quota allocation rule set request: %w", err)
	}

	ruleSet.Rules, err = r.withPreservedManagedRules(ctx, ruleSet.Rules)
	if err != nil {
		return fmt.Errorf("error on preserving managed quota allocation rules: %w", err)
	}

	replaceRequest := quotas.ReplaceQuotaAllocationRuleSetRequest{RuleSet: *ruleSet}
	log.Info("Replacing remote quota-allocation-rule-set", "quota-allocation-rule-set", utils.FormatJSON(replaceRequest))
	replaceResponse, httpResp, err := r.QuotaAllocationRulesClient.
		QuotaAllocationRuleSetServiceReplaceQuotaAllocationRuleSet(ctx).
		ReplaceQuotaAllocationRuleSetRequest(replaceRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on replacing remote quota-allocation-rule-set: %w", oapicxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote quota-allocation-rule-set replaced", "response", utils.FormatJSON(replaceResponse))
	return nil
}

func (r *QuotaAllocationRuleSetReconciler) withPreservedManagedRules(ctx context.Context, plannedRules []quotas.QuotaAllocationEntityTypeRule) ([]quotas.QuotaAllocationEntityTypeRule, error) {
	currentRuleSet, err := r.getCurrentRuleSet(ctx)
	if err != nil {
		return nil, err
	}
	if err := RejectManagedQuotaAllocationRuleCollisions(plannedRules, currentRuleSet.Rules); err != nil {
		return nil, err
	}
	return PreserveManagedQuotaAllocationRules(plannedRules, currentRuleSet.Rules), nil
}

func (r *QuotaAllocationRuleSetReconciler) getCurrentRuleSet(ctx context.Context) (*quotas.QuotaAllocationEntityTypeRuleSet, error) {
	getResponse, httpResp, err := r.QuotaAllocationRulesClient.
		QuotaAllocationRuleSetServiceGetQuotaAllocationRuleSet(ctx).
		Id(quotaAllocationRuleSetImportID).
		Execute()
	if err != nil {
		apiErr := oapicxsdk.NewAPIError(httpResp, err)
		if oapicxsdk.IsNotFound(apiErr) {
			return &quotas.QuotaAllocationEntityTypeRuleSet{}, nil
		}
		return nil, apiErr
	}

	if getResponse == nil || getResponse.RuleSet == nil {
		return &quotas.QuotaAllocationEntityTypeRuleSet{}, nil
	}

	return getResponse.RuleSet, nil
}

func (r *QuotaAllocationRuleSetReconciler) ensureSingleSelectedRuleSet(ctx context.Context, quotaAllocationRuleSet *coralogixv1alpha1.QuotaAllocationRuleSet) error {
	other, err := r.findOtherSelectedRuleSet(ctx, quotaAllocationRuleSet)
	if err != nil {
		return err
	}
	if other != nil {
		return fmt.Errorf(
			"only one selected QuotaAllocationRuleSet can manage account-level quota allocation rules; %s/%s already exists",
			other.Namespace,
			other.Name,
		)
	}
	return nil
}

func (r *QuotaAllocationRuleSetReconciler) findOtherSelectedRuleSet(ctx context.Context, quotaAllocationRuleSet *coralogixv1alpha1.QuotaAllocationRuleSet) (*coralogixv1alpha1.QuotaAllocationRuleSet, error) {
	ruleSets := &coralogixv1alpha1.QuotaAllocationRuleSetList{}
	if err := config.GetClient().List(ctx, ruleSets); err != nil {
		return nil, fmt.Errorf("error on listing quota allocation rule sets: %w", err)
	}

	for i := range ruleSets.Items {
		ruleSet := &ruleSets.Items[i]
		if ruleSet.UID == quotaAllocationRuleSet.UID {
			continue
		}
		if !ruleSet.DeletionTimestamp.IsZero() {
			continue
		}
		if !config.GetConfig().Selector.Matches(ruleSet.Labels, ruleSet.Namespace) {
			continue
		}
		return ruleSet, nil
	}

	return nil, nil
}

// PreserveManagedQuotaAllocationRules appends backend-managed rules that are not replaced by the planned entity types.
func PreserveManagedQuotaAllocationRules(
	plannedRules []quotas.QuotaAllocationEntityTypeRule,
	currentRules []quotas.QuotaAllocationEntityTypeRule,
) []quotas.QuotaAllocationEntityTypeRule {
	plannedEntityTypes := make(map[string]struct{}, len(plannedRules))
	for _, rule := range plannedRules {
		plannedEntityTypes[rule.EntityType] = struct{}{}
	}

	result := make([]quotas.QuotaAllocationEntityTypeRule, 0, len(plannedRules)+len(currentRules))
	result = append(result, plannedRules...)

	for _, rule := range currentRules {
		if rule.GetCxManaged() {
			if _, replacesManagedRule := plannedEntityTypes[rule.EntityType]; !replacesManagedRule {
				result = append(result, quotaAllocationRuleWithoutReadOnlyFields(rule))
			}
		}
	}

	return result
}

func quotaAllocationRuleWithoutReadOnlyFields(rule quotas.QuotaAllocationEntityTypeRule) quotas.QuotaAllocationEntityTypeRule {
	rule.CxManaged = nil
	return rule
}

// RejectManagedQuotaAllocationRuleCollisions rejects user rules that would replace backend-managed entity types.
func RejectManagedQuotaAllocationRuleCollisions(
	plannedRules []quotas.QuotaAllocationEntityTypeRule,
	currentRules []quotas.QuotaAllocationEntityTypeRule,
) error {
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

func (r *QuotaAllocationRuleSetReconciler) FinalizerName() string {
	return "quota-allocation-rule-set.coralogix.com/finalizer"
}

func (r *QuotaAllocationRuleSetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	quotaAllocationRuleSet := obj.(*coralogixv1alpha1.QuotaAllocationRuleSet)
	if err := r.replace(ctx, log, quotaAllocationRuleSet); err != nil {
		return err
	}

	return coralogixreconciler.AddFinalizer(ctx, log, quotaAllocationRuleSet, r)
}

func (r *QuotaAllocationRuleSetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	quotaAllocationRuleSet := obj.(*coralogixv1alpha1.QuotaAllocationRuleSet)
	if err := r.replace(ctx, log, quotaAllocationRuleSet); err != nil {
		return err
	}
	return coralogixreconciler.AddFinalizer(ctx, log, quotaAllocationRuleSet, r)
}

func (r *QuotaAllocationRuleSetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	quotaAllocationRuleSet := obj.(*coralogixv1alpha1.QuotaAllocationRuleSet)
	if other, err := r.findOtherSelectedRuleSet(ctx, quotaAllocationRuleSet); err != nil {
		return err
	} else if quotaAllocationRuleSetRemoteSynced(other) {
		log.Info(
			"Skipping remote QuotaAllocationRuleSet deletion because another selected resource is synced",
			"otherNamespace", other.Namespace,
			"otherName", other.Name,
		)
		return nil
	} else if other != nil {
		log.Info(
			"Continuing remote QuotaAllocationRuleSet deletion because another selected resource has not synced",
			"otherNamespace", other.Namespace,
			"otherName", other.Name,
		)
	}

	log.Info("Deleting QuotaAllocationRuleSet")
	ruleSet, err := r.getCurrentRuleSet(ctx)
	if err != nil {
		return fmt.Errorf("error on fetching quota-allocation-rule-set before deletion: %w", err)
	}

	preservedRules := PreserveManagedQuotaAllocationRules(nil, ruleSet.Rules)
	if len(preservedRules) == 0 {
		_, httpResp, err := r.QuotaAllocationRulesClient.
			QuotaAllocationRuleSetServiceDeleteQuotaAllocationRuleSet(ctx).
			Execute()
		if err != nil {
			apiErr := oapicxsdk.NewAPIError(httpResp, err)
			if !oapicxsdk.IsNotFound(apiErr) {
				log.Error(err, "Received an error while deleting a QuotaAllocationRuleSet")
				return apiErr
			}
		}
		log.Info("quota-allocation-rule-set was deleted from remote")
		return nil
	}

	replaceRequest := quotas.ReplaceQuotaAllocationRuleSetRequest{
		RuleSet: quotas.QuotaAllocationEntityTypeRuleSet{Rules: preservedRules},
	}
	log.Info("Preserving managed quota-allocation-rule-set rules", "quota-allocation-rule-set", utils.FormatJSON(replaceRequest))
	_, httpResp, err := r.QuotaAllocationRulesClient.
		QuotaAllocationRuleSetServiceReplaceQuotaAllocationRuleSet(ctx).
		ReplaceQuotaAllocationRuleSetRequest(replaceRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on preserving managed quota-allocation-rule-set rules: %w", oapicxsdk.NewAPIError(httpResp, err))
	}

	log.Info("quota-allocation-rule-set user-managed rules were deleted from remote")
	return nil
}

func quotaAllocationRuleSetRemoteSynced(ruleSet *coralogixv1alpha1.QuotaAllocationRuleSet) bool {
	if ruleSet == nil {
		return false
	}

	condition := meta.FindStatusCondition(ruleSet.Status.Conditions, utils.ConditionTypeRemoteSynced)
	return condition != nil &&
		condition.Status == metav1.ConditionTrue &&
		condition.ObservedGeneration == ruleSet.Generation
}

// SetupWithManager sets up the controller with the Manager.
func (r *QuotaAllocationRuleSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.QuotaAllocationRuleSet{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
