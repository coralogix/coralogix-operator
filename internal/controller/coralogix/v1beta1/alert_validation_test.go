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
	"fmt"

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

func minimalAlertSet(name string, itemCount int) *coralogixv1beta1.AlertSet {
	items := make([]coralogixv1beta1.AlertSetItem, itemCount)
	for i := range itemCount {
		key := fmt.Sprintf("alert-%03d", i)
		items[i] = minimalAlertSetItem(key)
	}
	return &coralogixv1beta1.AlertSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: coralogixv1beta1.AlertSetSpec{Alerts: items},
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

var _ = Describe("AlertSet validation", func() {
	DescribeTable("should enforce the item count",
		func(ctx context.Context, count int, shouldSucceed bool) {
			alertSet := minimalAlertSet(fmt.Sprintf("alert-set-count-%d", count), count)
			err := k8sClient.Create(ctx, alertSet)
			if shouldSucceed {
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, alertSet)).To(Succeed())
				return
			}
			Expect(err).To(HaveOccurred())
		},
		Entry("accepts one item", 1, true),
		Entry("accepts 100 items", 100, true),
		Entry("rejects zero items", 0, false),
		Entry("rejects 101 items", 101, false),
	)

	It("should reject duplicate keys", func(ctx context.Context) {
		alertSet := minimalAlertSet("alert-set-duplicate-keys", 2)
		alertSet.Spec.Alerts[1].Key = alertSet.Spec.Alerts[0].Key

		err := k8sClient.Create(ctx, alertSet)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate"))
	})

	DescribeTable("should reject invalid keys",
		func(ctx context.Context, key string) {
			alertSet := minimalAlertSet("alert-set-invalid-key", 1)
			alertSet.Spec.Alerts[0].Key = key

			Expect(k8sClient.Create(ctx, alertSet)).NotTo(Succeed())
		},
		Entry("empty", ""),
		Entry("uppercase", "Alert"),
		Entry("underscore", "alert_key"),
		Entry("more than 63 characters", "a123456789012345678901234567890123456789012345678901234567890123"),
	)
})
