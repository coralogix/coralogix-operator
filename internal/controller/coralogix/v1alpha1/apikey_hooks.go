// Copyright 2026 Coralogix Ltd.
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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	apikeysservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/api_keys_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// The hooks of the generated ApiKey kind keep the Secret of the released kind: <name>-secret in the
// namespace of the CR holds the value of the key under key-value, and the CR owns it.
//
//   - afterCreate writes the value of the Create answer.
//   - afterUpdate reads the key and writes its value, so a deleted or changed Secret is written
//     again on each reconcile. A hashed key has no value in Get: the Secret keeps the value that it
//     holds.

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// apiKeySecretKey is the key of the Secret that holds the value of the API key.
const apiKeySecretKey = "key-value"

func (r *ApiKeyReconciler) afterCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.ApiKey, resp *apikeysservice.CreateApiKeyResponse) error {
	return ensureAPIKeySecret(ctx, cr, &resp.Value)
}

func (r *ApiKeyReconciler) afterUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.ApiKey, _ map[string]interface{}) error {
	id := ptr.Deref(cr.Status.ID, "")
	got, httpResp, err := r.Client.ApiKeysServiceGetApiKey(ctx, id).Execute()
	if err != nil {
		// The text of the SDK error holds the response body, which holds the value of the key. The
		// error reaches the condition of the CR, so it has the status code only.
		return fmt.Errorf("error on getting remote ApiKey %s: status %d", id, cxsdk.Code(cxsdk.NewAPIError(httpResp, err)))
	}
	return ensureAPIKeySecret(ctx, cr, got.KeyInfo.Value)
}

// ensureAPIKeySecret makes the Secret of the CR hold value. It creates the Secret when it is
// missing, and changes only the key-value entry of a Secret that exists. A nil value, which a
// hashed key has, leaves an existing Secret as it is.
func ensureAPIKeySecret(ctx context.Context, cr *coralogixv1alpha1.ApiKey, value *string) error {
	c := config.GetClient()
	key := client.ObjectKey{Namespace: cr.Namespace, Name: cr.Name + "-secret"}
	existing := &corev1.Secret{}
	err := c.Get(ctx, key, existing)
	switch {
	case apierrors.IsNotFound(err):
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: coralogixv1alpha1.GroupVersion.String(),
					Kind:       "ApiKey",
					Name:       cr.Name,
					UID:        cr.UID,
					Controller: ptr.To(true),
				}},
			},
			Data: map[string][]byte{apiKeySecretKey: []byte(ptr.Deref(value, ""))},
		}
		if err := c.Create(ctx, secret); err != nil {
			return fmt.Errorf("error on creating the secret %s: %w", key.Name, err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("error on getting the secret %s: %w", key.Name, err)
	}
	if value == nil || string(existing.Data[apiKeySecretKey]) == *value {
		return nil
	}
	if existing.Data == nil {
		existing.Data = map[string][]byte{}
	}
	existing.Data[apiKeySecretKey] = []byte(*value)
	if err := c.Update(ctx, existing); err != nil {
		return fmt.Errorf("error on updating the secret %s: %w", key.Name, err)
	}
	return nil
}
