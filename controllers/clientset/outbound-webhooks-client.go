package clientset

import (
	"context"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

//go:generate mockgen -destination=../mock_clientset/mock_outboundwebhooks-client.go -package=mock_clientset -source=outbound-webhooks-client.go OutboundWebhooksClientInterface
type OutboundWebhooksClientInterface interface {
	Create(ctx context.Context, req *cxsdk.CreateOutgoingWebhookRequest) (*cxsdk.CreateOutgoingWebhookResponse, error)
	Get(ctx context.Context, req *cxsdk.GetOutgoingWebhookRequest) (*cxsdk.GetOutgoingWebhookResponse, error)
	Update(ctx context.Context, req *cxsdk.UpdateOutgoingWebhookRequest) (*cxsdk.UpdateOutgoingWebhookResponse, error)
	Delete(ctx context.Context, req *cxsdk.DeleteOutgoingWebhookRequest) (*cxsdk.DeleteOutgoingWebhookResponse, error)
	List(ctx context.Context, req *cxsdk.ListAllOutgoingWebhooksRequest) (*cxsdk.ListAllOutgoingWebhooksResponse, error)
}
