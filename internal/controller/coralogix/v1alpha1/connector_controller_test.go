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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func connectorSecretRef(name string) *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: "key"}
}

func TestConnectorSecretNamesCoverFieldsAndTemplates(t *testing.T) {
	connector := &coralogixv1alpha1.Connector{Spec: coralogixv1alpha1.ConnectorSpec{
		ConnectorConfig: coralogixv1alpha1.ConnectorConfig{Fields: []coralogixv1alpha1.ConnectorConfigField{
			{FieldName: "a", SecretKeyRef: connectorSecretRef("shared")},
			{FieldName: "b", Value: new(string)},
		}},
		ConfigOverrides: []coralogixv1alpha1.EntityTypeConfigOverrides{{EntityType: "alerts", Fields: []coralogixv1alpha1.TemplatedConnectorConfigField{
			{FieldName: "a", TemplateSecretKeyRef: connectorSecretRef("shared")},
			{FieldName: "c", TemplateSecretKeyRef: connectorSecretRef("templates")},
			{FieldName: "d", Template: "example-value"},
		}}},
	}}
	names := connectorSecretNames(connector)
	if len(names) != 2 || names[0] != "shared" || names[1] != "templates" {
		t.Fatalf("Secret names = %v", names)
	}
}

func TestConnectorSecretWatchSelectsReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := coralogixv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	withTemplate := func(name, secret string) *coralogixv1alpha1.Connector {
		return &coralogixv1alpha1.Connector{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"},
			Spec: coralogixv1alpha1.ConnectorSpec{ConfigOverrides: []coralogixv1alpha1.EntityTypeConfigOverrides{{
				EntityType: "alerts",
				Fields:     []coralogixv1alpha1.TemplatedConnectorConfigField{{FieldName: "f", TemplateSecretKeyRef: connectorSecretRef(secret)}},
			}}},
		}
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(withTemplate("first", "shared"), withTemplate("second", "other")).
		WithIndex(&coralogixv1alpha1.Connector{}, connectorSecretIndex, func(obj client.Object) []string {
			return connectorSecretNames(obj.(*coralogixv1alpha1.Connector))
		}).Build()
	previousClient := config.GetClient()
	t.Cleanup(func() { config.InitClient(previousClient) })
	config.InitClient(c)
	previousSelector := config.GetConfig().Selector
	t.Cleanup(func() { config.GetConfig().Selector = previousSelector })

	secret := func(namespace string) *corev1.Secret {
		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: namespace}}
	}
	requests := (&ConnectorReconciler{}).connectorsForSecret(context.Background(), secret("test"))
	if len(requests) != 1 || requests[0].Name != "first" {
		t.Fatalf("Secret event requests = %v", requests)
	}
	if requests = (&ConnectorReconciler{}).connectorsForSecret(context.Background(), secret("another")); len(requests) != 0 {
		t.Fatalf("cross-namespace Secret event requests = %v", requests)
	}
	config.GetConfig().Selector = config.Selector{LabelSelector: labels.SelectorFromSet(labels.Set{"selected": "true"})}
	if requests = (&ConnectorReconciler{}).connectorsForSecret(context.Background(), secret("test")); len(requests) != 0 {
		t.Fatalf("unselected connector requests = %v", requests)
	}
}
