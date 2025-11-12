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

	"github.com/coralogix/coralogix-operator/internal/utils"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	tcopolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/policies_service"
	archiveretentions "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/retentions_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// TCOLogsPoliciesReconciler reconciles a TCOLogsPolicies object
type TCOLogsPoliciesReconciler struct {
	TCOPoliciesClient       *tcopolicies.PoliciesServiceAPIService
	ArchiveRetentionsClient *archiveretentions.RetentionsServiceAPIService
	Interval                time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=tcologspolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=tcologspolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=tcologspolicies/finalizers,verbs=update

func (r *TCOLogsPoliciesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.TCOLogsPolicies{}, r)
}

func (r *TCOLogsPoliciesReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *TCOLogsPoliciesReconciler) overwrite(ctx context.Context, log logr.Logger, tcoLogsPolicies *coralogixv1alpha1.TCOLogsPolicies) error {
	overwriteRequest, err := tcoLogsPolicies.Spec.ExtractOverwriteLogPoliciesRequest(ctx, r.ArchiveRetentionsClient)
	if err != nil {
		return fmt.Errorf("error on extracting overwrite log policies request: %w", err)
	}
	log.Info("Overwriting remote tco-logs-policies", "tco-logs-policies", utils.FormatJSON(overwriteRequest))
	overwriteResponse, httpResp, err := r.TCOPoliciesClient.
		PoliciesServiceAtomicOverwriteLogPolicies(ctx).
		AtomicOverwriteLogPoliciesRequest(*overwriteRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on overwriting remote tco-logs-policies: %w", oapicxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote tco-logs-policies overwritten", "response", utils.FormatJSON(overwriteResponse))
	return nil
}

func (r *TCOLogsPoliciesReconciler) FinalizerName() string {
	return "tco-logs-policies.coralogix.com/finalizer"
}

func (r *TCOLogsPoliciesReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	tcoLogsPolicies := obj.(*coralogixv1alpha1.TCOLogsPolicies)
	if err := r.overwrite(ctx, log, tcoLogsPolicies); err != nil {
		return err
	}

	return coralogixreconciler.AddFinalizer(ctx, log, tcoLogsPolicies, r)
}

func (r *TCOLogsPoliciesReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	tcoLogsPolicies := obj.(*coralogixv1alpha1.TCOLogsPolicies)
	if err := r.overwrite(ctx, log, tcoLogsPolicies); err != nil {
		return err
	}
	return coralogixreconciler.AddFinalizer(ctx, log, tcoLogsPolicies, r)
}

func (r *TCOLogsPoliciesReconciler) HandleDeletion(ctx context.Context, log logr.Logger, _ client.Object) error {
	log.Info("Deleting TCOLogsPolicies")
	_, httpResp, err := r.TCOPoliciesClient.
		PoliciesServiceAtomicOverwriteLogPolicies(ctx).
		AtomicOverwriteLogPoliciesRequest(tcopolicies.AtomicOverwriteLogPoliciesRequest{Policies: nil}).
		Execute()
	if err != nil {
		if apiErr := oapicxsdk.NewAPIError(httpResp, err); !oapicxsdk.IsNotFound(apiErr) {
			log.Error(err, "Received an error while Deleting a TCOLogsPolicies")
			return apiErr
		}
	}

	log.Info("tco-logs-policies was deleted from remote")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCOLogsPoliciesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.TCOLogsPolicies{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
