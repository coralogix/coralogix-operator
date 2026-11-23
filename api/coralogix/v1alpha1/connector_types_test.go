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
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	connectorsservice "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/connectors_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// connectorTestSentinel returns a random value that is built at run time.
func connectorTestSentinel(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(buf)
}

func connectorTestClient(t *testing.T, data map[string][]byte) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "example-credentials", Namespace: "test"},
		Data:       data,
	}
	previousClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(previousClient) })
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build())
}

func templateSecretRef(key string) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: "example-credentials"},
		Key:                  key,
	}
}

func connectorWithOverrideFields(fields ...ConnectorTemplatedConnectorConfigField) *Connector {
	return &Connector{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test"},
		Spec: ConnectorSpec{
			Name:            "example",
			Type:            "pagerDuty",
			ConnectorConfig: ConnectorConfig{Fields: []ConnectorNotificationCenterConnectorConfigField{{FieldName: "integrationKey", Value: new(string)}}},
			ConfigOverrides: []ConnectorEntityTypeConfigOverrides{{EntityType: "alerts", Fields: fields}},
		},
	}
}

func TestConnectorOverrideTemplateReadsSecret(t *testing.T) {
	sentinel := connectorTestSentinel(t)
	connectorTestClient(t, map[string][]byte{"template": []byte(sentinel)})
	connector := connectorWithOverrideFields(
		ConnectorTemplatedConnectorConfigField{FieldName: "from-secret", TemplateSecretKeyRef: templateSecretRef("template")},
		ConnectorTemplatedConnectorConfigField{FieldName: "literal", Template: new("example-value")},
	)

	// The operator reads the Secret on every create and update.
	create, err := connector.CreateRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	update, err := connector.UpdateRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, fields := range [][]connectorsservice.TemplatedConnectorConfigField{create.Connector.ConfigOverrides[0].Fields, update.Connector.ConfigOverrides[0].Fields} {
		if len(fields) != 2 || fields[0].GetTemplate() != sentinel || fields[1].GetTemplate() != "example-value" {
			t.Fatalf("override templates do not match the sources")
		}
	}
}

func TestConnectorOverrideTemplateSecretErrorsHideValues(t *testing.T) {
	sentinel := connectorTestSentinel(t)
	connectorTestClient(t, map[string][]byte{"template": []byte(sentinel)})

	missingKey := connectorWithOverrideFields(ConnectorTemplatedConnectorConfigField{FieldName: "f", TemplateSecretKeyRef: templateSecretRef("absent")})
	_, err := missingKey.CreateRequest(context.Background())
	if err == nil || !strings.Contains(err.Error(), "fields[].template") || !strings.Contains(err.Error(), "absent") || !strings.Contains(err.Error(), "example-credentials") {
		t.Fatalf("missing key error = %v", err)
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatal("error contains the secret value")
	}

	missingSecret := connectorWithOverrideFields(ConnectorTemplatedConnectorConfigField{FieldName: "f", TemplateSecretKeyRef: &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: "other"}, Key: "template"}})
	if _, err := missingSecret.CreateRequest(context.Background()); err == nil || !strings.Contains(err.Error(), "other") {
		t.Fatalf("missing secret error = %v", err)
	}

	// The reference reads the namespace of the Connector, not another namespace.
	cross := connectorWithOverrideFields(ConnectorTemplatedConnectorConfigField{FieldName: "f", TemplateSecretKeyRef: templateSecretRef("template")})
	cross.Namespace = "another"
	if _, err := cross.CreateRequest(context.Background()); err == nil {
		t.Fatal("a Secret in another namespace must not be read")
	}
}

func TestConnectorOverrideTemplateRejectsTwoSources(t *testing.T) {
	sentinel := connectorTestSentinel(t)
	connectorTestClient(t, map[string][]byte{"template": []byte(sentinel)})
	connector := connectorWithOverrideFields(ConnectorTemplatedConnectorConfigField{FieldName: "f", Template: new("example-value"), TemplateSecretKeyRef: templateSecretRef("template")})
	_, err := connector.CreateRequest(context.Background())
	if err == nil || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("two sources error = %v", err)
	}
}
