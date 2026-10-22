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

package main

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
)

type Selector struct {
	LabelSelector     labels.Selector
	NamespaceSelector []string
}

func (s *Selector) Matches(resourceLabels map[string]string, namespace string) bool {
	return s.LabelSelector.Matches(labels.Set(resourceLabels)) && s.MatchesNamespace(namespace)
}

func (s *Selector) MatchesNamespace(namespace string) bool {
	if len(s.NamespaceSelector) == 0 {
		return true
	}
	for _, ns := range s.NamespaceSelector {
		if ns == namespace {
			return true
		}
	}
	return false
}

func parseSelector(labelSelector, namespaceSelector string) (*Selector, error) {
	parsedLabelSelector, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}

	return &Selector{
		LabelSelector:     parsedLabelSelector,
		NamespaceSelector: parseNamespaceSelector(namespaceSelector),
	}, nil
}

func parseNamespaceSelector(namespaceSelector string) []string {
	trimmed := strings.TrimSpace(namespaceSelector)
	if trimmed == "" {
		return []string{""} // empty string means "all namespaces"
	}

	var namespaceList []string
	for _, ns := range strings.Split(trimmed, ",") {
		ns = strings.TrimSpace(ns)
		if ns != "" {
			namespaceList = append(namespaceList, ns)
		}
	}

	return namespaceList
}
