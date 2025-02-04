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
	"k8s.io/utils/ptr"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("Connector", Ordered, func() {
	var (
		crClient            client.Client
		notificationsClient *cxsdk.NotificationsClient
		connectorID         string
		connector           *coralogixv1alpha1.Connector
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		notificationsClient = ClientsInstance.GetCoralogixClientSet().Notifications()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Connector")
		connectorName := "slack-connector"
		connector = &coralogixv1alpha1.Connector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      connectorName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.ConnectorSpec{
				Name:        "Slack Connector",
				Description: "A connector for Slack integration",
				ConnectorType: &coralogixv1alpha1.ConnectorType{
					Slack: &coralogixv1alpha1.ConnectorSlack{
						CommonFields: &coralogixv1alpha1.ConnectorSlackCommonFields{
							RawConfig: &coralogixv1alpha1.ConnectorSlackConfig{
								Integration: &coralogixv1alpha1.SlackIntegrationRef{
									BackendRef: &coralogixv1alpha1.SlackIntegrationBackendRef{
										Id: "slack_integration",
									},
								},
								Channel:         ptr.To("general"),
								FallbackChannel: "fallback_general",
							},
							StructuredConfig: &coralogixv1alpha1.ConnectorSlackConfig{
								Integration: &coralogixv1alpha1.SlackIntegrationRef{
									BackendRef: &coralogixv1alpha1.SlackIntegrationBackendRef{
										Id: "slack_integration",
									},
								},
								Channel:         ptr.To("general"),
								FallbackChannel: "fallback_general",
							},
						},
						Overrides: []coralogixv1alpha1.ConnectorSlackOverride{
							{
								EntityType: "alerts",
								RawConfig: &coralogixv1alpha1.ConnectorSlackConfigOverride{
									Channel: "override",
								},
								StructuredConfig: &coralogixv1alpha1.ConnectorSlackConfigOverride{
									Channel: "override",
								},
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, connector)).To(Succeed())

		By("Fetching the Connector ID")
		fetchedConnector := &coralogixv1alpha1.Connector{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: connectorName, Namespace: testNamespace}, fetchedConnector)).To(Succeed())
			if fetchedConnector.Status.Id != nil {
				connectorID = *fetchedConnector.Status.Id
				return nil
			}
			return fmt.Errorf("connector ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Connector exists in Coralogix backend")
		Eventually(func() error {
			_, err := notificationsClient.GetConnector(ctx, &cxsdk.GetConnectorRequest{
				Id: connectorID,
			})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the Connector")
		newConnectorName := "slack-connector-updated"
		modifiedConnector := connector.DeepCopy()
		modifiedConnector.Spec.Name = newConnectorName
		Expect(crClient.Patch(ctx, modifiedConnector, client.MergeFrom(connector))).To(Succeed())

		By("Verifying Connector is updated in Coralogix backend")
		Eventually(func() string {
			getConnectorRes, err := notificationsClient.GetConnector(ctx, &cxsdk.GetConnectorRequest{
				Id: connectorID,
			})
			Expect(err).ToNot(HaveOccurred())
			return getConnectorRes.GetConnector().GetName()
		}, time.Minute, time.Second).Should(Equal(newConnectorName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Connector")
		Expect(crClient.Delete(ctx, connector)).To(Succeed())

		By("Verifying Connector is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := notificationsClient.GetConnector(ctx, &cxsdk.GetConnectorRequest{
				Id: connectorID,
			})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})

	It("should deny creation of Connector with two types", func(ctx context.Context) {
		connector.Spec.ConnectorType.GenericHttps = &coralogixv1alpha1.ConnectorGenericHttps{
			Config: &coralogixv1alpha1.ConnectorGenericHttpsConfig{
				Url:                  "https://example.com",
				Method:               ptr.To("put"),
				AdditionalBodyFields: ptr.To("body"),
				AdditionalHeaders:    ptr.To("headers"),
			},
		}
		err := crClient.Create(ctx, connector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("only one connector type should be set"))
	})
})
