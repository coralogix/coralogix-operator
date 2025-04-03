/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// DashboardReconciler reconciles a Dashboard object
type DashboardReconciler struct {
	DashboardsClient *cxsdk.DashboardsClient
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
	createRequest := &cxsdk.CreateDashboardRequest{
		Dashboard: dashboardToCreate,
	}
	log.Info("Creating remote dashboard", "dashboard", protojson.Format(createRequest))
	createResponse, err := r.DashboardsClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote dashboard: %w", err)
	}
	log.Info("Remote dashboard created", "dashboard", protojson.Format(createResponse))

	dashboard.Status = coralogixv1alpha1.DashboardStatus{
		ID: pointer.String(createResponse.DashboardId.Value),
	}

	return nil
}

func (r *DashboardReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	dashboard := obj.(*coralogixv1alpha1.Dashboard)
	dashboardToUpdate, err := dashboard.Spec.ExtractDashboardFromSpec(ctx, dashboard.Namespace)
	if err != nil {
		return fmt.Errorf("error on extracting dashboard from spec: %w", err)
	}
	dashboardToUpdate.Id = utils.StringPointerToWrapperspbString(dashboard.Status.ID)
	updateRequest := &cxsdk.ReplaceDashboardRequest{
		Dashboard: dashboardToUpdate,
	}
	log.Info("Updating remote dashboard", "dashboard", protojson.Format(updateRequest))
	updateResponse, err := r.DashboardsClient.Replace(ctx, updateRequest)
	if err != nil {
		return fmt.Errorf("error on updating remote dashboard: %w", err)
	}
	log.Info("Remote dashboard updated", "dashboard", protojson.Format(updateResponse))

	return nil
}

func (r *DashboardReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	dashboard := obj.(*coralogixv1alpha1.Dashboard)
	id := *dashboard.Status.ID
	log.Info("Deleting dashboard from remote system", "id", id)
	_, err := r.DashboardsClient.Delete(ctx, &cxsdk.DeleteDashboardRequest{DashboardId: wrapperspb.String(id)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote dashboard", "id", id)
		return fmt.Errorf("error deleting remote dashboard %s: %w", id, err)
	}
	log.Info("Dashboard deleted from remote system", "id", id)
	return nil
}

func (r *DashboardReconciler) CheckIDInStatus(obj client.Object) bool {
	dashboard := obj.(*coralogixv1alpha1.Dashboard)
	return dashboard.Status.ID != nil && *dashboard.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *DashboardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Dashboard{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
