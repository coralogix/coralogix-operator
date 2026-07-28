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

package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// testCfg points at the envtest control plane the tests in this package run against. Using
// envtest here, like the other controller suites do, means `make unit-tests` no longer needs a
// Kind cluster to be provisioned first.
var testCfg *rest.Config

func TestMain(m *testing.M) {
	// The PrometheusRule controller watches monitoring.coreos.com resources, so envtest needs
	// the Prometheus Operator CRDs on top of ours. `make prometheus-crds` fetches them.
	prometheusCRDs := os.Getenv("PROMETHEUS_CRDS_DIR")
	if prometheusCRDs == "" {
		prometheusCRDs = filepath.Join("..", "..", "bin", "prometheus-crds")
	}

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			prometheusCRDs,
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start the test environment: %v\n", err)
		os.Exit(1)
	}
	testCfg = cfg

	code := m.Run()

	if err := testEnv.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop the test environment: %v\n", err)
	}

	os.Exit(code)
}
