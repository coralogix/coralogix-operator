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

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	presets "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/presets_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// PresetReconciler reconciles a Preset object
type PresetReconciler struct {
	PresetsClient *presets.PresetsServiceAPIService
	Interval      time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=presets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=presets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=presets/finalizers,verbs=update

func (r *PresetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Preset{}, r)
}

func (r *PresetReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *PresetReconciler) FinalizerName() string {
	return "preset.coralogix.com/finalizer"
}

func (r *PresetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	preset := obj.(*coralogixv1alpha1.Preset)
	createRequest, err := preset.ExtractPreset()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}

	log.Info("Creating remote Preset", "Preset", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.PresetsClient.
		PresetsServiceCreateCustomPreset(ctx).
		Preset1(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote Preset: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote preset created", "response", utils.FormatJSON(createResponse))

	preset.Status = coralogixv1alpha1.PresetStatus{
		Id: createResponse.Preset.Id,
	}

	return nil
}

func (r *PresetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	preset := obj.(*coralogixv1alpha1.Preset)
	updateRequest, err := preset.ExtractPreset()
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	updateRequest.Id = preset.Status.Id
	log.Info("Updating remote Preset", "Preset", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.PresetsClient.
		PresetsServiceReplaceCustomPreset(ctx).
		Preset1(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote Preset updated", "Preset", utils.FormatJSON(updateResponse))

	return nil
}

func (r *PresetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	preset := obj.(*coralogixv1alpha1.Preset)
	log.Info("Deleting Preset from remote system", "id", *preset.Status.Id)
	_, httpResp, err := r.PresetsClient.
		PresetsServiceDeleteCustomPreset(ctx, ptr.Deref(preset.Status.Id, "")).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote Preset", "id", *preset.Status.Id)
			return fmt.Errorf("error deleting remote Preset %s: %w", *preset.Status.Id, apiErr)
		}
	}
	log.Info("Preset deleted from remote system", "id", *preset.Status.Id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PresetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Preset{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
