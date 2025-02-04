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

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ScopeReconciler reconciles a Scope object
type ScopeReconciler struct {
	client.Client
	ScopesClient *cxsdk.ScopesClient
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=scopes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=scopes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=scopes/finalizers,verbs=update

var (
	scopeFinalizerName = "scope.coralogix.com/finalizer"
)

func (r *ScopeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"scope", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	scope := &coralogixv1alpha1.Scope{}
	if err := r.Get(ctx, req.NamespacedName, scope); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(scope.Status.ID, "") == "" {
		err := r.create(ctx, log, scope)
		if err != nil {
			log.Error(err, "Error on creating Scope")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !scope.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, scope)
		if err != nil {
			log.Error(err, "Error on deleting Scope")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(scope.GetLabels()) {
		err := r.deleteRemoteScope(ctx, log, *scope.Status.ID)
		if err != nil {
			log.Error(err, "Error on deleting Scope")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, scope)
	if err != nil {
		log.Error(err, "Error on updating Scope")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *ScopeReconciler) create(ctx context.Context, log logr.Logger, scope *coralogixv1alpha1.Scope) error {
	createRequest, err := scope.Spec.ExtractCreateScopeRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.V(1).Info("Creating remote scope", "scope", protojson.Format(createRequest))
	createResponse, err := r.ScopesClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote scope: %w", err)
	}
	log.V(1).Info("Remote scope created", "response", protojson.Format(createResponse))

	scope.Status = coralogixv1alpha1.ScopeStatus{
		ID: &createResponse.Scope.Id,
	}

	log.V(1).Info("Updating Scope status", "id", createResponse.Scope.Id)
	if err = r.Status().Update(ctx, scope); err != nil {
		if deleteErr := r.deleteRemoteScope(ctx, log, *scope.Status.ID); deleteErr != nil {
			return fmt.Errorf("error to delete scope after status update error. Update error: %w. Deletion error: %w", err, deleteErr)
		}
		return fmt.Errorf("error to update scope status: %w", err)
	}

	if !controllerutil.ContainsFinalizer(scope, scopeFinalizerName) {
		log.V(1).Info("Updating Scope to add finalizer", "id", createResponse.Scope.Id)
		controllerutil.AddFinalizer(scope, scopeFinalizerName)
		if err := r.Update(ctx, scope); err != nil {
			return fmt.Errorf("error on updating Scope: %w", err)
		}
	}

	return nil
}

func (r *ScopeReconciler) update(ctx context.Context, log logr.Logger, scope *coralogixv1alpha1.Scope) error {
	updateRequest, err := scope.Spec.ExtractUpdateScopeRequest(*scope.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.V(1).Info("Updating remote scope", "scope", protojson.Format(updateRequest))
	updateResponse, err := r.ScopesClient.Update(ctx, updateRequest)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("scope not found on remote, removing id from status")
			scope.Status = coralogixv1alpha1.ScopeStatus{
				ID: ptr.To(""),
			}
			if err = r.Status().Update(ctx, scope); err != nil {
				return fmt.Errorf("error on updating Scope status: %w", err)
			}
			return fmt.Errorf("scope not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating scope: %w", err)
	}
	log.V(1).Info("Remote scope updated", "scope", protojson.Format(updateResponse))

	return nil
}

func (r *ScopeReconciler) delete(ctx context.Context, log logr.Logger, scope *coralogixv1alpha1.Scope) error {
	if err := r.deleteRemoteScope(ctx, log, *scope.Status.ID); err != nil {
		return fmt.Errorf("error on deleting remote scope: %w", err)
	}

	log.V(1).Info("Removing finalizer from Scope")
	controllerutil.RemoveFinalizer(scope, scopeFinalizerName)
	if err := r.Update(ctx, scope); err != nil {
		return fmt.Errorf("error on updating Scope: %w", err)
	}

	return nil
}

func (r *ScopeReconciler) deleteRemoteScope(ctx context.Context, log logr.Logger, scopeID string) error {
	log.V(1).Info("Deleting scope from remote", "id", scopeID)
	if _, err := r.ScopesClient.Delete(ctx, &cxsdk.DeleteScopeRequest{Id: scopeID}); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote scope", "id", scopeID)
		return fmt.Errorf("error to delete remote scope %s: %w", scopeID, err)
	}
	log.V(1).Info("scope was deleted from remote", "id", scopeID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScopeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Scope{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
