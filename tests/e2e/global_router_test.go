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

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// Serial: both GlobalRouter containers route notifications for the whole account, so they must
// not run at the same time as each other when the suite runs with -procs > 1.
var _ = Describe("GlobalRouter", Ordered, Serial, func() {
	var (
		crClient            client.Client
		notificationsClient *cxsdk.NotificationsClient
		globalRouterID      string
		globalRouter        *coralogixv1alpha1.GlobalRouter
	)

	BeforeAll(func() {
		skipIfEphemeralTeam("the router's Slack connector references the shared team's Slack integration (SLACK_INTEGRATION_ID)")
		crClient = ClientsInstance.GetControllerRuntimeClient()
		notificationsClient = ClientsInstance.GetCoralogixClientSet().Notifications()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Slack Connector")
		connectorName := uniqueName("slack-connector-for-global-router")
		Expect(crClient.Create(ctx, getSampleSlackConnector(connectorName))).To(Succeed())

		By("Creating Slack Preset")
		presetName := uniqueName("slack-preset-for-global-router")
		Expect(crClient.Create(ctx, getSampleSlackPreset(presetName, testNamespace))).To(Succeed())

		// The GlobalRouter's rules reference both by name, and it cannot reach RemoteSynced until
		// each has an ID to resolve. Wait for them rather than relying on them winning that race -
		// under load they do not. The fallbackTargets spec below already does this.
		By("Waiting for the Connector and Preset to be synced so their IDs can be resolved")
		Eventually(func(g Gomega) *string {
			fetchedConnector := &coralogixv1alpha1.Connector{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: connectorName, Namespace: testNamespace}, fetchedConnector)).To(Succeed())
			return fetchedConnector.Status.Id
		}, time.Minute, time.Second).ShouldNot(BeNil())

		Eventually(func(g Gomega) *string {
			fetchedPreset := &coralogixv1alpha1.Preset{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: presetName, Namespace: testNamespace}, fetchedPreset)).To(Succeed())
			return fetchedPreset.Status.Id
		}, time.Minute, time.Second).ShouldNot(BeNil())

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
		newRuleName := uniqueName("updated-rule")
		modifiedRouter := globalRouter.DeepCopy()
		modifiedRouter.Spec.Rules[0].Name = newRuleName
		Expect(crClient.Patch(ctx, modifiedRouter, client.MergeFrom(globalRouter))).To(Succeed())

		By("Verifying GlobalRouter is updated in Coralogix backend")
		// The backend does not promise to return the rules in spec order, so collect the names and
		// compare them as a set instead of reading Rules[0]. The expected set comes from the patched
		// spec, so it also proves the other rules are kept.
		expectedRuleNames := make([]string, 0, len(modifiedRouter.Spec.Rules))
		for _, rule := range modifiedRouter.Spec.Rules {
			expectedRuleNames = append(expectedRuleNames, rule.Name)
		}

		Eventually(func() ([]string, error) {
			getRes, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{Id: globalRouterID})
			if err != nil {
				return nil, err
			}
			if getRes.GetRouter() == nil {
				return nil, nil
			}
			var ruleNames []string
			for _, rule := range getRes.GetRouter().Rules {
				ruleNames = append(ruleNames, ptr.Deref(rule.Name, ""))
			}
			return ruleNames, nil
		}, time.Minute, time.Second).Should(ConsistOf(expectedRuleNames))
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
			RoutingLabels: &coralogixv1alpha1.RoutingLabels{
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

// NC gap fields: GlobalRouter disabled flag and per-entity-type fallbackTargets.
var _ = Describe("GlobalRouter with disabled and fallbackTargets", Ordered, Serial, func() {
	var (
		crClient            client.Client
		notificationsClient *cxsdk.NotificationsClient
		routerID            string
		router              *coralogixv1alpha1.GlobalRouter
		routerName          string
		connector           *coralogixv1alpha1.Connector
		connectorName       string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		notificationsClient = ClientsInstance.GetCoralogixClientSet().Notifications()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating a connector referenced by the router")
		connectorName = fmt.Sprintf("connector-for-gaps-%s", gouuid.NewString())
		connector = getSampleGenericHttpsConnectorForRouterGaps(connectorName, testNamespace)
		Expect(crClient.Create(ctx, connector)).To(Succeed())

		By("Waiting for the Connector to be synced so its ID can be resolved")
		Eventually(func(g Gomega) bool {
			fetched := &coralogixv1alpha1.Connector{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: connectorName, Namespace: testNamespace}, fetched)).To(Succeed())
			return fetched.Status.Id != nil
		}, time.Minute, time.Second).Should(BeTrue())

		By("Creating the GlobalRouter with disabled + fallbackTargets")
		routerName = "global-router-gaps-" + gouuid.NewString()
		router = getSampleGlobalRouterWithGapFields(routerName, testNamespace, connectorName)
		Expect(crClient.Create(ctx, router)).To(Succeed())

		Eventually(func(g Gomega) error {
			fetched := &coralogixv1alpha1.GlobalRouter{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: routerName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetched.Status.Id != nil {
				routerID = *fetched.Status.Id
				return nil
			}
			return fmt.Errorf("GlobalRouter ID is not set")
		}, 2*time.Minute, time.Second).Should(Succeed())

		Eventually(func() error {
			_, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{Id: routerID})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the GlobalRouter and verifying it is removed from the backend")
		Expect(crClient.Delete(ctx, router)).To(Succeed())
		Eventually(func() codes.Code {
			_, err := notificationsClient.GetGlobalRouter(ctx, &cxsdk.GetGlobalRouterRequest{Id: routerID})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))

		By("Deleting the referenced connector")
		Expect(crClient.Delete(ctx, connector)).To(Succeed())
	})
})

func getSampleGlobalRouterWithGapFields(name, namespace, connectorName string) *coralogixv1alpha1.GlobalRouter {
	return &coralogixv1alpha1.GlobalRouter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.GlobalRouterSpec{
			Name:        name,
			Description: "global router with disabled + fallbackTargets",
			Disabled:    ptr.To(true),
			RoutingLabels: &coralogixv1alpha1.RoutingLabels{
				Environment: ptr.To(gouuid.NewString()),
				Service:     ptr.To(gouuid.NewString()),
				Team:        ptr.To(gouuid.NewString()),
			},
			Rules: []coralogixv1alpha1.RoutingRule{
				{
					Name:       "first-rule",
					EntityType: ptr.To("alerts"),
					Condition:  `alertDef.priority == "P1"`,
					Targets: []coralogixv1alpha1.RoutingTarget{
						{
							Connector: coralogixv1alpha1.NCRef{
								ResourceRef: &coralogixv1alpha1.ResourceRef{Name: connectorName},
							},
							Preset: &coralogixv1alpha1.NCRef{
								BackendRef: &coralogixv1alpha1.NCBackendRef{ID: "preset_system_generic_https_alerts_empty"},
							},
						},
					},
				},
			},
			// Per-entity-type fallback. On a non-default router the target references a connector only.
			FallbackTargets: []coralogixv1alpha1.FallbackTarget{
				{
					EntityType: "alerts",
					Target: coralogixv1alpha1.RoutingTarget{
						Connector: coralogixv1alpha1.NCRef{
							ResourceRef: &coralogixv1alpha1.ResourceRef{Name: connectorName},
						},
					},
				},
			},
		},
	}
}

func getSampleGenericHttpsConnectorForRouterGaps(name, namespace string) *coralogixv1alpha1.Connector {
	return &coralogixv1alpha1.Connector{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: coralogixv1alpha1.ConnectorSpec{
			Name:        name,
			Description: "generic https connector for global router gap-fields test",
			Type:        "genericHttps",
			ConnectorConfig: coralogixv1alpha1.ConnectorConfig{
				Fields: []coralogixv1alpha1.ConnectorConfigField{
					{FieldName: "url", Value: ptr.To("https://httpbun.org/post")},
					{FieldName: "method", Value: ptr.To("post")},
				},
			},
		},
	}
}
