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
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"
)

// unmarshal_test.go pins these behaviours on hand-written samples. The tests here assert
// they hold for every model reachable from Dashboard, so they keep covering the whole
// schema as the pinned SDK is bumped.

// Both spellings are injected because an unknown key that looks like a protobuf field
// name takes a different route through normalizeProtoObjectFieldNames.
const (
	unknownCamelKey = "zzUnknownKey"
	unknownSnakeKey = "zz_unknown_key"
	unknownValue    = "must-not-reach-the-api"
)

// The protobuf snake_case spelling must decode to exactly what camelCase decodes to.
func TestGeneratedModelsDecodeIdenticallyInBothSpellings(t *testing.T) {
	for _, name := range sortedModelNames(t) {
		modelType := reachableModels(t)[name]
		t.Run(name, func(t *testing.T) {
			camel, err := decodeGenerated(modelType, generatePayload(modelType, false, false))
			require.NoError(t, err, "camelCase payload must decode")

			snake, err := decodeGenerated(modelType, generatePayload(modelType, true, false))
			require.NoError(t, err, "protobuf snake_case payload must decode")

			require.Equal(t, camel, snake,
				"the same document authored in protobuf snake_case decoded differently")
		})
	}
}

// An unknown key that survives to the request body is rejected by the API with a 400.
func TestGeneratedModelsNeverReEmitUnknownKeys(t *testing.T) {
	for _, name := range sortedModelNames(t) {
		modelType := reachableModels(t)[name]
		t.Run(name, func(t *testing.T) {
			for _, useSnake := range []bool{false, true} {
				target, err := decodeGeneratedInto(modelType, generatePayload(modelType, useSnake, true))
				require.NoError(t, err, "payload with unknown keys must still decode (snake=%v)", useSnake)

				DiscardAdditionalProperties(target)
				encoded, err := json.Marshal(target)
				require.NoError(t, err)

				require.NotContains(t, string(encoded), unknownValue,
					"unknown key survived DiscardAdditionalProperties (snake=%v)", useSnake)
			}
		})
	}
}

// An enum field the walk cannot reach would decode without complaint, silently dropping
// the value. Each is authored out-of-spec in the protobuf spelling to prove it is reached.
func TestGeneratedEnumFieldsAreAllValidated(t *testing.T) {
	positions := enumFieldPositions(t)
	require.NotEmpty(t, positions, "expected to find enum fields reachable from Dashboard")

	for _, position := range positions {
		t.Run(position.model+"."+position.jsonName, func(t *testing.T) {
			value, expectedPath := position.value()
			encoded, err := json.Marshal(map[string]any{
				protobufJSONFieldName(position.jsonName): value,
			})
			require.NoError(t, err)

			target := reflect.New(position.modelType).Interface()
			err = Unmarshal(encoded, target)
			require.Error(t, err, "out-of-spec enum value was accepted")
			require.Contains(t, err.Error(), expectedPath,
				"error must name the offending field's JSON path")
			require.Contains(t, err.Error(), position.enumName,
				"error must name the enum type")
		})
	}
}

// generatePayload builds a JSON document for modelType with every field populated. Enum
// fields are omitted since a valid value cannot be derived from the type alone;
// TestGeneratedEnumFieldsAreAllValidated covers them.
func generatePayload(modelType reflect.Type, useSnake, injectUnknown bool) any {
	return generateValue(modelType, useSnake, injectUnknown)
}

func generateValue(t reflect.Type, useSnake, injectUnknown bool) any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeOf(time.Time{}) {
		return "2026-07-28T10:00:00Z"
	}

	kind := t.Kind()
	if kind == reflect.Struct {
		// Nothing outside the package carries field names to normalize.
		if t.PkgPath() != dashboardServicePackage {
			return map[string]any{}
		}
		object := map[string]any{}
		if injectUnknown {
			object[unknownCamelKey] = unknownValue
			object[unknownSnakeKey] = unknownValue
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonName := jsonFieldName(field)
			if enumType, _ := enumLeafType(field.Type); jsonName == "" || enumType != nil {
				continue
			}
			key := jsonName
			if useSnake {
				key = protobufJSONFieldName(jsonName)
			}
			object[key] = generateValue(field.Type, useSnake, injectUnknown)
		}
		return object
	}
	if kind == reflect.Slice {
		return []any{generateValue(t.Elem(), useSnake, injectUnknown)}
	}
	if kind == reflect.Map {
		if t.Elem().Kind() == reflect.Interface {
			// An empty-message oneOf arm, e.g. {"count": {}}. It has no fields, so any
			// key authored inside it is an unknown one and belongs in the injection.
			object := map[string]any{}
			if injectUnknown {
				object[unknownCamelKey] = unknownValue
				object[unknownSnakeKey] = unknownValue
			}
			return object
		}
		return map[string]any{"key": generateValue(t.Elem(), useSnake, injectUnknown)}
	}
	if kind == reflect.Interface {
		return map[string]any{}
	}
	if kind == reflect.String {
		return "generated"
	}
	if kind == reflect.Bool {
		return true
	}
	if kind == reflect.Int || kind == reflect.Int32 || kind == reflect.Int64 {
		return 1
	}
	if kind == reflect.Float32 || kind == reflect.Float64 {
		return 1.5
	}
	return nil
}

var oneOfGuard = regexp.MustCompile(`at most one of \[([^\]]*)\] may be set`)

// A fully populated document sets every arm of every oneOf group, which the models
// reject. Reading the group off the model's own error avoids restating the schema here.
func decodeGeneratedInto(modelType reflect.Type, payload any) (any, error) {
	for attempt := 0; ; attempt++ {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		target := reflect.New(modelType).Interface()
		err = Unmarshal(encoded, target)
		if err == nil {
			return target, nil
		}
		match := oneOfGuard.FindStringSubmatch(err.Error())
		if match == nil || attempt > 64 {
			return nil, err
		}
		group := strings.Split(match[1], ",")
		for i := range group {
			group[i] = strings.TrimSpace(group[i])
		}
		if !pruneOneOfGroup(payload, group) {
			return nil, err
		}
	}
}

func decodeGenerated(modelType reflect.Type, payload any) (map[string]any, error) {
	target, err := decodeGeneratedInto(modelType, payload)
	if err != nil {
		return nil, err
	}
	DiscardAdditionalProperties(target)
	encoded, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// pruneOneOfGroup keeps only the first present arm of group, in either spelling. It
// reports whether anything was removed, so an unrepairable decode fails instead of looping.
func pruneOneOfGroup(payload any, group []string) bool {
	pruned := false
	switch value := payload.(type) {
	case map[string]any:
		present := make([]string, 0, len(group))
		for _, arm := range group {
			for _, key := range []string{arm, protobufJSONFieldName(arm)} {
				if _, ok := value[key]; ok {
					present = append(present, key)
					break
				}
			}
		}
		for _, key := range present[min(1, len(present)):] {
			delete(value, key)
			pruned = true
		}
		for _, nested := range value {
			if pruneOneOfGroup(nested, group) {
				pruned = true
			}
		}
	case []any:
		for _, item := range value {
			if pruneOneOfGroup(item, group) {
				pruned = true
			}
		}
	}
	return pruned
}

type enumPosition struct {
	model     string
	modelType reflect.Type
	jsonName  string
	enumName  string
	inSlice   bool
}

// value returns the out-of-spec document and the JSON path the error must name.
func (p enumPosition) value() (any, string) {
	if p.inSlice {
		return []any{"NOT_A_REAL_ENUM_VALUE"}, p.jsonName + "[0]"
	}
	return "NOT_A_REAL_ENUM_VALUE", p.jsonName
}

// enumFieldPositions lists every enum-typed field on every reachable model.
func enumFieldPositions(t *testing.T) []enumPosition {
	t.Helper()
	var positions []enumPosition
	for _, name := range sortedModelNames(t) {
		modelType := reachableModels(t)[name]
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			jsonName := jsonFieldName(field)
			enumType, inSlice := enumLeafType(field.Type)
			if jsonName == "" || enumType == nil {
				continue
			}
			positions = append(positions, enumPosition{
				model:     name,
				modelType: modelType,
				jsonName:  jsonName,
				enumName:  enumType.Name(),
				inSlice:   inSlice,
			})
		}
	}
	return positions
}

// enumLeafType reports the enum a field holds, looking through pointers and slices, and
// whether it went through a slice. Returns nil for fields holding no enum.
func enumLeafType(t reflect.Type) (reflect.Type, bool) {
	inSlice := false
	for {
		kind := t.Kind()
		if kind == reflect.Pointer {
			t = t.Elem()
			continue
		}
		if kind == reflect.Slice {
			inSlice = true
			t = t.Elem()
			continue
		}
		if isEnumType(t) {
			return t, inSlice
		}
		return nil, false
	}
}

var (
	cachedModels map[string]reflect.Type
	cachedNames  []string
)

// The model graph is acyclic, so the walks in this file need no depth limit.
func reachableModels(t *testing.T) map[string]reflect.Type {
	t.Helper()
	if cachedModels == nil {
		cachedModels = map[string]reflect.Type{}
		collectModels(reflect.TypeOf(dashboards.Dashboard{}), cachedModels, map[reflect.Type]bool{})
	}
	return cachedModels
}

func sortedModelNames(t *testing.T) []string {
	t.Helper()
	if cachedNames == nil {
		for name := range reachableModels(t) {
			cachedNames = append(cachedNames, name)
		}
		sort.Strings(cachedNames)
	}
	return cachedNames
}

func collectModels(t reflect.Type, out map[string]reflect.Type, seen map[reflect.Type]bool) {
	t = derefType(t)
	kind := t.Kind()
	if kind == reflect.Struct {
		if t.PkgPath() != dashboardServicePackage || seen[t] {
			return
		}
		seen[t] = true
		out[t.Name()] = t
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() || field.Name == "AdditionalProperties" {
				continue
			}
			collectModels(field.Type, out, seen)
		}
		return
	}
	if kind == reflect.Slice || kind == reflect.Map {
		collectModels(t.Elem(), out, seen)
	}
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// jsonFieldName returns the field's JSON name, or "" for fields the decoder ignores.
func jsonFieldName(field reflect.StructField) string {
	if !field.IsExported() {
		return ""
	}
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "-" {
		return ""
	}
	return name
}
