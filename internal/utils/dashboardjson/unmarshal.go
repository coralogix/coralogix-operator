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

// Package dashboardjson decodes the opaque dashboard JSON carried in a Dashboard CR's
// spec into the generated dashboard_service models.
//
// spec.json is declared as a plain string, so the CRD validates nothing and this package
// is effectively the dashboard CRD's schema. It keeps accepting what protojson accepted
// before the move to REST: both the camelCase OpenAPI and snake_case protobuf spellings,
// and bare JSON numbers for int64/uint64 fields modeled as strings.
//
// Unknown keys are tolerated in two steps - Unmarshal leaves them in place,
// DiscardAdditionalProperties strips them before the value is sent. Enum values the
// pinned SDK does not know are rejected rather than dropped.
package dashboardjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"

	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"
)

var dashboardServicePackage = reflect.TypeOf(dashboards.Dashboard{}).PkgPath()

// Unmarshal decodes a dashboard JSON document into target, which must be a non-nil
// pointer to a dashboard_service model.
func Unmarshal(data []byte, target any) error {
	targetType := reflect.TypeOf(target)
	if targetType == nil || targetType.Kind() != reflect.Pointer || reflect.ValueOf(target).IsNil() {
		return fmt.Errorf("dashboard JSON target must be a non-nil pointer")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid JSON after top-level value")
		}
		return err
	}

	normalized, err := normalizeProtoFieldNames(raw, targetType, "")
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal normalized dashboard JSON: %w", err)
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		return fmt.Errorf("unmarshal normalized dashboard JSON: %w", err)
	}
	return nil
}

// normalizeProtoFieldNames rewrites raw so that it matches what the generated models
// expect, guided by the target type. path is the JSON path walked so far and is only
// used to locate errors inside what is often a several-thousand-line document.
func normalizeProtoFieldNames(raw any, targetType reflect.Type, path string) (any, error) {
	for targetType.Kind() == reflect.Pointer {
		targetType = targetType.Elem()
	}

	kind := targetType.Kind()
	if kind == reflect.Struct {
		object, ok := raw.(map[string]any)
		// Only models are descended into, so unknown subtrees and the
		// map[string]interface{} blobs the generator emits are left untouched.
		if !ok || targetType.PkgPath() != dashboardServicePackage {
			return raw, nil
		}
		return normalizeProtoObjectFieldNames(object, targetType, path)
	}
	if kind == reflect.Slice || kind == reflect.Array {
		return normalizeProtoList(raw, targetType, path)
	}
	if kind == reflect.Map {
		return normalizeProtoMap(raw, targetType, path)
	}
	if kind == reflect.String {
		return normalizeProtoString(raw, targetType, path)
	}
	return raw, nil
}

func normalizeProtoList(raw any, targetType reflect.Type, path string) (any, error) {
	items, ok := raw.([]any)
	if !ok {
		return raw, nil
	}
	normalized := make([]any, len(items))
	for i, item := range items {
		value, err := normalizeProtoFieldNames(item, targetType.Elem(), fmt.Sprintf("%s[%d]", path, i))
		if err != nil {
			return nil, err
		}
		normalized[i] = value
	}
	return normalized, nil
}

func normalizeProtoMap(raw any, targetType reflect.Type, path string) (any, error) {
	object, ok := raw.(map[string]any)
	if !ok || targetType.Key().Kind() != reflect.String {
		return raw, nil
	}
	normalized := make(map[string]any, len(object))
	for key, value := range object {
		normalizedValue, err := normalizeProtoFieldNames(value, targetType.Elem(), childPath(path, key))
		if err != nil {
			return nil, err
		}
		normalized[key] = normalizedValue
	}
	return normalized, nil
}

func normalizeProtoString(raw any, targetType reflect.Type, path string) (any, error) {
	if isEnumType(targetType) {
		return validateEnumValue(raw, targetType, path)
	}
	// protobuf int64/uint64 fields (e.g. seriesCountLimit) are modeled as Go strings
	// but commonly authored as bare JSON numbers.
	if number, ok := raw.(json.Number); ok {
		return number.String(), nil
	}
	return raw, nil
}

func normalizeProtoObjectFieldNames(object map[string]any, targetType reflect.Type, path string) (map[string]any, error) {
	normalized := make(map[string]any, len(object))
	consumed := make(map[string]struct{})

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonName == "" || jsonName == "-" {
			continue
		}

		value, found := object[jsonName]
		if found {
			consumed[jsonName] = struct{}{}
		}

		alias := protobufJSONFieldName(jsonName)
		if alias != jsonName {
			if aliasValue, aliasFound := object[alias]; aliasFound {
				// Preserve protojson's historical behavior when both spellings are
				// present: the protobuf spelling wins.
				value = aliasValue
				found = true
				consumed[alias] = struct{}{}
			}
		}

		if found {
			normalizedValue, err := normalizeProtoFieldNames(value, field.Type, childPath(path, jsonName))
			if err != nil {
				return nil, err
			}
			normalized[jsonName] = normalizedValue
		}
	}

	// Unknown keys are copied through rather than dropped here, so that
	// DiscardAdditionalProperties can strip them once the typed fields are in place.
	for key, value := range object {
		if _, ok := consumed[key]; !ok {
			normalized[key] = value
		}
	}
	return normalized, nil
}

func protobufJSONFieldName(jsonName string) string {
	var result strings.Builder
	for i, character := range jsonName {
		if unicode.IsUpper(character) {
			if i > 0 {
				result.WriteByte('_')
			}
			character = unicode.ToLower(character)
		}
		result.WriteRune(character)
	}
	return result.String()
}

func childPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

// validateEnumValue rejects the two enum spellings the generated models cannot decode.
// The models reject them too, but their error names no path - unhelpful in a document
// thousands of lines long. protojson discarded both silently, which renders a dashboard
// that differs from what was authored.
func validateEnumValue(raw any, enumType reflect.Type, path string) (any, error) {
	switch value := raw.(type) {
	case json.Number:
		// Rejected before normalizeProtoString's json.Number coercion turns it into a
		// string, which would misreport it as an unrecognized enum value.
		return nil, fmt.Errorf("%s: numeric enum values are not supported; use the string form of %s",
			path, enumType.Name())
	case string:
		if !isValidEnumValue(enumType, value) {
			return nil, fmt.Errorf("%s: %q is not a valid %s", path, value, enumType.Name())
		}
		return value, nil
	default:
		// Nulls and structurally wrong values are left to the model's own decoding.
		return raw, nil
	}
}

// isEnumType reports whether t is a generated enum model: a named string type in the
// dashboard_service package carrying an IsValid() bool method. If the generator stops
// emitting IsValid this degrades to plain-string handling rather than failing.
func isEnumType(t reflect.Type) bool {
	if t.Kind() != reflect.String || t.PkgPath() != dashboardServicePackage {
		return false
	}
	method, ok := t.MethodByName("IsValid")
	return ok && method.Type.NumIn() == 1 && method.Type.NumOut() == 1 && method.Type.Out(0).Kind() == reflect.Bool
}

// isValidEnumValue asks the generated model itself, which is the only authority on the
// accepted values: they are package-level variables, so the walk cannot read them.
func isValidEnumValue(enumType reflect.Type, value string) bool {
	enum := reflect.New(enumType).Elem()
	enum.SetString(value)
	results := enum.MethodByName("IsValid").Call(nil)
	return len(results) == 1 && results[0].Bool()
}
