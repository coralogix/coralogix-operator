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
)

type CallPropertiesCreator struct {
	targetUrl string
	apiKey    string
	//allowRetry bool
}

type CallProperties struct {
	Ctx         context.Context
	Connection  *grpc.ClientConn
	CallOptions []grpc.CallOption
}

func (c CallPropertiesCreator) GetCallProperties(ctx context.Context) (*CallProperties, error) {
	ctx = createAuthContext(ctx, c.apiKey)

	conn, err := createSecureConnection(c.targetUrl)
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

func NewCallPropertiesCreator(targetUrl, apiKey string) *CallPropertiesCreator {
	return &CallPropertiesCreator{
		targetUrl: targetUrl,
		apiKey:    apiKey,
	}
}
