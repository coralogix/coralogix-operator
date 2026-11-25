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

package utils

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGVKs(scheme *runtime.Scheme) []schema.GroupVersionKind {
	result := []schema.GroupVersionKind{
		{Group: MonitoringAPIGroup, Version: V1APIVersion, Kind: PrometheusRuleKind},
	}

	result = append(result, GetGVKsInVersion(V1alpha1APIVersion, scheme)...)
	result = append(result, GetGVKsInVersion(V1beta1APIVersion, scheme)...)
	return result
}

func GetGVKsInVersion(version string, scheme *runtime.Scheme) []schema.GroupVersionKind {
	var result []schema.GroupVersionKind

	groupVersion := schema.GroupVersion{Group: CoralogixAPIGroup, Version: version}
	knownTypes := scheme.KnownTypes(groupVersion)
	for kind := range knownTypes {
		// Skip v1alpha1 Alert since we pick it up from v1beta1
		if kind == AlertKind && version == V1alpha1APIVersion {
			continue
		}
		// skip List, Options and Event types. e.g. AlertList, ListOptions, WatchEvent
		if strings.HasSuffix(kind, "List") ||
			strings.HasSuffix(kind, "Options") ||
			strings.HasSuffix(kind, "Event") {
			continue
		}
		result = append(result, groupVersion.WithKind(kind))
	}

	return result
}
