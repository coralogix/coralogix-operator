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
	customenrichments "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/custom_enrichments_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// CustomEnrichmentReconciler reconciles a CustomEnrichment object
type CustomEnrichmentReconciler struct {
	CustomEnrichmentsClient *customenrichments.CustomEnrichmentsServiceAPIService
	Interval                time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=customenrichments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=customenrichments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=customenrichments/finalizers,verbs=update

func (r *CustomEnrichmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.CustomEnrichment{}, r)
}

func (r *CustomEnrichmentReconciler) FinalizerName() string {
	return "custom-enrichment.coralogix.com/finalizer"
}

func (r *CustomEnrichmentReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *CustomEnrichmentReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	customEnrichment := obj.(*coralogixv1alpha1.CustomEnrichment)
	createRequest, err := customEnrichment.ExtractCreateCustomEnrichmentRequest(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting create request from customEnrichment spec: %w", err)
	}

	log.Info("Creating remote customEnrichment", "customEnrichment", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.CustomEnrichmentsClient.
		CustomEnrichmentServiceCreateCustomEnrichment(ctx).
		CreateCustomEnrichmentRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote customEnrichment: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote customEnrichment created", "response", utils.FormatJSON(createResponse))

	customEnrichment.Status = coralogixv1alpha1.CustomEnrichmentStatus{
		Id: ptr.To(strconv.Itoa(int(*createResponse.CustomEnrichment.Id))),
	}

	return nil
}

func (r *CustomEnrichmentReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	customEnrichment := obj.(*coralogixv1alpha1.CustomEnrichment)
	updateRequest, err := customEnrichment.ExtractUpdateCustomEnrichmentRequest(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting update request from customEnrichment spec: %w", err)
	}

	log.Info("Updating remote customEnrichment", "customEnrichment", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.CustomEnrichmentsClient.
		CustomEnrichmentServiceUpdateCustomEnrichment(ctx).
		UpdateCustomEnrichmentRequest(*updateRequest).
		Execute()

	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote customEnrichment updated", "customEnrichment", utils.FormatJSON(updateResponse))

	return nil
}

func (r *CustomEnrichmentReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	customEnrichment := obj.(*coralogixv1alpha1.CustomEnrichment)
	log.Info("Deleting customEnrichment from remote system", "id", *customEnrichment.Status.Id)
	id, err := strconv.Atoi(*customEnrichment.Status.Id)
	if err != nil {
		return fmt.Errorf("error on converting custom-enrichment id to int: %w", err)
	}

	_, httpResp, err := r.CustomEnrichmentsClient.CustomEnrichmentServiceDeleteCustomEnrichment(ctx, int64(id)).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote customEnrichment", "id", *customEnrichment.Status.Id)
			return fmt.Errorf("error deleting remote customEnrichment %s: %w", *customEnrichment.Status.Id, apiErr)
		}
	}
	log.Info("CustomEnrichment deleted from remote system", "id", *customEnrichment.Status.Id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomEnrichmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.CustomEnrichment{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
