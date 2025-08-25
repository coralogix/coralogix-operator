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

package v1beta1

import (
	"context"
	"fmt"
	"time"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// AlertReconciler reconciles a Alert object
type AlertReconciler struct {
	CoralogixClientSet *cxsdk.ClientSet
	Interval           time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=alerts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=alerts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=alerts/finalizers,verbs=update

func (r *AlertReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1beta1.Alert{}, r)
}

func (r *AlertReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *AlertReconciler) FinalizerName() string {
	return "alert.coralogix.com/finalizer"
}

func (r *AlertReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	alert := obj.(*coralogixv1beta1.Alert)
	alertDefProperties, err := alert.Spec.ExtractAlertProperties(
		&coralogixv1beta1.GetResourceRefProperties{
			Ctx:       ctx,
			Log:       log,
			Clientset: r.CoralogixClientSet,
			Namespace: alert.Namespace,
		},
	)
	if err != nil {
		return fmt.Errorf("error on extracting alert properties: %w", err)
	}

	createRequest := &cxsdk.CreateAlertDefRequest{AlertDefProperties: alertDefProperties}
	log.Info("Creating remote alert", "alert", protojson.Format(createRequest))
	createResponse, err := r.CoralogixClientSet.Alerts().Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote alert: %w", err)
	}
	log.Info("Remote alert created", "response", protojson.Format(createResponse))
	alert.Status = coralogixv1beta1.AlertStatus{ID: &createResponse.AlertDef.Id.Value}
	return nil
}

func (r *AlertReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	alert := obj.(*coralogixv1beta1.Alert)
	alertDefProperties, err := alert.Spec.ExtractAlertProperties(
		&coralogixv1beta1.GetResourceRefProperties{
			Ctx:       ctx,
			Log:       log,
			Clientset: r.CoralogixClientSet,
			Namespace: alert.Namespace,
		},
	)
	if err != nil {
		return fmt.Errorf("error on extracting alert properties: %w", err)
	}

	if alert.Status.ID == nil {
		return fmt.Errorf("alert ID is missing")
	}
	updateRequest := &cxsdk.ReplaceAlertDefRequest{
		Id:                 wrapperspb.String(*alert.Status.ID),
		AlertDefProperties: alertDefProperties,
	}
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote alert", "alert", protojson.Format(updateRequest))
	updateResponse, err := r.CoralogixClientSet.Alerts().Replace(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote alert updated", "alert", protojson.Format(updateResponse))
	return nil
}

func (r *AlertReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	alert := obj.(*coralogixv1beta1.Alert)
	if alert.Status.ID == nil {
		return fmt.Errorf("alert ID is missing")
	}
	log.Info("Deleting alert from remote system", "id", *alert.Status.ID)
	_, err := r.CoralogixClientSet.Alerts().Delete(ctx, &cxsdk.DeleteAlertDefRequest{Id: wrapperspb.String(*alert.Status.ID)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote alert", "id", *alert.Status.ID)
		return fmt.Errorf("error deleting remote alert %s: %w", *alert.Status.ID, err)
	}
	log.Info("Alert deleted from remote system", "id", *alert.Status.ID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1beta1.Alert{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
