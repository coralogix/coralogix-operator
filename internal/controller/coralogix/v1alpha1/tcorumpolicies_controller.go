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

	"github.com/coralogix/coralogix-operator/v2/internal/utils"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"
	archiveretentions "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/retentions_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	"github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
)

// TCORumPoliciesReconciler reconciles a TCORumPolicies object
type TCORumPoliciesReconciler struct {
	TCOPoliciesClient       *tcopolicies.PoliciesServiceAPIService
	ArchiveRetentionsClient *archiveretentions.RetentionsServiceAPIService
	Interval                time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=tcorumpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=tcorumpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=tcorumpolicies/finalizers,verbs=update

func (r *TCORumPoliciesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.TCORumPolicies{}, r)
}

func (r *TCORumPoliciesReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *TCORumPoliciesReconciler) overwrite(ctx context.Context, log logr.Logger, tcoRumPolicies *coralogixv1alpha1.TCORumPolicies) error {
	overwriteRequest, err := tcoRumPolicies.Spec.ExtractOverwriteRumPoliciesRequest(ctx, r.ArchiveRetentionsClient)
	if err != nil {
		return fmt.Errorf("error on extracting overwrite rum policies request: %w", err)
	}
	log.Info("Overwriting remote tco-rum-policies", "tco-rum-policies", utils.FormatJSON(overwriteRequest))
	overwriteResponse, httpResp, err := r.TCOPoliciesClient.
		PoliciesServiceAtomicOverwriteRumPolicies(ctx).
		AtomicOverwriteRumPoliciesRequest(*overwriteRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on overwriting remote tco-rum-policies: %w", oapicxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote tco-rum-policies overwritten", "response", utils.FormatJSON(overwriteResponse))
	return nil
}

func (r *TCORumPoliciesReconciler) FinalizerName() string {
	return "tco-rum-policies.coralogix.com/finalizer"
}

func (r *TCORumPoliciesReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	tcoRumPolicies := obj.(*coralogixv1alpha1.TCORumPolicies)
	if err := r.overwrite(ctx, log, tcoRumPolicies); err != nil {
		return err
	}

	return coralogixreconciler.AddFinalizer(ctx, log, tcoRumPolicies, r)
}

func (r *TCORumPoliciesReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	tcoRumPolicies := obj.(*coralogixv1alpha1.TCORumPolicies)
	if err := r.overwrite(ctx, log, tcoRumPolicies); err != nil {
		return err
	}
	return coralogixreconciler.AddFinalizer(ctx, log, tcoRumPolicies, r)
}

func (r *TCORumPoliciesReconciler) HandleDeletion(ctx context.Context, log logr.Logger, _ client.Object) error {
	log.Info("Deleting TCORumPolicies")
	_, httpResp, err := r.TCOPoliciesClient.
		PoliciesServiceAtomicOverwriteRumPolicies(ctx).
		AtomicOverwriteRumPoliciesRequest(tcopolicies.AtomicOverwriteRumPoliciesRequest{Policies: nil}).
		Execute()
	if err != nil {
		if apiErr := oapicxsdk.NewAPIError(httpResp, err); !oapicxsdk.IsNotFound(apiErr) {
			return fmt.Errorf("error on deleting remote tco-rum-policies: %w", apiErr)
		}
	}

	log.Info("tco-rum-policies was deleted from remote")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCORumPoliciesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.TCORumPolicies{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
