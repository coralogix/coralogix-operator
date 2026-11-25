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

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("OutboundWebhook Secret reference validation", func() {
	ref := &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "webhook-values"}, Key: "field"}
	newWebhook := func(name string, webhookType coralogixv1alpha1.OutboundWebhookType) *coralogixv1alpha1.OutboundWebhook {
		return &coralogixv1alpha1.OutboundWebhook{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: coralogixv1alpha1.OutboundWebhookSpec{
				Name:                name,
				OutboundWebhookType: webhookType,
			},
		}
	}

	It("accepts the old literal field and the new Secret reference", func(ctx context.Context) {
		literal := newWebhook("pagerduty-literal", coralogixv1alpha1.OutboundWebhookType{PagerDuty: &coralogixv1alpha1.PagerDuty{ServiceKey: "fake-literal-service-key"}})
		Expect(k8sClient.Create(ctx, literal, client.DryRunAll)).To(Succeed())
		secret := newWebhook("pagerduty-secret", coralogixv1alpha1.OutboundWebhookType{PagerDuty: &coralogixv1alpha1.PagerDuty{ServiceKeySecretKeyRef: ref}})
		Expect(k8sClient.Create(ctx, secret, client.DryRunAll)).To(Succeed())
		url := newWebhook("generic-secret-url", coralogixv1alpha1.OutboundWebhookType{GenericWebhook: &coralogixv1alpha1.GenericWebhook{Method: coralogixv1alpha1.GenericWebhookMethodTypePost, UrlSecretKeyRef: ref}})
		Expect(k8sClient.Create(ctx, url, client.DryRunAll)).To(Succeed())
	})

	It("rejects missing and duplicate sources", func(ctx context.Context) {
		missing := newWebhook("pagerduty-missing", coralogixv1alpha1.OutboundWebhookType{PagerDuty: &coralogixv1alpha1.PagerDuty{}})
		Expect(k8sClient.Create(ctx, missing, client.DryRunAll)).ToNot(Succeed())
		duplicate := newWebhook("pagerduty-duplicate", coralogixv1alpha1.OutboundWebhookType{PagerDuty: &coralogixv1alpha1.PagerDuty{ServiceKey: "fake-literal-service-key", ServiceKeySecretKeyRef: ref}})
		Expect(k8sClient.Create(ctx, duplicate, client.DryRunAll)).ToNot(Succeed())
		header := newWebhook("generic-duplicate-header", coralogixv1alpha1.OutboundWebhookType{GenericWebhook: &coralogixv1alpha1.GenericWebhook{
			Method:            coralogixv1alpha1.GenericWebhookMethodTypePost,
			Url:               "https://example.invalid",
			Headers:           map[string]string{"Authorization": "fake-literal-header-value"},
			HeadersFromSecret: map[string]corev1.SecretKeySelector{"Authorization": *ref},
		}})
		Expect(k8sClient.Create(ctx, header, client.DryRunAll)).ToNot(Succeed())
	})
})
