package uiplugin

import (
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func createDashboardsPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string) (*UIPluginInfo, error) {
	pluginName := "observability-ui-" + name
	readerRoleName := plugin.Name + "-datasource-reader"
	datasourcesNamespace := "openshift-config-managed"

	return &UIPluginInfo{
		Image:             image,
		Name:              pluginName,
		ConsoleName:       pluginTypeToConsoleName[plugin.Spec.Type],
		DisplayName:       "Console Enhanced Dashboards",
		ResourceNamespace: namespace,
		LegacyProxies: []osv1alpha1.ConsolePluginProxy{
			{
				Type:      osv1alpha1.ProxyTypeService,
				Alias:     "backend",
				Authorize: true,
				Service: osv1alpha1.ConsolePluginProxyServiceConfig{
					Name:      pluginName,
					Namespace: namespace,
					Port:      port,
				},
			},
		},
		Proxies: []osv1.ConsolePluginProxy{
			{
				Alias:         "backend",
				Authorization: "UserToken",
				Endpoint: osv1.ConsolePluginProxyEndpoint{
					Type: osv1.ProxyTypeService,
					Service: &osv1.ConsolePluginProxyServiceConfig{
						Name:      pluginName,
						Namespace: namespace,
						Port:      port,
					},
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
				Name:      pluginName + "-rolebinding",
				Namespace: datasourcesNamespace,
			},
			Subjects: []rbacv1.Subject{
				{
					APIGroup:  corev1.SchemeGroupVersion.Group,
					Kind:      "ServiceAccount",
					Name:      pluginName + serviceAccountSuffix,
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Kind:     "Role",
				Name:     readerRoleName,
			},
		},
	}, nil
}
