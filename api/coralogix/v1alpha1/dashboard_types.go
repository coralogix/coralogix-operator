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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	"github.com/coralogix/coralogix-operator/internal/config"
)

// DashboardSpec defines the desired state of Dashboard.
// See also https://coralogix.com/docs/user-guides/custom-dashboards/getting-started/
// +kubebuilder:validation:XValidation:rule="!(has(self.json) && has(self.configMapRef))", message="Only one of json or configMapRef can be declared at the same time"
type DashboardSpec struct {
	// +optional
	Json *string `json:"json,omitempty"`
	// GzipJson the model's JSON compressed with Gzip. Base64-encoded when in YAML.
	// +optional
	GzipJson []byte `json:"gzipJson,omitempty"`
	// model from configmap
	//+optional
	ConfigMapRef *v1.ConfigMapKeySelector `json:"configMapRef,omitempty"`
	// +optional
	FolderRef *DashboardFolderRef `json:"folderRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) || has(self.resourceRef)", message="One of backendRef or resourceRef is required"
// +kubebuilder:validation:XValidation:rule="!(has(self.backendRef) && has(self.resourceRef))", message="Only one of backendRef or resourceRef can be declared at the same time"
type DashboardFolderRef struct {
	// +optional
	// +kubebuilder:validation:XValidation:rule="has(self.id) || has(self.path)",message="One of id or path is required"
	// +kubebuilder:validation:XValidation:rule="!(has(self.id) && has(self.path))",message="Only one of id or path can be declared at the same time"
	BackendRef *DashboardFolderRefBackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

func (in *DashboardSpec) ExtractDashboardFromSpec(ctx context.Context, namespace string) (*cxsdk.Dashboard, error) {
	contentJson, err := ExtractJsonContentFromSpec(ctx, namespace, in)
	if err != nil {
		return nil, err
	}

	dashboard := new(cxsdk.Dashboard)
	JSONUnmarshal := protojson.UnmarshalOptions{
		DiscardUnknown: true,
		AllowPartial:   true,
	}

	if err = JSONUnmarshal.Unmarshal([]byte(contentJson), dashboard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contentJson: %w", err)
	}

	dashboard, err = expandDashboardFolder(ctx, namespace, in, dashboard)
	if err != nil {
		return nil, err
	}

	return dashboard, nil
}

func expandDashboardFolder(ctx context.Context, namespace string, in *DashboardSpec, dashboard *cxsdk.Dashboard) (*cxsdk.Dashboard, error) {
	if folderRef := in.FolderRef; folderRef != nil {
		if backendRef := folderRef.BackendRef; backendRef != nil {
			if id := backendRef.ID; id != nil {
				dashboard.Folder = &cxsdk.DashboardFolderID{
					FolderId: &cxsdk.UUID{
						Value: *id,
					},
				}
			} else if path := backendRef.Path; path != nil {
				segments := strings.Split(*path, "/")
				dashboard.Folder = &cxsdk.DashboardFolderPath{
					FolderPath: &cxsdk.FolderPath{
						Segments: segments,
					},
				}
			}
		} else if resourceRef := folderRef.ResourceRef; resourceRef != nil {
			folderId, err := GetFolderIdFromFolderCR(ctx, namespace, *resourceRef)
			if err != nil {
				return nil, err
			}
			dashboard.Folder = &cxsdk.DashboardFolderID{
				FolderId: &cxsdk.UUID{
					Value: folderId,
				},
			}
		} else {
			return nil, fmt.Errorf("folderRef.BackendRef or folderRef.ResourceRef is required")
		}
	}

	return dashboard, nil
}

func ExtractJsonContentFromSpec(ctx context.Context, namespace string, in *DashboardSpec) (string, error) {
	if json := in.Json; json != nil {
		return *json, nil
	} else if gzipJson := in.GzipJson; gzipJson != nil {
		content, err := Unzip(gzipJson)
		if err != nil {
			return "", fmt.Errorf("failed to gunzip contentJson: %w", err)
		}
		return string(content), nil
	} else if configMapRef := in.ConfigMapRef; configMapRef != nil {
		dashboardConfigMap := &v1.ConfigMap{}
		if err := config.GetClient().Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapRef.Name}, dashboardConfigMap); err != nil {
			return "", err
		}
		if content, ok := dashboardConfigMap.Data[configMapRef.Key]; ok {
			return content, nil
		}

		return "", fmt.Errorf("cannot find key '%v' in config map '%v'", configMapRef.Key, configMapRef.Name)
	}

	return "", fmt.Errorf("json, gzipContentJson or configMapRef is required")
}

func Unzip(compressed []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(gz)
}

// DashboardStatus defines the observed state of Dashboard.
type DashboardStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (d *Dashboard) GetConditions() []metav1.Condition {
	return d.Status.Conditions
}

func (d *Dashboard) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions
}

func (d *Dashboard) HasIDInStatus() bool {
	return d.Status.ID != nil && *d.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:conversion:hub
// +kubebuilder:subresource:status

// Dashboard is the Schema for the dashboards API.
//
// **Added in v0.4.0**
type Dashboard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DashboardSpec   `json:"spec,omitempty"`
	Status DashboardStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DashboardList contains a list of Dashboard.
type DashboardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dashboard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dashboard{}, &DashboardList{})
}
