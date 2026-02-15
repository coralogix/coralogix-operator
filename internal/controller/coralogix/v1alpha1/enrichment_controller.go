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

func (r *EnrichmentReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	enrichment := obj.(*coralogixv1alpha1.Enrichment)
	createRequest, err := enrichment.ExtractEnrichmentsCreationRequest(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting enrichments creation request: %w", err)
	}
	log.Info("Creating remote enrichment", "enrichment", utils.FormatJSON(createRequest))
	addResponse, httpResp, err := r.EnrichmentsClient.
		EnrichmentServiceAddEnrichments(ctx).
		EnrichmentsCreationRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote enrichment: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote enrichment created", "response", utils.FormatJSON(addResponse))

	if len(addResponse.Enrichments) == 0 {
		return fmt.Errorf("no enrichments created, empty response")
	}

	if len(addResponse.Enrichments) > 1 {
		return fmt.Errorf("unexpected multiple enrichments created, expected 1 but got %d", len(addResponse.Enrichments))
	}

	id := addResponse.Enrichments[0].Id
	log.Info("Updating enrichment status with enrichment ID", "enrichmentId", id)
	enrichment.Status.Id = ptr.To(strconv.Itoa(int(id)))
	return nil
}

// HandleUpdate performs delete-then-recreate because the enrichment API does not support update.
func (r *EnrichmentReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	if err := r.HandleDeletion(ctx, log, obj); err != nil {
		return fmt.Errorf("error on deleting enrichment during update: %w", err)
	}

	if err := r.HandleCreation(ctx, log, obj); err != nil {
		return fmt.Errorf("error on creating enrichment during update: %w", err)
	}

	return nil
}

func (r *EnrichmentReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	enrichment := obj.(*coralogixv1alpha1.Enrichment)
	id, err := strconv.Atoi(*enrichment.Status.Id)
	if err != nil {
		return fmt.Errorf("error converting enrichment ID to int: %w", err)
	}

	log.Info("Deleting enrichments from remote", "ids", id)
	_, httpResp, err := r.EnrichmentsClient.
		EnrichmentServiceRemoveEnrichments(ctx).
		EnrichmentIds([]int64{int64(id)}).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote enrichments", "id", id)
			return fmt.Errorf("delete remote enrichments: %w", apiErr)
		}
	}
	log.Info("Enrichments deleted from remote", "id", id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnrichmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Enrichment{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
