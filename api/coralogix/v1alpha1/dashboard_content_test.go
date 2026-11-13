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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

const dashboardSamplesDir = "../../../config/samples/v1alpha1/dashboards"

// sampleDocuments returns the YAML documents of a committed sample, so that these tests
// exercise exactly what ships in config/samples rather than a copy that can drift.
func sampleDocuments(t *testing.T, name string) []string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dashboardSamplesDir, name))
	require.NoError(t, err)
	return strings.Split(string(content), "\n---\n")
}

func loadDashboardSample(t *testing.T, name string) *Dashboard {
	t.Helper()
	dashboard := new(Dashboard)
	require.NoError(t, yaml.Unmarshal([]byte(sampleDocuments(t, name)[0]), dashboard))
	return dashboard
}

// useFakeClient points the package-level client at a fake for the duration of a test,
// which is what DecodeContent reads for a configMapRef.
func useFakeClient(t *testing.T, objects ...client.Object) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, AddToScheme(scheme))

	previous := config.CrClient
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build())
	t.Cleanup(func() { config.InitClient(previous) })
}

func extractFromJSON(ctx context.Context, json string) (*dashboards.Dashboard, error) {
	spec := &DashboardSpec{Json: ptr.To(json)}
	return spec.DecodeContent(ctx, "default")
}

// dashboardJSONWith wraps a widget definition in the minimal required envelope.
func dashboardJSONWith(definition string) string {
	return fmt.Sprintf(`{"name": "test", "layout": {"sections": [{"rows": [{"widgets": [{"definition": %s}]}]}]}}`, definition)
}

func TestDecodeContentParsesJSONSample(t *testing.T) {
	sample := loadDashboardSample(t, "dashboard-json.yaml")

	dashboard, err := sample.Spec.DecodeContent(context.Background(), "default")
	require.NoError(t, err)

	require.Equal(t, "OpenTelemetry Collector Dashboard", dashboard.Name)
	// The sample ships an unknown key on purpose; it must be tolerated at parse time and
	// stripped before the value is returned, since the API rejects unknown keys with a 400.
	require.NotContains(t, dashboard.AdditionalProperties, "unknownKey")

	lineChart := dashboard.Layout.Sections[0].Rows[0].Widgets[0].Definition.LineChart
	require.NotNil(t, lineChart)
	require.NotEmpty(t, lineChart.QueryDefinitions)
	require.Equal(t, "20", *lineChart.QueryDefinitions[0].SeriesCountLimit)
}

// Regression guard for the silent data loss this migration fixes: arcDisplay, showMinMax
// and layoutColumns do not exist in the proto the gRPC client was compiled against, so
// protojson's DiscardUnknown deleted them before the request was ever built. They must
// now survive all the way into the typed model.
func TestDecodeContentPreservesFieldsTheOldProtoLacked(t *testing.T) {
	sample := loadDashboardSample(t, "dashboard-import.yaml")

	dashboard, err := sample.Spec.DecodeContent(context.Background(), "default")
	require.NoError(t, err)

	widget := dashboard.Layout.Sections[0].Rows[0].Widgets[0]
	require.Equal(t, int32(12), *widget.LayoutColumns)

	gauge := widget.Definition.Gauge
	require.NotNil(t, gauge.ArcDisplay)
	require.True(t, *gauge.ArcDisplay.ValueArc)
	require.True(t, *gauge.ArcDisplay.ThresholdArc)
	require.NotNil(t, gauge.ShowMinMax)
	require.False(t, *gauge.ShowMinMax)
}

func TestDecodeContentReadsGzipJSON(t *testing.T) {
	sample := loadDashboardSample(t, "dashboard-gzip.yaml")

	dashboard, err := sample.Spec.DecodeContent(context.Background(), "default")
	require.NoError(t, err)

	require.NotEmpty(t, dashboard.Name)
}

func TestDecodeContentReadsConfigMapRef(t *testing.T) {
	documents := sampleDocuments(t, "dashboard-configmap.yaml")
	require.Len(t, documents, 2, "sample is expected to hold a Dashboard and its ConfigMap")

	sample := new(Dashboard)
	require.NoError(t, yaml.Unmarshal([]byte(documents[0]), sample))
	configMap := new(corev1.ConfigMap)
	require.NoError(t, yaml.Unmarshal([]byte(documents[1]), configMap))
	configMap.Namespace = "default"
	useFakeClient(t, configMap)

	dashboard, err := sample.Spec.DecodeContent(context.Background(), "default")
	require.NoError(t, err)

	require.Equal(t, "OpenTelemetry Collector Dashboard", dashboard.Name)
}

// protojson accepted the protobuf spellings and bare numbers for int64 fields, so CRs in
// the wild rely on both.
func TestDecodeContentAcceptsProtobufSpellingsAndBareNumbers(t *testing.T) {
	json := dashboardJSONWith(`{"line_chart": {"query_definitions": [{"id": "q1", "query": {}, "series_count_limit": 20}]}}`)

	dashboard, err := extractFromJSON(context.Background(), json)
	require.NoError(t, err)

	queryDefinition := dashboard.Layout.Sections[0].Rows[0].Widgets[0].Definition.LineChart.QueryDefinitions[0]
	require.Equal(t, "20", *queryDefinition.SeriesCountLimit)
}

// Deliberate deviation from protojson, which discarded these silently: a dropped enum
// renders a dashboard that differs from what the author exported.
func TestDecodeContentRejectsOutOfSpecEnums(t *testing.T) {
	t.Run("unknown value", func(t *testing.T) {
		_, err := extractFromJSON(context.Background(), dashboardJSONWith(`{"gauge": {"thresholdBy": "THRESHOLD_BY_FUTURE"}}`))

		require.ErrorContains(t, err, "failed to decode the JSON content")
		require.ErrorContains(t, err, "layout.sections[0].rows[0].widgets[0].definition.gauge.thresholdBy")
		require.ErrorContains(t, err, `"THRESHOLD_BY_FUTURE" is not a valid GaugeThresholdBy`)
	})

	t.Run("numeric value", func(t *testing.T) {
		_, err := extractFromJSON(context.Background(), dashboardJSONWith(`{"gauge": {"thresholdBy": 1}}`))

		require.ErrorContains(t, err, "definition.gauge.thresholdBy")
		require.ErrorContains(t, err, "numeric enum values are not supported")
	})
}

func TestDecodeContentRequiresAContentSource(t *testing.T) {
	_, err := (&DashboardSpec{}).DecodeContent(context.Background(), "default")

	require.ErrorContains(t, err, "json, gzipJson, or configMapRef is required")
}
