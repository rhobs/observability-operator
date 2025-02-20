package uiplugin

import (
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

var namespace = "openshift-operators"
var name = "monitoring"
var image = "quay.io/monitoring-foo-test:123"

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
			ACM: uiv1alpha1.AdvancedClusterManagementReference{
				Enabled: true,
				Alertmanager: uiv1alpha1.AlertmanagerReference{
					Url: "https://alertmanager.open-cluster-management-observability.svc:9095",
				},
				ThanosQuerier: uiv1alpha1.ThanosQuerierReference{
					Url: "https://rbac-query-proxy.open-cluster-management-observability.svc:8443",
				},
			},
			Perses: uiv1alpha1.PersesReference{
				Enabled:     true,
				ServiceName: "perses-api-http",
				Namespace:   "perses-operator",
			},
			Incidents: uiv1alpha1.IncidentsReference{
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
			Perses: uiv1alpha1.PersesReference{
				Enabled:     true,
				ServiceName: "perses-api-http",
				Namespace:   "perses-operator",
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
			Perses: uiv1alpha1.PersesReference{
				Enabled:   true,
				Namespace: "perses-operator",
			},
		},
	},
}

var pluginConfigPersesDefaultServiceName = &uiv1alpha1.UIPlugin{
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
			Perses: uiv1alpha1.PersesReference{
				Enabled:     true,
				ServiceName: "foo-perses-api-http",
			},
		},
	},
}

var pluginConfigPersesDefaultNamespace = &uiv1alpha1.UIPlugin{
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
			Perses: uiv1alpha1.PersesReference{
				Enabled:   true,
				Namespace: "foo-perses-operator",
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
			Perses: uiv1alpha1.PersesReference{},
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
			ACM: uiv1alpha1.AdvancedClusterManagementReference{
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
			ACM: uiv1alpha1.AdvancedClusterManagementReference{
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
			ACM: uiv1alpha1.AdvancedClusterManagementReference{
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
			Incidents: uiv1alpha1.IncidentsReference{
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

func containsFeatureFlag(pluginInfo *UIPluginInfo) (bool, bool, bool) {
	acmAlertingFound, persesFound, incidentsFound := false, false, false
	var featuresIndex int

	// Loop through the array to find the index of "-features"
	for i, arg := range pluginInfo.ExtraArgs {
		if strings.Contains(arg, "-features") {
			featuresIndex = i
			break
		}
	}

	// Get "-features=" list from ExtraArgs field
	// (e.g. "-features='acm-alerting', 'perses-dashboards', 'incidents'")
	re := regexp.MustCompile(`-features=([a-zA-Z0-9,\-]+)`)
	featuresList := re.FindString(pluginInfo.ExtraArgs[featuresIndex])

	// Get individual feature strings, by spliting string after "=" and between ","
	features := strings.Split(strings.Split(featuresList, "=")[1], ",")

	// Check if features are listed
	for _, feature := range features {
		if feature == "acm-alerting" {
			acmAlertingFound = true
		}
		if feature == "perses-dashboards" {
			persesFound = true
		}
		if feature == "incidents" {
			incidentsFound = true
		}
	}

	return acmAlertingFound, persesFound, incidentsFound
}

func containsProxy(pluginInfo *UIPluginInfo) (bool, bool, bool) {
	alertmanagerFound, thanosFound, persesFound := false, false, false

	for _, proxy := range pluginInfo.Proxies {
		if proxy.Alias == "alertmanager-proxy" {
			alertmanagerFound = true
		}
		if proxy.Alias == "thanos-proxy" {
			thanosFound = true
		}
		if proxy.Alias == "perses" {
			persesFound = true
		}
	}
	return alertmanagerFound, thanosFound, persesFound
}

var features = []string{}
var acmVersion = "v2.11"
var clusterVersion = "v4.18"

func getPluginInfo(plugin *uiv1alpha1.UIPlugin, features []string, clusterVersion string, acmVersion string) (*UIPluginInfo, error) {
	return createMonitoringPluginInfo(plugin, namespace, name, image, features, acmVersion, clusterVersion)
}

func TestCreateMonitoringPluginInfo(t *testing.T) {
	/** Postive Test - ALL  **/
	t.Run("Test createMonitoringPluginInfo with all monitoring configurations", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigAll, features, clusterVersion, acmVersion)
		assert.Assert(t, error == nil)

		alertmanagerProxyFound, thanosProxyFound, persesProxyFound := containsProxy(pluginInfo)
		assert.Assert(t, alertmanagerProxyFound == true)
		assert.Assert(t, thanosProxyFound == true)
		assert.Assert(t, persesProxyFound == true)

		acmAlertingFlagFound, persesFlagFound, incidentsFlagFound := containsFeatureFlag(pluginInfo)
		assert.Assert(t, acmAlertingFlagFound == true)
		assert.Assert(t, persesFlagFound == true)
		assert.Assert(t, incidentsFlagFound == true)
	})

	/** Postive Test - ACM  **/
	t.Run("Test createMonitoringPluginInfo with AMC configuration only", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigACM, features, clusterVersion, acmVersion)
		assert.Assert(t, error == nil)

		alertmanagerProxyFound, thanosProxyFound, persesProxyFound := containsProxy(pluginInfo)
		assert.Assert(t, alertmanagerProxyFound == true)
		assert.Assert(t, thanosProxyFound == true)
		assert.Assert(t, persesProxyFound == false)

		acmAlertingFlagFound, persesFlagFound, incidentsFlagFound := containsFeatureFlag(pluginInfo)
		assert.Assert(t, acmAlertingFlagFound == true)
		assert.Assert(t, persesFlagFound == false)
		assert.Assert(t, incidentsFlagFound == false)
	})

	t.Run("Test validACMConfig() with valid and invalid acmVersions", func(t *testing.T) {

		// UIPlugin monitoring ACM feature is only supported in v2.11+
		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "v2.11.3") == true)
		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "v2.11") == true)
		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "2.11") == true)
		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "v2.11") == true)

		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "2.10") == false)
		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "v2.10") == false)
		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "1.0.0") == false)
		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "v1.0.0") == false)

		assert.Assert(t, validateACMConfig(pluginConfigACM.Spec.Monitoring, "acm version not found") == false)
	})

	/** Postive Test - Perses  **/
	t.Run("Test createMonitoringPluginInfo with Perses configuration only", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigPerses, features, clusterVersion, acmVersion)
		assert.Assert(t, error == nil)

		alertmanagerProxyFound, thanosProxyFound, persesProxyFound := containsProxy(pluginInfo)
		assert.Assert(t, alertmanagerProxyFound == false)
		assert.Assert(t, thanosProxyFound == false)
		assert.Assert(t, persesProxyFound == true)

		acmAlertingFlagFound, persesFlagFound, incidentsFlagFound := containsFeatureFlag(pluginInfo)
		assert.Assert(t, acmAlertingFlagFound == false)
		assert.Assert(t, persesFlagFound == true)
		assert.Assert(t, incidentsFlagFound == false)

	})

	t.Run("Test createMonitoringPluginInfo with Perses default namespace and namespace", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigPersesDefault, features, clusterVersion, acmVersion)
		assert.Assert(t, error == nil)

		alertmanagerProxyFound, thanosProxyFound, persesProxyFound := containsProxy(pluginInfo)
		assert.Assert(t, alertmanagerProxyFound == false)
		assert.Assert(t, thanosProxyFound == false)
		assert.Assert(t, persesProxyFound == true)

		acmAlertingFlagFound, persesFlagFound, incidentsFlagFound := containsFeatureFlag(pluginInfo)
		assert.Assert(t, acmAlertingFlagFound == false)
		assert.Assert(t, persesFlagFound == true)
		assert.Assert(t, incidentsFlagFound == false)

	})

	t.Run("Test createMonitoringPluginInfo with Perses default serviceName", func(t *testing.T) {
		// should not throw an error because serviceName is allowed to be empty
		// a default serviceName will be assigned
		pluginInfo, error := getPluginInfo(pluginConfigPersesDefaultServiceName, features, clusterVersion, acmVersion)
		assert.Assert(t, error == nil)

		alertmanagerProxyFound, thanosProxyFound, persesProxyFound := containsProxy(pluginInfo)
		assert.Assert(t, alertmanagerProxyFound == false)
		assert.Assert(t, thanosProxyFound == false)
		assert.Assert(t, persesProxyFound == true)

		acmAlertingFlagFound, persesFlagFound, incidentsFlagFound := containsFeatureFlag(pluginInfo)
		assert.Assert(t, acmAlertingFlagFound == false)
		assert.Assert(t, persesFlagFound == true)
		assert.Assert(t, incidentsFlagFound == false)
	})

	t.Run("Test createMonitoringPluginInfo with Perses default namespace", func(t *testing.T) {
		// should not throw an error because namespace is allowed to be empty
		// a default namespace will be assigned
		pluginInfo, error := getPluginInfo(pluginConfigPersesDefaultNamespace, features, clusterVersion, acmVersion)
		assert.Assert(t, error == nil)

		alertmanagerProxyFound, thanosProxyFound, persesProxyFound := containsProxy(pluginInfo)
		assert.Assert(t, alertmanagerProxyFound == false)
		assert.Assert(t, thanosProxyFound == false)
		assert.Assert(t, persesProxyFound == true)

		acmAlertingFlagFound, persesFlagFound, incidentsFlagFound := containsFeatureFlag(pluginInfo)
		assert.Assert(t, acmAlertingFlagFound == false)
		assert.Assert(t, persesFlagFound == true)
		assert.Assert(t, incidentsFlagFound == false)
	})

	/** Postive Test - Incidients **/
	t.Run("Test createMonitoringPluginInfo with Incidents configuration only", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigIncidents, features, clusterVersion, acmVersion)
		assert.Assert(t, error == nil)

		alertmanagerProxyFound, thanosProxyFound, persesProxyFound := containsProxy(pluginInfo)
		assert.Assert(t, alertmanagerProxyFound == false)
		assert.Assert(t, thanosProxyFound == false)
		assert.Assert(t, persesProxyFound == false)

		acmAlertingFlagFound, persesFlagFound, incidentsFlagFound := containsFeatureFlag(pluginInfo)
		assert.Assert(t, acmAlertingFlagFound == false)
		assert.Assert(t, persesFlagFound == false)
		assert.Assert(t, incidentsFlagFound == true)
	})

	t.Run("Test validateIncidentsConfig() with valid and invalid clusterVersion formats", func(t *testing.T) {
		// should not throw an error because all these are valid formats for clusterVersion
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.19.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.19.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.19.0") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.19.0") == true)

		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.18.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.18.0-0.nightly-2024-06-06-064349") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.18") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.18.0") == true)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.18.0") == true)

		// should be invalid clusterVersion because UIPlugin incident feature is supported in OCP v4.18+
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.17.21-0.nightly-2024-06-06-064349") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.17.0-0.nightly-2024-06-06-064349") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "v4.17.0") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.17.0") == false)
		assert.Assert(t, validateIncidentsConfig(pluginConfigIncidents.Spec.Monitoring, "4.17") == false)

	})

	/** NEGATIVE TESTS **/

	/** Negative Tests - ACM **/
	t.Run("Test createMonitoringPluginInfo with missing URL from thanos", func(t *testing.T) {
		// this should throw an error because thanosQuerier.URL is not set
		pluginInfo, error := getPluginInfo(pluginConfigAlertmanager, features, clusterVersion, acmVersion)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
	})

	t.Run("Test createMonitoringPluginInfo with missing URL from alertmanager ", func(t *testing.T) {
		// this should throw an error because alertManager.URL is not set
		pluginInfo, error := getPluginInfo(pluginConfigThanos, features, clusterVersion, acmVersion)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
	})

	/** Negative Tests - Perses **/
	t.Run("Test createMonitoringPluginInfo with missing Perses enabled field ", func(t *testing.T) {
		// this should throw an error because 'enabled: true' is not set
		pluginInfo, error := getPluginInfo(pluginConfigPersesEmpty, features, clusterVersion, acmVersion)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
	})

	/** Negative Tests - ALL **/
	t.Run("Test createMonitoringPluginInfo with malform UIPlugin custom resource", func(t *testing.T) {
		// this should throw an error because UIPlugin doesn't include alertmanager, thanos, perses, or incidents
		pluginInfo, error := getPluginInfo(pluginMalformed, features, clusterVersion, acmVersion)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
	})
}
