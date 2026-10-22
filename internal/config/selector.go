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

package config

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type Selector struct {
	LabelSelector     labels.Selector
	NamespaceSelector labels.Selector
}

func (s Selector) Predicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return s.Matches(e.Object.GetLabels(), e.Object.GetNamespace())
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return s.Matches(e.ObjectNew.GetLabels(), e.ObjectNew.GetNamespace()) ||
				s.Matches(e.ObjectOld.GetLabels(), e.ObjectOld.GetNamespace())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return s.Matches(e.Object.GetLabels(), e.Object.GetNamespace())
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return s.Matches(e.Object.GetLabels(), e.Object.GetNamespace())
		},
	}
}

func (s Selector) Matches(resourceLabels map[string]string, namespace string) bool {
	if labelSelector := s.LabelSelector; labelSelector != nil {
		if !labelSelector.Matches(labels.Set(resourceLabels)) {
			return false
		}
	}

	if namespaceSelector := s.NamespaceSelector; namespaceSelector != nil {
		match, err := isNamespaceMatch(s.NamespaceSelector, namespace)
		if err != nil {
			return false
		}
		return match
	}

	return true
}

func isNamespaceMatch(selector labels.Selector, namespace string) (bool, error) {
	ns := &corev1.Namespace{}
	if err := GetClient().Get(context.Background(), client.ObjectKey{Name: namespace}, ns); err != nil {
		return false, fmt.Errorf("error getting namespace: %w", err)
	}
	return selector.Matches(labels.Set(ns.Labels)), nil
}

func parseSelector(labelSelectorStr, namespaceSelectorStr string) (*Selector, error) {
	labelSelector, err := stringToLabelSelector(labelSelectorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}

	namespaceSelector, err := stringToLabelSelector(namespaceSelectorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse namespace selector: %w", err)
	}

	return &Selector{
		LabelSelector:     labelSelector,
		NamespaceSelector: namespaceSelector,
	}, nil
}

func stringToLabelSelector(selectorStr string) (labels.Selector, error) {
	if selectorStr == "" {
		return labels.Everything(), nil
	}
	var labelSelector metav1.LabelSelector
	if err := json.Unmarshal([]byte(selectorStr), &labelSelector); err != nil {
		return nil, fmt.Errorf("failed to unmarshal label selector JSON: %s, %w", selectorStr, err)
	}
	return metav1.LabelSelectorAsSelector(&labelSelector)
}
