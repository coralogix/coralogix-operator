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
	"time"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// ScopeReconciler reconciles a Scope object
type ScopeReconciler struct {
	ScopesClient *cxsdk.ScopesClient
	Interval     time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=scopes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=scopes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=scopes/finalizers,verbs=update

var _ coralogixreconciler.CoralogixReconciler = &ScopeReconciler{}

func (r *ScopeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Scope{}, r)
}

func (r *ScopeReconciler) FinalizerName() string {
	return "scope.coralogix.com/finalizer"
}

func (r *ScopeReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ScopeReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	scope := obj.(*coralogixv1alpha1.Scope)
	createRequest, err := scope.Spec.ExtractCreateScopeRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.Info("Creating remote scope", "scope", protojson.Format(createRequest))
	createResponse, err := r.ScopesClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote scope: %w", err)
	}
	log.Info("Remote scope created", "response", protojson.Format(createResponse))

	scope.Status = coralogixv1alpha1.ScopeStatus{
		ID: &createResponse.Scope.Id,
	}

	return nil
}

func (r *ScopeReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	scope := obj.(*coralogixv1alpha1.Scope)
	updateRequest, err := scope.Spec.ExtractUpdateScopeRequest(*scope.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote scope", "scope", protojson.Format(updateRequest))
	updateResponse, err := r.ScopesClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote scope updated", "scope", protojson.Format(updateResponse))

	return nil
}

func (r *ScopeReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	scope := obj.(*coralogixv1alpha1.Scope)
	id := *scope.Status.ID
	log.Info("Deleting scope from remote system", "id", id)
	_, err := r.ScopesClient.Delete(ctx, &cxsdk.DeleteScopeRequest{Id: id})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote scope", "id", id)
		return fmt.Errorf("error deleting remote scope %s: %w", id, err)
	}
	log.Info("Scope deleted from remote system", "id", id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScopeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Scope{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
