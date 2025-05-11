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
	"os/exec"
	"path/filepath"
	"strings"

	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var (
	scheme = k8sruntime.NewScheme()
	log    = ctrl.Log.WithName("cxo-observer")
)

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme))
}

func main() {
	ctx := context.Background()
	initConfig(log)

	log.Info("Collecting resources that are part of the operator installation")
	log.V(1).Info("Collecting Deployment")
	depGvk := schema.GroupVersionKind{
		Group:   appsAPIGroup,
		Version: utils.V1APIVersion,
		Kind:    deploymentKind,
	}
	if err := collectOperatorResource(ctx, log, depGvk, cfg.ChartName); err != nil {
		log.Error(err, "Failed to collect Deployment")
	}

	log.V(1).Info("Collecting Service")
	svcGvk := schema.GroupVersionKind{
		Group:   coreAPIGroup,
		Version: utils.V1APIVersion,
		Kind:    serviceKind,
	}
	if err := collectOperatorResource(ctx, log, svcGvk, cfg.ChartName); err != nil {
		log.Error(err, "Failed to collect Service")
	}

	log.V(1).Info("Collecting ServiceAccount")
	saGvk := schema.GroupVersionKind{
		Group:   coreAPIGroup,
		Version: utils.V1APIVersion,
		Kind:    serviceAccountKind,
	}
	if err := collectOperatorResource(ctx, log, saGvk, cfg.ChartName); err != nil {
		log.Error(err, "Failed to collect ServiceAccount")
	}

	log.V(1).Info("Collecting ClusterRole")
	crGvk := schema.GroupVersionKind{
		Group:   rbacAPIGroup,
		Version: utils.V1APIVersion,
		Kind:    clusterRoleKind,
	}
	if err := collectOperatorResource(ctx, log, crGvk, cfg.ChartName); err != nil {
		log.Error(err, "Failed to collect ClusterRole")
	}

	log.V(1).Info("Collecting ClusterRoleBinding")
	crbGvk := schema.GroupVersionKind{
		Group:   rbacAPIGroup,
		Version: utils.V1APIVersion,
		Kind:    clusterRoleBindingKind,
	}
	if err := collectOperatorResource(ctx, log, crbGvk, cfg.ChartName); err != nil {
		log.Error(err, "Failed to collect ClusterRoleBinding")
	}

	log.V(1).Info("Collecting CRDs")
	crdGVK := schema.GroupVersionKind{
		Group:   apiExtensionsAPIGroup,
		Version: utils.V1APIVersion,
		Kind:    crdKind,
	}
	for _, gvk := range cfg.GVKs {
		crdName := strings.ToLower(gvk.Kind) + "s.coralogix.com"
		if gvk.Kind == utils.TCOLogsPoliciesKind || gvk.Kind == utils.TCOTracesPoliciesKind {
			crdName = strings.ToLower(gvk.Kind) + ".coralogix.com"
		}
		if gvk.Kind == utils.PrometheusRuleKind {
			crdName = strings.ToLower(gvk.Kind) + "s.monitoring.coreos.com"
		}

		if err := collectOperatorResource(ctx, log, crdGVK, crdName); err != nil {
			log.Error(err, "Failed to collect CRD", "crd", crdName)
		}
	}

	log.Info("Collecting logs")
	if err := collectLogs(ctx, log); err != nil {
		log.Error(err, "Failed to collect logs")
	}

	log.Info("Collecting custom resources")
	for _, ns := range cfg.Selector.NamespaceSelector {
		if err := collectCRsInNamespace(ctx, log, ns); err != nil {
			log.Error(err, "Failed to collect custom resources in namespace", "namespace", ns)
		}
	}

	if err := compress(); err != nil {
		log.Error(err, "Failed to compress output dir")
	}
}

func compress() error {
	absPath, err := filepath.Abs(filepath.Join("temp", "cxo-observer"))
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	parentDir := filepath.Dir(absPath)
	baseName := filepath.Base(absPath)
	archivePath := filepath.Join(".", "cxo-observer.tar.gz")

	cmd := exec.Command("tar", "-czf", archivePath, "-C", parentDir, baseName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	if err := os.RemoveAll(parentDir); err != nil {
		return fmt.Errorf("failed to remove original dir: %w", err)
	}

	return nil
}
