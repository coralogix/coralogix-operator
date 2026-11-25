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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	gouuid "github.com/google/uuid"
	"k8s.io/utils/ptr"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	webhooks "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/outgoing_webhooks_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
)

// beforeCreate gives a Generic or Send Log webhook a new UUID when the CR sets none. The API needs
// one, and the released CR had no field for it.
func (r *OutboundWebhookReconciler) beforeCreate(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.OutboundWebhook, body *webhooks.CreateOutgoingWebhookRequest) error { //nolint:unparam // the generated lifecycle fixes the signature
	fillWebhookUUIDs(&body.Data)
	return nil
}

// beforeUpdate gives a Generic or Send Log webhook a new UUID when the CR sets none, on each Update,
// as the released kind did.
func (r *OutboundWebhookReconciler) beforeUpdate(_ context.Context, _ logr.Logger, _ *coralogixv1alpha1.OutboundWebhook, body *webhooks.UpdateOutgoingWebhookRequest) error { //nolint:unparam // the generated lifecycle fixes the signature
	fillWebhookUUIDs(&body.Data)
	return nil
}

// fillWebhookUUIDs sets the UUID that the CR leaves out or leaves empty: the API rejects an empty
// one. A UUID that the CR sets stays as it is, so running it twice on one body changes nothing.
func fillWebhookUUIDs(data *webhooks.OutgoingWebhookInputData) {
	if generic := data.GenericWebhook; generic != nil && ptr.Deref(generic.Uuid, "") == "" {
		generic.Uuid = ptr.To(gouuid.NewString())
	}
	if sendLog := data.SendLog; sendLog != nil && ptr.Deref(sendLog.Uuid, "") == "" {
		sendLog.Uuid = ptr.To(gouuid.NewString())
	}
}

// afterCreate reads the external id of the new webhook into the status. Alerts name a webhook by
// this id.
func (r *OutboundWebhookReconciler) afterCreate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.OutboundWebhook, _ *webhooks.CreateOutgoingWebhookResponse) error {
	return r.readWebhookExternalID(ctx, cr)
}

// afterUpdate reads the external id only when the status has none: after a failed afterCreate, or
// for a CR whose status lost it. A resync of a CR with the id sends only the Update.
func (r *OutboundWebhookReconciler) afterUpdate(ctx context.Context, _ logr.Logger, cr *coralogixv1alpha1.OutboundWebhook, _ map[string]interface{}) error {
	if ptr.Deref(cr.Status.ExternalId, "") != "" {
		return nil
	}
	return r.readWebhookExternalID(ctx, cr)
}

// readWebhookExternalID sets status.externalId from a Get of the webhook. The error has the HTTP
// status only: the body of an answer can hold a credential of the webhook.
func (r *OutboundWebhookReconciler) readWebhookExternalID(ctx context.Context, cr *coralogixv1alpha1.OutboundWebhook) error {
	id := ptr.Deref(cr.Status.ID, "")
	if id == "" {
		return errors.New("the OutboundWebhook has no status id")
	}
	resp, httpResp, err := r.Client.OutgoingWebhooksServiceGetOutgoingWebhook(ctx, id).Execute()
	if err != nil {
		return fmt.Errorf("get the external id of OutboundWebhook %s failed (HTTP %d)", id, cxsdk.Code(cxsdk.NewAPIError(httpResp, err)))
	}
	if resp == nil {
		return fmt.Errorf("get the external id of OutboundWebhook %s answered no webhook", id)
	}
	cr.Status.ExternalId = ptr.To(strconv.FormatInt(resp.Webhook.ExternalId, 10))
	return nil
}
