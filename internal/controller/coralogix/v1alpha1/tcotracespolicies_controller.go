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

	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

// TCOTracesPoliciesReconciler reconciles a TCOTracesPolicies object
type TCOTracesPoliciesReconciler struct {
	client.Client
	TCOClient *cxsdk.TCOPoliciesClient
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=tcotracespolicies/finalizers,verbs=update

var (
	tcoTracesPoliciesFinalizerName = "tco-traces-policies.coralogix.com/finalizer"
)

func (r *TCOTracesPoliciesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"tcoTracesPolicies", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	tcoTracesPolicies := &coralogixv1alpha1.TCOTracesPolicies{}
	if err := r.Get(ctx, req.NamespacedName, tcoTracesPolicies); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	if !tcoTracesPolicies.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, tcoTracesPolicies)
		if err != nil {
			log.Error(err, "Error on deleting TCOTracesPolicies")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.overwrite(ctx, log, tcoTracesPolicies)
	if err != nil {
		log.Error(err, "Error on overwriting TCOTracesPolicies")
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *TCOTracesPoliciesReconciler) overwrite(ctx context.Context, log logr.Logger, tcoTracesPolicies *coralogixv1alpha1.TCOTracesPolicies) error {
	overwriteRequest, err := tcoTracesPolicies.Spec.ExtractOverwriteTracesPoliciesRequest()
	if err != nil {
		return fmt.Errorf("error on extracting overwrite log policies request: %w", err)
	}
	log.V(1).Info("Overwriting remote tco-Traces-policies", "tco-Traces-policies", protojson.Format(overwriteRequest))
	overwriteResponse, err := r.TCOClient.OverwriteTCOTracesPolicies(ctx, overwriteRequest)
	if err != nil {
		return fmt.Errorf("error on overwriting remote tco-Traces-policies: %w", err)
	}
	log.V(1).Info("Remote tco-Traces-policies overwritten", "response", protojson.Format(overwriteResponse))

	if !controllerutil.ContainsFinalizer(tcoTracesPolicies, tcoTracesPoliciesFinalizerName) {
		log.V(1).Info("Updating TCOTracesPolicies to add finalizer", "name", tcoTracesPolicies.Name)
		controllerutil.AddFinalizer(tcoTracesPolicies, tcoTracesPoliciesFinalizerName)
		if err = r.Update(ctx, tcoTracesPolicies); err != nil {
			return fmt.Errorf("error on updating TCOTracesPolicies: %w", err)
		}
	}

	return nil
}

func (r *TCOTracesPoliciesReconciler) delete(ctx context.Context, log logr.Logger, tcoTracesPolicies *coralogixv1alpha1.TCOTracesPolicies) error {
	if _, err := r.TCOClient.OverwriteTCOTracesPolicies(ctx, &cxsdk.AtomicOverwriteSpanPoliciesRequest{}); err != nil && cxsdk.Code(err) != codes.NotFound {
		return fmt.Errorf("error to delete remote tco-Traces-policies: %w", err)
	}
	log.V(1).Info("tco-Traces-policies was deleted from remote", "name", tcoTracesPolicies.Name)

	log.V(1).Info("Removing finalizer from TCOTracesPolicies")
	controllerutil.RemoveFinalizer(tcoTracesPolicies, tcoTracesPoliciesFinalizerName)
	if err := r.Update(ctx, tcoTracesPolicies); err != nil {
		return fmt.Errorf("error on updating TCOTracesPolicies: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCOTracesPoliciesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.TCOTracesPolicies{}).
		Complete(r)
}
