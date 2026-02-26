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
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	e2m "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/events2metrics_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/v2/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// Events2MetricReconciler reconciles a Events2Metric object
type Events2MetricReconciler struct {
	Interval  time.Duration
	E2MClient *e2m.Events2MetricsServiceAPIService
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
	e2mCR := obj.(*coralogixv1alpha1.Events2Metric)
	createRequest := e2mCR.Spec.ExtractCreateE2MRequest()

	log.Info("Creating remote E2M", "E2M", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.E2MClient.
		Events2MetricServiceCreateE2M(ctx).
		Events2MetricServiceCreateE2MRequest(createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote E2M: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote E2M created", "response", utils.FormatJSON(createResponse))

	e2mCR.Status = coralogixv1alpha1.Events2MetricStatus{
		Id: extractE2MId(createResponse.GetE2m()),
	}

	return nil
}

func (r *Events2MetricReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	e2mCR := obj.(*coralogixv1alpha1.Events2Metric)
	updateRequest := e2mCR.Spec.ExtractReplaceE2MRequest(ptr.Deref(e2mCR.Status.Id, ""))

	log.Info("Updating remote E2M", "E2M", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.E2MClient.
		Events2MetricServiceReplaceE2M(ctx).
		Events2MetricServiceReplaceE2MRequest(updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote E2M updated", "E2M", utils.FormatJSON(updateResponse))

	return nil
}

func (r *Events2MetricReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	e2mCR := obj.(*coralogixv1alpha1.Events2Metric)
	id := ptr.Deref(e2mCR.Status.Id, "")
	log.Info("Deleting E2M from remote system", "id", id)
	_, httpResp, err := r.E2MClient.
		Events2MetricServiceDeleteE2M(ctx, id).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(err, "Error deleting remote E2M", "id", id)
			return fmt.Errorf("error deleting remote E2M %s: %w", id, apiErr)
		}
	}
	log.Info("E2M deleted from remote system", "id", id)
	return nil
}

func extractE2MId(e2mResp e2m.E2M) *string {
	if e2mResp.E2MLogsQuery != nil {
		return e2mResp.E2MLogsQuery.Id
	}
	if e2mResp.E2MSpansQuery != nil {
		return e2mResp.E2MSpansQuery.Id
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Events2MetricReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Events2Metric{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
