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
	enrichments "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/enrichments_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// EnrichmentReconciler reconciles an Enrichment object.
type EnrichmentReconciler struct {
	EnrichmentsClient *enrichments.EnrichmentsServiceAPIService
	Interval          time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=enrichments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=enrichments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=enrichments/finalizers,verbs=update
// +kubebuilder:rbac:groups=coralogix.com,resources=customenrichments,verbs=get;list

func (r *EnrichmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Enrichment{}, r)
}

func (r *EnrichmentReconciler) FinalizerName() string {
	return "enrichment.coralogix.com/finalizer"
}

func (r *EnrichmentReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *EnrichmentReconciler) Overwrite(ctx context.Context, log logr.Logger, enr *coralogixv1alpha1.Enrichment) error {
	overwriteRequest, err := enr.ExtractAtomicOverwriteRequest(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting enrichments creation request: %w", err)
	}
	log.Info("Overwriting remote enrichments", "enrichment", utils.FormatJSON(overwriteRequest))
	overwriteResponse, httpResp, err := r.EnrichmentsClient.
		EnrichmentServiceAtomicOverwriteEnrichments(ctx).
		EnrichmentServiceAtomicOverwriteEnrichmentsRequest(*overwriteRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on overwriting remote enrichments: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote enrichments overwritten", "response", utils.FormatJSON(overwriteResponse))
	return nil
}

func (r *EnrichmentReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	enrichment := obj.(*coralogixv1alpha1.Enrichment)
	if err := r.Overwrite(ctx, log, enrichment); err != nil {
		return err
	}

	return coralogixreconciler.AddFinalizer(ctx, log, enrichment, r)
}

func (r *EnrichmentReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	enrichment := obj.(*coralogixv1alpha1.Enrichment)
	if err := r.Overwrite(ctx, log, enrichment); err != nil {
		return err
	}

	return coralogixreconciler.AddFinalizer(ctx, log, enrichment, r)
}

func (r *EnrichmentReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	log.Info("Deleting remote enrichments")
	overwriteRequest := enrichments.EnrichmentServiceAtomicOverwriteEnrichmentsRequest{
		RequestEnrichments: []enrichments.EnrichmentRequestModel{},
	}
	_, httpResp, err := r.EnrichmentsClient.
		EnrichmentServiceAtomicOverwriteEnrichments(ctx).
		EnrichmentServiceAtomicOverwriteEnrichmentsRequest(overwriteRequest).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(apiErr, "Received an error while Deleting remote enrichments")
			return apiErr
		}
	}

	log.Info("Remote enrichments deleted")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnrichmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Enrichment{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
