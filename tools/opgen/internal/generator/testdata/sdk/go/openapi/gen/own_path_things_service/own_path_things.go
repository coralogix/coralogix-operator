// Package own_path_things_service is a small stand-in for an SDK package that openapi-generator writes.
package own_path_things_service

import (
	"context"
	"errors"
	"net/http"
)

type OwnPathThing struct {
	ThingId *string `json:"thingId,omitempty"`
	Name    string  `json:"name"`
	Note    *string `json:"note,omitempty"`
}

type CreateOwnPathThingRequest struct {
	Name string  `json:"name"`
	Note *string `json:"note,omitempty"`
}

type UpdateOwnPathThingRequest struct {
	Name *string `json:"name,omitempty"`
	Note *string `json:"note,omitempty"`
}

type GetOwnPathThingResponse struct {
	Thing *OwnPathThing `json:"thing,omitempty"`
}

type CreateOwnPathThingResponse struct {
	Thing *OwnPathThing `json:"thing,omitempty"`
}

type UpdateOwnPathThingResponse struct {
	Thing *OwnPathThing `json:"thing,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type OwnPathThingsServiceAPIService struct{}

func NewOwnPathThingsServiceAPIService() *OwnPathThingsServiceAPIService {
	return &OwnPathThingsServiceAPIService{}
}

type ApiOwnPathThingsServiceCreateOwnPathThingRequest struct {
	ctx  context.Context
	body *CreateOwnPathThingRequest
}

func (a *OwnPathThingsServiceAPIService) OwnPathThingsServiceCreateOwnPathThing(ctx context.Context) ApiOwnPathThingsServiceCreateOwnPathThingRequest {
	return ApiOwnPathThingsServiceCreateOwnPathThingRequest{ctx: ctx}
}

func (r ApiOwnPathThingsServiceCreateOwnPathThingRequest) CreateOwnPathThingRequest(createOwnPathThingRequest CreateOwnPathThingRequest) ApiOwnPathThingsServiceCreateOwnPathThingRequest {
	r.body = &createOwnPathThingRequest
	return r
}

func (r ApiOwnPathThingsServiceCreateOwnPathThingRequest) Execute() (*CreateOwnPathThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiOwnPathThingsServiceGetOwnPathThingRequest struct {
	ctx     context.Context
	thingId string
}

func (a *OwnPathThingsServiceAPIService) OwnPathThingsServiceGetOwnPathThing(ctx context.Context, thingId string) ApiOwnPathThingsServiceGetOwnPathThingRequest {
	return ApiOwnPathThingsServiceGetOwnPathThingRequest{ctx: ctx, thingId: thingId}
}

func (r ApiOwnPathThingsServiceGetOwnPathThingRequest) Execute() (*GetOwnPathThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiOwnPathThingsServiceUpdateOwnPathThingRequest struct {
	ctx     context.Context
	thingId string
	body    *UpdateOwnPathThingRequest
}

func (a *OwnPathThingsServiceAPIService) OwnPathThingsServiceUpdateOwnPathThing(ctx context.Context, thingId string) ApiOwnPathThingsServiceUpdateOwnPathThingRequest {
	return ApiOwnPathThingsServiceUpdateOwnPathThingRequest{ctx: ctx, thingId: thingId}
}

func (r ApiOwnPathThingsServiceUpdateOwnPathThingRequest) UpdateOwnPathThingRequest(updateOwnPathThingRequest UpdateOwnPathThingRequest) ApiOwnPathThingsServiceUpdateOwnPathThingRequest {
	r.body = &updateOwnPathThingRequest
	return r
}

func (r ApiOwnPathThingsServiceUpdateOwnPathThingRequest) Execute() (*UpdateOwnPathThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiOwnPathThingsServiceDeleteOwnPathThingRequest struct {
	ctx     context.Context
	thingId string
}

func (a *OwnPathThingsServiceAPIService) OwnPathThingsServiceDeleteOwnPathThing(ctx context.Context, thingId string) ApiOwnPathThingsServiceDeleteOwnPathThingRequest {
	return ApiOwnPathThingsServiceDeleteOwnPathThingRequest{ctx: ctx, thingId: thingId}
}

func (r ApiOwnPathThingsServiceDeleteOwnPathThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
