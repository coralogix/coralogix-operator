/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

const testNamespace = "coralogix-e2e-test"

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coralogix operator E2E test suite")
}

var _ = BeforeSuite(func(ctx context.Context) {
	region := strings.ToLower(os.Getenv("CORALOGIX_REGION"))
	apiKey := os.Getenv("CORALOGIX_API_KEY")

	By("Initializing clients")
	ClientsInstance.InitCoralogixClientSet(cxsdk.CoralogixGrpcEndpointFromRegion(region), apiKey, apiKey)
	Expect(ClientsInstance.InitControllerRuntimeClient()).To(Succeed())
	Expect(ClientsInstance.InitK8sClient()).To(Succeed())

	k8sClient := ClientsInstance.GetK8sClient()

	By("Creating test namespace")
	_, err := k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	By("Validating that the controller-manager pod is running")
	Eventually(func() corev1.PodPhase {
		podList, err := k8sClient.CoreV1().
			Pods("coralogix-operator-system").
			List(ctx, metav1.ListOptions{LabelSelector: "control-plane=controller-manager"})
		Expect(err).NotTo(HaveOccurred())
		return podList.Items[0].Status.Phase
	}, time.Minute, time.Second).Should(Equal(corev1.PodRunning))
})

var _ = AfterSuite(func(ctx context.Context) {
	By("Deleting test namespace")
	k8sClient := ClientsInstance.GetK8sClient()
	Expect(k8sClient.CoreV1().Namespaces().Delete(ctx, testNamespace, metav1.DeleteOptions{})).To(Succeed())
})
