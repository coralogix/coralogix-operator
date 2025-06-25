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
)

var metricsLog = logf.Log.WithName("metrics")

const (
	remoteSynced   = "RemoteSynced"
	remoteUnsynced = "RemoteUnsynced"
)

func RegisterMetrics() error {
	metricsLog.V(1).Info("Registering metrics")
	for _, metric := range metricsList {
		err := metrics.Registry.Register(metric)
		if err != nil {
			return err
		}
	}

	return nil
}

var metricsList = []prometheus.Collector{
	operatorInfoMetric,
	resourceInfoMetric,
}

var (
	operatorInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_build_info",
			Help: "Coralogix Operator build information.",
		},
		[]string{"go_version", "operator_version", "coralogix_url"},
	)
	resourceInfoMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cx_operator_resource_info",
			Help: "Coralogix Operator custom resource information.",
		},
		[]string{"kind", "name", "namespace", "status"},
	)
)

func SetOperatorInfoMetric(goVersion, operatorVersion, url string) {
	metricsLog.V(1).Info("Setting operator info metric",
		"go_version", goVersion,
		"operator_version", operatorVersion,
		"coralogix_url", url,
	)
	operatorInfoMetric.WithLabelValues(goVersion, operatorVersion, url).Set(1)
}

func SetResourceInfoMetricSynced(kind, name, namespace string) {
	metricsLog.V(1).Info("Setting resource info metric synced",
		"kind", kind,
		"name", name,
		"namespace", namespace,
	)
	resourceInfoMetric.WithLabelValues(kind, name, namespace, remoteSynced).Set(1)
	resourceInfoMetric.DeleteLabelValues(kind, name, namespace, remoteUnsynced)
}

func SetResourceInfoMetricUnsynced(kind, name, namespace string) {
	metricsLog.V(1).Info("Setting resource info metric unsynced",
		"kind", kind,
		"name", name,
		"namespace", namespace,
	)
	resourceInfoMetric.WithLabelValues(kind, name, namespace, remoteUnsynced).Set(1)
	resourceInfoMetric.DeleteLabelValues(kind, name, namespace, remoteSynced)
}

func DeleteResourceInfoMetric(kind, name, namespace string) {
	metricsLog.V(1).Info("Deleting resource info metric",
		"kind", kind,
		"name", name,
		"namespace", namespace,
	)
	resourceInfoMetric.DeleteLabelValues(kind, name, namespace, remoteSynced)
	resourceInfoMetric.DeleteLabelValues(kind, name, namespace, remoteUnsynced)
}
