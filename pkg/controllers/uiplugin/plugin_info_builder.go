package uiplugin

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

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
	uiv1alpha1.TypeDashboards:           "console-dashboards-plugin",
	uiv1alpha1.TypeTroubleshootingPanel: "troubleshooting-panel-console-plugin",
	uiv1alpha1.TypeDistributedTracing:   "distributed-tracing-console-plugin",
	uiv1alpha1.TypeLogging:              "logging-view-plugin",
	uiv1alpha1.TypeMonitoring:           "monitoring-console-plugin",
}

func ConsoleNameForType(pluginType uiv1alpha1.UIPluginType) string {
	return pluginTypeToConsoleName[pluginType]
}
