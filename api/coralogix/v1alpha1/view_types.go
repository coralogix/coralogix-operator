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
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api/coralogix"
	"github.com/coralogix/coralogix-operator/internal/config"
)

// ViewSpec defines the desired state of View.
type ViewSpec struct {
	// Name of the view.
	Name string `json:"name"`

	// SearchQuery is the search query for the view.
	// +optional
	SearchQuery *SearchQuery `json:"searchQuery,omitempty"`

	// TimeSelection is the time selection for the view. Exactly one of quickSelection or customSelection must be set.
	// +kubebuilder:validation:XValidation:rule="has(self.quickSelection) != has(self.customSelection)",message="Exactly one of quickSelection or customSelection must be set"
	TimeSelection TimeSelection `json:"timeSelection"`

	// Filters is the filters for the view.
	Filters SelectedFilters `json:"filters"`

	// Folder is the folder to which the view belongs.
	// +optional
	Folder *Folder `json:"folder,omitempty"`
}

type SearchQuery struct {
	// Query is the search query.
	Query string `json:"query"`
}

type TimeSelection struct {
	// QuickSelection is the quick selection for the view.
	// +optional
	QuickSelection *QuickTimeSelection `json:"quickSelection,omitempty"`

	// CustomSelection is the custom selection for the view.
	// +optional
	CustomSelection *CustomTimeSelection `json:"customSelection,omitempty"`
}

type QuickTimeSelection struct {
	// Seconds is the number of seconds for the quick selection.
	Seconds uint32 `json:"seconds"`
}

type CustomTimeSelection struct {
	// FromTime is the start time for the custom selection.
	FromTime metav1.Time `json:"fromTime"`

	// ToTime is the end time for the custom selection.
	ToTime metav1.Time `json:"toTime"`
}

type SelectedFilters struct {
	// Filters is the list of filters for the view.
	// +kubebuilder:validation:MinItems=1
	Filters []ViewFilter `json:"filters"`
}

type ViewFilter struct {
	// Name is the name of the filter.
	Name string `json:"name"`

	// SelectedValues is the selected values for the filter.
	SelectedValues map[string]bool `json:"selectedValues"`
}

type Folder struct {
	// ViewFolder custom resource name and namespace. If namespace is not set, the View namespace will be used.
	ResourceRef *ResourceRef `json:"resourceRef"`
}

func (v *View) ExtractCreateRequest(ctx context.Context, log logr.Logger) (*cxsdk.CreateViewRequest, error) {
	timeSelection, err := v.Spec.ExtractTimeSelection()
	if err != nil {
		return nil, fmt.Errorf("error on extracting time selection: %w", err)
	}

	folderId, err := v.ExtractFolderId(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("error on extracting folder id: %w", err)
	}

	query := ""
	if sq := v.Spec.SearchQuery; sq != nil {
		query = sq.Query
	}

	return &cxsdk.CreateViewRequest{
		Name: utils.StringPointerToWrapperspbString(ptr.To(v.Spec.Name)),
		SearchQuery: &cxsdk.SearchQuery{
			Query: utils.StringPointerToWrapperspbString(ptr.To(query)),
		},
		TimeSelection: timeSelection,
		Filters:       v.Spec.ExtractFilters(),
		FolderId:      utils.StringPointerToWrapperspbString(folderId),
	}, nil
}

func (v *View) ExtractReplaceRequest(ctx context.Context, log logr.Logger) (*cxsdk.ReplaceViewRequest, error) {
	viewId, err := strconv.Atoi(*v.Status.ID)
	if err != nil {
		return nil, fmt.Errorf("error on converting view id to int: %w", err)
	}

	timeSelection, err := v.Spec.ExtractTimeSelection()
	if err != nil {
		return nil, fmt.Errorf("error on extracting time selection: %w", err)
	}

	folderId, err := v.ExtractFolderId(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("error on extracting folder id: %w", err)
	}

	query := ""
	if sq := v.Spec.SearchQuery; sq != nil {
		query = sq.Query
	}

	return &cxsdk.ReplaceViewRequest{
		View: &cxsdk.View{
			Id:   wrapperspb.Int32(int32(viewId)),
			Name: utils.StringPointerToWrapperspbString(ptr.To(v.Spec.Name)),
			SearchQuery: &cxsdk.SearchQuery{
				Query: utils.StringPointerToWrapperspbString(ptr.To(query)),
			},
			TimeSelection: timeSelection,
			Filters:       v.Spec.ExtractFilters(),
			FolderId:      utils.StringPointerToWrapperspbString(folderId),
		},
	}, nil
}

func (v *View) ExtractFolderId(ctx context.Context, log logr.Logger) (*string, error) {
	if v.Spec.Folder == nil {
		return nil, nil
	}

	namespace := v.Namespace
	if resourceRefNs := v.Spec.Folder.ResourceRef.Namespace; resourceRefNs != nil {
		namespace = *resourceRefNs
	}

	log.Info("Extracting view folder ID", "namespace", namespace, "name", v.Spec.Folder.ResourceRef.Name)
	vf := &ViewFolder{}
	if err := config.GetClient().Get(ctx, client.ObjectKey{Name: v.Spec.Folder.ResourceRef.Name, Namespace: namespace}, vf); err != nil {
		return nil, err
	}

	if !config.GetConfig().Selector.Matches(vf.Labels, vf.Namespace) {
		return nil, fmt.Errorf("view folder %s does not match selector", vf.Name)
	}

	if vf.Status.ID == nil {
		return nil, fmt.Errorf("ID is not populated for ViewFolder %s", v.Spec.Folder.ResourceRef.Name)
	}

	return vf.Status.ID, nil
}

func (s *ViewSpec) ExtractTimeSelection() (*cxsdk.TimeSelection, error) {
	if s.TimeSelection.QuickSelection != nil {
		return &cxsdk.TimeSelection{
			SelectionType: &cxsdk.ViewTimeSelectionQuick{
				QuickSelection: &cxsdk.QuickTimeSelection{
					Seconds: s.TimeSelection.QuickSelection.Seconds,
				},
			},
		}, nil
	} else if s.TimeSelection.CustomSelection != nil {
		return &cxsdk.TimeSelection{
			SelectionType: &cxsdk.ViewTimeSelectionCustom{
				CustomSelection: &cxsdk.CustomTimeSelection{
					FromTime: timestamppb.New(s.TimeSelection.CustomSelection.FromTime.Time),
					ToTime:   timestamppb.New(s.TimeSelection.CustomSelection.ToTime.Time),
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("no time selection provided")
}

func (s *ViewSpec) ExtractFilters() *cxsdk.SelectedFilters {
	var filters []*cxsdk.ViewFilter
	for _, filter := range s.Filters.Filters {
		filters = append(filters, &cxsdk.ViewFilter{
			Name:           utils.StringPointerToWrapperspbString(ptr.To(filter.Name)),
			SelectedValues: filter.SelectedValues,
		})
	}
	return &cxsdk.SelectedFilters{
		Filters: filters,
	}
}

// ViewStatus defines the observed state of View.
type ViewStatus struct {
	// +optional
	ID *string `json:"id,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (v *View) GetConditions() []metav1.Condition {
	return v.Status.Conditions
}

func (v *View) SetConditions(conditions []metav1.Condition) {
	v.Status.Conditions = conditions
}

func (v *View) HasIDInStatus() bool {
	return v.Status.ID != nil && *v.Status.ID != ""
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// View is the Schema for the Views API.
// See also https://coralogix.com/docs/user-guides/monitoring-and-insights/explore-screen/custom-views/
//
// **Added in v0.4.0**
type View struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ViewSpec   `json:"spec,omitempty"`
	Status ViewStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ViewList contains a list of View.
type ViewList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []View `json:"items"`
}

func init() {
	SchemeBuilder.Register(&View{}, &ViewList{})
}
