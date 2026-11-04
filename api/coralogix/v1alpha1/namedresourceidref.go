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
	"math"
	"strconv"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	slos "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"
)

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type NamedResourceIDRef struct {
	// +optional
	BackendRef *NamedBackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.id) != has(self.name)",message="Exactly one of id or name must be set"
type NamedBackendRef struct {
	// +optional
	// +kubebuilder:validation:MinLength=1
	ID *string `json:"id,omitempty"`
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name *string `json:"name,omitempty"`
}

// NamedBackendOnlyRef keeps a released reference that has no resourceRef arm.
type NamedBackendOnlyRef struct {
	BackendRef NamedBackendRef `json:"backendRef"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type NamedInt32ResourceIDRef struct {
	// +optional
	BackendRef *NamedInt32BackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.id) != has(self.name)",message="Exactly one of id or name must be set"
type NamedInt32BackendRef struct {
	// +optional
	ID *int32 `json:"id,omitempty"`
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name *string `json:"name,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.backendRef) != has(self.resourceRef)",message="Exactly one of backendRef or resourceRef must be set"
type NamedInt64ResourceIDRef struct {
	// +optional
	BackendRef *NamedInt64BackendRef `json:"backendRef,omitempty"`
	// +optional
	ResourceRef *ResourceRef `json:"resourceRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.id) != has(self.name)",message="Exactly one of id or name must be set"
type NamedInt64BackendRef struct {
	// +optional
	ID *int64 `json:"id,omitempty"`
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name *string `json:"name,omitempty"`
}

// NameLookupContext holds the second SDK client and caches names during one request conversion.
// +kubebuilder:object:generate=false
type NameLookupContext struct {
	clientSet *cxsdk.ClientSet
	cache     map[string]string
}

func NewNameLookupContext(clientSet *cxsdk.ClientSet) *NameLookupContext {
	return &NameLookupContext{clientSet: clientSet}
}

// ResolveNamedResourceID resolves an id, a custom resource, or a backend name.
func ResolveNamedResourceID(ctx context.Context, namespace string, ref NamedResourceIDRef, kind, version, lookupKey string, lookup *NameLookupContext, byName func(context.Context, *cxsdk.ClientSet, string) (string, error)) (string, error) {
	if ref.BackendRef != nil {
		if ref.BackendRef.ID != nil {
			return *ref.BackendRef.ID, nil
		}
		if ref.BackendRef.Name == nil {
			return "", fmt.Errorf("the %s backend reference has no id or name", kind)
		}
		if lookup == nil || lookup.clientSet == nil || byName == nil {
			return "", fmt.Errorf("the %s name lookup is not configured", kind)
		}
		key := kind + "\x00" + version + "\x00" + lookupKey + "\x00" + *ref.BackendRef.Name
		if id, ok := lookup.cache[key]; ok {
			return id, nil
		}
		id, err := byName(ctx, lookup.clientSet, *ref.BackendRef.Name)
		if err != nil {
			return "", err
		}
		if id == "" {
			return "", fmt.Errorf("%s with name %q has no id", kind, *ref.BackendRef.Name)
		}
		if lookup.cache == nil {
			lookup.cache = map[string]string{}
		}
		lookup.cache[key] = id
		return id, nil
	}
	if ref.ResourceRef == nil {
		return "", fmt.Errorf("the reference to a %s sets neither backendRef nor resourceRef", kind)
	}
	if version == "" {
		version = GroupVersion.Version
	}
	return ResolveResourceIDVersion(ctx, namespace, ResourceIDRef{ResourceRef: ref.ResourceRef}, kind, version)
}

// ResolveNamedNumericResourceID resolves a numeric backend id, backend name, or custom resource.
func ResolveNamedNumericResourceID(ctx context.Context, namespace string, backendID *int64, backendName *string, resourceRef *ResourceRef, kind, version, lookupKey string, lookup *NameLookupContext, byName func(context.Context, *cxsdk.ClientSet, string) (string, error), bits int) (int64, error) {
	if backendID != nil {
		if bits == 32 && (*backendID < math.MinInt32 || *backendID > math.MaxInt32) {
			return 0, fmt.Errorf("backend id %d is outside int32 range", *backendID)
		}
		return *backendID, nil
	}
	ref := NamedResourceIDRef{ResourceRef: resourceRef}
	if backendName != nil {
		ref.BackendRef = &NamedBackendRef{Name: backendName}
	}
	id, err := ResolveNamedResourceID(ctx, namespace, ref, kind, version, lookupKey, lookup, byName)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseInt(id, 10, bits)
	if err != nil {
		return 0, fmt.Errorf("%s id %q is not an int%d: %w", kind, id, bits, err)
	}
	return parsed, nil
}

// LookupArchiveRetentionName resolves a retention name through the Retentions service.
func LookupArchiveRetentionName(ctx context.Context, clientSet *cxsdk.ClientSet, name string) (string, error) {
	response, httpResponse, err := clientSet.ArchiveRetentions().RetentionsServiceGetRetentions(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("list archive retentions: %w", cxsdk.NewAPIError(httpResponse, err))
	}
	if response == nil {
		return "", fmt.Errorf("the archive retentions response is empty")
	}
	var id string
	for _, retention := range response.Retentions {
		if retention.Name != nil && *retention.Name == name && retention.Id != nil {
			id = *retention.Id
		}
	}
	if id == "" {
		return "", fmt.Errorf("archive retention with name %q was not found", name)
	}
	return id, nil
}

// LookupWebhookName resolves an outgoing webhook name through the Webhooks service.
func LookupWebhookName(ctx context.Context, clientSet *cxsdk.ClientSet, name string) (string, error) {
	response, httpResponse, err := clientSet.Webhooks().OutgoingWebhooksServiceListAllOutgoingWebhooks(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("list outgoing webhooks: %w", cxsdk.NewAPIError(httpResponse, err))
	}
	if response == nil {
		return "", fmt.Errorf("the outgoing webhooks response is empty")
	}
	var id string
	for _, webhook := range response.Deployed {
		if webhook.Name != nil && *webhook.Name == name && webhook.ExternalId != nil {
			id = strconv.FormatInt(*webhook.ExternalId, 10)
		}
	}
	if id == "" {
		return "", fmt.Errorf("outgoing webhook with name %q was not found", name)
	}
	return id, nil
}

// LookupAlertName resolves an alert name through the Alerts service.
func LookupAlertName(ctx context.Context, clientSet *cxsdk.ClientSet, name string) (string, error) {
	response, httpResponse, err := clientSet.Alerts().AlertDefsServiceListAlertDefs(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("list alerts: %w", cxsdk.NewAPIError(httpResponse, err))
	}
	if response == nil {
		return "", fmt.Errorf("the alerts response is empty")
	}
	var id string
	for _, alert := range response.GetAlertDefs() {
		properties := alert.GetAlertDefProperties()
		if properties.Name != nil && *properties.Name == name {
			id = alert.GetId()
		}
	}
	if id == "" {
		return "", fmt.Errorf("alert with name %q was not found", name)
	}
	return id, nil
}

// LookupSLOName resolves an SLO name through the SLO service.
func LookupSLOName(ctx context.Context, clientSet *cxsdk.ClientSet, name string) (string, error) {
	filters := []slos.SloFilter{{
		Field: &slos.SloFilterField{ConstFilter: slos.SLOCONSTANTFILTERFIELD_SLO_CONST_FILTER_FIELD_SLO_NAME.Ptr()},
		Predicate: &slos.SloFilterPredicate{Is: &slos.IsFilterPredicate{
			StringValues: &slos.StringValues{Values: []string{name}},
		}},
	}}
	response, httpResponse, err := clientSet.SLOs().SlosServiceListSlos(ctx).Filters(filters).Execute()
	if err != nil {
		return "", fmt.Errorf("list SLOs: %w", cxsdk.NewAPIError(httpResponse, err))
	}
	if response == nil {
		return "", fmt.Errorf("the SLO response is empty")
	}
	for _, slo := range response.Slos {
		if slo.Name == name && slo.Id != nil {
			return *slo.Id, nil
		}
	}
	return "", fmt.Errorf("SLO with name %q was not found", name)
}
