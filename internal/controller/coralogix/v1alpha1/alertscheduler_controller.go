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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// AlertSchedulerReconciler reconciles a AlertScheduler object
type AlertSchedulerReconciler struct {
	AlertSchedulerClient *cxsdk.AlertSchedulerClient
	Interval             time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=alertschedulers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=alertschedulers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=alertschedulers/finalizers,verbs=update

func (r *AlertSchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.AlertScheduler{}, r)
}

func (r *AlertSchedulerReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *AlertSchedulerReconciler) FinalizerName() string {
	return "alert-scheduler.coralogix.com/finalizer"
}

func (r *AlertSchedulerReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	alertScheduler := obj.(*coralogixv1alpha1.AlertScheduler)
	createRequest, err := alertScheduler.ExtractCreateAlertSchedulerRequest()

	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.Info("Creating remote AlertScheduler", "AlertScheduler", protojson.Format(createRequest))
	createResponse, err := r.AlertSchedulerClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote AlertScheduler: %w", err)
	}
	log.Info("Remote alertScheduler created", "response", protojson.Format(createResponse))

	alertScheduler.Status = coralogixv1alpha1.AlertSchedulerStatus{
		ID: ptr.To(createResponse.AlertSchedulerRule.GetUniqueIdentifier()),
	}

	return nil
}

func (r *AlertSchedulerReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	alertScheduler := obj.(*coralogixv1alpha1.AlertScheduler)
	updateRequest, err := alertScheduler.ExtractUpdateAlertSchedulerRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote AlertScheduler", "AlertScheduler", protojson.Format(updateRequest))
	updateResponse, err := r.AlertSchedulerClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote AlertScheduler updated", "AlertScheduler", protojson.Format(updateResponse))

	return nil
}

func (r *AlertSchedulerReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	alertScheduler := obj.(*coralogixv1alpha1.AlertScheduler)
	log.Info("Deleting AlertScheduler from remote system", "id", *alertScheduler.Status.ID)
	_, err := r.AlertSchedulerClient.Delete(ctx,
		&cxsdk.DeleteAlertSchedulerRuleRequest{
			AlertSchedulerRuleId: *alertScheduler.Status.ID,
		})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote AlertScheduler", "id", *alertScheduler.Status.ID)
		return fmt.Errorf("error deleting remote AlertScheduler %s: %w", *alertScheduler.Status.ID, err)
	}
	log.Info("AlertScheduler deleted from remote system", "id", *alertScheduler.Status.ID)
	return nil
}

func (r *AlertSchedulerReconciler) CheckIDInStatus(obj client.Object) bool {
	alertScheduler := obj.(*coralogixv1alpha1.AlertScheduler)
	return alertScheduler.Status.ID != nil && *alertScheduler.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertSchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.AlertScheduler{}).
		Complete(r)
}
