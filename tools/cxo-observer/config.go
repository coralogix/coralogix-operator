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
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var (
	cfg = &Config{
		Selector: &Selector{
			LabelSelector: labels.Everything(),
		},
	}
	once sync.Once
)

type Config struct {
	ChartName      string
	ChartNamespace string
	GVKs           []schema.GroupVersionKind
	Selector       *Selector
	Client         client.Client
}

func initConfig(log logr.Logger) {
	once.Do(func() {
		var err error
		var namespaceSelector string
		var labelSelector string
		flag.StringVar(&cfg.ChartName, "chart-name", "coralogix-operator",
			"The name of Coralogix Operator Helm chart release.")
		flag.StringVar(&cfg.ChartNamespace, "chart-namespace", "",
			"The namespace of Coralogix Operator Helm chart release.")
		flag.StringVar(&namespaceSelector, "namespace-selector", "",
			"A comma-separated list of namespaces to filter custom resources.")
		flag.StringVar(&labelSelector, "label-selector", labelSelector,
			"A comma-separated list of key=value labels to filter custom resources.")
		flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
			flag.PrintDefaults()
		}

		opts := zap.Options{}
		opts.BindFlags(flag.CommandLine)
		flag.Parse()

		ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

		if cfg.ChartNamespace == "" {
			log.Error(errors.New("chart-namespace is required"), "Failed to initialize config")
			os.Exit(1)
		}

		cfg.Client, err = getClient()
		if err != nil {
			log.Error(err, "Failed to initialize client")
			os.Exit(1)
		}

		cfg.GVKs = utils.GetGVKs(scheme)

		cfg.Selector, err = parseSelector(labelSelector, namespaceSelector)
		if err != nil {
			log.Error(err, "Failed to parse selector")
			os.Exit(1)
		}
	})
}

func getClient() (client.Client, error) {
	var err error
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return nil, err
	}

	return c, nil
}
