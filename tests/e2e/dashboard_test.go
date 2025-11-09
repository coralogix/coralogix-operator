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

var _ = PDescribe("Dashboard", Ordered, func() {
	var (
		crClient         client.Client
		dashboardsClient *cxsdk.DashboardsClient
		dashboard        *coralogixv1alpha1.Dashboard
		dashboardName    = "dashboard-sample"
		dashboardID      string
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		dashboardsClient = ClientsInstance.GetCoralogixClientSet().Dashboards()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Dashboard")
		dashboard = getSampleDashboard(testDashboardJson)
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
		Eventually(func() error {
			_, err := dashboardsClient.Get(ctx, &cxsdk.GetDashboardRequest{DashboardId: wrapperspb.String(dashboardID)})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the Dashboard")
		modifiedDashboard := dashboard.DeepCopy()
		modifiedDashboard.Spec.Json = ptr.To(testUpdatedDashboardJson)
		Expect(crClient.Patch(ctx, modifiedDashboard, client.MergeFrom(dashboard))).To(Succeed())

		By("Verifying Dashboard is updated in Coralogix backend")
		Eventually(func() string {
			getDashboardRes, err := dashboardsClient.Get(ctx, &cxsdk.GetDashboardRequest{DashboardId: wrapperspb.String(dashboardID)})
			Expect(err).ToNot(HaveOccurred())
			return getDashboardRes.Dashboard.Name.GetValue()
		}, time.Minute, time.Second).Should(Equal("Test Updated Dashboard"))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Dashboard")
		Expect(crClient.Delete(ctx, dashboard)).To(Succeed())

		By("Verifying Dashboard is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := dashboardsClient.Get(ctx, &cxsdk.GetDashboardRequest{DashboardId: wrapperspb.String(dashboardID)})
			return cxsdk.Code(err)
		}).Should(Equal(codes.NotFound))
	})
})

func getSampleDashboard(json string) *coralogixv1alpha1.Dashboard {
	return &coralogixv1alpha1.Dashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dashboard-sample",
			Namespace: testNamespace,
		},
		Spec: coralogixv1alpha1.DashboardSpec{
			Json: &json,
		},
	}
}

const testDashboardJson = `{
  "id": "ExtweBMerfqxXCn3d84Xp",
  "name": "Test Dashboard",
  "layout": {
    "sections": [
      {
        "id": {
          "value": "ab19801c-bdbb-428e-999d-703c1d7a5ff4"
        },
        "rows": [],
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
}`

const testUpdatedDashboardJson = `{
  "id": "ExtweBMerfqxXCn3d84Xp",
  "name": "Test Updated Dashboard",
  "layout": {
    "sections": [
      {
        "id": {
          "value": "ab19801c-bdbb-428e-999d-703c1d7a5ff4"
        },
        "rows": [],
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
}`
