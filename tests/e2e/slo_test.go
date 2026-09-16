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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	slos "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/slos_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

// sloAPMService names a service from the tenant's APM Service Catalog. The APM SLO specs
// cannot run without one, because the Coralogix API rejects service names that are not in
// the catalog.
var sloAPMService = os.Getenv("CORALOGIX_SLO_APM_SERVICE")

// requireSLOAPMService skips the APM SLO specs locally and fails them in CI. A repository
// secret that exists but is not mapped into a workflow's env block looks exactly like a
// missing one, so CI has to be loud about it instead of skipping.
func requireSLOAPMService() {
	if sloAPMService != "" {
		return
	}

	if os.Getenv("CI") != "" {
		Fail("CORALOGIX_SLO_APM_SERVICE must be set in CI")
	}
	Skip("set CORALOGIX_SLO_APM_SERVICE to a service from the APM Service Catalog to run the APM SLO specs locally")
}

var _ = Describe("SLO", Ordered, func() {
	var (
		crClient   client.Client
		slosClient *cxsdk.SLOsClient
		sloID      string
		slo        *coralogixv1alpha1.SLO
		sloName    = "slo-sample"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		slosClient = ClientsInstance.GetCoralogixClientSet().SLOs()
		slo = getSampleSlo(sloName, testNamespace)
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating SLO")
		Expect(crClient.Create(ctx, slo)).To(Succeed())

		By("Fetching the SLO ID and verifying it is synced with the backend")
		fetchedSlo := &coralogixv1alpha1.SLO{}
		Eventually(func(g Gomega) {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: sloName, Namespace: testNamespace}, fetchedSlo)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedSlo.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedSlo.Status.PrintableStatus).To(Equal("RemoteSynced"))

			g.Expect(fetchedSlo.Status.ID).ToNot(BeNil())
			sloID = *fetchedSlo.Status.ID
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying SLO exists in Coralogix backend")
		Eventually(func() error {
			_, err := slosClient.Get(ctx, &cxsdk.GetServiceSloRequest{
				Id: sloID,
			})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the slo")
		newSloName := uniqueName("slo-sample-updated")
		modifiedSlo := slo.DeepCopy()
		modifiedSlo.Spec.Name = newSloName
		Expect(crClient.Patch(ctx, modifiedSlo, client.MergeFrom(slo))).To(Succeed())

		By("Verifying slo is updated in Coralogix backend")
		Eventually(func() string {
			getSloRes, err := slosClient.Get(ctx, &cxsdk.GetServiceSloRequest{
				Id: sloID,
			})
			Expect(err).ToNot(HaveOccurred())
			return getSloRes.Slo.Name
		}, time.Minute, time.Second).Should(Equal(newSloName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the slo")
		Expect(crClient.Delete(ctx, slo)).To(Succeed())

		By("Verifying slo is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := slosClient.Get(ctx, &cxsdk.GetServiceSloRequest{
				Id: sloID,
			})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})

func getSampleSlo(name, namespace string) *coralogixv1alpha1.SLO {
	timeFrame := coralogixv1alpha1.SloTimeFrame7d
	return &coralogixv1alpha1.SLO{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.SLOSpec{
			Name:        name,
			Description: ptr.To("This is a sample slo"),
			Labels: &map[string]string{
				"team": "e2e-test",
			},
			SliType: coralogixv1alpha1.SliType{
				RequestBasedMetricSli: &coralogixv1alpha1.RequestBasedMetricSli{
					GoodEvents: coralogixv1alpha1.SloMetricEvent{
						Query: "sum(rate(coralogix_logs_events_total{app=\"coralogix-slo-example\", status=\"success\"}[1m]))",
					},
					TotalEvents: coralogixv1alpha1.SloMetricEvent{
						Query: "sum(rate(coralogix_logs_events_total{app=\"coralogix-slo-example\", status=\"success\"}[1m]))",
					},
					GroupByLabels: []string{"app", "status"},
				},
			},
			TargetThresholdPercentage: *resource.NewQuantity(10, resource.DecimalSI),
			Window: coralogixv1alpha1.SloWindow{
				TimeFrame: &timeFrame,
			},
		},
	}
}

var _ = Describe("SLO validation", func() {
	var crClient client.Client

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
	})

	It("Should be rejected when window is unspecified", func(ctx context.Context) {
		By("Creating an SLO with a window the API has no implementation for")
		slo := getSampleWindowBasedSlo(uniqueName("slo-invalid-window"))
		slo.Spec.SliType.WindowBasedMetricSli.Window = "unspecified"

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.sliType.windowBasedMetric.window"))
	})

	It("Should be rejected when no SLI type is set", func(ctx context.Context) {
		By("Creating an SLO with an empty sliType")
		slo := getSampleWindowBasedSlo(uniqueName("slo-no-sli"))
		slo.Spec.SliType = coralogixv1alpha1.SliType{}

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of requestBasedMetric, windowBasedMetric or apmSli must be set"))
	})

	It("Should be rejected when two SLI types are set", func(ctx context.Context) {
		By("Creating an SLO with both windowBasedMetric and apmSli")
		slo := getSampleWindowBasedSlo(uniqueName("slo-two-slis"))
		slo.Spec.SliType.ApmSli = &coralogixv1alpha1.ApmSli{
			Services:    []string{"any-service"},
			ErrorConfig: &coralogixv1alpha1.ApmErrorSli{},
		}

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of requestBasedMetric, windowBasedMetric or apmSli must be set"))
	})

	It("Should be rejected when apmSli has neither errorConfig nor latencyConfig", func(ctx context.Context) {
		By("Creating an APM SLO with no SLI branch")
		slo := getSampleAPMErrorSlo(uniqueName("slo-apm-no-branch"), "any-service")
		slo.Spec.SliType.ApmSli.ErrorConfig = nil

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of errorConfig or latencyConfig must be set"))
	})

	It("Should be rejected when apmSli has both errorConfig and latencyConfig", func(ctx context.Context) {
		By("Creating an APM SLO with both SLI branches")
		slo := getSampleAPMErrorSlo(uniqueName("slo-apm-both-branches"), "any-service")
		slo.Spec.SliType.ApmSli.LatencyConfig = &coralogixv1alpha1.ApmLatencySli{
			TimeWindow: "5m",
			Average:    &coralogixv1alpha1.ApmLatencyAverage{},
		}

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of errorConfig or latencyConfig must be set"))
	})

	It("Should be rejected when latencyConfig has neither quantile nor average", func(ctx context.Context) {
		By("Creating an APM latency SLO with no query type")
		slo := getSampleAPMLatencySlo(uniqueName("slo-apm-no-query-type"), "any-service")
		slo.Spec.SliType.ApmSli.LatencyConfig.Quantile = nil

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of quantile or average must be set"))
	})

	It("Should be rejected when latencyConfig has both quantile and average", func(ctx context.Context) {
		By("Creating an APM latency SLO with both query types")
		slo := getSampleAPMLatencySlo(uniqueName("slo-apm-both-query-types"), "any-service")
		slo.Spec.SliType.ApmSli.LatencyConfig.Average = &coralogixv1alpha1.ApmLatencyAverage{}

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of quantile or average must be set"))
	})

	It("Should be rejected when productType is apm without an apmSli", func(ctx context.Context) {
		By("Creating a window-based SLO with productType apm")
		slo := getSampleWindowBasedSlo(uniqueName("slo-apm-product-no-sli"))
		slo.Spec.ProductType = ptr.To(coralogixv1alpha1.SloProductType("apm"))

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("productType 'apm' requires sliType.apmSli"))
	})

	It("Should accept an apmSli filter with an empty values list", func(ctx context.Context) {
		By("Creating an APM SLO with a filter that has no values")
		slo := getSampleAPMErrorSlo(uniqueName("slo-apm-empty-filter-values"), "any-service")
		slo.Spec.SliType.ApmSli.Filters = []coralogixv1alpha1.ApmFilter{
			{Key: "http.method"},
		}

		// The API accepts and stores an empty values list, so admission must not reject
		// it. The reconcile then fails on the unknown service name, which is expected.
		Expect(crClient.Create(ctx, slo)).To(Succeed())
		Expect(crClient.Delete(ctx, slo)).To(Succeed())
	})

	It("Should be rejected when apmSli has no services", func(ctx context.Context) {
		By("Creating an APM SLO with an empty services list")
		slo := getSampleAPMErrorSlo(uniqueName("slo-apm-no-services"), "any-service")
		slo.Spec.SliType.ApmSli.Services = []string{}

		err := crClient.Create(ctx, slo)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.sliType.apmSli.services"))
	})

	It("Should fail the reconcile when windowBasedMetric has no query", func(ctx context.Context) {
		By("Creating a window-based SLO with no query")
		name := uniqueName("slo-no-query")
		slo := getSampleWindowBasedSlo(name)
		slo.Spec.SliType.WindowBasedMetricSli.Query = nil

		// query is optional in the CRD, so admission accepts this. The reconcile must
		// report a clear error rather than panic on the nil dereference.
		Expect(crClient.Create(ctx, slo)).To(Succeed())
		DeferCleanup(func(ctx context.Context) {
			Expect(client.IgnoreNotFound(crClient.Delete(ctx, slo))).To(Succeed())
		})

		By("Waiting for the RemoteSynced condition to report the error")
		fetched := &coralogixv1alpha1.SLO{}
		Eventually(func(g Gomega) string {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, fetched)).To(Succeed())
			condition := meta.FindStatusCondition(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)
			g.Expect(condition).ToNot(BeNil())
			return condition.Message
		}, time.Minute, time.Second).Should(ContainSubstring("windowBasedMetric.query is required"))
	})
})

var _ = Describe("SLO window-based", Ordered, func() {
	var (
		crClient   client.Client
		slosClient *slos.SlosServiceAPIService
		sloID      string
		slo        *coralogixv1alpha1.SLO
		sloName    string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		slosClient = newOpenAPIClientSet().SLOs()
		sloName = uniqueName("slo-window-based")
		slo = getSampleWindowBasedSlo(sloName)
	})

	It("Should reconcile a window-based SLO with missingDataStrategy and ownershipTags", func(ctx context.Context) {
		By("Creating the SLO")
		Expect(crClient.Create(ctx, slo)).To(Succeed())

		By("Waiting for the SLO to be synced")
		sloID = waitForSyncedSLOID(ctx, crClient, sloName)

		By("Verifying the stored SLO carries the new fields")
		getRes, _, err := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
		Expect(err).ToNot(HaveOccurred())
		Expect(getRes.Slo.WindowBasedMetricSli).ToNot(BeNil())
		Expect(getRes.Slo.WindowBasedMetricSli.GetMissingDataStrategy()).
			To(Equal(slos.MISSINGDATASTRATEGY_MISSING_DATA_STRATEGY_GOOD))
		Expect(getRes.Slo.OwnershipTags).ToNot(BeNil())
		Expect(getRes.Slo.OwnershipTags.Environment.GetStaticValues()).To(Equal([]string{"production"}))
		Expect(getRes.Slo.OwnershipTags.Team.GetLabelKeys()).To(Equal([]string{"team"}))
	})

	It("Should apply a changed missingDataStrategy on the replace path", func(ctx context.Context) {
		By("Patching missingDataStrategy to bad")
		modified := slo.DeepCopy()
		modified.Spec.SliType.WindowBasedMetricSli.MissingDataStrategy =
			ptr.To(coralogixv1alpha1.MissingDataStrategy("bad"))
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(slo))).To(Succeed())

		By("Verifying the replace reached the backend")
		Eventually(func(g Gomega) slos.MissingDataStrategy {
			getRes, _, err := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getRes.Slo.WindowBasedMetricSli).ToNot(BeNil())
			return getRes.Slo.WindowBasedMetricSli.GetMissingDataStrategy()
		}, time.Minute, time.Second).Should(Equal(slos.MISSINGDATASTRATEGY_MISSING_DATA_STRATEGY_BAD))
		slo = modified
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the SLO")
		Expect(crClient.Delete(ctx, slo)).To(Succeed())

		By("Verifying the SLO is gone from the backend")
		Eventually(func() int {
			_, httpResp, _ := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
			if httpResp == nil {
				return 0
			}
			return httpResp.StatusCode
		}, time.Minute, time.Second).Should(Equal(404))
	})
})

var _ = Describe("SLO APM", Ordered, func() {
	var (
		crClient   client.Client
		slosClient *slos.SlosServiceAPIService
		sloID      string
		slo        *coralogixv1alpha1.SLO
		sloName    string
	)

	BeforeAll(func() {
		requireSLOAPMService()
		crClient = ClientsInstance.GetControllerRuntimeClient()
		slosClient = newOpenAPIClientSet().SLOs()
		sloName = uniqueName("slo-apm-latency")
		slo = getSampleAPMLatencySlo(sloName, sloAPMService)
	})

	It("Should reconcile an APM latency SLO", func(ctx context.Context) {
		By("Creating the SLO")
		Expect(crClient.Create(ctx, slo)).To(Succeed())

		By("Waiting for the SLO to be synced")
		sloID = waitForSyncedSLOID(ctx, crClient, sloName)

		By("Verifying the stored SLO is an APM latency SLO")
		getRes, _, err := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
		Expect(err).ToNot(HaveOccurred())
		Expect(getRes.Slo.GetProductType()).To(Equal(slos.SLOPRODUCTTYPE_SLO_PRODUCT_TYPE_APM))
		Expect(getRes.Slo.ApmSli).ToNot(BeNil())
		Expect(getRes.Slo.ApmSli.GetServices()).To(Equal([]string{sloAPMService}))
		Expect(getRes.Slo.ApmSli.LatencyConfig).ToNot(BeNil())
		Expect(getRes.Slo.ApmSli.LatencyConfig.Quantile).ToNot(BeNil())
		Expect(getRes.Slo.ApmSli.LatencyConfig.Quantile.GetPercentile()).To(BeNumerically("~", 0.95, 0.001))
	})

	It("Should switch the latency query type from quantile to average", func(ctx context.Context) {
		By("Patching latencyConfig from quantile to average")
		modified := slo.DeepCopy()
		modified.Spec.SliType.ApmSli.LatencyConfig.Quantile = nil
		modified.Spec.SliType.ApmSli.LatencyConfig.Average = &coralogixv1alpha1.ApmLatencyAverage{}
		// A merge patch cannot remove a key, so replace the object outright.
		Expect(crClient.Update(ctx, modified)).To(Succeed())

		By("Verifying the replace reached the backend")
		Eventually(func(g Gomega) bool {
			getRes, _, err := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getRes.Slo.ApmSli).ToNot(BeNil())
			g.Expect(getRes.Slo.ApmSli.LatencyConfig).ToNot(BeNil())
			return getRes.Slo.ApmSli.LatencyConfig.Average != nil &&
				getRes.Slo.ApmSli.LatencyConfig.Quantile == nil
		}, time.Minute, time.Second).Should(BeTrue())
		slo = modified
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the SLO")
		Expect(crClient.Delete(ctx, slo)).To(Succeed())

		By("Verifying the SLO is gone from the backend")
		Eventually(func() int {
			_, httpResp, _ := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
			if httpResp == nil {
				return 0
			}
			return httpResp.StatusCode
		}, time.Minute, time.Second).Should(Equal(404))
	})
})

var _ = Describe("SLO APM error", Ordered, func() {
	var (
		crClient   client.Client
		slosClient *slos.SlosServiceAPIService
		sloID      string
		slo        *coralogixv1alpha1.SLO
		sloName    string
	)

	BeforeAll(func() {
		requireSLOAPMService()
		crClient = ClientsInstance.GetControllerRuntimeClient()
		slosClient = newOpenAPIClientSet().SLOs()
		sloName = uniqueName("slo-apm-error")
		slo = getSampleAPMErrorSlo(sloName, sloAPMService)
		slo.Spec.SliType.ApmSli.Filters = []coralogixv1alpha1.ApmFilter{
			{Key: "http.method", Values: []string{"POST"}},
		}
		slo.Spec.SliType.ApmSli.GroupingKeys = []string{"http.route"}
	})

	It("Should reconcile an APM error SLO", func(ctx context.Context) {
		By("Creating the SLO")
		Expect(crClient.Create(ctx, slo)).To(Succeed())

		By("Waiting for the SLO to be synced")
		sloID = waitForSyncedSLOID(ctx, crClient, sloName)

		By("Verifying errorConfig round-trips as an empty object")
		getRes, _, err := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
		Expect(err).ToNot(HaveOccurred())
		Expect(getRes.Slo.ApmSli).ToNot(BeNil())
		// A nil map would have been dropped from the request, which the API rejects as
		// "Unsupported APM SLI type". Seeing it come back proves the empty map was sent.
		Expect(getRes.Slo.ApmSli.ErrorConfig).ToNot(BeNil())
		Expect(getRes.Slo.ApmSli.LatencyConfig).To(BeNil())
		Expect(getRes.Slo.ApmSli.GetGroupingKeys()).To(Equal([]string{"http.route"}))
		Expect(getRes.Slo.ApmSli.GetFilters()).To(HaveLen(1))
		Expect(getRes.Slo.ApmSli.GetFilters()[0].GetKey()).To(Equal("http.method"))
		Expect(getRes.Slo.ApmSli.GetFilters()[0].GetValues()).To(Equal([]string{"POST"}))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the SLO")
		Expect(crClient.Delete(ctx, slo)).To(Succeed())

		By("Verifying the SLO is gone from the backend")
		Eventually(func() int {
			_, httpResp, _ := slosClient.SlosServiceGetSlo(ctx, sloID).Execute()
			if httpResp == nil {
				return 0
			}
			return httpResp.StatusCode
		}, time.Minute, time.Second).Should(Equal(404))
	})
})

// waitForSyncedSLOID waits for the operator to reconcile the SLO and returns its remote ID.
func waitForSyncedSLOID(ctx context.Context, crClient client.Client, name string) string {
	var sloID string
	fetched := &coralogixv1alpha1.SLO{}
	Eventually(func(g Gomega) {
		g.Expect(crClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNamespace}, fetched)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
		g.Expect(fetched.Status.ID).ToNot(BeNil())
		sloID = *fetched.Status.ID
	}, time.Minute, time.Second).Should(Succeed())
	return sloID
}

func getSampleWindowBasedSlo(name string) *coralogixv1alpha1.SLO {
	timeFrame := coralogixv1alpha1.SloTimeFrame7d
	return &coralogixv1alpha1.SLO{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.SLOSpec{
			Name:        name,
			Description: ptr.To("This is a sample window-based slo"),
			SliType: coralogixv1alpha1.SliType{
				WindowBasedMetricSli: &coralogixv1alpha1.WindowBasedMetricSli{
					Query: &coralogixv1alpha1.SloMetricEvent{
						Query: "avg(avg_over_time(request_duration_seconds[1m]))",
					},
					Window:              "5m",
					ComparisonOperator:  "greaterThan",
					Threshold:           resource.MustParse("0.5"),
					MissingDataStrategy: ptr.To(coralogixv1alpha1.MissingDataStrategy("good")),
				},
			},
			OwnershipTags: &coralogixv1alpha1.SloOwnershipTags{
				Environment: &coralogixv1alpha1.SloOwnershipTag{
					StaticValues: []string{"production"},
				},
				Team: &coralogixv1alpha1.SloOwnershipTag{
					LabelKeys: []string{"team"},
				},
			},
			TargetThresholdPercentage: *resource.NewQuantity(99, resource.DecimalSI),
			Window: coralogixv1alpha1.SloWindow{
				TimeFrame: &timeFrame,
			},
		},
	}
}

func getSampleAPMErrorSlo(name, service string) *coralogixv1alpha1.SLO {
	slo := getSampleAPMSlo(name, service)
	slo.Spec.SliType.ApmSli.ErrorConfig = &coralogixv1alpha1.ApmErrorSli{}
	return slo
}

func getSampleAPMLatencySlo(name, service string) *coralogixv1alpha1.SLO {
	slo := getSampleAPMSlo(name, service)
	slo.Spec.SliType.ApmSli.LatencyConfig = &coralogixv1alpha1.ApmLatencySli{
		TimeWindow: "5m",
		// milliseconds, so 500ms
		Threshold: ptr.To(resource.MustParse("500")),
		Quantile: &coralogixv1alpha1.ApmLatencyQuantile{
			Percentile: ptr.To(resource.MustParse("0.95")),
		},
	}
	return slo
}

// getSampleAPMSlo returns an APM SLO with no SLI branch selected. The callers pick one,
// because a CEL rule requires exactly one of errorConfig or latencyConfig.
func getSampleAPMSlo(name, service string) *coralogixv1alpha1.SLO {
	timeFrame := coralogixv1alpha1.SloTimeFrame7d
	return &coralogixv1alpha1.SLO{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.SLOSpec{
			Name:        name,
			Description: ptr.To("This is a sample apm slo"),
			ProductType: ptr.To(coralogixv1alpha1.SloProductType("apm")),
			SliType: coralogixv1alpha1.SliType{
				ApmSli: &coralogixv1alpha1.ApmSli{
					Services: []string{service},
				},
			},
			TargetThresholdPercentage: *resource.NewQuantity(99, resource.DecimalSI),
			Window: coralogixv1alpha1.SloWindow{
				TimeFrame: &timeFrame,
			},
		},
	}
}
