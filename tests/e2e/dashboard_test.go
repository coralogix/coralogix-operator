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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	dashboards "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/dashboard_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("Dashboard", Ordered, func() {
	var (
		crClient         client.Client
		dashboardsClient *dashboards.DashboardServiceAPIService
		dashboard        *coralogixv1alpha1.Dashboard
		dashboardName    = "dashboard-sample"
		dashboardID      string
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		dashboardsClient = newDashboardOpenAPIClientSet().Dashboards()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Dashboard")
		dashboardJson := getDashboardJson("Test Dashboard")
		dashboard = &coralogixv1alpha1.Dashboard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dashboardName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.DashboardSpec{
				Json: &dashboardJson,
			},
		}
		Expect(crClient.Create(ctx, dashboard)).To(Succeed())

		By("Fetching the Dashboard ID")
		fetchedDashboard := &coralogixv1alpha1.Dashboard{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: dashboardName, Namespace: testNamespace}, fetchedDashboard)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedDashboard.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetchedDashboard.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetchedDashboard.Status.ID != nil {
				dashboardID = *fetchedDashboard.Status.ID
				return nil
			}
			return fmt.Errorf("Dashboard ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Dashboard exists in Coralogix backend")
		var getResponse *dashboards.GetDashboardResponse
		Eventually(func() error {
			var httpResp *http.Response
			var err error
			getResponse, httpResp, err = dashboardsClient.DashboardsServiceGetDashboard(ctx, dashboardID).Execute()
			return cxsdk.NewAPIError(httpResp, err)
		}, time.Minute, time.Second).Should(Succeed())

		// arcDisplay, showMinMax and layoutColumns are absent from the protobuf the gRPC
		// client was compiled against, so they were silently dropped before the request
		// was built. Asserting them here is why this resource is verified over REST.
		By("Verifying fields the gRPC client used to drop were persisted")
		remoteWidget := getResponse.Dashboard.Layout.Sections[0].Rows[0].Widgets[0]
		Expect(remoteWidget.LayoutColumns).To(HaveValue(Equal(int32(12))))
		Expect(remoteWidget.Definition.Gauge.ArcDisplay).ToNot(BeNil())
		Expect(remoteWidget.Definition.Gauge.ArcDisplay.ValueArc).To(HaveValue(BeTrue()))
		Expect(remoteWidget.Definition.Gauge.ShowMinMax).To(HaveValue(BeTrue()))
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the Dashboard")
		modifiedDashboard := dashboard.DeepCopy()
		modifiedDashboard.Spec.Json = ptr.To(getDashboardJson("Test Updated Dashboard"))
		Expect(crClient.Patch(ctx, modifiedDashboard, client.MergeFrom(dashboard))).To(Succeed())

		By("Verifying Dashboard is updated in Coralogix backend")
		Eventually(func() string {
			getResponse, httpResp, err := dashboardsClient.DashboardsServiceGetDashboard(ctx, dashboardID).Execute()
			Expect(cxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
			return getResponse.Dashboard.Name
		}, time.Minute, time.Second).Should(Equal("Test Updated Dashboard"))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Dashboard")
		Expect(crClient.Delete(ctx, dashboard)).To(Succeed())

		By("Verifying Dashboard is deleted from Coralogix backend")
		Eventually(func() int {
			_, httpResp, err := dashboardsClient.DashboardsServiceGetDashboard(ctx, dashboardID).Execute()
			return cxsdk.Code(cxsdk.NewAPIError(httpResp, err))
		}, time.Minute, time.Second).Should(Equal(http.StatusNotFound))
	})
})

var _ = Describe("Dashboard import", Ordered, func() {
	var (
		crClient         client.Client
		dashboardsClient *dashboards.DashboardServiceAPIService
		dashboard        *coralogixv1alpha1.Dashboard
		dashboardName    = "dashboard-import-sample"
		dashboardID      string
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		dashboardsClient = newDashboardOpenAPIClientSet().Dashboards()
	})

	It("Should adopt a pre-existing remote dashboard", func(ctx context.Context) {
		By("Creating a remote dashboard directly, outside of the operator")
		remoteDashboard := new(dashboards.Dashboard)
		// A controlled camelCase fixture, so plain json.Unmarshal is enough here; the
		// compatibility handling for customer-authored JSON is covered by unit tests.
		Expect(json.Unmarshal([]byte(getDashboardJson("Pre-existing Dashboard")), remoteDashboard)).To(Succeed())
		createResponse, httpResp, err := dashboardsClient.
			DashboardsServiceCreateDashboard(ctx).
			CreateDashboardRequestDataStructure(dashboards.CreateDashboardRequestDataStructure{
				Dashboard: *remoteDashboard,
				RequestId: "cx-operator-e2e-import-seed",
			}).
			Execute()
		Expect(cxsdk.NewAPIError(httpResp, err)).ToNot(HaveOccurred())
		Expect(createResponse.DashboardId).ToNot(BeNil())
		dashboardID = *createResponse.DashboardId

		By("Creating a Dashboard CR with the import annotation")
		dashboardJson := getDashboardJson("Pre-existing Dashboard")
		dashboard = &coralogixv1alpha1.Dashboard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dashboardName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					coralogixv1alpha1.ImportDashboardIDAnnotationKey: dashboardID,
				},
			},
			Spec: coralogixv1alpha1.DashboardSpec{
				Json: &dashboardJson,
			},
		}
		Expect(crClient.Create(ctx, dashboard)).To(Succeed())

		By("Verifying the CR adopts the existing remote dashboard ID")
		fetchedDashboard := &coralogixv1alpha1.Dashboard{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: dashboardName, Namespace: testNamespace}, fetchedDashboard)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedDashboard.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			if fetchedDashboard.Status.ID != nil {
				return nil
			}
			return fmt.Errorf("Dashboard ID is not set")
		}, time.Minute, time.Second).Should(Succeed())
		Expect(*fetchedDashboard.Status.ID).To(Equal(dashboardID))
		Expect(fetchedDashboard.Status.Imported).To(BeTrue())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Dashboard")
		Expect(crClient.Delete(ctx, dashboard)).To(Succeed())

		By("Verifying Dashboard is deleted from Coralogix backend")
		Eventually(func() int {
			_, httpResp, err := dashboardsClient.DashboardsServiceGetDashboard(ctx, dashboardID).Execute()
			return cxsdk.Code(cxsdk.NewAPIError(httpResp, err))
		}, time.Minute, time.Second).Should(Equal(http.StatusNotFound))
	})
})

func newDashboardOpenAPIClientSet() *cxsdk.ClientSet {
	builder := cxsdk.NewConfigBuilder().WithAPIKeyEnv()
	if domain := os.Getenv("CORALOGIX_DOMAIN"); domain != "" {
		builder = builder.WithDomain(domain)
	} else {
		builder = builder.WithRegionEnv()
	}
	return cxsdk.NewClientSet(builder.Build())
}

// getDashboardJson carries arcDisplay, showMinMax and layoutColumns deliberately: those
// three fields are missing from the proto the gRPC client used, so they double as the
// round-trip assertion that this migration delivers them to the API.
// The deprecated showInnerArc/showOuterArc are set because the backend rejects a gauge
// that omits them, even when arcDisplay supersedes them.
func getDashboardJson(name string) string {
	return fmt.Sprintf(`{
  "name": "%s",
  "layout": {
    "sections": [
      {
        "id": {
          "value": "ab19801c-bdbb-428e-999d-703c1d7a5ff4"
        },
        "rows": [
          {
            "id": {
              "value": "ab19801c-bdbb-428e-999d-703c1d7a5ff5"
            },
            "appearance": {
              "height": 16
            },
            "widgets": [
              {
                "id": {
                  "value": "ab19801c-bdbb-428e-999d-703c1d7a5ff6"
                },
                "title": "Gauge with arcs",
                "definition": {
                  "gauge": {
                    "query": {
                      "metrics": {
                        "promqlQuery": {
                          "value": "vector(2)"
                        },
                        "aggregation": "AGGREGATION_UNSPECIFIED",
                        "filters": [],
                        "editorMode": "METRICS_QUERY_EDITOR_MODE_TEXT",
                        "promqlQueryType": "PROM_QL_QUERY_TYPE_INSTANT"
                      }
                    },
                    "min": 0,
                    "max": 100,
                    "showInnerArc": true,
                    "showOuterArc": true,
                    "unit": "UNIT_NUMBER",
                    "thresholds": [],
                    "thresholdBy": "THRESHOLD_BY_UNSPECIFIED",
                    "thresholdType": "THRESHOLD_TYPE_RELATIVE",
                    "arcDisplay": {
                      "valueArc": true,
                      "thresholdArc": true
                    },
                    "showMinMax": true
                  }
                },
                "layoutColumns": 12
              }
            ]
          }
        ],
        "options": {
          "custom": {
            "name": "New Section",
            "collapsed": false,
            "color": {
              "predefined": "SECTION_PREDEFINED_COLOR_UNSPECIFIED"
            }
          }
        }
      }
    ]
  },
  "variables": [],
  "filters": [],
  "relativeTimeFrame": "900s",
  "annotations": [],
  "off": {}
}`, name)
}
