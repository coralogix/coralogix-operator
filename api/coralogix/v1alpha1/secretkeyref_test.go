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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func TestResolveSecretKeyRef(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	previous := config.GetClient()
	t.Cleanup(func() { config.InitClient(previous) })
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "team"}, Data: map[string][]byte{"token": []byte("s3cret"), "empty": {}}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "other"}, Data: map[string][]byte{"token": []byte("other-s3cret")}},
	).Build())

	selector := func(name, key string, optional bool) corev1.SecretKeySelector {
		return corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: key, Optional: &optional}
	}
	tests := []struct {
		name      string
		namespace string
		ref       corev1.SecretKeySelector
		want      string
		wantErr   string
	}{
		{name: "found", namespace: "team", ref: selector("credentials", "token", false), want: "s3cret"},
		{name: "reads the namespace of the resource", namespace: "other", ref: selector("credentials", "token", false), want: "other-s3cret"},
		{name: "empty value", namespace: "team", ref: selector("credentials", "empty", false), want: ""},
		{name: "missing secret", namespace: "team", ref: selector("absent", "token", false), wantErr: "failed to get secret 'absent'"},
		{name: "missing key", namespace: "team", ref: selector("credentials", "absent", false), wantErr: "cannot find key 'absent' in secret 'credentials'"},
		{name: "optional does not skip a missing secret", namespace: "team", ref: selector("absent", "token", true), wantErr: "failed to get secret 'absent'"},
		{name: "optional does not skip a missing key", namespace: "team", ref: selector("credentials", "absent", true), wantErr: "cannot find key 'absent' in secret 'credentials'"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveSecretKeyRef(context.Background(), test.namespace, test.ref)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want one that contains %q", err, test.wantErr)
				}
				if strings.Contains(err.Error(), "s3cret") {
					t.Fatalf("the error holds a secret value: %v", err)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
		})
	}
}
