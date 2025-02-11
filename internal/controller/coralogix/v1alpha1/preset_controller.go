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

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
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
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// PresetReconciler reconciles a Preset object
type PresetReconciler struct {
	client.Client
	NotificationsClient *cxsdk.NotificationsClient
	Scheme              *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=presets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=presets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=presets/finalizers,verbs=update

var (
	presetFinalizerName = "preset.coralogix.com/finalizer"
)

func (r *PresetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"preset", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	preset := &coralogixv1alpha1.Preset{}
	if err := r.Client.Get(ctx, req.NamespacedName, preset); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(preset.Status.Id, "") == "" {
		err := r.create(ctx, log, preset)
		if err != nil {
			log.Error(err, "Error on creating Preset")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !preset.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, preset)
		if err != nil {
			log.Error(err, "Error on deleting Preset")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(preset.GetLabels()) {
		err := r.deleteRemotePreset(ctx, log, preset.Status.Id)
		if err != nil {
			log.Error(err, "Error on deleting Preset")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, preset)
	if err != nil {
		log.Error(err, "Error on updating Preset")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *PresetReconciler) create(ctx context.Context, log logr.Logger, preset *coralogixv1alpha1.Preset) error {
	createReq := preset.Spec.ExtractCreatePresetRequest()
	log.V(1).Info("Creating remote preset", "preset", protojson.Format(createReq))
	createRes, err := r.NotificationsClient.CreateCustomPreset(ctx, createReq)
	if err != nil {
		return fmt.Errorf("error on creating remote preset: %w", err)
	}
	log.V(1).Info("Remote preset created", "response", protojson.Format(createRes))

	preset.Status = coralogixv1alpha1.PresetStatus{
		Id: createRes.Preset.Id,
	}

	log.V(1).Info("Updating Preset status", "id", createRes.Preset.Id)
	if err = r.Status().Update(ctx, preset); err != nil {
		if err := r.deleteRemotePreset(ctx, log, preset.Status.Id); err != nil {
			return fmt.Errorf("error to delete preset after status update error -\n%v", preset)
		}
		return fmt.Errorf("error to update preset status -\n%v", preset)
	}

	if !controllerutil.ContainsFinalizer(preset, presetFinalizerName) {
		log.V(1).Info("Updating Preset to add finalizer", "id", createRes.Preset.Id)
		controllerutil.AddFinalizer(preset, presetFinalizerName)
		if err := r.Update(ctx, preset); err != nil {
			return fmt.Errorf("error on updating Preset: %w", err)
		}
	}

	return nil
}

func (r *PresetReconciler) update(ctx context.Context, log logr.Logger, preset *coralogixv1alpha1.Preset) error {
	replaceReq := preset.Spec.ExtractReplacePresetRequest(preset.Status.Id)
	log.V(1).Info("Updating remote preset", "preset", protojson.Format(replaceReq))
	replaceRes, err := r.NotificationsClient.ReplaceCustomPreset(ctx, replaceReq)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("preset not found on remote, removing id from status")
			preset.Status = coralogixv1alpha1.PresetStatus{
				Id: ptr.To(""),
			}
			if err = r.Status().Update(ctx, preset); err != nil {
				return fmt.Errorf("error on updating Preset status: %w", err)
			}
			return fmt.Errorf("preset not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating preset: %w", err)
	}
	log.V(1).Info("Remote preset updated", "preset", protojson.Format(replaceRes))

	return nil
}

func (r *PresetReconciler) delete(ctx context.Context, log logr.Logger, preset *coralogixv1alpha1.Preset) error {
	if err := r.deleteRemotePreset(ctx, log, preset.Status.Id); err != nil {
		return fmt.Errorf("error on deleting remote preset: %w", err)
	}

	log.V(1).Info("Removing finalizer from Preset")
	controllerutil.RemoveFinalizer(preset, presetFinalizerName)
	if err := r.Update(ctx, preset); err != nil {
		return fmt.Errorf("error on updating Preset: %w", err)
	}

	return nil
}

func (r *PresetReconciler) deleteRemotePreset(ctx context.Context, log logr.Logger, id *string) error {
	log.V(1).Info("Deleting remote preset", "id", *id)
	_, err := r.NotificationsClient.DeleteCustomPreset(ctx, &cxsdk.DeleteCustomPresetRequest{
		Identifier: &cxsdk.PresetIdentifier{
			Value: &cxsdk.PresetIdentifierIDValue{
				Id: *id,
			},
		},
	})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		return fmt.Errorf("error on deleting remote preset: %w", err)
	}
	log.V(1).Info("Remote preset deleted", "id", *id)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PresetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Preset{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
