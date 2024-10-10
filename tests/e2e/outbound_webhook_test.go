/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"

	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
)

var _ = Describe("Outbound Webhook", Ordered, func() {
	var (
		crClient              client.Client
		OutboundWebhookClient *cxsdk.WebhooksClient
		outboundWebhookID     string
		outBoundWebhhok       *coralogixv1alpha1.OutboundWebhook
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		OutboundWebhookClient = ClientsInstance.GetCoralogixClientSet().Webhooks()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Defining an Outbound webhook resource")
		outboundWebhookName := "slack-outbound-webhook"
		outBoundWebhhok = &coralogixv1alpha1.OutboundWebhook{
			ObjectMeta: metav1.ObjectMeta{
				Name:      outboundWebhookName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.OutboundWebhookSpec{
				Name: outboundWebhookName,
				OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
					Slack: &coralogixv1alpha1.Slack{
						Url: "https://hooks.slack.com/services",
						Attachments: []coralogixv1alpha1.SlackConfigAttachment{
							{
								Type:     "MetricSnapshot",
								IsActive: true,
							},
						},
						Digests: []coralogixv1alpha1.SlackConfigDigest{
							{
								Type:     "FlowAnomalies",
								IsActive: true,
							},
						},
					},
				},
			},
		}

		By("Creating the OutboundWebhook resource in the cluster")
		Expect(crClient.Create(ctx, outBoundWebhhok)).To(Succeed())

		By("Fetching the OutboundWebhook ID")
		fetchedOutboundWebhook := &coralogixv1alpha1.OutboundWebhook{}
		Eventually(func(g Gomega) error {
			err := crClient.Get(ctx, types.NamespacedName{Name: outboundWebhookName, Namespace: testNamespace}, fetchedOutboundWebhook)
			g.Expect(err).NotTo(HaveOccurred())

			if fetchedOutboundWebhook.Status.ID != nil {
				outboundWebhookID = *fetchedOutboundWebhook.Status.ID
				return nil
			}

			return fmt.Errorf("OutboundWebhook ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying OutboundWebhook exists in Coralogix backend")
		Eventually(func() error {
			_, err := OutboundWebhookClient.Get(ctx, &cxsdk.GetOutgoingWebhookRequest{Id: wrapperspb.String(outboundWebhookID)})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the OutboundWebhook resource")
		newOutboundWebhookName := "slack-outbound-webhook-updated"
		modifiedOutboundWebhook := outBoundWebhhok.DeepCopy()
		modifiedOutboundWebhook.Spec.Name = newOutboundWebhookName
		err := crClient.Patch(ctx, modifiedOutboundWebhook, client.MergeFrom(outBoundWebhhok))
		Expect(err).NotTo(HaveOccurred())

		By("Verifying OutboundWebhook is updated in Coralogix backend")
		Eventually(func(g Gomega) string {
			getOutboundWebhookRes, err := OutboundWebhookClient.Get(ctx, &cxsdk.GetOutgoingWebhookRequest{Id: wrapperspb.String(outboundWebhookID)})
			g.Expect(err).NotTo(HaveOccurred())
			return getOutboundWebhookRes.GetWebhook().GetName().GetValue()
		}, time.Minute, time.Second).Should(Equal(newOutboundWebhookName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the OutboundWebhook resource")
		err := crClient.Delete(ctx, outBoundWebhhok)
		Expect(err).NotTo(HaveOccurred())

		By("Verifying OutboundWebhook is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := OutboundWebhookClient.Get(ctx, &cxsdk.GetOutgoingWebhookRequest{Id: wrapperspb.String(outboundWebhookID)})
			return status.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})
