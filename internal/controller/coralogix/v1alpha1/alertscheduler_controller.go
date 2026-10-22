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
	alertscheduler "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/alert_scheduler_rule_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// AlertSchedulerReconciler reconciles a AlertScheduler object
type AlertSchedulerReconciler struct {
	AlertSchedulerClient *alertscheduler.AlertSchedulerRuleServiceAPIService
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
	alertSchedulerCRD := obj.(*coralogixv1alpha1.AlertScheduler)
	alertSchedulerRule, err := alertSchedulerCRD.ExtractAlertSchedulerRule()
	if err != nil {
		return fmt.Errorf("error on extracting alert scheduler rule: %w", err)
	}

	createRequest := alertscheduler.CreateAlertSchedulerRuleRequestDataStructure{
		AlertSchedulerRule: *alertSchedulerRule,
	}

	log.Info("Creating remote AlertScheduler", "AlertScheduler", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.AlertSchedulerClient.
		AlertSchedulerRuleServiceCreateAlertSchedulerRule(ctx).
		CreateAlertSchedulerRuleRequestDataStructure(createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote AlertScheduler: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote alertScheduler created", "response", utils.FormatJSON(createResponse))

	alertSchedulerCRD.Status = coralogixv1alpha1.AlertSchedulerStatus{
		ID: ptr.To(createResponse.AlertSchedulerRule.GetUniqueIdentifier()),
	}

	return nil
}

func (r *AlertSchedulerReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	alertSchedulerCRD := obj.(*coralogixv1alpha1.AlertScheduler)
	alertSchedulerRule, err := alertSchedulerCRD.ExtractAlertSchedulerRule()
	if err != nil {
		return fmt.Errorf("error on extracting alert scheduler rule: %w", err)
	}

	alertSchedulerRule.UniqueIdentifier = alertSchedulerCRD.Status.ID

	updateRequest := alertscheduler.UpdateAlertSchedulerRuleRequestDataStructure{
		AlertSchedulerRule: *alertSchedulerRule,
	}

	log.Info("Updating remote AlertScheduler", "AlertScheduler", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.AlertSchedulerClient.
		AlertSchedulerRuleServiceUpdateAlertSchedulerRule(ctx).
		UpdateAlertSchedulerRuleRequestDataStructure(updateRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on updating remote AlertScheduler: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote AlertScheduler updated", "AlertScheduler", utils.FormatJSON(updateResponse))

	return nil
}

func (r *AlertSchedulerReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	alertSchedulerCRD := obj.(*coralogixv1alpha1.AlertScheduler)
	log.Info("Deleting AlertScheduler from remote system", "id", *alertSchedulerCRD.Status.ID)

	_, httpResp, err := r.AlertSchedulerClient.
		AlertSchedulerRuleServiceDeleteAlertSchedulerRule(ctx, ptr.Deref(alertSchedulerCRD.Status.ID, "")).
		Execute()

	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(apiErr, "Error deleting remote AlertScheduler", "id", *alertSchedulerCRD.Status.ID)
			return fmt.Errorf("error deleting remote AlertScheduler %s: %w", *alertSchedulerCRD.Status.ID, apiErr)
		}
	}

	log.Info("AlertScheduler deleted from remote system", "id", *alertSchedulerCRD.Status.ID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertSchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.AlertScheduler{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
