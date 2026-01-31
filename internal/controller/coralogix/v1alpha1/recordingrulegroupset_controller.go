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
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	recordingrules "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/recording_rules_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// RecordingRuleGroupSetReconciler reconciles a RecordingRuleGroupSet object
type RecordingRuleGroupSetReconciler struct {
	RecordingRulesClient        *recordingrules.RecordingRulesServiceAPIService
	Interval                    time.Duration
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

func (r *RecordingRuleGroupSetReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *RecordingRuleGroupSetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	recordingRuleGroupSet := obj.(*coralogixv1alpha1.RecordingRuleGroupSet)
	createRequest := recordingrules.CreateRuleGroupSet{
		Name:   ptr.To(fmt.Sprintf("%s%s", recordingRuleGroupSet.Name, r.RecordingRuleGroupSetSuffix)),
		Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
	}
	log.Info("Creating remote recordingRuleGroupSet", "recordingRuleGroupSet", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.RecordingRulesClient.
		RuleGroupSetsCreate(ctx).
		CreateRuleGroupSet(createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote recordingRuleGroupSet: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote recordingRuleGroupSet created", "response", utils.FormatJSON(createResponse))
	recordingRuleGroupSet.Status = coralogixv1alpha1.RecordingRuleGroupSetStatus{
		ID: createResponse.Id,
	}

	return nil
}

func (r *RecordingRuleGroupSetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	recordingRuleGroupSet := obj.(*coralogixv1alpha1.RecordingRuleGroupSet)
	updateRequest := recordingrules.UpdateRuleGroupSet{
		Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
	}
	log.Info("Updating remote recordingRuleGroupSet", "recordingRuleGroupSet", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.RecordingRulesClient.
		RuleGroupSetsUpdate(ctx, *recordingRuleGroupSet.Status.ID).
		UpdateRuleGroupSet(updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote recordingRuleGroupSet updated", "recordingRuleGroupSet", utils.FormatJSON(updateResponse))
	return nil
}

func (r *RecordingRuleGroupSetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	recordingRuleGroupSet := obj.(*coralogixv1alpha1.RecordingRuleGroupSet)
	log.Info("Deleting recordingRuleGroupSet from remote system", "id", *recordingRuleGroupSet.Status.ID)
	_, httpResp, err := r.RecordingRulesClient.
		RuleGroupSetsDelete(ctx, *recordingRuleGroupSet.Status.ID).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote recordingRuleGroupSet", "id", *recordingRuleGroupSet.Status.ID)
			return fmt.Errorf("error deleting remote recordingRuleGroupSet %s: %w", *recordingRuleGroupSet.Status.ID, apiErr)
		}
	}
	log.Info("RecordingRuleGroupSet deleted from remote system", "id", *recordingRuleGroupSet.Status.ID)
	return nil
}

func (r *RecordingRuleGroupSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RecordingRuleGroupSet{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
