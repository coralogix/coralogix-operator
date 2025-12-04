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

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	archivemetrics "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/metrics_data_archive_service"
	"github.com/coralogix/coralogix-operator/internal/utils"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// ArchiveMetricsTargetReconciler reconciles a ArchiveMetricsTarget object
type ArchiveMetricsTargetReconciler struct {
	ArchiveMetricsTargetsClient *archivemetrics.MetricsDataArchiveServiceAPIService
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
	log.Info("Creating remote archivemetricstarget", "archivemetricstarget", utils.FormatJSON(configureTenantRequest))
	createResponse, httpResp, err := r.ArchiveMetricsTargetsClient.
		MetricsConfiguratorPublicServiceConfigureTenant(ctx).
		MetricsConfiguratorPublicServiceConfigureTenantRequest(*configureTenantRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote archivemetricstarget: %w", cxsdk.NewAPIError(httpResp, err))
	}
	updateRequest, err := archiveMetricsTarget.Spec.ExtractUpdateRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update archivemetricstarget request: %w", err)
	}
	_, httpResp, err = r.ArchiveMetricsTargetsClient.
		MetricsConfiguratorPublicServiceUpdate(ctx).
		MetricsConfiguratorPublicServiceUpdateRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote archivemetricstarget created", "response", utils.FormatJSON(createResponse))

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
	log.Info("Updating remote archivemetricstarget", "archivemetricstarget", utils.FormatJSON(updateRequest))
	_, httpResp, err := r.ArchiveMetricsTargetsClient.
		MetricsConfiguratorPublicServiceUpdate(ctx).
		MetricsConfiguratorPublicServiceUpdateRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote archivemetricstarget updated")

	return nil
}

func (r *ArchiveMetricsTargetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	_, httpResp, err := r.ArchiveMetricsTargetsClient.
		MetricsConfiguratorPublicServiceDisableArchive(ctx).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deactivating remote archivemetricstarget")
			return fmt.Errorf("error deactivating remote archivemetricstarget: %w", err)
		}
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
