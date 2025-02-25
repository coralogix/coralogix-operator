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
	"google.golang.org/protobuf/types/known/wrapperspb"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// IntegrationReconciler reconciles a Integration object
type IntegrationReconciler struct {
	IntegrationsClient *cxsdk.IntegrationsClient
}

// +kubebuilder:rbac:groups=coralogix.com,resources=integrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=integrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=integrations/finalizers,verbs=update

var (
	integrationFinalizerName = "integration.coralogix.com/finalizer"
)

func (r *IntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Integration{}, r)
}

func (r *IntegrationReconciler) FinalizerName() string {
	return "integration.coralogix.com/finalizer"
}

func (r *IntegrationReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	integration := obj.(*coralogixv1alpha1.Integration)
	createRequest, err := integration.Spec.ExtractCreateIntegrationRequest()
	if err != nil {
		return fmt.Errorf("error on extracting create integration request: %w", err)
	}
	log.V(1).Info("Creating remote integration", "integration", protojson.Format(createRequest))
	createResponse, err := r.IntegrationsClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote integration: %w", err)
	}
	log.V(1).Info("Remote integration created", "response", protojson.Format(createResponse))

	integration.Status = coralogixv1alpha1.IntegrationStatus{
		Id: &createResponse.IntegrationId.Value,
	}

	return nil
}

func (r *IntegrationReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	integration := obj.(*coralogixv1alpha1.Integration)
	updateRequest, err := integration.Spec.ExtractUpdateIntegrationRequest(*integration.Status.Id)
	if err != nil {
		return fmt.Errorf("error on extracting update integration request: %w", err)
	}
	log.V(1).Info("Updating remote integration", "integration", protojson.Format(updateRequest))
	updateResponse, err := r.IntegrationsClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.V(1).Info("Remote integration updated", "integration", protojson.Format(updateResponse))

	return nil
}

func (r *IntegrationReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	integration := obj.(*coralogixv1alpha1.Integration)
	log.V(1).Info("Deleting integration from remote system", "id", *integration.Status.Id)
	_, err := r.IntegrationsClient.Delete(ctx, &cxsdk.DeleteIntegrationRequest{IntegrationId: wrapperspb.String(*integration.Status.Id)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote integration", "id", *integration.Status.Id)
		return fmt.Errorf("error deleting remote integration %s: %w", *integration.Status.Id, err)
	}
	log.V(1).Info("integration deleted from remote system", "id", *integration.Status.Id)
	return nil
}

func (r *IntegrationReconciler) CheckIDInStatus(obj client.Object) bool {
	integration := obj.(*coralogixv1alpha1.Integration)
	return integration.Status.Id != nil && *integration.Status.Id != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *IntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Integration{}).
		WithEventFilter(utils.GetSelector().Predicate()).
		Complete(r)
}
