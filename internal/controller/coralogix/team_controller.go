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

package coralogix

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	TeamsClient clientset.TeamsClientInterface
	Scheme      *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=teams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=teams/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=teams/finalizers,verbs=update

var (
	teamFinalizerName = "team.coralogix.com/finalizer"
)

func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"team", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	team := &coralogixv1alpha1.Team{}
	if err := r.Get(ctx, req.NamespacedName, team); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if ptr.Deref(team.Status.Id, 0) == 0 {
		err := r.create(ctx, log, team)
		if err != nil {
			log.Error(err, "Error on creating Team")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !team.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, team)
		if err != nil {
			log.Error(err, "Error on deleting Team")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, team)
	if err != nil {
		log.Error(err, "Error on updating Team")
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *TeamReconciler) create(ctx context.Context, log logr.Logger, team *coralogixv1alpha1.Team) error {
	createRequest := team.Spec.ExtractCreateTeamRequest()
	log.V(1).Info("Creating remote team", "team", protojson.Format(createRequest))
	createResponse, err := r.TeamsClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote team: %w", err)
	}
	log.V(1).Info("Remote team created", "response", protojson.Format(createResponse))

	log.V(1).Info("Getting remote team", "team", protojson.Format(createRequest))
	getRequest := &cxsdk.GetTeamRequest{
		TeamId: &cxsdk.TeamID{
			Id: createResponse.TeamId.Id,
		},
	}
	getResponse, err := r.TeamsClient.Get(ctx, getRequest)
	if err != nil {
		return fmt.Errorf("error on getting remote team: %w", err)
	}

	log.V(1).Info("Updating Team status", "id", getResponse.TeamId)
	team.Status = coralogixv1alpha1.TeamStatus{
		Id:        ptr.To(getResponse.TeamId.Id),
		Retention: ptr.To(getResponse.Retention),
	}
	if err = r.Status().Update(ctx, team); err != nil {
		if err := r.deleteRemoteTeam(ctx, log, *team.Status.Id); err != nil {
			return fmt.Errorf("error to delete team after status update error -\n%v", team)
		}
		return fmt.Errorf("error to update team status -\n%v", team)
	}

	if !controllerutil.ContainsFinalizer(team, teamFinalizerName) {
		log.V(1).Info("Updating Team to add finalizer", "id", createResponse.TeamId)
		controllerutil.AddFinalizer(team, teamFinalizerName)
		if err := r.Update(ctx, team); err != nil {
			return fmt.Errorf("error on updating Team: %w", err)
		}
	}

	return nil
}

func (r *TeamReconciler) delete(ctx context.Context, log logr.Logger, team *coralogixv1alpha1.Team) error {
	if err := r.deleteRemoteTeam(ctx, log, *team.Status.Id); err != nil {
		return fmt.Errorf("error to delete team -\n%v", team)
	}

	controllerutil.RemoveFinalizer(team, teamFinalizerName)
	if err := r.Update(ctx, team); err != nil {
		return fmt.Errorf("error to update team -\n%v", team)
	}

	return nil
}

func (r *TeamReconciler) update(ctx context.Context, log logr.Logger, team *coralogixv1alpha1.Team) error {
	updateRequest := team.Spec.ExtractUpdateTeamRequest(*team.Status.Id)
	log.V(1).Info("Updating remote team", "team", protojson.Format(updateRequest))
	updateResponse, err := r.TeamsClient.Update(ctx, updateRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			team.Status = coralogixv1alpha1.TeamStatus{}
			if err = r.Status().Update(ctx, team); err != nil {
				return fmt.Errorf("error to update team status -\n%v", team)
			}
			return fmt.Errorf("team %d not found on remote, removed id from status", *team.Status.Id)
		}
		return fmt.Errorf("error to update team -\n%v", team)
	}
	log.V(1).Info("Remote team updated", "alert", protojson.Format(updateResponse))
	return nil
}

func (r *TeamReconciler) deleteRemoteTeam(ctx context.Context, log logr.Logger, teamId uint32) error {
	log.V(1).Info("Deleting team from remote", "id", teamId)
	deleteRequest := &cxsdk.DeleteTeamRequest{
		TeamId: &cxsdk.TeamID{
			Id: teamId,
		},
	}
	if _, err := r.TeamsClient.Delete(ctx, deleteRequest); err != nil && status.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote team", "id", teamId)
		return fmt.Errorf("error to delete remote team -\n%v", teamId)
	}
	log.V(1).Info("team was deleted from remote", "id", teamId)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Team{}).
		Complete(r)
}
