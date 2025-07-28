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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("Preset", Ordered, func() {
	var (
		crClient            client.Client
		notificationsClient *cxsdk.NotificationsClient
		presetID            string
		preset              *coralogixv1alpha1.Preset
	)

	BeforeAll(func() {
		crClient = ClientsInstance.GetControllerRuntimeClient()
		notificationsClient = ClientsInstance.GetCoralogixClientSet().Notifications()
	})

	It("Should be created successfully", func(ctx context.Context) {
		By("Creating Preset")
		presetName := fmt.Sprintf("slack-preset-%d", time.Now().Unix())
		preset = getSampleSlackPreset(presetName, testNamespace)
		Expect(crClient.Create(ctx, preset)).To(Succeed())

		By("Fetching the Preset ID")
		fetchedPreset := &coralogixv1alpha1.Preset{}
		Eventually(func(g Gomega) error {
			g.Expect(crClient.Get(ctx, types.NamespacedName{Name: presetName, Namespace: testNamespace}, fetchedPreset)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(fetchedPreset.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			if fetchedPreset.Status.Id != nil {
				presetID = *fetchedPreset.Status.Id
				return nil
			}
			return fmt.Errorf("preset ID is not set")
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying Preset exists in Coralogix backend")
		Eventually(func() error {
			_, err := notificationsClient.GetPreset(ctx, &cxsdk.GetPresetRequest{Id: presetID})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should be updated successfully", func(ctx context.Context) {
		By("Patching the Preset")
		newPresetName := "slack-preset-updated"
		modifiedPreset := preset.DeepCopy()
		modifiedPreset.Spec.Name = newPresetName
		Expect(crClient.Patch(ctx, modifiedPreset, client.MergeFrom(preset))).To(Succeed())

		By("Verifying Preset is updated in Coralogix backend")
		Eventually(func() string {
			getPresetRes, err := notificationsClient.GetPreset(ctx, &cxsdk.GetPresetRequest{Id: presetID})
			Expect(err).ToNot(HaveOccurred())
			return getPresetRes.GetPreset().GetName()
		}, time.Minute, time.Second).Should(Equal(newPresetName))
	})

	It("Should be deleted successfully", func(ctx context.Context) {
		By("Deleting the Preset")
		Expect(crClient.Delete(ctx, preset)).To(Succeed())

		By("Verifying Preset is deleted from Coralogix backend")
		Eventually(func() codes.Code {
			_, err := notificationsClient.GetPreset(ctx, &cxsdk.GetPresetRequest{Id: presetID})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})
})

func getSampleSlackPreset(name, namespace string) *coralogixv1alpha1.Preset {
	parentID := "preset_system_slack_alerts_basic"

	return &coralogixv1alpha1.Preset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.PresetSpec{
			Name:          name,
			Description:   "This is a sample slack preset",
			ConnectorType: "slack",
			EntityType:    "alerts",
			ParentId:      &parentID,
			ConfigOverrides: []coralogixv1alpha1.ConfigOverride{
				{
					ConditionType: coralogixv1alpha1.ConditionType{
						MatchEntityTypeAndSubType: &coralogixv1alpha1.MatchEntityTypeAndSubType{
							EntitySubType: "logsImmediateTriggered",
						},
					},
					MessageConfig: coralogixv1alpha1.MessageConfig{
						Fields: []coralogixv1alpha1.MessageConfigField{
							{
								FieldName: "title",
								Template:  "CUSTOM PRESET OVERRIDE: {{alert.status}} {{alertDef.priority}} - {{alertDef.name}}",
							},
							{
								FieldName: "description",
								Template:  "{{alertDef.description}}",
							},
						},
					},
				},
			},
		},
	}
}
