package uiplugin

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

var pluginConfigAll = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			ACM: &uiv1alpha1.AdvancedClusterManagementReference{
				Enabled: true,
				Alertmanager: uiv1alpha1.AlertmanagerReference{
					Url: "https://alertmanager.open-cluster-management-observability.svc:9095",
				},
				ThanosQuerier: uiv1alpha1.ThanosQuerierReference{
					Url: "https://rbac-query-proxy.open-cluster-management-observability.svc:8443",
				},
			},
			Perses: &uiv1alpha1.PersesReference{
				Enabled: true,
			},
			Incidents: &uiv1alpha1.IncidentsReference{
				Enabled: true,
			},
		},
	},
}

var pluginConfigPerses = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			Perses: &uiv1alpha1.PersesReference{
				Enabled: true,
			},
		},
	},
}

var pluginConfigPersesDefault = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			Perses: &uiv1alpha1.PersesReference{
				Enabled: true,
			},
		},
	},
}

var pluginConfigPersesEmpty = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			Perses: &uiv1alpha1.PersesReference{},
		},
	},
}

var pluginConfigACM = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			ACM: &uiv1alpha1.AdvancedClusterManagementReference{
				Enabled: true,
				Alertmanager: uiv1alpha1.AlertmanagerReference{
					Url: "https://alertmanager.open-cluster-management-observability.svc:9095",
				},
				ThanosQuerier: uiv1alpha1.ThanosQuerierReference{
					Url: "https://rbac-query-proxy.open-cluster-management-observability.svc:8443",
				},
			},
		},
	},
}

var pluginConfigThanos = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			ACM: &uiv1alpha1.AdvancedClusterManagementReference{
				ThanosQuerier: uiv1alpha1.ThanosQuerierReference{
					Url: "https://rbac-query-proxy.open-cluster-management-observability.svc:8443",
				},
			},
		},
	},
}

var pluginConfigAlertmanager = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			ACM: &uiv1alpha1.AdvancedClusterManagementReference{
				Alertmanager: uiv1alpha1.AlertmanagerReference{
					Url: "https://alertmanager.open-cluster-management-observability.svc:9095",
				},
			},
		},
	},
}

var pluginConfigIncidents = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{
			Incidents: &uiv1alpha1.IncidentsReference{
				Enabled: true,
			},
		},
	},
}

var pluginMalformed = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "monitoring-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type:       "monitoring",
		Monitoring: &uiv1alpha1.MonitoringConfig{},
	},
}

type featureFlagsStatus struct {
	acmAlerting bool
	perses      bool
	incidents   bool
}

func containsFeatureFlag(pluginInfo *UIPluginInfo) featureFlagsStatus {
	var status featureFlagsStatus
	var featuresArg string

	for _, arg := range pluginInfo.ExtraArgs {
		if strings.HasPrefix(arg, "-features=") {
			featuresArg = arg
			break
		}
	}

	if featuresArg == "" {
		return status // No features argument found
	}

	valuePart := strings.TrimPrefix(featuresArg, "-features=")
	if valuePart == "" {
		return status // No features listed after "="
	}

	features := strings.Split(valuePart, ",")

	for _, feature := range features {
		trimmedFeature := strings.TrimSpace(feature)
		switch trimmedFeature {
		case "acm-alerting":
			status.acmAlerting = true
		case "perses-dashboards":
			status.perses = true
		case "incidents":
			status.incidents = true
		}
	}
	return status
}

type proxiesStatus struct {
	alertmanager bool
	thanos       bool
	perses       bool
}

func containsProxy(pluginInfo *UIPluginInfo) proxiesStatus {
	var status proxiesStatus

	for _, proxy := range pluginInfo.Proxies {
		switch proxy.Alias {
		case "alertmanager-proxy":
			status.alertmanager = true
		case "thanos-proxy":
			status.thanos = true
		case "perses":
			status.perses = true
		}
	}
	return status
}

func containsHealthAnalyzer(pluginInfo *UIPluginInfo) bool {
	return pluginInfo.HealthAnalyzerImage == healthAnalyzerImage
}

func containsPerses(pluginInfo *UIPluginInfo) bool {
	return pluginInfo.PersesImage == persesImage
}

const healthAnalyzerImage = "quay.io/health-analyzer-foo-test:123"
const persesImage = "quay.io/perses-foo-test:123"

func getPluginInfo(plugin *uiv1alpha1.UIPlugin, features []string, clusterVersion string) (*UIPluginInfo, error) {
	const (
		namespace = "openshift-operators"
		name      = "monitoring"
		image     = "quay.io/monitoring-foo-test:123"
	)

	return createMonitoringPluginInfo(plugin, namespace, name, image, features, clusterVersion, healthAnalyzerImage, persesImage)
}

func TestCreateMonitoringPluginInfo(t *testing.T) {
	featuresForTest := []string{}

	type expectedComponents struct {
		persesImage    bool
		healthAnalyzer bool
	}

	type testCase struct {
		name                  string
		pluginConfig          *uiv1alpha1.UIPlugin
		proxies               proxiesStatus
		featureFlags          featureFlagsStatus
		components            expectedComponents
		expectedErrorMessage  string
		clusterVersionsToTest []string
	}

	testCfgs := []testCase{
		{
			name:         "All monitoring configurations",
			pluginConfig: pluginConfigAll,
			proxies: proxiesStatus{
				alertmanager: true,
				thanos:       true,
				perses:       true,
			},
			featureFlags: featureFlagsStatus{
				acmAlerting: true,
				perses:      true,
				incidents:   true,
			},
			components: expectedComponents{
				persesImage:    true,
				healthAnalyzer: true,
			},
			clusterVersionsToTest: []string{"v4.19"},
		},
		{
			name:         "All monitoring configurations",
			pluginConfig: pluginConfigAll,
			proxies: proxiesStatus{
				alertmanager: true,
				thanos:       true,
				perses:       true,
			},
			featureFlags: featureFlagsStatus{
				acmAlerting: true,
				perses:      true,
				incidents:   false, // Differs for v4.18
			},
			components: expectedComponents{
				persesImage:    true,
				healthAnalyzer: false, // Differs for v4.18
			},
			clusterVersionsToTest: []string{"v4.18"},
		},
		{
			name:         "ACM configuration only",
			pluginConfig: pluginConfigACM,
			proxies: proxiesStatus{
				alertmanager: true,
				thanos:       true,
				perses:       false,
			},
			featureFlags: featureFlagsStatus{
				acmAlerting: true,
				perses:      false,
				incidents:   false,
			},
			components: expectedComponents{
				persesImage:    false,
				healthAnalyzer: false,
			},
			clusterVersionsToTest: []string{"v4.19", "v4.18"},
		},
		{
			name:         "Perses configuration only",
			pluginConfig: pluginConfigPerses,
			proxies: proxiesStatus{
				alertmanager: false,
				thanos:       false,
				perses:       true,
			},
			featureFlags: featureFlagsStatus{
				acmAlerting: false,
				perses:      true,
				incidents:   false,
			},
			components: expectedComponents{
				persesImage:    true,
				healthAnalyzer: false,
			},
			clusterVersionsToTest: []string{"v4.19", "v4.18"},
		},
		{
			name:         "Perses default namespace and name",
			pluginConfig: pluginConfigPersesDefault,
			proxies: proxiesStatus{
				alertmanager: false,
				thanos:       false,
				perses:       true,
			},
			featureFlags: featureFlagsStatus{
				acmAlerting: false,
				perses:      true,
				incidents:   false,
			},
			components: expectedComponents{
				persesImage:    true,
				healthAnalyzer: false,
			},
			clusterVersionsToTest: []string{"v4.19", "v4.18"},
		},
		{
			name:                  "Incidents configuration only",
			pluginConfig:          pluginConfigIncidents,
			expectedErrorMessage:  "all uiplugin monitoring configurations are invalid or not supported in this cluster version",
			clusterVersionsToTest: []string{"v4.18"},
		},
		{
			name:         "Incidents configuration only",
			pluginConfig: pluginConfigIncidents,
			proxies: proxiesStatus{
				alertmanager: false,
				thanos:       false,
				perses:       false,
			},
			featureFlags: featureFlagsStatus{
				acmAlerting: false,
				perses:      false,
				incidents:   true,
			},
			components: expectedComponents{
				persesImage:    false,
				healthAnalyzer: true,
			},
			clusterVersionsToTest: []string{"v4.19"},
		},
	}

	for _, tc := range testCfgs {
		for _, cv := range tc.clusterVersionsToTest {
			t.Run(tc.name+"_"+cv, func(t *testing.T) {
				pluginInfo, err := getPluginInfo(tc.pluginConfig, featuresForTest, cv)

				if tc.expectedErrorMessage != "" {
					assert.ErrorContains(t, err, tc.expectedErrorMessage, "Expected an error for invalid configuration")
					assert.Assert(t, pluginInfo == nil, "Expected pluginInfo to be nil")
					return
				}

				assert.NilError(t, err, "getPluginInfo returned an unexpected error")
				if err != nil {
					return
				}

				actualProxies := containsProxy(pluginInfo)
				assert.Equal(t, actualProxies.alertmanager, tc.proxies.alertmanager, "Alertmanager proxy mismatch")
				assert.Equal(t, actualProxies.thanos, tc.proxies.thanos, "Thanos proxy mismatch")
				assert.Equal(t, actualProxies.perses, tc.proxies.perses, "Perses proxy mismatch")

				actualFlags := containsFeatureFlag(pluginInfo)
				assert.Equal(t, actualFlags.acmAlerting, tc.featureFlags.acmAlerting, "ACM alerting flag mismatch")
				assert.Equal(t, actualFlags.perses, tc.featureFlags.perses, "Perses flag mismatch")
				assert.Equal(t, actualFlags.incidents, tc.featureFlags.incidents, "Incidents flag mismatch")

				assert.Equal(t, containsHealthAnalyzer(pluginInfo), tc.components.healthAnalyzer, "Health analyzer mismatch")
				assert.Equal(t, containsPerses(pluginInfo), tc.components.persesImage, "Perses image mismatch")
			})
		}
	}

	type invalidConfigTestCase struct {
		name         string
		pluginConfig *uiv1alpha1.UIPlugin
	}

	negativeTestCfgs := []invalidConfigTestCase{
		{
			name:         "Missing URL from thanos in ACM config",
			pluginConfig: pluginConfigAlertmanager, // ACM enabled, Alertmanager URL set, Thanos URL missing
		},
		{
			name:         "Missing URL from alertmanager in ACM config",
			pluginConfig: pluginConfigThanos, // ACM enabled, Thanos URL set, Alertmanager URL missing
		},
		{
			name:         "Missing Perses enabled field when Perses config is present",
			pluginConfig: pluginConfigPersesEmpty, // PersesReference{}
		},
		{
			name:         "Malformed UIPlugin custom resource (no monitoring sections enabled)",
			pluginConfig: pluginMalformed, // MonitoringConfig{}
		},
	}

	clusterVersionsToTest := []string{"v4.19", "v4.18"} // This variable is used for negative tests

	for _, cv := range clusterVersionsToTest {
		t.Run(fmt.Sprintf("NegativeTests_ClusterVersion_%s", cv), func(t *testing.T) {
			for _, tc := range negativeTestCfgs {
				t.Run(tc.name, func(t *testing.T) {
					pluginInfo, err := getPluginInfo(tc.pluginConfig, featuresForTest, cv)
					assert.Assert(t, err != nil, "Expected an error for invalid configuration")
					assert.Assert(t, pluginInfo == nil, "Expected pluginInfo to be nil on error")
				})
			}
		})
	}

	t.Run("Test validateIncidentsConfig() with valid and invalid clusterVersion formats", func(t *testing.T) {
		// should not throw an error because all these are valid formats for clusterVersion
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.20.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.20.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.20") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.20.0") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.20.0") == true)

		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.19.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.19.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.19") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.19.0") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.19.0") == true)

		// should be invalid clusterVersion because UIPlugin incident feature is supported in OCP v4.19+
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.18.0-0.nightly-2024-06-06-064349") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.18.0-0.nightly-2024-06-06-064349") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.18") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.18.0") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.18.0") == false)
	})
}
