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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ClientsInstance.InitCoralogixClientSet(region, apiKey, apiKey)
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

	By("Validating that the operator deployment is available")
	Eventually(func() bool {
		depList, err := k8sClient.AppsV1().
			Deployments("coralogix-operator-system").
			List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		dep := depList.Items[0]
		for _, condition := range dep.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Minute, time.Second).Should(BeTrue())
})

var _ = AfterSuite(func(ctx context.Context) {
	By("Deleting test namespace")
	k8sClient := ClientsInstance.GetK8sClient()
	Expect(k8sClient.CoreV1().Namespaces().Delete(ctx, testNamespace, metav1.DeleteOptions{})).To(Succeed())
	Eventually(func() bool {
		_, err := k8sClient.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, time.Minute, time.Second).Should(BeTrue())

	By("Giving the operator some time to clean up")
	time.Sleep(30 * time.Second)
})
