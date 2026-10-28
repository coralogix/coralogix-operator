// Package things_service is a small stand-in for an SDK package that openapi-generator writes.
package things_service

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type ThingKind string

const (
	THINGKIND_THING_KIND_UNSPECIFIED ThingKind = "THING_KIND_UNSPECIFIED"
	THINGKIND_THING_KIND_STANDARD    ThingKind = "THING_KIND_STANDARD"
	THINGKIND_THING_KIND_ADVANCED    ThingKind = "THING_KIND_ADVANCED"
)

type ThingPriority string

const (
	THINGPRIORITY_THING_PRIORITY_UNSPECIFIED          ThingPriority = "THING_PRIORITY_UNSPECIFIED"
	THINGPRIORITY_THING_PRIORITY_LOW                  ThingPriority = "THING_PRIORITY_LOW"
	THINGPRIORITY_THING_PRIORITY_PAGER_DUTY_INCIDENTS ThingPriority = "THING_PRIORITY_PAGER_DUTY_INCIDENTS"
)

type Thing struct {
	Id           string            `json:"id"`
	Name         string            `json:"name"`
	Description  *string           `json:"description,omitempty"`
	Enabled      bool              `json:"enabled"`
	Kind         ThingKind         `json:"kind"`
	Priority     *ThingPriority    `json:"priority,omitempty"`
	Quota        *string           `json:"quota,omitempty"`
	Config       ThingConfig       `json:"config"`
	Settings     *ThingSettings    `json:"settings,omitempty"`
	Items        []ThingItem       `json:"items,omitempty"`
	Destinations []string          `json:"destinations,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	CreateTime   time.Time         `json:"createTime"`
}

type ThingConfig struct {
	Http  *HttpThingConfig  `json:"http,omitempty"`
	Queue *QueueThingConfig `json:"queue,omitempty"`
}

type HttpThingConfig struct {
	Endpoint string `json:"endpoint"`
}

type QueueThingConfig struct {
	Topic string `json:"topic"`
}

type ThingSettings struct {
	Mode     string  `json:"mode"`
	Retries  *int32  `json:"retries,omitempty"`
	Region   *string `json:"region,omitempty"`
	Revision string  `json:"revision"`
}

type ThingSettingsCreate struct {
	Mode    string  `json:"mode"`
	Retries *int32  `json:"retries,omitempty"`
	Region  *string `json:"region,omitempty"`
}

type ThingSettingsUpdate struct {
	Mode    string `json:"mode"`
	Retries *int32 `json:"retries,omitempty"`
}

type ThingItem struct {
	Name   string `json:"name"`
	Weight *int32 `json:"weight,omitempty"`
}

type CreateThingRequest struct {
	Name         string               `json:"name"`
	Description  *string              `json:"description,omitempty"`
	Enabled      *bool                `json:"enabled,omitempty"`
	Kind         ThingKind            `json:"kind"`
	Priority     *ThingPriority       `json:"priority,omitempty"`
	Quota        *string              `json:"quota,omitempty"`
	Config       ThingConfig          `json:"config"`
	Settings     *ThingSettingsCreate `json:"settings,omitempty"`
	Items        []ThingItem          `json:"items,omitempty"`
	Destinations []string             `json:"destinations,omitempty"`
	Tags         []string             `json:"tags,omitempty"`
	Labels       map[string]string    `json:"labels,omitempty"`
}

type ReplaceThingRequest struct {
	Name         string               `json:"name"`
	Description  *string              `json:"description,omitempty"`
	Enabled      *bool                `json:"enabled,omitempty"`
	Priority     *ThingPriority       `json:"priority,omitempty"`
	Quota        *string              `json:"quota,omitempty"`
	Config       ThingConfig          `json:"config"`
	Settings     *ThingSettingsUpdate `json:"settings,omitempty"`
	Items        []ThingItem          `json:"items,omitempty"`
	Destinations []string             `json:"destinations,omitempty"`
	Tags         []string             `json:"tags,omitempty"`
	Labels       map[string]string    `json:"labels,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type ThingsServiceAPIService struct{}

func NewThingsServiceAPIService() *ThingsServiceAPIService { return &ThingsServiceAPIService{} }

type ApiThingsServiceCreateThingRequest struct {
	ctx  context.Context
	body *CreateThingRequest
}

func (a *ThingsServiceAPIService) ThingsServiceCreateThing(ctx context.Context) ApiThingsServiceCreateThingRequest {
	return ApiThingsServiceCreateThingRequest{ctx: ctx}
}

func (r ApiThingsServiceCreateThingRequest) CreateThingRequest(createThingRequest CreateThingRequest) ApiThingsServiceCreateThingRequest {
	r.body = &createThingRequest
	return r
}

func (r ApiThingsServiceCreateThingRequest) Execute() (*Thing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiThingsServiceGetThingRequest struct {
	ctx context.Context
	id  string
}

func (a *ThingsServiceAPIService) ThingsServiceGetThing(ctx context.Context, id string) ApiThingsServiceGetThingRequest {
	return ApiThingsServiceGetThingRequest{ctx: ctx, id: id}
}

func (r ApiThingsServiceGetThingRequest) Execute() (*Thing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiThingsServiceReplaceThingRequest struct {
	ctx  context.Context
	id   string
	body *ReplaceThingRequest
}

func (a *ThingsServiceAPIService) ThingsServiceReplaceThing(ctx context.Context, id string) ApiThingsServiceReplaceThingRequest {
	return ApiThingsServiceReplaceThingRequest{ctx: ctx, id: id}
}

func (r ApiThingsServiceReplaceThingRequest) ReplaceThingRequest(replaceThingRequest ReplaceThingRequest) ApiThingsServiceReplaceThingRequest {
	r.body = &replaceThingRequest
	return r
}

func (r ApiThingsServiceReplaceThingRequest) Execute() (*Thing, *http.Response, error) {
	return nil, nil, errFake
}

type ApiThingsServiceDeleteThingRequest struct {
	ctx context.Context
	id  string
}

func (a *ThingsServiceAPIService) ThingsServiceDeleteThing(ctx context.Context, id string) ApiThingsServiceDeleteThingRequest {
	return ApiThingsServiceDeleteThingRequest{ctx: ctx, id: id}
}

func (r ApiThingsServiceDeleteThingRequest) Execute() (map[string]interface{}, *http.Response, error) {
	return nil, nil, errFake
}
