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
	"reflect"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/coralogix/coralogix-operator/v2/api/coralogix/v1beta1"
	"github.com/coralogix/coralogix-operator/v2/internal/config"
)

func TestAlertSchedulerExtractFilter(t *testing.T) {
	const whatExpression = "source logs | filter true"

	tests := []struct {
		name          string
		filter        Filter
		wantMetaLabel bool
		wantIDs       []string
		wantAllAlerts bool
		wantErr       string
	}{
		{
			name: "all alerts emits alert unique IDs filter without values",
			filter: Filter{
				WhatExpression: whatExpression,
				AllAlerts:      ptr.To(true),
			},
			wantAllAlerts: true,
		},
		{
			name: "meta labels emits meta labels filter",
			filter: Filter{
				WhatExpression: whatExpression,
				MetaLabels: []MetaLabel{
					{Key: "environment", Value: ptr.To("production")},
				},
			},
			wantMetaLabel: true,
		},
		{
			name: "direct alert unique IDs emits alert unique IDs filter with values",
			filter: Filter{
				WhatExpression: whatExpression,
				AlertUniqueIDs: []string{"alert-id-1", "alert-id-2"},
			},
			wantIDs: []string{"alert-id-1", "alert-id-2"},
		},
		{
			name: "all alerts false is rejected",
			filter: Filter{
				WhatExpression: whatExpression,
				AllAlerts:      ptr.To(false),
			},
			wantErr: "allAlerts must be true when set",
		},
		{
			name: "empty direct alert unique ID is rejected",
			filter: Filter{
				WhatExpression: whatExpression,
				AlertUniqueIDs: []string{"alert-id-1", ""},
			},
			wantErr: "alertUniqueIds[1] must not be empty",
		},
		{
			name: "empty direct alert unique IDs list is rejected",
			filter: Filter{
				WhatExpression: whatExpression,
				AlertUniqueIDs: []string{},
			},
			wantErr: "alertUniqueIds must not be empty",
		},
		{
			name: "empty meta labels list is rejected",
			filter: Filter{
				WhatExpression: whatExpression,
				MetaLabels:     []MetaLabel{},
			},
			wantErr: "metaLabels must not be empty",
		},
		{
			name: "empty alerts list is rejected",
			filter: Filter{
				WhatExpression: whatExpression,
				Alerts:         []AlertRef{},
			},
			wantErr: "alerts must not be empty",
		},
		{
			name: "multiple selector modes are rejected",
			filter: Filter{
				WhatExpression: whatExpression,
				AllAlerts:      ptr.To(true),
				AlertUniqueIDs: []string{"alert-id-1"},
			},
			wantErr: "exactly one of",
		},
		{
			name: "no selector mode is rejected",
			filter: Filter{
				WhatExpression: whatExpression,
			},
			wantErr: "exactly one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduler := &AlertScheduler{
				Spec: AlertSchedulerSpec{
					Filter: tt.filter,
				},
			}

			filter, err := scheduler.extractFilter()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractFilter returned error: %v", err)
			}

			if tt.wantMetaLabel {
				metaLabelFilter := filter.AlertSchedulerRuleProtobufV1FilterAlertMetaLabels
				if metaLabelFilter == nil {
					t.Fatal("expected meta label filter, got nil")
				}
				if got := metaLabelFilter.GetWhatExpression(); got != whatExpression {
					t.Fatalf("whatExpression = %q, want %q", got, whatExpression)
				}
				if len(metaLabelFilter.AlertMetaLabels.Value) != 1 {
					t.Fatalf("meta label count = %d, want 1", len(metaLabelFilter.AlertMetaLabels.Value))
				}
				return
			}

			uniqueIDFilter := filter.AlertSchedulerRuleProtobufV1FilterAlertUniqueIds
			if uniqueIDFilter == nil {
				t.Fatal("expected alert unique IDs filter, got nil")
			}
			if got := uniqueIDFilter.GetWhatExpression(); got != whatExpression {
				t.Fatalf("whatExpression = %q, want %q", got, whatExpression)
			}
			gotIDs, ok := uniqueIDFilter.AlertUniqueIds.GetValueOk()
			if tt.wantAllAlerts {
				if ok {
					t.Fatalf("all-alert filter should omit alertUniqueIds.value, got %v", gotIDs)
				}
				return
			}
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Fatalf("alertUniqueIds = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestAlertSchedulerExtractFilterResolvesAlertRefs(t *testing.T) {
	const (
		namespace      = "default"
		alertName      = "referenced-alert"
		alertBackendID = "backend-alert-id"
		whatExpression = "source logs | filter true"
	)

	scheme := runtime.NewScheme()
	if err := v1beta1.AddToScheme(scheme); err != nil {
		t.Fatalf("add v1beta1 to scheme: %v", err)
	}

	alert := &v1beta1.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertName,
			Namespace: namespace,
		},
		Status: v1beta1.AlertStatus{
			ID: ptr.To(alertBackendID),
		},
	}
	config.InitClient(fake.NewClientBuilder().WithScheme(scheme).WithObjects(alert).Build())

	scheduler := &AlertScheduler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: AlertSchedulerSpec{
			Filter: Filter{
				WhatExpression: whatExpression,
				Alerts: []AlertRef{
					{
						ResourceRef: &ResourceRef{Name: alertName},
					},
				},
			},
		},
	}

	filter, err := scheduler.extractFilter()
	if err != nil {
		t.Fatalf("extractFilter returned error: %v", err)
	}

	uniqueIDFilter := filter.AlertSchedulerRuleProtobufV1FilterAlertUniqueIds
	if uniqueIDFilter == nil {
		t.Fatal("expected alert unique IDs filter, got nil")
	}
	if got := uniqueIDFilter.GetWhatExpression(); got != whatExpression {
		t.Fatalf("whatExpression = %q, want %q", got, whatExpression)
	}
	if got := uniqueIDFilter.AlertUniqueIds.GetValue(); !reflect.DeepEqual(got, []string{alertBackendID}) {
		t.Fatalf("alertUniqueIds = %v, want [%s]", got, alertBackendID)
	}
}

func TestAlertSchedulerExtractFilterRejectsMissingAlertRef(t *testing.T) {
	scheduler := &AlertScheduler{
		Spec: AlertSchedulerSpec{
			Filter: Filter{
				WhatExpression: "source logs | filter true",
				Alerts:         []AlertRef{{}},
			},
		},
	}

	_, err := scheduler.extractFilter()
	if err == nil || !strings.Contains(err.Error(), "resourceRef is required") {
		t.Fatalf("error = %v, want missing resourceRef error", err)
	}
}
