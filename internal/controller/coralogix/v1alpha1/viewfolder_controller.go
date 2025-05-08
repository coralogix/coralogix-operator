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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// ViewFolderReconciler reconciles a ViewFolder object
type ViewFolderReconciler struct {
	ViewFoldersClient *cxsdk.ViewFoldersClient
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
	createRequest := &cxsdk.CreateViewFolderRequest{
		Name: utils.StringPointerToWrapperspbString(ptr.To(viewFolder.Spec.Name)),
	}

	log.Info("Creating remote ViewFolder", "ViewFolder", protojson.Format(createRequest))
	createResponse, err := r.ViewFoldersClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote ViewFolder: %w", err)
	}
	log.Info("Remote viewFolder created", "response", protojson.Format(createResponse))

	viewFolder.Status = coralogixv1alpha1.ViewFolderStatus{
		ID: utils.WrapperspbStringToStringPointer(createResponse.Folder.GetId()),
	}

	return nil
}

func (r *ViewFolderReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	viewFolder := obj.(*coralogixv1alpha1.ViewFolder)
	replaceRequest := &cxsdk.ReplaceViewFolderRequest{
		Folder: &cxsdk.ViewFolder{
			Id:   utils.StringPointerToWrapperspbString(viewFolder.Status.ID),
			Name: utils.StringPointerToWrapperspbString(ptr.To(viewFolder.Spec.Name)),
		},
	}

	log.Info("Updating remote ViewFolder", "ViewFolder", protojson.Format(replaceRequest))
	updateResponse, err := r.ViewFoldersClient.Replace(ctx, replaceRequest)
	if err != nil {
		return err
	}
	log.Info("Remote ViewFolder updated", "ViewFolder", protojson.Format(updateResponse))

	return nil
}

func (r *ViewFolderReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	viewFolder := obj.(*coralogixv1alpha1.ViewFolder)
	log.Info("Deleting ViewFolder from remote system", "id", *viewFolder.Status.ID)
	_, err := r.ViewFoldersClient.Delete(ctx,
		&cxsdk.DeleteViewFolderRequest{
			Id: utils.StringPointerToWrapperspbString(viewFolder.Status.ID),
		},
	)

	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote ViewFolder", "id", *viewFolder.Status.ID)
		return fmt.Errorf("error deleting remote ViewFolder %s: %w", *viewFolder.Status.ID, err)
	}
	log.Info("ViewFolder deleted from remote system", "id", *viewFolder.Status.ID)
	return nil
}

func (r *ViewFolderReconciler) CheckIDInStatus(obj client.Object) bool {
	viewFolder := obj.(*coralogixv1alpha1.ViewFolder)
	return viewFolder.Status.ID != nil && *viewFolder.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *ViewFolderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ViewFolder{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
