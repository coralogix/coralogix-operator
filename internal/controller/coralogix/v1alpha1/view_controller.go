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
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	views "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/views_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ViewReconciler reconciles a View object
type ViewReconciler struct {
	ViewsClient *views.ViewsServiceAPIService
	Interval    time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=views,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=views/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=views/finalizers,verbs=update

func (r *ViewReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.View{}, r)
}

func (r *ViewReconciler) FinalizerName() string {
	return "view.coralogix.com/finalizer"
}

func (r *ViewReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ViewReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	view := obj.(*coralogixv1alpha1.View)
	createRequest, err := view.ExtractCreateRequest(ctx, log)
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.Info("Creating remote view", "view", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.ViewsClient.
		ViewsServiceCreateView(ctx).
		ViewFolder(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote view: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote view created", "response", utils.FormatJSON(createResponse))
	view.Status = coralogixv1alpha1.ViewStatus{
		ID: ptr.To(strconv.Itoa(int(createResponse.Id))),
	}

	return nil
}

func (r *ViewReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	view := obj.(*coralogixv1alpha1.View)
	viewId, err := strconv.Atoi(*view.Status.ID)
	if err != nil {
		return fmt.Errorf("error on converting view id to int: %w", err)
	}

	updateRequest, err := view.ExtractReplaceRequest(ctx, log)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote view", "view", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.ViewsClient.
		ViewsServiceReplaceView(ctx, int32(viewId)).
		View1(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote view updated", "view", utils.FormatJSON(updateResponse))

	return nil
}

func (r *ViewReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	view := obj.(*coralogixv1alpha1.View)
	log.Info("Deleting view from remote system", "id", *view.Status.ID)
	id, err := strconv.Atoi(*view.Status.ID)
	if err != nil {
		return fmt.Errorf("error on converting view id to int: %w", err)
	}

	_, httpResp, err := r.ViewsClient.ViewsServiceDeleteView(ctx, int32(id)).Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote view", "id", *view.Status.ID)
			return fmt.Errorf("error deleting remote view %s: %w", *view.Status.ID, apiErr)
		}
	}
	log.Info("View deleted from remote system", "id", *view.Status.ID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ViewReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.View{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
