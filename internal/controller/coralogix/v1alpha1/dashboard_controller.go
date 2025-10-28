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
	"net/http"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// DashboardReconciler reconciles a Dashboard object
type DashboardReconciler struct {
	DashboardsClient *dashboards.DashboardServiceAPIService
	Interval         time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=dashboards,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=dashboards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=dashboards/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *DashboardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Dashboard{}, r)
}

func (r *DashboardReconciler) FinalizerName() string {
	return "dashboard.coralogix.com/finalizer"
}

func (r *DashboardReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *DashboardReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	dashboard := obj.(*coralogixv1alpha1.Dashboard)
	dashboardToCreate, err := dashboard.Spec.ExtractDashboardFromSpec(ctx, dashboard.Namespace)
	if err != nil {
		return fmt.Errorf("error on extracting dashboard from spec: %w", err)
	}
	createRequest := dashboards.CreateDashboardRequestDataStructure{
		Dashboard: *dashboardToCreate,
	}
	log.Info("Creating remote dashboard", "dashboard", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.DashboardsClient.
		DashboardsServiceCreateDashboard(ctx).
		CreateDashboardRequestDataStructure(createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote dashboard: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote dashboard created", "dashboard", utils.FormatJSON(createResponse))

	dashboard.Status = coralogixv1alpha1.DashboardStatus{
		ID: createResponse.DashboardId,
	}

	return nil
}

func (r *DashboardReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	dashboard := obj.(*coralogixv1alpha1.Dashboard)
	dashboardToUpdate, err := dashboard.Spec.ExtractDashboardFromSpec(ctx, dashboard.Namespace)
	if err != nil {
		return fmt.Errorf("error on extracting dashboard from spec: %w", err)
	}
	updateRequest := dashboards.ReplaceDashboardRequestDataStructure{
		Dashboard: *dashboardToUpdate,
	}
	log.Info("Updating remote dashboard", "dashboard", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.DashboardsClient.
		DashboardsServiceReplaceDashboard(ctx).
		ReplaceDashboardRequestDataStructure(updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote dashboard updated", "dashboard", utils.FormatJSON(updateResponse))

	return nil
}

func (r *DashboardReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	dashboard := obj.(*coralogixv1alpha1.Dashboard)
	id := *dashboard.Status.ID
	log.Info("Deleting dashboard from remote system", "id", id)
	_, httpResp, err := r.DashboardsClient.
		DashboardsServiceDeleteDashboard(ctx, id).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); cxsdk.Code(apiErr) != http.StatusNotFound {
			log.Error(err, "Error deleting remote dashboard", "id", id)
			return fmt.Errorf("error deleting remote dashboard %s: %w", id, err)
		}
	}
	log.Info("Dashboard deleted from remote system", "id", id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DashboardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Dashboard{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
