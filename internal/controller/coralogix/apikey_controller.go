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

package coralogix

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

// ApiKeyReconciler reconciles a ApiKey object
type ApiKeyReconciler struct {
	client.Client
	ApiKeysClient clientset.ApiKeysClientInterface
	Scheme        *runtime.Scheme
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
	if err := r.Get(ctx, req.NamespacedName, apiKey); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}

	if ptr.Deref(apiKey.Status.Id, "") == "" {
		err := r.create(ctx, log, apiKey)
		if err != nil {
			log.Error(err, "Error on creating ApiKey")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		monitoring.ApiKeyInfoMetric.WithLabelValues(apiKey.Name, apiKey.Namespace).Set(1)
		return ctrl.Result{}, nil
	}

	if !apiKey.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.delete(ctx, log, apiKey)
		if err != nil {
			log.Error(err, "Error on deleting ApiKey")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		monitoring.ApiKeyInfoMetric.DeleteLabelValues(apiKey.Name, apiKey.Namespace)
		return ctrl.Result{}, nil
	}

	if !utils.GetLabelFilter().Matches(apiKey.GetLabels()) {
		err := r.deleteRemoteApiKey(ctx, log, apiKey.Status.Id)
		if err != nil {
			log.Error(err, "Error on deleting ApiKey")
			return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
		}
		monitoring.ApiKeyInfoMetric.DeleteLabelValues(apiKey.Name, apiKey.Namespace)
		return ctrl.Result{}, nil
	}

	err := r.update(ctx, log, apiKey)
	if err != nil {
		log.Error(err, "Error on updating ApiKey")
		return ctrl.Result{RequeueAfter: utils.DefaultErrRequeuePeriod}, err
	}
	monitoring.ApiKeyInfoMetric.WithLabelValues(apiKey.Name, apiKey.Namespace).Set(1)

	return ctrl.Result{}, nil
}

func (r *ApiKeyReconciler) create(ctx context.Context, log logr.Logger, apiKey *coralogixv1alpha1.ApiKey) error {
	createRequest := apiKey.Spec.ExtractCreateApiKeyRequest()
	log.V(1).Info("Creating remote api-key", "api-key", protojson.Format(createRequest))
	createResponse, err := r.ApiKeysClient.Create(ctx, createRequest)
	if err != nil {
		return fmt.Errorf("error on creating remote api-key: %w", err)
	}
	log.V(1).Info("Remote api-key created", "response", protojson.Format(createResponse))

	apiKey.Status = coralogixv1alpha1.ApiKeyStatus{
		Id: ptr.To(createResponse.KeyId),
	}

	log.V(1).Info("Updating ApiKey status", "id", createResponse.KeyId)
	if err = r.Status().Update(ctx, apiKey); err != nil {
		if err := r.deleteRemoteApiKey(ctx, log, apiKey.Status.Id); err != nil {
			return fmt.Errorf("error to delete api-key after status update error -\n%v", apiKey)
		}
		return fmt.Errorf("error to update api-key status -\n%v", apiKey)
	}

	if !controllerutil.ContainsFinalizer(apiKey, apiKeyFinalizerName) {
		log.V(1).Info("Updating ApiKey to add finalizer", "id", createResponse.KeyId)
		controllerutil.AddFinalizer(apiKey, apiKeyFinalizerName)
		if err := r.Update(ctx, apiKey); err != nil {
			return fmt.Errorf("error on updating ApiKey: %w", err)
		}
	}

	log.V(1).Info("Creating secret for ApiKey", "id", createResponse.KeyId)
	secret := buildSecret(apiKey, createResponse.GetValue())
	err = r.Client.Create(ctx, secret)
	if err != nil {
		return fmt.Errorf("error on creating secret: %w", err)
	}

	return nil
}

func (r *ApiKeyReconciler) update(ctx context.Context, log logr.Logger, apiKey *coralogixv1alpha1.ApiKey) error {
	updateRequest := apiKey.Spec.ExtractUpdateApiKeyRequest(*apiKey.Status.Id)
	log.V(1).Info("Updating remote api-key", "api-key", protojson.Format(updateRequest))
	updateResponse, err := r.ApiKeysClient.Update(ctx, updateRequest)
	if err != nil {
		if cxsdk.Code(err) == codes.NotFound {
			log.V(1).Info("api-key not found on remote, removing id from status")
			apiKey.Status = coralogixv1alpha1.ApiKeyStatus{
				Id: ptr.To(""),
			}
			if err = r.Status().Update(ctx, apiKey); err != nil {
				return fmt.Errorf("error on updating ApiKey status: %w", err)
			}
			return fmt.Errorf("api-key not found on remote: %w", err)
		}
		return fmt.Errorf("error on updating api-key: %w", err)
	}
	log.V(1).Info("Remote api-key updated", "api-key", protojson.Format(updateResponse))

	getResponse, err := r.ApiKeysClient.Get(ctx, &cxsdk.GetAPIKeyRequest{KeyId: *apiKey.Status.Id})
	if err != nil {
		return fmt.Errorf("error on getting remote api-key: %w", err)
	}

	existsSecret := &corev1.Secret{}
	err = r.Client.Get(ctx, client.ObjectKey{Name: apiKey.Name + "-secret", Namespace: apiKey.Namespace}, existsSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("secret is not found, probably was deleted, recreating it")
			if err := r.Client.Create(ctx, buildSecret(apiKey, getResponse.KeyInfo.GetValue())); err != nil {
				return fmt.Errorf("error on recreating secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("error on getting secret: %w", err)
	}

	desiredSecretKeyValue := getResponse.KeyInfo.GetValue()
	if string(existsSecret.Data["key-value"]) != desiredSecretKeyValue {
		log.V(1).Info("updating secret", "secret", apiKey.Name+"-secret")
		existsSecret.Data["key-value"] = []byte(desiredSecretKeyValue)
		if err := r.Client.Update(ctx, existsSecret); err != nil {
			return fmt.Errorf("error on updating secret: %w", err)
		}
	}

	return nil
}

func (r *ApiKeyReconciler) delete(ctx context.Context, log logr.Logger, apiKey *coralogixv1alpha1.ApiKey) error {
	if err := r.deleteRemoteApiKey(ctx, log, apiKey.Status.Id); err != nil {
		return fmt.Errorf("error on deleting remote api-key: %w", err)
	}

	log.V(1).Info("Removing finalizer from ApiKey")
	controllerutil.RemoveFinalizer(apiKey, apiKeyFinalizerName)
	if err := r.Update(ctx, apiKey); err != nil {
		return fmt.Errorf("error on updating ApiKey: %w", err)
	}

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

func (r *ApiKeyReconciler) deleteRemoteApiKey(ctx context.Context, log logr.Logger, apiKeyId *string) error {
	log.V(1).Info("Deleting api-key from remote", "id", apiKeyId)
	if _, err := r.ApiKeysClient.Delete(ctx, &cxsdk.DeleteAPIKeyRequest{KeyId: *apiKeyId}); err != nil && cxsdk.Code(err) != codes.NotFound {
		log.V(1).Error(err, "Error on deleting remote api-key", "id", apiKeyId)
		return fmt.Errorf("error to delete remote api-key -\n%v", apiKeyId)
	}
	log.V(1).Info("api-key was deleted from remote", "id", apiKeyId)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.ApiKey{}).
		WithEventFilter(utils.GetLabelFilter().Predicate()).
		Complete(r)
}
