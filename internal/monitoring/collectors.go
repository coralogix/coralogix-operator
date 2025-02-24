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

package monitoring

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	coralogixreconciler "github.com/coralogix/coralogix-operator/internal/controller/coralogix/coralogix-reconciler"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

func RegisterCollectors() error {
	metricsLog.V(1).Info("Registering collectors")
	resourceInfoCollector := NewResourceInfoCollector()
	if err := metrics.Registry.Register(resourceInfoCollector); err != nil {
		metricsLog.Error(err, "Failed to register collector", "collector", resourceInfoCollector)
		return err
	}
	return nil
}

type ResourceInfoCollector struct {
	resourceInfoMetric *prometheus.Desc
	gvks               []schema.GroupVersionKind
}

func NewResourceInfoCollector() *ResourceInfoCollector {
	return &ResourceInfoCollector{
		resourceInfoMetric: prometheus.NewDesc(
			"cx_operator_resource_info",
			"Coralogix Operator custom resource information.",
			[]string{"kind", "name", "namespace", "status"},
			nil,
		),
		gvks: getGVKsToMonitor(),
	}
}

func (c *ResourceInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	log.Log.V(1).Info("Describing metrics for ResourceInfoCollector")
	ch <- c.resourceInfoMetric
}

func (c *ResourceInfoCollector) Collect(ch chan<- prometheus.Metric) {
	log.Log.V(1).Info("Collecting metrics for ResourceInfoCollector")
	for _, gvk := range c.gvks {
		resources, err := listResourcesInGVK(gvk)
		if err != nil {
			log.Log.Error(err, "Failed to list resources in GVK", "gvk", gvk)
			continue
		}

		for _, resource := range resources {
			metric, err := prometheus.NewConstMetric(
				c.resourceInfoMetric,
				prometheus.GaugeValue,
				1,
				[]string{
					gvk.Kind,
					resource.GetName(),
					resource.GetNamespace(),
					"", // status will be extracted from the resource conditions
				}...,
			)

			if err != nil {
				log.Log.Error(err, "Failed to create metric for custom resource",
					"kind", gvk.Kind,
					"name", resource.GetName(),
					"namespace", resource.GetNamespace(),
				)
				continue
			}

			ch <- metric
		}
	}
}

func getGVKsToMonitor() []schema.GroupVersionKind {
	result := []schema.GroupVersionKind{
		{Group: utils.MonitoringAPIGroup, Version: utils.V1APIVersion, Kind: utils.PrometheusRuleKind},
	}

	cxGroupVersion := schema.GroupVersion{Group: utils.CoralogixAPIGroup, Version: utils.V1alpha1APIVersion}
	knownTypes := coralogixreconciler.GetScheme().KnownTypes(cxGroupVersion)
	for kind := range knownTypes {
		// Skip List types. e.g. "AlertList".
		if strings.HasSuffix(kind, "List") {
			continue
		}
		result = append(result, cxGroupVersion.WithKind(kind))
	}

	return result
}

func listResourcesInGVK(gvk schema.GroupVersionKind) ([]unstructured.Unstructured, error) {
	resourceList := &unstructured.UnstructuredList{}
	resourceList.SetGroupVersionKind(gvk)
	labelSelector := utils.GetLabelFilter().Selector

	if gvk.Kind == utils.PrometheusRuleKind {
		req, err := labels.NewRequirement(utils.TrackPrometheusRuleAlertsLabelKey, selection.Equals, []string{"true"})
		if err != nil {
			return nil, err
		}
		labelSelector = labelSelector.Add(*req)
	}

	if err := coralogixreconciler.GetClient().List(context.Background(), resourceList,
		&client.ListOptions{LabelSelector: labelSelector}); err != nil {
		return nil, err
	}

	return resourceList.Items, nil
}
