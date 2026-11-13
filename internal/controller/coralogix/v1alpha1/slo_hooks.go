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
	"errors"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	slosservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// validate checks the oneOf groups of apmSli that the CRD cannot check. errorConfig and average
// are empty objects, and Kubernetes 1.29 and earlier cannot evaluate a CEL rule that reads such a
// field, so the CRD has no rule for them.
func (r *SLOReconciler) validate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.SLO) error {
	apm := cr.Spec.SliType.ApmSli
	if apm == nil {
		return nil
	}
	if (apm.ErrorConfig == nil) == (apm.LatencyConfig == nil) {
		return errors.New("sliType.apmSli: exactly one of errorConfig or latencyConfig must be set")
	}
	if latency := apm.LatencyConfig; latency != nil && (latency.Quantile == nil) == (latency.Average == nil) {
		return errors.New("sliType.apmSli.latencyConfig: exactly one of quantile or average must be set")
	}
	return nil
}

// afterCreate keeps the revision of the created SLO in status.revision, as the released kind did.
func (r *SLOReconciler) afterCreate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.SLO, resp *slosservice.CreateSloResponse) error { //nolint:unparam // the generated lifecycle fixes the signature
	cr.Status.Revision = ptr.To(ptr.To(resp.Slo.GetRevision()).GetRevision())
	return nil
}

// afterUpdate sets status.revision only when it is missing, after a failed afterCreate. It does not
// refresh it on every Update: a status write starts another reconcile, which replaces again and
// moves the remote revision, so the value would stay one revision behind and cost one more
// Replace per update. The released kind did not refresh it either.
func (r *SLOReconciler) afterUpdate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.SLO, resp *slosservice.ReplaceSloResponse) error { //nolint:unparam // the generated lifecycle fixes the signature
	if cr.Status.Revision == nil {
		cr.Status.Revision = ptr.To(ptr.To(resp.Slo.GetRevision()).GetRevision())
	}
	return nil
}
