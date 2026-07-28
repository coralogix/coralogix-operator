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

import "reflect"

// DiscardAdditionalProperties restores protojson's historical DiscardUnknown behavior.
// The generated models collect unknown JSON keys into AdditionalProperties and re-emit
// them on marshal, which the API rejects with a 400.
//
// Order matters: run this only after Unmarshal has promoted the snake_case spellings
// into typed fields, or it deletes every field authored in the protobuf spelling.
func DiscardAdditionalProperties(value any) {
	discardAdditionalPropertiesValue(reflect.ValueOf(value))
}

func discardAdditionalPropertiesValue(value reflect.Value) {
	if !value.IsValid() {
		return
	}

	kind := value.Kind()
	if kind == reflect.Interface || kind == reflect.Pointer {
		if !value.IsNil() {
			discardAdditionalPropertiesValue(value.Elem())
		}
		return
	}
	if kind == reflect.Struct {
		if value.Type().PkgPath() != dashboardServicePackage {
			return
		}
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if value.Type().Field(i).Name == "AdditionalProperties" && field.CanSet() {
				field.SetZero()
				continue
			}
			discardAdditionalPropertiesValue(field)
		}
		return
	}
	if kind == reflect.Slice || kind == reflect.Array {
		for i := 0; i < value.Len(); i++ {
			discardAdditionalPropertiesValue(value.Index(i))
		}
		return
	}
	if kind == reflect.Map {
		if value.Type().Elem().Kind() == reflect.Interface {
			discardFreeFormObjectKeys(value)
			return
		}
		for _, key := range value.MapKeys() {
			discardAdditionalPropertiesValue(value.MapIndex(key))
		}
	}
}

// discardFreeFormObjectKeys empties a map[string]interface{} field. The generator emits
// that type for a message with no fields - the marker arms of a oneOf, such as
// {"off": {}} or {"count": {}} - so it has no AdditionalProperties to clear and every key
// it holds is unknown. The keys are deleted in place rather than the map being replaced:
// the map need not be settable, and a nil map is omitted by the generated MarshalJSON,
// which would silently drop the arm from the request instead of just its unknown keys.
func discardFreeFormObjectKeys(value reflect.Value) {
	for _, key := range value.MapKeys() {
		value.SetMapIndex(key, reflect.Value{})
	}
}
