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
	"log"
	"strconv"

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
		match, err := isNamespaceExistAndMatch(s.NamespaceSelector, namespace)
		if err != nil {
			log.Printf("Error getting namespace: %v", err)
			return false
		}
		return match
	}

	return true
}

func isNamespaceExistAndMatch(selector labels.Selector, namespace string) (bool, error) {
	ns := &corev1.Namespace{}
	if err := GetClient().Get(context.Background(), client.ObjectKey{Name: namespace}, ns); err != nil {
		return false, fmt.Errorf("error getting namespace: %w", err)
	}
	return selector.Matches(labels.Set(ns.Labels)), nil
}

func parseSelector(labelSelectorStr, namespaceSelectorStr string) (Selector, error) {
	var err error

	// Unquote the flags to remove extra escaping
	labelSelectorStr, err = strconv.Unquote(labelSelectorStr)
	if err != nil {
		// not fatal; maybe it wasn't quoted
	}
	namespaceSelectorStr, err = strconv.Unquote(namespaceSelectorStr)
	if err != nil {
		// not fatal; maybe it wasn't quoted
	}

	var labelSel metav1.LabelSelector
	if labelSelectorStr != "" {
		if err := json.Unmarshal([]byte(labelSelectorStr), &labelSel); err != nil {
			return Selector{}, fmt.Errorf("failed to parse labelSelector %s: %w", labelSelectorStr, err)
		}
	}
	var namespaceSel metav1.LabelSelector
	if namespaceSelectorStr != "" {
		if err := json.Unmarshal([]byte(namespaceSelectorStr), &namespaceSel); err != nil {
			return Selector{}, fmt.Errorf("failed to parse namespaceSelector %s: %w", namespaceSelectorStr, err)
		}
	}

	labelSelector, err := metav1.LabelSelectorAsSelector(&labelSel)
	if err != nil {
		return Selector{}, fmt.Errorf("failed to convert labelSelector %s: %w", labelSelectorStr, err)
	}
	namespaceSelector, err := metav1.LabelSelectorAsSelector(&namespaceSel)
	if err != nil {
		return Selector{}, fmt.Errorf("failed to convert namespaceSelector %s: %w", namespaceSelectorStr, err)
	}

	return Selector{
		LabelSelector:     labelSelector,
		NamespaceSelector: namespaceSelector,
	}, nil
}
