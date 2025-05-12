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

	"strconv"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// DataSetReconciler reconciles a DataSet object
type DataSetReconciler struct {
	DataSetsClient *cxsdk.DataSetClient
	Interval       time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=datasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=datasets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=datasets/finalizers,verbs=update

func (r *DataSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.DataSet{}, r)
}

func (r *DataSetReconciler) FinalizerName() string {
	return "data-set.coralogix.com/finalizer"
}

func (r *DataSetReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *DataSetReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	dataSet := obj.(*coralogixv1alpha1.DataSet)
	createRequest, err := dataSet.ExtractCreateDataSetRequest(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting create request: %w", err)
	}
	log.Info("Creating remote dataSet", "dataSet", protojson.Format(createRequest))
	createResponse, err := r.DataSetsClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote dataSet: %w", err)
	}
	log.Info("Remote dataSet created", "response", protojson.Format(createResponse))

	dataSet.Status = coralogixv1alpha1.DataSetStatus{
		ID: pointer.String(strconv.Itoa(int(createResponse.CustomEnrichment.Id))),
	}

	return nil
}

func (r *DataSetReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	dataSet := obj.(*coralogixv1alpha1.DataSet)
	updateRequest, err := dataSet.ExtractUpdateDataSetRequest(ctx)
	if err != nil {
		return fmt.Errorf("error on extracting update request: %w", err)
	}
	log.Info("Updating remote dataSet", "dataSet", protojson.Format(updateRequest))
	updateResponse, err := r.DataSetsClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote dataSet updated", "dataSet", protojson.Format(updateResponse))

	return nil
}

func (r *DataSetReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	dataSet := obj.(*coralogixv1alpha1.DataSet)
	log.Info("Deleting dataSet from remote system", "id", *dataSet.Status.ID)
	id, err := strconv.Atoi(*dataSet.Status.ID)
	if err != nil {
		return fmt.Errorf("error on converting data-set id to int: %w", err)
	}

	_, err = r.DataSetsClient.Delete(ctx, &cxsdk.DeleteDataSetRequest{
		CustomEnrichmentId: wrapperspb.UInt32(uint32(id)),
	})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote dataSet", "id", *dataSet.Status.ID)
		return fmt.Errorf("error deleting remote dataSet %s: %w", *dataSet.Status.ID, err)
	}
	log.Info("DataSet deleted from remote system", "id", *dataSet.Status.ID)
	return nil
}

func (r *DataSetReconciler) CheckIDInStatus(obj client.Object) bool {
	dataSet := obj.(*coralogixv1alpha1.DataSet)
	return dataSet.Status.ID != nil && *dataSet.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.DataSet{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
