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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var _ = Describe("Connector override template validation", func() {
	ref := &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "example-credentials"}, Key: "template"}
	newConnector := func(name string, field coralogixv1alpha1.TemplatedConnectorConfigField) *coralogixv1alpha1.Connector {
		value := "example-value"
		return &coralogixv1alpha1.Connector{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: coralogixv1alpha1.ConnectorSpec{
				Name:            name,
				Description:     "example",
				Type:            "pagerDuty",
				ConnectorConfig: coralogixv1alpha1.ConnectorConfig{Fields: []coralogixv1alpha1.ConnectorConfigField{{FieldName: "integrationKey", Value: &value}}},
				ConfigOverrides: []coralogixv1alpha1.EntityTypeConfigOverrides{{EntityType: "alerts", Fields: []coralogixv1alpha1.TemplatedConnectorConfigField{field}}},
			},
		}
	}

	It("accepts the old literal template and the new Secret reference", func(ctx context.Context) {
		literal := newConnector("template-literal", coralogixv1alpha1.TemplatedConnectorConfigField{FieldName: "integrationKey", Template: "example-value"})
		Expect(k8sClient.Create(ctx, literal, client.DryRunAll)).To(Succeed())
		secret := newConnector("template-secret", coralogixv1alpha1.TemplatedConnectorConfigField{FieldName: "integrationKey", TemplateSecretKeyRef: ref})
		Expect(k8sClient.Create(ctx, secret, client.DryRunAll)).To(Succeed())
	})

	It("rejects a missing source and two sources", func(ctx context.Context) {
		missing := newConnector("template-missing", coralogixv1alpha1.TemplatedConnectorConfigField{FieldName: "integrationKey"})
		Expect(k8sClient.Create(ctx, missing, client.DryRunAll)).ToNot(Succeed())
		both := newConnector("template-both", coralogixv1alpha1.TemplatedConnectorConfigField{FieldName: "integrationKey", Template: "example-value", TemplateSecretKeyRef: ref})
		Expect(k8sClient.Create(ctx, both, client.DryRunAll)).ToNot(Succeed())
	})
})
