package utils

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Diff struct {
	Name            string
	Desired, Actual interface{}
}

func ReverseMap[K, V comparable](m map[K]V) map[V]K {
	n := make(map[V]K)
	for k, v := range m {
		n[v] = k
	}
	return n
}

func SlicesWithUniqueValuesEqual[V comparable](a, b []V) bool {
	if len(a) != len(b) {
		return false
	}

	valuesSet := make(map[V]bool, len(a))
	for _, _a := range a {
		valuesSet[_a] = true
	}

	for _, _b := range b {
		if !valuesSet[_b] {
			return false
		}
	}

	return true
}

func GetKeys[K, V comparable](m map[K]V) []K {
	result := make([]K, 0)
	for k := range m {
		result = append(result, k)
	}
	return result
}

func StringSliceToWrappedStringSlice(arr []string) []*wrapperspb.StringValue {
	result := make([]*wrapperspb.StringValue, 0, len(arr))
	for _, s := range arr {
		result = append(result, wrapperspb.String(s))
	}
	return result
}

func WrappedStringSliceToStringSlice(arr []*wrapperspb.StringValue) []string {
	result := make([]string, 0, len(arr))
	for _, s := range arr {
		result = append(result, s.Value)
	}
	return result
}

func FloatToQuantity(n float64) resource.Quantity {
	return resource.MustParse(fmt.Sprintf("%f", n))
}

func PointerToString(o any) string {
	if o == nil {
		return "<nil>"
	}

	val := reflect.ValueOf(o)
	if val.Kind() == reflect.Interface {
		elm := val.Elem()
		if elm.Kind() == reflect.Ptr && !elm.IsNil() && elm.Elem().Kind() == reflect.Ptr {
			val = elm
		}
	}
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return val.String()
	}

	result := ""
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		if valueField.Kind() == reflect.Interface && !valueField.IsNil() {
			elm := valueField.Elem()
			if elm.Kind() == reflect.Ptr && !elm.IsNil() && elm.Elem().Kind() == reflect.Ptr {
				valueField = elm
			}
		}
		if valueField.Kind() == reflect.Ptr {
			valueField = valueField.Elem()
		}

		if valueField.Kind() == reflect.Struct {
			result += PointerToString(valueField.Interface())
		} else if valueField.Kind() == reflect.Ptr && valueField.IsNil() {
			result += fmt.Sprintf("Field Name: %s,\t Field Value: %v\n", typeField.Name, "<nil>")
		} else if valueField.IsZero() {
			result += fmt.Sprintf("Field Name: %s,\t Field Value: %v\n", typeField.Name, "<empty>")
		} else {
			result += fmt.Sprintf("Field Name: %s,\t Field Value: %v\n", typeField.Name, valueField.Interface())
		}
	}

	return result
}
