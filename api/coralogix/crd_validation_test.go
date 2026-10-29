// Copyright 2026 Coralogix Ltd.
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

package coralogix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

// Every CRD passes the checks that the API server runs when it creates the CRD, without a running
// API server. One of them is the CEL cost budget: a CRD over it fails here with "estimated rule
// cost exceeds budget", for example a generated kind whose rules sit in lists without a size limit.
func TestCRDsPassAPIServerValidation(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "config", "crd", "bases", "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no CRD in config/crd/bases")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if errs := validateCRDFile(t, path); len(errs) != 0 {
				t.Errorf("the API server would reject the CRD:\n%s", errs.ToAggregate())
			}
		})
	}
}

// The test fails on a CRD whose CEL rule is over the cost budget, so it is not silent.
func TestCRDValidationReportsCELBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coralogix.com_budgetprobes.yaml")
	if err := os.WriteFile(path, []byte(overBudgetCRD), 0o644); err != nil {
		t.Fatal(err)
	}
	errs := validateCRDFile(t, path)
	if errs == nil || !strings.Contains(errs.ToAggregate().Error(), "estimated rule cost exceeds budget") {
		t.Fatalf("want a CEL cost error, got: %v", errs.ToAggregate())
	}
}

// validateCRDFile runs the create steps of the API server on a CRD file: defaulting, conversion to
// the internal type, status.storedVersions as PrepareForCreate sets it, and validation.
func validateCRDFile(t *testing.T, path string) field.ErrorList {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scheme := runtime.NewScheme()
	install.Install(scheme)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.UnmarshalStrict(data, crd); err != nil {
		t.Fatal(err)
	}
	scheme.Default(crd)
	internal := &apiextensions.CustomResourceDefinition{}
	if err := scheme.Convert(crd, internal, nil); err != nil {
		t.Fatal(err)
	}
	for _, v := range internal.Spec.Versions {
		if v.Storage {
			internal.Status.StoredVersions = []string{v.Name}
		}
	}
	return validation.ValidateCustomResourceDefinition(context.Background(), internal)
}

// overBudgetCRD compares every pair of strings of every item of an unbounded list with a regular
// expression, so its estimated cost is far over the budget.
const overBudgetCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: budgetprobes.coralogix.com
spec:
  group: coralogix.com
  names:
    kind: BudgetProbe
    listKind: BudgetProbeList
    plural: budgetprobes
    singular: budgetprobe
  scope: Namespaced
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              groups:
                type: array
                items:
                  type: object
                  properties:
                    names:
                      type: array
                      items:
                        type: string
                x-kubernetes-validations:
                - rule: self.all(g, g.names.all(a, g.names.all(b, a.matches(b))))
                  message: over the budget on purpose
`
