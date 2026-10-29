package uiplugin

import (
	"fmt"
	"testing"

	"golang.org/x/mod/semver"
	"gotest.tools/v3/assert"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

// Ensure that all versions parse correctly.
func TestCompatibilityMatrixVersions(t *testing.T) {
	for _, v := range compatibilityMatrix {
		t.Run(string(v.PluginType), func(t *testing.T) {
			// MinClusterVersion is always required.
			assert.Assert(t, v.MinClusterVersion != "")
			assert.Equal(t, semver.IsValid(v.MinClusterVersion), true)

			if v.MaxClusterVersion != "" {
				assert.Equal(t, semver.IsValid(v.MaxClusterVersion), true)
			}

			if v.MinClusterVersion != "" && v.MaxClusterVersion != "" {
				assert.Equal(t, semver.Compare(v.MinClusterVersion, v.MaxClusterVersion), -1)
			}
		})
	}
}

// Ensure that there's only one empty max version per plugin.
func TestCompatibilityMatrixMaxVersions(t *testing.T) {
	cm := map[uiv1alpha1.UIPluginType]struct{}{}
	for _, v := range compatibilityMatrix {
		if v.MaxClusterVersion != "" {
			continue
		}

		_, found := cm[v.PluginType]
		assert.Assert(t, !found, string(v.PluginType))
		cm[v.PluginType] = struct{}{}
	}
}

func TestLookupImageAndFeatures(t *testing.T) {
	for _, tc := range []struct {
		pluginType       uiv1alpha1.UIPluginType
		clusterVersion   string
		expectedKey      string
		expectedErr      error
		expectedFeatures []string
		supportLevel     SupportLevel
	}{
		// errors before 4.22
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "4.21",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("plugin %q: no compatible image found for cluster version %q", uiv1alpha1.TypeLogging, "v4.21"),
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeTroubleshootingPanel,
			clusterVersion: "4.21",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("plugin %q: no compatible image found for cluster version %q", uiv1alpha1.TypeTroubleshootingPanel, "v4.21"),
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.21",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("plugin %q: no compatible image found for cluster version %q", uiv1alpha1.TypeDistributedTracing, "v4.21"),
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeMonitoring,
			clusterVersion: "4.21",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("plugin %q: no compatible image found for cluster version %q", uiv1alpha1.TypeMonitoring, "v4.21"),
			supportLevel:   GeneralAvailability,
		},
		// Nightly versions
		{
			pluginType:       uiv1alpha1.TypeLogging,
			clusterVersion:   "v4.22.0-0.nightly-2024-06-06-064349",
			expectedKey:      "ui-logging",
			expectedErr:      nil,
			expectedFeatures: []string{},
			supportLevel:     GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeTroubleshootingPanel,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-troubleshooting-panel",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:       uiv1alpha1.TypeMonitoring,
			clusterVersion:   "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:      "ui-monitoring",
			expectedFeatures: []string{},
			expectedErr:      nil,
			supportLevel:     GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-distributed-tracing",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		// Errors
		{
			pluginType:     "non-existent-plugin",
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "",
			expectedErr:    fmt.Errorf(`plugin "non-existent-plugin": no compatible image found for cluster version "v4.24.0-0.nightly-2024-03-11-200348"`),
			supportLevel:   "",
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.10",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("dynamic plugins not supported before 4.11"),
			supportLevel:   "",
		},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.pluginType, tc.clusterVersion), func(t *testing.T) {
			info, err := lookupImageAndFeatures(tc.pluginType, tc.clusterVersion)

			if tc.expectedErr != nil {
				assert.Error(t, err, tc.expectedErr.Error())
				return
			}

			assert.NilError(t, err)

			t.Logf("%s == %s", tc.expectedKey, info.ImageKey)
			assert.Equal(t, tc.expectedKey, info.ImageKey)

			assert.Equal(t, tc.supportLevel, info.SupportLevel)

			if tc.expectedFeatures != nil {
				assert.DeepEqual(t, tc.expectedFeatures, info.Features)
			}
		})
	}
}
