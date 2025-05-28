package uiplugin

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type loggingConfig struct {
	LogsLimit int32         `yaml:"logsLimit,omitempty"`
	Timeout   time.Duration `yaml:"timeout,omitempty"`
	Schema    string        `yaml:"schema,omitempty"`
}

func createLoggingPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string, ctx context.Context, dk dynamic.Interface, logger logr.Logger) (*UIPluginInfo, error) {
	lokiStack, err := getLokiStack(plugin, ctx, dk, logger)
	if err != nil {
		return nil, err
	}

	lokiStackName := lokiStack.Name
	lokiStackNamespace := lokiStack.Namespace

	config := plugin.Spec.Logging

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
					Name:      fmt.Sprintf("%s-gateway-http", lokiStackName),
					Namespace: lokiStackNamespace,
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
						Name:      fmt.Sprintf("%s-gateway-http", lokiStackName),
						Namespace: lokiStackNamespace,
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
	if cfg == nil {
		return "", nil
	}

	if cfg.LogsLimit == 0 && cfg.Timeout == "" && cfg.Schema == "" {
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
		Schema:    cfg.Schema,
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

var lokiStackResource = schema.GroupVersionResource{
	Group: "loki.grafana.com", Version: "v1", Resource: "lokistacks",
}

// getLokiStack returns the LokiStack resource to use for the logging plugin.
// It either uses the explicitly configured LokiStack or discovers one from the cluster.
func getLokiStack(plugin *uiv1alpha1.UIPlugin, ctx context.Context, client dynamic.Interface, logger logr.Logger) (*types.NamespacedName, error) {
	config := plugin.Spec.Logging

	searchNamespace := OpenshiftLoggingNs
	if config != nil && config.LokiStack != nil && config.LokiStack.Namespace != "" {
		searchNamespace = config.LokiStack.Namespace
	}

	if config != nil && config.LokiStack != nil && config.LokiStack.Name != "" {
		lokiStack, err := client.Resource(lokiStackResource).Namespace(searchNamespace).Get(ctx, config.LokiStack.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get LokiStack %s in namespace %s: %w", config.LokiStack.Name, searchNamespace, err)
		}

		return &types.NamespacedName{
			Name:      lokiStack.GetName(),
			Namespace: searchNamespace,
		}, nil
	}

	lokiStacks, err := client.Resource(lokiStackResource).Namespace(searchNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Info("Failed to list LokiStacks in namespace, will use default", "namespace", searchNamespace, "error", err.Error())
	}

	if lokiStacks != nil && len(lokiStacks.Items) > 0 {
		return &types.NamespacedName{
			Name:      lokiStacks.Items[0].GetName(),
			Namespace: searchNamespace,
		}, nil
	}

	return &types.NamespacedName{
		Name:      "loki-stack",
		Namespace: searchNamespace,
	}, nil
}
