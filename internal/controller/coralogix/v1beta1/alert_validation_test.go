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

package v1beta1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	coralogixv1beta1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
)

func minimalAlert(name string) *coralogixv1beta1.Alert {
	return &coralogixv1beta1.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: coralogixv1beta1.AlertSpec{
			Name:     name,
			Priority: "p5",
			TypeDefinition: coralogixv1beta1.AlertTypeDefinition{
				LogsImmediate: &coralogixv1beta1.LogsImmediate{},
			},
		},
	}
}

var _ = Describe("Alert validation", func() {
	It("should reject a phantom alert that also sets a notification group", func(ctx context.Context) {
		alert := minimalAlert("phantom-with-notification-group")
		alert.Spec.PhantomMode = true
		alert.Spec.NotificationGroup = &coralogixv1beta1.NotificationGroup{}

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Phantom alerts must not have a notification group set"))
	})

	It("should accept a non-phantom alert with a notification group", func(ctx context.Context) {
		alert := minimalAlert("non-phantom-with-notification-group")
		alert.Spec.PhantomMode = false
		alert.Spec.NotificationGroup = &coralogixv1beta1.NotificationGroup{}

		Expect(k8sClient.Create(ctx, alert)).To(Succeed())
		Expect(k8sClient.Delete(ctx, alert)).To(Succeed())
	})

	It("should accept a phantom alert with no notification group", func(ctx context.Context) {
		alert := minimalAlert("phantom-without-notification-group")
		alert.Spec.PhantomMode = true

		Expect(k8sClient.Create(ctx, alert)).To(Succeed())
		Expect(k8sClient.Delete(ctx, alert)).To(Succeed())
	})
})
