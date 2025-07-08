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
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	clientmetrics "k8s.io/client-go/tools/metrics"
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
	clientmetrics.RequestResult = &ResultAdapter{requestsTotalMetric}
	clientmetrics.RequestLatency = &LatencyAdapter{requestsLatencyMetric}
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
	requestsTotalMetric,
	requestsLatencyMetric,
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
	requestsTotalMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cx_operator_client_requests_total",
			Help: "Total number of Coralogix Operator's in-cluster requests by status code and verb.",
		},
		[]string{"code", "verb"},
	)
	requestsLatencyMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cx_operator_client_requests_latency_seconds",
			Help:    "Histogram of latencies for the Coralogix Operator's in-cluster requests by verb.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.4, 0.6, 0.8, 1.0, 2.0},
		},
		[]string{"verb"},
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

var _ clientmetrics.ResultMetric = &ResultAdapter{}

type ResultAdapter struct {
	metric *prometheus.CounterVec
}

func (r *ResultAdapter) Increment(_ context.Context, code, verb, _ string) {
	r.metric.WithLabelValues(code, verb).Inc()
}

var _ clientmetrics.LatencyMetric = &LatencyAdapter{}

type LatencyAdapter struct {
	metric *prometheus.HistogramVec
}

func (l *LatencyAdapter) Observe(_ context.Context, verb string, _ url.URL, latency time.Duration) {
	l.metric.WithLabelValues(verb).Observe(latency.Seconds())
}
