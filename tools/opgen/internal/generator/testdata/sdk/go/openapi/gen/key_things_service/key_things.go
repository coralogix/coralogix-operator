// Package key_things_service is a small stand-in for an SDK package that openapi-generator writes.
package key_things_service

import (
	"context"
	"errors"
	"net/http"
)

type KeyThing struct {
	Id             string                 `json:"id"`
	Name           string                 `json:"name"`
	Note           *string                `json:"note,omitempty"`
	Enabled        bool                   `json:"enabled"`
	KeyPermissions KeyThingKeyPermissions `json:"keyPermissions"`
}

type KeyThingKeyPermissions struct {
	Presets     []KeyThingPreset `json:"presets"`
	Permissions []string         `json:"permissions"`
}

type KeyThingPreset struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type CreateKeyThingRequest struct {
	Name           string                              `json:"name"`
	Note           *string                             `json:"note,omitempty"`
	KeyPermissions CreateKeyThingRequestKeyPermissions `json:"keyPermissions"`
}

type CreateKeyThingRequestKeyPermissions struct {
	Presets     []string `json:"presets,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type UpdateKeyThingRequest struct {
	NewName     *string              `json:"newName,omitempty"`
	IsEnabled   *bool                `json:"isEnabled,omitempty"`
	Note        *string              `json:"note,omitempty"`
	Presets     *KeyThingPresets     `json:"presets,omitempty"`
	Permissions *KeyThingPermissions `json:"permissions,omitempty"`
}

type KeyThingPresets struct {
	Presets []string `json:"presets,omitempty"`
}

type KeyThingPermissions struct {
	Permissions []string `json:"permissions,omitempty"`
}

type CreateKeyThingResponse struct {
	KeyId string `json:"keyId"`
}

type GetKeyThingResponse struct {
	KeyInfo KeyThing `json:"keyInfo"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type KeyThingsServiceAPIService struct{}

func NewKeyThingsServiceAPIService() *KeyThingsServiceAPIService {
	return &KeyThingsServiceAPIService{}
}

type ApiKeyThingsServiceCreateKeyThingRequest struct {
	ctx  context.Context
	body *CreateKeyThingRequest
}

func (a *KeyThingsServiceAPIService) KeyThingsServiceCreateKeyThing(ctx context.Context) ApiKeyThingsServiceCreateKeyThingRequest {
	return ApiKeyThingsServiceCreateKeyThingRequest{ctx: ctx}
}

func (r ApiKeyThingsServiceCreateKeyThingRequest) CreateKeyThingRequest(createKeyThingRequest CreateKeyThingRequest) ApiKeyThingsServiceCreateKeyThingRequest {
	r.body = &createKeyThingRequest
	return r
}

func (r ApiKeyThingsServiceCreateKeyThingRequest) Execute() (*CreateKeyThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiKeyThingsServiceGetKeyThingRequest struct {
	ctx   context.Context
	keyId string
}

func (a *KeyThingsServiceAPIService) KeyThingsServiceGetKeyThing(ctx context.Context, keyId string) ApiKeyThingsServiceGetKeyThingRequest {
	return ApiKeyThingsServiceGetKeyThingRequest{ctx: ctx, keyId: keyId}
}

func (r ApiKeyThingsServiceGetKeyThingRequest) Execute() (*GetKeyThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiKeyThingsServiceUpdateKeyThingRequest struct {
	ctx   context.Context
	keyId string
	body  *UpdateKeyThingRequest
}

func (a *KeyThingsServiceAPIService) KeyThingsServiceUpdateKeyThing(ctx context.Context, keyId string) ApiKeyThingsServiceUpdateKeyThingRequest {
	return ApiKeyThingsServiceUpdateKeyThingRequest{ctx: ctx, keyId: keyId}
}

func (r ApiKeyThingsServiceUpdateKeyThingRequest) UpdateKeyThingRequest(updateKeyThingRequest UpdateKeyThingRequest) ApiKeyThingsServiceUpdateKeyThingRequest {
	r.body = &updateKeyThingRequest
	return r
}

func (r ApiKeyThingsServiceUpdateKeyThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}

type ApiKeyThingsServiceDeleteKeyThingRequest struct {
	ctx   context.Context
	keyId string
}

func (a *KeyThingsServiceAPIService) KeyThingsServiceDeleteKeyThing(ctx context.Context, keyId string) ApiKeyThingsServiceDeleteKeyThingRequest {
	return ApiKeyThingsServiceDeleteKeyThingRequest{ctx: ctx, keyId: keyId}
}

func (r ApiKeyThingsServiceDeleteKeyThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
