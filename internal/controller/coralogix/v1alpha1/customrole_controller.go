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
	"strconv"

	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
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
)

// CustomRoleReconciler reconciles a CustomRole object
type CustomRoleReconciler struct {
	client.Client
	CustomRolesClient *cxsdk.RolesClient
	Scheme            *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=customroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=customroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=customroles/finalizers,verbs=update

var (
	customRoleFinalizerName = "custom-role.coralogix.com/finalizer"
)

func (r *CustomRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"customRole", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	customRole := &coralogixv1alpha1.CustomRole{}
	if err := r.Get(ctx, req.NamespacedName, customRole); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(customRole.Status.ID, "") == "" {
		err := r.create(ctx, log, customRole)
		if err != nil {
			log.Error(err, "Error on creating CustomRole")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !customRole.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, customRole)
		if err != nil {
			log.Error(err, "Error on deleting CustomRole")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, customRole)
	if err != nil {
		log.Error(err, "Error on updating CustomRole")
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *CustomRoleReconciler) create(ctx context.Context, log logr.Logger, customRole *coralogixv1alpha1.CustomRole) error {
	createRequest := customRole.Spec.ExtractCreateCustomRoleRequest()
	log.V(1).Info("Creating remote custom-role", "custom-role", protojson.Format(createRequest))
	createResponse, err := r.CustomRolesClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote custom-role: %w", err)
	}
	log.V(1).Info("Remote custom-role created", "response", protojson.Format(createResponse))

	id := strconv.Itoa(int(createResponse.Id))
	customRole.Status = coralogixv1alpha1.CustomRoleStatus{
		ID: &id,
	}

	log.V(1).Info("Updating CustomRole status", "id", id)
	if err = r.Status().Update(ctx, customRole); err != nil {
		if deleteErr := r.deleteRemoteCustomRole(ctx, log, *customRole.Status.ID); deleteErr != nil {
			return fmt.Errorf("error to delete custom-role after status update error. Update error: %w. Deletion error: %w", err, deleteErr)
		}
		return fmt.Errorf("error to update custom-role status: %w", err)
	}

	if !controllerutil.ContainsFinalizer(customRole, customRoleFinalizerName) {
		log.V(1).Info("Updating CustomRole to add finalizer", "id", id)
		controllerutil.AddFinalizer(customRole, customRoleFinalizerName)
		if err := r.Update(ctx, customRole); err != nil {
			return fmt.Errorf("error on updating CustomRole: %w", err)
		}
	}

	return nil
}

func (r *CustomRoleReconciler) update(ctx context.Context, log logr.Logger, customRole *coralogixv1alpha1.CustomRole) error {
	updateRequest, err := customRole.Spec.ExtractUpdateCustomRoleRequest(*customRole.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.V(1).Info("Updating remote custom-role", "custom-role", protojson.Format(updateRequest))
	updateResponse, err := r.CustomRolesClient.Update(ctx, updateRequest)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("custom-role not found on remote, removing id from status")
			customRole.Status = coralogixv1alpha1.CustomRoleStatus{
				ID: ptr.To(""),
			}
			if err = r.Status().Update(ctx, customRole); err != nil {
				return fmt.Errorf("error on updating CustomRole status: %w", err)
			}
			return fmt.Errorf("custom-role not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating custom-role: %w", err)
	}
	log.V(1).Info("Remote custom-role updated", "custom-role", protojson.Format(updateResponse))

	return nil
}

func (r *CustomRoleReconciler) delete(ctx context.Context, log logr.Logger, customRole *coralogixv1alpha1.CustomRole) error {
	if err := r.deleteRemoteCustomRole(ctx, log, *customRole.Status.ID); err != nil {
		return fmt.Errorf("error on deleting remote custom-role: %w", err)
	}

	log.V(1).Info("Removing finalizer from CustomRole")
	controllerutil.RemoveFinalizer(customRole, customRoleFinalizerName)
	if err := r.Update(ctx, customRole); err != nil {
		return fmt.Errorf("error on updating CustomRole: %w", err)
	}

	return nil
}

func (r *CustomRoleReconciler) deleteRemoteCustomRole(ctx context.Context, log logr.Logger, customRoleID string) error {
	log.V(1).Info("Deleting custom-role from remote", "id", customRoleID)
	id, err := strconv.Atoi(customRoleID)
	if err != nil {
		return fmt.Errorf("error on converting custom-role id to int: %w", err)
	}

	if _, err := r.CustomRolesClient.Delete(ctx, &cxsdk.DeleteRoleRequest{RoleId: uint32(id)}); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote custom-role", "id", customRoleID)
		return fmt.Errorf("error to delete remote custom-role %s: %w", customRoleID, err)
	}
	log.V(1).Info("custom-role was deleted from remote", "id", customRoleID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.CustomRole{}).
		Complete(r)
}
