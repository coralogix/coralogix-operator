// Package batch_things_service is a small stand-in for an SDK package that openapi-generator writes.
package batch_things_service

import (
	"context"
	"errors"
	"net/http"
)

type BatchThing struct {
	Id   string  `json:"id"`
	Name string  `json:"name"`
	Note *string `json:"note,omitempty"`
}

type CreateBatchThingRequest struct {
	Name string  `json:"name"`
	Note *string `json:"note,omitempty"`
}

type ReplaceBatchThingRequest struct {
	Id   string  `json:"id"`
	Name string  `json:"name"`
	Note *string `json:"note,omitempty"`
}

type GetBatchThingsResponse struct {
	Things []BatchThing `json:"things"`
}

type CreateBatchThingResponse struct {
	Thing BatchThing `json:"thing"`
}

type ReplaceBatchThingResponse struct {
	Thing BatchThing `json:"thing"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type BatchThingsServiceAPIService struct{}

func NewBatchThingsServiceAPIService() *BatchThingsServiceAPIService {
	return &BatchThingsServiceAPIService{}
}

type ApiBatchThingsServiceCreateBatchThingRequest struct {
	ctx  context.Context
	body *CreateBatchThingRequest
}

func (a *BatchThingsServiceAPIService) BatchThingsServiceCreateBatchThing(ctx context.Context) ApiBatchThingsServiceCreateBatchThingRequest {
	return ApiBatchThingsServiceCreateBatchThingRequest{ctx: ctx}
}

func (r ApiBatchThingsServiceCreateBatchThingRequest) CreateBatchThingRequest(createBatchThingRequest CreateBatchThingRequest) ApiBatchThingsServiceCreateBatchThingRequest {
	r.body = &createBatchThingRequest
	return r
}

func (r ApiBatchThingsServiceCreateBatchThingRequest) Execute() (*CreateBatchThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiBatchThingsServiceGetBatchThingsByIdsRequest struct {
	ctx context.Context
	ids *[]string
}

func (a *BatchThingsServiceAPIService) BatchThingsServiceGetBatchThingsByIds(ctx context.Context) ApiBatchThingsServiceGetBatchThingsByIdsRequest {
	return ApiBatchThingsServiceGetBatchThingsByIdsRequest{ctx: ctx}
}

func (r ApiBatchThingsServiceGetBatchThingsByIdsRequest) Ids(ids []string) ApiBatchThingsServiceGetBatchThingsByIdsRequest {
	r.ids = &ids
	return r
}

func (r ApiBatchThingsServiceGetBatchThingsByIdsRequest) Execute() (*GetBatchThingsResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiBatchThingsServiceReplaceBatchThingRequest struct {
	ctx  context.Context
	body *ReplaceBatchThingRequest
}

func (a *BatchThingsServiceAPIService) BatchThingsServiceReplaceBatchThing(ctx context.Context) ApiBatchThingsServiceReplaceBatchThingRequest {
	return ApiBatchThingsServiceReplaceBatchThingRequest{ctx: ctx}
}

func (r ApiBatchThingsServiceReplaceBatchThingRequest) ReplaceBatchThingRequest(replaceBatchThingRequest ReplaceBatchThingRequest) ApiBatchThingsServiceReplaceBatchThingRequest {
	r.body = &replaceBatchThingRequest
	return r
}

func (r ApiBatchThingsServiceReplaceBatchThingRequest) Execute() (*ReplaceBatchThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiBatchThingsServiceDeleteBatchThingRequest struct {
	ctx context.Context
	id  string
}

func (a *BatchThingsServiceAPIService) BatchThingsServiceDeleteBatchThing(ctx context.Context, id string) ApiBatchThingsServiceDeleteBatchThingRequest {
	return ApiBatchThingsServiceDeleteBatchThingRequest{ctx: ctx, id: id}
}

func (r ApiBatchThingsServiceDeleteBatchThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
