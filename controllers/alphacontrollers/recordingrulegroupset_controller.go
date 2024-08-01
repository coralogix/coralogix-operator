/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alphacontrollers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	rrg "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/recording-rules-groups/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var recordingRuleGroupSetFinalizerName = "recordingrulegroupset.coralogix.com/finalizer"

// RecordingRuleGroupSetReconciler reconciles a RecordingRuleGroupSet object
type RecordingRuleGroupSetReconciler struct {
	client.Client
	CoralogixClientSet clientset.ClientSetInterface
	Scheme             *runtime.Scheme
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
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if ptr.Deref(recordingRuleGroupSet.Status.ID, "") == "" {
		if err := r.create(ctx, recordingRuleGroupSet); err != nil {
			log.Error(err, "Failed to create RecordingRuleGroupSet", "error", err)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !recordingRuleGroupSet.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.delete(ctx, recordingRuleGroupSet); err != nil {
			log.Error(err, "Failed to delete RecordingRuleGroupSet", "error", err)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.update(ctx, recordingRuleGroupSet); err != nil {
		log.Error(err, "Failed to update RecordingRuleGroupSet", "error", err)
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *RecordingRuleGroupSetReconciler) create(ctx context.Context, recordingRuleGroupSet *coralogixv1alpha1.RecordingRuleGroupSet) error {
	response, err := r.CoralogixClientSet.
		RecordingRuleGroups().
		CreateRecordingRuleGroupSet(ctx, &rrg.CreateRuleGroupSet{
			Name:   ptr.To(recordingRuleGroupSet.Name),
			Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
		})

	if err != nil {
		return fmt.Errorf("failed to create recording rule groupSet: %w", err)
	}

	recordingRuleGroupSet.Status.ID = ptr.To(response.Id)

	if err := r.Status().Update(ctx, recordingRuleGroupSet); err != nil {
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
	remoteRecordingRule, err := r.CoralogixClientSet.RecordingRuleGroups().GetRecordingRuleGroupSet(ctx, &rrg.FetchRuleGroupSet{
		Id: *recordingRuleGroupSet.Status.ID,
	})

	if err != nil {
		if status.Code(err) == codes.NotFound {
			recordingRuleGroupSet.Status.ID = nil
			if err := r.Status().Update(ctx, recordingRuleGroupSet); err != nil {
				return fmt.Errorf("failed to update recording rule groupSet status: %w", err)
			}
			return err
		}
		return fmt.Errorf("failed to get recording rule groupSet: %w", err)
	}

	if _, err := r.CoralogixClientSet.
		RecordingRuleGroups().
		UpdateRecordingRuleGroupSet(ctx, &rrg.UpdateRuleGroupSet{
			Id:     remoteRecordingRule.Id,
			Groups: recordingRuleGroupSet.Spec.ExtractRecordingRuleGroups(),
			Name:   ptr.To(recordingRuleGroupSet.Name),
		}); err != nil {
		return fmt.Errorf("failed to update recording rule groupSet: %w", err)
	}

	return nil
}

func (r *RecordingRuleGroupSetReconciler) delete(ctx context.Context, recordingRuleGroupSet *coralogixv1alpha1.RecordingRuleGroupSet) error {
	_, err := r.CoralogixClientSet.RecordingRuleGroups().DeleteRecordingRuleGroupSet(ctx, &rrg.DeleteRuleGroupSet{
		Id: *recordingRuleGroupSet.Status.ID,
	})

	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to delete recording rule groupSet: %w", err)
	}

	controllerutil.RemoveFinalizer(recordingRuleGroupSet, alertFinalizerName)
	if err = r.Update(ctx, recordingRuleGroupSet); err != nil {
		return fmt.Errorf("failed to remove finalizer from recording rule groupSet: %w", err)
	}

	return nil
}

func (r *RecordingRuleGroupSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RecordingRuleGroupSet{}).
		Complete(r)
}
