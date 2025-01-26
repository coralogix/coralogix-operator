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

package e2e

import (
	"context"
	"time"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/common"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("AlertmanagerConfig", Ordered, func() {
	var (
		crClient client.Client
		config   *prometheus.AlertmanagerConfig
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating secret")
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "slack-webhook-secret",
				Namespace: testNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"webhook-url": []byte("aHR0cHM6Ly9zbGFjay5jb20vYXBpL2NoYXQucG9zdE1lc3NhZ2U="),
			},
		}
		Expect(crClient.Create(ctx, secret)).To(Succeed())

		By("Creating AlertmanagerConfig")
		config = &prometheus.AlertmanagerConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "monitoring.coreos.com/v1alpha1",
				Kind:       "AlertmanagerConfig",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "slack-config",
				Namespace: testNamespace,
				Labels: map[string]string{
					"app.coralogix.com/track-alertmanager-config": "true",
				},
			},
			Spec: prometheus.AlertmanagerConfigSpec{
				Route: &prometheus.Route{
					GroupBy:        []string{"alertname"},
					Receiver:       "slack-default",
					RepeatInterval: "3h",
					Routes: []apiextensionsv1.JSON{
						{
							Raw: []byte(`{
							"receiver": "slack-general",
							"matchers": [
								{
									"matchType": "=~",
									"name": "slack_channel",
									"value": ".+"
								}
							],
							"continue": true
						}`),
						},
					},
				},
				Receivers: []prometheus.Receiver{
					{
						Name: "slack-general",
						SlackConfigs: []prometheus.SlackConfig{
							{
								APIURL: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "slack-webhook-secret",
									},
									Key: "webhook-url",
								},
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, config)).To(Succeed())

		By("Verifying underlying OutboundWebhook was created")
		fetchedOutboundWebhook := &coralogixv1alpha1.OutboundWebhook{}
		Eventually(func() error {
			return crClient.Get(ctx, types.NamespacedName{Name: "slack-general.slack.0", Namespace: testNamespace}, fetchedOutboundWebhook)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the AlertmanagerConfig")
		modifiedConfig := config.DeepCopy()
		modifiedConfig.Spec.Receivers[0].Name = "slack-general-updated"
		Expect(crClient.Patch(ctx, modifiedConfig, client.MergeFrom(config))).To(Succeed())

		By("Verifying underlying outboundWebhook was updated")
		Eventually(func() error {
			fetchedOutboundWebhook := &coralogixv1alpha1.OutboundWebhook{}
			return crClient.Get(ctx, types.NamespacedName{Name: "slack-general-updated.slack.0", Namespace: testNamespace}, fetchedOutboundWebhook)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the AlertmanagerConfig")
		Expect(crClient.Delete(ctx, config)).To(Succeed())

		By("Verifying underlying outboundWebhook was deleted")
		fetchedOutboundWebhook := &coralogixv1alpha1.OutboundWebhook{}
		Eventually(func() bool {
			err := crClient.Get(ctx, types.NamespacedName{Name: "slack-general-updated.slack.0", Namespace: testNamespace}, fetchedOutboundWebhook)
			return errors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())
	})
})
