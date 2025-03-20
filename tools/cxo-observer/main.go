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
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var log = ctrl.Log.WithName("cxo-observer")

func main() {
	// and what about logging?
	// panic or os.Exit(1)
	initConfig(log)

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		log.Error(err, "Failed to create output directory")
		os.Exit(1)
	}

	ctx := context.Background()
	for gvk, requestedResource := range cfg.RequestedResources {
		if err := collectResource(ctx, gvk, requestedResource); err != nil {
			log.Error(err, "Failed to dump resource", "gvk", gvk)
			os.Exit(1)
		}
	}
}

func collectResource(ctx context.Context, gvk schema.GroupVersionKind, requestedResource RequestedResource) error {
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)
	for _, namespace := range requestedResource.Namespaces {
		for _, name := range requestedResource.Names {
			objKey := client.ObjectKey{Namespace: namespace, Name: name}
			if err := cfg.Client.Get(ctx, objKey, resource); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}
			data, err := yaml.Marshal(resource)
			if err != nil {
				return err
			}

			//mkdir for the namespace and then gvk kind
			namespaceDir := filepath.Join(cfg.OutputDir, namespace)
			if err := os.MkdirAll(namespaceDir, 0755); err != nil {
				return err
			}

			gvkDir := filepath.Join(namespaceDir, strings.ToLower(gvk.Kind))
			if err := os.MkdirAll(gvkDir, 0755); err != nil {
				return err
			}

			filePath := filepath.Join(gvkDir, name+".yaml")
			if err := os.WriteFile(filePath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
