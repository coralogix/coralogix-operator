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

package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("OutboundWebhook", Ordered, func() {
	var (
		crClient               client.Client
		OutboundWebhooksClient *cxsdk.WebhooksClient
		outboundWebhookID      string
		outboundWebhookName    = "slack-outbound-webhook"
		outBoundWebhook        *coralogixv1alpha1.OutboundWebhook
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		OutboundWebhooksClient = ClientsInstance.GetCoralogixClientSet().Webhooks()
		outBoundWebhook = getSampleWebhook(outboundWebhookName, testNamespace)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating OutboundWebhook")
		Expect(crClient.Create(ctx, outBoundWebhook)).To(Succeed())

		By("Fetching the OutboundWebhook ID")
		fetchedOutboundWebhook := &coralogixv1alpha1.OutboundWebhook{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: outboundWebhookName, Namespace: testNamespace}, fetchedOutboundWebhook)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedOutboundWebhook.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedOutboundWebhook.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedOutboundWebhook.Status.ID != nil {
				outboundWebhookID = *fetchedOutboundWebhook.Status.ID
				return nil
			}
			return fmt.Errorf("OutboundWebhook ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying OutboundWebhook exists in Coralogix backend")
		Eventually(func() error {
			_, err := OutboundWebhooksClient.Get(ctx, &cxsdk.GetOutgoingWebhookRequest{Id: wrapperspb.String(outboundWebhookID)})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the OutboundWebhook")
		newOutboundWebhookName := "slack-outbound-webhook-updated"
		modifiedOutboundWebhook := outBoundWebhook.DeepCopy()
		modifiedOutboundWebhook.Spec.Name = newOutboundWebhookName
		Expect(crClient.Patch(ctx, modifiedOutboundWebhook, client.MergeFrom(outBoundWebhook))).To(Succeed())

		By("Verifying OutboundWebhook is updated in Coralogix backend")
		Eventually(func() string {
			getOutboundWebhookRes, err := OutboundWebhooksClient.Get(ctx, &cxsdk.GetOutgoingWebhookRequest{Id: wrapperspb.String(outboundWebhookID)})
			Expect(err).ToNot(HaveOccurred())
			return getOutboundWebhookRes.GetWebhook().GetName().GetValue()
		}, time.Minute, time.Second).Should(Equal(newOutboundWebhookName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the OutboundWebhook")
		Expect(crClient.Delete(ctx, outBoundWebhook)).To(Succeed())

		By("Verifying OutboundWebhook is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := OutboundWebhooksClient.Get(ctx, &cxsdk.GetOutgoingWebhookRequest{Id: wrapperspb.String(outboundWebhookID)})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})

	It("should deny creation of OutboundWebhook without type", func(ctx context.Context) {
		outBoundWebhook.Spec.OutboundWebhookType = coralogixv1alpha1.OutboundWebhookType{}
		err := crClient.Create(ctx, outBoundWebhook)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of the following fields must be set: genericWebhook, slack, pagerDuty, sendLog, emailGroup, microsoftTeams, jira, opsgenie, demisto, awsEventBridge"))
	})

	It("should deny creation of OutboundWebhook with two types", func(ctx context.Context) {
		outBoundWebhook.Spec.OutboundWebhookType.SendLog = &coralogixv1alpha1.SendLog{
			Payload: `{"key1": "value1", "key2": "value2"}`,
			Url:     "https://example.com",
		}
		err := crClient.Create(ctx, outBoundWebhook)
		Expect(err.Error()).To(ContainSubstring("Exactly one of the following fields must be set: genericWebhook, slack, pagerDuty, sendLog, emailGroup, microsoftTeams, jira, opsgenie, demisto, awsEventBridge"))
	})
})

func getSampleWebhook(name, namespace string) *coralogixv1alpha1.OutboundWebhook {
	return &coralogixv1alpha1.OutboundWebhook{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.OutboundWebhookSpec{
			Name: name,
			OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
				PagerDuty: &coralogixv1alpha1.PagerDuty{
					ServiceKey: "12345678-1234-1234-1234-123456789012",
				},
			},
		},
	}
}
