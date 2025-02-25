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
	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strings"

	"github.com/coralogix/coralogix-operator/internal/utils"
)

var metricsLog = logf.Log.WithName("metrics")

func SetupMetrics() error {
	metricsLog.V(1).Info("Setting up metrics")
	if err := RegisterMetrics(); err != nil {
		return err
	}

	if err := RegisterCollectors(); err != nil {
		return err
	}

	return nil
}

func RegisterMetrics() error {
	metricsLog.V(1).Info("Registering metrics")
	for _, metric := range metricsList {
		err := metrics.Registry.Register(metric)
		if err != nil {
			metricsLog.Error(err, "Failed to register metric", "metric", metric)
			return err
		}
	}

	return nil
}

var metricsList = []prometheus.Collector{
	operatorInfoMetric,
	resourceRejectionsTotalMetric,
}

var operatorInfoMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "cx_operator_info",
		Help: "Coralogix Operator information.",
	},
	[]string{"go_version", "operator_version", "coralogix_url", "label_selector", "namespace_selector"},
)

func SetOperatorInfoMetric(goVersion, operatorVersion, url string) {
	labelSelector := utils.GetSelector().LabelSelector.String()
	namespaceSelector := strings.Join(utils.GetSelector().NamespaceSelector, ",")
	metricsLog.V(1).Info("Setting operator info metric",
		"go_version", goVersion,
		"operator_version", operatorVersion,
		"coralogix_url", url,
		"label_selector", labelSelector,
		"namespace_selector", namespaceSelector,
	)
	operatorInfoMetric.WithLabelValues(goVersion, operatorVersion, url, labelSelector, namespaceSelector).Set(1)
}

var resourceRejectionsTotalMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "cx_operator_resource_rejections_total",
		Help: "The total count of rejections by Coralogix Operator validation webhook.",
	},
	[]string{"kind", "name", "namespace"},
)

func IncResourceRejectionsTotalMetric(kind, name, namespace string) {
	metricsLog.V(1).Info("Incrementing resource total rejected metric", "kind", kind)
	resourceRejectionsTotalMetric.WithLabelValues(kind, name, namespace).Inc()
}
