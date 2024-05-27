package uiplugin

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type CompatibilityEntry struct {
	PluginType        uiv1alpha1.UIPluginType
	MinClusterVersion string
	MaxClusterVersion string
	ImageKey          string
	Features          []string
}

var compatibilityMatrix = []CompatibilityEntry{
	{
		PluginType:        uiv1alpha1.TypeDashboards,
		MinClusterVersion: "v4.11",
		MaxClusterVersion: "",
		ImageKey:          "ui-dashboards",
		Features:          []string{},
	},
}

func getImageKeyForPluginType(pluginType uiv1alpha1.UIPluginType, clusterVersion string) (string, error) {
	if !strings.HasPrefix(clusterVersion, "v") {
		clusterVersion = "v" + clusterVersion
	}

	// No console plugins are supported before 4.11
	if semver.Compare(clusterVersion, "v4.11") < 0 {
		return "", fmt.Errorf("dynamic plugins not supported before 4.11")
	}

	for _, entry := range compatibilityMatrix {
		if entry.PluginType == pluginType {
			if entry.MaxClusterVersion == "" && semver.Compare(clusterVersion, entry.MinClusterVersion) >= 0 {
				return entry.ImageKey, nil
			}

			if semver.Compare(clusterVersion, entry.MinClusterVersion) >= 0 && semver.Compare(clusterVersion, entry.MaxClusterVersion) <= 0 {
				return entry.ImageKey, nil
			}
		}
	}

	return "", fmt.Errorf("no compatible image found for plugin type %s and cluster version %s", pluginType, clusterVersion)
}
