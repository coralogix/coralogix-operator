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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var _ = Describe("Connector override template validation", func() {
	ref := &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "example-credentials"}, Key: "template"}
	newConnector := func(name string, field coralogixv1alpha1.ConnectorTemplatedConnectorConfigField) *coralogixv1alpha1.Connector {
		value := "example-value"
		return &coralogixv1alpha1.Connector{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: coralogixv1alpha1.ConnectorSpec{
				Name:            name,
				Description:     "example",
				Type:            "pagerDuty",
				ConnectorConfig: coralogixv1alpha1.ConnectorConfig{Fields: []coralogixv1alpha1.ConnectorNotificationCenterConnectorConfigField{{FieldName: "integrationKey", Value: &value}}},
				ConfigOverrides: []coralogixv1alpha1.ConnectorEntityTypeConfigOverrides{{EntityType: "alerts", Fields: []coralogixv1alpha1.ConnectorTemplatedConnectorConfigField{field}}},
			},
		}
	}

	It("accepts the old literal template and the new Secret reference", func(ctx context.Context) {
		literal := newConnector("template-literal", coralogixv1alpha1.ConnectorTemplatedConnectorConfigField{FieldName: "integrationKey", Template: new("example-value")})
		Expect(k8sClient.Create(ctx, literal, client.DryRunAll)).To(Succeed())
		secret := newConnector("template-secret", coralogixv1alpha1.ConnectorTemplatedConnectorConfigField{FieldName: "integrationKey", TemplateSecretKeyRef: ref})
		Expect(k8sClient.Create(ctx, secret, client.DryRunAll)).To(Succeed())
	})

	It("rejects two sources with a clear message", func(ctx context.Context) {
		both := newConnector("template-both", coralogixv1alpha1.ConnectorTemplatedConnectorConfigField{FieldName: "integrationKey", Template: new("example-value"), TemplateSecretKeyRef: ref})
		err := k8sClient.Create(ctx, both, client.DryRunAll)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsInvalid(err)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("Only one of template or templateSecretKeyRef can be set"))
	})

	It("keeps a stored empty template valid through a typed update and delete", func(ctx context.Context) {
		old := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "coralogix.com/v1alpha1",
			"kind":       "Connector",
			"metadata":   map[string]interface{}{"name": "template-empty", "namespace": "default"},
			"spec": map[string]interface{}{
				"name":        "template-empty",
				"description": "example",
				"type":        "pagerDuty",
				"connectorConfig": map[string]interface{}{"fields": []interface{}{
					map[string]interface{}{"fieldName": "integrationKey", "value": "example-value"},
				}},
				"configOverrides": []interface{}{map[string]interface{}{
					"entityType": "alerts",
					"fields":     []interface{}{map[string]interface{}{"fieldName": "integrationKey", "template": ""}},
				}},
			},
		}}
		Expect(k8sClient.Create(ctx, old)).To(Succeed())

		// The typed object keeps the empty template (a set *string) and writes it back. The update must still pass admission.
		typed := &coralogixv1alpha1.Connector{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(old), typed)).To(Succeed())
		controllerutil.AddFinalizer(typed, "connector.coralogix.com/finalizer")
		Expect(k8sClient.Update(ctx, typed)).To(Succeed())
		controllerutil.RemoveFinalizer(typed, "connector.coralogix.com/finalizer")
		Expect(k8sClient.Update(ctx, typed)).To(Succeed())

		Expect(k8sClient.Delete(ctx, typed)).To(Succeed())
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(old), &coralogixv1alpha1.Connector{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})
})
