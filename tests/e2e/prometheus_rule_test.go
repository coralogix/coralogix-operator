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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("PrometheusRule", Ordered, func() {
	var (
		crClient             client.Client
		promRule             *prometheus.PrometheusRule
		promRuleName         = "prometheus-rules"
		alertName            = "test-alert"
		alertResourceName    = promRuleName + "-" + alertName + "-0"
		newAlertName         = "test-alert-updated"
		newAlertResourceName = promRuleName + "-" + newAlertName + "-0"
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating PrometheusRule")
		promRule = &prometheus.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      promRuleName,
				Namespace: testNamespace,
				Labels: map[string]string{
					"app.coralogix.com/track-recording-rules": "true",
					"app.coralogix.com/track-alerting-rules":  "true",
				},
			},
			Spec: prometheus.PrometheusRuleSpec{
				Groups: []prometheus.RuleGroup{
					{
						Name:     "example.rules",
						Interval: "60s",
						Rules: []prometheus.Rule{
							{
								Alert: alertName,
								Expr:  intstr.FromString("up == 0"), // Short test expression
								For:   "5m",
								Annotations: map[string]string{
									"cxMinNonNullValuesPercentage": "20",
								},
								Labels: map[string]string{
									"severity":      "critical",
									"slack_channel": "#observability",
								},
							},
							{
								Record: "ExampleRecord",
								Expr:   intstr.FromString("vector(1)"),
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, promRule)).To(Succeed())

		By("Verifying underlying Alert and RecordingRuleGroupSet were created")
		fetchedAlert := &coralogixv1alpha1.Alert{}
		Eventually(func() error {
			return crClient.Get(ctx, types.NamespacedName{Name: alertResourceName, Namespace: testNamespace}, fetchedAlert)
		}, time.Minute, time.Second).Should(Succeed())

		fetchedRecordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{}
		Eventually(func() error {
			return crClient.Get(ctx, types.NamespacedName{Name: promRuleName, Namespace: testNamespace}, fetchedRecordingRuleGroupSet)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the PrometheusRule")
		modifiedPromRule := promRule.DeepCopy()
		modifiedPromRule.Spec.Groups[0].Rules[0].Alert = newAlertName
		Expect(crClient.Patch(ctx, modifiedPromRule, client.MergeFrom(promRule))).To(Succeed())

		By("Verifying underlying Alert was updated")
		Eventually(func() error {
			fetchedAlert := &coralogixv1alpha1.Alert{}
			return crClient.Get(ctx, types.NamespacedName{Name: newAlertResourceName, Namespace: testNamespace}, fetchedAlert)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the PrometheusRule")
		Expect(crClient.Delete(ctx, promRule)).To(Succeed())

		By("Verifying underlying Alert and RecordingRuleGroupSet were deleted")
		fetchedAlert := &coralogixv1alpha1.Alert{}
		Eventually(func() bool {
			err := crClient.Get(ctx, types.NamespacedName{Name: newAlertResourceName, Namespace: testNamespace}, fetchedAlert)
			return errors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())

		fetchedRecordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{}
		Eventually(func() bool {
			err := crClient.Get(ctx, types.NamespacedName{Name: promRuleName, Namespace: testNamespace}, fetchedRecordingRuleGroupSet)
			return errors.IsNotFound(err)
		}, time.Minute, time.Second).Should(BeTrue())
	})
})