// Copyright 2026 Coralogix Ltd.
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/v2/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/v2/internal/utils"
)

var _ = Describe("ConfigurationGroup", Ordered, func() {
	var (
		crClient             client.Client
		configurationGroup   *coralogixv1alpha1.ConfigurationGroup
		configurationGroupID string
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating ConfigurationGroup")
		name := uniqueName("configuration-group")
		configurationGroup = &coralogixv1alpha1.ConfigurationGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: coralogixv1alpha1.ConfigurationGroupSpec{
				Name:        name,
				Description: ptr.To("e2e configuration group"),
				Tags:        []string{"e2e"},
				Family: coralogixv1alpha1.ConfigurationFamilySpec{
					Active:           ptr.To(true),
					CollectorVersion: ptr.To("0.114.0"),
					RemoteConfigurations: []coralogixv1alpha1.RemoteConfigurationSpec{
						{
							Name: "default",
							RawConfiguration: `receivers:
  otlp:
    protocols:
      grpc: {}
`,
							AgentSelector: map[string]string{
								"cx.agent.type": "agent",
							},
						},
					},
				},
			},
		}
		Expect(crClient.Create(ctx, configurationGroup)).To(Succeed())

		By("Fetching the ConfigurationGroup ID")
		fetched := &coralogixv1alpha1.ConfigurationGroup{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx,
				types.NamespacedName{Name: name, Namespace: testNamespace},
				fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if fetched.Status.ID != nil {
				configurationGroupID = *fetched.Status.ID
				return nil
			}
			return fmt.Errorf("ConfigurationGroup ID is not set")
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the ConfigurationGroup")
		updatedDescription := uniqueName("updated")
		modified := configurationGroup.DeepCopy()
		modified.Spec.Description = ptr.To(updatedDescription)
		Expect(crClient.Patch(ctx, modified, client.MergeFrom(configurationGroup))).To(Succeed())

		By("Verifying ConfigurationGroup stays synced")
		fetched := &coralogixv1alpha1.ConfigurationGroup{}
		Eventually(func(g Gomega) {
			g.Expect(crClient.Get(ctx,
				types.NamespacedName{Name: configurationGroup.Name, Namespace: testNamespace},
				fetched)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetched.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(fetched.Status.ID).ToNot(BeNil())
			g.Expect(*fetched.Status.ID).To(Equal(configurationGroupID))
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the ConfigurationGroup")
		Expect(crClient.Delete(ctx, configurationGroup)).To(Succeed())

		By("Verifying ConfigurationGroup is removed from the cluster")
		Eventually(func() error {
			return crClient.Get(ctx,
				types.NamespacedName{Name: configurationGroup.Name, Namespace: testNamespace},
				&coralogixv1alpha1.ConfigurationGroup{})
		}, time.Minute, time.Second).ShouldNot(Succeed())
	})
})
