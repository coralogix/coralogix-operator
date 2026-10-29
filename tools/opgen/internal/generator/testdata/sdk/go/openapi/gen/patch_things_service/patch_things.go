// Package patch_things_service is a small stand-in for an SDK package that openapi-generator writes.
package patch_things_service

import (
	"context"
	"errors"
	"net/http"
)

type PatchThingKind string

const (
	PATCHTHINGKIND_PATCH_THING_KIND_UNSPECIFIED PatchThingKind = "PATCH_THING_KIND_UNSPECIFIED"
	PATCHTHINGKIND_PATCH_THING_KIND_ALERTS      PatchThingKind = "PATCH_THING_KIND_ALERTS"
	PATCHTHINGKIND_PATCH_THING_KIND_CASES       PatchThingKind = "PATCH_THING_KIND_CASES"
)

type PatchThing struct {
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Enabled     bool              `json:"enabled"`
	Kind        PatchThingKind    `json:"kind"`
	Target      *PatchThingTarget `json:"target,omitempty"`
}

type PatchThingTarget struct {
	Some   *PatchThingSome   `json:"some,omitempty"`
	Prefix *PatchThingPrefix `json:"prefix,omitempty"`
}

type PatchThingSome struct {
	Ids []string `json:"ids"`
}

type PatchThingPrefix struct {
	Value string `json:"value"`
}

type CreatePatchThingRequest struct {
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Kind        PatchThingKind    `json:"kind"`
	Target      *PatchThingTarget `json:"target,omitempty"`
}

type UpdatePatchThingRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Target      *PatchThingTarget `json:"target,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type PatchThingsServiceAPIService struct{}

func NewPatchThingsServiceAPIService() *PatchThingsServiceAPIService {
	return &PatchThingsServiceAPIService{}
}

type ApiPatchThingsServiceCreatePatchThingRequest struct {
	ctx  context.Context
	body *CreatePatchThingRequest
}

func (a *PatchThingsServiceAPIService) PatchThingsServiceCreatePatchThing(ctx context.Context) ApiPatchThingsServiceCreatePatchThingRequest {
	return ApiPatchThingsServiceCreatePatchThingRequest{ctx: ctx}
}

func (r ApiPatchThingsServiceCreatePatchThingRequest) CreatePatchThingRequest(createPatchThingRequest CreatePatchThingRequest) ApiPatchThingsServiceCreatePatchThingRequest {
	r.body = &createPatchThingRequest
	return r
}

func (r ApiPatchThingsServiceCreatePatchThingRequest) Execute() (*PatchThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiPatchThingsServiceGetPatchThingRequest struct {
	ctx context.Context
	id  string
}

func (a *PatchThingsServiceAPIService) PatchThingsServiceGetPatchThing(ctx context.Context, id string) ApiPatchThingsServiceGetPatchThingRequest {
	return ApiPatchThingsServiceGetPatchThingRequest{ctx: ctx, id: id}
}

func (r ApiPatchThingsServiceGetPatchThingRequest) Execute() (*PatchThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiPatchThingsServiceUpdatePatchThingRequest struct {
	ctx        context.Context
	id         string
	body       *UpdatePatchThingRequest
	updateMask *string
}

func (a *PatchThingsServiceAPIService) PatchThingsServiceUpdatePatchThing(ctx context.Context, id string) ApiPatchThingsServiceUpdatePatchThingRequest {
	return ApiPatchThingsServiceUpdatePatchThingRequest{ctx: ctx, id: id}
}

func (r ApiPatchThingsServiceUpdatePatchThingRequest) UpdatePatchThingRequest(updatePatchThingRequest UpdatePatchThingRequest) ApiPatchThingsServiceUpdatePatchThingRequest {
	r.body = &updatePatchThingRequest
	return r
}

func (r ApiPatchThingsServiceUpdatePatchThingRequest) UpdateMask(updateMask string) ApiPatchThingsServiceUpdatePatchThingRequest {
	r.updateMask = &updateMask
	return r
}

func (r ApiPatchThingsServiceUpdatePatchThingRequest) Execute() (*PatchThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiPatchThingsServiceDeletePatchThingRequest struct {
	ctx context.Context
	id  string
}

func (a *PatchThingsServiceAPIService) PatchThingsServiceDeletePatchThing(ctx context.Context, id string) ApiPatchThingsServiceDeletePatchThingRequest {
	return ApiPatchThingsServiceDeletePatchThingRequest{ctx: ctx, id: id}
}

func (r ApiPatchThingsServiceDeletePatchThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
