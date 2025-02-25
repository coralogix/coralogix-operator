package e2e

import (
	"context"
	"fmt"
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

var _ = PDescribe("GlobalRouter", Ordered, func() {
	var (
		crClient            client.Client
		notificationsClient *cxsdk.NotificationsClient
		globalRouterID      string
		globalRouter        *coralogixv1alpha1.GlobalRouter
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		notificationsClient = ClientsInstance.GetCoralogixClientSet().Notifications()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Slack Connector")
		connectorName := "slack-connector-for-global-router"
		connector := getSampleSlackConnector(connectorName, testNamespace)
		Expect(crClient.Create(ctx, connector)).To(Succeed())

		By("Creating Slack Preset")
		presetName := "slack-preset-for-global-router"
		preset := getSampleSlackPreset(presetName, testNamespace)
		Expect(crClient.Create(ctx, preset)).To(Succeed())

		By("Creating GlobalRouter")
		globalRouterName := "global-router-sample"
		globalRouter = &coralogixv1alpha1.GlobalRouter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      globalRouterName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.GlobalRouterSpec{
				Name:        globalRouterName,
				Description: "This is a sample global router",
				EntityType:  "alerts",
				Fallback: []coralogixv1alpha1.RoutingTarget{
					{
						Connector: &coralogixv1alpha1.NCRef{
							ResourceRef: &coralogixv1alpha1.ResourceRef{Name: connectorName},
						},
						Preset: &coralogixv1alpha1.NCRef{
							ResourceRef: &coralogixv1alpha1.ResourceRef{Name: presetName},
						},
					},
				},
				Rules: []coralogixv1alpha1.RoutingRule{
					{
						Name:      "first-rule",
						Condition: "alertDef.priority == P1",
						Targets: []coralogixv1alpha1.RoutingTarget{
							{
								Connector: &coralogixv1alpha1.NCRef{
									ResourceRef: &coralogixv1alpha1.ResourceRef{Name: connectorName},
								},
								Preset: &coralogixv1alpha1.NCRef{
									ResourceRef: &coralogixv1alpha1.ResourceRef{Name: presetName},
								},
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, globalRouter)).To(Succeed())

		By("Fetching the GlobalRouter ID")
		fetchedGlobalRouter := &coralogixv1alpha1.GlobalRouter{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: globalRouterName, Namespace: testNamespace}, fetchedGlobalRouter)).To(Succeed())
			if fetchedGlobalRouter.Status.ID != nil {
				globalRouterID = *fetchedGlobalRouter.Status.ID
				return nil
			}
			return fmt.Errorf("GlobalRouter ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying GlobalRouter exists in Coralogix backend")
		Eventually(func() error {
			_, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{
				Identifier: &cxsdk.GlobalRouterIdentifier{
					Value: &cxsdk.GlobalRouterIdentifierIDValue{
						Id: globalRouterID,
					},
				},
			})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the GlobalRouter")
		newRouterName := "Updated Global Router Name"
		modifiedRouter := globalRouter.DeepCopy()
		modifiedRouter.Spec.Name = newRouterName
		Expect(crClient.Patch(ctx, modifiedRouter, client.MergeFrom(globalRouter))).To(Succeed())

		By("Verifying GlobalRouter is updated in Coralogix backend")
		Eventually(func() string {
			getRes, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{
				Identifier: &cxsdk.GlobalRouterIdentifier{
					Value: &cxsdk.GlobalRouterIdentifierIDValue{
						Id: globalRouterID,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			return getRes.GetRouter().GetName()
		}, time.Minute, time.Second).Should(Equal(newRouterName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the GlobalRouter")
		Expect(crClient.Delete(ctx, globalRouter)).To(Succeed())

		By("Verifying GlobalRouter is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{
				Identifier: &cxsdk.GlobalRouterIdentifier{
					Value: &cxsdk.GlobalRouterIdentifierIDValue{
						Id: globalRouterID,
					},
				},
			})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})
