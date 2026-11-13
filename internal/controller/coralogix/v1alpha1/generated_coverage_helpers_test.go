// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// resolveEveryReference makes every resourceRef of a generated spec resolve to a status id, so the
// Create request of a coverage spec can be built with no cluster objects. It returns the function
// that restores the client of the operator.
func resolveEveryReference() func() {
	scheme := runtime.NewScheme()
	if err := coralogixv1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	original := config.GetClient()
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(_ context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
			target, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return nil
			}
			target.SetName(key.Name)
			target.SetNamespace(key.Namespace)
			return unstructured.SetNestedField(target.Object, "id-"+key.Name, "status", "id")
		},
	}).Build())
	return func() { config.InitClient(original) }
}
