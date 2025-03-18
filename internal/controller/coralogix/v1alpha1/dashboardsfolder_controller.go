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

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

// DashboardsFolderReconciler reconciles a DashboardsFolder object
type DashboardsFolderReconciler struct {
	DashboardsFoldersClient *cxsdk.DashboardsFoldersClient
	Interval                time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=dashboardsfolders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=dashboardsfolders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=dashboardsfolders/finalizers,verbs=update

func (r *DashboardsFolderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.DashboardsFolder{}, r)
}

func (r *DashboardsFolderReconciler) FinalizerName() string {
	return "dashboards-folder.coralogix.com/finalizer"
}

func (r *DashboardsFolderReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *DashboardsFolderReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	folder := obj.(*coralogixv1alpha1.DashboardsFolder)
	folderToCreate := folder.Spec.ExtractDashboardsFolderFromSpec()

	createRequest := &cxsdk.CreateDashboardFolderRequest{
		Folder: folderToCreate,
	}
	log.V(1).Info("Creating remote dashboards-folder", "folder", protojson.Format(createRequest))
	createResponse, err := r.DashboardsFoldersClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote dashboard: %w", err)
	}
	log.V(1).Info("Remote dashboard dashboards-folder", "folder", protojson.Format(createResponse))

	return nil
}

func (r *DashboardsFolderReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	folder := obj.(*coralogixv1alpha1.DashboardsFolder)
	folderToUpdate := folder.Spec.ExtractDashboardsFolderFromSpec()
	updateRequest := &cxsdk.ReplaceDashboardFolderRequest{
		Folder: folderToUpdate,
	}
	log.V(1).Info("Updating remote dashboards-folder", "folder", protojson.Format(updateRequest))
	createResponse, err := r.DashboardsFoldersClient.Replace(ctx, updateRequest)
	if err != nil {
		return fmt.Errorf("error on updating remote dashboard: %w", err)
	}
	log.V(1).Info("Remote dashboards-folder updated", "folder", protojson.Format(createResponse))

	return nil
}

func (r *DashboardsFolderReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	folder := obj.(*coralogixv1alpha1.DashboardsFolder)
	id := folder.Spec.CustomID
	log.V(1).Info("Deleting dashboards-folder from remote system", "id", id)
	_, err := r.DashboardsFoldersClient.Delete(ctx, &cxsdk.DeleteDashboardFolderRequest{FolderId: wrapperspb.String(id)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote dashboards-folder", "id", id)
		return fmt.Errorf("error deleting remote dashboards-folder %s: %w", id, err)
	}
	log.V(1).Info("Dashboards-folder deleted from remote", "id", id)
	return nil
}

func (r *DashboardsFolderReconciler) CheckIDInStatus(obj client.Object) bool {
	folder := obj.(*coralogixv1alpha1.DashboardsFolder)
	return folder.Status.ID != nil && *folder.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *DashboardsFolderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.DashboardsFolder{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
