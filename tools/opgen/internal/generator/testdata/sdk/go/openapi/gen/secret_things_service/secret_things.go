// Package secret_things_service is a stand-in for the generated SDK.
package secret_things_service

import (
	"context"
	"errors"
	"net/http"
)

type SecretThing struct {
	Id       *string             `json:"id,omitempty"`
	Name     string              `json:"name"`
	ApiKey   string              `json:"apiKey"`
	Token    *string             `json:"token,omitempty"`
	Settings SecretThingSettings `json:"settings"`
}

type SecretThingSettings struct {
	Fields []SecretThingField `json:"fields,omitempty"`
}

type SecretThingField struct {
	FieldName string  `json:"fieldName"`
	Value     *string `json:"value,omitempty"`
}

type SecretThingResponse struct {
	Item *SecretThing `json:"item,omitempty"`
}

type CreateSecretThingRequest struct {
	Item SecretThing `json:"item"`
}

type ReplaceSecretThingRequest struct {
	Item SecretThing `json:"item"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type SecretThingsServiceAPIService struct{}

func NewSecretThingsServiceAPIService() *SecretThingsServiceAPIService {
	return &SecretThingsServiceAPIService{}
}

type ApiSecretThingsServiceCreateSecretThingRequest struct {
	body *CreateSecretThingRequest
}

func (a *SecretThingsServiceAPIService) SecretThingsServiceCreateSecretThing(ctx context.Context) ApiSecretThingsServiceCreateSecretThingRequest {
	return ApiSecretThingsServiceCreateSecretThingRequest{}
}

func (r ApiSecretThingsServiceCreateSecretThingRequest) CreateSecretThingRequest(createSecretThingRequest CreateSecretThingRequest) ApiSecretThingsServiceCreateSecretThingRequest {
	r.body = &createSecretThingRequest
	return r
}

func (r ApiSecretThingsServiceCreateSecretThingRequest) Execute() (*SecretThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiSecretThingsServiceGetSecretThingRequest struct{}

func (a *SecretThingsServiceAPIService) SecretThingsServiceGetSecretThing(ctx context.Context, itemId string) ApiSecretThingsServiceGetSecretThingRequest {
	return ApiSecretThingsServiceGetSecretThingRequest{}
}

func (r ApiSecretThingsServiceGetSecretThingRequest) Execute() (*SecretThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiSecretThingsServiceReplaceSecretThingRequest struct {
	body *ReplaceSecretThingRequest
}

func (a *SecretThingsServiceAPIService) SecretThingsServiceReplaceSecretThing(ctx context.Context) ApiSecretThingsServiceReplaceSecretThingRequest {
	return ApiSecretThingsServiceReplaceSecretThingRequest{}
}

func (r ApiSecretThingsServiceReplaceSecretThingRequest) ReplaceSecretThingRequest(replaceSecretThingRequest ReplaceSecretThingRequest) ApiSecretThingsServiceReplaceSecretThingRequest {
	r.body = &replaceSecretThingRequest
	return r
}

func (r ApiSecretThingsServiceReplaceSecretThingRequest) Execute() (*SecretThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiSecretThingsServiceDeleteSecretThingRequest struct{}

func (a *SecretThingsServiceAPIService) SecretThingsServiceDeleteSecretThing(ctx context.Context, itemId string) ApiSecretThingsServiceDeleteSecretThingRequest {
	return ApiSecretThingsServiceDeleteSecretThingRequest{}
}

func (r ApiSecretThingsServiceDeleteSecretThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
