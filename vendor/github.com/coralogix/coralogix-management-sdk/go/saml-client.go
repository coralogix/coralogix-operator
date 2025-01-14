// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cxsdk

import (
	"context"

	saml "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/aaa/sso/v2"
)

// GetSPParametersRequest is the request for GetSPParameters.
type GetSPParametersRequest = saml.GetSPParametersRequest

// GetSPParametersResponse is the response for GetSPParameters.
type GetSPParametersResponse = saml.GetSPParametersResponse

// SPParameters is a data structure that holds the SAML service provider parameters.
type SPParameters = saml.SPParameters

// SetIDPParametersRequest is the request for SetIDPParameters.
type SetIDPParametersRequest = saml.SetIDPParametersRequest

// IDPParameters is a data structure that holds the SAML identity provider parameters.
type IDPParameters = saml.IDPParameters

// IDPParametersContent is a data structure that holds the SAML identity provider parameters metadata.
type IDPParametersContent = saml.IDPParameters_MetadataContent

// IDPParametersURL is a data structure that holds the SAML identity provider parameters metadata URL.
type IDPParametersURL = saml.IDPParameters_MetadataUrl

// SetSamlActiveRequest is the request for SetActive.
type SetSamlActiveRequest = saml.SetActiveRequest

// GetSamlConfigurationRequest is the request for GetConfiguration.
type GetSamlConfigurationRequest = saml.GetConfigurationRequest

// GetSamlConfigurationResponse is the response for GetConfiguration.
type GetSamlConfigurationResponse = saml.GetConfigurationResponse

const samlFeatureGroupID = "aaa"

// SamlClient is a client for the Coralogix SAML API.
type SamlClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

// GetSPParameters returns the SAML service provider parameters for a given team.
func (s SamlClient) GetSPParameters(ctx context.Context, req *saml.GetSPParametersRequest) (*saml.GetSPParametersResponse, error) {
	callProperties, err := s.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := saml.NewSamlConfigurationServiceClient(conn)

	response, err := client.GetSPParameters(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, saml.SamlConfigurationService_GetSPParameters_FullMethodName, samlFeatureGroupID)
	}
	return response, nil
}

// SetIDPParameters sets the SAML identity provider parameters for a given team.
func (s SamlClient) SetIDPParameters(cxt context.Context, req *saml.SetIDPParametersRequest) (*saml.SetIDPParametersResponse, error) {
	callProperties, err := s.callPropertiesCreator.GetTeamsLevelCallProperties(cxt)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := saml.NewSamlConfigurationServiceClient(conn)

	response, err := client.SetIDPParameters(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, saml.SamlConfigurationService_SetIDPParameters_FullMethodName, samlFeatureGroupID)
	}
	return response, nil
}

// SetActive sets the SAML configuration active state for a given team.
func (s SamlClient) SetActive(ctx context.Context, req *saml.SetActiveRequest) (*saml.SetActiveResponse, error) {
	callProperties, err := s.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := saml.NewSamlConfigurationServiceClient(conn)

	response, err := client.SetActive(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, saml.SamlConfigurationService_SetActive_FullMethodName, samlFeatureGroupID)
	}
	return response, nil
}

// GetConfiguration returns the SAML configuration for a given team.
func (s SamlClient) GetConfiguration(ctx context.Context, req *saml.GetConfigurationRequest) (*saml.GetConfigurationResponse, error) {
	callProperties, err := s.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := saml.NewSamlConfigurationServiceClient(conn)

	response, err := client.GetConfiguration(callProperties.Ctx, req, callProperties.CallOptions...)
	if err != nil {
		return nil, NewSdkAPIError(err, saml.SamlConfigurationService_GetConfiguration_FullMethodName, samlFeatureGroupID)
	}
	return response, nil
}

// NewSamlClient creates a new SAML client.
func NewSamlClient(callPropertiesCreator *CallPropertiesCreator) *SamlClient {
	return &SamlClient{callPropertiesCreator: callPropertiesCreator}
}
