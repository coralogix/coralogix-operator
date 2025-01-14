package common

import (
	"context"
	"fmt"
	"strconv"

	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	"github.com/go-logr/logr"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +k8s:deepcopy-gen=false
type ListingAlertsAndWebhooksProperties struct {
	Ctx             context.Context
	Log             logr.Logger
	Client          client.Client
	AlertNameToId   map[string]string
	WebhookNameToId map[string]uint32
	Clientset       clientset.ClientSetInterface
	Namespace       string
}

func ConvertCRDNameToIntegrationID(name string, properties *ListingAlertsAndWebhooksProperties) (*wrapperspb.UInt32Value, error) {
	client, ctx, namespace := properties.Client, properties.Ctx, properties.Namespace
	webhook := &OutboundWebhook{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook, name: %s, namespace: %s, error: %w", name, namespace, err)
	}

	if webhook.Status.ExternalID == nil {
		return nil, fmt.Errorf("webhook %s has no external-id", name)
	}

	externalID, err := strconv.Atoi(*webhook.Status.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("webhook %s has invalid external-id", name)
	}

	return wrapperspb.UInt32(uint32(externalID)), nil
}
