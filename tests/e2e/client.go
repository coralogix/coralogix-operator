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
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	prometheusv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
)

var ClientsInstance = &Clients{}

type Clients struct {
	CxClientSet *cxsdk.ClientSet
	CrClient    client.Client
	K8sClient   *kubernetes.Clientset
}

func (c *Clients) InitCoralogixClientSet(targetURL, teamsLevelAPIKey string, userLevelAPIKey string) {
	if c.CxClientSet == nil {
		c.CxClientSet = cxsdk.NewClientSet(targetURL, teamsLevelAPIKey, userLevelAPIKey)
	}
}

func (c *Clients) InitControllerRuntimeClient() error {
	if c.CrClient == nil {
		crClient, err := client.New(config.GetConfigOrDie(), client.Options{})
		if err != nil {
			return err
		}
		if err = prometheusv1.AddToScheme(crClient.Scheme()); err != nil {
			return err
		}
		if err = prometheusv1alpha1.AddToScheme(crClient.Scheme()); err != nil {
			return err
		}
		if err = coralogixv1alpha1.AddToScheme(crClient.Scheme()); err != nil {
			return err
		}
		c.CrClient = crClient
	}
	return nil
}

func (c *Clients) InitK8sClient() error {
	if c.K8sClient == nil {
		k8sClient, err := kubernetes.NewForConfig(config.GetConfigOrDie())
		if err != nil {
			return err
		}
		c.K8sClient = k8sClient
	}
	return nil
}

func (c *Clients) GetCoralogixClientSet() *cxsdk.ClientSet {
	return c.CxClientSet
}

func (c *Clients) GetControllerRuntimeClient() client.Client {
	return c.CrClient
}

func (c *Clients) GetK8sClient() *kubernetes.Clientset {
	return c.K8sClient
}
