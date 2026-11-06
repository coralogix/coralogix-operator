// Package hook_delete_things_service is a small stand-in for an SDK package that openapi-generator writes.
package hook_delete_things_service

import (
	"context"
	"errors"
	"net/http"
)

type HookDeleteThing struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

type CreateHookDeleteThingRequest struct {
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

type ReplaceHookDeleteThingRequest struct {
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type HookDeleteThingsServiceAPIService struct{}

func NewHookDeleteThingsServiceAPIService() *HookDeleteThingsServiceAPIService {
	return &HookDeleteThingsServiceAPIService{}
}

type ApiHookDeleteThingsServiceCreateHookDeleteThingRequest struct {
	ctx  context.Context
	body *CreateHookDeleteThingRequest
}

func (a *HookDeleteThingsServiceAPIService) HookDeleteThingsServiceCreateHookDeleteThing(ctx context.Context) ApiHookDeleteThingsServiceCreateHookDeleteThingRequest {
	return ApiHookDeleteThingsServiceCreateHookDeleteThingRequest{ctx: ctx}
}

func (r ApiHookDeleteThingsServiceCreateHookDeleteThingRequest) CreateHookDeleteThingRequest(createHookDeleteThingRequest CreateHookDeleteThingRequest) ApiHookDeleteThingsServiceCreateHookDeleteThingRequest {
	r.body = &createHookDeleteThingRequest
	return r
}

func (r ApiHookDeleteThingsServiceCreateHookDeleteThingRequest) Execute() (*HookDeleteThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiHookDeleteThingsServiceGetHookDeleteThingRequest struct {
	ctx context.Context
	id  string
}

func (a *HookDeleteThingsServiceAPIService) HookDeleteThingsServiceGetHookDeleteThing(ctx context.Context, id string) ApiHookDeleteThingsServiceGetHookDeleteThingRequest {
	return ApiHookDeleteThingsServiceGetHookDeleteThingRequest{ctx: ctx, id: id}
}

func (r ApiHookDeleteThingsServiceGetHookDeleteThingRequest) Execute() (*HookDeleteThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiHookDeleteThingsServiceReplaceHookDeleteThingRequest struct {
	ctx  context.Context
	id   string
	body *ReplaceHookDeleteThingRequest
}

func (a *HookDeleteThingsServiceAPIService) HookDeleteThingsServiceReplaceHookDeleteThing(ctx context.Context, id string) ApiHookDeleteThingsServiceReplaceHookDeleteThingRequest {
	return ApiHookDeleteThingsServiceReplaceHookDeleteThingRequest{ctx: ctx, id: id}
}

func (r ApiHookDeleteThingsServiceReplaceHookDeleteThingRequest) ReplaceHookDeleteThingRequest(replaceHookDeleteThingRequest ReplaceHookDeleteThingRequest) ApiHookDeleteThingsServiceReplaceHookDeleteThingRequest {
	r.body = &replaceHookDeleteThingRequest
	return r
}

func (r ApiHookDeleteThingsServiceReplaceHookDeleteThingRequest) Execute() (*HookDeleteThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiHookDeleteThingsServiceDeleteHookDeleteThingRequest struct {
	ctx context.Context
	id  string
}

func (a *HookDeleteThingsServiceAPIService) HookDeleteThingsServiceDeleteHookDeleteThing(ctx context.Context, id string) ApiHookDeleteThingsServiceDeleteHookDeleteThingRequest {
	return ApiHookDeleteThingsServiceDeleteHookDeleteThingRequest{ctx: ctx, id: id}
}

func (r ApiHookDeleteThingsServiceDeleteHookDeleteThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
