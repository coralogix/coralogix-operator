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

package coralogixreconciler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	oapisdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/v2/api/coralogix"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/monitoring"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// CoralogixReconciler defines the required methods for all Coralogix controllers.
type CoralogixReconciler interface {
	HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error
	FinalizerName() string
	RequeueInterval() time.Duration
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func ReconcileResource(ctx context.Context, req ctrl.Request, obj coralogix.Object, r CoralogixReconciler) (ctrl.Result, error) {
	if err := config.GetClient().Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
	}

	gvk := objToGVK(obj)
	log := log.FromContext(ctx).WithValues(
		"gvk", gvk,
		"name", req.Name,
		"namespace", req.Namespace)
	log = log.V(logVerbosity(obj))

	if !obj.HasIDInStatus() {
		log.Info("Resource ID is missing; handling creation for resource")
		if err := r.HandleCreation(ctx, log, obj); err != nil {
			if oapisdk.IsDeserializationError(err) {
				return ManageErrorWithRequeue(ctx, obj, utils.ReasonDeserializationError, err)
			}
			log.Error(err, "Error handling creation")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteCreationFailed, err)
		}

		if err := config.GetClient().Status().Update(ctx, obj); err != nil {
			log.Error(err, "Error updating status after creation")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}

		if err := AddFinalizer(ctx, log, obj, r); err != nil {
			log.Error(err, "Error adding finalizer")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}

		return ManageSuccessWithRequeue(ctx, obj, r.RequeueInterval())
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		log.Info("Resource is being deleted; handling deletion")
		if err := r.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error deleting from remote")
			if oapisdk.IsDeserializationError(err) {
				return ManageErrorWithRequeue(ctx, obj, utils.ReasonDeserializationError, err)
			}
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteDeletionFailed, err)
		}

		if err := RemoveFinalizer(ctx, log, obj, r); err != nil {
			log.Error(err, "Error removing finalizer")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}

		monitoring.DeleteResourceInfoMetric(
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			obj.GetNamespace(),
		)
		return ctrl.Result{}, nil
	}

	if !config.GetConfig().Selector.Matches(obj.GetLabels(), obj.GetNamespace()) {
		log.Info("Resource doesn't match selector; handling deletion")
		if err := r.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error deleting from remote")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteDeletionFailed, err)
		}

		if err := RemoveFinalizer(ctx, log, obj, r); err != nil {
			log.Error(err, "Error removing finalizer")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}

		if err := removeField(ctx, obj, "status"); err != nil {
			log.Error(err, "Error removing id from status")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}

		monitoring.DeleteResourceInfoMetric(
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			obj.GetNamespace(),
		)
		return ctrl.Result{}, nil
	}

	log.Info("Handling update")
	if err := r.HandleUpdate(ctx, log, obj); err != nil {
		log.Error(err, "Error handling update")
		if cxsdk.Code(err) == codes.NotFound || oapisdk.IsNotFound(err) {
			log.Info("resource not found on remote")
			if err := removeField(ctx, obj, "status", "id"); err != nil {
				log.Error(err, "Error removing id from status")
				return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
			}

			return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteResourceNotFound, fmt.Errorf("%s not found on remote: %w", gvk, err))
		} else if oapisdk.IsDeserializationError(err) {
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonDeserializationError, err)
		}
		return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteUpdateFailed, fmt.Errorf("error on updating %s: %w", gvk, err))
	}

	return ManageSuccessWithRequeue(ctx, obj, r.RequeueInterval())
}

func removeField(ctx context.Context, obj client.Object, fields ...string) error {
	u := &unstructured.Unstructured{}
	if err := config.GetScheme().Convert(obj, u, nil); err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	unstructured.RemoveNestedField(u.Object, fields...)

	return config.GetClient().Status().Update(ctx, u)
}

func AddFinalizer(ctx context.Context, log logr.Logger, obj client.Object, r CoralogixReconciler) error {
	if !controllerutil.ContainsFinalizer(obj, r.FinalizerName()) {
		log.Info("Adding finalizer")
		controllerutil.AddFinalizer(obj, r.FinalizerName())
		if err := config.GetClient().Update(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func RemoveFinalizer(ctx context.Context, log logr.Logger, obj client.Object, r CoralogixReconciler) error {
	log.Info("Removing finalizer")
	controllerutil.RemoveFinalizer(obj, r.FinalizerName())
	return config.GetClient().Update(ctx, obj)
}

func ManageErrorWithRequeue(ctx context.Context, obj coralogix.Object, reason string, err error) (reconcile.Result, error) {
	// in case of update conflict, don't try to update conditions, as it will fail with the same error.
	// instead, requeue the request and without flooding with error logs.
	if errors.IsConflict(err) {
		return reconcile.Result{Requeue: true}, nil
	}

	conditions := obj.GetConditions()

	// in case of deserialization error, don't update the condition again to avoid infinite loop, and exit silently.
	if reason == utils.ReasonDeserializationError &&
		utils.GetReasonForRemoteSyncedCondition(conditions) == utils.ReasonDeserializationError {
		return reconcile.Result{}, nil
	}

	if utils.SetSyncedConditionFalse(&conditions, obj.GetGeneration(), reason, err.Error()) || obj.GetPrintableStatus() != "RemoteUnsynced" {
		obj.SetConditions(conditions)
		obj.SetPrintableStatus("RemoteUnsynced")
		if err := config.GetClient().Status().Update(ctx, obj); err != nil {
			if errors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
		}
	}

	monitoring.SetResourceInfoMetricUnsynced(
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetName(),
		obj.GetNamespace(),
	)

	return reconcile.Result{}, err
}

func ManageSuccessWithRequeue(ctx context.Context, obj coralogix.Object, interval time.Duration) (reconcile.Result, error) {
	conditions := obj.GetConditions()
	if utils.SetSyncedConditionTrue(&conditions, obj.GetGeneration(), utils.ReasonRemoteSyncedSuccessfully) || obj.GetPrintableStatus() != "RemoteSynced" {
		obj.SetConditions(conditions)
		obj.SetPrintableStatus("RemoteSynced")
		if err := config.GetClient().Status().Update(ctx, obj); err != nil {
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}
	}

	monitoring.SetResourceInfoMetricSynced(
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetName(),
		obj.GetNamespace(),
	)

	return reconcile.Result{RequeueAfter: interval}, nil
}

func objToGVK(obj client.Object) string {
	gvks, _, _ := config.GetScheme().ObjectKinds(obj)
	if len(gvks) == 0 {
		return ""
	}
	return gvks[0].String()
}

func logVerbosity(obj runtime.Object) int {
	const defaultVerbosity = 1
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return defaultVerbosity
	}

	val, exists := metaObj.GetAnnotations()[utils.LogVerbosityAnnotationKey]
	if !exists {
		return defaultVerbosity
	}

	verbosity, err := strconv.Atoi(val)
	if err != nil || verbosity < 0 {
		return defaultVerbosity
	}

	return verbosity
}
