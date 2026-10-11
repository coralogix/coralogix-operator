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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cfggroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/fleet_manager_configuration_groups"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

func presetFamily() *coralogixv1alpha1.PresetConfigurationFamilySpec {
	return &coralogixv1alpha1.PresetConfigurationFamilySpec{
		ChartName:             "CHART_NAME_OTEL_INTEGRATION",
		ChartVersion:          "0.0.289",
		IntegrationVersion:    ptr.To("0.8.0"),
		Metadata:              map[string]string{"ClusterName": "prod"},
		ObservabilityFeatures: `{"logsCollection":{"enabled":true}}`,
	}
}

func rawFamily() *coralogixv1alpha1.RawConfigurationFamilySpec {
	return &coralogixv1alpha1.RawConfigurationFamilySpec{
		CollectorVersion: ptr.To("0.114.0"),
		RemoteConfigurations: []coralogixv1alpha1.RemoteConfigurationSpec{{
			Name:             ptr.To("default"),
			RawConfiguration: "receivers: {}",
		}},
	}
}

func configurationGroupWithFamily(name string, family coralogixv1alpha1.ConfigurationFamilySpec) *coralogixv1alpha1.ConfigurationGroup {
	return &coralogixv1alpha1.ConfigurationGroup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: coralogixv1alpha1.ConfigurationGroupSpec{
			Name:   name,
			Family: family,
		},
	}
}

var _ = Describe("ConfigurationGroup validation", func() {
	It("should accept a preset family", func(ctx context.Context) {
		group := configurationGroupWithFamily("cg-preset", coralogixv1alpha1.ConfigurationFamilySpec{Preset: presetFamily()})
		Expect(k8sClient.Create(ctx, group)).To(Succeed())
		Expect(k8sClient.Delete(ctx, group)).To(Succeed())
	})

	It("should accept a raw family", func(ctx context.Context) {
		group := configurationGroupWithFamily("cg-raw", coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()})
		Expect(k8sClient.Create(ctx, group)).To(Succeed())
		Expect(k8sClient.Delete(ctx, group)).To(Succeed())
	})

	DescribeTable("should reject switching a family between preset and raw",
		func(ctx context.Context, name string, from, to coralogixv1alpha1.ConfigurationFamilySpec) {
			group := configurationGroupWithFamily(name, from)
			Expect(k8sClient.Create(ctx, group)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(k8sClient.Delete(ctx, group)).To(Succeed())
			})

			group.Spec.Family = to
			err := k8sClient.Update(ctx, group)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot switch between preset and raw"))
		},
		Entry("raw to preset", "cg-raw-to-preset",
			coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()},
			coralogixv1alpha1.ConfigurationFamilySpec{Preset: presetFamily()}),
		Entry("preset to raw", "cg-preset-to-raw",
			coralogixv1alpha1.ConfigurationFamilySpec{Preset: presetFamily()},
			coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()}),
	)

	It("should accept updating the content of the same family kind", func(ctx context.Context) {
		group := configurationGroupWithFamily("cg-raw-update", coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()})
		Expect(k8sClient.Create(ctx, group)).To(Succeed())
		DeferCleanup(func(ctx context.Context) {
			Expect(k8sClient.Delete(ctx, group)).To(Succeed())
		})

		group.Spec.Family.Raw.CollectorVersion = ptr.To("0.115.0")
		group.Spec.Family.Description = ptr.To("updated")
		Expect(k8sClient.Update(ctx, group)).To(Succeed())
	})

	It("should leave family.active unset by default", func(ctx context.Context) {
		group := configurationGroupWithFamily("cg-active-default", coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()})
		Expect(k8sClient.Create(ctx, group)).To(Succeed())
		DeferCleanup(func(ctx context.Context) {
			Expect(k8sClient.Delete(ctx, group)).To(Succeed())
		})

		fetched := &coralogixv1alpha1.ConfigurationGroup{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(group), fetched)).To(Succeed())
		Expect(fetched.Spec.Family.Active).To(BeNil())
	})

	It("should reject a family with both preset and raw", func(ctx context.Context) {
		group := configurationGroupWithFamily("cg-both", coralogixv1alpha1.ConfigurationFamilySpec{
			Preset: presetFamily(),
			Raw:    rawFamily(),
		})
		err := k8sClient.Create(ctx, group)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of preset or raw is required"))
	})

	It("should reject a family with neither preset nor raw", func(ctx context.Context) {
		group := configurationGroupWithFamily("cg-neither", coralogixv1alpha1.ConfigurationFamilySpec{})
		err := k8sClient.Create(ctx, group)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of preset or raw is required"))
	})

	DescribeTable("should reject a chartName the API does not accept",
		func(ctx context.Context, chartName string) {
			preset := presetFamily()
			preset.ChartName = chartName
			group := configurationGroupWithFamily("cg-bad-chart", coralogixv1alpha1.ConfigurationFamilySpec{Preset: preset})
			Expect(k8sClient.Create(ctx, group)).NotTo(Succeed())
		},
		Entry("unspecified", "CHART_NAME_UNSPECIFIED"),
		Entry("opentelemetry collector", "CHART_NAME_OPENTELEMETRY_COLLECTOR"),
		Entry("camelCase spelling", "otelIntegration"),
	)

	DescribeTable("should reject a name the API does not accept",
		func(ctx context.Context, name string) {
			group := configurationGroupWithFamily("cg-bad-name", coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()})
			group.Spec.Name = name
			Expect(k8sClient.Create(ctx, group)).NotTo(Succeed())
		},
		Entry("empty", ""),
		Entry("longer than 256 characters", strings.Repeat("n", 257)),
	)

	DescribeTable("should reject values over the API size limits",
		func(ctx context.Context, mutate func(*coralogixv1alpha1.ConfigurationGroup)) {
			group := configurationGroupWithFamily("cg-too-big", coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()})
			mutate(group)
			Expect(k8sClient.Create(ctx, group)).NotTo(Succeed())
		},
		Entry("description", func(g *coralogixv1alpha1.ConfigurationGroup) {
			g.Spec.Description = ptr.To(strings.Repeat("d", 4097))
		}),
		Entry("family description", func(g *coralogixv1alpha1.ConfigurationGroup) {
			g.Spec.Family.Description = ptr.To(strings.Repeat("d", 4097))
		}),
		Entry("more than 64 tags", func(g *coralogixv1alpha1.ConfigurationGroup) {
			for i := range 65 {
				g.Spec.Tags = append(g.Spec.Tags, fmt.Sprintf("tag%d", i))
			}
		}),
		Entry("tag", func(g *coralogixv1alpha1.ConfigurationGroup) {
			g.Spec.Tags = []string{strings.Repeat("t", 257)}
		}),
		Entry("metadata value", func(g *coralogixv1alpha1.ConfigurationGroup) {
			g.Spec.Family.Raw.Metadata = map[string]string{"key": strings.Repeat("v", 4097)}
		}),
		Entry("remote configuration name", func(g *coralogixv1alpha1.ConfigurationGroup) {
			g.Spec.Family.Raw.RemoteConfigurations[0].Name = ptr.To(strings.Repeat("n", 257))
		}),
		Entry("more than 128 remote configurations", func(g *coralogixv1alpha1.ConfigurationGroup) {
			remote := g.Spec.Family.Raw.RemoteConfigurations[0]
			for range 128 {
				g.Spec.Family.Raw.RemoteConfigurations = append(g.Spec.Family.Raw.RemoteConfigurations, remote)
			}
		}),
	)

	It("should reject metadata with more than 64 entries", func(ctx context.Context) {
		raw := rawFamily()
		raw.Metadata = map[string]string{}
		for i := range 65 {
			raw.Metadata[fmt.Sprintf("key%d", i)] = "value"
		}
		group := configurationGroupWithFamily("cg-big-metadata", coralogixv1alpha1.ConfigurationFamilySpec{Raw: raw})
		Expect(k8sClient.Create(ctx, group)).NotTo(Succeed())
	})

})

var _ = Describe("ConfigurationGroup expansion", func() {
	It("should expand a preset family into create and update requests", func() {
		group := configurationGroupWithFamily("cg", coralogixv1alpha1.ConfigurationFamilySpec{Preset: presetFamily()})

		createReq, err := expandCreateRequest(group)
		Expect(err).NotTo(HaveOccurred())
		createFamily := createReq.Family
		Expect(createFamily.Raw).To(BeNil())
		Expect(createFamily.Preset).NotTo(BeNil())
		Expect(createFamily.Preset.ChartName).To(Equal(cfggroups.CHARTNAME_CHART_NAME_OTEL_INTEGRATION))
		Expect(createFamily.Preset.ChartVersion).To(Equal("0.0.289"))
		Expect(createFamily.Preset.GetIntegrationVersion()).To(Equal("0.8.0"))
		Expect(createFamily.Preset.ObservabilityFeatures).To(MatchJSON(`{"logsCollection":{"enabled":true}}`))

		updateReq, updateMask, err := expandUpdateRequest(group)
		Expect(err).NotTo(HaveOccurred())
		updateFamily := updateReq.Family
		Expect(updateMask).To(Equal("name,description,tags,priorityOrder,family.description,family.active,family.preset"))
		Expect(updateFamily.Raw).To(BeNil())
		Expect(updateFamily.Preset).NotTo(BeNil())
		Expect(updateFamily.Preset.ChartName).To(HaveValue(Equal(cfggroups.CHARTNAME_CHART_NAME_OTEL_INTEGRATION)))
		Expect(updateFamily.Preset.ChartVersion).To(HaveValue(Equal("0.0.289")))
		Expect(updateFamily.Preset.ObservabilityFeatures).To(HaveValue(MatchJSON(`{"logsCollection":{"enabled":true}}`)))
		Expect(updateFamily.Preset.Metadata).To(Equal(map[string]string{"ClusterName": "prod"}))
	})

	It("should expand a raw family into create and update requests", func() {
		group := configurationGroupWithFamily("cg", coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()})

		createReq, err := expandCreateRequest(group)
		Expect(err).NotTo(HaveOccurred())
		Expect(createReq.Family.Preset).To(BeNil())
		Expect(createReq.Family.Raw.GetCollectorVersion()).To(Equal("0.114.0"))
		Expect(createReq.Family.Raw.RemoteConfigurations).To(HaveLen(1))

		updateReq, updateMask, err := expandUpdateRequest(group)
		Expect(err).NotTo(HaveOccurred())
		Expect(updateReq.Family.Preset).To(BeNil())
		Expect(updateMask).To(Equal("name,description,tags,priorityOrder,family.description,family.active,family.raw"))
		Expect(updateReq.Family.Raw.RemoteConfigurations[0].RawConfiguration).To(Equal("receivers: {}"))
		Expect(updateReq.GetDescription()).To(Equal(""))
		Expect(updateReq.GetPriorityOrder()).To(Equal(int32(0)))
		Expect(updateReq.Family.Raw.Metadata).To(Equal(map[string]string{}))
		Expect(updateReq.Family.Raw.RemoteConfigurations).To(HaveLen(1))
		Expect(updateReq.Family.Active).To(Equal(ptr.To(false)))
	})

})
