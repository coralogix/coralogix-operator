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

package coralogix

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

func ReverseMap[K, V comparable](m map[K]V) map[V]K {
	n := make(map[V]K)
	for k, v := range m {
		n[v] = k
	}
	return n
}

func FloatToQuantity(n float64) resource.Quantity {
	return resource.MustParse(fmt.Sprintf("%f", n))
}

func QuantitiesToFloats32(arr []resource.Quantity) []float32 {
	result := make([]float32, 0, len(arr))
	for _, q := range arr {
		result = append(result, float32(q.AsApproximateFloat64()))
	}
	return result
}
