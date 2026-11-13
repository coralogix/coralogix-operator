// Package content_things_service is a stand-in for the generated SDK.
package content_things_service

import (
	"context"
	"errors"
	"net/http"
)

type ContentThing struct {
	Id     *string            `json:"id,omitempty"`
	Name   string             `json:"name"`
	Layout ContentThingLayout `json:"layout"`
	Mode   *string            `json:"mode,omitempty"`
}

type ContentThingLayout struct {
	Rows    *int32 `json:"rows,omitempty"`
	Columns *int32 `json:"columns,omitempty"`
}

type CreateContentThingRequest struct {
	Item      ContentThing `json:"item"`
	Policy    *string      `json:"policy,omitempty"`
	Locked    *bool        `json:"locked,omitempty"`
	RequestId string       `json:"requestId"`
}

type ReplaceContentThingRequest struct {
	Item      *ContentThing `json:"item,omitempty"`
	Policy    *string       `json:"policy,omitempty"`
	Locked    *bool         `json:"locked,omitempty"`
	RequestId string        `json:"requestId"`
}

type CreateContentThingResponse struct {
	ThingId *string `json:"thingId,omitempty"`
}

type GetContentThingResponse struct {
	Item   *ContentThing `json:"item,omitempty"`
	Policy *string       `json:"policy,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type ContentThingsServiceAPIService struct{}

func NewContentThingsServiceAPIService() *ContentThingsServiceAPIService {
	return &ContentThingsServiceAPIService{}
}

type ApiContentThingsServiceCreateContentThingRequest struct {
	body *CreateContentThingRequest
}

func (a *ContentThingsServiceAPIService) ContentThingsServiceCreateContentThing(ctx context.Context) ApiContentThingsServiceCreateContentThingRequest {
	return ApiContentThingsServiceCreateContentThingRequest{}
}

func (r ApiContentThingsServiceCreateContentThingRequest) CreateContentThingRequest(createContentThingRequest CreateContentThingRequest) ApiContentThingsServiceCreateContentThingRequest {
	r.body = &createContentThingRequest
	return r
}

func (r ApiContentThingsServiceCreateContentThingRequest) Execute() (*CreateContentThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiContentThingsServiceGetContentThingRequest struct{}

func (a *ContentThingsServiceAPIService) ContentThingsServiceGetContentThing(ctx context.Context, thingId string) ApiContentThingsServiceGetContentThingRequest {
	return ApiContentThingsServiceGetContentThingRequest{}
}

func (r ApiContentThingsServiceGetContentThingRequest) Execute() (*GetContentThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiContentThingsServiceReplaceContentThingRequest struct {
	body *ReplaceContentThingRequest
}

func (a *ContentThingsServiceAPIService) ContentThingsServiceReplaceContentThing(ctx context.Context) ApiContentThingsServiceReplaceContentThingRequest {
	return ApiContentThingsServiceReplaceContentThingRequest{}
}

func (r ApiContentThingsServiceReplaceContentThingRequest) ReplaceContentThingRequest(replaceContentThingRequest ReplaceContentThingRequest) ApiContentThingsServiceReplaceContentThingRequest {
	r.body = &replaceContentThingRequest
	return r
}

func (r ApiContentThingsServiceReplaceContentThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}

type ApiContentThingsServiceDeleteContentThingRequest struct{}

func (a *ContentThingsServiceAPIService) ContentThingsServiceDeleteContentThing(ctx context.Context, thingId string) ApiContentThingsServiceDeleteContentThingRequest {
	return ApiContentThingsServiceDeleteContentThingRequest{}
}

func (r ApiContentThingsServiceDeleteContentThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
