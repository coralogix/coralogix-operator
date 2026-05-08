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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	integrations "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/integration_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func TestExtractParameters(t *testing.T) {
	const ns = "default"

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}

	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: ns},
		Data:       map[string][]byte{"token": []byte("abc123")},
	}

	ptrTrue := true

	tests := []struct {
		name        string
		spec        IntegrationSpec
		wantErr     bool
		wantParams  map[string]string
		wantMissing []string // parameter keys that must NOT appear in the result
	}{
		{
			name: "inline params merged with secret-sourced param",
			spec: IntegrationSpec{
				Parameters: runtime.RawExtension{Raw: []byte(`{"AppName":"prod"}`)},
				ParametersFromSecret: map[string]corev1.SecretKeySelector{
					"Token": {
						LocalObjectReference: corev1.LocalObjectReference{Name: "creds"},
						Key:                  "token",
					},
				},
			},
			wantParams: map[string]string{"AppName": "prod", "Token": "abc123"},
		},
		{
			name: "no inline params, only secret-sourced",
			spec: IntegrationSpec{
				ParametersFromSecret: map[string]corev1.SecretKeySelector{
					"Token": {
						LocalObjectReference: corev1.LocalObjectReference{Name: "creds"},
						Key:                  "token",
					},
				},
			},
			wantParams: map[string]string{"Token": "abc123"},
		},
		{
			name: "key collision between Parameters and ParametersFromSecret",
			spec: IntegrationSpec{
				Parameters: runtime.RawExtension{Raw: []byte(`{"Token":"inline"}`)},
				ParametersFromSecret: map[string]corev1.SecretKeySelector{
					"Token": {
						LocalObjectReference: corev1.LocalObjectReference{Name: "creds"},
						Key:                  "token",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "required ref, secret missing",
			spec: IntegrationSpec{
				ParametersFromSecret: map[string]corev1.SecretKeySelector{
					"Token": {
						LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
						Key:                  "token",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "required ref, key missing in secret",
			spec: IntegrationSpec{
				ParametersFromSecret: map[string]corev1.SecretKeySelector{
					"Token": {
						LocalObjectReference: corev1.LocalObjectReference{Name: "creds"},
						Key:                  "missing",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "optional ref, secret missing — skipped",
			spec: IntegrationSpec{
				Parameters: runtime.RawExtension{Raw: []byte(`{"AppName":"prod"}`)},
				ParametersFromSecret: map[string]corev1.SecretKeySelector{
					"Token": {
						LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
						Key:                  "token",
						Optional:             &ptrTrue,
					},
				},
			},
			wantParams:  map[string]string{"AppName": "prod"},
			wantMissing: []string{"Token"},
		},
		{
			name: "optional ref, key missing in secret — skipped",
			spec: IntegrationSpec{
				Parameters: runtime.RawExtension{Raw: []byte(`{"AppName":"prod"}`)},
				ParametersFromSecret: map[string]corev1.SecretKeySelector{
					"Token": {
						LocalObjectReference: corev1.LocalObjectReference{Name: "creds"},
						Key:                  "missing",
						Optional:             &ptrTrue,
					},
				},
			},
			wantParams:  map[string]string{"AppName": "prod"},
			wantMissing: []string{"Token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()
			config.InitClient(fakeClient)

			params, err := tt.spec.ExtractParameters(context.Background(), ns)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil; params=%v", params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := stringParamsByKey(params)
			for k, v := range tt.wantParams {
				if got[k] != v {
					t.Errorf("param %q: got %q, want %q", k, got[k], v)
				}
			}
			for _, k := range tt.wantMissing {
				if _, present := got[k]; present {
					t.Errorf("param %q should be absent (optional skip), got %q", k, got[k])
				}
			}
		})
	}
}

func stringParamsByKey(params []integrations.Parameter) map[string]string {
	out := map[string]string{}
	for _, p := range params {
		if p.ParameterStringValue != nil && p.ParameterStringValue.Key != nil {
			out[*p.ParameterStringValue.Key] = p.ParameterStringValue.StringValue
		}
	}
	return out
}
