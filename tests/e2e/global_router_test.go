package e2e

import (
	"context"
	"fmt"
	"time"

	gouuid "github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("GlobalRouter", Ordered, func() {
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
		connectorName := fmt.Sprintf("slack-connector-for-global-router-%d", time.Now().Unix())
		Expect(crClient.Create(ctx, getSampleSlackConnector(connectorName, testNamespace))).To(Succeed())

		By("Creating Slack Preset")
		presetName := fmt.Sprintf("slack-preset-for-global-router-%d", time.Now().Unix())
		Expect(crClient.Create(ctx, getSampleSlackPreset(presetName, testNamespace))).To(Succeed())

		By("Creating GlobalRouter")
		globalRouterName := "global-router-sample" + gouuid.NewString()
		globalRouter = getSampleGlobalRouter(globalRouterName, testNamespace, connectorName, presetName)
		Expect(crClient.Create(ctx, globalRouter)).To(Succeed())

		By("Fetching the GlobalRouter ID")
		fetchedGlobalRouter := &coralogixv1alpha1.GlobalRouter{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: globalRouterName, Namespace: testNamespace}, fetchedGlobalRouter)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedGlobalRouter.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedGlobalRouter.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedGlobalRouter.Status.Id != nil {
				globalRouterID = *fetchedGlobalRouter.Status.Id
				return nil
			}
			return fmt.Errorf("GlobalRouter ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying GlobalRouter exists in Coralogix backend")
		Eventually(func() error {
			_, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{Id: globalRouterID})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the GlobalRouter")
		newRuleName := "Updated Rule Name"
		modifiedRouter := globalRouter.DeepCopy()
		modifiedRouter.Spec.Rules[0].Name = newRuleName
		Expect(crClient.Patch(ctx, modifiedRouter, client.MergeFrom(globalRouter))).To(Succeed())

		By("Verifying GlobalRouter is updated in Coralogix backend")
		Eventually(func() string {
			getRes, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{Id: globalRouterID})
			Expect(err).ToNot(HaveOccurred())
			return *getRes.GetRouter().Rules[0].Name
		}, time.Minute, time.Second).Should(Equal(newRuleName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the GlobalRouter")
		Expect(crClient.Delete(ctx, globalRouter)).To(Succeed())

		By("Verifying GlobalRouter is deleted in Coralogix backend")
		Eventually(func() codes.Code {
			_, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{Id: globalRouterID})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})

func getSampleGlobalRouter(globalRouterName, testNamespace, slackConnectorName, slackPresetName string) *coralogixv1alpha1.GlobalRouter {
	return &coralogixv1alpha1.GlobalRouter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      globalRouterName,
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.GlobalRouterSpec{
			Name:        globalRouterName,
			Description: "global router example",
			RoutingLabels: coralogixv1alpha1.RoutingLabels{
				Environment: ptr.To(gouuid.NewString()),
				Service:     ptr.To(gouuid.NewString()),
				Team:        ptr.To(gouuid.NewString()),
			},
			Rules: []coralogixv1alpha1.RoutingRule{
				{
					Name:      "first-rule",
					Condition: `alertDef.priority == "P1"`,
					Targets: []coralogixv1alpha1.RoutingTarget{
						{
							Connector: coralogixv1alpha1.NCRef{
								ResourceRef: &coralogixv1alpha1.ResourceRef{
									Name: slackConnectorName,
								},
							},
							Preset: &coralogixv1alpha1.NCRef{
								BackendRef: &coralogixv1alpha1.NCBackendRef{
									ID: "preset_system_slack_alerts_basic",
								},
							},
						},
					},
				},
				{
					Name:      "second-rule",
					Condition: `alertDef.priority == "P2"`,
					Targets: []coralogixv1alpha1.RoutingTarget{
						{
							Connector: coralogixv1alpha1.NCRef{
								ResourceRef: &coralogixv1alpha1.ResourceRef{
									Name: slackConnectorName,
								},
							},
							Preset: &coralogixv1alpha1.NCRef{
								ResourceRef: &coralogixv1alpha1.ResourceRef{
									Name: slackPresetName,
								},
							},
						},
					},
				},
			},
		},
	}
}
