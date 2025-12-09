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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("Connector", Ordered, func() {
	var (
		crClient            client.Client
		notificationsClient *cxsdk.NotificationsClient
		connectorID         string
		connector           *coralogixv1alpha1.Connector
		connectorName       string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		notificationsClient = ClientsInstance.GetCoralogixClientSet().Notifications()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Connector")
		connectorName = fmt.Sprintf("slack-connector-%d", time.Now().Unix())
		connector = getSampleSlackConnector(connectorName, testNamespace)
		Expect(crClient.Create(ctx, connector)).To(Succeed())

		By("Fetching the Connector ID")
		fetchedConnector := &coralogixv1alpha1.Connector{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: connectorName, Namespace: testNamespace}, fetchedConnector)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedConnector.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedConnector.Status.PrintableStatus).To(Equal("RemoteSynced"))
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

	It("After deleted from Coralogix backend directly, it should be recreated based on configured interval",
		func(ctx context.Context) {
			By("Deleting the Connector from Coralogix backend")
			_, err := notificationsClient.DeleteConnector(ctx, &cxsdk.DeleteConnectorRequest{Id: connectorID})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying Connector is populated with a new ID after configured interval")
			var newConnectorID string
			Eventually(func() bool {
				fetchedConnector := &coralogixv1alpha1.Connector{}
				Expect(crClient.Get(ctx, types.NamespacedName{Name: connectorName, Namespace: testNamespace}, fetchedConnector)).To(Succeed())
				if fetchedConnector.Status.Id == nil {
					return false
				}
				newConnectorID = *fetchedConnector.Status.Id
				return newConnectorID != connectorID
			}, 2*time.Minute, time.Second).Should(BeTrue())

			By("Verifying Scope with new the ID exists in Coralogix backend")
			Eventually(func() error {
				_, err := notificationsClient.GetConnector(ctx, &cxsdk.GetConnectorRequest{
					Id: newConnectorID,
				})

				return err
			}, time.Minute, time.Second).Should(Succeed())
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
})

func getSampleSlackConnector(name, namespace string) *coralogixv1alpha1.Connector {
	return &coralogixv1alpha1.Connector{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: coralogixv1alpha1.ConnectorSpec{
			Name:        name,
			Description: "Slack connector",
			Type:        "slack",
			ConnectorConfig: coralogixv1alpha1.ConnectorConfig{
				Fields: []coralogixv1alpha1.ConnectorConfigField{
					{FieldName: "channel", Value: "general"},
					{FieldName: "integrationId", Value: "Slack"},
					{FieldName: "fallbackChannel", Value: "fallback_general"},
				},
			},
			ConfigOverrides: []coralogixv1alpha1.EntityTypeConfigOverrides{
				{
					EntityType: "alerts",
					Fields: []coralogixv1alpha1.TemplatedConnectorConfigField{
						{
							FieldName: "channel",
							Template:  "{{alertDef.priority}}",
						},
					},
				},
			},
		},
	}
}
