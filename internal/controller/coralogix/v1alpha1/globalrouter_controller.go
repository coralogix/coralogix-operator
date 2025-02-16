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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// GlobalRouterReconciler reconciles a GlobalRouter object
type GlobalRouterReconciler struct {
	client.Client
	NotificationsClient *cxsdk.NotificationsClient
}

// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters/finalizers,verbs=update

var (
	globalRouterFinalizerName = "global-router.coralogix.com/finalizer"
)

func (r *GlobalRouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"globalRouter", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	globalRouter := &coralogixv1alpha1.GlobalRouter{}
	if err := r.Get(ctx, req.NamespacedName, globalRouter); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(globalRouter.Status.ID, "") == "" {
		err := r.create(ctx, log, globalRouter)
		if err != nil {
			log.Error(err, "Error on creating GlobalRouter")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !globalRouter.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, globalRouter)
		if err != nil {
			log.Error(err, "Error on deleting GlobalRouter")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(globalRouter.GetLabels()) {
		err := r.deleteRemoteGlobalRouter(ctx, log, *globalRouter.Status.ID)
		if err != nil {
			log.Error(err, "Error on deleting GlobalRouter")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, globalRouter)
	if err != nil {
		log.Error(err, "Error on updating GlobalRouter")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *GlobalRouterReconciler) create(ctx context.Context, log logr.Logger, globalRouter *coralogixv1alpha1.GlobalRouter) error {
	createRequest, err := globalRouter.ExtractCreateGlobalRouterRequest(&coralogixv1alpha1.ResourceRefProperties{
		Client:    r.Client,
		Namespace: globalRouter.Namespace,
	})
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.V(1).Info("Creating remote global-router", "global-router", protojson.Format(createRequest))
	createResponse, err := r.NotificationsClient.CreateGlobalRouter(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote global-router: %w", err)
	}
	log.V(1).Info("Remote global-router created", "response", protojson.Format(createResponse))

	id := createResponse.GetRouter().GetId()
	globalRouter.Status = coralogixv1alpha1.GlobalRouterStatus{
		ID: &id,
	}

	log.V(1).Info("Updating GlobalRouter status", "id", id)
	if err = r.Status().Update(ctx, globalRouter); err != nil {
		if deleteErr := r.deleteRemoteGlobalRouter(ctx, log, *globalRouter.Status.ID); deleteErr != nil {
			return fmt.Errorf("error to delete global-router after status update error. Update error: %w. Deletion error: %w", err, deleteErr)
		}
		return fmt.Errorf("error to update global-router status: %w", err)
	}

	if !controllerutil.ContainsFinalizer(globalRouter, globalRouterFinalizerName) {
		log.V(1).Info("Updating GlobalRouter to add finalizer", "id", id)
		controllerutil.AddFinalizer(globalRouter, globalRouterFinalizerName)
		if err := r.Update(ctx, globalRouter); err != nil {
			return fmt.Errorf("error on updating GlobalRouter: %w", err)
		}
	}

	return nil
}

func (r *GlobalRouterReconciler) update(ctx context.Context, log logr.Logger, globalRouter *coralogixv1alpha1.GlobalRouter) error {
	updateRequest, err := globalRouter.ExtractUpdateGlobalRouterRequest(&coralogixv1alpha1.ResourceRefProperties{
		Client:    r.Client,
		Namespace: globalRouter.Namespace,
	})
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.V(1).Info("Updating remote global-router", "global-router", protojson.Format(updateRequest))
	updateResponse, err := r.NotificationsClient.ReplaceGlobalRouter(ctx, updateRequest)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("global-router not found on remote, removing id from status")
			globalRouter.Status = coralogixv1alpha1.GlobalRouterStatus{
				ID: ptr.To(""),
			}
			if err = r.Status().Update(ctx, globalRouter); err != nil {
				return fmt.Errorf("error on updating GlobalRouter status: %w", err)
			}
			return fmt.Errorf("global-router not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating global-router: %w", err)
	}
	log.V(1).Info("Remote global-router updated", "global-router", protojson.Format(updateResponse))

	return nil
}

func (r *GlobalRouterReconciler) delete(ctx context.Context, log logr.Logger, globalRouter *coralogixv1alpha1.GlobalRouter) error {
	if err := r.deleteRemoteGlobalRouter(ctx, log, *globalRouter.Status.ID); err != nil {
		return fmt.Errorf("error on deleting remote global-router: %w", err)
	}

	log.V(1).Info("Removing finalizer from GlobalRouter")
	controllerutil.RemoveFinalizer(globalRouter, globalRouterFinalizerName)
	if err := r.Update(ctx, globalRouter); err != nil {
		return fmt.Errorf("error on updating GlobalRouter: %w", err)
	}

	return nil
}

func (r *GlobalRouterReconciler) deleteRemoteGlobalRouter(ctx context.Context, log logr.Logger, globalRouterID string) error {
	log.V(1).Info("Deleting global-router from remote", "id", globalRouterID)

	_, err := r.NotificationsClient.DeleteGlobalRouter(ctx,
		&cxsdk.DeleteGlobalRouterRequest{
			Identifier: &cxsdk.GlobalRouterIdentifier{
				Value: &cxsdk.GlobalRouterIdentifierIDValue{
					Id: globalRouterID,
				},
			},
		},
	)
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote global-router", "id", globalRouterID)
		return fmt.Errorf("error to delete remote global-router %s: %w", globalRouterID, err)
	}
	log.V(1).Info("global-router was deleted from remote", "id", globalRouterID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GlobalRouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.GlobalRouter{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
