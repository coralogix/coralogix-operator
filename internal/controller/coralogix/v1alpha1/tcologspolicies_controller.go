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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// TCOLogsPoliciesReconciler reconciles a TCOLogsPolicies object
type TCOLogsPoliciesReconciler struct {
	client.Client
	CoralogixClientSet *cxsdk.ClientSet
	Scheme             *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=tcologspolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=tcologspolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=tcologspolicies/finalizers,verbs=update

var (
	tcoLogsPoliciesFinalizerName = "tco-logs-policies.coralogix.com/finalizer"
)

func (r *TCOLogsPoliciesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"tcoLogsPolicies", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	tcoLogsPolicies := &coralogixv1alpha1.TCOLogsPolicies{}
	if err := r.Get(ctx, req.NamespacedName, tcoLogsPolicies); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if !tcoLogsPolicies.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, tcoLogsPolicies)
		if err != nil {
			log.Error(err, "Error on deleting TCOLogsPolicies")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(tcoLogsPolicies.GetLabels()) {
		err := r.deleteRemoteTCOLogsPolicies(ctx, log)
		if err != nil {
			log.Error(err, "Error on deleting TCOLogsPolicies")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.overwrite(ctx, log, tcoLogsPolicies)
	if err != nil {
		log.Error(err, "Error on overwriting TCOLogsPolicies")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *TCOLogsPoliciesReconciler) overwrite(ctx context.Context, log logr.Logger, tcoLogsPolicies *coralogixv1alpha1.TCOLogsPolicies) error {
	overwriteRequest, err := tcoLogsPolicies.Spec.ExtractOverwriteLogPoliciesRequest(ctx, r.CoralogixClientSet)
	if err != nil {
		return fmt.Errorf("error on extracting overwrite log policies request: %w", err)
	}
	log.V(1).Info("Overwriting remote tco-logs-policies", "tco-logs-policies", protojson.Format(overwriteRequest))
	overwriteResponse, err := r.CoralogixClientSet.TCOPolicies().OverwriteTCOLogsPolicies(ctx, overwriteRequest)
	if err != nil {
		return fmt.Errorf("error on overwriting remote tco-logs-policies: %w", err)
	}
	log.V(1).Info("Remote tco-logs-policies overwritten", "response", protojson.Format(overwriteResponse))

	if !controllerutil.ContainsFinalizer(tcoLogsPolicies, tcoLogsPoliciesFinalizerName) {
		log.V(1).Info("Updating TCOLogsPolicies to add finalizer", "name", tcoLogsPolicies.Name)
		controllerutil.AddFinalizer(tcoLogsPolicies, tcoLogsPoliciesFinalizerName)
		if err = r.Update(ctx, tcoLogsPolicies); err != nil {
			return fmt.Errorf("error on updating TCOLogsPolicies: %w", err)
		}
	}

	return nil
}

func (r *TCOLogsPoliciesReconciler) delete(ctx context.Context, log logr.Logger, tcoLogsPolicies *coralogixv1alpha1.TCOLogsPolicies) error {
	if err := r.deleteRemoteTCOLogsPolicies(ctx, log); err != nil {
		return fmt.Errorf("error on deleting TCOLogsPolicies: %w", err)
	}

	log.V(1).Info("Removing finalizer from TCOLogsPolicies")
	controllerutil.RemoveFinalizer(tcoLogsPolicies, tcoLogsPoliciesFinalizerName)
	if err := r.Update(ctx, tcoLogsPolicies); err != nil {
		return fmt.Errorf("error on updating TCOLogsPolicies: %w", err)
	}

	return nil
}

func (r *TCOLogsPoliciesReconciler) deleteRemoteTCOLogsPolicies(ctx context.Context, log logr.Logger) error {
	deleteTCOLogsPoliciesRequest := &cxsdk.AtomicOverwriteLogPoliciesRequest{}
	log.V(1).Info("Deleting TCOLogsPolicies")
	if _, err := r.CoralogixClientSet.TCOPolicies().OverwriteTCOLogsPolicies(ctx, deleteTCOLogsPoliciesRequest); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Received an error while Deleting a TCOLogsPolicies")
		return err
	}

	log.V(1).Info("tco-logs-policies was deleted from remote")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCOLogsPoliciesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.TCOLogsPolicies{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
