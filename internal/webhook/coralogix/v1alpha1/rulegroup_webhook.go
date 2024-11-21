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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
)

// nolint:unused
// log is for logging in this package.
var rulegrouplog = logf.Log.WithName("rulegroup-resource")

// SetupRuleGroupWebhookWithManager registers the webhook for RuleGroup in the manager.
func SetupRuleGroupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1alpha1.RuleGroup{}).
		WithValidator(&RuleGroupCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-rulegroup,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=rulegroups,verbs=create;update,versions=v1alpha1,name=vrulegroup-v1alpha1.kb.io,admissionReviewVersions=v1

// RuleGroupCustomValidator struct is responsible for validating the RuleGroup resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type RuleGroupCustomValidator struct{}

var _ webhook.CustomValidator = &RuleGroupCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type RuleGroup.
func (v *RuleGroupCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	rulegroup, ok := obj.(*coralogixv1alpha1.RuleGroup)
	if !ok {
		return nil, fmt.Errorf("expected a RuleGroup object but got %T", obj)
	}
	rulegrouplog.Info("Validation for RuleGroup upon creation", "name", rulegroup.GetName())

	return validateRulesTypesSet(*rulegroup)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type RuleGroup.
func (v *RuleGroupCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	rulegroup, ok := newObj.(*coralogixv1alpha1.RuleGroup)
	if !ok {
		return nil, fmt.Errorf("expected a RuleGroup object for the newObj but got %T", newObj)
	}
	rulegrouplog.Info("Validation for RuleGroup upon update", "name", rulegroup.GetName())

	return validateRulesTypesSet(*rulegroup)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type RuleGroup.
func (v *RuleGroupCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateRulesTypesSet(ruleGroup coralogixv1alpha1.RuleGroup) (admission.Warnings, error) {
	var warnings admission.Warnings
	var errorsMessages []string

	for _, subGroup := range ruleGroup.Spec.RuleSubgroups {
		for _, rule := range subGroup.Rules {
			typesSet := getRuleTypesSet(rule)
			if len(typesSet) == 0 {
				msg := fmt.Sprintf("at least one rule type should be set in rule '%s'", rule.Name)
				warnings = append(warnings, msg)
				errorsMessages = append(errorsMessages, msg)
			}

			if len(typesSet) > 1 {
				msg := fmt.Sprintf("only one rule type should be set in rule '%s', but got: %v", rule.Name, typesSet)
				warnings = append(warnings, msg)
				errorsMessages = append(errorsMessages, msg)
			}
		}
	}

	if len(errorsMessages) > 0 {
		monitoring.TotalRejectedRulesGroupsMetric.Inc()
		return warnings, fmt.Errorf("%v", errorsMessages)
	}

	return nil, nil
}

func getRuleTypesSet(rule coralogixv1alpha1.Rule) []string {
	var typesSet []string
	if rule.Parse != nil {
		typesSet = append(typesSet, "Parse")
	}
	if rule.Block != nil {
		typesSet = append(typesSet, "Block")
	}
	if rule.JsonExtract != nil {
		typesSet = append(typesSet, "JsonExtract")
	}
	if rule.Replace != nil {
		typesSet = append(typesSet, "Replace")
	}
	if rule.ExtractTimestamp != nil {
		typesSet = append(typesSet, "ExtractTimestamp")
	}
	if rule.RemoveFields != nil {
		typesSet = append(typesSet, "RemoveFields")
	}
	if rule.JsonStringify != nil {
		typesSet = append(typesSet, "JsonStringify")
	}
	if rule.Extract != nil {
		typesSet = append(typesSet, "Extract")
	}
	if rule.ParseJsonField != nil {
		typesSet = append(typesSet, "ParseJsonField")
	}

	return typesSet

}
