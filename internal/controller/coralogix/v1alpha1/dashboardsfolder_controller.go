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
	"time"

	"github.com/coralogix/coralogix-operator/internal/utils"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	dashboardsfolders "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_folders_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// DashboardsFolderReconciler reconciles a DashboardsFolder object
type DashboardsFolderReconciler struct {
	DashboardsFoldersClient *dashboardsfolders.DashboardFoldersServiceAPIService
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
	folder, ok := obj.(*coralogixv1alpha1.DashboardsFolder)
	if !ok {
		return fmt.Errorf("object is not a DashboardsFolder, but %T", obj)
	}
	folderToCreate, err := folder.Spec.ExtractDashboardsFolderFromSpec(ctx, folder.Namespace)
	if err != nil {
		return fmt.Errorf("error on extracting dashboards-folder from spec: %w", err)
	}

	createRequest := &dashboardsfolders.CreateDashboardFolderRequestDataStructure{
		Folder: folderToCreate,
	}
	log.Info("Creating remote dashboards-folder", "folder", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.DashboardsFoldersClient.
		DashboardFoldersServiceCreateDashboardFolder(ctx).
		CreateDashboardFolderRequestDataStructure(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote dashboard-folder: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote dashboard dashboards-folder created", "folder", utils.FormatJSON(createResponse))

	folder.Status = coralogixv1alpha1.DashboardsFolderStatus{ID: createResponse.FolderId}
	return nil
}

func (r *DashboardsFolderReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	folder := obj.(*coralogixv1alpha1.DashboardsFolder)
	folderToUpdate, err := folder.Spec.ExtractDashboardsFolderFromSpec(ctx, folder.Namespace)
	if err != nil {
		return fmt.Errorf("error on extracting dashboards-folder from spec: %w", err)
	}
	if folder.Status.ID == nil {
		return fmt.Errorf("no ID in status, cannot update remote dashboards-folder")
	}
	folderToUpdate.Id = folder.Status.ID
	updateRequest := &dashboardsfolders.ReplaceDashboardFolderRequestDataStructure{
		Folder: folderToUpdate,
	}
	log.Info("Updating remote dashboards-folder", "folder", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.DashboardsFoldersClient.
		DashboardFoldersServiceReplaceDashboardFolder(ctx).
		ReplaceDashboardFolderRequestDataStructure(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote dashboards-folder updated", "folder", utils.FormatJSON(updateResponse))

	return nil
}

func (r *DashboardsFolderReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	folder := obj.(*coralogixv1alpha1.DashboardsFolder)
	id := folder.Status.ID
	if id == nil {
		log.Info("No ID in status, nothing to delete", "folder", folder)
		return nil
	}
	log.Info("Deleting dashboards-folder from remote system", "id", id)
	_, httpResp, err := r.DashboardsFoldersClient.
		DashboardFoldersServiceDeleteDashboardFolder(context.Background(), *id).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote dashboards-folder", "id", id)
			return fmt.Errorf("error deleting remote dashboards-folder %s: %w", *id, apiErr)
		}
	}
	log.Info("Dashboards-folder deleted from remote", "id", id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DashboardsFolderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.DashboardsFolder{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
