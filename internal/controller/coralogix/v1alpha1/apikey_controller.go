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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ApiKeyReconciler reconciles a ApiKey object
type ApiKeyReconciler struct {
	ApiKeysClient *cxsdk.ApikeysClient
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
	log := log.FromContext(ctx).WithValues(
		"apiKey", req.NamespacedName.Name,
		"namespace", req.NamespacedName.Namespace,
	)

	apiKey := &coralogixv1alpha1.ApiKey{}
	if err := config.GetClient().Get(ctx, req.NamespacedName, apiKey); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if ptr.Deref(apiKey.Status.Id, "") == "" {
		if err, reason := r.create(ctx, log, apiKey); err != nil {
			log.Error(err, "Error on creating ApiKey")
			return coralogixreconciler.ManageErrorWithRequeue(ctx, apiKey, reason, err)
		}
		return coralogixreconciler.ManageSuccessWithRequeue(ctx, apiKey, r.Interval)
	}

	if !apiKey.ObjectMeta.DeletionTimestamp.IsZero() {
		if err, reason := r.delete(ctx, log, apiKey); err != nil {
			log.Error(err, "Error on deleting ApiKey")
			return coralogixreconciler.ManageErrorWithRequeue(ctx, apiKey, reason, err)
		}
		return ctrl.Result{}, nil
	}

	if !config.GetConfig().Selector.Matches(apiKey.GetLabels(), apiKey.GetNamespace()) {
		if err := r.deleteRemoteApiKey(ctx, log, apiKey.Status.Id); err != nil {
			log.Error(err, "Error on deleting ApiKey")
			return coralogixreconciler.ManageErrorWithRequeue(ctx, apiKey, utils.ReasonRemoteDeletionFailed, err)
		}
		return ctrl.Result{}, nil
	}

	if err, reason := r.update(ctx, log, apiKey); err != nil {
		log.Error(err, "Error on updating ApiKey")
		return coralogixreconciler.ManageErrorWithRequeue(ctx, apiKey, reason, err)
	}

	return coralogixreconciler.ManageSuccessWithRequeue(ctx, apiKey, r.Interval)
}

func (r *ApiKeyReconciler) create(ctx context.Context, log logr.Logger, apiKey *coralogixv1alpha1.ApiKey) (error, string) {
	createRequest := apiKey.Spec.ExtractCreateApiKeyRequest()
	log.Info("Creating remote api-key", "api-key", protojson.Format(createRequest))
	createResponse, err := r.ApiKeysClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote api-key: %w", err), utils.ReasonRemoteCreationFailed
	}
	log.Info("Remote api-key created", "response", protojson.Format(createResponse))

	apiKey.Status = coralogixv1alpha1.ApiKeyStatus{
		Id: ptr.To(createResponse.KeyId),
	}

	log.Info("Updating ApiKey status", "id", createResponse.KeyId)
	if err = config.GetClient().Status().Update(ctx, apiKey); err != nil {
		if err := r.deleteRemoteApiKey(ctx, log, apiKey.Status.Id); err != nil {
			return fmt.Errorf("error to delete api-key after status update error: %w", err), utils.ReasonRemoteUpdateFailed
		}
		return fmt.Errorf("error to update api-key status: %w", err), utils.ReasonRemoteUpdateFailed
	}

	if !controllerutil.ContainsFinalizer(apiKey, apiKeyFinalizerName) {
		log.Info("Updating ApiKey to add finalizer", "id", createResponse.KeyId)
		controllerutil.AddFinalizer(apiKey, apiKeyFinalizerName)
		if err = config.GetClient().Update(ctx, apiKey); err != nil {
			return fmt.Errorf("error on updating ApiKey: %w", err), utils.ReasonInternalK8sError
		}
	}

	log.Info("Creating secret for ApiKey", "id", createResponse.KeyId)
	secret := buildSecret(apiKey, createResponse.GetValue())
	err = config.GetClient().Create(ctx, secret)
	if err != nil {
		return fmt.Errorf("error on creating secret: %w", err), utils.ReasonInternalK8sError
	}

	return nil, ""
}

func (r *ApiKeyReconciler) update(ctx context.Context, log logr.Logger, apiKey *coralogixv1alpha1.ApiKey) (error, string) {
	updateRequest := apiKey.Spec.ExtractUpdateApiKeyRequest(*apiKey.Status.Id)
	log.Info("Updating remote api-key", "api-key", protojson.Format(updateRequest))
	updateResponse, err := r.ApiKeysClient.Update(ctx, updateRequest)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.Info("api-key not found on remote, removing id from status")
			apiKey.Status = coralogixv1alpha1.ApiKeyStatus{
				Id: ptr.To(""),
			}
			if err = config.GetClient().Status().Update(ctx, apiKey); err != nil {
				return fmt.Errorf("error on updating ApiKey status: %w", err), utils.ReasonInternalK8sError
			}
			return fmt.Errorf("api-key not found on remote: %w", err), utils.ReasonRemoteResourceNotFound
		}
		return fmt.Errorf("error on updating api-key: %w", err), utils.ReasonRemoteUpdateFailed
	}
	log.Info("Remote api-key updated", "api-key", protojson.Format(updateResponse))

	getResponse, err := r.ApiKeysClient.Get(ctx, &cxsdk.GetAPIKeyRequest{KeyId: *apiKey.Status.Id})
	if err != nil {
		return fmt.Errorf("error on getting remote api-key: %w", err), utils.ReasonRemoteUpdateFailed
	}

	existsSecret := &corev1.Secret{}
	err = config.GetClient().Get(ctx, client.ObjectKey{Name: apiKey.Name + "-secret", Namespace: apiKey.Namespace}, existsSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("secret is not found, probably was deleted, recreating it")
			if err := config.GetClient().Create(ctx, buildSecret(apiKey, getResponse.KeyInfo.GetValue())); err != nil {
				return fmt.Errorf("error on recreating secret: %w", err), utils.ReasonInternalK8sError
			}
			return nil, ""
		}
		return fmt.Errorf("error on getting secret: %w", err), utils.ReasonInternalK8sError
	}

	desiredSecretKeyValue := getResponse.KeyInfo.GetValue()
	if string(existsSecret.Data["key-value"]) != desiredSecretKeyValue {
		log.Info("updating secret", "secret", apiKey.Name+"-secret")
		existsSecret.Data["key-value"] = []byte(desiredSecretKeyValue)
		if err := config.GetClient().Update(ctx, existsSecret); err != nil {
			return fmt.Errorf("error on updating secret: %w", err), utils.ReasonInternalK8sError
		}
	}

	return nil, ""
}

func (r *ApiKeyReconciler) delete(ctx context.Context, log logr.Logger, apiKey *coralogixv1alpha1.ApiKey) (error, string) {
	if err := r.deleteRemoteApiKey(ctx, log, apiKey.Status.Id); err != nil {
		return fmt.Errorf("error on deleting remote api-key: %w", err), utils.ReasonRemoteDeletionFailed
	}

	log.Info("Removing finalizer from ApiKey")
	controllerutil.RemoveFinalizer(apiKey, apiKeyFinalizerName)
	if err := config.GetClient().Update(ctx, apiKey); err != nil {
		return fmt.Errorf("error on updating ApiKey: %w", err), utils.ReasonInternalK8sError
	}

	return nil, ""
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

func (r *ApiKeyReconciler) deleteRemoteApiKey(ctx context.Context, log logr.Logger, apiKeyId *string) error {
	log.Info("Deleting api-key from remote", "id", apiKeyId)
	if _, err := r.ApiKeysClient.Delete(ctx, &cxsdk.DeleteAPIKeyRequest{KeyId: *apiKeyId}); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.Error(err, "Error on deleting remote api-key", "id", apiKeyId)
		return fmt.Errorf("error to delete remote api-key -\n%v", apiKeyId)
	}
	log.Info("api-key was deleted from remote", "id", apiKeyId)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ApiKey{}).
		WithEventFilter(config.GetConfig().Selector.Predicate()).
		Complete(r)
}
