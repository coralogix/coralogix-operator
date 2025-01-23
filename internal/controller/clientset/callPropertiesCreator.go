// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientset

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	utils "github.com/coralogix/coralogix-operator/api"
)

var (
	RegionToGrpcUrl = map[string]string{
		"APAC1":   "ng-api-grpc.app.coralogix.in:443",
		"AP1":     "ng-api-grpc.app.coralogix.in:443",
		"APAC2":   "ng-api-grpc.coralogixsg.com:443",
		"AP2":     "ng-api-grpc.coralogixsg.com:443",
		"APAC3":   "ng-api-grpc.ap3.coralogix.com:443",
		"AP3":     "ng-api-grpc.ap3.coralogix.com:443",
		"EUROPE1": "ng-api-grpc.coralogix.com:443",
		"EU1":     "ng-api-grpc.coralogix.com:443",
		"EUROPE2": "ng-api-grpc.eu2.coralogix.com:443",
		"EU2":     "ng-api-grpc.eu2.coralogix.com:443",
		"USA1":    "ng-api-grpc.coralogix.us:443",
		"US1":     "ng-api-grpc.coralogix.us:443",
		"USA2":    "ng-api-grpc.cx498.coralogix.com:443",
		"US2":     "ng-api-grpc.cx498.coralogix.com:443",
	}
	OperatorRegionToSdkRegion = map[string]string{
		"APAC1":   "AP1",
		"AP1":     "AP1",
		"APAC2":   "AP2",
		"AP2":     "AP2",
		"APAC3":   "AP3",
		"AP3":     "AP3",
		"EUROPE1": "EU1",
		"EU1":     "EU1",
		"EUROPE2": "EU2",
		"EU2":     "EU2",
		"USA1":    "US1",
		"US1":     "US1",
		"USA2":    "US2",
		"US2":     "US2",
	}
	ValidRegions = utils.GetKeys(RegionToGrpcUrl)
)

type CallPropertiesCreator struct {
	region string
	apiKey string
	//allowRetry bool
}

type CallProperties struct {
	Ctx         context.Context
	Connection  *grpc.ClientConn
	CallOptions []grpc.CallOption
}

func (c CallPropertiesCreator) GetCallProperties(ctx context.Context) (*CallProperties, error) {
	ctx = createAuthContext(ctx, c.apiKey)

	var targetUrl string
	if _, ok := RegionToGrpcUrl[c.region]; ok {
		targetUrl = RegionToGrpcUrl[c.region]
	} else {
		targetUrl = fmt.Sprintf("ng-api-grpc.%s:443", c.region)
	}

	conn, err := createSecureConnection(targetUrl)
	if err != nil {
		return nil, err
	}

	callOptions := createCallOptions()

	return &CallProperties{Ctx: ctx, Connection: conn, CallOptions: callOptions}, nil
}

func createCallOptions() []grpc.CallOption {
	var callOptions []grpc.CallOption
	callOptions = append(callOptions, grpc_retry.WithMax(5))
	callOptions = append(callOptions, grpc_retry.WithBackoff(grpc_retry.BackoffLinear(time.Second)))
	return callOptions
}

func createSecureConnection(targetUrl string) (*grpc.ClientConn, error) {
	return grpc.Dial(targetUrl,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
}

func createAuthContext(ctx context.Context, apiKey string) context.Context {
	md := metadata.New(map[string]string{"Authorization": fmt.Sprintf("Bearer %s", apiKey)})
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx
}

func NewCallPropertiesCreator(region, apiKey string) *CallPropertiesCreator {
	return &CallPropertiesCreator{
		region: region,
		apiKey: apiKey,
	}
}
