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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

// nolint:unused
var alertschedulerlog = logf.Log.WithName("alertscheduler-resource")

// SetupAlertSchedulerWebhookWithManager registers the webhook for AlertScheduler in the manager.
func SetupAlertSchedulerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralogixv1alpha1.AlertScheduler{}).
		WithValidator(&AlertSchedulerCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-coralogix-com-v1alpha1-alertscheduler,mutating=false,failurePolicy=fail,sideEffects=None,groups=coralogix.com,resources=alertschedulers,verbs=create;update,versions=v1alpha1,name=valertscheduler-v1alpha1.kb.io,admissionReviewVersions=v1

// AlertSchedulerCustomValidator struct is responsible for validating the AlertScheduler resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type AlertSchedulerCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &AlertSchedulerCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type AlertScheduler.
func (v *AlertSchedulerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	alertscheduler, ok := obj.(*coralogixv1alpha1.AlertScheduler)
	if !ok {
		return nil, fmt.Errorf("expected a AlertScheduler object but got %T", obj)
	}
	alertschedulerlog.Info("Validation for AlertScheduler upon creation", "name", alertscheduler.GetName())

	var errs error
	if err := validateFilter(alertscheduler.Spec.Filter); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := validateSchedule(alertscheduler.Spec.Schedule); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return admission.Warnings{errs.Error()}, errs
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type AlertScheduler.
func (v *AlertSchedulerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	alertscheduler, ok := newObj.(*coralogixv1alpha1.AlertScheduler)
	if !ok {
		return nil, fmt.Errorf("expected a AlertScheduler object for the newObj but got %T", newObj)
	}
	alertschedulerlog.Info("Validation for AlertScheduler upon update", "name", alertscheduler.GetName())

	var errs error
	if err := validateFilter(alertscheduler.Spec.Filter); err != nil {
		errs = errors.Join(errs, err)
	}

	if err := validateSchedule(alertscheduler.Spec.Schedule); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return admission.Warnings{errs.Error()}, errs
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type AlertScheduler.
func (v *AlertSchedulerCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateFilter(filter *coralogixv1alpha1.Filter) error {
	if (filter.MetaLabels == nil || len(filter.MetaLabels) == 0) &&
		(filter.Alerts == nil || len(filter.Alerts) == 0) {
		return fmt.Errorf("filter must contain at least one of the fields: metaLabels or alerts")
	}
	if (filter.MetaLabels != nil || len(filter.MetaLabels) != 0) &&
		(filter.Alerts != nil || len(filter.Alerts) != 0) {
		return fmt.Errorf("filter must contain only one of the fields: metaLabels or alerts")
	}

	return nil
}

func validateSchedule(schedule *coralogixv1alpha1.Schedule) error {
	if schedule.OneTime == nil && schedule.Recurring == nil {
		return fmt.Errorf("schedule must contain only one of the fields: oneTime or recurring")
	}

	if schedule.OneTime != nil && schedule.Recurring != nil {
		return fmt.Errorf("schedule must contain only one of the fields: oneTime or recurring")
	}

	if schedule.OneTime != nil {
		if err := validateTimeFrame(schedule.OneTime); err != nil {
			return err
		}
	}

	if schedule.Recurring != nil {
		if err := validateRecurring(schedule.Recurring); err != nil {
			return err
		}
	}

	return nil
}

func validateRecurring(recurring *coralogixv1alpha1.Recurring) error {
	if recurring.Always == nil && recurring.Dynamic == nil {
		return fmt.Errorf("recurring must contain only one of the fields: always or dynamic")
	}

	if recurring.Always != nil && recurring.Dynamic != nil {
		return fmt.Errorf("recurring must contain only one of the fields: always or dynamic")
	}

	if recurring.Dynamic != nil {
		if err := validateTimeFrame(recurring.Dynamic.TimeFrame); err != nil {
			return err
		}
	}

	return nil
}

func validateTimeFrame(timeFrame *coralogixv1alpha1.TimeFrame) error {
	if timeFrame.EndTime == nil && timeFrame.Duration == nil {
		return fmt.Errorf("timeFrame must contain only one of the fields: endTime or duration")
	}

	if timeFrame.EndTime != nil && timeFrame.Duration != nil {
		return fmt.Errorf("timeFrame must contain only one of the fields: endTime or duration")
	}

	return nil
}
