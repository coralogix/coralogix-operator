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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	apikeys "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/api_keys_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ApiKeyReconciler reconciles a ApiKey object
type ApiKeyReconciler struct {
	ApiKeysClient *apikeys.APIKeysServiceAPIService
	Interval      time.Duration
}

// +kubebuilder:rbac:groups=coralogix.com,resources=apikeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coralogix.com,resources=apikeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coralogix.com,resources=apikeys/finalizers,verbs=update

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

var (
	apiKeyFinalizerName = "api-key.coralogix.com/finalizer"
)

func (r *ApiKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return coralogixreconciler.ReconcileResource(ctx, req, &coralogixv1alpha1.ApiKey{}, r)
}

func (r *ApiKeyReconciler) FinalizerName() string {
	return apiKeyFinalizerName
}

func (r *ApiKeyReconciler) RequeueInterval() time.Duration {
	return r.Interval
}

func (r *ApiKeyReconciler) HandleCreation(ctx context.Context, log logr.Logger, obj client.Object) error {
	apiKey := obj.(*coralogixv1alpha1.ApiKey)
	createRequest := apiKey.Spec.ExtractCreateApiKeyRequest()
	log.Info("Creating remote api-key", "api-key", utils.FormatJSON(createRequest))
	createResponse, httpResp, err := r.ApiKeysClient.
		ApiKeysServiceCreateApiKey(context.Background()).
		CreateApiKeyRequest(*createRequest).
		Execute()
	if err != nil {
		return fmt.Errorf("error on creating remote api-key: %w", cxsdk.NewAPIError(httpResp, err))
	}
	log.Info("Remote api-key created", "response", utils.FormatJSON(createResponse))

	apiKey.Status = coralogixv1alpha1.ApiKeyStatus{
		Id: createResponse.KeyId,
	}

	log.Info("Creating secret for ApiKey", "id", createResponse.KeyId)
	secret := buildSecret(apiKey, createResponse.GetValue())
	err = config.GetClient().Create(ctx, secret)
	if err != nil {
		return fmt.Errorf("error on creating secret: %w", err)
	}

	return nil
}

func (r *ApiKeyReconciler) HandleUpdate(ctx context.Context, log logr.Logger, obj client.Object) error {
	apiKey := obj.(*coralogixv1alpha1.ApiKey)
	updateRequest := apiKey.Spec.ExtractUpdateApiKeyRequest()
	log.Info("Updating remote api-key", "api-key", utils.FormatJSON(updateRequest))
	updateResponse, httpResp, err := r.ApiKeysClient.
		ApiKeysServiceUpdateApiKey(context.Background(), *apiKey.Status.Id).
		UpdateApiKeyRequest(*updateRequest).
		Execute()
	if err != nil {
		return cxsdk.NewAPIError(httpResp, err)
	}
	log.Info("Remote api-key updated", "api-key", utils.FormatJSON(updateResponse))

	getResponse, httpResp, err := r.ApiKeysClient.
		ApiKeysServiceGetApiKey(context.Background(), *apiKey.Status.Id).
		Execute()
	if err != nil {
		return fmt.Errorf("error on getting remote api-key: %w", cxsdk.NewAPIError(httpResp, err))
	}

	existsSecret := &corev1.Secret{}
	err = config.GetClient().Get(ctx, client.ObjectKey{Name: apiKey.Name + "-secret", Namespace: apiKey.Namespace}, existsSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("secret is not found, probably was deleted, recreating it")
			if err := config.GetClient().Create(ctx, buildSecret(apiKey, getResponse.KeyInfo.GetValue())); err != nil {
				return fmt.Errorf("error on recreating secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("error on getting secret: %w", err)
	}

	desiredSecretKeyValue := getResponse.KeyInfo.GetValue()
	if string(existsSecret.Data["key-value"]) != desiredSecretKeyValue {
		log.Info("updating secret", "secret", apiKey.Name+"-secret")
		existsSecret.Data["key-value"] = []byte(desiredSecretKeyValue)
		if err := config.GetClient().Update(ctx, existsSecret); err != nil {
			return fmt.Errorf("error on updating secret: %w", err)
		}
	}

	return nil
}

func (r *ApiKeyReconciler) HandleDeletion(ctx context.Context, log logr.Logger, obj client.Object) error {
	apiKey := obj.(*coralogixv1alpha1.ApiKey)
	apiKeyId := apiKey.Status.Id
	_, httpResp, err := r.ApiKeysClient.
		ApiKeysServiceDeleteApiKey(ctx, *apiKeyId).
		Execute()
	if err != nil {
		if apiErr := cxsdk.NewAPIError(httpResp, err); !cxsdk.IsNotFound(apiErr) {
			log.Error(apiErr, "Error on deleting remote api-key", "id", apiKeyId)
			return fmt.Errorf("error to delete remote api-key -\n%v", apiKeyId)
		}
	}
	log.Info("api-key was deleted from remote", "id", apiKeyId)

	return nil
}

func buildSecret(apiKey *coralogixv1alpha1.ApiKey, keyValue string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      apiKey.Name + "-secret",
			Namespace: apiKey.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: apiKey.APIVersion,
				Kind:       apiKey.Kind,
				Name:       apiKey.Name,
				UID:        apiKey.UID,
				Controller: ptr.To(true),
			}},
		},
		Data: map[string][]byte{
			"key-value": []byte(keyValue),
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ApiKey{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
