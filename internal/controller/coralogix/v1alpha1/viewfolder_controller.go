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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	viewfolders "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/folders_for_views_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ViewFolderReconciler reconciles a ViewFolder object
type ViewFolderReconciler struct {
	ViewFoldersClient *viewfolders.FoldersForViewsServiceAPIService
	Interval          time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=viewfolders,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=viewfolders/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=viewfolders/finalizers,verbs=update

func (r *ViewFolderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.ViewFolder{}, r)
}

func (r *ViewFolderReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ViewFolderReconciler) FinalizerName() string {
	return "view-folder.coralogix.com/finalizer"
}

func (r *ViewFolderReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	viewFolder := obj.(*coralogixv1alpha1.ViewFolder)
	createRequest := &viewfolders.CreateViewFolderRequest{
		Name: viewfolders.PtrString(viewFolder.Spec.Name),
	}

	log.Info("Creating remote ViewFolder", "ViewFolder", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.
		ViewFoldersClient.
		ViewsFoldersServiceCreateViewFolder(ctx).
		CreateViewFolderRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote ViewFolder: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote viewFolder created", "response", utils.FormatJSON(createResponse))

	viewFolder.Status = coralogixv1alpha1.ViewFolderStatus{
		ID: createResponse.Id,
	}

	return nil
}

func (r *ViewFolderReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	viewFolder := obj.(*coralogixv1alpha1.ViewFolder)
	replaceRequest := &viewfolders.ViewFolder1{
		Id:   viewFolder.Status.ID,
		Name: viewFolder.Spec.Name,
	}

	log.Info("Updating remote ViewFolder", "ViewFolder", utils.FormatJSON(replaceRequest))
	updateResponse, httpResp, err := r.ViewFoldersClient.
		ViewsFoldersServiceReplaceViewFolder(ctx).
		ViewFolder1(*replaceRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote ViewFolder updated", "ViewFolder", utils.FormatJSON(updateResponse))

	return nil
}

func (r *ViewFolderReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	viewFolder := obj.(*coralogixv1alpha1.ViewFolder)
	log.Info("Deleting ViewFolder from remote system", "id", *viewFolder.Status.ID)
	_, httpResp, err := r.ViewFoldersClient.ViewsFoldersServiceDeleteViewFolder(ctx, *viewFolder.Status.ID).
		Execute()

	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote ViewFolder", "id", *viewFolder.Status.ID)
			return fmt.Errorf("error deleting remote ViewFolder %s: %w", *viewFolder.Status.ID, apiErr)
		}
	}
	log.Info("ViewFolder deleted from remote system", "id", *viewFolder.Status.ID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ViewFolderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ViewFolder{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
