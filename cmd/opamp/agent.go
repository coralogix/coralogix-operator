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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

const (
	agentType    = "coralogix-operator"
	agentVersion = "v4.0.0"
	serverUrl    = "https://ingress.eu2.coralogix.com/opamp/v1"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientID := uuid.New()

	zapLogger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("[FATAL] Failed to create zap logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLogger.Sync()

	logger := newLoggerFromZap(zapLogger.With(zap.String("component", "opamp-client")))
	logger.Infof(ctx, "Starting OpAMP client")

	go handleSignals(cancel, logger)

	if err := startOpAMPClient(ctx, clientID, logger); err != nil {
		logger.Errorf(ctx, "failed to start OpAMP client: %v", err)
		os.Exit(1)
	}

	<-ctx.Done()
	logger.Infof(ctx, "shutting down")
}

func startOpAMPClient(ctx context.Context, instanceID uuid.UUID, logger types.Logger) error {
	httpClient := client.NewHTTP(logger)
	httpClient.SetPollingInterval(10 * time.Second)

	desc := &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			kv("service.instance.id", instanceID.String()),
			kv("service.name", agentType),
			kv("service.version", agentVersion),
		},
		NonIdentifyingAttributes: []*protobufs.KeyValue{
			kv("cx.agent.type", "operator"),
		},
	}

	if err := httpClient.SetAgentDescription(desc); err != nil {
		return fmt.Errorf("failed to set agent description: %w", err)
	}

	settings := types.StartSettings{
		OpAMPServerURL: serverUrl,
		InstanceUid:    types.InstanceUid(instanceID),
		Header: http.Header{
			"Authorization": []string{"Bearer " + os.Getenv("CORALOGIX_SEND_YOUR_DATA_API_KEY")},
		},
		Callbacks: types.Callbacks{
			OnConnect: func(_ context.Context) {
				logger.Debugf(ctx, "connected to OpAMP server")
			},
			OnConnectFailed: func(_ context.Context, err error) {
				logger.Errorf(ctx, "connection failed: %v", err)
			},
		},
	}

	return httpClient.Start(ctx, settings)
}

func handleSignals(cancelFunc context.CancelFunc, logger *opAMPLogger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	logger.Infof(context.Background(), "Received termination signal")
	cancelFunc()
}

func kv(k, v string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key: k,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{StringValue: v},
		},
	}
}
