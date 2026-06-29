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
	"sort"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	aiapplications "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_applications_service"
	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// AICustomEvaluationReconciler reconciles an AICustomEvaluation object.
type AICustomEvaluationReconciler struct {
	AIApplicationsClient *aiapplications.AIApplicationsServiceAPIService
	AIEvaluationsClient  *aievaluations.AIEvaluationsServiceAPIService
	Interval             time.Duration
}

type aiCustomEvaluationApplicationKey struct {
	Application string
	Subsystem   string
}

// +kubebuilder:rbac:groups=coralogix.com,resources=aicustomevaluations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=aicustomevaluations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=aicustomevaluations/finalizers,verbs=update

func (r *AICustomEvaluationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.AICustomEvaluation{}, r)
}

func (r *AICustomEvaluationReconciler) FinalizerName() string {
	return "ai-custom-evaluation.coralogix.com/finalizer"
}

func (r *AICustomEvaluationReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *AICustomEvaluationReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	aiCustomEvaluation := obj.(*coralogixv1alpha1.AICustomEvaluation)
	applications, err := r.resolveApplications(ctx, aiCustomEvaluation.Spec.Applications)
	if err != nil {
		return err
	}

	createRequest, err := aiCustomEvaluation.ExtractCreateAICustomEvaluationRequest(aiCustomEvaluationApplicationIDs(applications))
	if err != nil {
		return fmt.Errorf("error on extracting create AICustomEvaluation request: %w", err)
	}

	log.Info("Creating remote AICustomEvaluation", "AICustomEvaluation", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.AIEvaluationsClient.
		AiEvaluationsServiceCreateCustomEvaluation(ctx).
		AiEvaluationsServiceCreateCustomEvaluationRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote AICustomEvaluation: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote AICustomEvaluation created", "response", utils.FormatJSON(createResponse))

	createdEvaluation := createResponse.GetItem()
	id := (&createdEvaluation).GetId()
	if id == "" {
		return fmt.Errorf("remote AICustomEvaluation response did not include an id")
	}

	aiCustomEvaluation.Status = coralogixv1alpha1.AICustomEvaluationStatus{
		Id:             ptr.To(id),
		ApplicationIds: aiCustomEvaluationApplicationIDs(applications),
		Applications:   applications,
	}

	return nil
}

func (r *AICustomEvaluationReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	aiCustomEvaluation := obj.(*coralogixv1alpha1.AICustomEvaluation)
	applications, err := r.resolveApplications(ctx, aiCustomEvaluation.Spec.Applications)
	if err != nil {
		return err
	}

	updateRequest, err := aiCustomEvaluation.ExtractUpdateAICustomEvaluationRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update AICustomEvaluation request: %w", err)
	}

	id := ptr.Deref(aiCustomEvaluation.Status.Id, "")
	log.Info("Updating remote AICustomEvaluation", "id", id)
	updateResponse, httpResp, err := r.AIEvaluationsClient.
		AiEvaluationsServiceUpdateCustomEvaluation(ctx, id).
		AiEvaluationsServiceUpdateCustomEvaluationRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote AICustomEvaluation updated", "response", utils.FormatJSON(updateResponse))

	examplesUpdateRequest, err := aiCustomEvaluation.ExtractUpdateAICustomEvaluationExamplesRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update AICustomEvaluation examples request: %w", err)
	}

	if examplesUpdateRequest != nil {
		log.Info("Clearing remote AICustomEvaluation examples", "id", id)
		examplesUpdateResponse, httpResp, err := r.AIEvaluationsClient.
			AiEvaluationsServiceUpdateCustomEvaluation(ctx, id).
			AiEvaluationsServiceUpdateCustomEvaluationRequest(*examplesUpdateRequest).
			Execute()
		if err != nil {
			return cxsdk.NewAPIError(httpResp, err)
		}
		log.Info("Remote AICustomEvaluation examples cleared", "response", utils.FormatJSON(examplesUpdateResponse))
	}

	if err := r.reconcileApplicationLinks(ctx, id, currentAICustomEvaluationApplicationIDs(aiCustomEvaluation.Status), aiCustomEvaluationApplicationIDs(applications)); err != nil {
		return err
	}

	aiCustomEvaluation.Status.ApplicationIds = aiCustomEvaluationApplicationIDs(applications)
	aiCustomEvaluation.Status.Applications = applications

	return nil
}

func (r *AICustomEvaluationReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	aiCustomEvaluation := obj.(*coralogixv1alpha1.AICustomEvaluation)
	id := ptr.Deref(aiCustomEvaluation.Status.Id, "")
	log.Info("Deleting AICustomEvaluation from remote system", "id", id)
	_, httpResp, err := r.AIEvaluationsClient.
		AiEvaluationsServiceDeleteCustomEvaluation(ctx, id).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			return fmt.Errorf("error deleting remote AICustomEvaluation %s: %w", id, apiErr)
		}
	}
	log.Info("AICustomEvaluation deleted from remote system", "id", id)
	return nil
}

func (r *AICustomEvaluationReconciler) resolveApplications(
	ctx context.Context,
	selectors []coralogixv1alpha1.AICustomEvaluationApplicationSelector,
) ([]coralogixv1alpha1.AICustomEvaluationApplicationStatus, error) {
	if len(selectors) == 0 {
		return []coralogixv1alpha1.AICustomEvaluationApplicationStatus{}, nil
	}

	allApplications, err := r.listAIApplications(ctx)
	if err != nil {
		return nil, err
	}

	applicationsBySelector := make(map[aiCustomEvaluationApplicationKey][]coralogixv1alpha1.AICustomEvaluationApplicationStatus, len(allApplications))
	for _, application := range allApplications {
		if application.Id == "" || application.Application == "" {
			continue
		}
		key := aiCustomEvaluationApplicationKey{
			Application: application.Application,
			Subsystem:   application.Subsystem,
		}
		applicationsBySelector[key] = append(applicationsBySelector[key], application)
	}

	resolved := make([]coralogixv1alpha1.AICustomEvaluationApplicationStatus, 0, len(selectors))
	for _, selector := range selectors {
		matches := applicationsBySelector[aiCustomEvaluationApplicationKey{
			Application: selector.Application,
			Subsystem:   selector.Subsystem,
		}]
		if len(matches) == 0 {
			return nil, fmt.Errorf("AI application not found: no AI application named %q with subsystem %q was found", selector.Application, selector.Subsystem)
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("ambiguous AI application selector: found %d AI applications named %q with subsystem %q", len(matches), selector.Application, selector.Subsystem)
		}
		resolved = append(resolved, matches[0])
	}

	sortAICustomEvaluationApplications(resolved)
	return resolved, nil
}

func (r *AICustomEvaluationReconciler) listAIApplications(ctx context.Context) ([]coralogixv1alpha1.AICustomEvaluationApplicationStatus, error) {
	const pageSize = int32(200)
	var applications []coralogixv1alpha1.AICustomEvaluationApplicationStatus
	for pageOffset := int64(0); ; pageOffset++ {
		resp, httpResp, err := r.AIApplicationsClient.
			AiApplicationsServiceListAiApplications(ctx).
			PageSize(pageSize).
			PageOffset(pageOffset).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("error on listing AI applications: %w", cxsdk.NewAPIError(httpResp, err))
		}

		page := resp.GetAiApplications()
		for _, application := range page {
			applications = append(applications, coralogixv1alpha1.AICustomEvaluationApplicationStatus{
				Id:          application.GetId(),
				Application: application.GetApplication(),
				Subsystem:   application.GetSubsystem(),
			})
		}
		if len(page) < int(pageSize) {
			break
		}
	}

	return applications, nil
}

func (r *AICustomEvaluationReconciler) reconcileApplicationLinks(ctx context.Context, customEvaluationID string, currentApplicationIDs []string, desiredApplicationIDs []string) error {
	current := aiCustomEvaluationStringSet(currentApplicationIDs)
	desired := aiCustomEvaluationStringSet(desiredApplicationIDs)

	toLink := make([]string, 0)
	for id := range desired {
		if _, ok := current[id]; !ok {
			toLink = append(toLink, id)
		}
	}
	sort.Strings(toLink)

	toUnlink := make([]string, 0)
	for id := range current {
		if _, ok := desired[id]; !ok {
			toUnlink = append(toUnlink, id)
		}
	}
	sort.Strings(toUnlink)

	for _, id := range toLink {
		_, httpResp, err := r.AIEvaluationsClient.
			AiEvaluationsServiceLinkCustomEvaluation(ctx, customEvaluationID, id).
			Execute()
		if err != nil {
			return fmt.Errorf("error linking AI application %q to AICustomEvaluation %q: %w", id, customEvaluationID, cxsdk.NewAPIError(httpResp, err))
		}
	}

	for _, id := range toUnlink {
		_, httpResp, err := r.AIEvaluationsClient.
			AiEvaluationsServiceUnlinkCustomEvaluationFromApp(ctx, customEvaluationID, id).
			Execute()
		if err != nil {
			return fmt.Errorf("error unlinking AI application %q from AICustomEvaluation %q: %w", id, customEvaluationID, cxsdk.NewAPIError(httpResp, err))
		}
	}

	return nil
}

func aiCustomEvaluationApplicationIDs(applications []coralogixv1alpha1.AICustomEvaluationApplicationStatus) []string {
	ids := make([]string, 0, len(applications))
	for _, application := range applications {
		ids = append(ids, application.Id)
	}
	sort.Strings(ids)
	return ids
}

func currentAICustomEvaluationApplicationIDs(status coralogixv1alpha1.AICustomEvaluationStatus) []string {
	if len(status.ApplicationIds) > 0 {
		ids := append([]string(nil), status.ApplicationIds...)
		sort.Strings(ids)
		return ids
	}

	return aiCustomEvaluationApplicationIDs(status.Applications)
}

func sortAICustomEvaluationApplications(applications []coralogixv1alpha1.AICustomEvaluationApplicationStatus) {
	sort.Slice(applications, func(i, j int) bool {
		if applications[i].Id == applications[j].Id {
			if applications[i].Application == applications[j].Application {
				return applications[i].Subsystem < applications[j].Subsystem
			}
			return applications[i].Application < applications[j].Application
		}
		return applications[i].Id < applications[j].Id
	})
}

func aiCustomEvaluationStringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *AICustomEvaluationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.AICustomEvaluation{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
