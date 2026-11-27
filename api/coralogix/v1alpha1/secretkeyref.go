// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// ResolveSecretKeyRef returns the value of the Secret key that ref names, read from the Secret in
// namespace, the namespace of the referring resource. Generated kinds use it for a string field
// that a valueFrom line of the operator generator sources from a Secret: the API takes only the
// literal, so the operator sends the value that it reads.
//
// A missing Secret or key is an error, also when ref.Optional is set. The error names the Secret
// and the key, never the value.
func ResolveSecretKeyRef(ctx context.Context, namespace string, ref corev1.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret '%s': %w", ref.Name, err)
	}
	value, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("cannot find key '%s' in secret '%s'", ref.Key, ref.Name)
	}
	return string(value), nil
}
