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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// TCOTracesPoliciesReconciler reconciles a TCOTracesPolicies object
type TCOTracesPoliciesReconciler struct {
	CoralogixClientSet *cxsdk.ClientSet
	TCOPoliciesClient  *tcopolicies.PoliciesServiceAPIService
	Interval           time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies/finalizers,verbs=update

func (r *TCOTracesPoliciesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.TCOTracesPolicies{}, r)
}

func (r *TCOTracesPoliciesReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *TCOTracesPoliciesReconciler) overwrite(ctx context.Context, log logr.Logger, tcoTracesPolicies *coralogixv1alpha1.TCOTracesPolicies) error {
	overwriteRequest, err := tcoTracesPolicies.Spec.ExtractOverwriteTracesPoliciesRequest(ctx, r.CoralogixClientSet)
	if err != nil {
		return fmt.Errorf("error on extracting overwrite log policies request: %w", err)
	}
	log.Info("Overwriting remote tco-Traces-policies", "tco-Traces-policies", utils.FormatJSON(overwriteRequest))
	overwriteResponse, httpResp, err := r.TCOPoliciesClient.
		PoliciesServiceAtomicOverwriteSpanPolicies(ctx).
		AtomicOverwriteSpanPoliciesRequest(*overwriteRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on overwriting remote tco-Traces-policies: %w", oapicxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote tco-Traces-policies overwritten", "response", utils.FormatJSON(overwriteResponse))

	return nil
}

func (r *TCOTracesPoliciesReconciler) FinalizerName() string {
	return "tco-traces-policies.coralogix.com/finalizer"
}

func (r *TCOTracesPoliciesReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	tcoTracesPolicies := obj.(*coralogixv1alpha1.TCOTracesPolicies)
	if err := r.overwrite(ctx, log, tcoTracesPolicies); err != nil {
		return err
	}
	return coralogixreconciler.AddFinalizer(ctx, log, tcoTracesPolicies, r)
}

func (r *TCOTracesPoliciesReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	tcoTracesPolicies := obj.(*coralogixv1alpha1.TCOTracesPolicies)
	if err := r.overwrite(ctx, log, tcoTracesPolicies); err != nil {
		return err
	}
	return coralogixreconciler.AddFinalizer(ctx, log, tcoTracesPolicies, r)
}

func (r *TCOTracesPoliciesReconciler) HandleDeletion(ctx context.Context, log logr.Logger, _ client.Object) error {
	log.Info("Deleting TCOTracesPolicies")
	_, httpResp, err := r.TCOPoliciesClient.
		PoliciesServiceAtomicOverwriteSpanPolicies(ctx).
		AtomicOverwriteSpanPoliciesRequest(tcopolicies.AtomicOverwriteSpanPoliciesRequest{Policies: nil}).
		Execute()
	if err != nil {
		if apiErr := oapicxsdk.NewAPIError(httpResp, err); !oapicxsdk.IsNotFound(apiErr) {
			log.Error(err, "Received an error while Deleting a TCOTracesPolicies")
			return apiErr
		}
	}

	log.Info("tco-traces-policies was deleted from remote")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCOTracesPoliciesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.TCOTracesPolicies{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
