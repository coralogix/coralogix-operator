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
	"google.golang.org/protobuf/types/known/wrapperspb"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// EnrichmentReconciler reconciles a Enrichment object
type EnrichmentReconciler struct {
	EnrichmentsClient *cxsdk.EnrichmentsClient
	Interval          time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=enrichments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=enrichments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=enrichments/finalizers,verbs=update

var (
	enrichmentFinalizerName = "enrichment.coralogix.com/finalizer"
)

func (r *EnrichmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"enrichment", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	enrichment := &coralogixv1alpha1.Enrichment{}
	if err := config.GetClient().Get(ctx, req.NamespacedName, enrichment); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if enrichment.Status.ID == nil {
		if err, reason := r.create(ctx, log, enrichment); err != nil {
			log.Error(err, "Error on creating Enrichment")
			return coralogixreconciler.ManageErrorWithRequeue(ctx, enrichment, reason, err)
		}
		return coralogixreconciler.ManageSuccessWithRequeue(ctx, enrichment, r.Interval)
	}

	if !enrichment.ObjectMeta.DeletionTimestamp.IsZero() {
		if err, reason := r.delete(ctx, log, enrichment); err != nil {
			log.Error(err, "Error on deleting Enrichment")
			return coralogixreconciler.ManageErrorWithRequeue(ctx, enrichment, reason, err)
		}
		return ctrl.Result{}, nil
	}

	if !config.GetConfig().Selector.Matches(enrichment.GetLabels(), enrichment.GetNamespace()) {
		if err := r.deleteRemoteEnrichment(ctx, log, enrichment.Status.ID); err != nil {
			log.Error(err, "Error on deleting Enrichment")
			return coralogixreconciler.ManageErrorWithRequeue(ctx, enrichment, utils.ReasonRemoteDeletionFailed, err)
		}
		return ctrl.Result{}, nil
	}

	if err, reason := r.update(ctx, log, enrichment); err != nil {
		log.Error(err, "Error on updating Enrichment")
		return coralogixreconciler.ManageErrorWithRequeue(ctx, enrichment, reason, err)
	}

	return coralogixreconciler.ManageSuccessWithRequeue(ctx, enrichment, r.Interval)
}

func (r *EnrichmentReconciler) create(ctx context.Context, log logr.Logger, enrichment *coralogixv1alpha1.Enrichment) (error, string) {
	createRequest, err := enrichment.ExtractCreateEnrichmentRequest(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err), utils.ReasonRemoteCreationFailed
	}
	log.Info("Creating remote enrichment", "enrichment", protojson.Format(createRequest))
	createResponse, err := r.EnrichmentsClient.Add(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote enrichment: %w", err), utils.ReasonRemoteCreationFailed
	}
	log.Info("Remote enrichment created", "response", protojson.Format(createResponse))

	if len(createResponse.Enrichments) == 0 {
		return fmt.Errorf("no enrichment created"), utils.ReasonRemoteCreationFailed
	}

	if len(createResponse.Enrichments) != 1 {
		return fmt.Errorf("multiple or no enrichments created"), utils.ReasonRemoteCreationFailed
	}

	enrichmentId := createResponse.Enrichments[0].Id
	enrichment.Status = coralogixv1alpha1.EnrichmentStatus{
		ID: ptr.To(enrichmentId),
	}

	log.Info("Updating Enrichment status", "id", enrichmentId)
	if err = config.GetClient().Status().Update(ctx, enrichment); err != nil {
		if err := r.deleteRemoteEnrichment(ctx, log, enrichment.Status.ID); err != nil {
			return fmt.Errorf("error to delete enrichment after status update error: %w", err), utils.ReasonRemoteUpdateFailed
		}
		return fmt.Errorf("error to update enrichment status: %w", err), utils.ReasonRemoteUpdateFailed
	}

	if !controllerutil.ContainsFinalizer(enrichment, enrichmentFinalizerName) {
		log.Info("Updating Enrichment to add finalizer", "id", enrichmentId)
		controllerutil.AddFinalizer(enrichment, enrichmentFinalizerName)
		if err = config.GetClient().Update(ctx, enrichment); err != nil {
			return fmt.Errorf("error on updating Enrichment: %w", err), utils.ReasonInternalK8sError
		}
	}

	return nil, ""
}

func (r *EnrichmentReconciler) update(ctx context.Context, log logr.Logger, enrichment *coralogixv1alpha1.Enrichment) (error, string) {
	err, reason := r.delete(ctx, log, enrichment)
	if err != nil {
		return fmt.Errorf("error on deleting remote enrichment for update: %w", err), reason
	}

	err, reason = r.create(ctx, log, enrichment)
	if err != nil {
		return fmt.Errorf("error on creating remote enrichment after delete: %w", err), reason
	}

	return nil, ""
}

func (r *EnrichmentReconciler) delete(ctx context.Context, log logr.Logger, enrichment *coralogixv1alpha1.Enrichment) (error, string) {
	if err := r.deleteRemoteEnrichment(ctx, log, enrichment.Status.ID); err != nil {
		return fmt.Errorf("error on deleting remote enrichment: %w", err), utils.ReasonRemoteDeletionFailed
	}

	log.Info("Removing finalizer from Enrichment")
	controllerutil.RemoveFinalizer(enrichment, enrichmentFinalizerName)
	if err := config.GetClient().Update(ctx, enrichment); err != nil {
		return fmt.Errorf("error on updating Enrichment: %w", err), utils.ReasonInternalK8sError
	}

	return nil, ""
}

func (r *EnrichmentReconciler) deleteRemoteEnrichment(ctx context.Context, log logr.Logger, enrichmentId *uint32) error {
	log.Info("Deleting enrichment from remote", "id", enrichmentId)
	deleteReq := &cxsdk.DeleteEnrichmentsRequest{
		EnrichmentIds: []*wrapperspb.UInt32Value{
			wrapperspb.UInt32(*enrichmentId),
		},
	}

	if err := r.EnrichmentsClient.Delete(ctx, deleteReq); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error on deleting remote enrichment", "id", enrichmentId)
		return fmt.Errorf("error to delete remote enrichment -\n%v", enrichmentId)
	}
	log.Info("enrichment was deleted from remote", "id", enrichmentId)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnrichmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Enrichment{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
