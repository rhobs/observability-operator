package observability_ui_plugin

import (
	"testing"

	obsui "github.com/rhobs/observability-operator/pkg/apis/observability-ui/v1alpha1"

	"gotest.tools/v3/assert"
)

func TestCompatibilityMatrixSpec(t *testing.T) {
	tt := []struct {
		pluginType     obsui.UIPluginType
		clusterVersion string
		expectedKey    string
	}{
		{
			pluginType:     obsui.TypeDashboards,
			clusterVersion: "4.10",
			expectedKey:    "",
		},
		{
			pluginType:     obsui.TypeDashboards,
			clusterVersion: "4.11",
			expectedKey:    "ui-dashboards",
		},
		{
			pluginType:     obsui.TypeDashboards,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-dashboards",
		},
		{
			pluginType:     "non-existent-plugin",
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "",
		},
	}

	for _, tc := range tt {
		actualKey := getImageKeyForPluginType(tc.pluginType, tc.clusterVersion)
		assert.DeepEqual(t, tc.expectedKey, actualKey)
	}
}
