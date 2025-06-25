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
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// PresetReconciler reconciles a Preset object
type PresetReconciler struct {
	NotificationsClient *cxsdk.NotificationsClient
	Interval            time.Duration
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
	createRequest, err := preset.ExtractCreateCustomPresetRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}

	log.Info("Creating remote Preset", "Preset", protojson.Format(createRequest))
	createResponse, err := r.NotificationsClient.CreateCustomPreset(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote Preset: %w", err)
	}
	log.Info("Remote preset created", "response", protojson.Format(createResponse))

	preset.Status = coralogixv1alpha1.PresetStatus{
		Id: createResponse.Preset.Id,
	}

	return nil
}

func (r *PresetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	preset := obj.(*coralogixv1alpha1.Preset)
	updateRequest, err := preset.ExtractUpdateCustomPresetRequest()
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote Preset", "Preset", protojson.Format(updateRequest))
	updateResponse, err := r.NotificationsClient.ReplaceCustomPreset(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote Preset updated", "Preset", protojson.Format(updateResponse))

	return nil
}

func (r *PresetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	preset := obj.(*coralogixv1alpha1.Preset)
	log.Info("Deleting Preset from remote system", "id", *preset.Status.Id)
	_, err := r.NotificationsClient.DeleteCustomPreset(ctx,
		&cxsdk.DeleteCustomPresetRequest{
			Id: ptr.Deref(preset.Status.Id, ""),
		})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote Preset", "id", *preset.Status.Id)
		return fmt.Errorf("error deleting remote Preset %s: %w", *preset.Status.Id, err)
	}
	log.Info("Preset deleted from remote system", "id", *preset.Status.Id)
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
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
