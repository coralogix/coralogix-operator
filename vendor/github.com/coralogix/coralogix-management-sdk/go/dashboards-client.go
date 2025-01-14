// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cxsdk

import (
	"context"

	dashboards "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/dashboards/v1"
)

// DashboardsClient is a client for the Coralogix Dashboards API.
type DashboardsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// CreateDashboardRequest is a request to create a dashboard.
type CreateDashboardRequest = dashboards.CreateDashboardRequest

// ReplaceDashboardRequest is a request to replace a dashboard.
type ReplaceDashboardRequest = dashboards.ReplaceDashboardRequest

// DeleteDashboardRequest is a request to delete a dashboard.
type DeleteDashboardRequest = dashboards.DeleteDashboardRequest

// GetDashboardRequest is a request to get a dashboard.
type GetDashboardRequest = dashboards.GetDashboardRequest

// PinDashboardRequest is a request to pin a dashboard.
type PinDashboardRequest = dashboards.PinDashboardRequest

// UnpinDashboardRequest is a request to unpin a dashboard.
type UnpinDashboardRequest = dashboards.UnpinDashboardRequest

// DashboardFolderPath is a dashboard folder path.
type DashboardFolderPath = dashboards.Dashboard_FolderPath

// DashboardFolderID is a dashboard folder id.
type DashboardFolderID = dashboards.Dashboard_FolderId

// FolderPath is a dashboard folder path.
type FolderPath = dashboards.FolderPath

// Dashboard is a dashboard.
type Dashboard = dashboards.Dashboard

// RowStyle is a style for a row.
type RowStyle = dashboards.RowStyle

// RowStyle values.
const (
	RowStyleOneLine     = dashboards.RowStyle_ROW_STYLE_ONE_LINE
	RowStyleTwoLine     = dashboards.RowStyle_ROW_STYLE_TWO_LINE
	RowStyleCondensed   = dashboards.RowStyle_ROW_STYLE_CONDENSED
	RowStyleJSON        = dashboards.RowStyle_ROW_STYLE_JSON
	RowStyleList        = dashboards.RowStyle_ROW_STYLE_LIST
	RowStyleUnspecified = dashboards.RowStyle_ROW_STYLE_UNSPECIFIED
)

// DashboardLegend is the type for the legends.
type DashboardLegend = dashboards.Legend

// DashboardLegendColumn is the column type for the legends.
type DashboardLegendColumn = dashboards.Legend_LegendColumn

// LegendColumn values.
const (
	LegendColumnUnspecified = dashboards.Legend_LEGEND_COLUMN_UNSPECIFIED
	LegendColumnMin         = dashboards.Legend_LEGEND_COLUMN_MIN
	LegendColumnMax         = dashboards.Legend_LEGEND_COLUMN_MAX
	LegendColumnSum         = dashboards.Legend_LEGEND_COLUMN_SUM
	LegendColumnAvg         = dashboards.Legend_LEGEND_COLUMN_AVG
	LegendColumnLast        = dashboards.Legend_LEGEND_COLUMN_LAST
	LegendColumnName        = dashboards.Legend_LEGEND_COLUMN_NAME
)

// LegendPlacement is a type for the legend placement.
type LegendPlacement = dashboards.Legend_LegendPlacement

// LegendPlacement values.
const (
	LegendPlacementUnspecified = dashboards.Legend_LEGEND_PLACEMENT_UNSPECIFIED
	LegendPlacementAuto        = dashboards.Legend_LEGEND_PLACEMENT_AUTO
	LegendPlacementBottom      = dashboards.Legend_LEGEND_PLACEMENT_BOTTOM
	LegendPlacementSide        = dashboards.Legend_LEGEND_PLACEMENT_SIDE
	LegendPlacementHidden      = dashboards.Legend_LEGEND_PLACEMENT_HIDDEN
)

// OrderDirection is the sorting direction
type OrderDirection = dashboards.OrderDirection

// OrderDirection values.
const (
	OrderDirectionAsc         = dashboards.OrderDirection_ORDER_DIRECTION_ASC
	OrderDirectionDesc        = dashboards.OrderDirection_ORDER_DIRECTION_DESC
	OrderDirectionUnspecified = dashboards.OrderDirection_ORDER_DIRECTION_UNSPECIFIED
)

// LineChartTooltip is a tooltip for the line chart.
type LineChartTooltip = dashboards.LineChart_Tooltip

// LineChartTooltipType is the type of the line chart tooltip.
type LineChartTooltipType = dashboards.LineChart_TooltipType

// LineChartTooltipType values.
const (
	LineChartToolTipTypeUnspecified = dashboards.LineChart_TOOLTIP_TYPE_UNSPECIFIED
	LineChartToolTipTypeAll         = dashboards.LineChart_TOOLTIP_TYPE_ALL
	LineChartToolTipTypeSingle      = dashboards.LineChart_TOOLTIP_TYPE_SINGLE
)

// ScaleType is the type of the scale.
type ScaleType = dashboards.ScaleType

// ScaleType values.
const (
	ScaleTypeUnspecified = dashboards.ScaleType_SCALE_TYPE_UNSPECIFIED
	ScaleTypeLinear      = dashboards.ScaleType_SCALE_TYPE_LINEAR
	ScaleTypeLogarithmic = dashboards.ScaleType_SCALE_TYPE_LOGARITHMIC
)

// Unit is the unit for a widget.
type Unit = dashboards.Unit

// Unit values.
const (
	UnitUnspecified  = dashboards.Unit_UNIT_UNSPECIFIED
	UnitMicroseconds = dashboards.Unit_UNIT_MICROSECONDS
	UnitMilliseconds = dashboards.Unit_UNIT_MILLISECONDS
	UnitNanoseconds  = dashboards.Unit_UNIT_NANOSECONDS
	UnitSeconds      = dashboards.Unit_UNIT_SECONDS
	UnitBytes        = dashboards.Unit_UNIT_BYTES
	UnitKbytes       = dashboards.Unit_UNIT_KBYTES
	UnitMbytes       = dashboards.Unit_UNIT_MBYTES
	UnitGbytes       = dashboards.Unit_UNIT_GBYTES
	UnitBytesIec     = dashboards.Unit_UNIT_BYTES_IEC
	UnitKibytes      = dashboards.Unit_UNIT_KIBYTES
	UnitMibytes      = dashboards.Unit_UNIT_MIBYTES
	UnitGibytes      = dashboards.Unit_UNIT_GIBYTES
	UnitEurCents     = dashboards.Unit_UNIT_EUR_CENTS
	UnitEur          = dashboards.Unit_UNIT_EUR
	UnitUsdCents     = dashboards.Unit_UNIT_USD_CENTS
	UnitUsd          = dashboards.Unit_UNIT_USD
	UnitCustom       = dashboards.Unit_UNIT_CUSTOM
	UnitPercent01    = dashboards.Unit_UNIT_PERCENT_ZERO_ONE
	UnitPercent100   = dashboards.Unit_UNIT_PERCENT_ZERO_HUNDRED
)

// GaugeUnit is the unit for a widget.
type GaugeUnit = dashboards.Gauge_Unit

// GaugeUnit values.
const (
	GaugeUnitUnspecified  = dashboards.Gauge_UNIT_UNSPECIFIED
	GaugeUnitMicroseconds = dashboards.Gauge_UNIT_MICROSECONDS
	GaugeUnitMilliseconds = dashboards.Gauge_UNIT_MILLISECONDS
	GaugeUnitNanoseconds  = dashboards.Gauge_UNIT_NANOSECONDS
	GaugeUnitNumber       = dashboards.Gauge_UNIT_NUMBER
	GaugeUnitPercent      = dashboards.Gauge_UNIT_PERCENT
	GaugeUnitSeconds      = dashboards.Gauge_UNIT_SECONDS
	GaugeUnitBytes        = dashboards.Gauge_UNIT_BYTES
	GaugeUnitKbytes       = dashboards.Gauge_UNIT_KBYTES
	GaugeUnitMbytes       = dashboards.Gauge_UNIT_MBYTES
	GaugeUnitGbytes       = dashboards.Gauge_UNIT_GBYTES
	GaugeUnitBytesIec     = dashboards.Gauge_UNIT_BYTES_IEC
	GaugeUnitKibytes      = dashboards.Gauge_UNIT_KIBYTES
	GaugeUnitMibytes      = dashboards.Gauge_UNIT_MIBYTES
	GaugeUnitGibytes      = dashboards.Gauge_UNIT_GIBYTES
	GaugeUnitEurCents     = dashboards.Gauge_UNIT_EUR_CENTS
	GaugeUnitEur          = dashboards.Gauge_UNIT_EUR
	GaugeUnitUsdCents     = dashboards.Gauge_UNIT_USD_CENTS
	GaugeUnitUsd          = dashboards.Gauge_UNIT_USD
	GaugeUnitCustom       = dashboards.Gauge_UNIT_CUSTOM
	GaugeUnitPercent01    = dashboards.Gauge_UNIT_PERCENT_ZERO_ONE
	GaugeUnitPercent100   = dashboards.Gauge_UNIT_PERCENT_ZERO_HUNDRED
)

// PieChartLabelSource is the source of the labels for the pie chart.
type PieChartLabelSource = dashboards.PieChart_LabelSource

// PieChartLabelSource values.
const (
	PieChartLabelSourceUnspecified = dashboards.PieChart_LABEL_SOURCE_UNSPECIFIED
	PieChartLabelSourceInner       = dashboards.PieChart_LABEL_SOURCE_INNER
	PieChartLabelSourceStack       = dashboards.PieChart_LABEL_SOURCE_STACK
)

// PieChartLabelDefinition is a label definition type.
type PieChartLabelDefinition = dashboards.PieChart_LabelDefinition

// PieChartStackDefinition is a stack definition type.
type PieChartStackDefinition = dashboards.PieChart_StackDefinition

// BarChartStackDefinition is a stack definition type.
type BarChartStackDefinition = dashboards.BarChart_StackDefinition

// BarChartDataprimeQuery is a type for bar chart data prime query.
type BarChartDataprimeQuery = dashboards.BarChart_DataprimeQuery

// HorizontalBarChartStackDefinition is the source of the labels for the horizontal bar chart.
type HorizontalBarChartStackDefinition = dashboards.HorizontalBarChart_StackDefinition

// SpansAggregation is a type for spans aggregation.
type SpansAggregation = dashboards.SpansAggregation

// SpansAggregationMetricAggregation is a type for metric aggregation.
type SpansAggregationMetricAggregation = dashboards.SpansAggregation_MetricAggregation_

// SpansAggregationMetricAggregationInner is a type for spans aggregation metric aggregation.
type SpansAggregationMetricAggregationInner = dashboards.SpansAggregation_MetricAggregation

// SpansAggregationDimensionAggregation is a type for aggregation.
type SpansAggregationDimensionAggregation = dashboards.SpansAggregation_DimensionAggregation_

// SpansAggregationDimensionAggregationInner is a type for spans aggregation metric aggregation.
type SpansAggregationDimensionAggregationInner = dashboards.SpansAggregation_DimensionAggregation

// SpansAggregationDimensionAggregationDimensionField is a type for spans aggregation dimension aggregation dimension field.
type SpansAggregationDimensionAggregationDimensionField = dashboards.SpansAggregation_DimensionAggregation_DimensionField

// SpansAggregationDimensionAggregationDimensionField values.
const (
	SpansAggregationDimensionAggregationDimensionFieldUnspecified = dashboards.SpansAggregation_DimensionAggregation_DIMENSION_FIELD_UNSPECIFIED
	SpansAggregationDimensionAggregationDimensionFieldTraceID     = dashboards.SpansAggregation_DimensionAggregation_DIMENSION_FIELD_TRACE_ID
)

// SpansAggregationMetricAggregationMetricAggregationType is a type for spans aggregation metric aggregation metric aggregation type.
type SpansAggregationMetricAggregationMetricAggregationType = dashboards.SpansAggregation_MetricAggregation_MetricAggregationType

// SpansAggregationMetricAggregationMetricTypeUnspecified values.
const (
	SpansAggregationMetricAggregationMetricTypeUnspecified  = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_UNSPECIFIED
	SpansAggregationMetricAggregationMetricTypeMin          = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_MIN
	SpansAggregationMetricAggregationMetricTypeMax          = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_MAX
	SpansAggregationMetricAggregationMetricTypeAverage      = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_AVERAGE
	SpansAggregationMetricAggregationMetricTypeSum          = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_SUM
	SpansAggregationMetricAggregationMetricTypePercentile99 = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_PERCENTILE_99
	SpansAggregationMetricAggregationMetricTypePercentile95 = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_PERCENTILE_95
	SpansAggregationMetricAggregationMetricTypePercentile50 = dashboards.SpansAggregation_MetricAggregation_METRIC_AGGREGATION_TYPE_PERCENTILE_50
)

// SpansAggregationMetricAggregationMetricField is a type for spans aggregation metric aggregation metric field.
type SpansAggregationMetricAggregationMetricField = dashboards.SpansAggregation_MetricAggregation_MetricField

// SpansAggregationMetricAggregationMetricField values.
const (
	SpansAggregationMetricAggregationMetricFieldUnspecified = dashboards.SpansAggregation_MetricAggregation_METRIC_FIELD_UNSPECIFIED
	SpansAggregationMetricAggregationMetricFieldDuration    = dashboards.SpansAggregation_MetricAggregation_METRIC_FIELD_DURATION
)

// UUID is a type wrapper for UUID.
type UUID = dashboards.UUID

// GaugeQueryMetrics is a type for gauge query for metrics.
type GaugeQueryMetrics = dashboards.Gauge_Query_Metrics

// GaugeQueryLogs is a type for gauge query for logs.
type GaugeQueryLogs = dashboards.Gauge_Query_Logs

// GaugeQuerySpans is a type for gauge query for spans.
type GaugeQuerySpans = dashboards.Gauge_Query_Spans

// GaugeSpansQuery is a type of gauge query for spans.
type GaugeSpansQuery = dashboards.Gauge_SpansQuery

// GaugeQuery is a type for gauge query.
type GaugeQuery = dashboards.Gauge_Query

// GaugeAggregation is a type for gauge aggregation.
type GaugeAggregation = dashboards.Gauge_Aggregation

// GaugeAggregation values.
const (
	GaugeAggregationUnspecified = dashboards.Gauge_AGGREGATION_UNSPECIFIED
	GaugeAggregationLast        = dashboards.Gauge_AGGREGATION_LAST
	GaugeAggregationMin         = dashboards.Gauge_AGGREGATION_MIN
	GaugeAggregationMax         = dashboards.Gauge_AGGREGATION_MAX
	GaugeAggregationAvg         = dashboards.Gauge_AGGREGATION_AVG
	GaugeAggregationSum         = dashboards.Gauge_AGGREGATION_SUM
)

// RPC names.
const (
	CreateDashboardRPC         = dashboards.DashboardsService_CreateDashboard_FullMethodName
	ReplaceDashboardRPC        = dashboards.DashboardsService_ReplaceDashboard_FullMethodName
	DeleteDashboardRPC         = dashboards.DashboardsService_DeleteDashboard_FullMethodName
	GetDashboardRPC            = dashboards.DashboardsService_GetDashboard_FullMethodName
	PinDashboardRPC            = dashboards.DashboardsService_PinDashboard_FullMethodName
	UnpinDashboardRPC          = dashboards.DashboardsService_UnpinDashboard_FullMethodName
	ReplaceDefaultDashboardRPC = dashboards.DashboardsService_ReplaceDefaultDashboard_FullMethodName
	AssignDashboardFolderRPC   = dashboards.DashboardsService_AssignDashboardFolder_FullMethodName
)

// SortByType is the type for sorting by.
type SortByType = dashboards.SortByType

// SortByType values.
const (
	SortByTypeUnspecified = dashboards.SortByType_SORT_BY_TYPE_UNSPECIFIED
	SortByTypeValue       = dashboards.SortByType_SORT_BY_TYPE_VALUE
	SortByTypeName        = dashboards.SortByType_SORT_BY_TYPE_NAME
)

// SpanFieldMetadataField values.
const (
	SpanFieldMetadataFieldUnspecified     = dashboards.SpanField_METADATA_FIELD_UNSPECIFIED
	SpanFieldMetadataFieldApplicationName = dashboards.SpanField_METADATA_FIELD_APPLICATION_NAME
	SpanFieldMetadataFieldSubsystemName   = dashboards.SpanField_METADATA_FIELD_SUBSYSTEM_NAME
	SpanFieldMetadataFieldServiceName     = dashboards.SpanField_METADATA_FIELD_SERVICE_NAME
	SpanFieldMetadataFieldOperationName   = dashboards.SpanField_METADATA_FIELD_OPERATION_NAME
)

// SpansAggregationDimensionAggregationType is the type for spans aggregation dimension aggregation type.
type SpansAggregationDimensionAggregationType = dashboards.SpansAggregation_DimensionAggregation_DimensionAggregationType

// SpansAggregationDimensionAggregationType values.
const (
	SpansAggregationDimensionAggregationTypeUnspecified = dashboards.SpansAggregation_DimensionAggregation_DIMENSION_AGGREGATION_TYPE_UNSPECIFIED
	SpansAggregationDimensionAggregationTypeUniqueCount = dashboards.SpansAggregation_DimensionAggregation_DIMENSION_AGGREGATION_TYPE_UNIQUE_COUNT
	SpansAggregationDimensionAggregationTypeErrorCount  = dashboards.SpansAggregation_DimensionAggregation_DIMENSION_AGGREGATION_TYPE_ERROR_COUNT
)

// DatasetScope is the scope of the dataset.
type DatasetScope = dashboards.DatasetScope

// DatasetScope values.
const (
	DatasetScopeUnspecified = dashboards.DatasetScope_DATASET_SCOPE_UNSPECIFIED
	DatasetScopeUserData    = dashboards.DatasetScope_DATASET_SCOPE_USER_DATA
	DatasetScopeLabel       = dashboards.DatasetScope_DATASET_SCOPE_LABEL
	DatasetScopeMetadata    = dashboards.DatasetScope_DATASET_SCOPE_METADATA
)

// DataModeType is the type for data mode.
type DataModeType = dashboards.DataModeType

// DataModeType values.
const (
	DataModeTypeHighUnspecified = dashboards.DataModeType_DATA_MODE_TYPE_HIGH_UNSPECIFIED
	DataModeTypeArchive         = dashboards.DataModeType_DATA_MODE_TYPE_ARCHIVE
)

// GaugeThreshold is the type for gauge threshold.
type GaugeThreshold = dashboards.Gauge_Threshold

// GaugeThresholdBy is the type for gauge threshold by.
type GaugeThresholdBy = dashboards.Gauge_ThresholdBy

// GaugeThresholdBy values.
const (
	GaugeThresholdByUnspecified = dashboards.Gauge_THRESHOLD_BY_UNSPECIFIED
	GaugeThresholdByValue       = dashboards.Gauge_THRESHOLD_BY_VALUE
	GaugeThresholdByBackground  = dashboards.Gauge_THRESHOLD_BY_BACKGROUND
)

// MultiSelectRefreshStrategy is the type for multi select refresh strategy.
type MultiSelectRefreshStrategy = dashboards.MultiSelect_RefreshStrategy

// MultiSelectRefreshStrategy values.
const (
	MultiSelectRefreshStrategyUnspecified       = dashboards.MultiSelect_REFRESH_STRATEGY_UNSPECIFIED
	MultiSelectRefreshStrategyOnDashboardLoad   = dashboards.MultiSelect_REFRESH_STRATEGY_ON_DASHBOARD_LOAD
	MultiSelectRefreshStrategyOnTimeFrameChange = dashboards.MultiSelect_REFRESH_STRATEGY_ON_TIME_FRAME_CHANGE
)

// Annotation is an annotation for dashboards.
type Annotation = dashboards.Annotation

// DashboardFilter is a filter for dashboards.
type DashboardFilter = dashboards.Filter

// DashboardFilterSource is a filter source for dashboards.
type DashboardFilterSource = dashboards.Filter_Source

// DashboardFilterSourceSpans is a filter source for dashboards.
type DashboardFilterSourceSpans = dashboards.Filter_Source_Spans

// DashboardFilterSourceLogs is a filter source for dashboards.
type DashboardFilterSourceLogs = dashboards.Filter_Source_Logs

// DashboardFilterSourceMetrics is a filter source for dashboards.
type DashboardFilterSourceMetrics = dashboards.Filter_Source_Metrics

// DashboardFilterSpansFilter is a filter type.
type DashboardFilterSpansFilter = dashboards.Filter_SpansFilter

// DashboardFilterLogsFilter is a filter type.
type DashboardFilterLogsFilter = dashboards.Filter_LogsFilter

// DashboardFilterMetricsFilter is a filter type.
type DashboardFilterMetricsFilter = dashboards.Filter_MetricsFilter

// SpanField is a filter type.
type SpanField = dashboards.SpanField

// SpanFieldMetadataField is the type for span field metadata field.
type SpanFieldMetadataField = dashboards.SpanField_MetadataField_

// SpanFieldMetadataFieldInner is the type for span field metadata field.
type SpanFieldMetadataFieldInner = dashboards.SpanField_MetadataField

// SpanFieldTagField is a tag field type for spans.
type SpanFieldTagField = dashboards.SpanField_TagField

// SpanFieldProcessTagField is a tag field type for spans.
type SpanFieldProcessTagField = dashboards.SpanField_ProcessTagField

// MultiSelectValueDisplayOptions is a type for display options.
type MultiSelectValueDisplayOptions = dashboards.MultiSelect_ValueDisplayOptions

// MultiSelectSource is a source for multi select.
type MultiSelectSource = dashboards.MultiSelect_Source

// MultiSelectSourceQuery is a source for multi select.
type MultiSelectSourceQuery = dashboards.MultiSelect_Source_Query

// MultiSelectQuery is a query type for multi select.
type MultiSelectQuery = dashboards.MultiSelect_Query

// MultiSelectQuerySource is a source for multi select.
type MultiSelectQuerySource = dashboards.MultiSelect_QuerySource

// MultiSelectQueryLogsQuery is a query type for multi select.
type MultiSelectQueryLogsQuery = dashboards.MultiSelect_Query_LogsQuery_

// MultiSelectQueryLogsQueryInner is a query type for multi select.
type MultiSelectQueryLogsQueryInner = dashboards.MultiSelect_Query_LogsQuery

// MultiSelectQueryLogsQueryType is a query type for multi select.
type MultiSelectQueryLogsQueryType = dashboards.MultiSelect_Query_LogsQuery_Type

// MultiSelectSourceSpanField is a source for multi select.
type MultiSelectSourceSpanField = dashboards.MultiSelect_Source_SpanField

// MultiSelectSpanFieldSource is a source for multi select.
type MultiSelectSpanFieldSource = dashboards.MultiSelect_SpanFieldSource

// MultiSelectSourceConstantList is a source for multi select.
type MultiSelectSourceConstantList = dashboards.MultiSelect_Source_ConstantList

// MultiSelectConstantListSource is a source for multi select.
type MultiSelectConstantListSource = dashboards.MultiSelect_ConstantListSource

// MultiSelectSourceMetricLabel is a source for multi select.
type MultiSelectSourceMetricLabel = dashboards.MultiSelect_Source_MetricLabel

// MultiSelectMetricLabelSource is a source for multi select.
type MultiSelectMetricLabelSource = dashboards.MultiSelect_MetricLabelSource

// MultiSelectLogsPathSource is a source for multi select.
type MultiSelectLogsPathSource = dashboards.MultiSelect_LogsPathSource

// MultiSelectSourceLogsPath is a source for multi select.
type MultiSelectSourceLogsPath = dashboards.MultiSelect_Source_LogsPath

// WidgetDefinitionMarkdown is a widget definition.
type WidgetDefinitionMarkdown = dashboards.Widget_Definition_Markdown

// WidgetDefinitionHorizontalBarChart is a widget definition.
type WidgetDefinitionHorizontalBarChart = dashboards.Widget_Definition_HorizontalBarChart

// WidgetDefinitionLineChart is a widget definition.
type WidgetDefinitionLineChart = dashboards.Widget_Definition_LineChart

// WidgetDefinitionDataTable is a widget definition.
type WidgetDefinitionDataTable = dashboards.Widget_Definition_DataTable

// WidgetDefinitionGauge is a widget definition.
type WidgetDefinitionGauge = dashboards.Widget_Definition_Gauge

// WidgetDefinitionPieChart is a widget definition.
type WidgetDefinitionPieChart = dashboards.Widget_Definition_PieChart

// WidgetDefinitionBarChart is a widget definition.
type WidgetDefinitionBarChart = dashboards.Widget_Definition_BarChart

// Markdown is a widget definition inner type.
type Markdown = dashboards.Markdown

// HorizontalBarChart is a widget definition inner type.
type HorizontalBarChart = dashboards.HorizontalBarChart

// LineChart is a widget definition inner type.
type LineChart = dashboards.LineChart

// DataTable is a widget definition inner type.
type DataTable = dashboards.DataTable

// Gauge is a widget definition inner type.
type Gauge = dashboards.Gauge

// PieChart is a widget definition inner type.
type PieChart = dashboards.PieChart

// BarChart is a widget definition inner type.
type BarChart = dashboards.BarChart

// WidgetDefinition is a widget definition.
type WidgetDefinition = dashboards.Widget_Definition

// DashboardRow is a row in a dashboard.
type DashboardRow = dashboards.Row

// DashboardRowAppearance is the appearance of a row in a dashboard.
type DashboardRowAppearance = dashboards.Row_Appearance

// DashboardWidget is a widget in a dashboard.
type DashboardWidget = dashboards.Widget

// DashboardWidgetAppearance is the appearance of a widget in a dashboard.
type DashboardWidgetAppearance = dashboards.Widget_Appearance

// MultiSelectQueryMetricsQueryStringOrVariable is a query string type.
type MultiSelectQueryMetricsQueryStringOrVariable = dashboards.MultiSelect_Query_MetricsQuery_StringOrVariable

// MultiSelectQueryMetricsQueryStringOrVariableString is a string query type.
type MultiSelectQueryMetricsQueryStringOrVariableString = dashboards.MultiSelect_Query_MetricsQuery_StringOrVariable_StringValue

// MultiSelectQueryMetricsQueryStringOrVariableVariable is a variable query type.
type MultiSelectQueryMetricsQueryStringOrVariableVariable = dashboards.MultiSelect_Query_MetricsQuery_StringOrVariable_VariableName

// MultiSelectQueryMetricsQuerySelection is a selection for a multi select query.
type MultiSelectQueryMetricsQuerySelection = dashboards.MultiSelect_Query_MetricsQuery_Selection

// MultiSelectQueryMetricsQuerySelectionList is a selection for a multi select query.
type MultiSelectQueryMetricsQuerySelectionList = dashboards.MultiSelect_Query_MetricsQuery_Selection_List

// MultiSelectQueryMetricsQuerySelectionListSelection is a selection for a multi select query.
type MultiSelectQueryMetricsQuerySelectionListSelection = dashboards.MultiSelect_Query_MetricsQuery_Selection_ListSelection

// MultiSelectQueryMetricsQueryOperatorEquals is a selection for a multi select query.
type MultiSelectQueryMetricsQueryOperatorEquals = dashboards.MultiSelect_Query_MetricsQuery_Operator_Equals

// MultiSelectQueryMetricsQueryEquals is a selection for a multi select query.
type MultiSelectQueryMetricsQueryEquals = dashboards.MultiSelect_Query_MetricsQuery_Equals

// MultiSelectQueryMetricsQueryOperator is a selection for a multi select query.
type MultiSelectQueryMetricsQueryOperator = dashboards.MultiSelect_Query_MetricsQuery_Operator

// MultiSelectQueryMetricsQueryOperatorNotEquals is a selection for a multi select query.
type MultiSelectQueryMetricsQueryOperatorNotEquals = dashboards.MultiSelect_Query_MetricsQuery_Operator_NotEquals

// MultiSelectQueryMetricsQueryNotEquals is a selection for a multi select query.
// MultiSelectQueryMetricsQueryNotEquals is a metrics query operator.
type MultiSelectQueryMetricsQueryNotEquals = dashboards.MultiSelect_Query_MetricsQuery_NotEquals

// MultiSelectQuerySpansQuery is a spans query.
type MultiSelectQuerySpansQuery = dashboards.MultiSelect_Query_SpansQuery_

// MultiSelectQuerySpansQueryInner is a spans query inner type.
type MultiSelectQuerySpansQueryInner = dashboards.MultiSelect_Query_SpansQuery

// MultiSelectQuerySpansQueryType is a spans query type.
type MultiSelectQuerySpansQueryType = dashboards.MultiSelect_Query_SpansQuery_Type

// MultiSelectQuerySpansQueryTypeFieldName is a field name for the spans query.
type MultiSelectQuerySpansQueryTypeFieldName = dashboards.MultiSelect_Query_SpansQuery_Type_FieldName_

// MultiSelectQuerySpansQueryTypeFieldNameInner is a field name for the spans query.
type MultiSelectQuerySpansQueryTypeFieldNameInner = dashboards.MultiSelect_Query_SpansQuery_Type_FieldName

// MultiSelectQuerySpansQueryTypeFieldValue is a field value for the spans query.
type MultiSelectQuerySpansQueryTypeFieldValue = dashboards.MultiSelect_Query_SpansQuery_Type_FieldValue_

// MultiSelectQuerySpansQueryTypeFieldValueInner is a field value for the spans query.
type MultiSelectQuerySpansQueryTypeFieldValueInner = dashboards.MultiSelect_Query_SpansQuery_Type_FieldValue

// MultiSelectQueryLogsQueryTypeFieldName is a field name for the logs query.
type MultiSelectQueryLogsQueryTypeFieldName = dashboards.MultiSelect_Query_LogsQuery_Type_FieldName_

// MultiSelectQueryLogsQueryTypeFieldNameInner is a field name for the logs query.
type MultiSelectQueryLogsQueryTypeFieldNameInner = dashboards.MultiSelect_Query_LogsQuery_Type_FieldName

// MultiSelectQueryLogsQueryTypeFieldValue is a field value for the logs query.
type MultiSelectQueryLogsQueryTypeFieldValue = dashboards.MultiSelect_Query_LogsQuery_Type_FieldValue_

// MultiSelectQueryLogsQueryTypeFieldValueInner is a field value for the logs query.
type MultiSelectQueryLogsQueryTypeFieldValueInner = dashboards.MultiSelect_Query_LogsQuery_Type_FieldValue

// MultiSelectQueryMetricsQuery is a metrics query.
type MultiSelectQueryMetricsQuery = dashboards.MultiSelect_Query_MetricsQuery_

// MultiSelectQueryMetricsQueryInner is a metrics query.
type MultiSelectQueryMetricsQueryInner = dashboards.MultiSelect_Query_MetricsQuery

// MultiSelectQueryMetricsQueryType is a metrics query.
type MultiSelectQueryMetricsQueryType = dashboards.MultiSelect_Query_MetricsQuery_Type

// MultiSelectQueryMetricsQueryTypeMetricName is a field name for the metrics query.
type MultiSelectQueryMetricsQueryTypeMetricName = dashboards.MultiSelect_Query_MetricsQuery_Type_MetricName_

// MultiSelectQueryMetricsQueryTypeMetricNameInner is a field name for the metrics query.
type MultiSelectQueryMetricsQueryTypeMetricNameInner = dashboards.MultiSelect_Query_MetricsQuery_Type_MetricName

// MultiSelectQueryMetricsQueryTypeLabelValue is a field value for the metrics query.
type MultiSelectQueryMetricsQueryTypeLabelValue = dashboards.MultiSelect_Query_MetricsQuery_Type_LabelValue_

// MultiSelectQueryMetricsQueryTypeLabelValueInner is a field value for the metrics query.
type MultiSelectQueryMetricsQueryTypeLabelValueInner = dashboards.MultiSelect_Query_MetricsQuery_Type_LabelValue

// MultiSelectQueryMetricsQueryTypeLabelName is a field value for the metrics query.
type MultiSelectQueryMetricsQueryTypeLabelName = dashboards.MultiSelect_Query_MetricsQuery_Type_LabelName_

// MultiSelectQueryMetricsQueryTypeLabelNameInner is a field value for the metrics query.
type MultiSelectQueryMetricsQueryTypeLabelNameInner = dashboards.MultiSelect_Query_MetricsQuery_Type_LabelName

// MultiSelectQueryMetricsQueryMetricsLabelFilter is a metrics query label filter.
type MultiSelectQueryMetricsQueryMetricsLabelFilter = dashboards.MultiSelect_Query_MetricsQuery_MetricsLabelFilter

// GaugeMetricsQuery is a gauge metrics query.
type GaugeMetricsQuery = dashboards.Gauge_MetricsQuery

// DashboardMetricsFilter is a metrics filter.
type DashboardMetricsFilter = dashboards.Filter_MetricsFilter

// DashboardFilterOperatorEquals is a filter operator.
type DashboardFilterOperatorEquals = dashboards.Filter_Operator_Equals

// DashboardFilterEquals is a filter operator: equals.
type DashboardFilterEquals = dashboards.Filter_Equals

// DashboardFilterEqualsSelection is a filter equals selection.
type DashboardFilterEqualsSelection = dashboards.Filter_Equals_Selection

// DashboardFilterEqualsSelectionList is a filter equals selection list.
type DashboardFilterEqualsSelectionList = dashboards.Filter_Equals_Selection_List

// DashboardFilterEqualsSelectionListSelection is a filter equals selection list selection.
type DashboardFilterEqualsSelectionListSelection = dashboards.Filter_Equals_Selection_ListSelection

// DashboardFilterEqualsSelectionAll is a filter equals selection all.
type DashboardFilterEqualsSelectionAll = dashboards.Filter_Equals_Selection_All

// DashboardFilterEqualsSelectionAllSelection is a filter equals selection all selection.
type DashboardFilterEqualsSelectionAllSelection = dashboards.Filter_Equals_Selection_AllSelection

// DashboardFilterOperator is a filter operator.
type DashboardFilterOperator = dashboards.Filter_Operator

// DashboardFilterOperatorNotEquals is a filter operator.
type DashboardFilterOperatorNotEquals = dashboards.Filter_Operator_NotEquals

// DashboardFilterNotEquals is a filter operator not equals.
type DashboardFilterNotEquals = dashboards.Filter_NotEquals

// DashboardFilterNotEqualsSelection is a filter not equals selection.
type DashboardFilterNotEqualsSelection = dashboards.Filter_NotEquals_Selection

// DashboardFilterNotEqualsSelectionList is a filter not equals selection list.
type DashboardFilterNotEqualsSelectionList = dashboards.Filter_NotEquals_Selection_List

// DashboardFilterNotEqualsSelectionListSelection is a filter not equals selection list selection.
type DashboardFilterNotEqualsSelectionListSelection = dashboards.Filter_NotEquals_Selection_ListSelection

// DashboardPromQlQuery is a promQL query.
type DashboardPromQlQuery = dashboards.PromQlQuery

// DashboardLuceneQuery is a lucene query.
type DashboardLuceneQuery = dashboards.LuceneQuery

// GaugeLogsQuery is a logs query for gauges.
type GaugeLogsQuery = dashboards.Gauge_LogsQuery

// LogsAggregation is a logs aggregation.
type LogsAggregation = dashboards.LogsAggregation

// LogsAggregationCount is a logs aggregation type.
type LogsAggregationCount = dashboards.LogsAggregation_Count_

// LogsAggregationCountInner is a logs aggregation type.
type LogsAggregationCountInner = dashboards.LogsAggregation_Count

// LogsAggregationCountDistinct is a logs aggregation type.
type LogsAggregationCountDistinct = dashboards.LogsAggregation_CountDistinct_

// LogsAggregationCountDistinctInner is a logs aggregation type.
type LogsAggregationCountDistinctInner = dashboards.LogsAggregation_CountDistinct

// LogsAggregationSum is a logs aggregation type.
type LogsAggregationSum = dashboards.LogsAggregation_Sum_

// LogsAggregationSumInner is a logs aggregation type.
type LogsAggregationSumInner = dashboards.LogsAggregation_Sum

// LogsAggregationAverage is a logs aggregation type.
type LogsAggregationAverage = dashboards.LogsAggregation_Average_

// LogsAggregationAverageInner is a logs aggregation type.
type LogsAggregationAverageInner = dashboards.LogsAggregation_Average

// LogsAggregationMin is a logs aggregation type.
type LogsAggregationMin = dashboards.LogsAggregation_Min_

// LogsAggregationMinInner is a logs aggregation type.
type LogsAggregationMinInner = dashboards.LogsAggregation_Min

// LogsAggregationMax is a logs aggregation type.
type LogsAggregationMax = dashboards.LogsAggregation_Max_

// LogsAggregationMaxInner is a logs aggregation type.
type LogsAggregationMaxInner = dashboards.LogsAggregation_Max

// LogsAggregationPercentile is a logs aggregation type.
type LogsAggregationPercentile = dashboards.LogsAggregation_Percentile_

// LogsAggregationPercentileInner is a logs aggregation type.
type LogsAggregationPercentileInner = dashboards.LogsAggregation_Percentile

// AnnotationSource is an annotation variant.
type AnnotationSource = dashboards.Annotation_Source

// AnnotationSourceLogs is an annotation variant.
type AnnotationSourceLogs = dashboards.Annotation_Source_Logs

// AnnotationLogsSource is an annotation variant.
type AnnotationLogsSource = dashboards.Annotation_LogsSource

// AnnotationLogsSourceStrategy is an annotation variant.
type AnnotationLogsSourceStrategy = dashboards.Annotation_LogsSource_Strategy

// AnnotationSourceSpans is an annotation variant.
type AnnotationSourceSpans = dashboards.Annotation_Source_Spans

// AnnotationSpansSource is an annotation variant.
type AnnotationSpansSource = dashboards.Annotation_SpansSource

// AnnotationSpansSourceStrategy is an annotation variant.
type AnnotationSpansSourceStrategy = dashboards.Annotation_SpansSource_Strategy

// AnnotationSourceMetrics is an annotation variant.
type AnnotationSourceMetrics = dashboards.Annotation_Source_Metrics

// AnnotationMetricsSource is an annotation variant.
type AnnotationMetricsSource = dashboards.Annotation_MetricsSource

// AnnotationMetricsSourceStrategy is an annotation variant.
type AnnotationMetricsSourceStrategy = dashboards.Annotation_MetricsSource_Strategy

// AnnotationSpansSourceStrategyDurationInner is an annotation variant.
type AnnotationSpansSourceStrategyDurationInner = dashboards.Annotation_SpansSource_Strategy_Duration

// AnnotationSpansSourceStrategyDuration is an annotation variant.
type AnnotationSpansSourceStrategyDuration = dashboards.Annotation_SpansSource_Strategy_Duration_

// AnnotationSpansSourceStrategyRange is an annotation variant.
type AnnotationSpansSourceStrategyRange = dashboards.Annotation_SpansSource_Strategy_Range_

// AnnotationSpansSourceStrategyRangeInner is an annotation variant.
type AnnotationSpansSourceStrategyRangeInner = dashboards.Annotation_SpansSource_Strategy_Range

// AnnotationSpansSourceStrategyInstant is an annotation variant.
type AnnotationSpansSourceStrategyInstant = dashboards.Annotation_SpansSource_Strategy_Instant_

// AnnotationSpansSourceStrategyInstantInner is an annotation variant.
type AnnotationSpansSourceStrategyInstantInner = dashboards.Annotation_SpansSource_Strategy_Instant

// AnnotationLogsSourceStrategyDurationInner is an annotation variant.
type AnnotationLogsSourceStrategyDurationInner = dashboards.Annotation_LogsSource_Strategy_Duration

// AnnotationLogsSourceStrategyDuration is an annotation variant.
type AnnotationLogsSourceStrategyDuration = dashboards.Annotation_LogsSource_Strategy_Duration_

// AnnotationLogsSourceStrategyRange is an annotation variant.
type AnnotationLogsSourceStrategyRange = dashboards.Annotation_LogsSource_Strategy_Range_

// AnnotationLogsSourceStrategyRangeInner is an annotation variant.
type AnnotationLogsSourceStrategyRangeInner = dashboards.Annotation_LogsSource_Strategy_Range

// AnnotationLogsSourceStrategyInstant is an annotation variant.
type AnnotationLogsSourceStrategyInstant = dashboards.Annotation_LogsSource_Strategy_Instant_

// AnnotationLogsSourceStrategyInstantInner is an annotation variant.
type AnnotationLogsSourceStrategyInstantInner = dashboards.Annotation_LogsSource_Strategy_Instant

// AnnotationMetricsSourceStartTimeMetric is an annotation variant.
type AnnotationMetricsSourceStartTimeMetric = dashboards.Annotation_MetricsSource_StartTimeMetric

// AnnotationMetricsSourceStrategyStartTimeMetric is an annotation variant.
type AnnotationMetricsSourceStrategyStartTimeMetric = dashboards.Annotation_MetricsSource_Strategy_StartTimeMetric

// AnnotationMetricsSourceStrategyRange is an annotation variant.
// type AnnotationMetricsSourceStrategyRange = dashboards.Annotation_MetricsSource_Strategy_Range_

// // AnnotationMetricsSourceStrategyRangeInner is an annotation variant.
// type AnnotationMetricsSourceStrategyRangeInner = dashboards.Annotation_MetricsSource_Strategy_Range

// // AnnotationMetricsSourceStrategyInstant is an annotation variant.
// type AnnotationMetricsSourceStrategyInstant = dashboards.Annotation_MetricsSource_Strategy_Instant_

// // AnnotationMetricsSourceStrategyInstantInner is an annotation variant.
// type AnnotationMetricsSourceStrategyInstantInner = dashboards.Annotation_MetricsSource_Strategy_Instant

// DashboardLayout is a dashboard layout type.
type DashboardLayout = dashboards.Layout

// DashboardSection is a dashboard layout type.
type DashboardSection = dashboards.Section

// DashboardSectionOptions is a dashboard layout type.
type DashboardSectionOptions = dashboards.SectionOptions

// DashboardSectionOptionsCustom is a customizable dashboard layout option.
type DashboardSectionOptionsCustom = dashboards.SectionOptions_Custom

// CustomSectionOptions is a type for customizing section options.
type CustomSectionOptions = dashboards.CustomSectionOptions

// DashboardSectionColor is a dashboard layout type.
type DashboardSectionColor = dashboards.SectionColor

// DashboardSectionColorPredefined is a dashboard layout type.
type DashboardSectionColorPredefined = dashboards.SectionColor_Predefined

// DashboardSectionColorPredefinedColor is a dashboard layout type.
type DashboardSectionColorPredefinedColor = dashboards.SectionPredefinedColor

var (
	// DashboardSectionPredefinedColorValueLookup is a map of predefined color values to strings.
	DashboardSectionPredefinedColorValueLookup = dashboards.SectionPredefinedColor_value
	// DashboardSectionPredefinedColorNameLookup is a map of predefined strings to color values.
	DashboardSectionPredefinedColorNameLookup = dashboards.SectionPredefinedColor_name
)

// DashboardTwoMinutes is a auto refresh setting
type DashboardTwoMinutes = dashboards.Dashboard_TwoMinutes

// DashboardAutoRefreshTwoMinutes is a auto refresh setting
type DashboardAutoRefreshTwoMinutes = dashboards.Dashboard_AutoRefreshTwoMinutes

// DashboardFiveMinutes is a auto refresh setting
type DashboardFiveMinutes = dashboards.Dashboard_FiveMinutes

// DashboardAutoRefreshFiveMinutes is a auto refresh setting
type DashboardAutoRefreshFiveMinutes = dashboards.Dashboard_AutoRefreshFiveMinutes

// DashboardOff is a auto refresh setting
type DashboardOff = dashboards.Dashboard_Off

// DashboardAutoRefreshOff is a auto refresh setting
type DashboardAutoRefreshOff = dashboards.Dashboard_AutoRefreshOff

// HorizontalBarChartYAxisViewBy is a type for the horizontal bar chart y axis.
type HorizontalBarChartYAxisViewBy = dashboards.HorizontalBarChart_YAxisViewBy

// HorizontalBarChartYAxisViewByCategory is a type for the horizontal bar chart y axis.
type HorizontalBarChartYAxisViewByCategory = dashboards.HorizontalBarChart_YAxisViewBy_Category

// HorizontalBarChartYAxisViewByValue is a type for the horizontal bar chart y axis.
type HorizontalBarChartYAxisViewByValue = dashboards.HorizontalBarChart_YAxisViewBy_Value

// BarChartXAxis is a type for the  bar chart x axis.
type BarChartXAxis = dashboards.BarChart_XAxis

// BarChartXAxisTime is a type for the  bar chart x axis.
type BarChartXAxisTime = dashboards.BarChart_XAxis_Time

// BarChartXAxisValue is a type for the  bar chart x axis.
type BarChartXAxisValue = dashboards.BarChart_XAxis_Value

// BarChartXAxisByTime is a type for the  bar chart x axis.
type BarChartXAxisByTime = dashboards.BarChart_XAxis_XAxisByTime

// BarChartXAxisByValue is a type for the  bar chart x axis.
type BarChartXAxisByValue = dashboards.BarChart_XAxis_XAxisByValue

// BarChartQuery is a type for bar chart data.
type BarChartQuery = dashboards.BarChart_Query

// BarChartQueryLogs is a type for bar chart data.
type BarChartQueryLogs = dashboards.BarChart_Query_Logs

// BarChartQueryMetrics is a type for bar chart data.
type BarChartQueryMetrics = dashboards.BarChart_Query_Metrics

// BarChartQuerySpans is a type for bar chart data.
type BarChartQuerySpans = dashboards.BarChart_Query_Spans

// BarChartQueryDataprime is a type for bar chart data.
type BarChartQueryDataprime = dashboards.BarChart_Query_Dataprime

// HorizontalBarChartQuery is a type for bar chart data.
type HorizontalBarChartQuery = dashboards.HorizontalBarChart_Query

// HorizontalBarChartQueryLogs is a type for bar chart data.
type HorizontalBarChartQueryLogs = dashboards.HorizontalBarChart_Query_Logs

// HorizontalBarChartQueryMetrics is a type for bar chart data.
type HorizontalBarChartQueryMetrics = dashboards.HorizontalBarChart_Query_Metrics

// HorizontalBarChartQuerySpans is a type for bar chart data.
type HorizontalBarChartQuerySpans = dashboards.HorizontalBarChart_Query_Spans

// HorizontalBarChartQueryDataprime is a type for bar chart data.
type HorizontalBarChartQueryDataprime = dashboards.HorizontalBarChart_Query_Dataprime

// HorizontalBarChartLogsQuery is a type for bar chart data.
type HorizontalBarChartLogsQuery = dashboards.HorizontalBarChart_LogsQuery

// HorizontalBarChartMetricsQuery is a type for bar chart data.
type HorizontalBarChartMetricsQuery = dashboards.HorizontalBarChart_MetricsQuery

// HorizontalBarChartSpansQuery is a type for bar chart data.
type HorizontalBarChartSpansQuery = dashboards.HorizontalBarChart_SpansQuery

// BarChartLogsQuery is a type for data in bar chart.
type BarChartLogsQuery = dashboards.BarChart_LogsQuery

// BarChartMetricsQuery is a type for data in bar chart.
type BarChartMetricsQuery = dashboards.BarChart_MetricsQuery

// BarChartSpansQuery is a type for data in bar chart.
type BarChartSpansQuery = dashboards.BarChart_SpansQuery

// ObservationField is a type for data in bar chart.
type ObservationField = dashboards.ObservationField

// DashboardsColorsBy is a type for dashboard coloring.
type DashboardsColorsBy = dashboards.ColorsBy

// DashboardsColorsByStack is a type for dashboard coloring.
type DashboardsColorsByStack = dashboards.ColorsBy_Stack

// DashboardsColorsByGroupBy is a type for dashboard coloring.
type DashboardsColorsByGroupBy = dashboards.ColorsBy_GroupBy

// DashboardsColorsByAggregation is a type for dashboard coloring.
type DashboardsColorsByAggregation = dashboards.ColorsBy_Aggregation

// DashboardsColorsByStackInner is a type for dashboard coloring.
type DashboardsColorsByStackInner = dashboards.ColorsBy_ColorsByStack

// DashboardsColorsByGroupByInner is a type for dashboard coloring.
type DashboardsColorsByGroupByInner = dashboards.ColorsBy_ColorsByGroupBy

// DashboardsColorsByAggregationInner is a type for dashboard coloring.
type DashboardsColorsByAggregationInner = dashboards.ColorsBy_ColorsByAggregation

// PieChartQuery is a type for data for charts.
type PieChartQuery = dashboards.PieChart_Query

// PieChartQueryDataprime is a type for data for charts.
type PieChartQueryDataprime = dashboards.PieChart_Query_Dataprime

// PieChartDataprimeQuery is a type for data for charts.
type PieChartDataprimeQuery = dashboards.PieChart_DataprimeQuery

// DashboardDataprimeQuery is a type for data for charts.
type DashboardDataprimeQuery = dashboards.DataprimeQuery

// PieChartQuerySpans is a type for data for charts.
type PieChartQuerySpans = dashboards.PieChart_Query_Spans

// PieChartSpansQuery is a type for data for charts.
type PieChartSpansQuery = dashboards.PieChart_SpansQuery

// PieChartQueryMetrics is a type for data for charts.
type PieChartQueryMetrics = dashboards.PieChart_Query_Metrics

// PieChartMetricsQuery is a type for data for charts.
type PieChartMetricsQuery = dashboards.PieChart_MetricsQuery

// PieChartQueryLogs is a type for data for charts.
type PieChartQueryLogs = dashboards.PieChart_Query_Logs

// PieChartLogsQuery is a type for data for charts.
type PieChartLogsQuery = dashboards.PieChart_LogsQuery

// DashboardVariable is a type for data for charts.
type DashboardVariable = dashboards.Variable

// DashboardConstant is a type for data for charts.
type DashboardConstant = dashboards.Constant

// DashboardMultiSelect is a type for data for charts.
type DashboardMultiSelect = dashboards.MultiSelect

// DashboardMultiSelectSource is a type for data for charts.
type DashboardMultiSelectSource = dashboards.MultiSelect_Source

// DashboardMultiSelectSelectionAll is a type for data for charts.
type DashboardMultiSelectSelectionAll = dashboards.MultiSelect_Selection_All

// DashboardMultiSelectAllSelection is a type for data for charts.
type DashboardMultiSelectAllSelection = dashboards.MultiSelect_Selection_AllSelection

// DashboardMultiSelectSelectionList is a type for data for charts.
type DashboardMultiSelectSelectionList = dashboards.MultiSelect_Selection_List

// DashboardMultiSelectListSelection is a type for data for charts.
type DashboardMultiSelectListSelection = dashboards.MultiSelect_Selection_ListSelection

// DashboardMultiSelectSelection is a type for data for charts.
type DashboardMultiSelectSelection = dashboards.MultiSelect_Selection

// DashboardVariableDefinition is a type for data for charts.
type DashboardVariableDefinition = dashboards.Variable_Definition

// DashboardVariableDefinitionConstant is a type for data for charts.
type DashboardVariableDefinitionConstant = dashboards.Variable_Definition_Constant

// DashboardVariableDefinitionMultiSelect is a type for data for charts.
type DashboardVariableDefinitionMultiSelect = dashboards.Variable_Definition_MultiSelect

// LineChartQuery is a type for data for charts.
type LineChartQuery = dashboards.LineChart_Query

// LineChartQueryDefinition is a type for data for charts.
type LineChartQueryDefinition = dashboards.LineChart_QueryDefinition

// LineChartResolution is a type for data for charts.
type LineChartResolution = dashboards.LineChart_Resolution

// LineChartQueryLogs is a type for data for charts.
type LineChartQueryLogs = dashboards.LineChart_Query_Logs

// LineChartLogsQuery is a type for data for charts.
type LineChartLogsQuery = dashboards.LineChart_LogsQuery

// LineChartQueryDataprime is a type for data for charts.
type LineChartQueryDataprime = dashboards.LineChart_Query_Dataprime

// LineChartQueryMetrics is a type for data for charts.
type LineChartQueryMetrics = dashboards.LineChart_Query_Metrics

// LineChartMetricsQuery is a type for data for charts.
type LineChartMetricsQuery = dashboards.LineChart_MetricsQuery

// LineChartQuerySpans is a type for data for charts.
type LineChartQuerySpans = dashboards.LineChart_Query_Spans

// LineChartSpansQuery is a type for data for charts.
type LineChartSpansQuery = dashboards.LineChart_SpansQuery

// DashboardDataTable is a type for dashboard charts.
type DashboardDataTable = dashboards.DataTable

// DashboardDataTableQuery is a type for dashboard charts.
type DashboardDataTableQuery = dashboards.DataTable_Query

// DashboardDataTableQueryDataprime is a type for dashboard charts.
type DashboardDataTableQueryDataprime = dashboards.DataTable_Query_Dataprime

// DashboardDataTableDataprimeQuery is a type for dashboard charts.
type DashboardDataTableDataprimeQuery = dashboards.DataTable_DataprimeQuery

// DashboardDataTableQueryMetrics is a type for dashboard charts.
type DashboardDataTableQueryMetrics = dashboards.DataTable_Query_Metrics

// DashboardDataTableMetricsQuery is a type for dashboard charts.
type DashboardDataTableMetricsQuery = dashboards.DataTable_MetricsQuery

// DashboardDataTableQueryLogs is a type for dashboard charts.
type DashboardDataTableQueryLogs = dashboards.DataTable_Query_Logs

// DashboardDataTableLogsQuery is a type for dashboard charts.
type DashboardDataTableLogsQuery = dashboards.DataTable_LogsQuery

// DashboardDataTableLogsQueryGrouping is a type for dashboard charts.
type DashboardDataTableLogsQueryGrouping = dashboards.DataTable_LogsQuery_Grouping

// DashboardDataTableLogsQueryAggregation is a type for dashboard charts.
type DashboardDataTableLogsQueryAggregation = dashboards.DataTable_LogsQuery_Aggregation

// DashboardDataTableQuerySpans is a type for dashboard charts.
type DashboardDataTableQuerySpans = dashboards.DataTable_Query_Spans

// DashboardDataTableSpansQuery is a type for dashboard charts.
type DashboardDataTableSpansQuery = dashboards.DataTable_SpansQuery

// DashboardDataTableSpansQueryGrouping is a type for dashboard charts.
type DashboardDataTableSpansQueryGrouping = dashboards.DataTable_SpansQuery_Grouping

// DashboardDataTableSpansQueryAggregation is a type for dashboard charts.
type DashboardDataTableSpansQueryAggregation = dashboards.DataTable_SpansQuery_Aggregation

// DashboardDataTableColumn is a type for dashboard charts.
type DashboardDataTableColumn = dashboards.DataTable_Column

// DashboardRelativeTimeFrame is a type for dashboard charts.
type DashboardRelativeTimeFrame = dashboards.Dashboard_RelativeTimeFrame

// DashboardAbsoluteTimeFrame is a type for dashboard charts.
type DashboardAbsoluteTimeFrame = dashboards.Dashboard_AbsoluteTimeFrame

// DashboardTimeFrame is a type for dashboard charts.
type DashboardTimeFrame = dashboards.TimeFrame

// DashboardOrderingField is a type for dashboard charts.
type DashboardOrderingField = dashboards.OrderingField

const dashboardsFeatureGroupID = "dashboards"

// Create Creates a new dashboard.
func (d DashboardsClient) Create(ctx context.Context, req *CreateDashboardRequest) (*dashboards.CreateDashboardResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardsServiceClient(conn)

	response, err := client.CreateDashboard(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, CreateDashboardRPC, dashboardsFeatureGroupID)
	}
	return response, nil
}

// Get gets a dashboard.
func (d DashboardsClient) Get(ctx context.Context, req *GetDashboardRequest) (*dashboards.GetDashboardResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardsServiceClient(conn)

	response, err := client.GetDashboard(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, GetDashboardRPC, dashboardsFeatureGroupID)
	}
	return response, nil
}

// List lists all dashboards.
func (d DashboardsClient) List(ctx context.Context) (*dashboards.GetDashboardCatalogResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetUserLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardCatalogServiceClient(conn)

	response, err := client.GetDashboardCatalog(callProperties.Ctx, &dashboards.GetDashboardCatalogRequest{}, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, dashboards.DashboardCatalogService_GetDashboardCatalog_FullMethodName, dashboardsFeatureGroupID)
	}
	return response, nil
}

// Replace replaces a dashboard.
func (d DashboardsClient) Replace(ctx context.Context, req *ReplaceDashboardRequest) (*dashboards.ReplaceDashboardResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardsServiceClient(conn)

	return client.ReplaceDashboard(callProperties.Ctx, req, callProperties.CallOptions...)
}

// Delete deletes a dashboard.
func (d DashboardsClient) Delete(ctx context.Context, req *DeleteDashboardRequest) (*dashboards.DeleteDashboardResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardsServiceClient(conn)

	response, err := client.DeleteDashboard(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, DeleteDashboardRPC, dashboardsFeatureGroupID)
	}
	return response, nil
}

// Pin pins a dashboard.
func (d DashboardsClient) Pin(ctx context.Context, req *PinDashboardRequest) (*dashboards.PinDashboardResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetUserLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardsServiceClient(conn)

	response, err := client.PinDashboard(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, PinDashboardRPC, dashboardsFeatureGroupID)
	}
	return response, nil
}

// Unpin unpins a dashboard.
func (d DashboardsClient) Unpin(ctx context.Context, req *UnpinDashboardRequest) (*dashboards.UnpinDashboardResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetUserLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardsServiceClient(conn)

	response, err := client.UnpinDashboard(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, UnpinDashboardRPC, dashboardsFeatureGroupID)
	}
	return response, nil
}

// AssignToFolder assigns a dashboard to a folder.
func (d DashboardsClient) AssignToFolder(ctx context.Context, req *dashboards.AssignDashboardFolderRequest) (*dashboards.AssignDashboardFolderResponse, error) {
	callProperties, err := d.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dashboards.NewDashboardsServiceClient(conn)

	response, err := client.AssignDashboardFolder(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, AssignDashboardFolderRPC, dashboardsFeatureGroupID)
	}
	return response, nil
}

// NewDashboardsClient creates a new DashboardsClient.
func NewDashboardsClient(c *CallPropertiesCreator) *DashboardsClient {
	return &DashboardsClient{callPropertiesCreator: c}
}
