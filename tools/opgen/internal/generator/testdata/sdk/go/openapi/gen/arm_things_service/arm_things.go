// Package arm_things_service is a small stand-in for an SDK package that openapi-generator writes.
package arm_things_service

import (
	"context"
	"errors"
	"net/http"
)

type ArmThingHook struct {
	Token   string            `json:"token"`
	Note    *string           `json:"note,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type ArmThing struct {
	Id    string                 `json:"id"`
	Name  string                 `json:"name"`
	Url   *string                `json:"url,omitempty"`
	Hook  *ArmThingHook          `json:"hook,omitempty"`
	Quiet map[string]interface{} `json:"quiet,omitempty"`
}

type CreateArmThingRequest struct {
	Name  string                 `json:"name"`
	Url   *string                `json:"url,omitempty"`
	Hook  *ArmThingHook          `json:"hook,omitempty"`
	Quiet map[string]interface{} `json:"quiet,omitempty"`
}

// ArmThing1 is the Replace body. openapi-generator adds 1 because a component has its title.
type ArmThing1 struct {
	Id    string                 `json:"id"`
	Name  string                 `json:"name"`
	Url   *string                `json:"url,omitempty"`
	Hook  *ArmThingHook          `json:"hook,omitempty"`
	Quiet map[string]interface{} `json:"quiet,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type ArmThingsServiceAPIService struct{}

func NewArmThingsServiceAPIService() *ArmThingsServiceAPIService {
	return &ArmThingsServiceAPIService{}
}

type ApiArmThingsServiceCreateArmThingRequest struct {
	ctx  context.Context
	body *CreateArmThingRequest
}

func (a *ArmThingsServiceAPIService) ArmThingsServiceCreateArmThing(ctx context.Context) ApiArmThingsServiceCreateArmThingRequest {
	return ApiArmThingsServiceCreateArmThingRequest{ctx: ctx}
}

func (r ApiArmThingsServiceCreateArmThingRequest) CreateArmThingRequest(createArmThingRequest CreateArmThingRequest) ApiArmThingsServiceCreateArmThingRequest {
	r.body = &createArmThingRequest
	return r
}

func (r ApiArmThingsServiceCreateArmThingRequest) Execute() (*ArmThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiArmThingsServiceGetArmThingRequest struct {
	ctx context.Context
	id  string
}

func (a *ArmThingsServiceAPIService) ArmThingsServiceGetArmThing(ctx context.Context, id string) ApiArmThingsServiceGetArmThingRequest {
	return ApiArmThingsServiceGetArmThingRequest{ctx: ctx, id: id}
}

func (r ApiArmThingsServiceGetArmThingRequest) Execute() (*ArmThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiArmThingsServiceReplaceArmThingRequest struct {
	ctx  context.Context
	body *ArmThing1
}

func (a *ArmThingsServiceAPIService) ArmThingsServiceReplaceArmThing(ctx context.Context) ApiArmThingsServiceReplaceArmThingRequest {
	return ApiArmThingsServiceReplaceArmThingRequest{ctx: ctx}
}

func (r ApiArmThingsServiceReplaceArmThingRequest) ArmThing1(armThing1 ArmThing1) ApiArmThingsServiceReplaceArmThingRequest {
	r.body = &armThing1
	return r
}

func (r ApiArmThingsServiceReplaceArmThingRequest) Execute() (*ArmThing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiArmThingsServiceDeleteArmThingRequest struct {
	ctx context.Context
	id  string
}

func (a *ArmThingsServiceAPIService) ArmThingsServiceDeleteArmThing(ctx context.Context, id string) ApiArmThingsServiceDeleteArmThingRequest {
	return ApiArmThingsServiceDeleteArmThingRequest{ctx: ctx, id: id}
}

func (r ApiArmThingsServiceDeleteArmThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
