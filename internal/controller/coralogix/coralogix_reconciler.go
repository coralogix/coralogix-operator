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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/coralogix/coralogix-operator/internal/utils"
)

// CoralogixReconciler defines the required methods for all Coralogix controllers.
type CoralogixReconciler interface {
	GetClient() client.Client
	HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error)
	HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error
	FinalizerName() string
}

type BaseReconciler struct {
	coralogixReconciler CoralogixReconciler
}

func NewBaseReconciler(coralogixReconciler CoralogixReconciler) *BaseReconciler {
	return &BaseReconciler{coralogixReconciler: coralogixReconciler}
}

func (r *BaseReconciler) ReconcileResource(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, error) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	log := log.FromContext(ctx).WithValues(
		"kind", kind,
		"name", req.NamespacedName.Name, "namespace", req.NamespacedName.Namespace)

	if err := r.coralogixReconciler.GetClient().Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	hasID, err := r.CheckIDInStatus(ctx, obj)
	if err != nil {
		log.Error(err, "Error checking for ID in status")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}
	if !hasID {
		log.V(1).Info("Resource ID is missing; handling creation for resource")
		if createdObj, err := r.coralogixReconciler.HandleCreation(ctx, log, obj); err != nil {
			log.Error(err, "Error handling creation")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		} else if err = r.coralogixReconciler.GetClient().Status().Update(ctx, createdObj); err != nil {
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		log.V(1).Info("Adding finalizer")
		if err := r.AddFinalizer(ctx, log, obj); err != nil {
			log.Error(err, "Error adding finalizer")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		log.V(1).Info("Resource is being deleted; handling deletion")
		if err := r.coralogixReconciler.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error handling deletion")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		log.V(1).Info("Removing finalizer")
		if err := r.RemoveFinalizer(ctx, log, obj); err != nil {
			log.Error(err, "Error removing finalizer")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(obj.GetLabels()) {
		log.V(1).Info("Resource labels do not match label filter; handling deletion")
		if err := r.coralogixReconciler.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error deleting from remote")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Handling update")
	if err := r.coralogixReconciler.HandleUpdate(ctx, log, obj); err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info(fmt.Sprintf("%s not found on remote, recreating it", kind))
			if err2 := unstructured.SetNestedField(obj.(*unstructured.Unstructured).Object, "", "status", "id"); err2 != nil {
				return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("error on updating %s status: %v", kind, err2)
			}
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("%s not found on remote: %w", kind, err)
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("error on updating %s: %w", kind, err)
	}

	return ctrl.Result{}, nil
}

func (r *BaseReconciler) CheckIDInStatus(ctx context.Context, obj client.Object) (bool, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	if err := r.coralogixReconciler.GetClient().Get(ctx, types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, u); err != nil {
		return false, err
	}

	_, found, err := unstructured.NestedString(u.Object, "status", "id")
	return found, err
}

func (r *BaseReconciler) AddFinalizer(ctx context.Context, log logr.Logger, obj client.Object) error {
	if !controllerutil.ContainsFinalizer(obj, r.coralogixReconciler.FinalizerName()) {
		log.V(1).Info(fmt.Sprintf("Adding finalizer to %s", obj.GetObjectKind().GroupVersionKind().Kind))
		controllerutil.AddFinalizer(obj, r.coralogixReconciler.FinalizerName())
		if err := r.coralogixReconciler.GetClient().Update(ctx, obj); err != nil {
			return fmt.Errorf("error updating %s: %w", obj.GetObjectKind().GroupVersionKind(), err)
		}
	}
	return nil
}

func (r *BaseReconciler) RemoveFinalizer(ctx context.Context, log logr.Logger, obj client.Object) error {
	log.V(1).Info(fmt.Sprintf("Removing finalizer from %s", obj.GetObjectKind().GroupVersionKind()))
	controllerutil.RemoveFinalizer(obj, r.coralogixReconciler.FinalizerName())
	if err := r.coralogixReconciler.GetClient().Update(ctx, obj); err != nil {
		return fmt.Errorf("error updating %s: %w", obj.GetObjectKind().GroupVersionKind(), err)
	}
	return nil
}
