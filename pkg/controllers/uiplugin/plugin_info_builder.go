package uiplugin

import (
	"context"
	"fmt"
	"maps"

	"github.com/go-logr/logr"
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
	Proxies                    []PluginProxy
	Role                       *rbacv1.Role
	RoleBinding                *rbacv1.RoleBinding
	ClusterRoles               []*rbacv1.ClusterRole
	ClusterRoleBindings        []*rbacv1.ClusterRoleBinding
	ConfigMap                  *corev1.ConfigMap
	ResourceNamespace          string
	PersesImage                string
	AreMonitoringFeatsDisabled bool
	TLSMinVersion              string
	TLSCiphers                 []string
}

var pluginTypeToConsoleName = map[uiv1alpha1.UIPluginType]string{
	// Deprecated: retained for console deregistration during cleanup of existing Dashboards CRs.
	//nolint:staticcheck
	uiv1alpha1.TypeDashboards:           "console-dashboards-plugin",
	uiv1alpha1.TypeTroubleshootingPanel: "troubleshooting-panel-console-plugin",
	uiv1alpha1.TypeDistributedTracing:   "distributed-tracing-console-plugin",
	uiv1alpha1.TypeLogging:              "logging-view-plugin",
	uiv1alpha1.TypeMonitoring:           "monitoring-console-plugin",
}

func ConsoleNameForType(pluginType uiv1alpha1.UIPluginType) string {
	return pluginTypeToConsoleName[pluginType]
}

func PluginInfoBuilder(ctx context.Context, k client.Client, dk dynamic.Interface, plugin *uiv1alpha1.UIPlugin, pluginConf UIPluginsConfiguration, clusterVersion string, logger logr.Logger) (*UIPluginInfo, error) {
	conf := UIPluginBuildConfig{
		Images:         pluginConf.Images,
		Namespace:      pluginConf.ResourcesNamespace,
		ClusterVersion: clusterVersion,
		DynamicClient:  dk,
	}
	conf.ApplyTLSProfile(pluginConf.TLSProfile)

	if plugin.Spec.Type == uiv1alpha1.TypeTroubleshootingPanel {
		conf.LokiServiceNames = map[string]string{}
		conf.TempoServiceNames = map[string]string{}

		var err error
		conf.LokiServiceNames[OpenshiftLoggingNs], err = getLokiServiceName(ctx, k, OpenshiftLoggingNs)
		if err != nil {
			return nil, err
		}
		conf.LokiServiceNames[OpenshiftNetobservNs], err = getLokiServiceName(ctx, k, OpenshiftNetobservNs)
		if err != nil {
			return nil, err
		}
		conf.TempoServiceNames[OpenshiftTracingNs], err = getTempoServiceName(ctx, k, OpenshiftTracingNs)
		if err != nil {
			return nil, err
		}
	}

	return buildPluginInfo(ctx, plugin, conf, logger)
}

func buildPluginInfo(ctx context.Context, plugin *uiv1alpha1.UIPlugin, conf UIPluginBuildConfig, logger logr.Logger) (*UIPluginInfo, error) {
	compatibilityInfo, err := lookupImageAndFeatures(plugin.Spec.Type, conf.ClusterVersion)
	if err != nil {
		return nil, err
	}

	image := conf.Images[compatibilityInfo.ImageKey]
	if image == "" {
		return nil, fmt.Errorf("no image provided for plugin type %s with key %s", plugin.Spec.Type, compatibilityInfo.ImageKey)
	}

	var pluginInfo *UIPluginInfo
	switch plugin.Spec.Type {
	case uiv1alpha1.TypeDistributedTracing:
		pluginInfo, err = createDistributedTracingPluginInfo(plugin, conf.Namespace, plugin.Name, image, compatibilityInfo.Features)

	case uiv1alpha1.TypeLogging:
		pluginInfo, err = createLoggingPluginInfo(plugin, conf.Namespace, plugin.Name, image, compatibilityInfo.Features, ctx, conf.DynamicClient, logger, conf.Images["korrel8r"])

	case uiv1alpha1.TypeTroubleshootingPanel:
		pluginInfo, err = createTroubleshootingPanelPluginInfo(plugin, conf.Namespace, plugin.Name, image, compatibilityInfo.Features, conf.ClusterVersion, logger)
		if err == nil {
			pluginInfo.Korrel8rImage = conf.Images["korrel8r"]
			pluginInfo.LokiServiceNames = maps.Clone(conf.LokiServiceNames)
			pluginInfo.TempoServiceNames = maps.Clone(conf.TempoServiceNames)
		}

	case uiv1alpha1.TypeMonitoring:
		pluginInfo, err = createMonitoringPluginInfo(plugin, conf.Namespace, plugin.Name, image, compatibilityInfo.Features, conf.ClusterVersion, conf.Images["health-analyzer"], conf.Images["perses"])

	default:
		return nil, fmt.Errorf("plugin type not supported: %s", plugin.Spec.Type)
	}
	if err != nil {
		return nil, err
	}

	pluginInfo.TLSMinVersion = conf.TLSMinVersion
	pluginInfo.TLSCiphers = conf.TLSCiphers
	return pluginInfo, nil
}
