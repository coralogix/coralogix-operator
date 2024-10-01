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

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coralogix operator E2E test suite")
}

func init() {
	err := ClientsInstance.InitK8sClient()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize k8s client: %v", err))
	}

	err = ClientsInstance.InitControllerRuntimeClient()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize controller runtime client: %v", err))
	}

	region := os.Getenv("CORALOGIX_REGION")
	apiKey := os.Getenv("CORALOGIX_API_KEY")
	ClientsInstance.InitCoralogixClientSet(cxsdk.CoralogixGrpcEndpointFromRegion(region), apiKey, apiKey)
}

var _ = BeforeSuite(func() {
	const operatorImage = "tests.com/coralogix-operator:v0.0.1"

	By("Building the controller-manager image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", operatorImage))
	_, err := Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	By("Loading the controller-manager image on Kind")
	err = LoadImageToKindClusterWithName(operatorImage)
	Expect(err).NotTo(HaveOccurred())

	By("Installing CRDs")
	cmd = exec.Command("make", "install")
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	By("Deploying the controller-manager")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", operatorImage))
	_, err = Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	By("Validating that the controller-manager pod is running")
	Eventually(func() error {
		cmd = exec.Command("kubectl", "get",
			"pods", "-l", "app.kubernetes.io/name=coralogix-operator-controller-manager",
			"-o", "jsonpath={.items[*].status}", "-n", "coralogix-operator-system",
		)
		status, err := Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		if !strings.Contains(string(status), "\"phase\":\"Running\"") {
			return fmt.Errorf("coralogix operator pod in %s status", status)
		}
		return nil
	}, time.Minute, time.Second).Should(Succeed())
})
