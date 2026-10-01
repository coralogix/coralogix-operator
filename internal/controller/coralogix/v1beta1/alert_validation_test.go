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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

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

func analyticsImmediateAlert(name string) *coralogixv1beta1.Alert {
	alert := minimalAlert(name)
	alert.Spec.TypeDefinition = coralogixv1beta1.AlertTypeDefinition{
		AnalyticsImmediate: &coralogixv1beta1.AnalyticsImmediate{
			DataprimeQuery:   coralogixv1beta1.DataprimeQuery{Query: "source logs | count"},
			TimeframeMinutes: 15,
		},
	}
	return alert
}

func analyticsThresholdRule(threshold string, priority coralogixv1beta1.AlertPriority) coralogixv1beta1.AnalyticsThresholdRule {
	return coralogixv1beta1.AnalyticsThresholdRule{
		Condition: coralogixv1beta1.AnalyticsThresholdRuleCondition{Threshold: resource.MustParse(threshold)},
		Override:  &coralogixv1beta1.AlertOverride{Priority: priority},
	}
}

func analyticsThresholdAlert(name string, rules ...coralogixv1beta1.AnalyticsThresholdRule) *coralogixv1beta1.Alert {
	alert := minimalAlert(name)
	alert.Spec.TypeDefinition = coralogixv1beta1.AlertTypeDefinition{
		AnalyticsThreshold: &coralogixv1beta1.AnalyticsThreshold{
			DataprimeQuery:   coralogixv1beta1.DataprimeQuery{Query: "source logs | count as c"},
			TimeframeMinutes: 15,
			Rules:            rules,
		},
	}
	return alert
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

	It("should accept a minimal analytics immediate alert", func(ctx context.Context) {
		alert := analyticsImmediateAlert("analytics-immediate-minimal")
		alert.Spec.GroupByKeys = []string{"applicationname"}

		Expect(k8sClient.Create(ctx, alert)).To(Succeed())
		Expect(k8sClient.Delete(ctx, alert)).To(Succeed())
	})

	It("should accept a minimal analytics threshold alert", func(ctx context.Context) {
		alert := analyticsThresholdAlert("analytics-threshold-minimal", coralogixv1beta1.AnalyticsThresholdRule{
			Condition: coralogixv1beta1.AnalyticsThresholdRuleCondition{Threshold: resource.MustParse("1")},
		})

		Expect(k8sClient.Create(ctx, alert)).To(Succeed())
		Expect(k8sClient.Delete(ctx, alert)).To(Succeed())
	})

	It("should reject an alert with both analytics types set", func(ctx context.Context) {
		alert := analyticsImmediateAlert("analytics-both-types")
		alert.Spec.TypeDefinition.AnalyticsThreshold = analyticsThresholdAlert("unused", analyticsThresholdRule("1", "p1")).Spec.TypeDefinition.AnalyticsThreshold

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("analyticsImmediate, analyticsThreshold must be set"))
	})

	It("should reject an analytics alert combined with another alert type", func(ctx context.Context) {
		alert := analyticsImmediateAlert("analytics-with-logs-immediate")
		alert.Spec.TypeDefinition.LogsImmediate = &coralogixv1beta1.LogsImmediate{}

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("analyticsImmediate, analyticsThreshold must be set"))
	})

	It("should reject an analytics alert with a zero time frame", func(ctx context.Context) {
		alert := analyticsImmediateAlert("analytics-zero-timeframe")
		alert.Spec.TypeDefinition.AnalyticsImmediate.TimeframeMinutes = 0

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("timeframeMinutes"))
	})

	It("should reject an analytics alert with an evaluation delay above three hours", func(ctx context.Context) {
		alert := analyticsImmediateAlert("analytics-evaluation-delay-too-big")
		alert.Spec.TypeDefinition.AnalyticsImmediate.EvaluationDelayMs = ptr.To(int32(10800001))

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("evaluationDelayMs"))
	})

	It("should reject an analytics alert with an empty query", func(ctx context.Context) {
		alert := analyticsImmediateAlert("analytics-empty-query")
		alert.Spec.TypeDefinition.AnalyticsImmediate.DataprimeQuery.Query = ""

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dataprimeQuery.query"))
	})

	It("should reject an analytics threshold alert without rules", func(ctx context.Context) {
		err := k8sClient.Create(ctx, analyticsThresholdAlert("analytics-threshold-no-rules"))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("rules"))
	})

	It("should reject an analytics threshold alert with more than five rules", func(ctx context.Context) {
		alert := analyticsThresholdAlert("analytics-threshold-six-rules",
			analyticsThresholdRule("1", "p1"), analyticsThresholdRule("2", "p2"), analyticsThresholdRule("3", "p3"),
			analyticsThresholdRule("4", "p4"), analyticsThresholdRule("5", "p5"), analyticsThresholdRule("6", "p5"))

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("must have at most 5 items"))
	})

	It("should reject an analytics threshold alert with an invalid operator", func(ctx context.Context) {
		alert := analyticsThresholdAlert("analytics-threshold-invalid-operator", analyticsThresholdRule("1", "p1"))
		alert.Spec.TypeDefinition.AnalyticsThreshold.Operator = ptr.To(coralogixv1beta1.AnalyticsThresholdOperator("greaterThan"))

		err := k8sClient.Create(ctx, alert)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("operator"))
	})
})
