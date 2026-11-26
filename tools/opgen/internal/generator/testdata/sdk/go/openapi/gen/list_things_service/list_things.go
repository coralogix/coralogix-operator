// Package list_things_service is a small stand-in for an SDK package that openapi-generator writes.
package list_things_service

import (
	"context"
	"errors"
	"net/http"
)

type ListThing struct {
	Id       *string            `json:"id,omitempty"`
	Name     *string            `json:"name,omitempty"`
	Settings *ListThingSettings `json:"settings,omitempty"`
}

type ListThingSettings struct {
	Note  *string `json:"note,omitempty"`
	Level *int64  `json:"level,omitempty"`
}

type CreateListThingRequest struct {
	Name  *string `json:"name,omitempty"`
	Note  *string `json:"note,omitempty"`
	Level *int64  `json:"level,omitempty"`
}

type UpdateListThingRequest struct {
	Name       *string `json:"name,omitempty"`
	Note       *string `json:"note,omitempty"`
	Level      *int64  `json:"level,omitempty"`
	UpdateMask *string `json:"updateMask,omitempty"`
}

type GetListThingsResponse struct {
	Items []ListThing `json:"items,omitempty"`
}

type CreateListThingResponse struct {
	Item *ListThing `json:"item,omitempty"`
}

type UpdateListThingResponse struct {
	Item *ListThing `json:"item,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type ListThingsServiceAPIService struct{}

func NewListThingsServiceAPIService() *ListThingsServiceAPIService {
	return &ListThingsServiceAPIService{}
}

type ApiListThingsServiceGetListThingsRequest struct {
	ctx context.Context
}

func (a *ListThingsServiceAPIService) ListThingsServiceGetListThings(ctx context.Context) ApiListThingsServiceGetListThingsRequest {
	return ApiListThingsServiceGetListThingsRequest{ctx: ctx}
}

func (r ApiListThingsServiceGetListThingsRequest) Execute() (*GetListThingsResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiListThingsServiceCreateListThingRequest struct {
	ctx  context.Context
	body *CreateListThingRequest
}

func (a *ListThingsServiceAPIService) ListThingsServiceCreateListThing(ctx context.Context) ApiListThingsServiceCreateListThingRequest {
	return ApiListThingsServiceCreateListThingRequest{ctx: ctx}
}

func (r ApiListThingsServiceCreateListThingRequest) CreateListThingRequest(createListThingRequest CreateListThingRequest) ApiListThingsServiceCreateListThingRequest {
	r.body = &createListThingRequest
	return r
}

func (r ApiListThingsServiceCreateListThingRequest) Execute() (*CreateListThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiListThingsServiceUpdateListThingRequest struct {
	ctx  context.Context
	id   string
	body *UpdateListThingRequest
}

func (a *ListThingsServiceAPIService) ListThingsServiceUpdateListThing(ctx context.Context, id string) ApiListThingsServiceUpdateListThingRequest {
	return ApiListThingsServiceUpdateListThingRequest{ctx: ctx, id: id}
}

func (r ApiListThingsServiceUpdateListThingRequest) UpdateListThingRequest(updateListThingRequest UpdateListThingRequest) ApiListThingsServiceUpdateListThingRequest {
	r.body = &updateListThingRequest
	return r
}

func (r ApiListThingsServiceUpdateListThingRequest) Execute() (*UpdateListThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiListThingsServiceDeleteListThingRequest struct {
	ctx context.Context
	id  string
}

func (a *ListThingsServiceAPIService) ListThingsServiceDeleteListThing(ctx context.Context, id string) ApiListThingsServiceDeleteListThingRequest {
	return ApiListThingsServiceDeleteListThingRequest{ctx: ctx, id: id}
}

func (r ApiListThingsServiceDeleteListThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
