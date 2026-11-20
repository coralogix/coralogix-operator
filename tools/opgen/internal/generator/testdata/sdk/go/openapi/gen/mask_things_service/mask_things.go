// Package mask_things_service is a small stand-in for an SDK package that openapi-generator writes.
package mask_things_service

import (
	"context"
	"errors"
	"net/http"
)

type MaskThing struct {
	Id          string        `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	Rule        MaskThingRule `json:"rule"`
}

type MaskThingRule struct {
	Enabled *bool             `json:"enabled,omitempty"`
	Note    *string           `json:"note,omitempty"`
	Some    *PatchThingSome   `json:"some,omitempty"`
	Prefix  *PatchThingPrefix `json:"prefix,omitempty"`
}

type PatchThingSome struct {
	Ids []string `json:"ids"`
}

type PatchThingPrefix struct {
	Value string `json:"value"`
}

type CreateMaskThingRequest struct {
	Name        string        `json:"name"`
	Description *string       `json:"description,omitempty"`
	Rule        MaskThingRule `json:"rule"`
}

type UpdateMaskThingRequest struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Rule        *MaskThingRule `json:"rule,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type MaskThingsServiceAPIService struct{}

func NewMaskThingsServiceAPIService() *MaskThingsServiceAPIService {
	return &MaskThingsServiceAPIService{}
}

type ApiMaskThingsServiceCreateMaskThingRequest struct {
	ctx  context.Context
	body *CreateMaskThingRequest
}

func (a *MaskThingsServiceAPIService) MaskThingsServiceCreateMaskThing(ctx context.Context) ApiMaskThingsServiceCreateMaskThingRequest {
	return ApiMaskThingsServiceCreateMaskThingRequest{ctx: ctx}
}

func (r ApiMaskThingsServiceCreateMaskThingRequest) CreateMaskThingRequest(createMaskThingRequest CreateMaskThingRequest) ApiMaskThingsServiceCreateMaskThingRequest {
	r.body = &createMaskThingRequest
	return r
}

func (r ApiMaskThingsServiceCreateMaskThingRequest) Execute() (*MaskThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiMaskThingsServiceGetMaskThingRequest struct {
	ctx context.Context
	id  string
}

func (a *MaskThingsServiceAPIService) MaskThingsServiceGetMaskThing(ctx context.Context, id string) ApiMaskThingsServiceGetMaskThingRequest {
	return ApiMaskThingsServiceGetMaskThingRequest{ctx: ctx, id: id}
}

func (r ApiMaskThingsServiceGetMaskThingRequest) Execute() (*MaskThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiMaskThingsServiceUpdateMaskThingRequest struct {
	ctx        context.Context
	id         string
	body       *UpdateMaskThingRequest
	updateMask *string
}

func (a *MaskThingsServiceAPIService) MaskThingsServiceUpdateMaskThing(ctx context.Context, id string) ApiMaskThingsServiceUpdateMaskThingRequest {
	return ApiMaskThingsServiceUpdateMaskThingRequest{ctx: ctx, id: id}
}

func (r ApiMaskThingsServiceUpdateMaskThingRequest) UpdateMaskThingRequest(updateMaskThingRequest UpdateMaskThingRequest) ApiMaskThingsServiceUpdateMaskThingRequest {
	r.body = &updateMaskThingRequest
	return r
}

func (r ApiMaskThingsServiceUpdateMaskThingRequest) UpdateMask(updateMask string) ApiMaskThingsServiceUpdateMaskThingRequest {
	r.updateMask = &updateMask
	return r
}

func (r ApiMaskThingsServiceUpdateMaskThingRequest) Execute() (*MaskThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiMaskThingsServiceDeleteMaskThingRequest struct {
	ctx context.Context
	id  string
}

func (a *MaskThingsServiceAPIService) MaskThingsServiceDeleteMaskThing(ctx context.Context, id string) ApiMaskThingsServiceDeleteMaskThingRequest {
	return ApiMaskThingsServiceDeleteMaskThingRequest{ctx: ctx, id: id}
}

func (r ApiMaskThingsServiceDeleteMaskThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
