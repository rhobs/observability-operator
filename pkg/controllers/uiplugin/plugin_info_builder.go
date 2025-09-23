package uiplugin

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type UIPluginInfo struct {
	Image                      string
	Korrel8rImage              string
	HealthAnalyzerImage        string
	LokiServiceNames           map[string]string
	TempoServiceNames          map[string]string
	Name                       string
	ConsoleName                string
	DisplayName                string
	ExtraArgs                  []string
	LegacyProxies              []osv1alpha1.ConsolePluginProxy
	Proxies                    []osv1.ConsolePluginProxy
	Role                       *rbacv1.Role
	RoleBinding                *rbacv1.RoleBinding
	ClusterRoles               []*rbacv1.ClusterRole
	ClusterRoleBindings        []*rbacv1.ClusterRoleBinding
	ConfigMap                  *corev1.ConfigMap
	ResourceNamespace          string
	PersesImage                string
	AreMonitoringFeatsDisabled bool
}

var pluginTypeToConsoleName = map[uiv1alpha1.UIPluginType]string{
	uiv1alpha1.TypeDashboards:           "console-dashboards-plugin",
	uiv1alpha1.TypeTroubleshootingPanel: "troubleshooting-panel-console-plugin",
	uiv1alpha1.TypeDistributedTracing:   "distributed-tracing-console-plugin",
	uiv1alpha1.TypeLogging:              "logging-view-plugin",
	uiv1alpha1.TypeMonitoring:           "monitoring-console-plugin",
}

func PluginInfoBuilder(ctx context.Context, k client.Client, dk dynamic.Interface, plugin *uiv1alpha1.UIPlugin, pluginConf UIPluginsConfiguration, compatibilityInfo CompatibilityEntry, clusterVersion string, logger logr.Logger) (*UIPluginInfo, error) {
	image := pluginConf.Images[compatibilityInfo.ImageKey]
	if image == "" {
		return nil, fmt.Errorf("no image provided for plugin type %s with key %s", plugin.Spec.Type, compatibilityInfo.ImageKey)
	}

	namespace := pluginConf.ResourcesNamespace
	switch plugin.Spec.Type {
	case uiv1alpha1.TypeDashboards:
		return createDashboardsPluginInfo(plugin, namespace, plugin.Name, image)

	case uiv1alpha1.TypeTroubleshootingPanel:
		pluginInfo, err := createTroubleshootingPanelPluginInfo(plugin, namespace, plugin.Name, image, []string{})
		if err != nil {
			return nil, err
		}

		pluginInfo.Korrel8rImage = pluginConf.Images["korrel8r"]
		pluginInfo.LokiServiceNames[OpenshiftLoggingNs], err = getLokiServiceName(ctx, k, OpenshiftLoggingNs)
		if err != nil {
			return nil, err
		}

		pluginInfo.LokiServiceNames[OpenshiftNetobservNs], err = getLokiServiceName(ctx, k, OpenshiftNetobservNs)
		if err != nil {
			return nil, err
		}

		pluginInfo.TempoServiceNames[OpenshiftTracingNs], err = getTempoServiceName(ctx, k, OpenshiftTracingNs)
		if err != nil {
			return nil, err
		}

		return pluginInfo, nil

	case uiv1alpha1.TypeDistributedTracing:
		return createDistributedTracingPluginInfo(plugin, namespace, plugin.Name, image, []string{})

	case uiv1alpha1.TypeLogging:
		return createLoggingPluginInfo(plugin, namespace, plugin.Name, image, compatibilityInfo.Features, ctx, dk, logger)

	case uiv1alpha1.TypeMonitoring:
		return createMonitoringPluginInfo(plugin, namespace, plugin.Name, image, compatibilityInfo.Features, clusterVersion, pluginConf.Images["health-analyzer"], pluginConf.Images["perses"])
	}

	return nil, fmt.Errorf("plugin type not supported: %s", plugin.Spec.Type)
}
