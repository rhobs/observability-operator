package uiplugin

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func createTroubleshootingPanelPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string) (*UIPluginInfo, error) {
	troubleshootingPanelConfig := plugin.Spec.TroubleshootingPanel
	korrel8rSvcName := "korrel8r"
	monitorClusterroleName := "cluster-monitoring"
	alertmanagerRoleName := "monitoring-alertmanager-view"
	monitoringNamespace := "openshift-monitoring"

	configYaml, err := marshalTroubleshootingPanelPluginConfig(troubleshootingPanelConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating plugin configuration file: %w", err)
	}

	extraArgs := []string{
		"-plugin-config-path=/etc/plugin/config/config.yaml",
	}

	if len(features) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("-features=%s", strings.Join(features, ",")))
	}

	pluginInfo := &UIPluginInfo{
		Image:             image,
		Name:              plugin.Name,
		ConsoleName:       pluginTypeToConsoleName[plugin.Spec.Type],
		DisplayName:       "Troubleshooting Panel Console Plugin",
		ResourceNamespace: namespace,
		LokiServiceNames:  make(map[string]string),
		TempoServiceNames: make(map[string]string),
		ExtraArgs:         extraArgs,
		LegacyProxies: []osv1alpha1.ConsolePluginProxy{
			{
				Type:      osv1alpha1.ProxyTypeService,
				Alias:     "korrel8r",
				Authorize: true,
				Service: osv1alpha1.ConsolePluginProxyServiceConfig{
					Name:      korrel8rSvcName,
					Namespace: namespace,
					Port:      port,
				},
			},
		},
		Proxies: []osv1.ConsolePluginProxy{
			{
				Alias:         "korrel8r",
				Authorization: "UserToken",
				Endpoint: osv1.ConsolePluginProxyEndpoint{
					Type: osv1.ProxyTypeService,
					Service: &osv1.ConsolePluginProxyServiceConfig{
						Name:      korrel8rSvcName,
						Namespace: namespace,
						Port:      port,
					},
				},
			},
		},
		ConfigMap: &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string]string{
				"config.yaml": configYaml,
			},
		},
		RoleBinding: &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "RoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertmanagerRoleName + "-rolebinding",
				Namespace: monitoringNamespace,
			},
			Subjects: []rbacv1.Subject{
				{
					APIGroup:  corev1.SchemeGroupVersion.Group,
					Kind:      "ServiceAccount",
					Name:      plugin.Name + "-sa",
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Kind:     "Role",
				Name:     alertmanagerRoleName,
			},
		},
		ClusterRoles: []*rbacv1.ClusterRole{
			korrel8rClusterRole(korrel8rSvcName),
		},
		ClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
			korrel8rClusterRoleBinding(monitorClusterroleName, plugin.Name, namespace),
			korrel8rClusterRoleBinding(korrel8rSvcName, plugin.Name, namespace),
		},
	}

	return pluginInfo, nil
}

func marshalTroubleshootingPanelPluginConfig(cfg *uiv1alpha1.TroubleshootingPanelConfig) (string, error) {
	if cfg == nil {
		return "", nil
	}

	if cfg.Timeout == "" {
		return "", nil
	}

	pluginCfg := struct {
		Timeout string `yaml:"timeout"`
	}{
		Timeout: cfg.Timeout,
	}

	buf := &bytes.Buffer{}
	if err := yaml.NewEncoder(buf).Encode(pluginCfg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getLokiServiceName(ctx context.Context, k client.Client, ns string) (string, error) {

	serviceList := &corev1.ServiceList{}
	if err := k.List(ctx, serviceList, client.InNamespace(ns)); err != nil {
		return "", err
	}

	// Accumulate services that contain "gateway-http" in their names
	for _, service := range serviceList.Items {
		if strings.Contains(service.Name, "gateway-http") && service.Labels["app.kubernetes.io/component"] == "lokistack-gateway" {
			return service.Name, nil
		}
	}
	return "", nil
}

func getTempoServiceName(ctx context.Context, k client.Client, ns string) (string, error) {

	serviceList := &corev1.ServiceList{}
	if err := k.List(ctx, serviceList, client.InNamespace(ns)); err != nil {
		return "", err
	}

	// Accumulate services that contain "gateway" in their names
	for _, service := range serviceList.Items {
		if strings.Contains(service.Name, "gateway") && service.Labels["app.kubernetes.io/component"] == "gateway" {
			return service.Name, nil
		}
	}
	return "", nil
}

func korrel8rClusterRole(name string) *rbacv1.ClusterRole {
	korrel8rClusterroleName := name + "-view"
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: korrel8rClusterroleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps", "endpoints", "events", "namespaces", "nodes", "pods", "persistentvolumeclaims", "persistentvolumes", "replicationcontrollers", "secrets", "serviceaccounts", "services"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"roles", "rolebindings", "clusterroles", "clusterrolebindings"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"statefulsets", "daemonsets", "deployments", "replicasets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"cronjobs", "jobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"autoscaling"},
				Resources: []string{"horizontalpodautoscalers"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"policy"},
				Resources: []string{"poddisruptionbudgets"},
				Verbs:     []string{"list", "watch"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses", "volumeattachments"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"networkpolicies", "ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"loki.grafana.com"},
				Resources: []string{"application", "audit", "infrastructure", "network"},
				Verbs:     []string{"get"},
			},
		},
	}
}

func korrel8rClusterRoleBinding(name string, serviceAccountName string, namespace string) *rbacv1.ClusterRoleBinding {
	korrel8rClusterroleBindingName := name + "-view"
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: korrel8rClusterroleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  corev1.SchemeGroupVersion.Group,
				Kind:      "ServiceAccount",
				Name:      serviceAccountName + "-sa",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     korrel8rClusterroleBindingName,
		},
	}
}
