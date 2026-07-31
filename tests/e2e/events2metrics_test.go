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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	events2metrics "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/events2metrics_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

func uniqueE2MMetricName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

var _ = Describe("Events2Metric", Ordered, func() {
	var (
		crClient  client.Client
		e2mClient *events2metrics.Events2MetricsServiceAPIService
		e2mID     string
		e2m       *coralogixv1alpha1.Events2Metric
		baseName  string
		e2mName   string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		e2mClient = newOpenAPIClientSet().Events2Metrics()
		baseName = uniqueE2MMetricName("request_count")
		e2mName = fmt.Sprintf("logs2metric-%d", time.Now().UnixNano())
		// Register in BeforeAll so cleanup runs after the Ordered container, not after the create It.
		DeferCleanup(func(ctx context.Context) {
			_ = crClient.Delete(ctx, &coralogixv1alpha1.Events2Metric{
				ObjectMeta: metav1.ObjectMeta{Name: e2mName, Namespace: testNamespace},
			})
		})
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating E2M")
		e2m = &coralogixv1alpha1.Events2Metric{
			ObjectMeta: metav1.ObjectMeta{
				Name:      e2mName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.Events2MetricSpec{
				Name:              e2mName,
				Description:       ptr.To("e2m from k8s operator"),
				DataSource:        ptr.To("default/logs"),
				PermutationsLimit: ptr.To(int32(100)),
				MetricLabels: []coralogixv1alpha1.MetricLabel{
					{
						TargetLabel: "status",
						SourceField: "status",
					},
				},
				MetricFields: []coralogixv1alpha1.MetricField{
					{
						TargetBaseMetricName: baseName,
						SourceField:          "request_count",
						Aggregations: []coralogixv1alpha1.MetricFieldAggregation{
							{
								AggType:          coralogixv1alpha1.AggregationTypeMin,
								TargetMetricName: "min_request_count",
							},
						},
					},
				},
				Query: coralogixv1alpha1.E2MQuery{
					Logs: &coralogixv1alpha1.E2MQueryLogs{
						Lucene:                 ptr.To("status:200 AND request_count:[* TO *]"),
						Alias:                  ptr.To("e2m-logs"),
						ApplicationNameFilters: []string{"test-app"},
						SubsystemNameFilters:   []string{"test-subsystem"},
						SeverityFilters:        []coralogixv1alpha1.L2MSeverity{coralogixv1alpha1.L2MSeverityCritical, coralogixv1alpha1.L2MSeverityError},
					},
				},
			},
		}

		Expect(crClient.Create(ctx, e2m)).To(Succeed())

		By("Fetching the E2M ID and Conditions")
		fetchedE2M := &coralogixv1alpha1.Events2Metric{}
		Eventually(func(g Gomega) {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: e2mName, Namespace: testNamespace}, fetchedE2M)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedE2M.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedE2M.Status.PrintableStatus).To(Equal("RemoteSynced"))
			g.Expect(fetchedE2M.Status.Id).ToNot(BeNil())
			e2mID = *fetchedE2M.Status.Id
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying E2M exists in Coralogix backend")
		Eventually(func(g Gomega) {
			getE2mRes, _, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getE2mRes.E2m).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.Description).ToNot(BeNil())
			g.Expect(*getE2mRes.E2m.Description).To(Equal("e2m from k8s operator"))
			g.Expect(getE2mRes.E2m.DataSource).ToNot(BeNil())
			g.Expect(*getE2mRes.E2m.DataSource).To(Equal("default/logs"))
			g.Expect(getE2mRes.E2m.Permutations).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.Permutations.Limit).To(Equal(int32(100)))
			g.Expect(getE2mRes.E2m.LogsQuery).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.LogsQuery.Lucene).ToNot(BeNil())
			g.Expect(*getE2mRes.E2m.LogsQuery.Lucene).To(Equal("status:200 AND request_count:[* TO *]"))
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the E2M beyond rename")
		// Account-wide E2M names must be unique; leftovers from interrupted runs collide on a fixed name.
		newE2MName := fmt.Sprintf("updated-logs2metric-%d", time.Now().UnixNano())
		modifiedE2M := e2m.DeepCopy()
		modifiedE2M.Spec.Name = newE2MName
		modifiedE2M.Spec.Description = ptr.To("updated description")
		modifiedE2M.Spec.PermutationsLimit = ptr.To(int32(250))
		modifiedE2M.Spec.MetricLabels = []coralogixv1alpha1.MetricLabel{
			{TargetLabel: "status", SourceField: "status"},
			{TargetLabel: "method", SourceField: "method"},
		}
		modifiedE2M.Spec.MetricFields = []coralogixv1alpha1.MetricField{
			{
				TargetBaseMetricName: baseName,
				SourceField:          "request_count",
				Aggregations: []coralogixv1alpha1.MetricFieldAggregation{
					{
						Enabled:          true,
						AggType:          coralogixv1alpha1.AggregationTypeMin,
						TargetMetricName: "min_request_count",
					},
					{
						Enabled:          false,
						AggType:          coralogixv1alpha1.AggregationTypeMax,
						TargetMetricName: "max_request_count",
					},
				},
			},
		}
		Expect(crClient.Patch(ctx, modifiedE2M, client.MergeFrom(e2m))).To(Succeed())
		e2m = modifiedE2M

		By("Verifying E2M is updated in Coralogix backend")
		Eventually(func(g Gomega) {
			getE2mRes, _, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getE2mRes.E2m).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.Name).To(Equal(newE2MName))
			g.Expect(getE2mRes.E2m.Description).ToNot(BeNil())
			g.Expect(*getE2mRes.E2m.Description).To(Equal("updated description"))
			g.Expect(getE2mRes.E2m.Permutations).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.Permutations.Limit).To(Equal(int32(250)))
			g.Expect(getE2mRes.E2m.MetricLabels).To(HaveLen(2))
			g.Expect(getE2mRes.E2m.MetricFields).To(HaveLen(1))
			// API expands each field into a full aggregation set; do not assert exact length.
			g.Expect(getE2mRes.E2m.MetricFields[0].Aggregations).ToNot(BeEmpty())

			disabledMaxFound := false
			for _, agg := range getE2mRes.E2m.MetricFields[0].Aggregations {
				if agg.AggType != nil && *agg.AggType == events2metrics.AGGTYPE_AGG_TYPE_MAX &&
					agg.Enabled != nil && !*agg.Enabled {
					disabledMaxFound = true
				}
			}
			g.Expect(disabledMaxFound).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should clear optional fields successfully", func(ctx context.Context) {
		By("Clearing optional string fields and permutationsLimit")
		fetched := &coralogixv1alpha1.Events2Metric{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: e2m.Name, Namespace: testNamespace}, fetched)).To(Succeed())
		fetched.Spec.Description = nil
		fetched.Spec.DataSource = nil
		fetched.Spec.PermutationsLimit = nil
		Expect(fetched.Spec.Query.Logs).ToNot(BeNil())
		fetched.Spec.Query.Logs.Lucene = nil
		fetched.Spec.Query.Logs.Alias = nil
		Expect(crClient.Update(ctx, fetched)).To(Succeed())
		e2m = fetched

		By("Verifying replace succeeds and optional fields are cleared or omitted remotely")
		Eventually(func(g Gomega) {
			current := &coralogixv1alpha1.Events2Metric{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: e2m.Name, Namespace: testNamespace}, current)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(current.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(current.Spec.Description).To(BeNil())
			g.Expect(current.Spec.DataSource).To(BeNil())
			g.Expect(current.Spec.PermutationsLimit).To(BeNil())
			g.Expect(current.Spec.Query.Logs).ToNot(BeNil())
			g.Expect(current.Spec.Query.Logs.Lucene).To(BeNil())
			g.Expect(current.Spec.Query.Logs.Alias).To(BeNil())

			getE2mRes, _, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getE2mRes.E2m).ToNot(BeNil())
			if getE2mRes.E2m.Description != nil {
				g.Expect(*getE2mRes.E2m.Description).To(BeEmpty())
			}
			if getE2mRes.E2m.DataSource != nil {
				g.Expect(*getE2mRes.E2m.DataSource).To(BeEmpty())
			}
			// Replace omits permutations when Spec.PermutationsLimit is nil.
			// The API resets the limit to its default (30000), not the prior value.
			g.Expect(getE2mRes.E2m.Permutations).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.Permutations.Limit).To(Equal(int32(30000)))
			g.Expect(getE2mRes.E2m.LogsQuery).ToNot(BeNil())
			if getE2mRes.E2m.LogsQuery.Lucene != nil {
				g.Expect(*getE2mRes.E2m.LogsQuery.Lucene).To(BeEmpty())
			}
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the E2M")
		Expect(crClient.Delete(ctx, e2m)).To(Succeed())

		By("Verifying E2M is deleted from Coralogix backend")
		Eventually(func(g Gomega) {
			_, httpResp, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).To(HaveOccurred())
			g.Expect(cxsdk.IsNotFound(cxsdk.NewAPIError(httpResp, err))).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed())
	})
})

var _ = Describe("Events2Metric spans samples histogram", Ordered, func() {
	var (
		crClient  client.Client
		e2mClient *events2metrics.Events2MetricsServiceAPIService
		e2mName   string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		e2mClient = newOpenAPIClientSet().Events2Metrics()
		e2mName = fmt.Sprintf("spans2metric-%d", time.Now().UnixNano())
		DeferCleanup(func(ctx context.Context) {
			_ = crClient.Delete(ctx, &coralogixv1alpha1.Events2Metric{
				ObjectMeta: metav1.ObjectMeta{Name: e2mName, Namespace: testNamespace},
			})
		})
	})

	It("Should create update and delete a spans E2M with samples and histogram aggregations", func(ctx context.Context) {
		durationMetric := uniqueE2MMetricName("span_duration")
		samplesMetric := uniqueE2MMetricName("span_samples")
		histogramMetric := uniqueE2MMetricName("span_histogram")

		e2m := &coralogixv1alpha1.Events2Metric{
			ObjectMeta: metav1.ObjectMeta{
				Name:      e2mName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.Events2MetricSpec{
				Name:              e2mName,
				Description:       ptr.To("spans e2m from k8s operator"),
				PermutationsLimit: ptr.To(int32(50)),
				MetricFields: []coralogixv1alpha1.MetricField{
					{
						TargetBaseMetricName: durationMetric,
						SourceField:          "duration",
						Aggregations: []coralogixv1alpha1.MetricFieldAggregation{
							{
								Enabled:          true,
								AggType:          coralogixv1alpha1.AggregationTypeMin,
								TargetMetricName: "min_duration",
							},
						},
					},
					{
						TargetBaseMetricName: samplesMetric,
						SourceField:          "duration",
						Aggregations: []coralogixv1alpha1.MetricFieldAggregation{
							{
								Enabled:          true,
								AggType:          coralogixv1alpha1.AggregationTypeSamples,
								TargetMetricName: "samples_duration",
								AggMetadata: coralogixv1alpha1.AggregationMetadata{
									Samples: &coralogixv1alpha1.SamplesMetadata{
										SampleType: coralogixv1alpha1.E2MAggSamplesSampleTypeMax,
									},
								},
							},
						},
					},
					{
						TargetBaseMetricName: histogramMetric,
						SourceField:          "duration",
						Aggregations: []coralogixv1alpha1.MetricFieldAggregation{
							{
								Enabled:          true,
								AggType:          coralogixv1alpha1.AggregationTypeHistogram,
								TargetMetricName: "histogram_duration",
								AggMetadata: coralogixv1alpha1.AggregationMetadata{
									Histogram: &coralogixv1alpha1.HistogramMetadata{
										Buckets: []resource.Quantity{
											resource.MustParse("0.1"),
											resource.MustParse("5.5"),
											resource.MustParse("100"),
										},
									},
								},
							},
						},
					},
				},
				Query: coralogixv1alpha1.E2MQuery{
					Spans: &coralogixv1alpha1.E2MQuerySpans{
						Lucene:                 ptr.To("service:nginx"),
						ApplicationNameFilters: []string{"test-app"},
						SubsystemNameFilters:   []string{"test-subsystem"},
						ActionFilters:          []string{"GET"},
						ServiceFilters:         []string{"nginx"},
					},
				},
			},
		}

		Expect(crClient.Create(ctx, e2m)).To(Succeed())

		var e2mID string
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.Events2Metric{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: e2mName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.Id).ToNot(BeNil())
			e2mID = *fetched.Status.Id
		}, time.Minute, time.Second).Should(Succeed())

		Eventually(func(g Gomega) {
			getE2mRes, _, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getE2mRes.E2m).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.Type).To(Equal(events2metrics.E2MTYPE_E2_M_TYPE_SPANS2_METRICS))
			g.Expect(getE2mRes.E2m.Description).ToNot(BeNil())
			g.Expect(*getE2mRes.E2m.Description).To(Equal("spans e2m from k8s operator"))
			g.Expect(getE2mRes.E2m.SpansQuery).ToNot(BeNil())
			g.Expect(getE2mRes.E2m.SpansQuery.Lucene).ToNot(BeNil())
			g.Expect(*getE2mRes.E2m.SpansQuery.Lucene).To(Equal("service:nginx"))
			g.Expect(getE2mRes.E2m.MetricFields).To(HaveLen(3))

			// The API expands each metric field into a full aggregation set (default
			// histogram has empty buckets) and may rewrite targetMetricName (e.g. to
			// cx_bucket). Match by TargetBaseMetricName + AggType instead.
			var foundSamples, foundHistogram bool
			for _, field := range getE2mRes.E2m.MetricFields {
				for _, agg := range field.Aggregations {
					if agg.AggType == nil {
						continue
					}
					switch {
					case field.TargetBaseMetricName == samplesMetric &&
						*agg.AggType == events2metrics.AGGTYPE_AGG_TYPE_SAMPLES:
						foundSamples = true
						g.Expect(agg.Samples).ToNot(BeNil())
						g.Expect(agg.Samples.SampleType).ToNot(BeNil())
						g.Expect(*agg.Samples.SampleType).To(Equal(events2metrics.SAMPLETYPE_SAMPLE_TYPE_MAX))
					case field.TargetBaseMetricName == histogramMetric &&
						*agg.AggType == events2metrics.AGGTYPE_AGG_TYPE_HISTOGRAM:
						foundHistogram = true
						g.Expect(agg.Histogram).ToNot(BeNil())
						g.Expect(agg.Histogram.Buckets).To(Equal([]float32{0.1, 5.5, 100}))
					}
				}
			}
			g.Expect(foundSamples).To(BeTrue())
			g.Expect(foundHistogram).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed())

		By("Updating spans description")
		modified := e2m.DeepCopy()
		modified.Spec.Description = ptr.To("updated spans e2m")
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(e2m))).To(Succeed())

		Eventually(func(g Gomega) {
			getE2mRes, _, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getE2mRes.E2m.Description).ToNot(BeNil())
			g.Expect(*getE2mRes.E2m.Description).To(Equal("updated spans e2m"))
		}, time.Minute, time.Second).Should(Succeed())

		By("Clearing optional spans string fields")
		fetched := &coralogixv1alpha1.Events2Metric{}
		Expect(crClient.Get(ctx, types.NamespacedName{Name: e2mName, Namespace: testNamespace}, fetched)).To(Succeed())
		fetched.Spec.Description = nil
		Expect(fetched.Spec.Query.Spans).ToNot(BeNil())
		fetched.Spec.Query.Spans.Lucene = nil
		Expect(crClient.Update(ctx, fetched)).To(Succeed())
		modified = fetched

		Eventually(func(g Gomega) {
			current := &coralogixv1alpha1.Events2Metric{}
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: e2mName, Namespace: testNamespace}, current)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(current.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(current.Spec.Description).To(BeNil())
			g.Expect(current.Spec.Query.Spans).ToNot(BeNil())
			g.Expect(current.Spec.Query.Spans.Lucene).To(BeNil())

			getE2mRes, _, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getE2mRes.E2m).ToNot(BeNil())
			if getE2mRes.E2m.Description != nil {
				g.Expect(*getE2mRes.E2m.Description).To(BeEmpty())
			}
			g.Expect(getE2mRes.E2m.SpansQuery).ToNot(BeNil())
			if getE2mRes.E2m.SpansQuery.Lucene != nil {
				g.Expect(*getE2mRes.E2m.SpansQuery.Lucene).To(BeEmpty())
			}
		}, time.Minute, time.Second).Should(Succeed())

		By("Deleting the E2M")
		Expect(crClient.Delete(ctx, modified)).To(Succeed())
		Eventually(func(g Gomega) {
			err := crClient.Get(ctx, types.NamespacedName{Name: e2mName, Namespace: testNamespace}, &coralogixv1alpha1.Events2Metric{})
			g.Expect(client.IgnoreNotFound(err)).To(Succeed())
			g.Expect(err).To(HaveOccurred())
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying E2M is deleted from Coralogix backend")
		Eventually(func(g Gomega) {
			_, httpResp, err := e2mClient.Events2MetricServiceGetE2M(ctx, e2mID).Execute()
			g.Expect(err).To(HaveOccurred())
			g.Expect(cxsdk.IsNotFound(cxsdk.NewAPIError(httpResp, err))).To(BeTrue())
		}, time.Minute, time.Second).Should(Succeed())
	})
})
