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

	corev1 "k8s.io/api/core/v1"
)

// ResolveSecretValue returns the value of a field that a custom resource sets either as a literal
// or as a key of a Secret in its namespace (the secretRef key of opgen). It returns nil when the CR
// sets neither, and an error when it sets both. The error names the field, the Secret, and the key,
// never the value.
func ResolveSecretValue(ctx context.Context, namespace, field string, literal *string, ref *corev1.SecretKeySelector) (*string, error) {
	if ref == nil {
		return literal, nil
	}
	if literal != nil {
		return nil, fmt.Errorf("%s has both a literal value and a Secret reference", field)
	}
	value, err := readSecret(ctx, *ref, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to read the Secret of %s: %w", field, err)
	}
	return &value, nil
}

// ResolveSecretValues returns a map whose values a custom resource sets either as literals or as
// keys of Secrets in its namespace (the secretRef key of opgen). It returns an error when a key has
// both sources. The error names the field and the key, never a value.
func ResolveSecretValues(ctx context.Context, namespace, field string, literals map[string]string, refs map[string]corev1.SecretKeySelector) (map[string]string, error) {
	if len(refs) == 0 {
		return literals, nil
	}
	values := make(map[string]string, len(literals)+len(refs))
	for key, value := range literals {
		values[key] = value
	}
	for key, ref := range refs {
		if _, ok := values[key]; ok {
			return nil, fmt.Errorf("%s: the key %q has both a literal value and a Secret reference", field, key)
		}
		value, err := readSecret(ctx, ref, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read the Secret of %s key %q: %w", field, key, err)
		}
		values[key] = value
	}
	return values, nil
}
