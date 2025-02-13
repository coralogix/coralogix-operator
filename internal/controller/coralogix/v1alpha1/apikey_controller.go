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

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	"github.com/coralogix/coralogix-operator/internal/utils"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApiKeyReconciler reconciles a ApiKey object
type ApiKeyReconciler struct {
	ApiKeysClient clientset.ApiKeysClientInterface
	Scheme        *runtime.Scheme
}

// +kubebuilder:rbac:groups=coralogix.com,resources=apikeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=apikeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=apikeys/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *ApiKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	result, err := coralogix.ReconcileResource(ctx, req, &coralogixv1alpha1.ApiKey{}, r)
	monitoring.ApiKeyInfoMetric.WithLabelValues(req.Name, req.Namespace).Set(1)
	return result, err
}

func (r *ApiKeyReconciler) FinalizerName() string {
	return "api-key.coralogix.com/finalizer"
}

func (r *ApiKeyReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) (client.Object, error) {
	apiKey := obj.(*coralogixv1alpha1.ApiKey)
	createRequest := apiKey.Spec.ExtractCreateApiKeyRequest()
	log.V(1).Info("Creating remote api-key", "api-key", protojson.Format(createRequest))
	createResponse, err := r.ApiKeysClient.Create(ctx, createRequest)
	if err != nil {
		return nil, fmt.Errorf("error on creating remote api-key: %w", err)
	}
	log.V(1).Info("Remote api-key created", "response", protojson.Format(createResponse))

	apiKey.Status = coralogixv1alpha1.ApiKeyStatus{
		Id: &createResponse.KeyId,
	}

	return apiKey, nil
}

func (r *ApiKeyReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	apiKey := obj.(*coralogixv1alpha1.ApiKey)
	updateRequest := apiKey.Spec.ExtractUpdateApiKeyRequest(*apiKey.Status.Id)
	log.V(1).Info("Updating remote api-key", "api-key", protojson.Format(updateRequest))
	updateResponse, err := r.ApiKeysClient.Update(ctx, updateRequest)
	if err != nil {
		return err
	}
	log.V(1).Info("Remote api-key updated", "api-key", protojson.Format(updateResponse))

	return nil
}

func (r *ApiKeyReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	apiKey := obj.(*coralogixv1alpha1.ApiKey)
	id := *apiKey.Status.Id
	log.V(1).Info("Deleting api-key from remote system", "id", id)
	_, err := r.ApiKeysClient.Delete(ctx, &cxsdk.DeleteAPIKeyRequest{KeyId: id})
	if err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error deleting remote api-key", "id", id)
		return fmt.Errorf("error deleting remote api-key %s: %w", id, err)
	}
	log.V(1).Info("Api-key deleted from remote system", "id", id)
	return nil
}

func (r *ApiKeyReconciler) CheckIDInStatus(obj client.Object) bool {
	apiKey := obj.(*coralogixv1alpha1.ApiKey)
	return apiKey.Status.Id != nil && *apiKey.Status.Id != ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ApiKey{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
