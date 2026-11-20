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
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	targetservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/target_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// The hooks of the generated ArchiveLogsTarget kind keep the behavior of the released kind:
//
//   - validate: the REST API sets only S3 targets, so a CR without s3Target (an IBM COS target)
//     fails before any call.
//   - afterCreate: the released kind activated the archive retentions after it set the target. An
//     inactive target (spec.isActive: false) activates nothing.
//   - afterUpdate: activates them when the previous run did not sync, so a failed afterCreate is
//     repaired. A synced resync makes no extra call.
//   - delete: the API cannot remove the target, so delete sets it inactive.

func (r *ArchiveLogsTargetReconciler) validate(_ context.Context, _ logr.Logger, cr *coralogixv1alpha1.ArchiveLogsTarget) error {
	if cr.Spec.S3Target == nil {
		return errors.New("spec.s3Target is required: the API sets only S3 targets")
	}
	return nil
}

func (r *ArchiveLogsTargetReconciler) afterCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.ArchiveLogsTarget, _ *targetservice.S3TargetServiceSetTargetResponse) error {
	if !active(cr) {
		return nil
	}
	return r.activateRetentions(ctx)
}

func (r *ArchiveLogsTargetReconciler) afterUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.ArchiveLogsTarget, _ *targetservice.S3TargetServiceSetTargetResponse) error {
	// The conditions are those of the previous run: the shared reconciler sets them after this hook.
	if !active(cr) || meta.IsStatusConditionTrue(cr.Status.Conditions, utils.ConditionTypeRemoteSynced) {
		return nil
	}
	return r.activateRetentions(ctx)
}

// active reports whether the spec sets the target active. The CRD defaults isActive to true.
func active(cr *coralogixv1alpha1.ArchiveLogsTarget) bool {
	return cr.Spec.IsActive == nil || *cr.Spec.IsActive
}

// activateRetentions activates the archive retentions of the team. A second activation answers the
// same, so it is safe to repeat.
func (r *ArchiveLogsTargetReconciler) activateRetentions(ctx context.Context) error {
	_, httpResp, err := r.ClientSet.ArchiveRetentions().RetentionsServiceActivateRetentions(ctx).Execute()
	if err != nil {
		return fmt.Errorf("error activating archive retentions: %w", cxsdk.NewAPIError(httpResp, err))
	}
	return nil
}

// delete sets the archive target inactive. It runs on deletion, on a selector mismatch, and again
// after a failure; setting an inactive target inactive again changes nothing.
func (r *ArchiveLogsTargetReconciler) delete(ctx context.Context, log logr.Logger, cr *coralogixv1alpha1.ArchiveLogsTarget) error {
	body, err := r.inactiveTarget(ctx, cr)
	if err != nil {
		return err
	}
	if body == nil {
		log.Info("The team has no S3 archive target to set inactive")
		return nil
	}
	log.Info("Setting the remote ArchiveLogsTarget inactive", "ArchiveLogsTarget", utils.FormatJSON(body))
	_, httpResp, err := r.Client.S3TargetServiceSetTarget(ctx).SetTargetResponse(*body).Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			return fmt.Errorf("error setting the remote ArchiveLogsTarget inactive: %w", apiErr)
		}
	}
	return nil
}

// inactiveTarget returns the request that sets the target inactive: the S3 target of the spec. A CR
// without one (an IBM COS target) sends the active S3 target that the team has, so the CR can be
// deleted; nil when the team has none, or when it is inactive already.
func (r *ArchiveLogsTargetReconciler) inactiveTarget(ctx context.Context, cr *coralogixv1alpha1.ArchiveLogsTarget) (*targetservice.SetTargetResponse, error) {
	if cr.Spec.S3Target != nil {
		body, err := cr.UpdateRequest(ctx)
		if err != nil {
			return nil, fmt.Errorf("error on building the request of ArchiveLogsTarget: %w", err)
		}
		body.IsActive = false
		return body, nil
	}
	got, httpResp, err := r.Client.S3TargetServiceGetTarget(ctx).Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			return nil, fmt.Errorf("error reading the remote ArchiveLogsTarget: %w", apiErr)
		}
		return nil, nil // the team has no target
	}
	// No S3 target, or one that is inactive already: nothing to set.
	if got == nil || got.Target.S3 == nil || got.Target.ArchiveSpec.IsActive == nil || !*got.Target.ArchiveSpec.IsActive {
		return nil, nil
	}
	// The POST replaces the target, so it keeps the format that the team has.
	return &targetservice.SetTargetResponse{IsActive: false, S3: *got.Target.S3, Format: got.Target.ArchiveSpec.Format}, nil
}
