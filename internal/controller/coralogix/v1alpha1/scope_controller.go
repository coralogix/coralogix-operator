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

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ScopeReconciler reconciles a Scope object
type ScopeReconciler struct {
	ScopesClient *cxsdk.ScopesClient
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=scopes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=scopes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=scopes/finalizers,verbs=update

var _ coralogix.CoralogixReconciler = &ScopeReconciler{}

func (r *ScopeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogix.ReconcileResource(ctx, req, &coralogixv1alpha1.Scope{}, r)
}

func (r *ScopeReconciler) FinalizerName() string {
	return "scope.coralogix.com/finalizer"
}

func (r *ScopeReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error) {
	scope := obj.(*coralogixv1alpha1.Scope)
	createRequest, err := scope.Spec.ExtractCreateScopeRequest()
	if err != nil {
		return nil, fmt.Errorf("error on extracting create request: %w", err)
	}
	log.V(1).Info("Creating remote scope", "scope", protojson.Format(createRequest))
	createResponse, err := r.ScopesClient.Create(ctx, createRequest)
	if err != nil {
		return nil, fmt.Errorf("error on creating remote scope: %w", err)
	}
	log.V(1).Info("Remote scope created", "response", protojson.Format(createResponse))

	scope.Status = coralogixv1alpha1.ScopeStatus{
		ID: &createResponse.Scope.Id,
	}

	return scope, nil
}

func (r *ScopeReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	scope := obj.(*coralogixv1alpha1.Scope)
	updateRequest, err := scope.Spec.ExtractUpdateScopeRequest(*scope.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.V(1).Info("Updating remote scope", "scope", protojson.Format(updateRequest))
	updateResponse, err := r.ScopesClient.Update(ctx, updateRequest)
	if err != nil {
		return fmt.Errorf("error on updating remote scope: %w", err)
	}
	log.V(1).Info("Remote scope updated", "scope", protojson.Format(updateResponse))

	return nil
}

func (r *ScopeReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	scope := obj.(*coralogixv1alpha1.Scope)
	id := *scope.Status.ID
	log.V(1).Info("Deleting scope from remote system", "id", id)
	_, err := r.ScopesClient.Delete(ctx, &cxsdk.DeleteScopeRequest{Id: id})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote scope", "id", id)
		return fmt.Errorf("error deleting remote scope %s: %w", id, err)
	}
	log.V(1).Info("Scope deleted from remote system", "id", id)
	return nil
}

func (r *ScopeReconciler) CheckIDInStatus(obj client.Object) bool {
	scope := obj.(*coralogixv1alpha1.Scope)
	return scope.Status.ID != nil && *scope.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScopeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Scope{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
