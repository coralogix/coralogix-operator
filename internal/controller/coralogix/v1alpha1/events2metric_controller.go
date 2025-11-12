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

	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// Events2MetricReconciler reconciles a Events2Metric object
type Events2MetricReconciler struct {
	Interval  time.Duration
	E2MClient *cxsdk.Events2MetricsClient
}

// +kubebuilder:rbac:groups=coralogix.com,resources=events2metrics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=events2metrics/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=events2metrics/finalizers,verbs=update

func (r *Events2MetricReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Events2Metric{}, r)
}

func (r *Events2MetricReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *Events2MetricReconciler) FinalizerName() string {
	return "events2metric.coralogix.com/finalizer"
}

func (r *Events2MetricReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	e2m := obj.(*coralogixv1alpha1.Events2Metric)
	createRequest := e2m.Spec.ExtractCreateE2MRequest()

	log.Info("Creating remote E2M", "E2M", protojson.Format(createRequest))
	createResponse, err := r.E2MClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote E2M: %w", err)
	}
	log.Info("Remote E2M created", "response", protojson.Format(createResponse))

	e2m.Status = coralogixv1alpha1.Events2MetricStatus{
		Id: utils.WrapperspbStringToStringPointer(createResponse.E2M.Id),
	}

	return nil
}

func (r *Events2MetricReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	e2m := obj.(*coralogixv1alpha1.Events2Metric)
	updateRequest := e2m.Spec.ExtractReplaceE2MRequest()
	updateRequest.E2M.Id = utils.StringPointerToWrapperspbString(e2m.Status.Id)

	log.Info("Updating remote E2M", "E2M", protojson.Format(updateRequest))
	updateResponse, err := r.E2MClient.Replace(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote E2M updated", "E2M", protojson.Format(updateResponse))

	return nil
}

func (r *Events2MetricReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	e2m := obj.(*coralogixv1alpha1.Events2Metric)
	log.Info("Deleting E2M from remote system", "id", *e2m.Status.Id)
	_, err := r.E2MClient.Delete(ctx,
		&cxsdk.DeleteE2MRequest{
			Id: utils.StringPointerToWrapperspbString(e2m.Status.Id),
		})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error deleting remote E2M", "id", *e2m.Status.Id)
		return fmt.Errorf("error deleting remote E2M %s: %w", *e2m.Status.Id, err)
	}
	log.Info("E2M deleted from remote system", "id", *e2m.Status.Id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Events2MetricReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Events2Metric{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
