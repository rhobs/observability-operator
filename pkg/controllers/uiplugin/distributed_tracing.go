package uiplugin

import (
	"bytes"
	"fmt"
	"strings"

	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func createDistributedTracingPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string) (*UIPluginInfo, error) {
	distributedTracingConfig := plugin.Spec.DistributedTracing

	configYaml, err := marshalDistributedTracingPluginConfig(distributedTracingConfig)
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
		ConsoleName:       "distributed-tracing-console-plugin",
		DisplayName:       "Distributed Tracing Console Plugin",
		ResourceNamespace: namespace,
		ExtraArgs:         extraArgs,
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
		ClusterRoles: []*rbacv1.ClusterRole{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.String(),
					Kind:       "ClusterRole",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      plugin.Name + "-cr",
					Namespace: namespace,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"tempo.grafana.com"},
						Resources: []string{"tempostacks"},
						Verbs:     []string{"list"},
					},
				},
			},
		},
		ClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacv1.SchemeGroupVersion.String(),
					Kind:       "ClusterRoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      plugin.Name + "-crb",
					Namespace: namespace,
				},
				Subjects: []rbacv1.Subject{{
					APIGroup:  corev1.SchemeGroupVersion.Group,
					Kind:      "ServiceAccount",
					Name:      plugin.Name + "-sa",
					Namespace: namespace,
				}},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.SchemeGroupVersion.Group,
					Kind:     "ClusterRole",
					Name:     plugin.Name + "-cr",
				},
			},
		},
	}

	return pluginInfo, nil
}

func marshalDistributedTracingPluginConfig(cfg *uiv1alpha1.DistributedTracingConfig) (string, error) {
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
