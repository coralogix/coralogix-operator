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
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	globalrouters "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/global_routers_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// GlobalRouterReconciler reconciles a GlobalRouter object
type GlobalRouterReconciler struct {
	GlobalRoutersClient *globalrouters.GlobalRoutersServiceAPIService
	Interval            time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=globalrouters/finalizers,verbs=update

func (r *GlobalRouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.GlobalRouter{}, r)
}

func (r *GlobalRouterReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *GlobalRouterReconciler) FinalizerName() string {
	return "global-router.coralogix.com/finalizer"
}

func (r *GlobalRouterReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	globalRouter := obj.(*coralogixv1alpha1.GlobalRouter)
	router, err := globalRouter.ExtractGlobalRouter(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}

	createRequest := &globalrouters.CreateGlobalRouterRequest{
		Router: router,
	}

	log.Info("Creating remote GlobalRouter", "GlobalRouter", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.GlobalRoutersClient.
		GlobalRoutersServiceCreateGlobalRouter(ctx).
		CreateGlobalRouterRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote GlobalRouter: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote globalRouter created", "response", utils.FormatJSON(createResponse))

	globalRouter.Status = coralogixv1alpha1.GlobalRouterStatus{
		Id: createResponse.Router.Id,
	}

	return nil
}

func (r *GlobalRouterReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	globalRouter := obj.(*coralogixv1alpha1.GlobalRouter)
	router, err := globalRouter.ExtractGlobalRouter(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}

	router.Id = globalRouter.Status.Id
	replaceRequest := &globalrouters.ReplaceGlobalRouterRequest{
		Router: router,
	}

	log.Info("Updating remote GlobalRouter", "GlobalRouter", utils.FormatJSON(replaceRequest))
	updateResponse, httpResp, err := r.GlobalRoutersClient.
		GlobalRoutersServiceReplaceGlobalRouter(ctx).
		ReplaceGlobalRouterRequest(*replaceRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote GlobalRouter updated", "GlobalRouter", utils.FormatJSON(updateResponse))

	return nil
}

func (r *GlobalRouterReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	globalRouter := obj.(*coralogixv1alpha1.GlobalRouter)
	log.Info("Deleting GlobalRouter from remote system", "id", *globalRouter.Status.Id)
	_, httpResp, err := r.GlobalRoutersClient.
		GlobalRoutersServiceDeleteGlobalRouter(ctx, ptr.Deref(globalRouter.Status.Id, "")).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote GlobalRouter", "id", *globalRouter.Status.Id)
			return fmt.Errorf("error deleting remote GlobalRouter %s: %w",
				*globalRouter.Status.Id, apiErr)
		}
	}
	log.Info("GlobalRouter deleted from remote system", "id", *globalRouter.Status.Id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GlobalRouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.GlobalRouter{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
