/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/go-logr/logr"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SLOReconciler reconciles a SLO object
type SLOReconciler struct {
	SLOsClient *cxsdk.SLOsClient
	client.Client
	Scheme   *runtime.Scheme
	Interval time.Duration
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
	extractedSLO, err := slo.Spec.ExtractSLO()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	createRequest := &cxsdk.CreateServiceSloRequest{
		Slo: extractedSLO,
	}
	log.Info("Creating remote slo", "slo", protojson.Format(createRequest))
	createResponse, err := r.SLOsClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote slo: %w", err)
	}
	log.Info("Remote slo created", "response", protojson.Format(createResponse))

	receivedSLO := createResponse.GetSlo()
	slo.Status = coralogixv1alpha1.SLOStatus{
		ID:       receivedSLO.Id,
		Revision: pointer.Int32(receivedSLO.Revision.Revision),
	}

	return nil
}

func (r *SLOReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	slo := obj.(*coralogixv1alpha1.SLO)
	extractedSLO, err := slo.Spec.ExtractSLO()
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	if slo.Status.ID == nil {
		return fmt.Errorf("slo id is nil")
	}
	extractedSLO.Id = slo.Status.ID
	updateRequest := &cxsdk.ReplaceServiceSloRequest{
		Slo: extractedSLO,
	}
	log.Info("Updating remote slo", "slo", protojson.Format(updateRequest))
	updateResponse, err := r.SLOsClient.Update(ctx, updateRequest)
	if err != nil {
		return fmt.Errorf("error on updating remote slo: %w", err)
	}
	log.Info("Remote slo updated", "response", protojson.Format(updateResponse))
	return nil
}

func (r *SLOReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	slo := obj.(*coralogixv1alpha1.SLO)
	if slo.Status.ID == nil {
		return fmt.Errorf("slo id is nil")
	}
	deleteRequest := &cxsdk.DeleteServiceSloRequest{
		Id: *slo.Status.ID,
	}
	log.Info("Deleting remote slo", "slo", protojson.Format(deleteRequest))
	deleteResponse, err := r.SLOsClient.Delete(ctx, deleteRequest)
	if err != nil {
		return fmt.Errorf("error on deleting remote slo: %w", err)
	}
	log.Info("Remote slo deleted", "response", protojson.Format(deleteResponse))

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
