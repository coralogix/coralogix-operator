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
	slos "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// SLOReconciler reconciles a SLO object
type SLOReconciler struct {
	SLOsClient *slos.SlosServiceAPIService
	Interval   time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=slos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=slos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=slos/finalizers,verbs=update

var _ coralogixreconciler.CoralogixReconciler = &SLOReconciler{}

func (r *SLOReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.SLO{}, r)
}

func (r *SLOReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	slo := obj.(*coralogixv1alpha1.SLO)
	createRequest, err := slo.ExtractSLOCreateRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}

	log.Info("Creating remote slo", "slo", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.SLOsClient.
		SlosServiceCreateSlo(ctx).
		SlosServiceReplaceSloRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote slo: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote slo created", "response", utils.FormatJSON(createResponse))
	receivedSLO := createResponse.GetSlo()

	var sloID string
	var revision int32
	switch {
	case receivedSLO.SloRequestBasedMetricSli != nil:
		sloID = receivedSLO.SloRequestBasedMetricSli.GetId()
		revision = ptr.To(receivedSLO.SloRequestBasedMetricSli.GetRevision()).GetRevision()
	case receivedSLO.SloWindowBasedMetricSli != nil:
		sloID = receivedSLO.SloWindowBasedMetricSli.GetId()
		revision = ptr.To(receivedSLO.SloWindowBasedMetricSli.GetRevision()).GetRevision()
	default:
		return fmt.Errorf("unknown slo type")
	}

	slo.Status = coralogixv1alpha1.SLOStatus{
		ID:       ptr.To(sloID),
		Revision: ptr.To(revision),
	}

	return nil
}

func (r *SLOReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	slo := obj.(*coralogixv1alpha1.SLO)
	updateRequest, err := slo.ExtractSLOUpdateRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	if slo.Status.ID == nil {
		return fmt.Errorf("slo id is nil")
	}

	log.Info("Updating remote slo", "slo", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.SLOsClient.
		SlosServiceReplaceSlo(ctx).
		SlosServiceReplaceSloRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote slo updated", "response", utils.FormatJSON(updateResponse))
	return nil
}

func (r *SLOReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	slo := obj.(*coralogixv1alpha1.SLO)
	if slo.Status.ID == nil {
		return fmt.Errorf("slo id is nil")
	}

	log.Info("Deleting remote slo", "sloId", *slo.Status.ID)
	deleteResponse, httpResp, err := r.SLOsClient.
		SlosServiceDeleteSlo(ctx, *slo.Status.ID).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			return fmt.Errorf("error on deleting remote slo: %w", err)
		}
	}
	log.Info("Remote slo deleted", "response", utils.FormatJSON(deleteResponse))

	return nil
}

func (r *SLOReconciler) FinalizerName() string {
	return "slo.coralogix.com/finalizer"
}

func (r *SLOReconciler) CheckIDInStatus(obj client.Object) bool {
	slo := obj.(*coralogixv1alpha1.SLO)
	return slo.Status.ID != nil
}

func (r *SLOReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

// SetupWithManager sets up the controller with the Manager.
func (r *SLOReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.SLO{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
