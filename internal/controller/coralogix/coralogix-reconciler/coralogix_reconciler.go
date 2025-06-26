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

	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// CoralogixReconciler defines the required methods for all Coralogix controllers.
type CoralogixReconciler interface {
	HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error
	FinalizerName() string
	CheckIDInStatus(obj client.Object) bool
	RequeueInterval() time.Duration
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func ReconcileResource(ctx context.Context, req ctrl.Request, obj client.Object, r CoralogixReconciler) (ctrl.Result, error) {
	if err := config.GetClient().Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
	}

	gvk := objToGVK(obj)
	log := log.FromContext(ctx).WithValues(
		"gvk", gvk,
		"name", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace)
	log = log.V(logVerbosity(obj))

	if !r.CheckIDInStatus(obj) {
		log.Info("Resource ID is missing; handling creation for resource")
		if err := r.HandleCreation(ctx, log, obj); err != nil {
			log.Error(err, "Error handling creation")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteCreationFailed, err)
		}

		if err := config.GetClient().Status().Update(ctx, obj); err != nil {
			log.Error(err, "Error updating status after creation; handling deletion")
			if err := r.HandleDeletion(ctx, log, obj); err != nil {
				log.Error(err, "Error deleting from remote after status update failure")
				return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteDeletionFailed, err)
			}
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}

		if err := AddFinalizer(ctx, log, obj, r); err != nil {
			log.Error(err, "Error adding finalizer")
			return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
		}

		return ManageSuccessWithRequeue(ctx, obj, r.RequeueInterval(), utils.ReasonRemoteCreatedSuccessfully)
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		log.Info("Resource is being deleted; handling deletion")
		if err := r.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error deleting from remote")
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
			"synced",
		)
		monitoring.DeleteResourceInfoMetric(
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			obj.GetNamespace(),
			"unsynced",
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

		return ctrl.Result{}, nil
	}

	log.Info("Handling update")
	if err := r.HandleUpdate(ctx, log, obj); err != nil {
		log.Error(err, "Error handling update")
		if cxsdk.Code(err) == codes.NotFound {
			log.Info("resource not found on remote")
			if err := removeField(ctx, obj, "status", "id"); err != nil {
				log.Error(err, "Error removing id from status")
				return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
			}

			return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteResourceNotFound, fmt.Errorf("%s not found on remote: %w", gvk, err))
		}
		return ManageErrorWithRequeue(ctx, obj, utils.ReasonRemoteUpdateFailed, fmt.Errorf("error on updating %s: %w", gvk, err))
	}

	return ManageSuccessWithRequeue(ctx, obj, r.RequeueInterval(), utils.ReasonRemoteUpdatedSuccessfully)
}

func removeField(ctx context.Context, obj client.Object, fields ...string) error {
	u := &unstructured.Unstructured{}
	if err := config.GetScheme().Convert(obj, u, nil); err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	unstructured.RemoveNestedField(u.Object, fields...)

	if err := config.GetClient().Status().Update(ctx, u); err != nil {
		return err
	}

	return nil
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
	if err := config.GetClient().Update(ctx, obj); err != nil {
		return err
	}
	return nil
}

func ManageErrorWithRequeue(ctx context.Context, obj client.Object, reason string, err error) (reconcile.Result, error) {
	// in case of update conflict, don't try to update conditions, as it will fail with the same error.
	// instead, requeue the request and without flooding with error logs.
	if errors.IsConflict(err) {
		return reconcile.Result{Requeue: true}, nil
	}

	if conditionsObj, ok := (obj).(utils.ConditionsObj); ok {
		conditions := conditionsObj.GetConditions()
		if utils.SetSyncedConditionFalse(&conditions, obj.GetGeneration(), reason, err.Error()) {
			conditionsObj.SetConditions(conditions)
			if err := config.GetClient().Status().Update(ctx, obj); err != nil {
				if errors.IsConflict(err) {
					return reconcile.Result{Requeue: true}, nil
				}
			}
		}
	}

	monitoring.DeleteResourceInfoMetric(
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetName(),
		obj.GetNamespace(),
		"synced",
	)
	monitoring.SetResourceInfoMetric(
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetName(),
		obj.GetNamespace(),
		"unsynced",
	)

	return reconcile.Result{}, err
}

func ManageSuccessWithRequeue(ctx context.Context, obj client.Object,
	interval time.Duration, reason string) (reconcile.Result, error) {
	if conditionsObj, ok := (obj).(utils.ConditionsObj); ok {
		conditions := conditionsObj.GetConditions()
		if utils.SetSyncedConditionTrue(&conditions, obj.GetGeneration(), reason) {
			conditionsObj.SetConditions(conditions)
			if err := config.GetClient().Status().Update(ctx, obj); err != nil {
				return ManageErrorWithRequeue(ctx, obj, utils.ReasonInternalK8sError, err)
			}
		}
	}

	monitoring.DeleteResourceInfoMetric(
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetName(),
		obj.GetNamespace(),
		"unsynced",
	)
	monitoring.SetResourceInfoMetric(
		obj.GetObjectKind().GroupVersionKind().Kind,
		obj.GetName(),
		obj.GetNamespace(),
		"synced",
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
