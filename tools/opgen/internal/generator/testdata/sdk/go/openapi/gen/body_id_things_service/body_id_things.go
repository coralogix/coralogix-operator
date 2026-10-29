// Package body_id_things_service is a small stand-in for an SDK package that openapi-generator writes.
package body_id_things_service

import (
	"context"
	"errors"
	"net/http"
)

type BodyIDThing struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

type CreateBodyIDThingRequest struct {
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

// BodyIDThing1 is the Replace body. openapi-generator adds 1 because a component has its title.
type BodyIDThing1 struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type BodyIDThingsServiceAPIService struct{}

func NewBodyIDThingsServiceAPIService() *BodyIDThingsServiceAPIService {
	return &BodyIDThingsServiceAPIService{}
}

type ApiBodyIDThingsServiceCreateBodyIDThingRequest struct {
	ctx  context.Context
	body *CreateBodyIDThingRequest
}

func (a *BodyIDThingsServiceAPIService) BodyIDThingsServiceCreateBodyIDThing(ctx context.Context) ApiBodyIDThingsServiceCreateBodyIDThingRequest {
	return ApiBodyIDThingsServiceCreateBodyIDThingRequest{ctx: ctx}
}

func (r ApiBodyIDThingsServiceCreateBodyIDThingRequest) CreateBodyIDThingRequest(createBodyIDThingRequest CreateBodyIDThingRequest) ApiBodyIDThingsServiceCreateBodyIDThingRequest {
	r.body = &createBodyIDThingRequest
	return r
}

func (r ApiBodyIDThingsServiceCreateBodyIDThingRequest) Execute() (*BodyIDThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiBodyIDThingsServiceGetBodyIDThingRequest struct {
	ctx context.Context
	id  string
}

func (a *BodyIDThingsServiceAPIService) BodyIDThingsServiceGetBodyIDThing(ctx context.Context, id string) ApiBodyIDThingsServiceGetBodyIDThingRequest {
	return ApiBodyIDThingsServiceGetBodyIDThingRequest{ctx: ctx, id: id}
}

func (r ApiBodyIDThingsServiceGetBodyIDThingRequest) Execute() (*BodyIDThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiBodyIDThingsServiceReplaceBodyIDThingRequest struct {
	ctx  context.Context
	body *BodyIDThing1
}

func (a *BodyIDThingsServiceAPIService) BodyIDThingsServiceReplaceBodyIDThing(ctx context.Context) ApiBodyIDThingsServiceReplaceBodyIDThingRequest {
	return ApiBodyIDThingsServiceReplaceBodyIDThingRequest{ctx: ctx}
}

func (r ApiBodyIDThingsServiceReplaceBodyIDThingRequest) BodyIDThing1(bodyIDThing1 BodyIDThing1) ApiBodyIDThingsServiceReplaceBodyIDThingRequest {
	r.body = &bodyIDThing1
	return r
}

func (r ApiBodyIDThingsServiceReplaceBodyIDThingRequest) Execute() (*BodyIDThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiBodyIDThingsServiceDeleteBodyIDThingRequest struct {
	ctx context.Context
	id  string
}

func (a *BodyIDThingsServiceAPIService) BodyIDThingsServiceDeleteBodyIDThing(ctx context.Context, id string) ApiBodyIDThingsServiceDeleteBodyIDThingRequest {
	return ApiBodyIDThingsServiceDeleteBodyIDThingRequest{ctx: ctx, id: id}
}

func (r ApiBodyIDThingsServiceDeleteBodyIDThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
