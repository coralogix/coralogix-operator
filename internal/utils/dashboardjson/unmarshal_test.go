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

package dashboardjson

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"
)

// dashboardWith wraps a widget definition in the minimal envelope the model requires
// (name and layout are declared required in the API's OpenAPI schema).
func dashboardWith(definition string) string {
	return fmt.Sprintf(`{
  "name": "test dashboard",
  "layout": {
    "sections": [
      {
        "rows": [
          {
            "widgets": [
              {
                "definition": %s
              }
            ]
          }
        ]
      }
    ]
  }
}`, definition)
}

func firstWidget(t *testing.T, dashboard *dashboards.Dashboard) dashboards.Widget {
	t.Helper()
	require.Len(t, dashboard.Layout.Sections, 1)
	require.Len(t, dashboard.Layout.Sections[0].Rows, 1)
	require.Len(t, dashboard.Layout.Sections[0].Rows[0].Widgets, 1)
	return dashboard.Layout.Sections[0].Rows[0].Widgets[0]
}

func TestUnmarshalAcceptsBothFieldNameSpellings(t *testing.T) {
	for name, definition := range map[string]string{
		"camelCase":  `{"dataTable": {"resultsPerPage": 20, "rowStyle": "ROW_STYLE_ONE_LINE"}}`,
		"snake_case": `{"data_table": {"results_per_page": 20, "row_style": "ROW_STYLE_ONE_LINE"}}`,
		"mixed":      `{"dataTable": {"results_per_page": 20, "rowStyle": "ROW_STYLE_ONE_LINE"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			dashboard := new(dashboards.Dashboard)
			require.NoError(t, Unmarshal([]byte(dashboardWith(definition)), dashboard))

			dataTable := firstWidget(t, dashboard).Definition.DataTable
			require.NotNil(t, dataTable)
			require.Equal(t, int32(20), *dataTable.ResultsPerPage)
			require.Equal(t, dashboards.ROWSTYLE_ROW_STYLE_ONE_LINE, *dataTable.RowStyle)
			require.Empty(t, dataTable.AdditionalProperties)
		})
	}
}

func TestUnmarshalAcceptsMixedCaseWithinOneObject(t *testing.T) {
	json := dashboardWith(`{"lineChart": {"query_definitions": [{"id": "q1", "query": {}}]}}`)

	dashboard := new(dashboards.Dashboard)
	require.NoError(t, Unmarshal([]byte(json), dashboard))

	lineChart := firstWidget(t, dashboard).Definition.LineChart
	require.NotNil(t, lineChart)
	require.Len(t, lineChart.QueryDefinitions, 1)
	require.Equal(t, "q1", lineChart.QueryDefinitions[0].Id)
}

func TestUnmarshalPrefersProtobufSpellingWhenBothArePresent(t *testing.T) {
	json := `{
  "name": "test dashboard",
  "layout": {"sections": []},
  "relativeTimeFrame": "900s",
  "relative_time_frame": "1800s"
}`

	dashboard := new(dashboards.Dashboard)
	require.NoError(t, Unmarshal([]byte(json), dashboard))

	require.Equal(t, "1800s", *dashboard.RelativeTimeFrame)
}

func TestUnmarshalKeepsUnknownKeysInAdditionalProperties(t *testing.T) {
	json := `{
  "name": "test dashboard",
  "layout": {"sections": []},
  "unknownTopLevelKey": "should be ignored",
  "variables": [{"unknownNestedKey": true}]
}`

	dashboard := new(dashboards.Dashboard)
	require.NoError(t, Unmarshal([]byte(json), dashboard))

	require.Equal(t, "should be ignored", dashboard.AdditionalProperties["unknownTopLevelKey"])
	require.Len(t, dashboard.Variables, 1)
	require.Equal(t, true, dashboard.Variables[0].AdditionalProperties["unknownNestedKey"])
}

func TestUnmarshalAcceptsInt64EitherAsNumberOrString(t *testing.T) {
	for name, seriesCountLimit := range map[string]string{
		"bare number": `20`,
		"string":      `"20"`,
	} {
		t.Run(name, func(t *testing.T) {
			definition := fmt.Sprintf(
				`{"lineChart": {"queryDefinitions": [{"id": "q1", "query": {}, "seriesCountLimit": %s}]}}`,
				seriesCountLimit)

			dashboard := new(dashboards.Dashboard)
			require.NoError(t, Unmarshal([]byte(dashboardWith(definition)), dashboard))

			queryDefinition := firstWidget(t, dashboard).Definition.LineChart.QueryDefinitions[0]
			require.Equal(t, "20", *queryDefinition.SeriesCountLimit)
		})
	}
}

func TestUnmarshalPreservesValidEnumValues(t *testing.T) {
	definition := `{"gauge": {"thresholdBy": "THRESHOLD_BY_VALUE", "legend": {"columns": ["LEGEND_COLUMN_NAME", "LEGEND_COLUMN_MAX"]}}}`

	dashboard := new(dashboards.Dashboard)
	require.NoError(t, Unmarshal([]byte(dashboardWith(definition)), dashboard))

	gauge := firstWidget(t, dashboard).Definition.Gauge
	require.Equal(t, dashboards.GAUGETHRESHOLDBY_THRESHOLD_BY_VALUE, *gauge.ThresholdBy)
	require.Equal(t, []dashboards.LegendColumn{
		dashboards.LEGENDCOLUMN_LEGEND_COLUMN_NAME,
		dashboards.LEGENDCOLUMN_LEGEND_COLUMN_MAX,
	}, gauge.Legend.Columns)
}

// An enum value the pinned SDK does not know is rejected rather than silently dropped
// the way protojson dropped it, and the error has to say where in the document it is.
func TestUnmarshalRejectsUnknownEnumValueNamingTheJSONPath(t *testing.T) {
	definition := `{"gauge": {"thresholdBy": "THRESHOLD_BY_FUTURE"}}`

	err := Unmarshal([]byte(dashboardWith(definition)), new(dashboards.Dashboard))

	require.ErrorContains(t, err, "layout.sections[0].rows[0].widgets[0].definition.gauge.thresholdBy")
	require.ErrorContains(t, err, `"THRESHOLD_BY_FUTURE" is not a valid GaugeThresholdBy`)
}

func TestUnmarshalRejectsUnknownEnumValueInRepeatedFieldWithElementIndex(t *testing.T) {
	definition := `{"gauge": {"legend": {"columns": ["LEGEND_COLUMN_NAME", "LEGEND_COLUMN_FUTURE"]}}}`

	err := Unmarshal([]byte(dashboardWith(definition)), new(dashboards.Dashboard))

	require.ErrorContains(t, err, "definition.gauge.legend.columns[1]")
	require.ErrorContains(t, err, `"LEGEND_COLUMN_FUTURE" is not a valid LegendColumn`)
}

// Numeric enums must be reported distinctly: the int64 coercion would otherwise turn 1
// into "1" and report it as an unrecognized enum value, which is misleading.
func TestUnmarshalRejectsNumericEnumWithADistinctMessage(t *testing.T) {
	definition := `{"gauge": {"thresholdBy": 1}}`

	err := Unmarshal([]byte(dashboardWith(definition)), new(dashboards.Dashboard))

	require.ErrorContains(t, err, "definition.gauge.thresholdBy")
	require.ErrorContains(t, err, "numeric enum values are not supported; use the string form of GaugeThresholdBy")
}

func TestUnmarshalRejectsTrailingContentAfterTopLevelValue(t *testing.T) {
	err := Unmarshal([]byte(`{"name": "a", "layout": {}} {"name": "b"}`), new(dashboards.Dashboard))

	require.ErrorContains(t, err, "invalid JSON after top-level value")
}

func TestUnmarshalRejectsMissingRequiredProperties(t *testing.T) {
	err := Unmarshal([]byte(`{"layout": {"sections": []}}`), new(dashboards.Dashboard))

	require.ErrorContains(t, err, "no value given for required property name")
}

func TestUnmarshalRejectsTwoOneOfArmsSet(t *testing.T) {
	definition := `{"dataTable": {"query": {"logs": {}, "spans": {}}}}`

	err := Unmarshal([]byte(dashboardWith(definition)), new(dashboards.Dashboard))

	require.ErrorContains(t, err, "may be set")
}

func TestUnmarshalRejectsNonPointerTarget(t *testing.T) {
	require.ErrorContains(t, Unmarshal([]byte(`{}`), dashboards.Dashboard{}), "non-nil pointer")
	require.ErrorContains(t, Unmarshal([]byte(`{}`), (*dashboards.Dashboard)(nil)), "non-nil pointer")
}

// DiscardAdditionalProperties runs after Unmarshal, so the typed fields promoted from
// the protobuf spelling survive while the unknown keys that the API would reject do not.
func TestDiscardAdditionalPropertiesAfterUnmarshal(t *testing.T) {
	json := `{
  "name": "test dashboard",
  "unknownTopLevelKey": "should be ignored",
  "layout": {
    "sections": [
      {
        "rows": [
          {
            "widgets": [
              {
                "definition": {
                  "data_table": {
                    "results_per_page": 20,
                    "unknownNestedKey": "should be ignored"
                  }
                }
              }
            ]
          }
        ]
      }
    ]
  }
}`

	dashboard := new(dashboards.Dashboard)
	require.NoError(t, Unmarshal([]byte(json), dashboard))
	require.NotEmpty(t, dashboard.AdditionalProperties)

	DiscardAdditionalProperties(dashboard)

	require.Empty(t, dashboard.AdditionalProperties)
	dataTable := firstWidget(t, dashboard).Definition.DataTable
	require.NotNil(t, dataTable)
	require.Equal(t, int32(20), *dataTable.ResultsPerPage)
	require.Empty(t, dataTable.AdditionalProperties)
}

// A message with no fields is modeled as a free-form map, not a struct, so unknown keys
// authored inside one have no AdditionalProperties to land in and have to be deleted from
// the map itself. The emptied arm still has to be sent - it is what selects the oneOf case.
func TestDiscardAdditionalPropertiesEmptiesUnknownKeysInFieldlessMessages(t *testing.T) {
	json := `{
  "name": "test dashboard",
  "layout": {"sections": []},
  "off": {"unknownKey": "should be ignored"},
  "variables": [
    {
      "name": "v",
      "definition": {"multiSelect": {"selection": {"all": {"unknownNestedKey": "should be ignored"}}}}
    }
  ]
}`

	dashboard := new(dashboards.Dashboard)
	require.NoError(t, Unmarshal([]byte(json), dashboard))
	require.Equal(t, "should be ignored", dashboard.Off["unknownKey"])

	DiscardAdditionalProperties(dashboard)

	require.NotNil(t, dashboard.Off)
	require.Empty(t, dashboard.Off)
	require.Len(t, dashboard.Variables, 1)
	allSelection := dashboard.Variables[0].Definition.MultiSelect.Selection.All
	require.NotNil(t, allSelection)
	require.Empty(t, allSelection)

	// The generated marshaler omits a nil map, so emptying the arm must not nil it out:
	// that would drop the whole arm from the request rather than just its unknown keys.
	serialized, err := dashboard.ToMap()
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{}, serialized["off"])
}
