// Copyright 2025 Coralogix Ltd.
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var _ = Describe("Extension", Ordered, func() {
	var (
		crClient         client.Client
		extensionsClient *cxsdk.ExtensionsClient
		extension        *coralogixv1alpha1.Extension
		extensionID      string

		extensionName = "k8sobservability"
	)

	BeforeEach(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		extensionsClient = ClientsInstance.GetCoralogixClientSet().Extensions()
		extension = &coralogixv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      extensionName,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.ExtensionSpec{
				Id:      "K8sObservability",
				Version: "0.0.2",
				ItemIds: []string{
					"013eb19e-14c0-4994-82c2-6e1603ca865c",
					"abd816a3-384d-4e19-8d9e-79395f21ece5",
					"8748b765-c973-4a75-b5ad-361135c43e02",
					"29c015f6-e60a-4edb-92dd-5267cfe6cc1c",
				},
			},
		}
	})

	It("Should be deployed successfully", func(ctx context.Context) {

		By("Deploying the Extension")
		Expect(crClient.Create(ctx, extension)).To(Succeed())

		By("Fetching the Extension ID")
		fetchedExtension := &coralogixv1alpha1.Extension{}
		Eventually(func(g Gomega) error {
			ok := g.Expect(crClient.Get(ctx, types.NamespacedName{Name: extensionName, Namespace: testNamespace}, fetchedExtension)).To(Succeed())
			if !ok {
				return fmt.Errorf("error fetching extension")
			}
			for _, condition := range fetchedExtension.Status.Conditions {
				if condition.Type == "Failed" && condition.Status == "True" {
					return fmt.Errorf("extension deployment failed: %s", condition.Message)
				}
			}
			if fetchedExtension.Status.ID != nil {
				extensionID = *fetchedExtension.Status.ID
				return nil
			}
			return fmt.Errorf("extension ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Extension is deployed in Coralogix backend")
		Eventually(func() error {
			deployedExtensions, err := extensionsClient.GetDeployed(ctx, &cxsdk.GetDeployedExtensionsRequest{})
			if err != nil {
				return fmt.Errorf("error fetching deployed extensions: %w", err)
			}
			for _, deployedExtension := range deployedExtensions.DeployedExtensions {
				if deployedExtension.Id.Value == extensionID {
					return nil
				}
			}
			return fmt.Errorf("extension with ID %s not found in Coralogix backend", extensionID)
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be undeployed successfully", func(ctx context.Context) {
		By("Undeploying the Extension")
		Expect(crClient.Delete(ctx, extension)).To(Succeed())

		By("Verifying Extension is undeployed in Coralogix backend")
		Eventually(func() error {
			deployedExtensions, err := extensionsClient.GetDeployed(ctx, &cxsdk.GetDeployedExtensionsRequest{})
			if err != nil {
				return fmt.Errorf("error fetching deployed extensions: %w", err)
			}
			for _, deployedExtension := range deployedExtensions.DeployedExtensions {
				if deployedExtension.Id.Value == extensionID {
					return fmt.Errorf("extension with ID %s still exists in Coralogix backend", extensionID)
				}
			}
			return nil
		}, time.Minute, time.Second).Should(Succeed(), "Extension should not be found in Coralogix backend after undeployment")
	})

})
