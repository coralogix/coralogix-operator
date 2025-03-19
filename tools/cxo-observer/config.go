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
	"flag"
	"fmt"
	"github.com/coralogix/coralogix-operator/internal/utils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strings"
	"sync"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixv1beta1 "github.com/coralogix/coralogix-operator/api/coralogix/v1beta1"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

var (
	cfg  = &Config{}
	once sync.Once
	//kinds = []string{
	//	utils.RuleGroupKind, utils.AlertKind, utils.RecordingRuleGroupSetKind, utils.OutboundWebhookKind,
	//	utils.ApiKeyKind, utils.CustomRoleKind, utils.ScopeKind, utils.GroupKind, utils.TCOLogsPoliciesKind,
	//	utils.TCOTracesPoliciesKind, utils.IntegrationKind, utils.ConnectorKind, utils.PresetKind,
	//	utils.GlobalRouterKind, utils.PrometheusRuleKind,
	//}
	kinds = []string{
		utils.ScopeKind,
	}
)

type Config struct {
	OutputDir          string
	ChartName          string
	ChartNamespace     string
	Client             client.Client
	RequestedResources map[schema.GroupVersionKind]RequestedResource
}

type RequestedResource struct {
	Names      []string
	Namespaces []string
}

type ResourceFlags struct {
	Names      *string
	Namespaces *string
}

func initConfig(log logr.Logger) {
	once.Do(func() {
		var err error
		flag.StringVar(&cfg.OutputDir, "output-dir", "./output",
			"The directory where the output files will be dumped")
		flag.StringVar(&cfg.ChartName, "chart-name", "coralogix-operator",
			"The name of Coralogix Operator Helm chart release")
		flag.StringVar(&cfg.ChartNamespace, "chart-namespace", "coralogix-operator-system",
			"The namespace of Coralogix Operator Helm chart release")

		requestedResources := getRequestedResources()

		opts := zap.Options{}
		opts.BindFlags(flag.CommandLine)
		flag.Parse()

		ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

		cfg.Client, err = getClient()
		if err != nil {
			log.Error(err, "Failed to create client")
			os.Exit(1)
		}

		cfg.RequestedResources = parseRequestedResources(requestedResources)
	})
}

func GetConfig() *Config {
	return cfg
}

func getClient() (client.Client, error) {
	var err error
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return nil, err
	}
	if err = prometheusv1.AddToScheme(c.Scheme()); err != nil {
		return nil, err
	}
	if err = coralogixv1alpha1.AddToScheme(c.Scheme()); err != nil {
		return nil, err
	}
	if err = coralogixv1beta1.AddToScheme(c.Scheme()); err != nil {
		return nil, err
	}

	return c, nil
}

func getRequestedResources() map[schema.GroupVersionKind]ResourceFlags {
	result := make(map[schema.GroupVersionKind]ResourceFlags)
	for _, kind := range kinds {
		var names string
		flag.StringVar(
			&names,
			fmt.Sprintf("%s-names", strings.ToLower(kind)),
			"",
			fmt.Sprintf("The %s resources to be collected", kind),
		)

		var namespaces string
		flag.StringVar(
			&namespaces,
			fmt.Sprintf("%s-namespaces", strings.ToLower(kind)),
			"",
			fmt.Sprintf("The %s resources namespaces to be collected", kind),
		)
		result[schema.GroupVersionKind{
			Group:   utils.CoralogixAPIGroup,
			Version: utils.V1alpha1APIVersion,
			Kind:    kind}] = ResourceFlags{
			Names:      &names,
			Namespaces: &namespaces,
		}
	}

	return result
}

func parseRequestedResources(requestedResources map[schema.GroupVersionKind]ResourceFlags) map[schema.GroupVersionKind]RequestedResource {
	result := make(map[schema.GroupVersionKind]RequestedResource)
	for gvk, flags := range requestedResources {
		result[gvk] = RequestedResource{
			Names:      strings.Split(*flags.Names, ","),
			Namespaces: strings.Split(*flags.Namespaces, ","),
		}
	}

	return result
}
