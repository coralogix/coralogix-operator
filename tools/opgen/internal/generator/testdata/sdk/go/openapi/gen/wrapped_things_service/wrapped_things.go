// Package wrapped_things_service is a stand-in for the generated SDK.
package wrapped_things_service

import (
	"context"
	"errors"
	"net/http"
)

type WrappedThing struct {
	Id   *string `json:"id,omitempty"`
	Name string  `json:"name"`
}

type WrappedThingResponse struct {
	Item *WrappedThing `json:"item,omitempty"`
}

type CreateWrappedThingRequest struct {
	Item      *WrappedThing `json:"item,omitempty"`
	Label     *string       `json:"label,omitempty"`
	RequestId string        `json:"requestId,omitempty"`
}

type ReplaceWrappedThingRequest struct {
	Item      *WrappedThing `json:"item,omitempty"`
	Id        *string       `json:"id,omitempty"`
	Label     *string       `json:"label,omitempty"`
	RequestId string        `json:"requestId,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type WrappedThingsServiceAPIService struct{}

func NewWrappedThingsServiceAPIService() *WrappedThingsServiceAPIService {
	return &WrappedThingsServiceAPIService{}
}

type ApiWrappedThingsServiceCreateWrappedThingRequest struct {
	body *CreateWrappedThingRequest
}

func (a *WrappedThingsServiceAPIService) WrappedThingsServiceCreateWrappedThing(ctx context.Context) ApiWrappedThingsServiceCreateWrappedThingRequest {
	return ApiWrappedThingsServiceCreateWrappedThingRequest{}
}

func (r ApiWrappedThingsServiceCreateWrappedThingRequest) CreateWrappedThingRequest(createWrappedThingRequest CreateWrappedThingRequest) ApiWrappedThingsServiceCreateWrappedThingRequest {
	r.body = &createWrappedThingRequest
	return r
}

func (r ApiWrappedThingsServiceCreateWrappedThingRequest) Execute() (*WrappedThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiWrappedThingsServiceGetWrappedThingRequest struct{}

func (a *WrappedThingsServiceAPIService) WrappedThingsServiceGetWrappedThing(ctx context.Context, itemId string) ApiWrappedThingsServiceGetWrappedThingRequest {
	return ApiWrappedThingsServiceGetWrappedThingRequest{}
}

func (r ApiWrappedThingsServiceGetWrappedThingRequest) Execute() (*WrappedThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiWrappedThingsServiceReplaceWrappedThingRequest struct {
	body *ReplaceWrappedThingRequest
}

func (a *WrappedThingsServiceAPIService) WrappedThingsServiceReplaceWrappedThing(ctx context.Context) ApiWrappedThingsServiceReplaceWrappedThingRequest {
	return ApiWrappedThingsServiceReplaceWrappedThingRequest{}
}

func (r ApiWrappedThingsServiceReplaceWrappedThingRequest) ReplaceWrappedThingRequest(replaceWrappedThingRequest ReplaceWrappedThingRequest) ApiWrappedThingsServiceReplaceWrappedThingRequest {
	r.body = &replaceWrappedThingRequest
	return r
}

func (r ApiWrappedThingsServiceReplaceWrappedThingRequest) Execute() (*WrappedThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiWrappedThingsServiceDeleteWrappedThingRequest struct{}

func (a *WrappedThingsServiceAPIService) WrappedThingsServiceDeleteWrappedThing(ctx context.Context, itemId string) ApiWrappedThingsServiceDeleteWrappedThingRequest {
	return ApiWrappedThingsServiceDeleteWrappedThingRequest{}
}

func (r ApiWrappedThingsServiceDeleteWrappedThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
