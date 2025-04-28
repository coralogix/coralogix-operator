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

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/coralogix/coralogix-operator/internal/config"
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
		gvks: utils.GetGVKs(config.GetScheme()),
	}
}

func (c *ResourceInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	metricsLog.V(1).Info("Describing metrics for ResourceInfoCollector")
	ch <- c.resourceInfoMetric
}

func (c *ResourceInfoCollector) Collect(ch chan<- prometheus.Metric) {
	metricsLog.V(1).Info("Collecting metrics for ResourceInfoCollector")
	for _, gvk := range c.gvks {
		resources, err := listResourcesInGVK(gvk)
		if err != nil {
			metricsLog.Error(err, "Failed to list resources in GVK", "gvk", gvk)
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
					getResourceStatus(resource),
				}...,
			)

			if err != nil {
				metricsLog.Error(err, "Failed to create metric for custom resource",
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

func listResourcesInGVK(gvk schema.GroupVersionKind) ([]unstructured.Unstructured, error) {
	var result []unstructured.Unstructured

	labelSelector := config.GetConfig().Selector.LabelSelector
	if gvk.Kind == utils.PrometheusRuleKind {
		req, err := labels.NewRequirement(utils.TrackPrometheusRuleAlertsLabelKey, selection.Equals, []string{"true"})
		if err != nil {
			return nil, err
		}
		labelSelector = labelSelector.Add(*req)
	}

	namespaces, err := filterNamespaces(config.GetConfig().Selector.NamespaceSelector)
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces.Items {
		resources := &unstructured.UnstructuredList{}
		resources.SetGroupVersionKind(gvk)
		if err = config.GetClient().List(context.Background(), resources,
			&client.ListOptions{LabelSelector: labelSelector, Namespace: ns.Name}); err != nil {
			return nil, err
		}
		result = append(result, resources.Items...)
	}

	return result, nil
}

func filterNamespaces(selector labels.Selector) (*corev1.NamespaceList, error) {
	namespacesList := &corev1.NamespaceList{}
	listOptions := &client.ListOptions{LabelSelector: selector}
	if err := config.GetClient().List(context.Background(), namespacesList, listOptions); err != nil {
		return nil, err
	}
	return namespacesList, nil
}

func getResourceStatus(resource unstructured.Unstructured) string {
	if resource.GetKind() == utils.PrometheusRuleKind {
		return ""
	}

	const unknownStatus = "unknown"
	conditions, found, err := unstructured.NestedSlice(resource.Object, "status", "conditions")
	if err != nil {
		metricsLog.Error(err, "Error extracting conditions from resource", "kind", resource.GetKind(), "name", resource.GetName(), "namespace", resource.GetNamespace())
		return unknownStatus
	}
	if !found {
		metricsLog.V(1).Info("No conditions found for resource", "kind", resource.GetKind(), "name", resource.GetName(), "namespace", resource.GetNamespace())
		return unknownStatus
	}

	for _, cond := range conditions {
		if conditionMap, ok := cond.(map[string]interface{}); ok {
			if conditionType, exists := conditionMap["type"].(string); exists &&
				conditionType == utils.ConditionTypeRemoteSynced {
				if reason, exists := conditionMap["reason"].(string); exists {
					return reason
				}
			}
		}
	}

	return unknownStatus
}
