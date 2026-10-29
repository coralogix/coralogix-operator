// Package cxsdk is a small stand-in for the client set of the management SDK.
package cxsdk

import (
	"errors"
	"net/http"

	patchthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/patch_things_service"
	refthings "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ref_things_service"
	things "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/things_service"
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

// RefThings returns the client of the ref things API.
func (c *ClientSet) RefThings() *refthings.RefThingsServiceAPIService {
	return refthings.NewRefThingsServiceAPIService()
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
