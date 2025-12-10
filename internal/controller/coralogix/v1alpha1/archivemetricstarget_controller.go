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
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
)

// ArchiveMetricsTargetReconciler reconciles a ArchiveMetricsTarget object
type ArchiveMetricsTargetReconciler struct {
	ArchiveMetricsTargetsClient *cxsdk.ArchiveMetricsClient
	Interval                    time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=archivemetricstargets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=archivemetricstargets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=archivemetricstargets/finalizers,verbs=update

var (
	archiveMetricsFinalizerName = "archivemetricstarget.coralogix.com/finalizer"
)

func (r *ArchiveMetricsTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.ArchiveMetricsTarget{}, r)
}

func (r *ArchiveMetricsTargetReconciler) FinalizerName() string {
	return archiveMetricsFinalizerName
}

func (r *ArchiveMetricsTargetReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

// We first configure the tenant and then update because we cannot specify the retention days in the configure request.
func (r *ArchiveMetricsTargetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	archiveMetricsTarget := obj.(*coralogixv1alpha1.ArchiveMetricsTarget)
	configureTenantRequest, err := archiveMetricsTarget.Spec.ExtractConfigureTenantRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create archivemetricstarget request: %w", err)
	}
	log.Info("Creating remote archivemetricstarget", "archivemetricstarget", protojson.Format(configureTenantRequest))
	createResponse, err := r.ArchiveMetricsTargetsClient.ConfigureTenant(ctx, configureTenantRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote archivemetricstarget: %w", err)
	}
	updateRequest, err := archiveMetricsTarget.Spec.ExtractUpdateRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update archivemetricstarget request: %w", err)
	}
	_, err = r.ArchiveMetricsTargetsClient.Update(ctx, updateRequest)
	if err != nil {
		return fmt.Errorf("error on updating remote archivemetricstarget: %w", err)
	}
	log.Info("Remote archivemetricstarget created", "response", protojson.Format(createResponse))

	id := "archiveMetricsTarget"
	archiveMetricsTarget.Status = coralogixv1alpha1.ArchiveMetricsTargetStatus{
		ID: &id,
	}

	return nil
}

func (r *ArchiveMetricsTargetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	archiveMetricsTarget := obj.(*coralogixv1alpha1.ArchiveMetricsTarget)
	updateRequest, err := archiveMetricsTarget.Spec.ExtractUpdateRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update archivemetricstarget request: %w", err)
	}
	log.Info("Updating remote archivemetricstarget", "archivemetricstarget", protojson.Format(updateRequest))
	_, err = r.ArchiveMetricsTargetsClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote archivemetricstarget updated")

	return nil
}

func (r *ArchiveMetricsTargetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	_, err := r.ArchiveMetricsTargetsClient.Disable(ctx)
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deactivating remote archivemetricstarget")
		return fmt.Errorf("error deactivating remote archivemetricstarget %w", err)
	}
	log.Info("archivemetricstarget deactivated in remote system")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArchiveMetricsTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ArchiveMetricsTarget{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
