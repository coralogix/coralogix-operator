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

	dataprime "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/dataprime/v1"
)

// QueryRequest is for running the query
type QueryRequest = dataprime.QueryRequest

// DataprimeQueryMetadata is the metadata for the query
type DataprimeQueryMetadata = dataprime.Metadata

// DataprimeClient is a client for the Coralogix Dataprime API.
type DataprimeClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

const dataprimeFeatureGroupID = "dataprime"

// Query runs a query.
func (c DataprimeClient) Query(ctx context.Context, req *QueryRequest) (dataprime.DataprimeQueryService_QueryClient, error) {
	callProperties, err := c.callPropertiesCreator.GetTeamsLevelCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := dataprime.NewDataprimeQueryServiceClient(conn)

	response, err := client.Query(ctx, req)
	if err != nil {
		return nil, NewSdkAPIError(err, dataprime.DataprimeQueryService_Query_FullMethodName, dataprimeFeatureGroupID)
	}
	return response, nil
}

// NewDataprimeClient creates a new dataprime client.
func NewDataprimeClient(c *CallPropertiesCreator) *DataprimeClient {
	return &DataprimeClient{callPropertiesCreator: c}
}
