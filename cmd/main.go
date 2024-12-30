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

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	prometheusv1alpha "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	utils "github.com/coralogix/coralogix-operator/api"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	controllers "github.com/coralogix/coralogix-operator/internal/controller"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
	coralogixcontrollers "github.com/coralogix/coralogix-operator/internal/controller/coralogix"
	"github.com/coralogix/coralogix-operator/internal/monitoring"
	webhookcoralogixv1alpha1 "github.com/coralogix/coralogix-operator/internal/webhook/coralogix/v1alpha1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme          = runtime.NewScheme()
	setupLog        = ctrl.Log.WithName("setup")
	regionToGrpcUrl = map[string]string{
		"APAC1":   "ng-api-grpc.app.coralogix.in:443",
		"AP1":     "ng-api-grpc.app.coralogix.in:443",
		"APAC2":   "ng-api-grpc.coralogixsg.com:443",
		"AP2":     "ng-api-grpc.coralogixsg.com:443",
		"EUROPE1": "ng-api-grpc.coralogix.com:443",
		"EU1":     "ng-api-grpc.coralogix.com:443",
		"EUROPE2": "ng-api-grpc.eu2.coralogix.com:443",
		"EU2":     "ng-api-grpc.eu2.coralogix.com:443",
		"USA1":    "ng-api-grpc.coralogix.us:443",
		"US1":     "ng-api-grpc.coralogix.us:443",
		"USA2":    "ng-api-grpc.cx498.coralogix.com:443",
		"US2":     "ng-api-grpc.cx498.coralogix.com:443",
	}
	validRegions = utils.GetKeys(regionToGrpcUrl)
)

func init() {
	utilruntime.Must(prometheus.AddToScheme(scheme))

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))

	utilruntime.Must(prometheusv1alpha.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	region := os.Getenv("CORALOGIX_REGION")
	flag.StringVar(&region, "region", region, fmt.Sprintf("The region of your Coralogix cluster. Can be one of %q. Conflicts with 'domain'.", validRegions))

	domain := os.Getenv("CORALOGIX_DOMAIN")
	flag.StringVar(&domain, "domain", domain, "The domain of your Coralogix cluster. Conflicts with 'region'.")

	apiKey := os.Getenv("CORALOGIX_API_KEY")
	flag.StringVar(&apiKey, "api-key", apiKey, "The proper api-key based on your Coralogix cluster's region.")

	enableWebhooks := os.Getenv("ENABLE_WEBHOOKS")
	flag.StringVar(&enableWebhooks, "enable-webhooks", enableWebhooks, "Enable webhooks for the operator. Default is false.")

	var prometheusRuleController bool
	flag.BoolVar(&prometheusRuleController, "prometheus-rule-controller", true, "Determine if the prometheus rule controller should be started. Default is true.")

	var recordingRuleGroupSetSuffix string
	flag.StringVar(&recordingRuleGroupSetSuffix, "recording-rule-group-set-suffix", "", "Suffix to be added to the RecordingRuleGroupSet")

	var webhookCertDir string
	flag.StringVar(&webhookCertDir, "webhook-cert-dir", "/tmp/k8s-webhook-server/serving-certs", "Directory containing the webhook certs")

	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if region != "" && domain != "" {
		err := fmt.Errorf("region and domain flags are mutually exclusive")
		setupLog.Error(err, "invalid arguments for running operator")
		os.Exit(1)
	}

	if region == "" && domain == "" {
		err := fmt.Errorf("region or domain must be set")
		setupLog.Error(err, "invalid arguments for running operator")
		os.Exit(1)
	}

	var targetUrl string
	if region != "" {
		if !slices.Contains(validRegions, region) {
			err := fmt.Errorf("region value is '%s', but can be one of %q", region, validRegions)
			setupLog.Error(err, "invalid arguments for running operator")
			os.Exit(1)
		}
		targetUrl = regionToGrpcUrl[region]
	} else if domain != "" {
		targetUrl = fmt.Sprintf("ng-api-grpc.%s:443", domain)
	}

	if apiKey == "" {
		err := fmt.Errorf("api-key can not be empty")
		setupLog.Error(err, "invalid arguments for running operator")
		os.Exit(1)
	}

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgrOpts := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9e1892e3.coralogix",
		PprofBindAddress:       "0.0.0.0:8888",
	}

	// Check if webhooks are enabled before setting up the webhook server
	if enableWebhooks == "true" {
		mgrOpts.WebhookServer = webhook.NewServer(webhook.Options{
			TLSOpts: tlsOpts,
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	sdkClientSet := cxsdk.NewClientSet(targetUrl, apiKey, apiKey)

	if err = (&coralogixcontrollers.RuleGroupReconciler{
		CoralogixClientSet: clientset.NewClientSet(targetUrl, apiKey),
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RuleGroup")
		os.Exit(1)
	}
	if err = (&coralogixcontrollers.AlertReconciler{
		CoralogixClientSet: clientset.NewClientSet(targetUrl, apiKey),
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Alert")
		os.Exit(1)
	}
	if prometheusRuleController {
		if err = (&controllers.PrometheusRuleReconciler{
			CoralogixClientSet: clientset.NewClientSet(targetUrl, apiKey),
			Client:             mgr.GetClient(),
			Scheme:             mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "RecordingRuleGroup")
			os.Exit(1)
		}
	}
	if err = (&coralogixcontrollers.RecordingRuleGroupSetReconciler{
		CoralogixClientSet:          clientset.NewClientSet(targetUrl, apiKey),
		Client:                      mgr.GetClient(),
		Scheme:                      mgr.GetScheme(),
		RecordingRuleGroupSetSuffix: recordingRuleGroupSetSuffix,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RecordingRuleGroupSet")
		os.Exit(1)
	}

	if err = (&coralogixcontrollers.OutboundWebhookReconciler{
		OutboundWebhooksClient: clientset.NewClientSet(targetUrl, apiKey).OutboundWebhooks(),
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OutboundWebhook")
		os.Exit(1)
	}
	if err = (&coralogixcontrollers.ApiKeyReconciler{
		ApiKeysClient: clientset.NewClientSet(targetUrl, apiKey).ApiKeys(),
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApiKey")
		os.Exit(1)
	}
	if err = (&coralogixcontrollers.CustomRoleReconciler{
		CustomRolesClient: sdkClientSet.Roles(),
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CustomRole")
		os.Exit(1)
	}

	if prometheusRuleController {
		if err = (&controllers.AlertmanagerConfigReconciler{
			CoralogixClientSet: clientset.NewClientSet(targetUrl, apiKey),
			Client:             mgr.GetClient(),
			Scheme:             mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "RecordingRuleGroup")
			os.Exit(1)
		}
	}

	if enableWebhooks == "true" {
		if err = webhookcoralogixv1alpha1.SetupOutboundWebhookWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OutboundWebhook")
			os.Exit(1)
		}

		if err = webhookcoralogixv1alpha1.SetupRuleGroupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "RuleGroup")
			os.Exit(1)
		}

		if err = webhookcoralogixv1alpha1.SetupApiKeyWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ApiKey")
			os.Exit(1)
		}
	} else {
		setupLog.Info("Webhooks are disabled")
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := monitoring.RegisterMetrics(); err != nil {
		setupLog.Error(err, "unable to register metrics")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
