// Package setting_things_service is a small stand-in for an SDK package that openapi-generator writes.
package setting_things_service

import (
	"context"
	"errors"
	"net/http"
)

type GetSettingThingResponse struct {
	Setting SettingThing `json:"setting"`
}

type SetSettingThingResponse struct {
	Setting SettingThing `json:"setting"`
}

type SettingThing struct {
	Item      *SettingThingItem      `json:"item,omitempty"`
	OtherItem *SettingThingOtherItem `json:"otherItem,omitempty"`
	Spec      SettingThingSpec       `json:"spec"`
}

type SettingThingSpec struct {
	IsActive *bool              `json:"isActive,omitempty"`
	Level    *SettingThingLevel `json:"level,omitempty"`
}

type SettingThingItem struct {
	Bucket string  `json:"bucket"`
	Region *string `json:"region,omitempty"`
}

type SettingThingOtherItem struct {
	Crn *string `json:"crn,omitempty"`
}

type SettingThingLevel string

const (
	SETTINGTHINGLEVEL_SETTING_THING_LEVEL_UNSPECIFIED SettingThingLevel = "SETTING_THING_LEVEL_UNSPECIFIED"
	SETTINGTHINGLEVEL_SETTING_THING_LEVEL_LOW         SettingThingLevel = "SETTING_THING_LEVEL_LOW"
	SETTINGTHINGLEVEL_SETTING_THING_LEVEL_HIGH        SettingThingLevel = "SETTING_THING_LEVEL_HIGH"
)

type SetSettingThingRequest struct {
	IsActive *bool              `json:"isActive,omitempty"`
	Level    *SettingThingLevel `json:"level,omitempty"`
	Item     SettingThingItem   `json:"item"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type SettingThingsServiceAPIService struct{}

func NewSettingThingsServiceAPIService() *SettingThingsServiceAPIService {
	return &SettingThingsServiceAPIService{}
}

type ApiSettingThingsServiceGetSettingThingRequest struct {
	ctx context.Context
}

func (a *SettingThingsServiceAPIService) SettingThingsServiceGetSettingThing(ctx context.Context) ApiSettingThingsServiceGetSettingThingRequest {
	return ApiSettingThingsServiceGetSettingThingRequest{ctx: ctx}
}

func (r ApiSettingThingsServiceGetSettingThingRequest) Execute() (*GetSettingThingResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiSettingThingsServiceSetSettingThingRequest struct {
	ctx  context.Context
	body *SetSettingThingRequest
}

func (a *SettingThingsServiceAPIService) SettingThingsServiceSetSettingThing(ctx context.Context) ApiSettingThingsServiceSetSettingThingRequest {
	return ApiSettingThingsServiceSetSettingThingRequest{ctx: ctx}
}

func (r ApiSettingThingsServiceSetSettingThingRequest) SetSettingThingRequest(setSettingThingRequest SetSettingThingRequest) ApiSettingThingsServiceSetSettingThingRequest {
	r.body = &setSettingThingRequest
	return r
}

func (r ApiSettingThingsServiceSetSettingThingRequest) Execute() (*SetSettingThingResponse, *http.Response, error) {
	return nil, nil, errFake
}
