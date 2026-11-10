package uiplugin

import (
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
			ClusterHealthAnalyzer: &uiv1alpha1.ClusterHealthAnalyzerReference{
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
				Enabled: true,
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
				Enabled: true,
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

var pluginConfigClusterHealthAnalyzer = &uiv1alpha1.UIPlugin{
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
			ClusterHealthAnalyzer: &uiv1alpha1.ClusterHealthAnalyzerReference{
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
	acmAlerting           bool
	perses                bool
	incidents             bool
	clusterHealthAnalyzer bool
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
		case "cluster-health-analyzer":
			status.clusterHealthAnalyzer = true
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

func getPluginInfo(plugin *uiv1alpha1.UIPlugin, features []string) (*UIPluginInfo, error) {
	const (
		namespace = "openshift-operators"
		name      = "monitoring"
		image     = "quay.io/monitoring-foo-test:123"
	)

	return createMonitoringPluginInfo(plugin, namespace, name, image, features, healthAnalyzerImage, persesImage)
}

func TestCreateMonitoringPluginInfoDisabledFeatures(t *testing.T) {
	tt := []struct {
		name   string
		config *uiv1alpha1.MonitoringConfig
	}{
		{
			name: "ACM disabled",
			config: &uiv1alpha1.MonitoringConfig{
				ACM: &uiv1alpha1.AdvancedClusterManagementReference{Enabled: false},
			},
		},
		{
			name:   "Perses disabled",
			config: pluginConfigPersesEmpty.Spec.Monitoring,
		},
		{
			name: "Incidents disabled",
			config: &uiv1alpha1.MonitoringConfig{
				Incidents: &uiv1alpha1.IncidentsReference{Enabled: false},
			},
		},
		{
			name: "Cluster health analyzer disabled",
			config: &uiv1alpha1.MonitoringConfig{
				ClusterHealthAnalyzer: &uiv1alpha1.ClusterHealthAnalyzerReference{Enabled: false},
			},
		},
		{
			name: "Perses and cluster health analyzer disabled",
			config: &uiv1alpha1.MonitoringConfig{
				Perses:                &uiv1alpha1.PersesReference{Enabled: false},
				ClusterHealthAnalyzer: &uiv1alpha1.ClusterHealthAnalyzerReference{Enabled: false},
			},
		},
		{
			name: "All optional features disabled",
			config: &uiv1alpha1.MonitoringConfig{
				ACM:                   &uiv1alpha1.AdvancedClusterManagementReference{Enabled: false},
				Perses:                &uiv1alpha1.PersesReference{Enabled: false},
				Incidents:             &uiv1alpha1.IncidentsReference{Enabled: false},
				ClusterHealthAnalyzer: &uiv1alpha1.ClusterHealthAnalyzerReference{Enabled: false},
			},
		},
	}

	for _, tc := range tt {
		for _, cv := range []string{"v4.18", "v4.19"} {
			t.Run(tc.name+"_"+cv, func(t *testing.T) {
				plugin := &uiv1alpha1.UIPlugin{
					Spec: uiv1alpha1.UIPluginSpec{
						Type:       uiv1alpha1.TypeMonitoring,
						Monitoring: tc.config,
					},
				}

				pluginInfo, err := getPluginInfo(plugin, nil, cv)
				assert.NilError(t, err)
				assert.Assert(t, pluginInfo != nil)
				assert.Equal(t, pluginInfo.AreMonitoringFeatsDisabled, false)
				assert.Equal(t, pluginInfo.Image, "quay.io/monitoring-foo-test:123")
				assert.Equal(t, pluginInfo.PersesImage, "")
				assert.Equal(t, pluginInfo.HealthAnalyzerImage, "")
				assert.Equal(t, containsFeatureFlag(pluginInfo), featureFlagsStatus{})
				assert.Equal(t, containsProxy(pluginInfo), proxiesStatus{})
				assert.Equal(t, len(pluginInfo.Proxies), 1)
				assert.Equal(t, pluginInfo.Proxies[0].Alias, "backend")
				assert.DeepEqual(t, pluginInfo.ExtraArgs, []string{
					"-config-path=/opt/app-root/config",
					"-static-path=/opt/app-root/web/dist",
					"-features=mcp-overview",
				})
			})
		}
	}
}

func TestCreateMonitoringPluginInfo(t *testing.T) {
	featuresForTest := []string{}

	type expectedComponents struct {
		persesImage           bool
		clusterHealthAnalyzer bool
	}

	type testCase struct {
		name                 string
		pluginConfig         *uiv1alpha1.UIPlugin
		proxies              proxiesStatus
		featureFlags         featureFlagsStatus
		components           expectedComponents
		expectedErrorMessage string
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
				acmAlerting:           true,
				perses:                true,
				clusterHealthAnalyzer: true,
				incidents:             false,
			},
			components: expectedComponents{
				persesImage:           true,
				clusterHealthAnalyzer: true,
			},
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
				acmAlerting:           true,
				perses:                false,
				clusterHealthAnalyzer: false,
				incidents:             false,
			},
			components: expectedComponents{
				persesImage:           false,
				clusterHealthAnalyzer: false,
			},
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
				acmAlerting:           false,
				perses:                true,
				clusterHealthAnalyzer: false,
				incidents:             false,
			},
			components: expectedComponents{
				persesImage:           true,
				clusterHealthAnalyzer: false,
			},
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
				acmAlerting:           false,
				perses:                true,
				clusterHealthAnalyzer: false,
				incidents:             false,
			},
			components: expectedComponents{
				persesImage:           true,
				clusterHealthAnalyzer: false,
			},
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
				acmAlerting:           false,
				perses:                false,
				clusterHealthAnalyzer: true,
				incidents:             false,
			},
			components: expectedComponents{
				persesImage:           false,
				clusterHealthAnalyzer: true,
			},
		},
		{
			name:         "ClusterHealthAnalyzer configuration only",
			pluginConfig: pluginConfigClusterHealthAnalyzer,
			proxies: proxiesStatus{
				alertmanager: false,
				thanos:       false,
				perses:       false,
			},
			featureFlags: featureFlagsStatus{
				acmAlerting:           false,
				perses:                false,
				clusterHealthAnalyzer: true,
				incidents:             false,
			},
			components: expectedComponents{
				persesImage:           false,
				clusterHealthAnalyzer: true,
			},
		},
	}

	for _, tc := range testCfgs {
		t.Run(tc.name, func(t *testing.T) {
			pluginInfo, err := getPluginInfo(tc.pluginConfig, featuresForTest)

			if tc.expectedErrorMessage != "" {
				assert.ErrorContains(t, err, tc.expectedErrorMessage, "Expected an error for invalid configuration")
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
			assert.Equal(t, actualFlags.clusterHealthAnalyzer, tc.featureFlags.clusterHealthAnalyzer, "Incidents flag mismatch")
			assert.Equal(t, actualFlags.incidents, tc.featureFlags.incidents, "Incidents flag mismatch")

			assert.Equal(t, containsHealthAnalyzer(pluginInfo), tc.components.clusterHealthAnalyzer, "Health analyzer mismatch")
			assert.Equal(t, containsPerses(pluginInfo), tc.components.persesImage, "Perses image mismatch")
		})
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
			name:         "Malformed UIPlugin custom resource (no monitoring sections enabled)",
			pluginConfig: pluginMalformed, // MonitoringConfig{}
		},
	}

	for _, tc := range negativeTestCfgs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getPluginInfo(tc.pluginConfig, featuresForTest)
			assert.Assert(t, err != nil, "Expected an error for invalid configuration")
		})
	}

	t.Run("Test validateIncidentsConfig() with valid and invalid clusterVersion formats", func(t *testing.T) {
		assertIncidentsSupport := func(version string, supported bool) {
			t.Helper()
			enabled, valid := validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, version)
			assert.Equal(t, enabled, supported, "Incidents enablement mismatch for %s", version)
			assert.Equal(t, valid, supported, "Incidents validity mismatch for %s", version)
		}

		assertIncidentsSupport("v4.21.0-0.nightly-2024-06-06-064349", true)
		assertIncidentsSupport("4.21.0-0.nightly-2024-06-06-064349", true)
		assertIncidentsSupport("v4.21", true)
		assertIncidentsSupport("v4.21.0", true)
		assertIncidentsSupport("4.21.0", true)

		assertIncidentsSupport("v4.20.0-0.nightly-2024-06-06-064349", true)
		assertIncidentsSupport("4.20.0-0.nightly-2024-06-06-064349", true)
		assertIncidentsSupport("v4.20", true)
		assertIncidentsSupport("v4.20.0", true)
		assertIncidentsSupport("4.20.0", true)

		assertIncidentsSupport("v4.19.0-0.nightly-2024-06-06-064349", true)
		assertIncidentsSupport("4.19.0-0.nightly-2024-06-06-064349", true)
		assertIncidentsSupport("v4.19", true)
		assertIncidentsSupport("v4.19.0", true)
		assertIncidentsSupport("4.19.0", true)

		// Incidents require OCP v4.19+.
		assertIncidentsSupport("v4.18.0-0.nightly-2024-06-06-064349", false)
		assertIncidentsSupport("4.18.0-0.nightly-2024-06-06-064349", false)
		assertIncidentsSupport("v4.18", false)
		assertIncidentsSupport("v4.18.0", false)
		assertIncidentsSupport("4.18.0", false)
	})
}
