// Package thing_policies_service is a small stand-in for an SDK package that openapi-generator writes.
package thing_policies_service

import (
	"context"
	"errors"
	"net/http"
)

type GetThingPoliciesResponse struct {
	Policies []ThingPolicy `json:"policies"`
}

type OverwriteThingPoliciesResponse struct {
	Policies []ThingPolicy `json:"policies,omitempty"`
}

type ThingPolicy struct {
	Id         string                 `json:"id"`
	Name       string                 `json:"name"`
	Enabled    *bool                  `json:"enabled,omitempty"`
	Tags       *string                `json:"tags,omitempty"`
	Rules      *ThingPolicyRules      `json:"rules,omitempty"`
	OtherRules *ThingPolicyOtherRules `json:"otherRules,omitempty"`
}

type OverwriteThingPoliciesRequest struct {
	Policies []CreateThingPolicyRequest `json:"policies"`
}

type CreateThingPolicyRequest struct {
	Item  CreateThingPolicyItem `json:"item"`
	Rules ThingPolicyRules      `json:"rules"`
}

type CreateThingPolicyItem struct {
	Name     string  `json:"name"`
	Disabled *bool   `json:"disabled,omitempty"`
	Tags     *string `json:"tags,omitempty"`
}

type ThingPolicyRules struct {
	Pattern *string  `json:"pattern,omitempty"`
	Levels  []string `json:"levels,omitempty"`
}

type ThingPolicyOtherRules struct {
	Code *string `json:"code,omitempty"`
}

var errFake = errors.New("the stand-in SDK sends no request")

type ThingPoliciesServiceAPIService struct{}

func NewThingPoliciesServiceAPIService() *ThingPoliciesServiceAPIService {
	return &ThingPoliciesServiceAPIService{}
}

type ApiThingPoliciesServiceGetThingPoliciesRequest struct {
	ctx context.Context
}

func (a *ThingPoliciesServiceAPIService) ThingPoliciesServiceGetThingPolicies(ctx context.Context) ApiThingPoliciesServiceGetThingPoliciesRequest {
	return ApiThingPoliciesServiceGetThingPoliciesRequest{ctx: ctx}
}

func (r ApiThingPoliciesServiceGetThingPoliciesRequest) Execute() (*GetThingPoliciesResponse, *http.Response, error) {
	return nil, nil, errFake
}

type ApiThingPoliciesServiceOverwriteThingPoliciesRequest struct {
	ctx  context.Context
	body *OverwriteThingPoliciesRequest
}

func (a *ThingPoliciesServiceAPIService) ThingPoliciesServiceOverwriteThingPolicies(ctx context.Context) ApiThingPoliciesServiceOverwriteThingPoliciesRequest {
	return ApiThingPoliciesServiceOverwriteThingPoliciesRequest{ctx: ctx}
}

func (r ApiThingPoliciesServiceOverwriteThingPoliciesRequest) OverwriteThingPoliciesRequest(overwriteThingPoliciesRequest OverwriteThingPoliciesRequest) ApiThingPoliciesServiceOverwriteThingPoliciesRequest {
	r.body = &overwriteThingPoliciesRequest
	return r
}

func (r ApiThingPoliciesServiceOverwriteThingPoliciesRequest) Execute() (*OverwriteThingPoliciesResponse, *http.Response, error) {
	return nil, nil, errFake
}
