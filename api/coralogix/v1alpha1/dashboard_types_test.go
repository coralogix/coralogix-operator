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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
// which is what ExtractJsonContentFromSpec and folder resourceRef resolution read.
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
	return spec.ExtractDashboardFromSpec(ctx, "default")
}

// dashboardJSONWith wraps a widget definition in the minimal required envelope.
func dashboardJSONWith(definition string) string {
	return fmt.Sprintf(`{"name": "test", "layout": {"sections": [{"rows": [{"widgets": [{"definition": %s}]}]}]}}`, definition)
}

func TestExtractDashboardFromSpecParsesJSONSample(t *testing.T) {
	sample := loadDashboardSample(t, "dashboard-json.yaml")

	dashboard, err := sample.Spec.ExtractDashboardFromSpec(context.Background(), "default")
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
func TestExtractDashboardFromSpecPreservesFieldsTheOldProtoLacked(t *testing.T) {
	sample := loadDashboardSample(t, "dashboard-import.yaml")

	dashboard, err := sample.Spec.ExtractDashboardFromSpec(context.Background(), "default")
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

func TestExtractDashboardFromSpecReadsGzipJSONAndResolvesFolderResourceRef(t *testing.T) {
	sample := loadDashboardSample(t, "dashboard-gzip.yaml")
	require.NotNil(t, sample.Spec.FolderRef.ResourceRef, "sample is expected to reference a folder by resourceRef")

	folderID := "3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f"
	useFakeClient(t, &DashboardsFolder{
		ObjectMeta: metav1.ObjectMeta{Name: sample.Spec.FolderRef.ResourceRef.Name, Namespace: "default"},
		Status:     DashboardsFolderStatus{ID: ptr.To(folderID)},
	})

	dashboard, err := sample.Spec.ExtractDashboardFromSpec(context.Background(), "default")
	require.NoError(t, err)

	require.NotEmpty(t, dashboard.Name)
	require.Equal(t, folderID, *dashboard.FolderId.Value)
}

func TestExtractDashboardFromSpecReadsConfigMapRef(t *testing.T) {
	documents := sampleDocuments(t, "dashboard-configmap.yaml")
	require.Len(t, documents, 2, "sample is expected to hold a Dashboard and its ConfigMap")

	sample := new(Dashboard)
	require.NoError(t, yaml.Unmarshal([]byte(documents[0]), sample))
	configMap := new(corev1.ConfigMap)
	require.NoError(t, yaml.Unmarshal([]byte(documents[1]), configMap))
	configMap.Namespace = "default"
	useFakeClient(t, configMap)

	dashboard, err := sample.Spec.ExtractDashboardFromSpec(context.Background(), "default")
	require.NoError(t, err)

	require.Equal(t, "OpenTelemetry Collector Dashboard", dashboard.Name)
}

func TestExtractDashboardFromSpecExpandsFolderBackendRef(t *testing.T) {
	json := `{"name": "test", "layout": {"sections": []}}`
	folderID := "3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f"

	t.Run("id", func(t *testing.T) {
		spec := &DashboardSpec{
			Json:      ptr.To(json),
			FolderRef: &DashboardFolderRef{BackendRef: &DashboardFolderRefBackendRef{ID: ptr.To(folderID)}},
		}

		dashboard, err := spec.ExtractDashboardFromSpec(context.Background(), "default")
		require.NoError(t, err)
		require.Equal(t, folderID, *dashboard.FolderId.Value)
		require.Nil(t, dashboard.FolderPath)
	})

	t.Run("multi-segment path", func(t *testing.T) {
		spec := &DashboardSpec{
			Json:      ptr.To(json),
			FolderRef: &DashboardFolderRef{BackendRef: &DashboardFolderRefBackendRef{Path: ptr.To("team/infra/observability")}},
		}

		dashboard, err := spec.ExtractDashboardFromSpec(context.Background(), "default")
		require.NoError(t, err)
		require.Equal(t, []string{"team", "infra", "observability"}, dashboard.FolderPath.Segments)
		require.Nil(t, dashboard.FolderId)
	})

	t.Run("missing folder resourceRef", func(t *testing.T) {
		useFakeClient(t)
		spec := &DashboardSpec{
			Json:      ptr.To(json),
			FolderRef: &DashboardFolderRef{ResourceRef: &ResourceRef{Name: "missing-folder"}},
		}

		_, err := spec.ExtractDashboardFromSpec(context.Background(), "default")
		require.ErrorContains(t, err, "failed to get DashboardsFolder")
	})

	t.Run("neither backendRef nor resourceRef", func(t *testing.T) {
		spec := &DashboardSpec{Json: ptr.To(json), FolderRef: &DashboardFolderRef{}}

		_, err := spec.ExtractDashboardFromSpec(context.Background(), "default")
		require.ErrorContains(t, err, "folderRef.BackendRef or folderRef.ResourceRef is required")
	})
}

// folder_id and folder_path are not a protobuf oneof, so a request naming both is not
// rejected by the schema - spec.folderRef has to replace the folder the spec content
// declared rather than being sent alongside it.
func TestExtractDashboardFromSpecFolderRefReplacesFolderInContent(t *testing.T) {
	folderID := "3d7f1c2a-9e4b-4a11-8f2d-1a2b3c4d5e6f"

	t.Run("id replaces a path in content", func(t *testing.T) {
		spec := &DashboardSpec{
			Json:      ptr.To(`{"name": "test", "layout": {}, "folderPath": {"segments": ["from", "content"]}}`),
			FolderRef: &DashboardFolderRef{BackendRef: &DashboardFolderRefBackendRef{ID: ptr.To(folderID)}},
		}

		dashboard, err := spec.ExtractDashboardFromSpec(context.Background(), "default")
		require.NoError(t, err)
		require.Equal(t, folderID, *dashboard.FolderId.Value)
		require.Nil(t, dashboard.FolderPath)
	})

	t.Run("path replaces an id in content", func(t *testing.T) {
		spec := &DashboardSpec{
			Json:      ptr.To(`{"name": "test", "layout": {}, "folderId": {"value": "` + folderID + `"}}`),
			FolderRef: &DashboardFolderRef{BackendRef: &DashboardFolderRefBackendRef{Path: ptr.To("team/infra")}},
		}

		dashboard, err := spec.ExtractDashboardFromSpec(context.Background(), "default")
		require.NoError(t, err)
		require.Equal(t, []string{"team", "infra"}, dashboard.FolderPath.Segments)
		require.Nil(t, dashboard.FolderId)
	})

	// Without a folderRef the content is authoritative and is passed through untouched.
	t.Run("content folder is kept when no folderRef is declared", func(t *testing.T) {
		spec := &DashboardSpec{
			Json: ptr.To(`{"name": "test", "layout": {}, "folderPath": {"segments": ["from", "content"]}}`),
		}

		dashboard, err := spec.ExtractDashboardFromSpec(context.Background(), "default")
		require.NoError(t, err)
		require.Equal(t, []string{"from", "content"}, dashboard.FolderPath.Segments)
		require.Nil(t, dashboard.FolderId)
	})
}

// protojson accepted the protobuf spellings and bare numbers for int64 fields, so CRs in
// the wild rely on both.
func TestExtractDashboardFromSpecAcceptsProtobufSpellingsAndBareNumbers(t *testing.T) {
	json := dashboardJSONWith(`{"line_chart": {"query_definitions": [{"id": "q1", "query": {}, "series_count_limit": 20}]}}`)

	dashboard, err := extractFromJSON(context.Background(), json)
	require.NoError(t, err)

	queryDefinition := dashboard.Layout.Sections[0].Rows[0].Widgets[0].Definition.LineChart.QueryDefinitions[0]
	require.Equal(t, "20", *queryDefinition.SeriesCountLimit)
}

// Deliberate deviation from protojson, which discarded these silently: a dropped enum
// renders a dashboard that differs from what the author exported.
func TestExtractDashboardFromSpecRejectsOutOfSpecEnums(t *testing.T) {
	t.Run("unknown value", func(t *testing.T) {
		_, err := extractFromJSON(context.Background(), dashboardJSONWith(`{"gauge": {"thresholdBy": "THRESHOLD_BY_FUTURE"}}`))

		require.ErrorContains(t, err, "failed to unmarshal contentJson")
		require.ErrorContains(t, err, "layout.sections[0].rows[0].widgets[0].definition.gauge.thresholdBy")
		require.ErrorContains(t, err, `"THRESHOLD_BY_FUTURE" is not a valid GaugeThresholdBy`)
	})

	t.Run("numeric value", func(t *testing.T) {
		_, err := extractFromJSON(context.Background(), dashboardJSONWith(`{"gauge": {"thresholdBy": 1}}`))

		require.ErrorContains(t, err, "definition.gauge.thresholdBy")
		require.ErrorContains(t, err, "numeric enum values are not supported")
	})
}

func TestExtractDashboardFromSpecRequiresAContentSource(t *testing.T) {
	_, err := (&DashboardSpec{}).ExtractDashboardFromSpec(context.Background(), "default")

	require.ErrorContains(t, err, "json, gzipContentJson or configMapRef is required")
}
