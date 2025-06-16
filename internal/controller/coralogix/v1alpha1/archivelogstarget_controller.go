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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// ArchiveLogsTargetReconciler reconciles a ArchiveLogsTarget object
type ArchiveLogsTargetReconciler struct {
	ArchiveLogsTargetsClient *cxsdk.ArchiveLogsClient
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
	log.Info("Creating remote archivelogstarget", "archivelogstarget", protojson.Format(createRequest))
	createResponse, err := r.ArchiveLogsTargetsClient.Update(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote archivelogstarget: %w", err)
	}
	log.Info("Remote archivelogstarget created", "response", protojson.Format(createResponse))

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
	log.Info("Updating remote archivelogstarget", "archivelogstarget", protojson.Format(updateRequest))
	updateResponse, err := r.ArchiveLogsTargetsClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote archivelogstarget updated", "archivelogstarget", protojson.Format(updateResponse))

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
	_, err = r.ArchiveLogsTargetsClient.Update(ctx, deleteTargetRequest)
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deactivating remote archivelogstarget")
		return fmt.Errorf("error deactivating remote archivelogstarget %w", err)
	}
	log.Info("archivelogstarget deactivated in remote system")
	return nil
}

func (r *ArchiveLogsTargetReconciler) CheckIDInStatus(obj client.Object) bool {
	archiveLogsTarget := obj.(*coralogixv1alpha1.ArchiveLogsTarget)
	return archiveLogsTarget.Status.ID != nil && *archiveLogsTarget.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArchiveLogsTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ArchiveLogsTarget{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
