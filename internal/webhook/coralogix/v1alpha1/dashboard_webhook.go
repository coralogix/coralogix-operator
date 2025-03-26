/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/coralogix/coralogix-operator/internal/config"
	"google.golang.org/protobuf/encoding/protojson"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var dashboardlog = logf.Log.WithName("dashboard-resource")

// SetupDashboardWebhookWithManager registers the webhook for Dashboard in the manager.
func SetupDashboardWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1alpha1.Dashboard{}).
		WithValidator(&DashboardCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-dashboard,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=dashboards,verbs=create;update,versions=v1alpha1,name=vdashboard-v1alpha1.kb.io,admissionReviewVersions=v1

// DashboardCustomValidator struct is responsible for validating the Dashboard resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type DashboardCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &DashboardCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Dashboard.
func (v *DashboardCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dashboard, ok := obj.(*coralogixv1alpha1.Dashboard)
	if !ok {
		return nil, fmt.Errorf("expected a Dashboard object but got %T", obj)
	}
	spec := dashboard.Spec
	return validateDashboardSpec(ctx, spec, dashboard.Namespace)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Dashboard.
func (v *DashboardCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	dashboard, ok := newObj.(*coralogixv1alpha1.Dashboard)
	if !ok {
		return nil, fmt.Errorf("expected a Dashboard object for the newObj but got %T", newObj)
	}
	dashboardlog.Info("Validation for Dashboard upon update", "name", dashboard.GetName())
	return validateDashboardSpec(ctx, dashboard.Spec, dashboard.Namespace)
}

func validateDashboardSpec(ctx context.Context, spec coralogixv1alpha1.DashboardSpec, namespace string) (admission.Warnings, error) {
	if contentJsonOptions := getContentJsonOptions(spec); len(contentJsonOptions) > 1 {
		return nil, fmt.Errorf("only one of the following fields can be set: %s", contentJsonOptions)
	} else if len(contentJsonOptions) == 0 {
		return nil, fmt.Errorf("at least one of the following fields must be set: %s", "json, gzipJson, configMapRef")
	}

	// in case of invalid request with configMapRef, returning a warning instead of an error
	if configMapRef := spec.ConfigMapRef; configMapRef != nil {
		dashboardConfigMap := &v1.ConfigMap{}
		if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapRef.Name}, dashboardConfigMap); err != nil {
			return admission.Warnings{"failed to get ConfigMap: " + err.Error()}, nil
		}
		if contentJson, ok := dashboardConfigMap.Data[configMapRef.Key]; ok {
			dashboardBackendSchema := new(cxsdk.Dashboard)
			if err := protojson.Unmarshal([]byte(contentJson), dashboardBackendSchema); err != nil {
				return admission.Warnings{fmt.Sprintf("failed to unmarshal contentJson from config map: %s", err.Error())}, nil
			}
			return nil, nil
		} else {
			return admission.Warnings{fmt.Sprintf("cannot find key '%v' in config map '%v'", configMapRef.Key, configMapRef.Name)}, nil
		}
	}

	var contentJson, source string
	if json := spec.Json; json != nil {
		source = "json"
		contentJson = *json
	} else if gzipJson := spec.GzipJson; gzipJson != nil {
		source = "gzipJson"
		content, err := coralogixv1alpha1.Gunzip(gzipJson)
		if err != nil {
			return nil, fmt.Errorf("failed to gunzip contentJson: %w", err)
		}
		contentJson = string(content)
	}

	dashboardBackendSchema := new(cxsdk.Dashboard)
	if err := protojson.Unmarshal([]byte(contentJson), dashboardBackendSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contentJson from %s: %w", source, err)
	}

	return nil, nil
}

func getContentJsonOptions(spec coralogixv1alpha1.DashboardSpec) []string {
	var options []string
	if spec.Json != nil {
		options = append(options, "json")
	}
	if spec.GzipJson != nil {
		options = append(options, "gzipJson")
	}
	if spec.ConfigMapRef != nil {
		options = append(options, "configMapRef")
	}
	return options
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Dashboard.
func (v *DashboardCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dashboard, ok := obj.(*coralogixv1alpha1.Dashboard)
	if !ok {
		return nil, fmt.Errorf("expected a Dashboard object but got %T", obj)
	}
	dashboardlog.Info("Validation for Dashboard upon deletion", "name", dashboard.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
