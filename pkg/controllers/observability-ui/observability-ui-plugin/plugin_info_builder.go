package observability_ui_plugin

import (
	"fmt"

	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	obsui "github.com/rhobs/observability-operator/pkg/apis/observability-ui/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObservabilityUIPluginInfo struct {
	Image              string
	Name               string
	ConsoleName        string
	DisplayName        string
	Proxies            []osv1alpha1.ConsolePluginProxy
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
}

func PluginInfoBuilder(plugin *obsui.ObservabilityUIPlugin, pluginConf ObservabilityUIPluginsConfiguration, clusterVersion string) (*ObservabilityUIPluginInfo, error) {
	imageKey, err := getImageKeyForPluginType(plugin.Spec.Type, clusterVersion)

	if err != nil {
		return nil, err
	}

	image := pluginConf.Images[imageKey]

	if image == "" {
		return nil, fmt.Errorf("no image provided for plugin type %s with key %s", plugin.Spec.Type, imageKey)
	}

	name := "observability-ui-" + plugin.Name

	switch plugin.Spec.Type {
	case obsui.TypeDashboards:
		{
			readerRoleName := plugin.Name + "-datasource-reader"

			pluginInfo := &ObservabilityUIPluginInfo{
				Image:       image,
				Name:        name,
				ConsoleName: "console-dashboards-plugin",
				DisplayName: "Console Enhanced Dashboards",
				Proxies: []osv1alpha1.ConsolePluginProxy{
					{
						Type:      osv1alpha1.ProxyTypeService,
						Alias:     "backend",
						Authorize: true,
						Service: osv1alpha1.ConsolePluginProxyServiceConfig{
							Name:      name,
							Namespace: plugin.Namespace,
							Port:      9443,
						},
					},
				},
				ClusterRole: &rbacv1.ClusterRole{
					TypeMeta: metav1.TypeMeta{
						APIVersion: rbacv1.SchemeGroupVersion.String(),
						Kind:       "ClusterRole",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: readerRoleName,
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"configmaps"},
							Verbs:     []string{"get", "list", "watch"},
						},
					},
				},
				ClusterRoleBinding: &rbacv1.ClusterRoleBinding{
					TypeMeta: metav1.TypeMeta{
						APIVersion: rbacv1.SchemeGroupVersion.String(),
						Kind:       "ClusterRoleBinding",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: name + "-rolebinding",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup:  corev1.SchemeGroupVersion.Group,
							Kind:      "ServiceAccount",
							Name:      name + "-sa",
							Namespace: plugin.Namespace,
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.SchemeGroupVersion.Group,
						Kind:     "ClusterRole",
						Name:     readerRoleName,
					},
				},
			}

			return pluginInfo, nil
		}
	}

	return nil, fmt.Errorf("plugin type not supported: %s", plugin.Spec.Type)
}
