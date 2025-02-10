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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/coralogix/coralogix-operator/internal/utils"
)

// CoralogixReconciler defines the required methods for all Coralogix controllers.
type CoralogixReconciler interface {
	HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error
	HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error
	CheckIDInStatus(obj client.Object) bool
	AddFinalizer(ctx context.Context, log logr.Logger, obj client.Object) error
	RemoveFinalizer(ctx context.Context, log logr.Logger, obj client.Object) error
}

func ReconcileResource(ctx context.Context, req ctrl.Request, client client.Client, obj client.Object, reconciler CoralogixReconciler) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"name", req.NamespacedName.Name, "namespace", req.NamespacedName.Namespace)

	if err := client.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if hasID := reconciler.CheckIDInStatus(obj); !hasID {
		log.V(1).Info("Resource ID is missing; handling creation for resource")
		if err := reconciler.HandleCreation(ctx, log, obj); err != nil {
			log.Error(err, "Error handling creation")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		log.V(1).Info("Adding finalizer")
		if err := reconciler.AddFinalizer(ctx, log, obj); err != nil {
			log.Error(err, "Error adding finalizer")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		log.V(1).Info("Resource is being deleted; handling deletion")
		if err := reconciler.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error handling deletion")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		log.V(1).Info("Removing finalizer")
		if err := reconciler.RemoveFinalizer(ctx, log, obj); err != nil {
			log.Error(err, "Error removing finalizer")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(obj.GetLabels()) {
		log.V(1).Info("Resource labels do not match label filter; handling deletion")
		if err := reconciler.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error deleting from remote")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Handling update")
	if err := reconciler.HandleUpdate(ctx, log, obj); err != nil {
		log.Error(err, "Error handling update")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}
