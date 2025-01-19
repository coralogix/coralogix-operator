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

// GroupReconciler reconciles a Group object
type GroupReconciler struct {
	client.Client
	CXClientSet *cxsdk.ClientSet
	Scheme      *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=groups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=groups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=groups/finalizers,verbs=update

var (
	groupFinalizerName = "group.coralogix.com/finalizer"
)

func (r *GroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"group", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	group := &coralogixv1alpha1.Group{}
	if err := r.Get(ctx, req.NamespacedName, group); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(group.Status.ID, "") == "" {
		err := r.create(ctx, log, group)
		if err != nil {
			log.Error(err, "Error on creating Group")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !group.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, group)
		if err != nil {
			log.Error(err, "Error on deleting Group")
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, group)
	if err != nil {
		log.Error(err, "Error on updating Group")
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *GroupReconciler) create(ctx context.Context, log logr.Logger, group *coralogixv1alpha1.Group) error {
	createRequest, err := group.ExtractCreateGroupRequest(ctx, r.Client, r.CXClientSet)
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}

	log.V(1).Info("Creating remote group", "group", protojson.Format(createRequest))
	createResponse, err := r.CXClientSet.Groups().Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote group: %w", err)
	}
	log.V(1).Info("Remote group created", "response", protojson.Format(createResponse))

	id := strconv.Itoa(int(createResponse.GroupId.Id))
	group.Status = coralogixv1alpha1.GroupStatus{
		ID: &id,
	}

	log.V(1).Info("Updating Group status", "id", id)
	if err = r.Status().Update(ctx, group); err != nil {
		if deleteErr := r.deleteRemoteGroup(ctx, log, *group.Status.ID); deleteErr != nil {
			return fmt.Errorf("error to delete group after status update error. Update error: %w. Deletion error: %w", err, deleteErr)
		}
		return fmt.Errorf("error to update group status: %w", err)
	}

	if !controllerutil.ContainsFinalizer(group, groupFinalizerName) {
		log.V(1).Info("Updating Group to add finalizer", "id", id)
		controllerutil.AddFinalizer(group, groupFinalizerName)
		if err := r.Update(ctx, group); err != nil {
			return fmt.Errorf("error on updating Group: %w", err)
		}
	}

	return nil
}

func (r *GroupReconciler) update(ctx context.Context, log logr.Logger, group *coralogixv1alpha1.Group) error {
	updateRequest, err := group.ExtractUpdateGroupRequest(ctx, r.Client, r.CXClientSet, *group.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.V(1).Info("Updating remote group", "group", protojson.Format(updateRequest))
	updateResponse, err := r.CXClientSet.Groups().Update(ctx, updateRequest)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("group not found on remote, removing id from status")
			group.Status = coralogixv1alpha1.GroupStatus{
				ID: ptr.To(""),
			}
			if err = r.Status().Update(ctx, group); err != nil {
				return fmt.Errorf("error on updating Group status: %w", err)
			}
			return fmt.Errorf("group not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating group: %w", err)
	}
	log.V(1).Info("Remote group updated", "group", protojson.Format(updateResponse))

	return nil
}

func (r *GroupReconciler) delete(ctx context.Context, log logr.Logger, group *coralogixv1alpha1.Group) error {
	if err := r.deleteRemoteGroup(ctx, log, *group.Status.ID); err != nil {
		return fmt.Errorf("error on deleting remote group: %w", err)
	}

	log.V(1).Info("Removing finalizer from Group")
	controllerutil.RemoveFinalizer(group, groupFinalizerName)
	if err := r.Update(ctx, group); err != nil {
		return fmt.Errorf("error on updating Group: %w", err)
	}

	return nil
}

func (r *GroupReconciler) deleteRemoteGroup(ctx context.Context, log logr.Logger, groupID string) error {
	log.V(1).Info("Deleting group from remote", "id", groupID)
	id, err := strconv.Atoi(groupID)
	if err != nil {
		return fmt.Errorf("error on converting group id to int: %w", err)
	}

	if _, err := r.CXClientSet.Groups().Delete(ctx, &cxsdk.DeleteTeamGroupRequest{
		GroupId: ptr.To(cxsdk.TeamGroupID{Id: uint32(id)}),
	}); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote group", "id", groupID)
		return fmt.Errorf("error to delete remote group %s: %w", groupID, err)
	}
	log.V(1).Info("group was deleted from remote", "id", groupID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Group{}).
		Complete(r)
}
