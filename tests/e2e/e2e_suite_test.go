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
	"strconv"
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

// uniqueName returns a DNS-1123 name that is unique across parallel CI jobs that
// share one Coralogix account. Second-granularity Unix timestamps collide when
// upgrade-test matrix jobs and other e2e workflows start together; a reused
// backend name lets one job delete or fail to create the other's resource.
func uniqueName(prefix string) string {
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	name := prefix + "-" + suffix
	const maxDNS1123 = 63
	if len(name) <= maxDNS1123 {
		return name
	}
	keep := maxDNS1123 - 1 - len(suffix)
	if keep < 1 {
		return suffix[len(suffix)-maxDNS1123:]
	}
	return prefix[:keep] + "-" + suffix
}

// accountWideBackendTimeout is for assertions on Coralogix objects that belong to the whole
// account or tenant rather than to this cluster - company IP access settings, archive targets and
// the like. Several e2e jobs run this suite against one account at a time, so another job can
// overwrite or clear such an object while a spec is asserting on it. The owning operator pushes
// its desired state back on its next reconcile (the specs that need this have a
// <KIND>_RECONCILE_INTERVAL_SECONDS set in CI), so allow enough time for that to land.
const accountWideBackendTimeout = 3 * time.Minute

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coralogix operator E2E test suite")
}

// initClients has to run on every Ginkgo process, since each one is a separate OS process
// with its own copy of ClientsInstance.
func initClients() {
	region := strings.ToLower(os.Getenv("CORALOGIX_REGION"))
	apiKey := os.Getenv("CORALOGIX_API_KEY")

	By("Initializing clients")
	ClientsInstance.InitCoralogixClientSet(region, apiKey, apiKey)
	Expect(ClientsInstance.InitControllerRuntimeClient()).To(Succeed())
	Expect(ClientsInstance.InitK8sClient()).To(Succeed())
}

var _ = SynchronizedBeforeSuite(func(ctx context.Context) []byte {
	initClients()
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

	return nil
}, func(_ context.Context, _ []byte) {
	initClients()
})

var _ = SynchronizedAfterSuite(func() {}, func(ctx context.Context) {
	By("Deleting test namespace")
	k8sClient := ClientsInstance.GetK8sClient()
	Expect(k8sClient.CoreV1().Namespaces().Delete(ctx, testNamespace, metav1.DeleteOptions{})).To(Succeed())

	// The namespace only disappears once the operator has removed the finalizer from every
	// resource in it, which means the remote Coralogix resources are already cleaned up by
	// the time this returns.
	Eventually(func() bool {
		_, err := k8sClient.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, 3*time.Minute, time.Second).Should(BeTrue())
})
