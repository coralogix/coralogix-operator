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

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// RecordingRuleGroupSetReconciler reconciles a RecordingRuleGroupSet object
type RecordingRuleGroupSetReconciler struct {
	RecordingRuleGroupSetClient clientset.RecordingRulesGroupsClientInterface
	RecordingRuleGroupSetSuffix string
}

//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/finalizers,verbs=update

func (r *RecordingRuleGroupSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.RecordingRuleGroupSet{}, r)
}

func (r *RecordingRuleGroupSetReconciler) FinalizerName() string {
	return "recordingrulegroupset.coralogix.com/finalizer"
}

func (r *RecordingRuleGroupSetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	recordingRuleGroupSet := obj.(*coralogixv1alpha1.RecordingRuleGroupSet)
	createRequest := &cxsdk.CreateRuleGroupSetRequest{
		Name:   ptr.To(fmt.Sprintf("%s%s", recordingRuleGroupSet.Name, r.RecordingRuleGroupSetSuffix)),
		Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
	}
	log.V(1).Info("Creating remote recordingRuleGroupSet", "recordingRuleGroupSet", protojson.Format(createRequest))
	createResponse, err := r.RecordingRuleGroupSetClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote recordingRuleGroupSet: %w", err)
	}
	log.V(1).Info("Remote recordingRuleGroupSet created", "response", protojson.Format(createResponse))
	recordingRuleGroupSet.Status = coralogixv1alpha1.RecordingRuleGroupSetStatus{
		ID: &createResponse.Id,
	}

	return nil
}

func (r *RecordingRuleGroupSetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	recordingRuleGroupSet := obj.(*coralogixv1alpha1.RecordingRuleGroupSet)
	updateRequest := &cxsdk.UpdateRuleGroupSetRequest{
		Id:     *recordingRuleGroupSet.Status.ID,
		Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
	}
	log.V(1).Info("Updating remote recordingRuleGroupSet", "recordingRuleGroupSet", protojson.Format(updateRequest))
	updateResponse, err := r.RecordingRuleGroupSetClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.V(1).Info("Remote recordingRuleGroupSet updated", "recordingRuleGroupSet", protojson.Format(updateResponse))
	return nil
}

func (r *RecordingRuleGroupSetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	recordingRuleGroupSet := obj.(*coralogixv1alpha1.RecordingRuleGroupSet)
	log.V(1).Info("Deleting recordingRuleGroupSet from remote system", "id", *recordingRuleGroupSet.Status.ID)
	_, err := r.RecordingRuleGroupSetClient.Delete(ctx, &cxsdk.DeleteRuleGroupSetRequest{Id: *recordingRuleGroupSet.Status.ID})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote recordingRuleGroupSet", "id", *recordingRuleGroupSet.Status.ID)
		return fmt.Errorf("error deleting remote recordingRuleGroupSet %s: %w", *recordingRuleGroupSet.Status.ID, err)
	}
	log.V(1).Info("RecordingRuleGroupSet deleted from remote system", "id", *recordingRuleGroupSet.Status.ID)
	return nil
}

func (r *RecordingRuleGroupSetReconciler) CheckIDInStatus(obj client.Object) bool {
	recordingRuleGroupSet := obj.(*coralogixv1alpha1.RecordingRuleGroupSet)
	return recordingRuleGroupSet.Status.ID != nil && *recordingRuleGroupSet.Status.ID != ""
}

func (r *RecordingRuleGroupSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RecordingRuleGroupSet{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
