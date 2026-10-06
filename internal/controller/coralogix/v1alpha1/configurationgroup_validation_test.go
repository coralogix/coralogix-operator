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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	cfggroups "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/fleet_manager_configuration_groups"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

func presetFamily() *coralogixv1alpha1.PresetConfigurationFamilySpec {
	return &coralogixv1alpha1.PresetConfigurationFamilySpec{
		ChartName:             "otelIntegration",
		ChartVersion:          "0.0.289",
		IntegrationVersion:    ptr.To("0.8.0"),
		Metadata:              map[string]string{"ClusterName": "prod"},
		ObservabilityFeatures: runtime.RawExtension{Raw: []byte(`{"logsCollection":{"enabled":true}}`)},
	}
}

func rawFamily() *coralogixv1alpha1.RawConfigurationFamilySpec {
	return &coralogixv1alpha1.RawConfigurationFamilySpec{
		CollectorVersion: ptr.To("0.114.0"),
		RemoteConfigurations: []coralogixv1alpha1.RemoteConfigurationSpec{{
			Name:             "default",
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

	It("should reject an unknown preset chartName", func(ctx context.Context) {
		preset := presetFamily()
		preset.ChartName = "CHART_NAME_UNSPECIFIED"
		group := configurationGroupWithFamily("cg-bad-chart", coralogixv1alpha1.ConfigurationFamilySpec{Preset: preset})
		Expect(k8sClient.Create(ctx, group)).NotTo(Succeed())
	})
})

var _ = Describe("ConfigurationGroup expansion", func() {
	It("should expand a preset family into create and replace requests", func() {
		group := configurationGroupWithFamily("cg", coralogixv1alpha1.ConfigurationFamilySpec{Preset: presetFamily()})

		createReq, err := expandCreateRequest(group)
		Expect(err).NotTo(HaveOccurred())
		createFamily := createReq.Group.Family
		Expect(createFamily.Raw).To(BeNil())
		Expect(createFamily.Preset).NotTo(BeNil())
		Expect(createFamily.Preset.ChartName).To(Equal(cfggroups.CHARTNAME_CHART_NAME_OTEL_INTEGRATION))
		Expect(createFamily.Preset.ChartVersion).To(Equal("0.0.289"))
		Expect(createFamily.Preset.GetIntegrationVersion()).To(Equal("0.8.0"))
		Expect(createFamily.Preset.ObservabilityFeatures).To(MatchJSON(`{"logsCollection":{"enabled":true}}`))

		replaceReq, err := expandReplaceRequest(group)
		Expect(err).NotTo(HaveOccurred())
		replaceFamily := replaceReq.Group.Family
		Expect(replaceFamily.Raw).To(BeNil())
		Expect(replaceFamily.Preset).NotTo(BeNil())
		Expect(replaceFamily.Preset.ChartName).To(Equal(cfggroups.CHARTNAME_CHART_NAME_OTEL_INTEGRATION))
		Expect(replaceFamily.Preset.Metadata).To(Equal(map[string]string{"ClusterName": "prod"}))
	})

	It("should expand a raw family into create and replace requests", func() {
		group := configurationGroupWithFamily("cg", coralogixv1alpha1.ConfigurationFamilySpec{Raw: rawFamily()})

		createReq, err := expandCreateRequest(group)
		Expect(err).NotTo(HaveOccurred())
		Expect(createReq.Group.Family.Preset).To(BeNil())
		Expect(createReq.Group.Family.Raw.GetCollectorVersion()).To(Equal("0.114.0"))
		Expect(createReq.Group.Family.Raw.RemoteConfigurations).To(HaveLen(1))

		replaceReq, err := expandReplaceRequest(group)
		Expect(err).NotTo(HaveOccurred())
		Expect(replaceReq.Group.Family.Preset).To(BeNil())
		Expect(replaceReq.Group.Family.Raw.Metadata).To(Equal(map[string]string{}))
		Expect(replaceReq.Group.Family.Raw.RemoteConfigurations).To(HaveLen(1))
	})

	It("should reject observabilityFeatures that are not a JSON object", func() {
		preset := presetFamily()
		preset.ObservabilityFeatures = runtime.RawExtension{Raw: []byte(`["not","an","object"]`)}
		group := configurationGroupWithFamily("cg", coralogixv1alpha1.ConfigurationFamilySpec{Preset: preset})
		_, err := expandCreateRequest(group)
		Expect(err).To(MatchError(ContainSubstring("must be a JSON object")))
	})
})
