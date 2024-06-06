package uiplugin

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func TestLookupImageAndFeatures(t *testing.T) {
	tt := []struct {
		pluginType     uiv1alpha1.UIPluginType
		clusterVersion string
		expectedKey    string
		expectedErr    error
	}{
		{
			pluginType:     uiv1alpha1.TypeDashboards,
			clusterVersion: "4.10",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("dynamic plugins not supported before 4.11"),
		},
		{
			pluginType:     uiv1alpha1.TypeDashboards,
			clusterVersion: "4.11",
			expectedKey:    "ui-dashboards",
			expectedErr:    nil,
		},
		{
			pluginType:     uiv1alpha1.TypeDashboards,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-dashboards",
			expectedErr:    nil,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "4.11",
			expectedKey:    "ui-logging",
			expectedErr:    nil,
		},
		{
			pluginType: uiv1alpha1.TypeTroubleshootingPanel,
			// This plugin requires changes made in the monitoring-plugin for Openshift 4.16
			// to render the "Troubleshooting Panel" button on the alert details page.
			clusterVersion: "4.15",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("no compatible image found for plugin type %s and cluster version %s", uiv1alpha1.TypeTroubleshootingPanel, "v4.15"),
		},
		{
			pluginType:     uiv1alpha1.TypeTroubleshootingPanel,
			clusterVersion: "4.16",
			expectedKey:    "ui-troubleshooting-panel",
			expectedErr:    nil,
		},
		{
			pluginType:     uiv1alpha1.TypeTroubleshootingPanel,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-troubleshooting-panel",
			expectedErr:    nil,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.10",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("dynamic plugins not supported before 4.11"),
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.11",
			expectedKey:    "ui-distributed-tracing",
			expectedErr:    nil,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-distributed-tracing",
			expectedErr:    nil,
		},
		{
			pluginType:     "non-existent-plugin",
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("no compatible image found for plugin type non-existent-plugin and cluster version v4.24.0-0.nightly-2024-03-11-200348"),
		},
	}

	for _, tc := range tt {
		info, err := lookupImageAndFeatures(tc.pluginType, tc.clusterVersion)

		assert.Equal(t, tc.expectedKey, info.ImageKey)

		if tc.expectedErr != nil {
			assert.Error(t, err, tc.expectedErr.Error())
		} else {
			assert.NilError(t, err)
		}
	}
}
