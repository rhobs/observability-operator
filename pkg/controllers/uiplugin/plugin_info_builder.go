package uiplugin

import (
	"context"
	"fmt"

	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type UIPluginInfo struct {
	Image               string
	Korrel8rImage       string
	LokiServiceNames    map[string]string
	Name                string
	ConsoleName         string
	DisplayName         string
	ExtraArgs           []string
	Proxies             []osv1alpha1.ConsolePluginProxy
	Role                *rbacv1.Role
	RoleBinding         *rbacv1.RoleBinding
	ClusterRoles        []*rbacv1.ClusterRole
	ClusterRoleBindings []*rbacv1.ClusterRoleBinding
	ConfigMap           *corev1.ConfigMap
	ResourceNamespace   string
}

func PluginInfoBuilder(ctx context.Context, k client.Client, plugin *uiv1alpha1.UIPlugin, pluginConf UIPluginsConfiguration, clusterVersion string) (*UIPluginInfo, error) {
	compatibilityInfo, err := lookupImageAndFeatures(plugin.Spec.Type, clusterVersion)
	if err != nil {
		return nil, err
	}

	image := pluginConf.Images[compatibilityInfo.ImageKey]
	if image == "" {
		return nil, fmt.Errorf("no image provided for plugin type %s with key %s", plugin.Spec.Type, compatibilityInfo.ImageKey)
	}

	namespace := pluginConf.ResourcesNamespace
	switch plugin.Spec.Type {
	case uiv1alpha1.TypeDashboards:
		name := "observability-ui-" + plugin.Name
		readerRoleName := plugin.Name + "-datasource-reader"
		datasourcesNamespace := "openshift-config-managed"

		pluginInfo := &UIPluginInfo{
			Image:             image,
			Name:              name,
			ConsoleName:       "console-dashboards-plugin",
			DisplayName:       "Console Enhanced Dashboards",
			ResourceNamespace: namespace,
			Proxies: []osv1alpha1.ConsolePluginProxy{
				{
					Type:      osv1alpha1.ProxyTypeService,
					Alias:     "backend",
					Authorize: true,
					Service: osv1alpha1.ConsolePluginProxyServiceConfig{
						Name:      name,
						Namespace: namespace,
						Port:      9443,
					},
				},
			},
			Role: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.String(),
					Kind:       "Role",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      readerRoleName,
					Namespace: datasourcesNamespace,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
						Verbs:     []string{"get", "list", "watch"},
					},
				},
			},
			RoleBinding: &rbacv1.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.String(),
					Kind:       "RoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name + "-rolebinding",
					Namespace: datasourcesNamespace,
				},
				Subjects: []rbacv1.Subject{
					{
						APIGroup:  corev1.SchemeGroupVersion.Group,
						Kind:      "ServiceAccount",
						Name:      name + "-sa",
						Namespace: namespace,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.SchemeGroupVersion.Group,
					Kind:     "Role",
					Name:     readerRoleName,
				},
			},
		}

		return pluginInfo, nil

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

		return pluginInfo, nil

	case uiv1alpha1.TypeDistributedTracing:
		return createDistributedTracingPluginInfo(plugin, namespace, plugin.Name, image, []string{})

	case uiv1alpha1.TypeLogging:
		return createLoggingPluginInfo(plugin, namespace, plugin.Name, image, compatibilityInfo.Features)
	}

	return nil, fmt.Errorf("plugin type not supported: %s", plugin.Spec.Type)
}
