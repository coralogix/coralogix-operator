package clientset

import (
	"context"
	"fmt"
	"strings"

	"github.com/coralogix/coralogix-operator/controllers/clientset/rest"
)

//go:generate mockgen -destination=../mock_clientset/mock_webhooks-client.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset WebhooksClientInterface
type WebhooksClientInterface interface {
	CreateWebhook(ctx context.Context, body string) (string, error)
	GetWebhook(ctx context.Context, webhookId string) (string, error)
	GetWebhooks(ctx context.Context) (string, error)
	UpdateWebhook(ctx context.Context, body string) (string, error)
	DeleteWebhook(ctx context.Context, webhookId string) (string, error)
}

type WebhooksClient struct {
	client *rest.Client
}

func (w WebhooksClient) CreateWebhook(ctx context.Context, body string) (string, error) {
	return w.client.Post(ctx, "/api/v1/external/integrations", "application/json", body)
}

func (w WebhooksClient) GetWebhook(ctx context.Context, webhookId string) (string, error) {
	return w.client.Get(ctx, fmt.Sprintf("/api/v1/external/integrations/%s", webhookId))
}

func (w WebhooksClient) GetWebhooks(ctx context.Context) (string, error) {
	return w.client.Get(ctx, "/api/v1/external/integrations/")
}

func (w WebhooksClient) UpdateWebhook(ctx context.Context, body string) (string, error) {
	return w.client.Post(ctx, "/api/v1/external/integrations", "application/json", body)
}

func (w WebhooksClient) DeleteWebhook(ctx context.Context, webhookId string) (string, error) {
	return w.client.Delete(ctx, fmt.Sprintf("/api/v1/external/integrations/%s", webhookId))
}

func NewWebhooksClient(c *CallPropertiesCreator) *WebhooksClient {
	targetUrl := "https://" + strings.Replace(c.targetUrl, "grpc", "http", 1)
	client := rest.NewRestClient(targetUrl, c.apiKey)
	return &WebhooksClient{client: client}
}
