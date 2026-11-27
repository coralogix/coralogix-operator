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
	"slices"
	"sort"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	aievaluations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ai_evaluations_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// The hooks of AICustomEvaluation do what the API contract cannot state:
//   - spec.criteria is sent as the fields safe and violates, and one list of examples that carry a
//     score: "1" for an acceptable example, "0" for a prohibited one.
//   - spec.applications names AI applications by application and subsystem. The API links them
//     by id: Create sends the ids, and Update links and unlinks them with calls of their own,
//     after it reads the links from the List, since the Update answer holds none.
//   - Update sends no update mask, so an empty examples list keeps the stored examples. When the
//     CR has none, a second Update with the mask "examples" clears them.

const (
	aiCustomEvaluationAcceptableScore = "1"
	aiCustomEvaluationProhibitedScore = "0"
	aiCustomEvaluationExamplesMask    = "examples"
	aiApplicationsPageSize            = int32(200)
)

// aiCustomEvaluationLinks holds the application ids that beforeUpdate resolved, for afterUpdate of
// the same reconcile: the selectors are resolved before the Update, as the released kind did, so a
// selector that matches no application fails before any write. The key is the UID of the CR, and
// the controller never reconciles one CR twice at the same time.
var aiCustomEvaluationLinks sync.Map

// beforeCreate sends the criteria and the application ids.
func (r *AICustomEvaluationReconciler) beforeCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.AICustomEvaluation, body *aievaluations.AiEvaluationsServiceCreateCustomEvaluationRequest) error {
	ids, err := r.resolveAICustomEvaluationApplications(ctx, cr.Spec.Applications)
	if err != nil {
		return err
	}
	safe, violates, examples := aiCustomEvaluationCriteria(cr.Spec.Criteria)
	body.Safe, body.Violates, body.Examples = &safe, &violates, examples
	body.ApplicationIds = append([]string(nil), ids...)
	return nil
}

// beforeUpdate sends the criteria, and resolves the applications for afterUpdate.
func (r *AICustomEvaluationReconciler) beforeUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.AICustomEvaluation, body *aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest) error {
	ids, err := r.resolveAICustomEvaluationApplications(ctx, cr.Spec.Applications)
	if err != nil {
		return err
	}
	aiCustomEvaluationLinks.Store(cr.UID, ids)
	safe, violates, examples := aiCustomEvaluationCriteria(cr.Spec.Criteria)
	body.Safe, body.Violates, body.Examples = &safe, &violates, examples
	return nil
}

// afterUpdate clears the examples when the CR has none, then links and unlinks the applications.
func (r *AICustomEvaluationReconciler) afterUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.AICustomEvaluation, _ *aievaluations.UpdateCustomEvaluationResponse) error {
	id := ptr.Deref(cr.Status.ID, "")
	// Take the ids first, so a failed call below leaves no entry behind.
	desired, err := r.resolvedAICustomEvaluationLinks(ctx, cr.UID, cr.Spec.Applications)
	if err != nil {
		return err
	}
	if _, _, examples := aiCustomEvaluationCriteria(cr.Spec.Criteria); len(examples) == 0 {
		clearExamples := aievaluations.AiEvaluationsServiceUpdateCustomEvaluationRequest{
			Examples:   examples,
			UpdateMask: ptr.To(aiCustomEvaluationExamplesMask),
		}
		_, httpResp, err := r.Client.AiEvaluationsServiceUpdateCustomEvaluation(ctx, id).AiEvaluationsServiceUpdateCustomEvaluationRequest(clearExamples).Execute()
		if err != nil {
			return fmt.Errorf("error on clearing the examples of AICustomEvaluation %s: %w", id, cxsdk.NewAPIError(httpResp, err))
		}
	}
	current, err := r.linkedAICustomEvaluationApplications(ctx, id)
	if err != nil {
		return err
	}
	return r.reconcileAICustomEvaluationLinks(ctx, id, current, desired)
}

// resolvedAICustomEvaluationLinks returns the application ids that beforeUpdate resolved, or
// resolves them when it has none.
func (r *AICustomEvaluationReconciler) resolvedAICustomEvaluationLinks(ctx context.Context, uid types.UID, selectors coralogixv1alpha1.AICustomEvaluationApplications) ([]string, error) {
	if ids, ok := aiCustomEvaluationLinks.LoadAndDelete(uid); ok {
		return ids.([]string), nil
	}
	return r.resolveAICustomEvaluationApplications(ctx, selectors)
}

// aiCustomEvaluationCriteria returns safe, violates, and the examples of the criteria: the
// acceptable examples first, then the prohibited ones. The examples are never nil, so a request
// always holds the list.
func aiCustomEvaluationCriteria(criteria *coralogixv1alpha1.AICustomEvaluationCriteria) (string, string, []aievaluations.CustomEvaluationExample) {
	var acceptable, prohibited coralogixv1alpha1.AICustomEvaluationCriterion
	if criteria != nil && criteria.Acceptable != nil {
		acceptable = *criteria.Acceptable
	}
	if criteria != nil && criteria.Prohibited != nil {
		prohibited = *criteria.Prohibited
	}
	examples := make([]aievaluations.CustomEvaluationExample, 0, len(acceptable.Examples)+len(prohibited.Examples))
	for _, example := range acceptable.Examples {
		examples = append(examples, aievaluations.CustomEvaluationExample{Conversation: ptr.To(example), Score: ptr.To(aiCustomEvaluationAcceptableScore)})
	}
	for _, example := range prohibited.Examples {
		examples = append(examples, aievaluations.CustomEvaluationExample{Conversation: ptr.To(example), Score: ptr.To(aiCustomEvaluationProhibitedScore)})
	}
	return ptr.Deref(acceptable.Flags, ""), ptr.Deref(prohibited.Flags, ""), examples
}

type aiApplicationSelectorKey struct {
	application, subsystem string
}

// resolveAICustomEvaluationApplications returns the sorted ids of the AI applications that the
// selectors name. Each selector must match exactly one application of the team.
func (r *AICustomEvaluationReconciler) resolveAICustomEvaluationApplications(ctx context.Context, selectors coralogixv1alpha1.AICustomEvaluationApplications) ([]string, error) {
	if len(selectors) == 0 {
		return []string{}, nil
	}
	ids, err := r.listAIApplications(ctx)
	if err != nil {
		return nil, err
	}
	resolved := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		matches := ids[aiApplicationSelectorKey{selector.Application, selector.Subsystem}]
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("AI application not found: no AI application named %q with subsystem %q was found", selector.Application, selector.Subsystem)
		case 1:
			resolved = append(resolved, matches[0])
		default:
			return nil, fmt.Errorf("ambiguous AI application selector: found %d AI applications named %q with subsystem %q", len(matches), selector.Application, selector.Subsystem)
		}
	}
	sort.Strings(resolved)
	return resolved, nil
}

// listAIApplications returns the ids of the AI applications of the team by application and
// subsystem. page_offset counts applications, so each page starts after the ones read before it.
func (r *AICustomEvaluationReconciler) listAIApplications(ctx context.Context) (map[aiApplicationSelectorKey][]string, error) {
	ids := map[aiApplicationSelectorKey][]string{}
	for offset := int64(0); ; offset += int64(aiApplicationsPageSize) {
		resp, httpResp, err := r.ClientSet.AIApplications().AiApplicationsServiceListAiApplications(ctx).
			PageSize(aiApplicationsPageSize).
			PageOffset(offset).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("error on listing AI applications: %w", cxsdk.NewAPIError(httpResp, err))
		}
		page := resp.GetAiApplications()
		for _, application := range page {
			id, name := application.GetId(), application.GetApplication()
			if id == "" || name == "" {
				continue
			}
			key := aiApplicationSelectorKey{name, application.GetSubsystem()}
			if !slices.Contains(ids[key], id) {
				ids[key] = append(ids[key], id)
			}
		}
		if len(page) < int(aiApplicationsPageSize) {
			return ids, nil
		}
	}
}

// linkedAICustomEvaluationApplications reads the application ids that the custom evaluation links
// to from the List: no call reads one custom evaluation.
func (r *AICustomEvaluationReconciler) linkedAICustomEvaluationApplications(ctx context.Context, id string) ([]string, error) {
	resp, httpResp, err := r.Client.AiEvaluationsServiceGetCustomEvaluations(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("error on listing remote AICustomEvaluations: %w", cxsdk.NewAPIError(httpResp, err))
	}
	for _, item := range resp.GetItems() {
		if item.GetId() == id {
			return item.GetApplicationIds(), nil
		}
	}
	return nil, fmt.Errorf("remote AICustomEvaluation %q not found", id)
}

// reconcileAICustomEvaluationLinks links the applications that are new, then unlinks the ones that
// are no longer wanted, each in the order of their ids.
func (r *AICustomEvaluationReconciler) reconcileAICustomEvaluationLinks(ctx context.Context, id string, current, desired []string) error {
	var toLink, toUnlink []string
	for _, applicationID := range desired {
		if !slices.Contains(current, applicationID) {
			toLink = append(toLink, applicationID)
		}
	}
	for _, applicationID := range current {
		if !slices.Contains(desired, applicationID) {
			toUnlink = append(toUnlink, applicationID)
		}
	}
	sort.Strings(toLink)
	sort.Strings(toUnlink)
	for _, applicationID := range toLink {
		_, httpResp, err := r.Client.AiEvaluationsServiceLinkCustomEvaluation(ctx, applicationID, id).Execute()
		if err != nil {
			return fmt.Errorf("error linking AI application %q to AICustomEvaluation %q: %w", applicationID, id, cxsdk.NewAPIError(httpResp, err))
		}
	}
	for _, applicationID := range toUnlink {
		_, httpResp, err := r.Client.AiEvaluationsServiceUnlinkCustomEvaluationFromApp(ctx, applicationID, id).Execute()
		if err != nil {
			return fmt.Errorf("error unlinking AI application %q from AICustomEvaluation %q: %w", applicationID, id, cxsdk.NewAPIError(httpResp, err))
		}
	}
	return nil
}
