package observability_ui_plugin

import (
	"strings"

	"golang.org/x/mod/semver"

	obsui "github.com/rhobs/observability-operator/pkg/apis/observability-ui/v1alpha1"
)

type CompatibilityEntry struct {
	PluginType        obsui.UIPluginType
	MinClusterVersion string
	MaxClusterVersion string
	ImageKey          string
	Features          []string
}

var compatibilityMatrix = []CompatibilityEntry{
	{
		PluginType:        obsui.TypeDashboards,
		MinClusterVersion: "v4.11",
		MaxClusterVersion: "",
		ImageKey:          "ui-dashboards",
		Features:          []string{},
	},
}

func getImageKeyForPluginType(pluginType obsui.UIPluginType, clusterVersion string) string {
	if !strings.HasPrefix(clusterVersion, "v") {
		clusterVersion = "v" + clusterVersion
	}

	// No console plugins are supported before 4.11
	if semver.Compare(clusterVersion, "v4.11") < 0 {
		return ""
	}

	for _, entry := range compatibilityMatrix {
		if entry.PluginType == pluginType {
			if entry.MaxClusterVersion == "" && semver.Compare(clusterVersion, entry.MinClusterVersion) >= 0 {
				return entry.ImageKey
			}

			if semver.Compare(clusterVersion, entry.MinClusterVersion) >= 0 && semver.Compare(clusterVersion, entry.MaxClusterVersion) < 0 {
				return entry.ImageKey
			}
		}
	}

	return ""
}
