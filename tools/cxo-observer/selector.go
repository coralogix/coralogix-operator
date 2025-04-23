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
	"context"
	"fmt"

	"github.com/coralogix/coralogix-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Selector struct {
	LabelSelector     labels.Selector
	NamespaceSelector labels.Selector
}

func (s Selector) Matches(resourceLabels map[string]string, namespace string) bool {
	if labelSelector := s.LabelSelector; labelSelector != nil {
		if !labelSelector.Matches(labels.Set(resourceLabels)) {
			return false
		}
	}
	match, err := isNamespaceExistAndMatch(s.NamespaceSelector, namespace)
	if err != nil {
		log.Error(err, "Error getting namespace")
		return false
	}

	return match
}

func isNamespaceExistAndMatch(selector labels.Selector, namespace string) (bool, error) {
	ns := &corev1.Namespace{}
	if err := config.GetClient().Get(context.Background(), client.ObjectKey{Name: namespace}, ns); err != nil {
		return false, fmt.Errorf("error getting namespace: %w", err)
	}
	var labels labels.Set
	labels = ns.Labels
	return selector.Matches(labels), nil
}

func parseSelector(labelSelectorStr, namespaceSelectorStr string) (Selector, error) {
	labelSelector, err := labels.Parse(labelSelectorStr)
	if err != nil {
		return Selector{}, fmt.Errorf("error parsing label selector: %w", err)
	}

	namespaceSelector, err := labels.Parse(namespaceSelectorStr)
	if err != nil {
		return Selector{}, fmt.Errorf("error parsing namespace selector: %w", err)
	}

	return Selector{
		LabelSelector:     labelSelector,
		NamespaceSelector: namespaceSelector,
	}, nil
}
