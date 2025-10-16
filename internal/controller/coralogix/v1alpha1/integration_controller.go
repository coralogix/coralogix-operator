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
	"net/http"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	integrations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/integration_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// IntegrationReconciler reconciles a Integration object
type IntegrationReconciler struct {
	IntegrationsClient *integrations.IntegrationServiceAPIService
	Interval           time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=integrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=integrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=integrations/finalizers,verbs=update

func (r *IntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Integration{}, r)
}

func (r *IntegrationReconciler) FinalizerName() string {
	return "integration.coralogix.com/finalizer"
}

func (r *IntegrationReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *IntegrationReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	integration := obj.(*coralogixv1alpha1.Integration)
	createRequest, err := integration.Spec.ExtractCreateIntegrationRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create integration request: %w", err)
	}
	log.Info("Creating remote integration", "integration", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.IntegrationsClient.
		IntegrationServiceSaveIntegration(ctx).
		SaveIntegrationRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote integration: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote integration created", "response", utils.FormatJSON(createResponse))

	integration.Status = coralogixv1alpha1.IntegrationStatus{
		Id: &createResponse.IntegrationId,
	}

	return nil
}

func (r *IntegrationReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	integration := obj.(*coralogixv1alpha1.Integration)
	updateRequest, err := integration.Spec.ExtractUpdateIntegrationRequest(*integration.Status.Id)
	if err != nil {
		return fmt.Errorf("error on extracting update integration request: %w", err)
	}
	log.Info("Updating remote integration", "integration", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.IntegrationsClient.
		IntegrationServiceUpdateIntegration(ctx).
		UpdateIntegrationRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote integration updated", "integration", utils.FormatJSON(updateResponse))

	return nil
}

func (r *IntegrationReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	integration := obj.(*coralogixv1alpha1.Integration)
	log.Info("Deleting integration from remote system", "id", *integration.Status.Id)
	_, httpResp, err := r.IntegrationsClient.IntegrationServiceDeleteIntegration(ctx, *integration.Status.Id).Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); cxsdk.Code(apiErr) != http.StatusNotFound {
			log.Error(err, "Error deleting remote integration", "id", *integration.Status.Id)
			return fmt.Errorf("error deleting remote integration %s: %w", *integration.Status.Id, apiErr)
		}
	}
	log.Info("integration deleted from remote system", "id", *integration.Status.Id)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Integration{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
