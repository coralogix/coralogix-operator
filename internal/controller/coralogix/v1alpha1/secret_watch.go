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
	"fmt"
	"reflect"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

// secretIndex is the field index of the Secret names that the spec of a resource references.
const secretIndex = "secretNames"

// watchSecrets indexes the resources of obj by the Secrets that their spec references, and returns
// the handler that enqueues the resources of a Secret when it changes. Generated controllers of
// kinds with a valueFrom line use it, so a rotated Secret reaches the remote resource.
func watchSecrets(mgr ctrl.Manager, obj client.Object, list client.ObjectList) (handler.EventHandler, error) {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), obj, secretIndex, secretNames); err != nil {
		return nil, fmt.Errorf("index Secret references: %w", err)
	}
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, secret client.Object) []reconcile.Request {
		return requestsForSecret(ctx, list.DeepCopyObject().(client.ObjectList), secret)
	}), nil
}

// requestsForSecret lists the resources in the namespace of secret that reference it and that the
// selector matches.
func requestsForSecret(ctx context.Context, list client.ObjectList, secret client.Object) []reconcile.Request {
	if err := config.GetClient().List(ctx, list,
		client.InNamespace(secret.GetNamespace()),
		client.MatchingFields{secretIndex: secret.GetName()}); err != nil {
		ctrl.Log.Info("Failed to list resources for Secret", "namespace", secret.GetNamespace(), "secret", secret.GetName())
		return nil
	}
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil
	}
	var requests []reconcile.Request
	for _, item := range items {
		obj := item.(client.Object)
		if config.GetConfig().Selector.Matches(obj.GetLabels(), obj.GetNamespace()) {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(obj)})
		}
	}
	return requests
}

var secretKeySelectorType = reflect.TypeFor[*corev1.SecretKeySelector]()

// secretNames returns the unique names of the Secrets that the spec of obj references, at any depth.
func secretNames(obj client.Object) []string {
	var names []string
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if v.Type() == secretKeySelectorType {
			if ref := v.Interface().(*corev1.SecretKeySelector); ref != nil && ref.Name != "" && !slices.Contains(names, ref.Name) {
				names = append(names, ref.Name)
			}
			return
		}
		// A scalar or a map holds no Secret reference.
		if v.Kind() == reflect.Pointer && !v.IsNil() {
			walk(v.Elem())
		}
		if v.Kind() == reflect.Struct {
			for i := range v.NumField() {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			for i := range v.Len() {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(obj).Elem().FieldByName("Spec"))
	return names
}
