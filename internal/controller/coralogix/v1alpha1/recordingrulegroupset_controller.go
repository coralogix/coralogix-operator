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

	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
)

var recordingRuleGroupSetFinalizerName = "recordingrulegroupset.coralogix.com/finalizer"

// RecordingRuleGroupSetReconciler reconciles a RecordingRuleGroupSet object
type RecordingRuleGroupSetReconciler struct {
	client.Client
	RecordingRuleGroupSetClient *cxsdk.RecordingRuleGroupSetsClient
	Scheme                      *runtime.Scheme
	RecordingRuleGroupSetSuffix string
}

//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/finalizers,verbs=update

func (r *RecordingRuleGroupSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"recordingRuleGroupSet", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	// recordingRuleClient := r.CoralogixClientSet.RecordingRuleGroups()

	recordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, recordingRuleGroupSet); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(recordingRuleGroupSet.Status.ID, "") == "" {
		if err := r.create(ctx, recordingRuleGroupSet); err != nil {
			log.Error(err, "Failed to create RecordingRuleGroupSet", "error", err)
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		monitoring.RecordingRuleGroupSetInfoMetric.WithLabelValues(recordingRuleGroupSet.Name, recordingRuleGroupSet.Namespace).Set(1)
		return ctrl.Result{}, nil
	}

	if !recordingRuleGroupSet.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.delete(ctx, recordingRuleGroupSet); err != nil {
			log.Error(err, "Failed to delete RecordingRuleGroupSet", "error", err)
			return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
		}
		monitoring.RecordingRuleGroupSetInfoMetric.DeleteLabelValues(recordingRuleGroupSet.Name, recordingRuleGroupSet.Namespace)
		return ctrl.Result{}, nil
	}

	if err := r.update(ctx, recordingRuleGroupSet); err != nil {
		log.Error(err, "Failed to update RecordingRuleGroupSet", "error", err)
		return ctrl.Result{RequeueAfter: coralogix.DefaultErrRequeuePeriod}, err
	}
	monitoring.RecordingRuleGroupSetInfoMetric.WithLabelValues(recordingRuleGroupSet.Name, recordingRuleGroupSet.Namespace).Set(1)

	return ctrl.Result{}, nil
}

func (r *RecordingRuleGroupSetReconciler) create(ctx context.Context, recordingRuleGroupSet *coralogixv1alpha1.RecordingRuleGroupSet) error {
	response, err := r.RecordingRuleGroupSetClient.
		Create(ctx, &cxsdk.CreateRuleGroupSetRequest{
			Name:   ptr.To(fmt.Sprintf("%s%s", recordingRuleGroupSet.Name, r.RecordingRuleGroupSetSuffix)),
			Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
		})

	if err != nil {
		return fmt.Errorf("failed to create recording rule groupSet: %w", err)
	}

	recordingRuleGroupSet.Status.ID = ptr.To(response.Id)

	if err := r.Status().Update(ctx, recordingRuleGroupSet); err != nil {
		if err := r.deleteRemoteRecordingRuleGroupSet(ctx, recordingRuleGroupSet.Status.ID); err != nil {
			return fmt.Errorf("failed to delete recording rule groupSet after status update error: %w", err)
		}
		return fmt.Errorf("failed to update recording rule groupSet status: %w", err)
	}

	if !controllerutil.ContainsFinalizer(recordingRuleGroupSet, recordingRuleGroupSetFinalizerName) {
		controllerutil.AddFinalizer(recordingRuleGroupSet, recordingRuleGroupSetFinalizerName)
	}

	if err := r.Client.Update(ctx, recordingRuleGroupSet); err != nil {
		return fmt.Errorf("failed to update recording rule groupSet: %w", err)
	}

	return nil
}

func (r *RecordingRuleGroupSetReconciler) update(ctx context.Context, recordingRuleGroupSet *coralogixv1alpha1.RecordingRuleGroupSet) error {
	remoteRecordingRule, err := r.RecordingRuleGroupSetClient.Get(ctx, &cxsdk.GetRuleGroupSetRequest{
		Id: *recordingRuleGroupSet.Status.ID,
	})

	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			recordingRuleGroupSet.Status.ID = nil
			if err := r.Status().Update(ctx, recordingRuleGroupSet); err != nil {
				return fmt.Errorf("failed to update recording rule groupSet status: %w", err)
			}
			return err
		}
		return fmt.Errorf("failed to get recording rule groupSet: %w", err)
	}

	if _, err := r.RecordingRuleGroupSetClient.
		Update(ctx, &cxsdk.UpdateRuleGroupSetRequest{
			Id:     remoteRecordingRule.Id,
			Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
		}); err != nil {
		return fmt.Errorf("failed to update recording rule groupSet: %w", err)
	}

	return nil
}

func (r *RecordingRuleGroupSetReconciler) delete(ctx context.Context, recordingRuleGroupSet *coralogixv1alpha1.RecordingRuleGroupSet) error {
	if err := r.deleteRemoteRecordingRuleGroupSet(ctx, recordingRuleGroupSet.Status.ID); err != nil {
		return fmt.Errorf("failed to delete recording rule groupSet: %w", err)
	}

	controllerutil.RemoveFinalizer(recordingRuleGroupSet, recordingRuleGroupSetFinalizerName)
	if err := r.Update(ctx, recordingRuleGroupSet); err != nil {
		return fmt.Errorf("failed to remove finalizer from recording rule groupSet: %w", err)
	}

	return nil
}

func (r *RecordingRuleGroupSetReconciler) deleteRemoteRecordingRuleGroupSet(ctx context.Context, id *string) error {
	if _, err := r.RecordingRuleGroupSetClient.Delete(ctx, &cxsdk.DeleteRuleGroupSetRequest{
		Id: *id}); err != nil && cxsdk.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to delete recording rule groupSet: %w", err)
	}
	return nil
}

func (r *RecordingRuleGroupSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RecordingRuleGroupSet{}).
		Complete(r)
}
