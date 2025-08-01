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

// GroupReconciler reconciles a Group object
type GroupReconciler struct {
	CXClientSet *cxsdk.ClientSet
	Interval    time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=groups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=groups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=groups/finalizers,verbs=update

func (r *GroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Group{}, r)
}

func (r *GroupReconciler) FinalizerName() string {
	return "group.coralogix.com/finalizer"
}

func (r *GroupReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *GroupReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	group := obj.(*coralogixv1alpha1.Group)
	createRequest, err := group.ExtractCreateGroupRequest(ctx, r.CXClientSet)
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.Info("Creating remote group", "group", protojson.Format(createRequest))
	createResponse, err := r.CXClientSet.Groups().Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote group: %w", err)
	}
	log.Info("Remote group created", "group", protojson.Format(createResponse))

	group.Status = coralogixv1alpha1.GroupStatus{
		ID: ptr.To(strconv.Itoa(int(createResponse.GroupId.Id))),
	}

	return nil
}

func (r *GroupReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	group := obj.(*coralogixv1alpha1.Group)
	updateRequest, err := group.ExtractUpdateGroupRequest(ctx, r.CXClientSet, *group.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote group", "group", protojson.Format(updateRequest))
	updateResponse, err := r.CXClientSet.Groups().Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote group updated", "group", protojson.Format(updateResponse))

	return nil
}

func (r *GroupReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	group := obj.(*coralogixv1alpha1.Group)
	log.Info("Deleting group from remote system", "id", *group.Status.ID)
	id, err := strconv.Atoi(*group.Status.ID)
	if err != nil {
		return fmt.Errorf("error on converting custom-role id to int: %w", err)
	}

	_, err = r.CXClientSet.Groups().Delete(ctx, &cxsdk.DeleteTeamGroupRequest{GroupId: &cxsdk.TeamGroupID{Id: uint32(id)}})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote group", "id", *group.Status.ID)
		return fmt.Errorf("error deleting remote group %s: %w", *group.Status.ID, err)
	}
	log.Info("Group deleted from remote system", "id", *group.Status.ID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Group{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
