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

package utils

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	DefaultErrRequeuePeriod = 60 * time.Second
)

var labelFilter *LabelFilter

type LabelFilter struct {
	Selector labels.Selector
}

func (f *LabelFilter) Predicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return f.Matches(e.Object.GetLabels())
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return f.Matches(e.ObjectNew.GetLabels()) || f.Matches(e.ObjectOld.GetLabels())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return f.Matches(e.Object.GetLabels())
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return f.Matches(e.Object.GetLabels())
		},
	}
}

func (f *LabelFilter) Matches(resourceLabels map[string]string) bool {
	return f.Selector.Matches(labels.Set(resourceLabels))
}

func (f *LabelFilter) String() string {
	return f.Selector.String()
}

func InitLabelFilter(selector string) error {
	parsedSelector, err := labels.Parse(selector)
	if err != nil {
		return fmt.Errorf("failed to parse label selector: %w", err)
	}
	labelFilter = &LabelFilter{Selector: parsedSelector}
	return nil
}

func GetLabelFilter() *LabelFilter {
	if labelFilter == nil {
		labelFilter = &LabelFilter{Selector: labels.Everything()}
	}
	return labelFilter
}
