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
	"maps"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

const secretRefTestValue = "fake-secret-value"

func secretRefTestKey(key string) corev1.SecretKeySelector {
	return corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "fake-values"}, Key: key}
}

func secretRefTestClient(t *testing.T) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-values", Namespace: "test"},
		Data:       map[string][]byte{"token": []byte(secretRefTestValue)},
	}
	previous := config.GetClient()
	t.Cleanup(func() { config.InitClient(previous) })
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build())
}

func TestResolveSecretValue(t *testing.T) {
	secretRefTestClient(t)
	ctx := context.Background()
	ref := secretRefTestKey("token")

	if got, err := ResolveSecretValue(ctx, "test", "spec.token", nil, nil); err != nil || got != nil {
		t.Fatalf("neither source: got %v, %v; want nil, nil", got, err)
	}
	if got, err := ResolveSecretValue(ctx, "test", "spec.token", ptr.To("literal"), nil); err != nil || ptr.Deref(got, "") != "literal" {
		t.Fatalf("literal: got %v, %v", got, err)
	}
	if got, err := ResolveSecretValue(ctx, "test", "spec.token", nil, &ref); err != nil || ptr.Deref(got, "") != secretRefTestValue {
		t.Fatalf("Secret: got %v, %v", got, err)
	}
	if _, err := ResolveSecretValue(ctx, "test", "spec.token", ptr.To("literal"), &ref); err == nil {
		t.Fatal("two sources: want an error")
	}
	missing := secretRefTestKey("missing")
	_, err := ResolveSecretValue(ctx, "test", "spec.token", nil, &missing)
	if err == nil || !strings.Contains(err.Error(), "spec.token") {
		t.Fatalf("missing key: got %v, want an error that names the field", err)
	}
	if _, err := ResolveSecretValue(ctx, "other", "spec.token", nil, &ref); err == nil {
		t.Fatal("a Secret of another namespace: want an error")
	}
}

func TestResolveSecretValues(t *testing.T) {
	secretRefTestClient(t)
	ctx := context.Background()
	literals := map[string]string{"X-Team": "platform"}
	refs := map[string]corev1.SecretKeySelector{"Authorization": secretRefTestKey("token")}

	if got, err := ResolveSecretValues(ctx, "test", "spec.headers", literals, nil); err != nil || !maps.Equal(got, literals) {
		t.Fatalf("literals only: got %v, %v", got, err)
	}
	got, err := ResolveSecretValues(ctx, "test", "spec.headers", literals, refs)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"X-Team": "platform", "Authorization": secretRefTestValue}; !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if literals["Authorization"] != "" {
		t.Fatal("the literal map changed")
	}
	_, err = ResolveSecretValues(ctx, "test", "spec.headers", map[string]string{"Authorization": "literal"}, refs)
	if err == nil || strings.Contains(err.Error(), "literal\"") || strings.Contains(err.Error(), secretRefTestValue) {
		t.Fatalf("two sources: got %v, want an error without the values", err)
	}
	if _, err := ResolveSecretValues(ctx, "test", "spec.headers", nil, map[string]corev1.SecretKeySelector{"A": secretRefTestKey("missing")}); err == nil {
		t.Fatal("missing key: want an error")
	}
}
