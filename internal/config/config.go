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
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var cfg *Config

type Config struct {
	Client            client.Client
	Scheme            *runtime.Scheme
	ReconcileInterval time.Duration
	Selector          *Selector
}

func InitConfig(client client.Client, scheme *runtime.Scheme, labelSelector, namespaceSelector, interval string) error {
	selector, err := parseSelector(labelSelector, namespaceSelector)
	if err != nil {
		return err
	}

	if interval == "" {
		interval = "0"
	}

	reconcileInterval, err := strconv.Atoi(interval)
	if err != nil {
		return err
	}

	cfg = &Config{
		Client:            client,
		Scheme:            scheme,
		Selector:          selector,
		ReconcileInterval: time.Second * time.Duration(reconcileInterval),
	}

	return nil
}

func GetConfig() *Config {
	return cfg
}
