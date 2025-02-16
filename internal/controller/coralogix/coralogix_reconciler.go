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

package coralogix

import (
	"context"
	"fmt"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/coralogix/coralogix-operator/internal/utils"
)

var Client client.Client
var Schema *runtime.Scheme

// CoralogixReconciler defines the required methods for all Coralogix controllers.
type CoralogixReconciler interface {
	HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error)
	HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error
	FinalizerName() string
	CheckIDInStatus(obj client.Object) bool
}

func ReconcileResource(ctx context.Context, req ctrl.Request, obj client.Object, r CoralogixReconciler) (ctrl.Result, error) {
	gvk := objToGVK(obj)
	log := log.FromContext(ctx).WithValues(
		"gvk", gvk,
		"name", req.NamespacedName.Name, "namespace", req.NamespacedName.Namespace)

	if err := Client.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if !r.CheckIDInStatus(obj) {
		log.V(1).Info("Resource ID is missing; handling creation for resource")
		var err error
		if obj, err = r.HandleCreation(ctx, log, obj); err != nil {
			log.Error(err, "Error handling creation")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		if err = Client.Status().Update(ctx, obj); err != nil {
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		log.V(1).Info("Adding finalizer")
		if err = AddFinalizer(ctx, log, obj, r); err != nil {
			log.Error(err, "Error adding finalizer")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		log.V(1).Info("Resource is being deleted; handling deletion")
		if err := r.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error handling deletion")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		log.V(1).Info("Removing finalizer")
		if err := RemoveFinalizer(ctx, log, obj, r); err != nil {
			log.Error(err, "Error removing finalizer")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(obj.GetLabels()) {
		log.V(1).Info("Error handling deletion")
		if err := r.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error deleting from remote")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Handling update")
	if err := r.HandleUpdate(ctx, log, obj); err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info(fmt.Sprintf("%s not found on remote, recreating it", gvk))
			if err2 := unstructured.SetNestedField(obj.(*unstructured.Unstructured).Object, "", "status", "id"); err2 != nil {
				return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("error on updating %s status: %v", gvk, err2)
			}
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("%s not found on remote: %w", gvk, err)
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("error on updating %s: %w", gvk, err)
	}

	return ctrl.Result{}, nil
}

func AddFinalizer(ctx context.Context, log logr.Logger, obj client.Object, r CoralogixReconciler) error {
	if !controllerutil.ContainsFinalizer(obj, r.FinalizerName()) {
		log.V(1).Info(fmt.Sprintf("Adding finalizer to %s", obj.GetObjectKind().GroupVersionKind().Kind))
		controllerutil.AddFinalizer(obj, r.FinalizerName())
		if err := Client.Update(ctx, obj); err != nil {
			return fmt.Errorf("error updating %s: %w", obj.GetObjectKind().GroupVersionKind(), err)
		}
	}
	return nil
}

func RemoveFinalizer(ctx context.Context, log logr.Logger, obj client.Object, r CoralogixReconciler) error {
	log.V(1).Info("Removing finalizer from %s", obj.GetObjectKind().GroupVersionKind())
	controllerutil.RemoveFinalizer(obj, r.FinalizerName())
	if err := Client.Update(ctx, obj); err != nil {
		return fmt.Errorf("error updating %s: %w", obj.GetObjectKind().GroupVersionKind(), err)
	}
	return nil
}

func objToGVK(obj client.Object) string {
	gvks, _, _ := Schema.ObjectKinds(obj)
	if len(gvks) == 0 {
		return ""
	}
	return gvks[0].String()
}
