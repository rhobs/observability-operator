package uiplugin

import (
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
			Alertmanager: uiv1alpha1.AlertmanagerReference{
				Url: "https://alertmanager.open-cluster-management-observability.svc:9095",
			},
			ThanosQuerier: uiv1alpha1.ThanosQuerierReference{
				Url: "https://rbac-query-proxy.open-cluster-management-observability.svc:8443",
			},
			Perses: uiv1alpha1.PersesReference{
				Name:      "perses-api-http",
				Namespace: "perses-operator",
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
				Name:      "perses-api-http",
				Namespace: "perses-operator",
			},
		},
	},
}

var pluginConfigPersesName = &uiv1alpha1.UIPlugin{
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
				Namespace: "perses-operator",
			},
		},
	},
}

var pluginConfigPersesNameSpace = &uiv1alpha1.UIPlugin{
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
				Name: "perses-api-http",
			},
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
			Alertmanager: uiv1alpha1.AlertmanagerReference{
				Url: "https://alertmanager.open-cluster-management-observability.svc:9095",
			},
			ThanosQuerier: uiv1alpha1.ThanosQuerierReference{
				Url: "https://rbac-query-proxy.open-cluster-management-observability.svc:8443",
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
			ThanosQuerier: uiv1alpha1.ThanosQuerierReference{
				Url: "https://rbac-query-proxy.open-cluster-management-observability.svc:8443",
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
			Alertmanager: uiv1alpha1.AlertmanagerReference{
				Url: "https://alertmanager.open-cluster-management-observability.svc:9095",
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

var acmVersion = "v2.11"
var features = []string{}

func getPluginInfo(plugin *uiv1alpha1.UIPlugin, features []string) (*UIPluginInfo, error) {
	return createMonitoringPluginInfo(plugin, namespace, name, image, features, acmVersion)
}

func TestCreateMonitoringPluginInfo(t *testing.T) {
	t.Run("Test createMonitoringPluginInfo with all monitoring configurations", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigAll, features)
		alertmanagerFound, thanosFound, persesFound := containsProxy(pluginInfo)

		assert.Assert(t, alertmanagerFound == true)
		assert.Assert(t, thanosFound == true)
		assert.Assert(t, persesFound == true)
		assert.Assert(t, error == nil)
	})

	t.Run("Test createMonitoringPluginInfo with AMC configuration only", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigACM, features)
		alertmanagerFound, thanosFound, persesFound := containsProxy(pluginInfo)

		assert.Assert(t, alertmanagerFound == true)
		assert.Assert(t, thanosFound == true)
		assert.Assert(t, persesFound == false)
		assert.Assert(t, error == nil)

	})

	t.Run("Test createMonitoringPluginInfo with Perses configuration only", func(t *testing.T) {
		pluginInfo, error := getPluginInfo(pluginConfigPerses, features)
		alertmanagerFound, thanosFound, persesFound := containsProxy(pluginInfo)

		assert.Assert(t, error == nil)
		assert.Assert(t, alertmanagerFound == false)
		assert.Assert(t, thanosFound == false)
		assert.Assert(t, persesFound == true)
	})

	t.Run("Test createMonitoringPluginInfo with missing URLs from thanos and alertmanager", func(t *testing.T) {
		// should not error because "perses-dashboards" feature enabled and persesName and persesNamespace are included in UIPlugin
		pluginInfo, error := getPluginInfo(pluginConfigPerses, features)
		alertmanagerFound, thanosFound, persesFound := containsProxy(pluginInfo)

		assert.Assert(t, error == nil)
		assert.Assert(t, alertmanagerFound == false)
		assert.Assert(t, thanosFound == false)
		assert.Assert(t, persesFound == true)
	})

	t.Run("Test createMonitoringPluginInfo with missing URL from thanos", func(t *testing.T) {
		errorMessage := AcmErrorMsg + PersesErrorMsg + ThanosEmptyMsg + PersesNameEmptyMsg + PersesNamespaceEmptyMsg

		// this should throw an error because thanosQuerier.URL is not set
		pluginInfo, error := getPluginInfo(pluginConfigAlertmanager, features)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
		assert.Equal(t, error.Error(), errorMessage)
	})

	t.Run("Test createMonitoringPluginInfo with missing URL from alertmanager ", func(t *testing.T) {
		errorMessage := AcmErrorMsg + PersesErrorMsg + AlertmanagerEmptyMsg + PersesNameEmptyMsg + PersesNamespaceEmptyMsg

		// this should throw an error because alertManager.URL is not set
		pluginInfo, error := getPluginInfo(pluginConfigThanos, features)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
		assert.Equal(t, error.Error(), errorMessage)
	})

	t.Run("Test createMonitoringPluginInfo with missing persesName ", func(t *testing.T) {
		errorMessage := AcmErrorMsg + PersesErrorMsg + AlertmanagerEmptyMsg + ThanosEmptyMsg + PersesNamespaceEmptyMsg

		// this should throw an error because persesName is not set
		pluginInfo, error := getPluginInfo(pluginConfigPersesNameSpace, features)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
		assert.Equal(t, error.Error(), errorMessage)
	})

	t.Run("Test createMonitoringPluginInfo with missing persesNamespace ", func(t *testing.T) {
		errorMessage := AcmErrorMsg + PersesErrorMsg + AlertmanagerEmptyMsg + ThanosEmptyMsg + PersesNameEmptyMsg

		// this should throw an error because persesNamespace is not set
		pluginInfo, error := getPluginInfo(pluginConfigPersesName, features)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
		assert.Equal(t, error.Error(), errorMessage)
	})

	t.Run("Test createMonitoringPluginInfo with malform UIPlugin custom resource", func(t *testing.T) {
		errorMessage := AcmErrorMsg + PersesErrorMsg + AlertmanagerEmptyMsg + ThanosEmptyMsg + PersesNameEmptyMsg + PersesNamespaceEmptyMsg

		// this should throw an error because UIPlugin doesn't include alertmanager, thanos, or perses
		pluginInfo, error := getPluginInfo(pluginMalformed, features)
		assert.Assert(t, pluginInfo == nil)
		assert.Assert(t, error != nil)
		assert.Equal(t, error.Error(), errorMessage)
	})
}
