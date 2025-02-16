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
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/coralogix/coralogix-operator/internal/utils"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PresetReconciler reconciles a Preset object
type PresetReconciler struct {
	NotificationsClient *cxsdk.NotificationsClient
}

// +kubebuilder:rbac:groups=coralogix.com,resources=presets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=presets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=presets/finalizers,verbs=update

var (
	presetFinalizerName = "preset.coralogix.com/finalizer"
)

func (r *PresetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogix.ReconcileResource(ctx, req, &coralogixv1alpha1.Preset{}, r)
}

func (r *PresetReconciler) FinalizerName() string {
	return "preset.coralogix.com/finalizer"
}

func (r *PresetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error) {
	preset := obj.(*coralogixv1alpha1.Preset)
	createRequest := preset.Spec.ExtractCreatePresetRequest()
	log.V(1).Info("Creating remote preset", "preset", protojson.Format(createRequest))
	createResponse, err := r.NotificationsClient.CreateCustomPreset(ctx, createRequest)
	if err != nil {
		return nil, fmt.Errorf("error on creating remote preset: %w", err)
	}
	log.V(1).Info("Remote preset created", "response", protojson.Format(createResponse))

	preset.Status = coralogixv1alpha1.PresetStatus{
		Id: createResponse.Preset.Id,
	}

	return preset, nil
}

func (r *PresetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	preset := obj.(*coralogixv1alpha1.Preset)
	updateRequest := preset.Spec.ExtractReplacePresetRequest(preset.Status.Id)
	log.V(1).Info("Updating remote preset", "preset", protojson.Format(updateRequest))
	updateResponse, err := r.NotificationsClient.ReplaceCustomPreset(ctx, updateRequest)
	if err != nil {
		return fmt.Errorf("error on updating remote preset: %w", err)
	}
	log.V(1).Info("Remote preset updated", "preset", protojson.Format(updateResponse))

	return nil
}

func (r *PresetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	preset := obj.(*coralogixv1alpha1.Preset)
	log.V(1).Info("Deleting preset from remote system", "id", *preset.Status.Id)
	_, err := r.NotificationsClient.DeleteCustomPreset(ctx, &cxsdk.DeleteCustomPresetRequest{Identifier: &cxsdk.PresetIdentifier{Value: &cxsdk.PresetIdentifierIDValue{Id: *preset.Status.Id}}})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote preset", "id", *preset.Status.Id)
		return fmt.Errorf("error deleting remote preset %s: %w", *preset.Status.Id, err)
	}
	log.V(1).Info("Preset deleted from remote system", "id", *preset.Status.Id)
	return nil
}

func (r *PresetReconciler) CheckIDInStatus(obj client.Object) bool {
	preset := obj.(*coralogixv1alpha1.Preset)
	return preset.Status.Id != nil && *preset.Status.Id != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *PresetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Preset{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
