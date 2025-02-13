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
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// TCOTracesPoliciesReconciler reconciles a TCOTracesPolicies object
type TCOTracesPoliciesReconciler struct {
	CoralogixClientSet *cxsdk.ClientSet
	Scheme             *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies/finalizers,verbs=update

func (r *TCOTracesPoliciesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogix.ReconcileResource(ctx, req, &coralogixv1alpha1.TCOTracesPolicies{}, r)
}

func (r *TCOTracesPoliciesReconciler) overwrite(ctx context.Context, log logr.Logger, tcoTracesPolicies *coralogixv1alpha1.TCOTracesPolicies) error {
	overwriteRequest, err := tcoTracesPolicies.Spec.ExtractOverwriteTracesPoliciesRequest(ctx, r.CoralogixClientSet)
	if err != nil {
		return fmt.Errorf("error on extracting overwrite log policies request: %w", err)
	}
	log.V(1).Info("Overwriting remote tco-Traces-policies", "tco-Traces-policies", protojson.Format(overwriteRequest))
	overwriteResponse, err := r.CoralogixClientSet.TCOPolicies().OverwriteTCOTracesPolicies(ctx, overwriteRequest)
	if err != nil {
		return fmt.Errorf("error on overwriting remote tco-Traces-policies: %w", err)
	}
	log.V(1).Info("Remote tco-Traces-policies overwritten", "response", protojson.Format(overwriteResponse))

	return nil
}

func (r *TCOTracesPoliciesReconciler) FinalizerName() string {
	return "tco-traces-policies.coralogix.com/finalizer"
}

func (r *TCOTracesPoliciesReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error) {
	tcoTracesPolicies := obj.(*coralogixv1alpha1.TCOTracesPolicies)
	if err := r.overwrite(ctx, log, tcoTracesPolicies); err != nil {
		return nil, err
	}
	return tcoTracesPolicies, nil
}

func (r *TCOTracesPoliciesReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	tcoTracesPolicies := obj.(*coralogixv1alpha1.TCOTracesPolicies)
	return r.overwrite(ctx, log, tcoTracesPolicies)
}

func (r *TCOTracesPoliciesReconciler) HandleDeletion(ctx context.Context, log logr.Logger, _ client.Object) error {
	deleteTCOTracesPoliciesRequest := &cxsdk.AtomicOverwriteSpanPoliciesRequest{}
	log.V(1).Info("Deleting TCOTracesPolicies")
	if _, err := r.CoralogixClientSet.TCOPolicies().OverwriteTCOTracesPolicies(ctx, deleteTCOTracesPoliciesRequest); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Received an error while Deleting a TCOTracesPolicies")
		return err
	}

	log.V(1).Info("tco-traces-policies was deleted from remote")
	return nil
}

func (r *TCOTracesPoliciesReconciler) CheckIDInStatus(_ client.Object) bool {
	return true
}

func (r *TCOTracesPoliciesReconciler) GVK() schema.GroupVersionKind {
	return new(coralogixv1alpha1.TCOTracesPolicies).GetObjectKind().GroupVersionKind()
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCOTracesPoliciesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.TCOTracesPolicies{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
