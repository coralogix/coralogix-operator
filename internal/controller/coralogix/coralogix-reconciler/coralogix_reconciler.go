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
	stderrors "errors"
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

// ErrRemoteResourceNotFound marks a remote 404 after its response body was discarded.
var ErrRemoteResourceNotFound = stderrors.New("remote resource not found")

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

func ReconcileResource(ctx context.Context, req ctrl.Request, obj coralogix.Object, r CoralogixReconciler) (ctrl.Result, error) {
	if err := config.GetClient().Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ManageErrorWithRequeue(ctx, obj, "get", utils.ReasonInternalK8sError, err)
	}

	gvk := objToGVK(obj)
	log := log.FromContext(ctx).WithValues(
		"gvk", gvk,
		"name", req.Name,
		"namespace", req.Namespace)
	log = log.V(logVerbosity(obj))

	// Deletion comes before creation. A CR that is being deleted and has no status id names no
	// remote resource: a create failed, or an update found the resource gone and removed the id.
	// Creating one now would only make a resource for the next reconcile to delete, or fail forever
	// on a name conflict and keep the finalizer. Only a kind whose remote resource does not depend on
	// the id, such as a singleton, still deletes it.
	if !obj.GetDeletionTimestamp().IsZero() {
		if !obj.HasIDInStatus() && !deletesWithoutStatusID(r) {
			log.Info("Resource is being deleted and has no ID; removing finalizer without remote deletion")
			if controllerutil.ContainsFinalizer(obj, r.FinalizerName()) {
				if err := RemoveFinalizer(ctx, log, obj, r); err != nil {
					return ManageErrorWithRequeue(ctx, obj, "remove-finalizer", utils.ReasonInternalK8sError, err)
				}
			}
			monitoring.DeleteResourceInfoMetric(
				obj.GetObjectKind().GroupVersionKind().Kind,
				obj.GetName(),
				obj.GetNamespace(),
			)
			return ctrl.Result{}, nil
		}

		log.Info("Resource is being deleted; handling deletion")
		if err := r.HandleDeletion(ctx, log, obj); err != nil {
			if reason, ok := hookReason(err); ok {
				return ManageErrorWithRequeue(ctx, obj, "delete", reason, err)
			}
			if oapisdk.IsDeserializationError(err) {
				return ManageErrorWithRequeue(ctx, obj, "delete", utils.ReasonDeserializationError, err)
			}
			return ManageErrorWithRequeue(ctx, obj, "delete", utils.ReasonRemoteDeletionFailed, err)
		}

		if err := RemoveFinalizer(ctx, log, obj, r); err != nil {
			return ManageErrorWithRequeue(ctx, obj, "remove-finalizer", utils.ReasonInternalK8sError, err)
		}

		monitoring.DeleteResourceInfoMetric(
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			obj.GetNamespace(),
		)
		return ctrl.Result{}, nil
	}

	if !obj.HasIDInStatus() {
		log.Info("Resource ID is missing; handling creation for resource")
		followUp, err := handleCreation(ctx, log, obj, r)
		if err != nil {
			if reason, ok := hookReason(err); ok {
				return ManageErrorWithRequeue(ctx, obj, "create", reason, err)
			}
			if oapisdk.IsDeserializationError(err) {
				return ManageErrorWithRequeue(ctx, obj, "create", utils.ReasonDeserializationError, err)
			}
			return ManageErrorWithRequeue(ctx, obj, "create", utils.ReasonRemoteCreationFailed, err)
		}

		if err := config.GetClient().Status().Update(ctx, obj); err != nil {
			return ManageErrorWithRequeue(ctx, obj, "update-status", utils.ReasonInternalK8sError, err)
		}

		if err := AddFinalizer(ctx, log, obj, r); err != nil {
			return ManageErrorWithRequeue(ctx, obj, "add-finalizer", utils.ReasonInternalK8sError, err)
		}

		if followUp != nil {
			// status.id and the finalizer are saved. A failure here keeps status.id, so the next run
			// takes the Update path and does not create a second remote resource.
			if reason, err := runFollowUp(ctx, obj, r, followUp); err != nil {
				return ManageErrorWithRequeue(ctx, obj, "create-follow-up", reason, err)
			}
		}

		return ManageSuccessWithRequeue(ctx, obj, r.RequeueInterval())
	}

	if !config.GetConfig().Selector.Matches(obj.GetLabels(), obj.GetNamespace()) {
		log.Info("Resource doesn't match selector; handling deletion")
		if err := r.HandleDeletion(ctx, log, obj); err != nil {
			return ManageErrorWithRequeue(ctx, obj, "delete", reasonOrDefault(err, utils.ReasonRemoteDeletionFailed), err)
		}

		if err := RemoveFinalizer(ctx, log, obj, r); err != nil {
			return ManageErrorWithRequeue(ctx, obj, "remove-finalizer", utils.ReasonInternalK8sError, err)
		}

		if err := clearRemoteStatus(ctx, obj, r); err != nil {
			return ManageErrorWithRequeue(ctx, obj, "clear-status", utils.ReasonInternalK8sError, err)
		}

		monitoring.DeleteResourceInfoMetric(
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			obj.GetNamespace(),
		)
		return ctrl.Result{}, nil
	}

	log.Info("Handling update")
	watch, err := watchStatus(obj, r)
	if err != nil {
		return ManageErrorWithRequeue(ctx, obj, "watch-status", utils.ReasonInternalK8sError, err)
	}
	if err := r.HandleUpdate(ctx, log, obj); err != nil {
		// A hook error comes first. It is never a remote NotFound: a hook can call another API and
		// get a 404 while the resource itself is still there.
		if reason, ok := hookReason(err); ok {
			return ManageErrorWithRequeue(ctx, obj, "update", reason, fmt.Errorf("error on updating %s: %w", gvk, err))
		}
		if stderrors.Is(err, ErrRemoteResourceNotFound) || cxsdk.Code(err) == codes.NotFound || oapisdk.IsNotFound(err) {
			log.Info("resource not found on remote")
			if err := removeField(ctx, obj, "status", "id"); err != nil {
				return ManageErrorWithRequeue(ctx, obj, "clear-status", utils.ReasonInternalK8sError, err)
			}

			return ManageErrorWithRequeue(ctx, obj, "update", utils.ReasonRemoteResourceNotFound, fmt.Errorf("%s not found on remote: %w", gvk, err))
		} else if oapisdk.IsDeserializationError(err) {
			return ManageErrorWithRequeue(ctx, obj, "update", utils.ReasonDeserializationError, err)
		}
		return ManageErrorWithRequeue(ctx, obj, "update", utils.ReasonRemoteUpdateFailed, fmt.Errorf("error on updating %s: %w", gvk, err))
	}
	if err := saveStatusIfChanged(ctx, obj, watch); err != nil {
		return ManageErrorWithRequeue(ctx, obj, "update-status", utils.ReasonInternalK8sError, err)
	}

	return ManageSuccessWithRequeue(ctx, obj, r.RequeueInterval())
}

// runFollowUp runs the step that comes after a Create that saved status.id, and saves the status
// that the step changed. On error it returns the reason of the RemoteSynced condition.
func runFollowUp(ctx context.Context, obj client.Object, r CoralogixReconciler, followUp func(context.Context) error) (string, error) {
	watch, err := watchStatus(obj, r)
	if err != nil {
		return utils.ReasonInternalK8sError, err
	}
	if err := followUp(ctx); err != nil {
		return reasonOrDefault(err, utils.ReasonHookFailed), err
	}
	if err := saveStatusIfChanged(ctx, obj, watch); err != nil {
		return utils.ReasonInternalK8sError, err
	}
	return "", nil
}

// saveStatusIfChanged saves the status when a hook changed it since the watch began.
func saveStatusIfChanged(ctx context.Context, obj client.Object, watch statusWatch) error {
	changed, err := watch.changed(obj)
	if err != nil || !changed {
		return err
	}
	return config.GetClient().Status().Update(ctx, obj)
}

func removeField(ctx context.Context, obj client.Object, fields ...string) error {
	u := &unstructured.Unstructured{}
	if err := config.GetScheme().Convert(obj, u, nil); err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	unstructured.RemoveNestedField(u.Object, fields...)

	return config.GetClient().Status().Update(ctx, u)
}

func removeMultipleFields(ctx context.Context, obj client.Object, fieldsList [][]string) error {
	u := &unstructured.Unstructured{}
	if err := config.GetScheme().Convert(obj, u, nil); err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	for _, fields := range fieldsList {
		unstructured.RemoveNestedField(u.Object, fields...)
	}

	return config.GetClient().Status().Update(ctx, u)
}

func clearRemoteStatus(ctx context.Context, obj client.Object, r CoralogixReconciler) error {
	return removeMultipleFields(ctx, obj, remoteStatusFields(r))
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

func ManageErrorWithRequeue(ctx context.Context, obj coralogix.Object, operation, reason string, err error) (reconcile.Result, error) {
	// in case of update conflict, don't try to update conditions, as it will fail with the same error.
	// instead, requeue the request and without flooding with error logs.
	if errors.IsConflict(err) {
		return reconcile.Result{Requeue: true}, nil
	}

	err = SafeReconcileError(operation, reason, err)

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

// SafeReconcileError keeps fixed diagnostic labels and drops raw error text.
func SafeReconcileError(operation, reason string, err error) error {
	code := oapisdk.Code(err)
	category := "unknown"
	var k8sErr errors.APIStatus
	var apiErr *oapisdk.APIError
	switch {
	case reason == utils.ReasonValidationFailed:
		category = "validation"
	case reason == utils.ReasonDeserializationError:
		category = "deserialization"
	case reason == utils.ReasonHookFailed:
		category = "hook"
	case reason == utils.ReasonInternalK8sError || stderrors.As(err, &k8sErr):
		category = "kubernetes"
	case code != 0:
		category = "remote-api"
	case stderrors.As(err, &apiErr):
		category = "transport"
	}

	if code != 0 {
		return fmt.Errorf("%s (operation=%s, category=%s, HTTP %d)", reason, operation, category, code)
	}
	return fmt.Errorf("%s (operation=%s, category=%s)", reason, operation, category)
}

func ManageSuccessWithRequeue(ctx context.Context, obj coralogix.Object, interval time.Duration) (reconcile.Result, error) {
	conditions := obj.GetConditions()
	if utils.SetSyncedConditionTrue(&conditions, obj.GetGeneration(), utils.ReasonRemoteSyncedSuccessfully) || obj.GetPrintableStatus() != "RemoteSynced" {
		obj.SetConditions(conditions)
		obj.SetPrintableStatus("RemoteSynced")
		if err := config.GetClient().Status().Update(ctx, obj); err != nil {
			return ManageErrorWithRequeue(ctx, obj, "update-status", utils.ReasonInternalK8sError, err)
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
