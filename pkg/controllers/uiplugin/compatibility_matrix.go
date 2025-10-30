package uiplugin

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type SupportLevel string

var (
	DevPreview          SupportLevel = "DevPreview"
	TechPreview         SupportLevel = "TechPreview"
	GeneralAvailability SupportLevel = "GeneralAvailability"
	Experimental_SSA    SupportLevel = "Experimental-SSA"
)

type CompatibilityEntry struct {
	PluginType uiv1alpha1.UIPluginType
	// Minimal OpenShift version supporting this plugin (inclusive).
	MinClusterVersion string
	// Maximal OpenShift version supporting this plugin (exclusive).
	MaxClusterVersion string
	ImageKey          string
	SupportLevel      SupportLevel
	Features          []string
}

type ListFunction func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error

var compatibilityMatrix = []CompatibilityEntry{
	{
		PluginType:        uiv1alpha1.TypeDashboards,
		MinClusterVersion: "v4.11",
		MaxClusterVersion: "",
		ImageKey:          "ui-dashboards",
		Features:          []string{},
		SupportLevel:      DevPreview,
	},
	{
		PluginType: uiv1alpha1.TypeTroubleshootingPanel,
		// This plugin requires the monitoring-plugin from OpenShift 4.16 (at
		// least) to render the "Troubleshooting Panel" button on the alert
		// details page.
		MinClusterVersion: "v4.16",
		MaxClusterVersion: "",
		ImageKey:          "ui-troubleshooting-panel",
		SupportLevel:      GeneralAvailability,
		Features:          []string{},
	},
	{
		PluginType:        uiv1alpha1.TypeDistributedTracing,
		MinClusterVersion: "v4.11",
		MaxClusterVersion: "v4.15",
		ImageKey:          "ui-distributed-tracing-pf4",
		SupportLevel:      GeneralAvailability,
		Features:          []string{},
	},
	{
		PluginType:        uiv1alpha1.TypeDistributedTracing,
		MinClusterVersion: "v4.15",
		MaxClusterVersion: "v4.19",
		ImageKey:          "ui-distributed-tracing-pf5",
		SupportLevel:      GeneralAvailability,
		Features:          []string{},
	},
	{
		PluginType:        uiv1alpha1.TypeDistributedTracing,
		MinClusterVersion: "v4.19",
		MaxClusterVersion: "",
		ImageKey:          "ui-distributed-tracing",
		SupportLevel:      GeneralAvailability,
		Features:          []string{},
	},
	{
		PluginType:        uiv1alpha1.TypeLogging,
		MinClusterVersion: "v4.11",
		MaxClusterVersion: "v4.12",
		ImageKey:          "ui-logging-pf4",
		SupportLevel:      GeneralAvailability,
		Features:          []string{},
	},
	{
		PluginType:        uiv1alpha1.TypeLogging,
		MinClusterVersion: "v4.12",
		MaxClusterVersion: "v4.13",
		ImageKey:          "ui-logging-pf4",
		SupportLevel:      GeneralAvailability,
		Features: []string{
			"dev-console",
		},
	},
	{
		PluginType:        uiv1alpha1.TypeLogging,
		MinClusterVersion: "v4.13",
		MaxClusterVersion: "v4.14",
		ImageKey:          "ui-logging-pf4",
		SupportLevel:      GeneralAvailability,
		Features: []string{
			"dev-console",
			"alerts",
		},
	},
	{
		PluginType:        uiv1alpha1.TypeLogging,
		MinClusterVersion: "v4.14",
		MaxClusterVersion: "v4.15",
		ImageKey:          "ui-logging-pf4",
		SupportLevel:      GeneralAvailability,
		Features: []string{
			"dev-console",
			"alerts",
			"dev-alerts",
		},
	},
	{
		PluginType:        uiv1alpha1.TypeLogging,
		MinClusterVersion: "v4.15",
		MaxClusterVersion: "",
		ImageKey:          "ui-logging",
		SupportLevel:      GeneralAvailability,
		Features: []string{
			"dev-console",
			"alerts",
			"dev-alerts",
		},
	},
	{
		PluginType:        uiv1alpha1.TypeMonitoring,
		MinClusterVersion: "v4.15",
		MaxClusterVersion: "v4.19",
		ImageKey:          "ui-monitoring-pf5",
		SupportLevel:      TechPreview,
		// feature flags for montioring are dynamically injected
		// based on the cluster version and and UIPlugin CR configurations
		Features: []string{},
	},
	{
		PluginType:        uiv1alpha1.TypeMonitoring,
		MinClusterVersion: "v4.19",
		MaxClusterVersion: "",
		ImageKey:          "ui-monitoring",
		SupportLevel:      GeneralAvailability,
		// feature flags for montioring are dynamically injected
		// based on the cluster version and and UIPlugin CR configurations
		Features: []string{},
	},
}

func lookupImageAndFeatures(pluginType uiv1alpha1.UIPluginType, clusterVersion string) (CompatibilityEntry, error) {
	if !strings.HasPrefix(clusterVersion, "v") {
		clusterVersion = "v" + clusterVersion
	}

	// No console plugins are supported before 4.11
	if semver.Compare(clusterVersion, "v4.11") < 0 {
		return CompatibilityEntry{}, fmt.Errorf("dynamic plugins not supported before 4.11")
	}

	for _, entry := range compatibilityMatrix {
		if entry.PluginType != pluginType {
			continue
		}

		matchedVersion, err := compareClusterVersion(entry, clusterVersion, pluginType)

		if err == nil {
			return matchedVersion, nil
		}
	}
	return CompatibilityEntry{}, fmt.Errorf("plugin %q: no compatible image found for cluster version %q", pluginType, clusterVersion)
}

func compareClusterVersion(entry CompatibilityEntry, clusterVersion string, pluginType uiv1alpha1.UIPluginType) (CompatibilityEntry, error) {
	canonicalMinClusterVersion := fmt.Sprintf("%s-0", semver.Canonical(entry.MinClusterVersion))
	canonicalMaxClusterVersion := fmt.Sprintf("%s-0", semver.Canonical(entry.MaxClusterVersion))

	if entry.MaxClusterVersion == "" && semver.Compare(clusterVersion, canonicalMinClusterVersion) >= 0 {
		return entry, nil
	}

	if semver.Compare(clusterVersion, canonicalMinClusterVersion) >= 0 && semver.Compare(clusterVersion, canonicalMaxClusterVersion) < 0 {
		return entry, nil
	}
	return CompatibilityEntry{}, fmt.Errorf("plugin %q: no compatible image found for cluster version %q", pluginType, clusterVersion)
}
