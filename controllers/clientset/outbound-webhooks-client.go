package clientset

import (
	"context"

	outboundwebhooks "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/outbound-webhooks"
)

//go:generate mockgen -destination=../mock_clientset/mock_outboundwebhooks-client.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset OutboundWebhooksClientInterface
type OutboundWebhooksClientInterface interface {
	CreateOutboundWebhook(ctx context.Context, req *outboundwebhooks.CreateOutgoingWebhookRequest) (*outboundwebhooks.CreateOutgoingWebhookResponse, error)
	GetOutboundWebhook(ctx context.Context, req *outboundwebhooks.GetOutgoingWebhookRequest) (*outboundwebhooks.GetOutgoingWebhookResponse, error)
	UpdateOutboundWebhook(ctx context.Context, req *outboundwebhooks.UpdateOutgoingWebhookRequest) (*outboundwebhooks.UpdateOutgoingWebhookResponse, error)
	DeleteOutboundWebhook(ctx context.Context, req *outboundwebhooks.DeleteOutgoingWebhookRequest) (*outboundwebhooks.DeleteOutgoingWebhookResponse, error)
}

type OutboundWebhooksClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

func (c OutboundWebhooksClient) CreateOutboundWebhook(ctx context.Context, req *outboundwebhooks.CreateOutgoingWebhookRequest) (*outboundwebhooks.CreateOutgoingWebhookResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := outboundwebhooks.NewOutgoingWebhooksServiceClient(conn)

	return client.CreateOutgoingWebhook(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (c OutboundWebhooksClient) GetOutboundWebhook(ctx context.Context, req *outboundwebhooks.GetOutgoingWebhookRequest) (*outboundwebhooks.GetOutgoingWebhookResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := outboundwebhooks.NewOutgoingWebhooksServiceClient(conn)

	return client.GetOutgoingWebhook(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (c OutboundWebhooksClient) UpdateOutboundWebhook(ctx context.Context, req *outboundwebhooks.UpdateOutgoingWebhookRequest) (*outboundwebhooks.UpdateOutgoingWebhookResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := outboundwebhooks.NewOutgoingWebhooksServiceClient(conn)

	return client.UpdateOutgoingWebhook(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (c OutboundWebhooksClient) DeleteOutboundWebhook(ctx context.Context, req *outboundwebhooks.DeleteOutgoingWebhookRequest) (*outboundwebhooks.DeleteOutgoingWebhookResponse, error) {
	callProperties, err := c.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := outboundwebhooks.NewOutgoingWebhooksServiceClient(conn)

	return client.DeleteOutgoingWebhook(callProperties.Ctx, req, callProperties.CallOptions...)
}

func NewOutboundWebhooksClient(c *CallPropertiesCreator) *OutboundWebhooksClient {
	return &OutboundWebhooksClient{callPropertiesCreator: c}
}
