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

package v1beta1

import (
	"context"
	"fmt"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	utils "github.com/coralogix/coralogix-operator/api"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var alertFinalizerName = "alert.coralogix.com/finalizer"

// AlertReconciler reconciles a Alert object
type AlertReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	CoralogixClientSet clientset.ClientSetInterface
}

// +kubebuilder:rbac:groups=coralogix.com,resources=alerts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=alerts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=alerts/finalizers,verbs=update

func (r *AlertReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		err error
	)
	log := log.FromContext(ctx).WithValues(
		"alert", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	log.V(1).Info("Reconciling Alert")
	alert := coralogixv1beta1.NewAlert()

	if err = r.Client.Get(ctx, req.NamespacedName, alert); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(alert.Status.ID, "") == "" {
		err = r.create(ctx, log, alert)
		if err != nil {
			log.Error(err, "Error on creating alert")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !alert.ObjectMeta.DeletionTimestamp.IsZero() {
		err = r.delete(ctx, log, alert)
		if err != nil {
			log.Error(err, "Error on deleting alert")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err = r.update(ctx, log, alert)
	if err != nil {
		log.Error(err, "Error on updating alert")
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *AlertReconciler) update(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) error {
	alertRequest := &cxsdk.ReplaceAlertDefRequest{
		AlertDefProperties: alert.Spec.ExtractAlertProperties(),
		Id:                 utils.StringPointerToWrapperspbString(alert.Status.ID),
	}

	log.V(1).Info("Updating remote alert", "alert", protojson.Format(alertRequest))
	remoteUpdatedAlert, err := r.CoralogixClientSet.AlertsV3().Replace(ctx, alertRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Info("alert not found on remote, recreating it")
			alert.Status = *coralogixv1beta1.NewDefaultAlertStatus()
			if err = r.Status().Update(ctx, alert); err != nil {
				return fmt.Errorf("error on updating alert status: %w", err)
			}
			return fmt.Errorf("alert not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating alert: %w", err)
	}
	log.V(1).Info("Remote alert updated", "alert", protojson.Format(remoteUpdatedAlert))
	return nil
}

func (r *AlertReconciler) delete(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) error {
	log.V(1).Info("Deleting remote alert", "alert", *alert.Status.ID)
	_, err := r.CoralogixClientSet.AlertsV3().Delete(ctx, &cxsdk.DeleteAlertDefRequest{
		Id: utils.StringPointerToWrapperspbString(alert.Status.ID),
	})
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("error on deleting alert: %w", err)
	}
	log.V(1).Info("Remote alert deleted", "alert", *alert.Status.ID)

	controllerutil.RemoveFinalizer(alert, alertFinalizerName)
	if err = r.Update(ctx, alert); err != nil {
		return fmt.Errorf("error on updating alert: %w", err)
	}

	return nil
}

func (r *AlertReconciler) create(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) error {
	alertRequest := &cxsdk.CreateAlertDefRequest{
		AlertDefProperties: alert.Spec.ExtractAlertProperties(),
	}

	log.V(1).Info("Creating remote alert", "alert", protojson.Format(alertRequest))
	response, err := r.CoralogixClientSet.AlertsV3().Create(ctx, alertRequest)
	if err != nil {
		return fmt.Errorf("error on creating alert: %w", err)
	}
	log.V(1).Info("Remote alert created", "response", protojson.Format(response))

	if err = r.Get(ctx, client.ObjectKeyFromObject(alert), alert); err != nil {
		return fmt.Errorf("error on getting alert: %w", err)
	}

	alert.Status.ID = utils.WrapperspbStringToStringPointer(response.GetAlertDef().GetId())
	if err = r.Status().Update(ctx, alert); err != nil {
		return fmt.Errorf("error on updating alert status: %w", err)
	}

	updated := false
	if alert.Spec.Labels == nil {
		alert.Spec.Labels = make(map[string]string)
	}

	if value, ok := alert.Spec.Labels["managed-by"]; !ok || value == "" {
		alert.Spec.Labels["managed-by"] = "coralogix-operator"
		updated = true
	}

	if !controllerutil.ContainsFinalizer(alert, alertFinalizerName) {
		controllerutil.AddFinalizer(alert, alertFinalizerName)
		updated = true
	}

	if updated {
		if err = r.Client.Update(ctx, alert); err != nil {
			return fmt.Errorf("error on updating alert: %w", err)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1beta1.Alert{}).
		Named("alert-v3").
		Complete(r)
}
