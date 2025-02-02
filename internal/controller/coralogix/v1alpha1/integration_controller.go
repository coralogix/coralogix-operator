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

// IntegrationReconciler reconciles a Integration object
type IntegrationReconciler struct {
	client.Client
	IntegrationsClient *cxsdk.IntegrationsClient
	Scheme             *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=integrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=integrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=integrations/finalizers,verbs=update

var (
	integrationFinalizerName = "integration.coralogix.com/finalizer"
)

func (r *IntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues(
		"integration", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	integration := &coralogixv1alpha1.Integration{}
	if err := r.Get(ctx, req.NamespacedName, integration); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(integration.Status.Id, "") == "" {
		err := r.create(ctx, log, integration)
		if err != nil {
			log.Error(err, "Error on creating Integration")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !integration.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, integration)
		if err != nil {
			log.Error(err, "Error on deleting Integration")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(integration.GetLabels()) {
		err := r.deleteRemoteIntegration(ctx, log, *integration.Status.Id)
		if err != nil {
			log.Error(err, "Error on deleting Integration")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, integration)
	if err != nil {
		log.Error(err, "Error on updating Integration")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	return ctrl.Result{}, nil
}

func (r *IntegrationReconciler) create(ctx context.Context, log logr.Logger, integration *coralogixv1alpha1.Integration) error {
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

	log.V(1).Info("Updating Integration status", "id", integration.Status.Id)
	if err = r.Status().Update(ctx, integration); err != nil {
		if deleteErr := r.deleteRemoteIntegration(ctx, log, *integration.Status.Id); deleteErr != nil {
			return fmt.Errorf("error to delete integration after status update error. Update error: %w. Deletion error: %w", err, deleteErr)
		}
		return fmt.Errorf("error to update integration status: %w", err)
	}

	if !controllerutil.ContainsFinalizer(integration, integrationFinalizerName) {
		log.V(1).Info("Updating Integration to add finalizer", "name", integration.Name)
		controllerutil.AddFinalizer(integration, integrationFinalizerName)
		if err := r.Update(ctx, integration); err != nil {
			return fmt.Errorf("error on updating Integration: %w", err)
		}
	}

	return nil
}

func (r *IntegrationReconciler) update(ctx context.Context, log logr.Logger, integration *coralogixv1alpha1.Integration) error {
	updateRequest, err := integration.Spec.ExtractUpdateIntegrationRequest(*integration.Status.Id)
	if err != nil {
		return fmt.Errorf("error on extracting update integration request: %w", err)
	}
	log.V(1).Info("Updating remote integration", "integration", protojson.Format(updateRequest))
	updateResponse, err := r.IntegrationsClient.Update(ctx, updateRequest)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("integration not found on remote, removing id from status")
			integration.Status = coralogixv1alpha1.IntegrationStatus{
				Id: ptr.To(""),
			}
			if err = r.Status().Update(ctx, integration); err != nil {
				return fmt.Errorf("error on updating Integration status: %w", err)
			}
			return fmt.Errorf("integration not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating integration: %w", err)
	}
	log.V(1).Info("Remote integration updated", "integration", protojson.Format(updateResponse))

	return nil
}

func (r *IntegrationReconciler) delete(ctx context.Context, log logr.Logger, integration *coralogixv1alpha1.Integration) error {
	if err := r.deleteRemoteIntegration(ctx, log, *integration.Status.Id); err != nil {
		return fmt.Errorf("error on deleting remote integration: %w", err)
	}

	log.V(1).Info("Removing finalizer from Integration")
	controllerutil.RemoveFinalizer(integration, integrationFinalizerName)
	if err := r.Update(ctx, integration); err != nil {
		return fmt.Errorf("error on updating Integration: %w", err)
	}

	return nil
}

func (r *IntegrationReconciler) deleteRemoteIntegration(ctx context.Context, log logr.Logger, integrationID string) error {
	log.V(1).Info("Deleting integration from remote", "id", integrationID)
	if _, err := r.IntegrationsClient.Delete(ctx, &cxsdk.DeleteIntegrationRequest{IntegrationId: wrapperspb.String(integrationID)}); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote integration", "id", integrationID)
		return fmt.Errorf("error to delete remote integration %s: %w", integrationID, err)
	}
	log.V(1).Info("integration was deleted from remote", "id", integrationID)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Integration{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
