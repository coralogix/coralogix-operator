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

package v1alpha1

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

var _ = Describe("AlertScheduler validation", func() {
	It("accepts all alert scheduler selector modes", func(ctx context.Context) {
		for _, tt := range []struct {
			name   string
			filter coralogixv1alpha1.Filter
		}{
			{
				name: "all-alerts",
				filter: coralogixv1alpha1.Filter{
					WhatExpression: "source logs | filter true",
					AllAlerts:      ptr.To(true),
				},
			},
			{
				name: "meta-labels",
				filter: coralogixv1alpha1.Filter{
					WhatExpression: "source logs | filter true",
					MetaLabels: []coralogixv1alpha1.MetaLabel{
						{Key: "environment", Value: ptr.To("production")},
					},
				},
			},
			{
				name: "alert-resource-refs",
				filter: coralogixv1alpha1.Filter{
					WhatExpression: "source logs | filter true",
					Alerts: []coralogixv1alpha1.AlertRef{
						{ResourceRef: &coralogixv1alpha1.ResourceRef{Name: "referenced-alert"}},
					},
				},
			},
			{
				name: "alert-unique-ids",
				filter: coralogixv1alpha1.Filter{
					WhatExpression: "source logs | filter true",
					AlertUniqueIDs: []string{"backend-alert-id"},
				},
			},
		} {
			scheduler := validAlertScheduler("valid-"+tt.name, tt.filter)
			Expect(k8sClient.Create(ctx, scheduler)).To(Succeed())
			Expect(k8sClient.Delete(ctx, scheduler)).To(Succeed())
		}
	})

	It("rejects allAlerts false", func(ctx context.Context) {
		scheduler := validAlertScheduler("invalid-all-alerts-false", coralogixv1alpha1.Filter{
			WhatExpression: "source logs | filter true",
			AllAlerts:      ptr.To(false),
		})

		err := k8sClient.Create(ctx, scheduler)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("allAlerts must be true when set"))
	})

	It("rejects filters with multiple selector modes", func(ctx context.Context) {
		scheduler := validAlertScheduler("invalid-multiple-selectors", coralogixv1alpha1.Filter{
			WhatExpression: "source logs | filter true",
			AllAlerts:      ptr.To(true),
			MetaLabels: []coralogixv1alpha1.MetaLabel{
				{Key: "environment", Value: ptr.To("production")},
			},
		})

		err := k8sClient.Create(ctx, scheduler)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of allAlerts, metaLabels, alerts, or alertUniqueIds must be set"))
	})

	It("rejects filters with no selector mode", func(ctx context.Context) {
		scheduler := validAlertScheduler("invalid-no-selector", coralogixv1alpha1.Filter{
			WhatExpression: "source logs | filter true",
		})

		err := k8sClient.Create(ctx, scheduler)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of allAlerts, metaLabels, alerts, or alertUniqueIds must be set"))
	})

	It("rejects empty direct alert IDs", func(ctx context.Context) {
		scheduler := validAlertScheduler("invalid-empty-alert-id", coralogixv1alpha1.Filter{
			WhatExpression: "source logs | filter true",
			AlertUniqueIDs: []string{"backend-alert-id", ""},
		})

		err := k8sClient.Create(ctx, scheduler)
		Expect(err).To(HaveOccurred())
		Expect(strings.ToLower(err.Error())).To(ContainSubstring("alertuniqueids"))
	})
})

func validAlertScheduler(name string, filter coralogixv1alpha1.Filter) *coralogixv1alpha1.AlertScheduler {
	return &coralogixv1alpha1.AlertScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: coralogixv1alpha1.AlertSchedulerSpec{
			Name:        name,
			Description: "validation test alert scheduler",
			Enabled:     true,
			Filter:      filter,
			Schedule: coralogixv1alpha1.Schedule{
				Operation: "mute",
				Recurring: &coralogixv1alpha1.Recurring{
					Always: &coralogixv1alpha1.Always{},
				},
			},
		},
	}
}
