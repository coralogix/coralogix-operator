// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"strconv"

	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/internal/config"
)

// DataSetSpec defines the desired state of DataSet.
// +kubebuilder:validation:XValidation:rule="has(self.csv) != has(self.configMapRef)", message="Exactly one of csv or configMapRef must be set"
type DataSetSpec struct {
	// The name of the data set.
	Name string `json:"name"`

	// The description of the data set.
	Description string `json:"description"`

	// Inline CSV data. Conflicts with ConfigMapRef.
	// +optional
	CSV *string `json:"csv,omitempty"`

	// Reference to a ConfigMap that contains the data set CSV. Conflicts with CSV.
	// +optional
	ConfigMapRef *corev1.ConfigMapKeySelector `json:"configMapRef,omitempty"`
}

func (d *DataSet) ExtractCreateDataSetRequest(ctx context.Context) (*cxsdk.CreateDataSetRequest, error) {
	var fileContent string
	var err error

	if d.Spec.CSV != nil {
		fileContent = *d.Spec.CSV
	} else if d.Spec.ConfigMapRef != nil {
		fileContent, err = readConfigMap(ctx, *d.Spec.ConfigMapRef, d.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read configmap: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either CSV or ConfigMapRef must be provided")
	}

	return &cxsdk.CreateDataSetRequest{
		Name:        wrapperspb.String(d.Spec.Name),
		Description: wrapperspb.String(d.Spec.Description),
		File: &cxsdk.File{
			Name:      wrapperspb.String(" "),
			Extension: wrapperspb.String("csv"),
			Content: &cxsdk.FileTextual{
				Textual: wrapperspb.String(fileContent),
			},
		},
	}, nil
}

func (d *DataSet) ExtractUpdateDataSetRequest(ctx context.Context) (*cxsdk.UpdateDataSetRequest, error) {
	if d.Status.ID == nil {
		return nil, fmt.Errorf("data set ID is not set")
	}

	id, err := strconv.Atoi(*d.Status.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ID to UInt32: %w", err)
	}

	var fileContent string
	if d.Spec.CSV != nil {
		fileContent = *d.Spec.CSV
	} else if d.Spec.ConfigMapRef != nil {
		fileContent, err = readConfigMap(ctx, *d.Spec.ConfigMapRef, d.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to read configmap: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either CSV or ConfigMapRef must be provided")
	}

	return &cxsdk.UpdateDataSetRequest{
		CustomEnrichmentId: wrapperspb.UInt32(uint32(id)),
		Name:               wrapperspb.String(d.Spec.Name),
		Description:        wrapperspb.String(d.Spec.Description),
		File: &cxsdk.File{
			Name:      wrapperspb.String(" "),
			Extension: wrapperspb.String("csv"),
			Content: &cxsdk.FileTextual{
				Textual: wrapperspb.String(fileContent),
			},
		},
	}, nil
}

func readConfigMap(ctx context.Context, configMapRef corev1.ConfigMapKeySelector, namespace string) (string, error) {
	cm := &corev1.ConfigMap{}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapRef.Name}, cm); err != nil {
		return "", err
	}

	if content, ok := cm.Data[configMapRef.Key]; ok {
		return content, nil
	}

	return "", fmt.Errorf("cannot find key '%v' in config map '%v'", configMapRef.Key, configMapRef.Name)
}

// DataSetStatus defines the observed state of DataSet.
type DataSetStatus struct { // +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (d *DataSet) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *DataSet) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DataSet is the Schema for the datasets API.
type DataSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataSetSpec   `json:"spec,omitempty"`
	Status DataSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataSetList contains a list of DataSet.
type DataSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataSet{}, &DataSetList{})
}
