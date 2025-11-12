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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("Events2Metric", Ordered, func() {
	var (
		crClient  client.Client
		e2mClient *cxsdk.Events2MetricsClient
		e2mID     string
		e2m       *coralogixv1alpha1.Events2Metric
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		e2mClient = ClientsInstance.GetCoralogixClientSet().Events2Metrics()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating E2M")
		e2mName := "logs2metric"
		e2m = &coralogixv1alpha1.Events2Metric{
			ObjectMeta: metav1.ObjectMeta{
				Name:      e2mName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.Events2MetricSpec{
				Name:              e2mName,
				Description:       ptr.To("e2m from k8s operator"),
				PermutationsLimit: ptr.To(int32(100)),
				MetricLabels: []coralogixv1alpha1.MetricLabel{
					{
						TargetLabel: "status",
						SourceField: "status",
					},
				},
				MetricFields: []coralogixv1alpha1.MetricField{
					{
						TargetBaseMetricName: "request_count",
						SourceField:          "request_count",
						Aggregations: []coralogixv1alpha1.MetricFieldAggregation{
							{
								AggType:          coralogixv1alpha1.AggregationTypeMin,
								TargetMetricName: "min_request_count",
								AggMetadata: coralogixv1alpha1.AggregationMetadata{
									Samples: &coralogixv1alpha1.SamplesMetadata{
										SampleType: coralogixv1alpha1.E2MAggSamplesSampleTypeMin,
									},
								},
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
		Eventually(func() error {
			_, err := e2mClient.Get(ctx, &cxsdk.GetE2MRequest{Id: wrapperspb.String(e2mID)})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the E2M")
		newE2MName := "updated-logs2metric"
		modifiedE2M := e2m.DeepCopy()
		modifiedE2M.Spec.Name = newE2MName
		Expect(crClient.Patch(ctx, modifiedE2M, client.MergeFrom(e2m))).To(Succeed())

		By("Verifying E2M is updated in Coralogix backend")
		Eventually(func() *wrapperspb.StringValue {
			getE2mRes, err := e2mClient.Get(ctx, &cxsdk.GetE2MRequest{Id: wrapperspb.String(e2mID)})
			Expect(err).ToNot(HaveOccurred())
			return getE2mRes.GetE2M().GetName()
		}, time.Minute, time.Second).Should(Equal(wrapperspb.String(newE2MName)))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the E2M")
		Expect(crClient.Delete(ctx, e2m)).To(Succeed())

		By("Verifying E2M is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := e2mClient.Get(ctx, &cxsdk.GetE2MRequest{Id: wrapperspb.String(e2mID)})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})
