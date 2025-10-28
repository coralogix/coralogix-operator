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
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// RuleGroupReconciler reconciles a RuleGroup object
type RuleGroupReconciler struct {
	RuleGroupClient *cxsdk.RuleGroupsClient
	Interval        time.Duration
}

//+kubebuilder:rbac:groups=coralogix.com,resources=rulegroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=rulegroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=rulegroups/finalizers,verbs=update

func (r *RuleGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.RuleGroup{}, r)
}

func (r *RuleGroupReconciler) FinalizerName() string {
	return "rulegroup.coralogix.com/finalizer"
}

func (r *RuleGroupReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *RuleGroupReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	ruleGroup := obj.(*coralogixv1alpha1.RuleGroup)
	createRequest := ruleGroup.Spec.ExtractCreateRuleGroupRequest()
	log.Info("Creating remote ruleGroup", "ruleGroup", protojson.Format(createRequest))
	createResponse, err := r.RuleGroupClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote ruleGroup: %w", err)
	}
	log.Info("Remote ruleGroup created", "response", protojson.Format(createResponse))
	ruleGroup.Status = coralogixv1alpha1.RuleGroupStatus{
		ID: &createResponse.RuleGroup.Id.Value,
	}

	return nil
}

func (r *RuleGroupReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	ruleGroup := obj.(*coralogixv1alpha1.RuleGroup)
	updateRequest := ruleGroup.Spec.ExtractUpdateRuleGroupRequest(*ruleGroup.Status.ID)
	log.Info("Updating remote ruleGroup", "ruleGroup", protojson.Format(updateRequest))
	updateResponse, err := r.RuleGroupClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote ruleGroup updated", "ruleGroup", protojson.Format(updateResponse))
	return nil
}

func (r *RuleGroupReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	ruleGroup := obj.(*coralogixv1alpha1.RuleGroup)
	id := *ruleGroup.Status.ID
	log.Info("Deleting ruleGroup from remote system", "id", id)
	_, err := r.RuleGroupClient.Delete(ctx, &cxsdk.DeleteRuleGroupRequest{GroupId: id})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote ruleGroup", "id", id)
		return fmt.Errorf("error deleting remote ruleGroup %s: %w", id, err)
	}
	log.Info("RuleGroup deleted from remote system", "id", id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RuleGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RuleGroup{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
