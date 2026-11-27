// Copyright 2026 Coralogix Ltd.
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

package v1alpha1

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	recordingrulesservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/recording_rules_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// beforeCreate names the remote set after the CR: the name of the CR and the operator suffix
// (--recording-rule-group-set-suffix). The CR has no name field, and Update sends no name, so the
// stored name stays. The hook sets the whole name, so running it again leaves the same body.
func (r *RecordingRuleGroupSetReconciler) beforeCreate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.RecordingRuleGroupSet, body *recordingrulesservice.CreateRuleGroupSet) error { //nolint:unparam // the generated lifecycle fixes the signature
	body.Name = ptr.To(cr.Name + config.GetConfig().RecordingRuleGroupSetSuffix)
	return nil
}
