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
	"github.com/coralogix/coralogix-operator/api/coralogix/common"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	alerts "github.com/coralogix/coralogix-operator/internal/controller/clientset/grpc/alerts/v2"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
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
		monitoring.AlertInfoMetric.WithLabelValues(alert.Name, alert.Namespace, getAlertType(alert)).Set(1)
		return ctrl.Result{}, nil
	}

	if !alert.ObjectMeta.DeletionTimestamp.IsZero() {
		err = r.delete(ctx, log, alert)
		if err != nil {
			log.Error(err, "Error on deleting alert")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		monitoring.AlertInfoMetric.DeleteLabelValues(alert.Name, alert.Namespace, getAlertType(alert))
		return ctrl.Result{}, nil
	}

	err = r.update(ctx, log, alert)
	if err != nil {
		log.Error(err, "Error on updating alert")
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}
	monitoring.AlertInfoMetric.WithLabelValues(alert.Name, alert.Namespace, getAlertType(alert)).Set(1)

	return ctrl.Result{}, nil
}

func (r *AlertReconciler) update(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) error {
	alertDefProperties, err := alert.Spec.ExtractAlertProperties(
		&common.ListingAlertsAndWebhooksProperties{
			Clientset: r.CoralogixClientSet,
			Ctx:       ctx,
			Log:       log,
			Client:    r.Client,
			Namespace: alert.Namespace,
		})
	if err != nil {
		return fmt.Errorf("error on extracting alert properties: %w", err)
	}

	alertRequest := &cxsdk.ReplaceAlertDefRequest{
		AlertDefProperties: alertDefProperties,
		Id:                 wrapperspb.String(*alert.Status.ID),
	}

	log.V(1).Info("Updating remote alert", "alert", protojson.Format(alertRequest))
	remoteUpdatedAlert, err := r.CoralogixClientSet.AlertsV3().Replace(ctx, alertRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.V(1).Info("alert not found on remote, recreating it")
			alert.Status = *coralogixv1beta1.NewDefaultAlertStatus()
			if err = r.Status().Update(ctx, alert); err != nil {
				return fmt.Errorf("error on updating alert status 1: %v", err)
			}
			return fmt.Errorf("alert not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating alert: %w", err)
	}
	log.V(1).Info("Remote alert updated", "alert", protojson.Format(remoteUpdatedAlert))
	return nil
}

func (r *AlertReconciler) delete(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) error {
	if err := r.deleteRemoteAlert(ctx, log, alert.Status.ID); err != nil {
		return fmt.Errorf("error on deleting remote alert: %w", err)
	}

	controllerutil.RemoveFinalizer(alert, alertFinalizerName)
	if err := r.Update(ctx, alert); err != nil {
		return fmt.Errorf("error on updating alert: %w", err)
	}

	return nil
}

func (r *AlertReconciler) create(ctx context.Context, log logr.Logger, alert *coralogixv1beta1.Alert) error {
	alertDefProperties, err := alert.Spec.ExtractAlertProperties(
		&common.ListingAlertsAndWebhooksProperties{
			Ctx:       ctx,
			Log:       log,
			Client:    r.Client,
			Clientset: r.CoralogixClientSet,
			Namespace: alert.Namespace,
		},
	)
	if err != nil {
		return fmt.Errorf("error on extracting alert properties: %w", err)
	}

	alertRequest := &cxsdk.CreateAlertDefRequest{
		AlertDefProperties: alertDefProperties,
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

	alert.Status.ID = pointer.String(response.GetAlertDef().GetId().GetValue())
	if err = r.Status().Update(ctx, alert); err != nil {
		if err2 := r.deleteRemoteAlert(ctx, log, alert.Status.ID); err2 != nil {
			return fmt.Errorf("error on deleting remote alert after status update error: %w", err2)
		}
		return fmt.Errorf("error on updating alert status: %v %v", err, alert)
	}

	updated := false
	if alert.Spec.EntityLabels == nil {
		alert.Spec.EntityLabels = make(map[string]string)
	}

	if value, ok := alert.Spec.EntityLabels["managed-by"]; !ok || value == "" {
		alert.Spec.EntityLabels["managed-by"] = "coralogix-operator"
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

func (r *AlertReconciler) deleteRemoteAlert(ctx context.Context, log logr.Logger, alertID *string) error {
	log.V(1).Info("Deleting remote alert", "alert", alertID)
	if _, err := r.CoralogixClientSet.Alerts().DeleteAlert(ctx, &alerts.DeleteAlertByUniqueIdRequest{
		Id: wrapperspb.String(*alertID)}); err != nil && status.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote alert", "alert", alertID)
		return fmt.Errorf("error on deleting alert: %w", err)
	}

	log.V(1).Info("Remote alert deleted", "alert", alertID)
	return nil
}

func getAlertType(alert *coralogixv1beta1.Alert) string {
	if alert.Spec.TypeDefinition.Flow != nil {
		return "flow"
	} else if alert.Spec.TypeDefinition.MetricAnomaly != nil {
		return "metric-anomaly"
	} else if alert.Spec.TypeDefinition.LogsAnomaly != nil {
		return "logs-anomaly"
	} else if alert.Spec.TypeDefinition.LogsImmediate != nil {
		return "logs-immediate"
	} else if alert.Spec.TypeDefinition.LogsNewValue != nil {
		return "logs-new-value"
	} else if alert.Spec.TypeDefinition.LogsRatioThreshold != nil {
		return "logs-ratio-threshold"
	} else if alert.Spec.TypeDefinition.LogsUniqueCount != nil {
		return "logs-unique-count"
	} else if alert.Spec.TypeDefinition.MetricThreshold != nil {
		return "metric-threshold"
	} else if alert.Spec.TypeDefinition.TracingThreshold != nil {
		return "tracing-threshold"
	} else if alert.Spec.TypeDefinition.LogsThreshold != nil {
		return "logs-threshold"
	} else if alert.Spec.TypeDefinition.LogsTimeRelativeThreshold != nil {
		return "logs-time-relative-threshold"
	}
	return "unknown"
}

// SetupWithManager sets up the controller with the Manager.
func (r *AlertReconciler) SetupWithManager(mgr ctrl.Manager) error {
	coralogixv1beta1.ClientSet = r.CoralogixClientSet
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1beta1.Alert{}).
		Complete(r)
}
