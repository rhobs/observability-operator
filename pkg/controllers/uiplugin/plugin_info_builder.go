package uiplugin

import (
	"context"
	"io/fs"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type UIPluginInfo struct {
	Scheme                     *runtime.Scheme
	ConfigFS                   fs.FS
	Images                     map[string]string
	Namespace                  string
	ClusterVersion             string
	LokiStackName              string
	LokiStackNamespace         string
	TLSCiphers                 []string
	TLSMinVersion              string
	LokiServiceNames           map[string]string
	TempoServiceNames          map[string]string
	TracingInstaller           *obsv1alpha1.ObservabilityInstaller
	Image                      string
	Korrel8rImage              string
	HealthAnalyzerImage        string
	TracingTenant              string
	Name                       string
	ConsoleName                string
	DisplayName                string
	ExtraArgs                  []string
	Proxies                    []PluginProxy
	ConfigMap                  *corev1.ConfigMap
	PersesImage                string
	AreMonitoringFeatsDisabled bool
}

// ApplyTLSProfile fills the TLSMinVersion and TLSCiphers fields from an
// OpenShift TLS security profile. OpenSSL cipher names from the profile are
// converted to the IANA names expected by the plugin server's
// -tls-cipher-suites flag.
func (c *UIPluginInfo) ApplyTLSProfile(profile configv1.TLSProfileSpec) {
	c.TLSMinVersion = string(profile.MinTLSVersion)
	c.TLSCiphers = libgocrypto.OpenSSLToIANACipherSuites(profile.Ciphers)
}

var pluginTypeToConsoleName = map[uiv1alpha1.UIPluginType]string{
	uiv1alpha1.TypeDashboards:           "console-dashboards-plugin", //nolint:staticcheck // needed for deregistration of deprecated type
	uiv1alpha1.TypeTroubleshootingPanel: "troubleshooting-panel-console-plugin",
	uiv1alpha1.TypeDistributedTracing:   "distributed-tracing-console-plugin",
	uiv1alpha1.TypeLogging:              "logging-view-plugin",
	uiv1alpha1.TypeMonitoring:           "monitoring-console-plugin",
}

func ConsoleNameForType(pluginType uiv1alpha1.UIPluginType) string {
	return pluginTypeToConsoleName[pluginType]
}

// PluginInfoBuilder constructs a UIPluginInfo with configuration fields
// populated from the controller state, including per-plugin-type cluster
// lookups (Tempo/Loki service names, LokiStack).
func PluginInfoBuilder(ctx context.Context, k client.Client, dk dynamic.Interface, scheme *runtime.Scheme, configFS fs.FS, plugin *uiv1alpha1.UIPlugin, pluginConf UIPluginsConfiguration, clusterVersion string, logger logr.Logger) (UIPluginInfo, error) {
	info := UIPluginInfo{
		Scheme:         scheme,
		ConfigFS:       configFS,
		Images:         pluginConf.Images,
		Namespace:      pluginConf.ResourcesNamespace,
		ClusterVersion: clusterVersion,
	}
	info.ApplyTLSProfile(pluginConf.TLSProfile)

	switch plugin.Spec.Type {
	case uiv1alpha1.TypeDistributedTracing:
		info.TempoServiceNames = make(map[string]string)
		if name, _ := getTempoServiceName(ctx, k, OpenshiftTracingNs); name != "" {
			info.TempoServiceNames[OpenshiftTracingNs] = name
		}

	case uiv1alpha1.TypeTroubleshootingPanel:
		info.LokiServiceNames = make(map[string]string)
		info.LokiServiceNames[OpenshiftLoggingNs], _ = getLokiServiceName(ctx, k, OpenshiftLoggingNs)
		info.LokiServiceNames[OpenshiftNetobservNs], _ = getLokiServiceName(ctx, k, OpenshiftNetobservNs)
		info.TempoServiceNames = make(map[string]string)
		if name, _ := getTempoServiceName(ctx, k, OpenshiftTracingNs); name != "" {
			info.TempoServiceNames[OpenshiftTracingNs] = name
		}

	case uiv1alpha1.TypeLogging:
		lokiStack, err := getLokiStack(plugin, ctx, dk, logger)
		if err != nil {
			return info, err
		}
		info.LokiStackName = lokiStack.Name
		info.LokiStackNamespace = lokiStack.Namespace
	}

	return info, nil
}
