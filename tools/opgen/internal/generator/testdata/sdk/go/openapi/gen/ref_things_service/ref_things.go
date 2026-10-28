// Package ref_things_service is a small stand-in for an SDK package that openapi-generator writes.
package ref_things_service

import (
	"context"
	"errors"
	"net/http"
)

type RefThing struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	ThingId  string  `json:"thingId"`
	ParentId *string `json:"parentId,omitempty"`
}

type CreateRefThingRequest struct {
	Name     string  `json:"name"`
	ThingId  string  `json:"thingId"`
	ParentId *string `json:"parentId,omitempty"`
}

type ReplaceRefThingRequest struct {
	Name     string  `json:"name"`
	ThingId  string  `json:"thingId"`
	ParentId *string `json:"parentId,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type RefThingsServiceAPIService struct{}

func NewRefThingsServiceAPIService() *RefThingsServiceAPIService {
	return &RefThingsServiceAPIService{}
}

type ApiRefThingsServiceCreateRefThingRequest struct {
	ctx  context.Context
	body *CreateRefThingRequest
}

func (a *RefThingsServiceAPIService) RefThingsServiceCreateRefThing(ctx context.Context) ApiRefThingsServiceCreateRefThingRequest {
	return ApiRefThingsServiceCreateRefThingRequest{ctx: ctx}
}

func (r ApiRefThingsServiceCreateRefThingRequest) CreateRefThingRequest(createRefThingRequest CreateRefThingRequest) ApiRefThingsServiceCreateRefThingRequest {
	r.body = &createRefThingRequest
	return r
}

func (r ApiRefThingsServiceCreateRefThingRequest) Execute() (*RefThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiRefThingsServiceGetRefThingRequest struct {
	ctx context.Context
	id  string
}

func (a *RefThingsServiceAPIService) RefThingsServiceGetRefThing(ctx context.Context, id string) ApiRefThingsServiceGetRefThingRequest {
	return ApiRefThingsServiceGetRefThingRequest{ctx: ctx, id: id}
}

func (r ApiRefThingsServiceGetRefThingRequest) Execute() (*RefThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiRefThingsServiceReplaceRefThingRequest struct {
	ctx  context.Context
	id   string
	body *ReplaceRefThingRequest
}

func (a *RefThingsServiceAPIService) RefThingsServiceReplaceRefThing(ctx context.Context, id string) ApiRefThingsServiceReplaceRefThingRequest {
	return ApiRefThingsServiceReplaceRefThingRequest{ctx: ctx, id: id}
}

func (r ApiRefThingsServiceReplaceRefThingRequest) ReplaceRefThingRequest(replaceRefThingRequest ReplaceRefThingRequest) ApiRefThingsServiceReplaceRefThingRequest {
	r.body = &replaceRefThingRequest
	return r
}

func (r ApiRefThingsServiceReplaceRefThingRequest) Execute() (*RefThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiRefThingsServiceDeleteRefThingRequest struct {
	ctx context.Context
	id  string
}

func (a *RefThingsServiceAPIService) RefThingsServiceDeleteRefThing(ctx context.Context, id string) ApiRefThingsServiceDeleteRefThingRequest {
	return ApiRefThingsServiceDeleteRefThingRequest{ctx: ctx, id: id}
}

func (r ApiRefThingsServiceDeleteRefThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
