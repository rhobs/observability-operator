package uiplugin

import (
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func createDashboardsPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string) (*UIPluginInfo, error) {
	pluginName := "observability-ui-" + name

	return &UIPluginInfo{
		Image:       image,
		Name:        pluginName,
		ConsoleName: pluginTypeToConsoleName[plugin.Spec.Type],
		DisplayName: "Console Enhanced Dashboards",
		Proxies: []PluginProxy{
			{
				Alias:            "backend",
				ServiceName:      pluginName,
				ServiceNamespace: namespace,
				ServicePort:      port,
				Authorize:        true,
			},
		},
	}, nil
}
