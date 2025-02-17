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

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var (
	k8sClient client.Client
	scheme    *runtime.Scheme
)

func InitClient(c client.Client) {
	k8sClient = c
}

func GetClient() client.Client {
	return k8sClient
}

func InitScheme(s *runtime.Scheme) {
	scheme = s
}

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
		"name", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace)
	var err error

	if err = k8sClient.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if !r.CheckIDInStatus(obj) {
		log.V(1).Info("Resource ID is missing; handling creation for resource")
		if obj, err = r.HandleCreation(ctx, log, obj); err != nil {
			log.Error(err, "Error handling creation")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		if err = k8sClient.Status().Update(ctx, obj); err != nil {
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
		if err = r.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error handling deletion")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		log.V(1).Info("Removing finalizer")
		if err = RemoveFinalizer(ctx, log, obj, r); err != nil {
			log.Error(err, "Error removing finalizer")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}

		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(obj.GetLabels()) {
		log.V(1).Info("Resource doesn't match label filter, handling deletion")
		if err = r.HandleDeletion(ctx, log, obj); err != nil {
			log.Error(err, "Error deleting from remote")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	log.V(1).Info("Handling update")
	if err = r.HandleUpdate(ctx, log, obj); err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("resource not found on remote")
			uObj := &unstructured.Unstructured{}
			if err2 := scheme.Convert(obj, uObj, ctx); err2 != nil {
				return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("failed to convert object to unstructured: %w", err2)
			}
			if err2 := unstructured.SetNestedField(uObj.Object, "", "status", "id"); err2 != nil {
				return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("error on updating %s status: %v", gvk, err2)
			}
			if err2 := k8sClient.Status().Update(ctx, uObj); err2 != nil {
				return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("error on updating %s status: %w", gvk, err2)
			}
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("%s not found on remote: %w", gvk, err)
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, fmt.Errorf("error on updating %s: %w", gvk, err)
	}

	return ctrl.Result{}, nil
}

func AddFinalizer(ctx context.Context, log logr.Logger, obj client.Object, r CoralogixReconciler) error {
	if !controllerutil.ContainsFinalizer(obj, r.FinalizerName()) {
		log.V(1).Info("Adding finalizer")
		controllerutil.AddFinalizer(obj, r.FinalizerName())
		if err := k8sClient.Update(ctx, obj); err != nil {
			return fmt.Errorf("error updating k8s object: %w", err)
		}
	}
	return nil
}

func RemoveFinalizer(ctx context.Context, log logr.Logger, obj client.Object, r CoralogixReconciler) error {
	log.V(1).Info("Removing finalizer")
	controllerutil.RemoveFinalizer(obj, r.FinalizerName())
	if err := k8sClient.Update(ctx, obj); err != nil {
		return fmt.Errorf("error updating k8s object: %w", err)
	}
	return nil
}

func objToGVK(obj client.Object) string {
	gvks, _, _ := scheme.ObjectKinds(obj)
	if len(gvks) == 0 {
		return ""
	}
	return gvks[0].String()
}
