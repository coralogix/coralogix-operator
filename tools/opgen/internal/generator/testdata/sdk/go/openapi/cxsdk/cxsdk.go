// Package cxsdk is a small stand-in for the client set of the management SDK.
package cxsdk

import (
	"errors"
	batchthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/batch_things_service"
	listthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/list_things_service"
	"net/http"

	bodyidthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/body_id_things_service"
	contentthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/content_things_service"
	hookdeletethings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/hook_delete_things_service"
	hookthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/hook_things_service"
	keythings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/key_things_service"
	maskthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/mask_things_service"
	patchthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/patch_things_service"
	refthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ref_things_service"
	secretthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/secret_things_service"
	settingthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/setting_things_service"
	thingpolicies "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/thing_policies_service"
	things "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/things_service"
	wrappedthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/wrapped_things_service"
)

// ClientSet holds the API clients.
type ClientSet struct{}

// Things returns the client of the things API.
func (c *ClientSet) Things() *things.ThingsServiceAPIService {
	return things.NewThingsServiceAPIService()
}

// PatchThings returns the client of the patch things API.
func (c *ClientSet) PatchThings() *patchthings.PatchThingsServiceAPIService {
	return patchthings.NewPatchThingsServiceAPIService()
}

// MaskThings returns the client of the mask things API.
func (c *ClientSet) MaskThings() *maskthings.MaskThingsServiceAPIService {
	return maskthings.NewMaskThingsServiceAPIService()
}

// RefThings returns the client of the ref things API.
func (c *ClientSet) RefThings() *refthings.RefThingsServiceAPIService {
	return refthings.NewRefThingsServiceAPIService()
}

// BodyIDThings returns the client of the body id things API.
func (c *ClientSet) BodyIDThings() *bodyidthings.BodyIDThingsServiceAPIService {
	return bodyidthings.NewBodyIDThingsServiceAPIService()
}

func (c *ClientSet) ContentThings() *contentthings.ContentThingsServiceAPIService {
	return contentthings.NewContentThingsServiceAPIService()
}

func (c *ClientSet) SecretThings() *secretthings.SecretThingsServiceAPIService {
	return secretthings.NewSecretThingsServiceAPIService()
}

func (c *ClientSet) WrappedThings() *wrappedthings.WrappedThingsServiceAPIService {
	return wrappedthings.NewWrappedThingsServiceAPIService()
}

// APIError is an error with an HTTP status.
type APIError struct {
	StatusCode int
	Err        error
}

func (e *APIError) Error() string { return e.Err.Error() }

// NewAPIError returns err with the status of resp, or nil when err is nil.
func NewAPIError(resp *http.Response, err error) error {
	if err == nil {
		return nil
	}
	apiErr := &APIError{Err: err}
	if resp != nil {
		apiErr.StatusCode = resp.StatusCode
	}
	return apiErr
}

// IsNotFound reports whether err is a 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func (c *ClientSet) HookThings() *hookthings.HookThingsServiceAPIService {
	return hookthings.NewHookThingsServiceAPIService()
}

func (c *ClientSet) HookDeleteThings() *hookdeletethings.HookDeleteThingsServiceAPIService {
	return hookdeletethings.NewHookDeleteThingsServiceAPIService()
}

func (c *ClientSet) KeyThings() *keythings.KeyThingsServiceAPIService {
	return keythings.NewKeyThingsServiceAPIService()
}

func (c *ClientSet) BatchThings() *batchthings.BatchThingsServiceAPIService {
	return batchthings.NewBatchThingsServiceAPIService()
}

func (c *ClientSet) ListThings() *listthings.ListThingsServiceAPIService {
	return listthings.NewListThingsServiceAPIService()
}

func (c *ClientSet) SettingThings() *settingthings.SettingThingsServiceAPIService {
	return settingthings.NewSettingThingsServiceAPIService()
}

func (c *ClientSet) ThingPolicies() *thingpolicies.ThingPoliciesServiceAPIService {
	return thingpolicies.NewThingPoliciesServiceAPIService()
}
