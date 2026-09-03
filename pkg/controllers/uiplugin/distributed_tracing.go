package uiplugin

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability"
)

func createDistributedTracingPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string) (*UIPluginInfo, error) {
	distributedTracingConfig := plugin.Spec.DistributedTracing

	var timeout string
	if distributedTracingConfig != nil {
		timeout = distributedTracingConfig.Timeout
	}
	configYaml, err := marshalTimeoutConfig(timeout)
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
		DisplayName:       "Distributed Tracing Console Plugin",
		TracingTenant:     observability.TracingTenant,
		TempoServiceNames: map[string]string{OpenshiftTracingNs: defaultTempoService},
		ExtraArgs:         extraArgs,
		Proxies: []PluginProxy{
			{
				Alias:            "backend",
				ServiceName:      name,
				ServiceNamespace: namespace,
				ServicePort:      port,
				Authorize:        true,
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
	}

	return pluginInfo, nil
}
