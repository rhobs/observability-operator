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
		{
			pluginType:     uiv1alpha1.TypeDashboards,
			clusterVersion: "4.10",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("dynamic plugins not supported before 4.11"),
			supportLevel:   "",
		},
		{
			pluginType:     uiv1alpha1.TypeDashboards,
			clusterVersion: "4.11",
			expectedKey:    "ui-dashboards",
			expectedErr:    nil,
			supportLevel:   DevPreview,
		},
		{
			pluginType:     uiv1alpha1.TypeDashboards,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-dashboards",
			expectedErr:    nil,
			supportLevel:   DevPreview,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "4.13",
			expectedKey:    "ui-logging-pf4",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "v4.13.45",
			expectedKey:    "ui-logging-pf4",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "v4.14",
			expectedKey:    "ui-logging-pf4",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
				"dev-alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "v4.14.1",
			expectedKey:    "ui-logging-pf4",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
				"dev-alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "v4.15",
			expectedKey:    "ui-logging",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
				"dev-alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "v4.15.0-0.nightly-2024-06-06-064349",
			expectedKey:    "ui-logging",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
				"dev-alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "4.15.46",
			expectedKey:    "ui-logging",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
				"dev-alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "v4.16.9",
			expectedKey:    "ui-logging",
			expectedErr:    nil,
			expectedFeatures: []string{
				"dev-console",
				"alerts",
				"dev-alerts",
			},
			supportLevel: GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeLogging,
			clusterVersion: "4.11",
			expectedKey:    "ui-logging-pf4",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType: uiv1alpha1.TypeTroubleshootingPanel,
			// This plugin requires changes made in the monitoring-plugin for Openshift 4.16
			// to render the "Troubleshooting Panel" button on the alert details page.
			clusterVersion: "4.15",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("plugin %q: no compatible image found for cluster version %q", uiv1alpha1.TypeTroubleshootingPanel, "v4.15"),
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeTroubleshootingPanel,
			clusterVersion: "4.16",
			expectedKey:    "ui-troubleshooting-panel",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeTroubleshootingPanel,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-troubleshooting-panel",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.10",
			expectedKey:    "",
			expectedErr:    fmt.Errorf("dynamic plugins not supported before 4.11"),
			supportLevel:   "",
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.11",
			expectedKey:    "ui-distributed-tracing-pf4",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.14",
			expectedKey:    "ui-distributed-tracing-pf4",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.15",
			expectedKey:    "ui-distributed-tracing-pf5",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "v4.15.0-0.nightly-2024-06-06-064349",
			expectedKey:    "ui-distributed-tracing-pf5",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "v4.15.46",
			expectedKey:    "ui-distributed-tracing-pf5",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.18",
			expectedKey:    "ui-distributed-tracing-pf5",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.19",
			expectedKey:    "ui-distributed-tracing",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "ui-distributed-tracing",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeDistributedTracing,
			clusterVersion: "4.16.0-rc.3",
			expectedKey:    "ui-distributed-tracing-pf5",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     "non-existent-plugin",
			clusterVersion: "4.24.0-0.nightly-2024-03-11-200348",
			expectedKey:    "",
			expectedErr:    fmt.Errorf(`plugin "non-existent-plugin": no compatible image found for cluster version "v4.24.0-0.nightly-2024-03-11-200348"`),
			supportLevel:   "",
		},
		{
			pluginType:     uiv1alpha1.TypeTroubleshootingPanel,
			clusterVersion: "v4.16.0-0.nightly-2024-06-06-064349",
			expectedKey:    "ui-troubleshooting-panel",
			expectedErr:    nil,
			supportLevel:   GeneralAvailability,
		},
		{
			pluginType:     uiv1alpha1.TypeMonitoring,
			clusterVersion: "v4.14",
			expectedKey:    "ui-monitoring",
			expectedErr:    fmt.Errorf("plugin %q: no compatible image found for cluster version %q", uiv1alpha1.TypeMonitoring, "v4.14"),
			supportLevel:   TechPreview,
		},
		{
			pluginType:       uiv1alpha1.TypeMonitoring,
			clusterVersion:   "v4.15",
			expectedKey:      "ui-monitoring-pf5",
			expectedFeatures: []string{},
			expectedErr:      nil,
			supportLevel:     TechPreview,
		},
		{
			pluginType:       uiv1alpha1.TypeMonitoring,
			clusterVersion:   "v4.15.0-0.nightly-2024-06-06-064349",
			expectedKey:      "ui-monitoring-pf5",
			expectedFeatures: []string{},
			expectedErr:      nil,
			supportLevel:     TechPreview,
		},
		{
			pluginType:       uiv1alpha1.TypeMonitoring,
			clusterVersion:   "v4.19",
			expectedKey:      "ui-monitoring",
			expectedFeatures: []string{},
			expectedErr:      nil,
			supportLevel:     GeneralAvailability,
		},
		{
			pluginType:       uiv1alpha1.TypeMonitoring,
			clusterVersion:   "v4.19.0-0.nightly-2024-06-06-064349",
			expectedKey:      "ui-monitoring",
			expectedFeatures: []string{},
			expectedErr:      nil,
			supportLevel:     GeneralAvailability,
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
