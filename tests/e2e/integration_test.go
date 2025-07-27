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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("Integration", Ordered, func() {
	var (
		crClient           client.Client
		integrationsClient *cxsdk.IntegrationsClient
		integrationID      string
		integration        *coralogixv1alpha1.Integration
		integrationName    = "aws-metrics-collector-integration"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		integrationsClient = ClientsInstance.GetCoralogixClientSet().Integrations()
		integration = &coralogixv1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      integrationName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.IntegrationSpec{
				IntegrationKey: "aws-metrics-collector",
				Version:        "0.1.0",
				Parameters: runtime.RawExtension{
					Raw: []byte(`{
						"ApplicationName": "cxo",
						"SubsystemName": "aws-metrics-collector",
						"MetricNamespaces": [
							"AWS/S3"
						],
						"AwsRoleArn": "arn:aws:iam::123456789012:role/example-role",
						"IntegrationName": "cxo-integration-setup",
						"AwsRegion": "eu-north-1",
						"WithAggregations": false,
						"EnrichWithTags": true
					}`),
				},
			},
		}
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating the Integration")
		Expect(crClient.Create(ctx, integration)).To(Succeed())

		By("Fetching the Integration ID")
		fetchedIntegration := &coralogixv1alpha1.Integration{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: integrationName, Namespace: testNamespace}, fetchedIntegration)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedIntegration.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			if fetchedIntegration.Status.Id != nil {
				integrationID = *fetchedIntegration.Status.Id
				return nil
			}
			return fmt.Errorf("integration ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Integration exists in Coralogix backend")
		Eventually(func() error {
			_, err := integrationsClient.Get(ctx, &cxsdk.GetDeployedIntegrationRequest{
				IntegrationId: wrapperspb.String(integrationID),
			})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the Integration")
		modifiedIntegration := integration.DeepCopy()
		modifiedIntegration.Spec.Parameters = runtime.RawExtension{
			Raw: []byte(`{
						"ApplicationName": "cxo-updated",
						"SubsystemName": "aws-metrics-collector",
						"MetricNamespaces": [
							"AWS/S3"
						],
						"AwsRoleArn": "arn:aws:iam::123456789012:role/example-role",
						"IntegrationName": "cxo-integration-setup",
						"AwsRegion": "eu-north-1",
						"WithAggregations": false,
						"EnrichWithTags": true
					}`),
		}
		Expect(crClient.Patch(ctx, modifiedIntegration, client.MergeFrom(integration))).To(Succeed())

		By("Verifying Integration is updated in Coralogix backend")
		Eventually(func() string {
			getIntegrationRes, err := integrationsClient.Get(ctx, &cxsdk.GetDeployedIntegrationRequest{
				IntegrationId: wrapperspb.String(integrationID),
			})
			Expect(err).ToNot(HaveOccurred())
			parameters := getIntegrationRes.Integration.GetParameters()
			for _, parameter := range parameters {
				if parameter.Key == "ApplicationName" {
					return parameter.Value.(*cxsdk.IntegrationParameterStringValue).StringValue.Value
				}
			}
			return ""
		}, time.Minute, time.Second).Should(Equal("cxo-updated"))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Integration")
		Expect(crClient.Delete(ctx, integration)).To(Succeed())

		By("Verifying Integration is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := integrationsClient.Get(ctx, &cxsdk.GetDeployedIntegrationRequest{
				IntegrationId: wrapperspb.String(integrationID),
			})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})
