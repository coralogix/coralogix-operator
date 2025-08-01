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
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// CustomRoleReconciler reconciles a CustomRole object
type CustomRoleReconciler struct {
	CustomRolesClient *cxsdk.RolesClient
	Interval          time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=customroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=customroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=customroles/finalizers,verbs=update

func (r *CustomRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.CustomRole{}, r)
}

func (r *CustomRoleReconciler) FinalizerName() string {
	return "custom-role.coralogix.com/finalizer"
}

func (r *CustomRoleReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *CustomRoleReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	customRole := obj.(*coralogixv1alpha1.CustomRole)
	createRequest := customRole.Spec.ExtractCreateCustomRoleRequest()
	log.Info("Creating remote customRole", "customRole", protojson.Format(createRequest))
	createResponse, err := r.CustomRolesClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote customRole: %w", err)
	}
	log.Info("Remote customRole created", "response", protojson.Format(createResponse))

	customRole.Status = coralogixv1alpha1.CustomRoleStatus{
		ID: ptr.To(strconv.Itoa(int(createResponse.Id))),
	}

	return nil
}

func (r *CustomRoleReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	customRole := obj.(*coralogixv1alpha1.CustomRole)
	updateRequest, err := customRole.Spec.ExtractUpdateCustomRoleRequest(*customRole.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote customRole", "customRole", protojson.Format(updateRequest))
	updateResponse, err := r.CustomRolesClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote customRole updated", "customRole", protojson.Format(updateResponse))

	return nil
}

func (r *CustomRoleReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	customRole := obj.(*coralogixv1alpha1.CustomRole)
	log.Info("Deleting customRole from remote system", "id", *customRole.Status.ID)
	id, err := strconv.Atoi(*customRole.Status.ID)
	if err != nil {
		return fmt.Errorf("error on converting custom-role id to int: %w", err)
	}

	_, err = r.CustomRolesClient.Delete(ctx, &cxsdk.DeleteRoleRequest{RoleId: uint32(id)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote customRole", "id", *customRole.Status.ID)
		return fmt.Errorf("error deleting remote customRole %s: %w", *customRole.Status.ID, err)
	}
	log.Info("CustomRole deleted from remote system", "id", *customRole.Status.ID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.CustomRole{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
