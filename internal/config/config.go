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

package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	openapicxsdk "github.com/coralogix/coralogix-management-sdk/go/openapi/cxsdk"

	"github.com/coralogix/coralogix-operator/internal/utils"
)

var (
	cfg          = &Config{}
	CrClient     client.Client
	scheme       *runtime.Scheme
	once         sync.Once
	validRegions = []string{"AP1", "AP2", "AP3", "EU1", "EU2", "US1", "US2"}
)

type Config struct {
	CoralogixApiKey             string
	CoralogixGrpcUrl            string
	CoralogixOpenApiUrl         string
	Selector                    Selector
	ReconcileIntervals          map[string]time.Duration
	PrometheusRuleController    bool
	RecordingRuleGroupSetSuffix string
	MetricsAddr                 string
	ProbeAddr                   string
	EnableLeaderElection        bool
	LeaderElectionID            string
	SecureMetrics               bool
	EnableHTTP2                 bool
}

func InitConfig(setupLog logr.Logger) *Config {
	once.Do(func() {
		flag.StringVar(&cfg.MetricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
		flag.StringVar(&cfg.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
		flag.BoolVar(&cfg.EnableLeaderElection, "leader-elect", true,
			"Enable leader election for controller manager. "+
				"Enabling this will ensure there is only one active controller manager.")
		flag.StringVar(&cfg.LeaderElectionID, "leader-election-id", "coralogix-operator",
			"Name of the leader election lease. Used to manage the leader election process.")
		flag.BoolVar(&cfg.SecureMetrics, "metrics-secure", false,
			"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
		flag.BoolVar(&cfg.EnableHTTP2, "enable-http2", false,
			"If set, HTTP/2 will be enabled for the metrics and webhook servers")
		flag.BoolVar(&cfg.PrometheusRuleController, "prometheus-rule-controller", true,
			"Determine if the prometheus rule controller should be started. Default is true.")
		flag.StringVar(&cfg.RecordingRuleGroupSetSuffix, "recording-rule-group-set-suffix", "",
			"Suffix to be added to the RecordingRuleGroupSet")

		region := os.Getenv("CORALOGIX_REGION")
		flag.StringVar(&region, "region", region, fmt.Sprintf("The region of your Coralogix cluster. Can be one of %q. Conflicts with 'domain'.", validRegions))

		domain := os.Getenv("CORALOGIX_DOMAIN")
		flag.StringVar(&domain, "domain", domain, "The domain of your Coralogix cluster. Conflicts with 'region'.")

		apiKey := os.Getenv("CORALOGIX_API_KEY")
		flag.StringVar(&cfg.CoralogixApiKey, "api-key", apiKey, "The proper api-key based on your Coralogix cluster's region.")

		labelSelector := os.Getenv("LABEL_SELECTOR")
		flag.StringVar(&labelSelector, "label-selector", labelSelector, "A labelsSelector structure to filter resources by their labels.")

		namespaceSelector := os.Getenv("NAMESPACE_SELECTOR")
		flag.StringVar(&namespaceSelector, "namespace-selector", namespaceSelector, "A labelsSelector structure to filter resources by their namespaces' labels.")

		reconcileIntervals := getReconcileIntervals()

		opts := zap.Options{}
		opts.BindFlags(flag.CommandLine)
		flag.Parse()

		ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

		var err error
		cfg.CoralogixGrpcUrl, err = getCoralogixGrpcUrl(strings.ToUpper(region), domain)
		if err != nil {
			setupLog.Error(err, "invalid arguments for running operator")
			os.Exit(1)
		}

		cfg.CoralogixOpenApiUrl, err = getCoralogixOpenApiUrl(strings.ToUpper(region), domain)
		if err != nil {
			setupLog.Error(err, "invalid arguments for running operator")
			os.Exit(1)
		}

		if cfg.CoralogixApiKey == "" {
			setupLog.Error(fmt.Errorf("api-key can not be empty"),
				"invalid arguments for running operator")
			os.Exit(1)
		}

		selector, err := parseSelector(labelSelector, namespaceSelector)
		if err != nil {
			setupLog.Error(err, "invalid arguments for running operator")
			os.Exit(1)
		}
		cfg.Selector = *selector

		cfg.ReconcileIntervals, err = parseReconcileIntervals(reconcileIntervals)
		if err != nil {
			setupLog.Error(err, "invalid arguments for running operator")
			os.Exit(1)
		}
	})

	return cfg
}

func GetConfig() *Config {
	return cfg
}

func getReconcileIntervals() map[string]*string {
	result := make(map[string]*string)
	gvks := utils.GetGVKs(GetScheme())
	for _, gvk := range gvks {
		interval := os.Getenv(fmt.Sprintf("%s_RECONCILE_INTERVAL_SECONDS", strings.ToUpper(gvk.Kind)))
		flag.StringVar(
			&interval,
			fmt.Sprintf("%s-reconcile-interval-seconds", strings.ToLower(gvk.Kind)),
			interval,
			fmt.Sprintf("The interval in seconds between succeding reconciliations for %s. "+
				"Should be at least 30 seconds.", gvk.Kind),
		)
		result[gvk.Kind] = &interval
	}

	return result
}

func parseReconcileIntervals(intervals map[string]*string) (map[string]time.Duration, error) {
	result := make(map[string]time.Duration)
	for crd, interval := range intervals {
		// Default to 0 if not set, which means no custom interval for the CRD.
		// Leaving the operator to reconcile every 10 hours, according to the default manager settings.
		// More info: https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.3/pkg/cache#Options
		if *interval == "" {
			*interval = "0"
		}

		numericInterval, err := strconv.Atoi(*interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval value for %s: %w", crd, err)
		}

		if numericInterval != 0 && numericInterval < 30 {
			return nil, fmt.Errorf("interval value should be at least 30 seconds")
		}

		result[crd] = time.Second * time.Duration(numericInterval)
	}
	return result, nil
}

func getCoralogixGrpcUrl(region, domain string) (string, error) {
	if err := validateRegionAndDomain(region, domain); err != nil {
		return "", err
	}

	if region != "" {
		if !slices.Contains(validRegions, region) {
			return "", fmt.Errorf("region value is '%s', but can be one of %q", region, validRegions)
		}
		return region, nil
	}

	return domain, nil
}

func getCoralogixOpenApiUrl(region, domain string) (string, error) {
	if err := validateRegionAndDomain(region, domain); err != nil {
		return "", err
	}

	if region != "" {
		if !slices.Contains(validRegions, region) {
			return "", fmt.Errorf("region value is '%s', but can be one of %q", region, validRegions)
		}
		return openapicxsdk.URLFromRegion(strings.ToLower(region)), nil
	}

	return openapicxsdk.URLFromDomain(domain), nil
}

func validateRegionAndDomain(region, domain string) error {
	if region != "" && domain != "" {
		return fmt.Errorf("region and domain flags are mutually exclusive")
	}

	if region == "" && domain == "" {
		return fmt.Errorf("region or domain must be set")
	}

	return nil
}

func InitClient(c client.Client) {
	CrClient = c
}

func InitScheme(s *runtime.Scheme) {
	scheme = s
}

func GetClient() client.Client {
	return CrClient
}

func GetScheme() *runtime.Scheme {
	return scheme
}
