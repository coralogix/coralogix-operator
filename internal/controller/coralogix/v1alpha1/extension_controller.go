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
	"google.golang.org/protobuf/types/known/wrapperspb"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
)

// ExtensionReconciler reconciles a Extension object
type ExtensionReconciler struct {
	ExtensionsClient *cxsdk.ExtensionsClient
	Interval         time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=extensions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=extensions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=extensions/finalizers,verbs=update

var (
	extensionFinalizerName = "extension.coralogix.com/finalizer"
)

func (r *ExtensionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.Extension{}, r)
}

func (r *ExtensionReconciler) FinalizerName() string {
	return extensionFinalizerName
}

func (r *ExtensionReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ExtensionReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	deployedExtensionsResponse, err := r.ExtensionsClient.GetDeployed(ctx, &cxsdk.GetDeployedExtensionsRequest{})
	if err != nil {
		log.Error(err, "Error getting deployed extensions")
		// If we cannot get the deployed extensions, we should not proceed with deployment.
		return fmt.Errorf("error on getting deployed extensions: %w", err)
	}

	deployedExtension := obj.(*coralogixv1alpha1.Extension)
	deployRequest, err := deployedExtension.Spec.ExtractDeployExtensionRequest()
	if err != nil {
		log.Error(err, "Error extracting deploy extension request", "extension", deployedExtension.Name)
		// If the request cannot be extracted, we should not proceed with deployment.
		return fmt.Errorf("error on extracting deploy extension request: %w", err)
	}
	id := deployRequest.Id.GetValue()
	for _, ext := range deployedExtensionsResponse.DeployedExtensions {
		if ext.Id.GetValue() == id {
			log.Info("Extension already deployed", "id", id)
			deployedExtension.Status = coralogixv1alpha1.ExtensionStatus{
				ID: &ext.Id.Value,
			}
			// If the extension is already deployed, we can skip the deployment step.
			return nil
		}
	}
	log.Info("Deploying remote extension", "extension", protojson.Format(deployRequest))
	deployResponse, err := r.ExtensionsClient.Deploy(ctx, deployRequest)
	if err != nil {
		return fmt.Errorf("error on deploying remote extension: %w", err)
	}
	log.Info("Remote extension deployed", "response", protojson.Format(deployResponse))

	deployedExtension.Status = coralogixv1alpha1.ExtensionStatus{
		ID: &deployResponse.ExtensionDeployment.Id.Value,
	}

	return nil
}

func (r *ExtensionReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	extension := obj.(*coralogixv1alpha1.Extension)
	updateRequest, err := extension.Spec.ExtractUpdateExtensionRequest(*extension.Status.ID)
	if err != nil {
		return fmt.Errorf("error on extracting update Extension request: %w", err)
	}
	log.Info("Updating remote Extension", "Extension", protojson.Format(updateRequest))
	updateResponse, err := r.ExtensionsClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.Info("Remote Extension updated", "Extension", protojson.Format(updateResponse))

	return nil
}

func (r *ExtensionReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	extension := obj.(*coralogixv1alpha1.Extension)
	log.Info("Undeploying extension from remote system", "id", *extension.Status.ID)
	_, err := r.ExtensionsClient.Undeploy(ctx, &cxsdk.UndeployExtensionRequest{Id: wrapperspb.String(*extension.Status.ID)})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Undeploying remote extension", "id", *extension.Status.ID)
		return fmt.Errorf("error undeploying remote extension %s: %w", *extension.Status.ID, err)
	}
	log.Info("Extension undeployed from remote system", "id", *extension.Status.ID)
	return nil
}

func (r *ExtensionReconciler) CheckIDInStatus(obj client.Object) bool {
	Extension := obj.(*coralogixv1alpha1.Extension)
	return Extension.Status.ID != nil && *Extension.Status.ID != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.Extension{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
