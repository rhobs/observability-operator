package observability_ui_plugin

import (
	"fmt"
	"testing"

	obsui "github.com/rhobs/observability-operator/pkg/apis/observability-ui/v1alpha1"

	"gotest.tools/v3/assert"
)

func TestCompatibilityMatrixSpec(t *testing.T) {
	tt := []struct {
		pluginType     obsui.UIPluginType
		clusterVersion string
		expectedKey    string
		expectedErr    error
	}{
		{
			pluginType:     obsui.TypeDashboards,
			clusterVersion: "4.10",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("dynamic pluings not supported before 4.11"),
		},
		{
			pluginType:     obsui.TypeDashboards,
			clusterVersion: "4.11",
			expectedKey:    "ui-dashboards",
			expectedErr:    nil,
		},
		{
			pluginType:     obsui.TypeDashboards,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-dashboards",
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
		actualKey, err := getImageKeyForPluginType(tc.pluginType, tc.clusterVersion)
		assert.Equal(t, tc.expectedKey, actualKey)

		if tc.expectedErr != nil {
			assert.Error(t, err, tc.expectedErr.Error())
		} else {
			assert.NilError(t, err)
		}
	}
}
