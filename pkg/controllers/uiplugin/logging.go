package uiplugin

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type loggingConfig struct {
	LogsLimit int32         `yaml:"logsLimit,omitempty"`
	Timeout   time.Duration `yaml:"timeout,omitempty"`
}

func createLoggingPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string) (*UIPluginInfo, error) {
	config := plugin.Spec.Logging
	if config == nil {
		return nil, fmt.Errorf("logging configuration can not be empty for plugin type %s", plugin.Spec.Type)
	}

	if config.LokiStack.Name == "" {
		return nil, fmt.Errorf("LokiStack name can not be empty for plugin type %s", plugin.Spec.Type)
	}

	configYaml, err := marshalLoggingPluginConfig(config)
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
		Name:              name,
		ConsoleName:       pluginTypeToConsoleName[plugin.Spec.Type],
		DisplayName:       "Logging View",
		ExtraArgs:         extraArgs,
		ResourceNamespace: namespace,
		LegacyProxies: []osv1alpha1.ConsolePluginProxy{
			{
				Type:      "Service",
				Alias:     "backend",
				Authorize: true,
				Service: osv1alpha1.ConsolePluginProxyServiceConfig{
					Name:      fmt.Sprintf("%s-gateway-http", plugin.Spec.Logging.LokiStack.Name),
					Namespace: "openshift-logging", // TODO decide if we want to support LokiStack in other namespaces
					Port:      8080,
				},
			},
			{
				Type:      "Service",
				Alias:     "korrel8r",
				Authorize: true,
				Service: osv1alpha1.ConsolePluginProxyServiceConfig{
					Name:      korrel8rName,
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
						Name:      fmt.Sprintf("%s-gateway-http", plugin.Spec.Logging.LokiStack.Name),
						Namespace: "openshift-logging", // TODO decide if we want to support LokiStack in other namespaces
						Port:      8080,
					},
				},
			},
			{
				Alias:         "korrel8r",
				Authorization: "UserToken",
				Endpoint: osv1.ConsolePluginProxyEndpoint{
					Type: osv1.ProxyTypeService,
					Service: &osv1.ConsolePluginProxyServiceConfig{
						Name:      korrel8rName,
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
		ClusterRoles: []*rbacv1.ClusterRole{
			loggingClusterRole("application"),
			loggingClusterRole("infrastructure"),
			loggingClusterRole("audit"),
		},
	}

	return pluginInfo, nil
}

func marshalLoggingPluginConfig(cfg *uiv1alpha1.LoggingConfig) (string, error) {
	if cfg.LogsLimit == 0 && cfg.Timeout == "" {
		return "", nil
	}

	timeout := time.Duration(0)
	if cfg.Timeout != "" {
		var err error
		timeout, err = parseTimeoutValue(cfg.Timeout)
		if err != nil {
			return "", fmt.Errorf("can not parse timeout: %w", err)
		}
	}

	pluginCfg := loggingConfig{
		LogsLimit: cfg.LogsLimit,
		Timeout:   timeout,
	}

	buf := &bytes.Buffer{}
	if err := yaml.NewEncoder(buf).Encode(pluginCfg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func parseTimeoutValue(timeout string) (time.Duration, error) {
	duration, err := time.ParseDuration(timeout)
	if err == nil {
		return duration, nil
	}

	seconds, err := strconv.ParseUint(timeout, 10, 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(seconds) * time.Second, nil
}

func loggingClusterRole(tenant string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("cluster-logging-%s-view", tenant),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"loki.grafana.com",
				},
				Resources: []string{
					tenant,
				},
				ResourceNames: []string{
					"logs",
				},
				Verbs: []string{
					"get",
				},
			},
		},
	}
}
