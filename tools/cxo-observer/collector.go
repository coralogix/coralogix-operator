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
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/coralogix/coralogix-operator/internal/utils"
)

func collectOperatorResource(ctx context.Context, log logr.Logger, gvk schema.GroupVersionKind, name string) error {
	log.V(1).Info("Collecting resource", "kind", gvk.Kind, "name", name)

	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)
	if err := cfg.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: cfg.ChartNamespace}, resource); err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	dir := filepath.Join("temp", "cxo-observer", "operator-resources")
	fileName := strings.ToLower(resource.GetKind())
	if gvk.Kind == crdKind {
		dir = filepath.Join("temp", "cxo-observer", "operator-resources", "crds")
		fileName = strings.ToLower(resource.GetName())
	}

	if err := dumpResource(resource, dir, fileName); err != nil {
		return fmt.Errorf("failed to dump resource: %w", err)
	}

	return nil
}

func collectCRsInNamespace(ctx context.Context, log logr.Logger, ns string) error {
	if ns == "" {
		log.Info("Collecting custom resources in all namespaces")
	} else {
		log.Info("Collecting custom resources in namespace", "namespace", ns)
	}
	gvks := utils.GetGVKs(scheme)
	for _, gvk := range gvks {
		if err := collectGvkCRs(ctx, log, ns, gvk); err != nil {
			return fmt.Errorf("failed to collect %s CRs in namespace %s: %w", gvk.Kind, ns, err)
		}
	}
	return nil
}

func collectGvkCRs(ctx context.Context, log logr.Logger, ns string, gvk schema.GroupVersionKind) error {
	log.V(1).Info("Collecting CRs", "kind", gvk.Kind, "namespace", ns)

	resources := &unstructured.UnstructuredList{}
	resources.SetGroupVersionKind(gvk)
	labelSelector := cfg.Selector.LabelSelector
	if gvk.Kind == utils.PrometheusRuleKind {
		req, err := labels.NewRequirement(utils.TrackPrometheusRuleAlertsLabelKey, selection.Equals, []string{"true"})
		if err != nil {
			return fmt.Errorf("failed to create label requirement: %w", err)
		}
		labelSelector = labelSelector.Add(*req)
	}

	listOpts := &client.ListOptions{LabelSelector: labelSelector}
	if ns != "" {
		listOpts.Namespace = ns
	}

	if err := cfg.Client.List(ctx, resources, listOpts); err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	for _, resource := range resources.Items {
		path := filepath.Join(
			"temp", "cxo-observer", "custom-resources",
			resource.GetNamespace(),
			strings.ToLower(gvk.Group),
			strings.ToLower(gvk.Version),
			strings.ToLower(gvk.Kind))
		if err := dumpResource(&resource, path, resource.GetName()); err != nil {
			return fmt.Errorf("failed to dump resource: %w", err)
		}
	}

	return nil
}

func dumpResource(resource *unstructured.Unstructured, dir, fileName string) error {
	data, err := yaml.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	filePath := filepath.Join(dir, fileName+".yaml")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}
