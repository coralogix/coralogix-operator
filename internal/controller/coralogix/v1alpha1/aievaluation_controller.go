// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// AIEvaluationReconciler reconciles an AIEvaluation object.
type AIEvaluationReconciler struct {
	AIEvaluationsClient *aievaluations.AIEvaluationsServiceAPIService
	Interval            time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=aievaluations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=aievaluations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=aievaluations/finalizers,verbs=update

func (r *AIEvaluationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.AIEvaluation{}, r)
}

func (r *AIEvaluationReconciler) FinalizerName() string {
	return "ai-evaluation.coralogix.com/finalizer"
}

func (r *AIEvaluationReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *AIEvaluationReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	aiEvaluation := obj.(*coralogixv1alpha1.AIEvaluation)
	createRequest, err := aiEvaluation.ExtractCreateAIEvaluationRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create AIEvaluation request: %w", err)
	}

	log.Info("Creating remote AIEvaluation", "AIEvaluation", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.AIEvaluationsClient.
		AiEvaluationsServiceCreateAiEvaluation(ctx).
		AiEvaluationsServiceCreateAiEvaluationRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote AIEvaluation: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote AIEvaluation created", "response", utils.FormatJSON(createResponse))

	createdEvaluation := createResponse.GetAiEvaluation()
	id := (&createdEvaluation).GetId()
	if id == "" {
		return fmt.Errorf("remote AIEvaluation response did not include an id")
	}

	aiEvaluation.Status = coralogixv1alpha1.AIEvaluationStatus{
		Id: ptr.To(id),
	}

	return nil
}

func (r *AIEvaluationReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	aiEvaluation := obj.(*coralogixv1alpha1.AIEvaluation)
	updateRequest, err := aiEvaluation.ExtractUpdateAIEvaluationRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update AIEvaluation request: %w", err)
	}

	log.Info("Updating remote AIEvaluation", "id", ptr.Deref(aiEvaluation.Status.Id, ""))
	updateResponse, httpResp, err := r.AIEvaluationsClient.
		AiEvaluationsServiceUpdateAiEvaluation(ctx, ptr.Deref(aiEvaluation.Status.Id, "")).
		AiEvaluationsServiceUpdateAiEvaluationRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote AIEvaluation updated", "response", utils.FormatJSON(updateResponse))

	return nil
}

func (r *AIEvaluationReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	aiEvaluation := obj.(*coralogixv1alpha1.AIEvaluation)
	id := ptr.Deref(aiEvaluation.Status.Id, "")
	log.Info("Deleting AIEvaluation from remote system", "id", id)
	_, httpResp, err := r.AIEvaluationsClient.
		AiEvaluationsServiceDeleteAiEvaluation(ctx, id).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(apiErr, "Error deleting remote AIEvaluation", "id", id)
			return fmt.Errorf("error deleting remote AIEvaluation %s: %w", id, apiErr)
		}
	}
	log.Info("AIEvaluation deleted from remote system", "id", id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AIEvaluationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.AIEvaluation{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
