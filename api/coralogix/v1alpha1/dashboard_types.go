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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	"github.com/coralogix/coralogix-operator/internal/config"
	"google.golang.org/protobuf/encoding/protojson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DashboardSpec defines the desired state of Dashboard.
type DashboardSpec struct {
	// +optional
	ContentJson *string `json:"contentJson,omitempty"`
	// GzipJson the model's JSON compressed with Gzip. Base64-encoded when in YAML.
	// +optional
	GzipContentJson []byte `json:"gzipJson,omitempty"`
	// +optional
	URL *string `json:"url,omitempty"`
	// +optional
	FolderRef *DashboardFolderRef `json:"folderRef,omitempty"`
}

func (in *DashboardSpec) ExtractDashboardFromSpec(ctx context.Context, namespace string) (*cxsdk.Dashboard, error) {
	dashboard := new(cxsdk.Dashboard)
	var contentJson string
	if in.ContentJson != nil {
		contentJson = *in.ContentJson
	} else if in.GzipContentJson != nil {
		content, err := Gunzip(in.GzipContentJson)
		if err != nil {
			return nil, fmt.Errorf("failed to gunzip contentJson: %w", err)
		}
		contentJson = string(content)
	} else if url := in.URL; url != nil {
		//dashboard = *url
	}

	if err := protojson.Unmarshal([]byte(contentJson), dashboard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal contentJson: %w", err)
	}

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
			} else if resourceRef := folderRef.ResourceRef; resourceRef != nil {
				if resourceRef.Namespace == nil {
					resourceRef.Namespace = &namespace
				}
				df := &DashboardsFolder{}
				err := config.GetClient().Get(ctx, client.ObjectKey{Name: resourceRef.Name, Namespace: *resourceRef.Namespace}, df)
				if err != nil {
					return nil, fmt.Errorf("failed to get DashboardsFolder: %w", err)
				}
			}
		}
	}

	return dashboard, nil
}

func Gunzip(compressed []byte) ([]byte, error) {
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Dashboard is the Schema for the dashboards API.
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
