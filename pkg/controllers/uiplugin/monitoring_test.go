package uiplugin

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"gotest.tools/v3/assert"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var plugin = &uiv1alpha1.UIPlugin{
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
			PersesDashboards: uiv1alpha1.PersesDashboardsReference{
				Url: "https://perses-dashboards.svc:8443",
			},
		},
	},
}
var namespace = "openshift-operators"
var name = "monitoring"
var image = "quay.io/monitoring-foo-test:123"
var features = []string{"perses-dashboards", "acm-alerting"}

func TestCreateMonitoringPluginInfo(t *testing.T) {
	t.Run("Test createMontiroingPluginInfo", func(t *testing.T) {
		pluginInfo, _ := createMonitoringPluginInfo(plugin, namespace, name, image, features)

		containsPersesProxy := func() bool {
			for _, proxy := range pluginInfo.Proxies {
				if proxy.Alias == "perses-dashboards-proxy" {
					return true
				}
			}
			return false
		}

		assert.Assert(t, containsPersesProxy() == true)

		// JZ TO REMOVE -- for testing only to output pluginInfo object
		// prettyJSON, err := json.MarshalIndent(pluginInfo, "", "  ")
		// if err != nil {
		// 	log.Fatalf("Error pretty printing JSON: %v", err)
		// }
		// if error != nil {
		// 	log.Fatalf("Error pretty printing JSON: %v", err)
		// }
		// fmt.Println(string(prettyJSON))
		// assert.Equal(t, pluginInfo, true)
	})
}
