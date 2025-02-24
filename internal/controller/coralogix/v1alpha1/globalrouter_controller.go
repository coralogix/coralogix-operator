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

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixreconcile "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// GlobalRouterReconciler reconciles a GlobalRouter object
type GlobalRouterReconciler struct {
	NotificationsClient *cxsdk.NotificationsClient
}

// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters/finalizers,verbs=update

func (r *GlobalRouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconcile.ReconcileResource(ctx, req, &coralogixv1alpha1.GlobalRouter{}, r)
}

func (r *GlobalRouterReconciler) FinalizerName() string {
	return "global-router.coralogix.com/finalizer"
}

func (r *GlobalRouterReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error) {
	globalRouter := obj.(*coralogixv1alpha1.GlobalRouter)
	createRequest, err := globalRouter.ExtractCreateGlobalRouterRequest(&coralogixv1alpha1.ResourceRefProperties{
		Namespace: globalRouter.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("error on extracting create request: %w", err)
	}
	log.V(1).Info("Creating remote globalRouter", "globalRouter", protojson.Format(createRequest))
	createResponse, err := r.NotificationsClient.CreateGlobalRouter(ctx, createRequest)
	if err != nil {
		return nil, fmt.Errorf("error on creating remote globalRouter: %w", err)
	}
	log.V(1).Info("Remote globalRouter created", "response", protojson.Format(createResponse))

	globalRouter.Status = coralogixv1alpha1.GlobalRouterStatus{
		ID: createResponse.GetRouter().Id,
	}

	return globalRouter, nil
}

func (r *GlobalRouterReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	globalRouter := obj.(*coralogixv1alpha1.GlobalRouter)
	updateRequest, err := globalRouter.ExtractUpdateGlobalRouterRequest(&coralogixv1alpha1.ResourceRefProperties{
		Namespace: globalRouter.Namespace,
	})
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.V(1).Info("Updating remote globalRouter", "globalRouter", protojson.Format(updateRequest))
	updateResponse, err := r.NotificationsClient.ReplaceGlobalRouter(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.V(1).Info("Remote globalRouter updated", "globalRouter", protojson.Format(updateResponse))

	return nil
}

func (r *GlobalRouterReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	globalRouter := obj.(*coralogixv1alpha1.GlobalRouter)
	id := *globalRouter.Status.ID
	log.V(1).Info("Deleting globalRouter from remote system", "id", id)
	_, err := r.NotificationsClient.DeleteGlobalRouter(ctx, &cxsdk.DeleteGlobalRouterRequest{
		Identifier: &cxsdk.GlobalRouterIdentifier{
			Value: &cxsdk.GlobalRouterIdentifierIDValue{
				Id: id,
			},
		},
	})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote globalRouter", "id", id)
		return fmt.Errorf("error deleting remote globalRouter %s: %w", id, err)
	}
	log.V(1).Info("GlobalRouter deleted from remote system", "id", id)
	return nil
}

func (r *GlobalRouterReconciler) CheckIDInStatus(obj client.Object) bool {
	globalRouter := obj.(*coralogixv1alpha1.GlobalRouter)
	return globalRouter.Status.ID != nil && *globalRouter.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *GlobalRouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.GlobalRouter{}).
		WithEventFilter(utils.GetSelector().Predicate()).
		Complete(r)
}
