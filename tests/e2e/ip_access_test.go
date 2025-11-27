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
	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"
	ipaccess "github.com/coralogix/coralogix-management-sdk/go/openapi/gen/ip_access_service"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("IPAccess", func() {
	var (
		crClient       client.Client
		ipAccessClient *ipaccess.IPAccessServiceAPIService
		ipAccessCR     *coralogixv1alpha1.IPAccess
		ipAccessID     string
		crName         = "company-ip-access"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()

		cfg := cxsdk.NewConfigBuilder().WithAPIKeyEnv().WithRegionEnv().Build()
		ipAccessClient = cxsdk.NewIPAccessClient(cfg)

		ipAccessCR = &coralogixv1alpha1.IPAccess{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.IPAccessSpec{
				EnableCoralogixCustomerSupportAccess: "disabled",
				IPAccess: []coralogixv1alpha1.IPAccessRule{
					{
						Name:    ptr.To("Office Network"),
						IPRange: "31.154.215.114/32",
						Enabled: ptr.To(false),
					},
					{
						Name:    ptr.To("VPN"),
						IPRange: "198.51.100.0/24",
						Enabled: ptr.To(false),
					},
				},
			},
		}
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Deleting existing IPAccess settings if any")
		_, _, err := ipAccessClient.IpAccessServiceDeleteCompanyIpAccessSettings(ctx).Execute()
		Expect(err).ToNot(HaveOccurred())

		By("Creating IPAccess")
		Expect(crClient.Create(ctx, ipAccessCR)).To(Succeed())

		By("Fetching the IPAccess ID")
		Eventually(func(g Gomega) {
			fetched := &coralogixv1alpha1.IPAccess{}
			g.Expect(crClient.Get(ctx, client.ObjectKey{Name: crName, Namespace: testNamespace}, fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetched.Status.ID != nil {
				ipAccessID = *fetched.Status.ID
			}
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying IpAccess exists in Coralogix backend")
		Eventually(func(g Gomega) {
			getRes, _, err := ipAccessClient.IpAccessServiceGetCompanyIpAccessSettings(ctx).Id(ipAccessID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getRes.Settings).ToNot(BeNil())
			g.Expect(getRes.Settings.EnableCoralogixCustomerSupportAccess).ToNot(BeNil())
			g.Expect(*getRes.Settings.EnableCoralogixCustomerSupportAccess).To(Equal(ipaccess.CORALOGIXCUSTOMERSUPPORTACCESS_CORALOGIX_CUSTOMER_SUPPORT_ACCESS_DISABLED))

			g.Expect(getRes.Settings.IpAccess).ToNot(BeNil())
			found := false
			for _, rule := range *getRes.Settings.IpAccess {
				if rule.Name != nil && *rule.Name == "VPN" {
					found = true
					g.Expect(*rule.IpRange).To(Equal("198.51.100.0/24"))
				}
			}
			g.Expect(found).To(BeTrue(), "VPN rule should be present in backend")
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the IPAccess")
		newName := "Office Network Updated"
		modified := ipAccessCR.DeepCopy()
		modified.Spec.IPAccess[0].Name = ptr.To(newName)
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(ipAccessCR))).To(Succeed())

		By("Verifying IPAccess is updated in Coralogix backend")
		Eventually(func(g Gomega) {
			getRes, _, err := ipAccessClient.IpAccessServiceGetCompanyIpAccessSettings(ctx).Id(ipAccessID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(getRes.Settings).ToNot(BeNil())
			g.Expect(getRes.Settings.IpAccess).ToNot(BeNil())

			// Ensure updated entry exists
			found := false
			for _, rule := range *getRes.Settings.IpAccess {
				if rule.Name != nil && *rule.Name == newName {
					found = true
				}
			}
			g.Expect(found).To(BeTrue(), "Updated Office Network rule should be present in backend")
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting IPAccess CR")
		Expect(crClient.Delete(ctx, ipAccessCR)).To(Succeed())

		By("Verifying IPAccess settings were deleted in backend")
		Eventually(func(g Gomega) {
			res, _, err := ipAccessClient.IpAccessServiceGetCompanyIpAccessSettings(ctx).Id(ipAccessID).Execute()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(res.Settings.IpAccess).ToNot(BeNil())
			g.Expect(*res.Settings.IpAccess).To(BeEmpty())
		}, time.Minute, time.Second).Should(Succeed())
	})
})
