// Package hook_things_service is a small stand-in for an SDK package that openapi-generator writes.
package hook_things_service

import (
	"context"
	"errors"
	"net/http"
)

type HookThing struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

type CreateHookThingRequest struct {
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

type ReplaceHookThingRequest struct {
	Name  string `json:"name"`
	Limit *int32 `json:"limit,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type HookThingsServiceAPIService struct{}

func NewHookThingsServiceAPIService() *HookThingsServiceAPIService {
	return &HookThingsServiceAPIService{}
}

type ApiHookThingsServiceCreateHookThingRequest struct {
	ctx  context.Context
	body *CreateHookThingRequest
}

func (a *HookThingsServiceAPIService) HookThingsServiceCreateHookThing(ctx context.Context) ApiHookThingsServiceCreateHookThingRequest {
	return ApiHookThingsServiceCreateHookThingRequest{ctx: ctx}
}

func (r ApiHookThingsServiceCreateHookThingRequest) CreateHookThingRequest(createHookThingRequest CreateHookThingRequest) ApiHookThingsServiceCreateHookThingRequest {
	r.body = &createHookThingRequest
	return r
}

func (r ApiHookThingsServiceCreateHookThingRequest) Execute() (*HookThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiHookThingsServiceGetHookThingRequest struct {
	ctx context.Context
	id  string
}

func (a *HookThingsServiceAPIService) HookThingsServiceGetHookThing(ctx context.Context, id string) ApiHookThingsServiceGetHookThingRequest {
	return ApiHookThingsServiceGetHookThingRequest{ctx: ctx, id: id}
}

func (r ApiHookThingsServiceGetHookThingRequest) Execute() (*HookThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiHookThingsServiceReplaceHookThingRequest struct {
	ctx  context.Context
	id   string
	body *ReplaceHookThingRequest
}

func (a *HookThingsServiceAPIService) HookThingsServiceReplaceHookThing(ctx context.Context, id string) ApiHookThingsServiceReplaceHookThingRequest {
	return ApiHookThingsServiceReplaceHookThingRequest{ctx: ctx, id: id}
}

func (r ApiHookThingsServiceReplaceHookThingRequest) ReplaceHookThingRequest(replaceHookThingRequest ReplaceHookThingRequest) ApiHookThingsServiceReplaceHookThingRequest {
	r.body = &replaceHookThingRequest
	return r
}

func (r ApiHookThingsServiceReplaceHookThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}

type ApiHookThingsServiceDeleteHookThingRequest struct {
	ctx context.Context
	id  string
}

func (a *HookThingsServiceAPIService) HookThingsServiceDeleteHookThing(ctx context.Context, id string) ApiHookThingsServiceDeleteHookThingRequest {
	return ApiHookThingsServiceDeleteHookThingRequest{ctx: ctx, id: id}
}

func (r ApiHookThingsServiceDeleteHookThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
