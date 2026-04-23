package uiplugin

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
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
	TLSMinVersion              string
	TLSCiphers                 []string
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

	var pluginInfo *UIPluginInfo
	var err error

	switch plugin.Spec.Type {
	case uiv1alpha1.TypeDashboards:
		pluginInfo, err = createDashboardsPluginInfo(plugin, namespace, plugin.Name, image)
		if err != nil {
			return nil, err
		}

	case uiv1alpha1.TypeTroubleshootingPanel:
		pluginInfo, err = createTroubleshootingPanelPluginInfo(plugin, namespace, plugin.Name, image, []string{})
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

	case uiv1alpha1.TypeDistributedTracing:
		pluginInfo, err = createDistributedTracingPluginInfo(plugin, namespace, plugin.Name, image, []string{})
		if err != nil {
			return nil, err
		}

	case uiv1alpha1.TypeLogging:
		pluginInfo, err = createLoggingPluginInfo(plugin, namespace, plugin.Name, image, compatibilityInfo.Features, ctx, dk, logger, pluginConf.Images["korrel8r"])
		if err != nil {
			return nil, err
		}

	case uiv1alpha1.TypeMonitoring:
		pluginInfo, err = createMonitoringPluginInfo(plugin, namespace, plugin.Name, image, compatibilityInfo.Features, clusterVersion, pluginConf.Images["health-analyzer"], pluginConf.Images["perses"])
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("plugin type not supported: %s", plugin.Spec.Type)
	}

	if compatibilityInfo.SupportsTLSProfile {
		pluginInfo.TLSMinVersion = string(pluginConf.TLSProfile.MinTLSVersion)
		pluginInfo.TLSCiphers = libgocrypto.OpenSSLToIANACipherSuites(pluginConf.TLSProfile.Ciphers)
	} else {
		logger.Info("TLS profile not applied: plugin image does not support TLS profile flags",
			"plugin", plugin.Name,
			"pluginType", plugin.Spec.Type,
			"imageKey", compatibilityInfo.ImageKey)
	}

	return pluginInfo, err
}
