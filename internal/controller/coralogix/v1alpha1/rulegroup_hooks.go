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

	"github.com/go-logr/logr"

	rulegroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/rule_groups_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The hooks write the rule matchers and the subgroups of the request from the applications,
// subsystems, severities, and subgroups of the spec. Each run builds both lists whole, so a hook
// that runs again sends the same body.

func (r *RuleGroupReconciler) beforeCreate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.RuleGroup, body *rulegroups.RuleGroupsServiceCreateRuleGroupRequest) error { //nolint:unparam // the generated lifecycle fixes the signature
	body.RuleMatchers = cr.Spec.ExpandRuleMatchers()
	body.RuleSubgroups = cr.Spec.ExpandRuleSubgroups()
	return nil
}

func (r *RuleGroupReconciler) beforeUpdate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.RuleGroup, body *rulegroups.RuleGroupsServiceCreateRuleGroupRequest) error { //nolint:unparam // the generated lifecycle fixes the signature
	body.RuleMatchers = cr.Spec.ExpandRuleMatchers()
	body.RuleSubgroups = cr.Spec.ExpandRuleSubgroups()
	return nil
}
