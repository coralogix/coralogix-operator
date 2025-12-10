// Copyright 2025 Coralogix Ltd.
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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	targets "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/target_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// ArchiveLogsTargetReconciler reconciles a ArchiveLogsTarget object
type ArchiveLogsTargetReconciler struct {
	ArchiveLogsTargetsClient *targets.TargetServiceAPIService
	ArchiveRetentionsClient  *cxsdk.ArchiveRetentionsClient
	Interval                 time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=archivelogstargets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=archivelogstargets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=archivelogstargets/finalizers,verbs=update

var (
	archiveLogsFinalizerName = "archivelogstarget.coralogix.com/finalizer"
)

func (r *ArchiveLogsTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.ArchiveLogsTarget{}, r)
}

func (r *ArchiveLogsTargetReconciler) FinalizerName() string {
	return archiveLogsFinalizerName
}

func (r *ArchiveLogsTargetReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ArchiveLogsTargetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	archivelogstarget := obj.(*coralogixv1alpha1.ArchiveLogsTarget)
	createRequest, err := archivelogstarget.Spec.ExtractSetTargetRequest(true)
	if err != nil {
		return fmt.Errorf("error on extracting create archivelogstarget request: %w", err)
	}
	log.Info("Creating remote archivelogstarget", "archivelogstarget", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.ArchiveLogsTargetsClient.
		S3TargetServiceSetTarget(ctx).
		SetTargetResponse(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote archivelogstarget: %w", oapicxsdk.NewAPIError(httpResp, err))
	}
	_, err = r.ArchiveRetentionsClient.Activate(ctx, &cxsdk.ActivateRetentionsRequest{})
	if err != nil {
		return fmt.Errorf("error activating archive retentions: %w", err)
	}
	log.Info("Remote archivelogstarget created", "response", utils.FormatJSON(createResponse))

	id := "archiveLogsTaget"
	archivelogstarget.Status = coralogixv1alpha1.ArchiveLogsTargetStatus{
		ID: &id,
	}

	return nil
}

func (r *ArchiveLogsTargetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	archivelogstarget := obj.(*coralogixv1alpha1.ArchiveLogsTarget)
	updateRequest, err := archivelogstarget.Spec.ExtractSetTargetRequest(true)
	if err != nil {
		return fmt.Errorf("error on extracting update archivelogstarget request: %w", err)
	}
	log.Info("Updating remote archivelogstarget", "archivelogstarget", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.ArchiveLogsTargetsClient.
		S3TargetServiceSetTarget(ctx).
		SetTargetResponse(*updateRequest).
		Execute()
	if err != nil {
		return oapicxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote archivelogstarget updated", "archivelogstarget", utils.FormatJSON(updateResponse))

	return nil
}

func (r *ArchiveLogsTargetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	archivelogstarget := obj.(*coralogixv1alpha1.ArchiveLogsTarget)
	log.Info("Deactivating archivelogstarget in remote system")
	deleteTargetRequest, err := archivelogstarget.Spec.ExtractSetTargetRequest(false)
	if err != nil {
		log.Error(err, "Error extracting delete archivelogstarget request")
		return fmt.Errorf("error extracting delete archivelogstarget request: %w", err)
	}
	_, httpResp, err := r.ArchiveLogsTargetsClient.
		S3TargetServiceSetTarget(ctx).
		SetTargetResponse(*deleteTargetRequest).
		Execute()
	if err != nil {
		if apiErr := oapicxsdk.NewAPIError(httpResp, err); !oapicxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deactivating remote archivelogstarget")
			return fmt.Errorf("error deactivating remote archivelogstarget %w", apiErr)
		}
	}
	log.Info("archivelogstarget deactivated in remote system")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArchiveLogsTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ArchiveLogsTarget{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
