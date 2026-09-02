package uiplugin

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability"
)

const (
	korrel8rSvcName = "korrel8r"
)

func createTroubleshootingPanelPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string, clusterVersion string, logger logr.Logger) (*UIPluginInfo, error) {
	troubleshootingPanelConfig := plugin.Spec.TroubleshootingPanel

	var timeout string
	if troubleshootingPanelConfig != nil {
		timeout = troubleshootingPanelConfig.Timeout
	}
	configYaml, err := marshalTimeoutConfig(timeout)
	if err != nil {
		return nil, fmt.Errorf("error creating plugin configuration file: %w", err)
	}

	extraArgs := []string{
		"-plugin-config-path=/etc/plugin/config/config.yaml",
	}
	if plugin.Spec.TroubleshootingPanel != nil && plugin.Spec.TroubleshootingPanel.EnableAgentNavigation {
		if IsVersionAheadOrEqual(clusterVersion, "v4.22") {
			if !slices.Contains(features, "agent-navigation") {
				features = append(features, "agent-navigation")
			}
		} else {
			logger.Info("Agent Navigation only available as a Dev Preview in OpenShift 4.22+")
		}
	}
	if len(features) > 0 {
		extraArgs = append(extraArgs, fmt.Sprintf("-features=%s", strings.Join(features, ",")))
	}

	pluginInfo := &UIPluginInfo{
		Image:             image,
		Name:              plugin.Name,
		ConsoleName:       pluginTypeToConsoleName[plugin.Spec.Type],
		DisplayName:       "Troubleshooting Panel Console Plugin",
		LokiServiceNames:  make(map[string]string),
		TempoServiceNames: make(map[string]string),
		TracingTenant:     observability.TracingTenant,
		ExtraArgs:         extraArgs,
		Proxies: []PluginProxy{
			{
				Alias:            "korrel8r",
				ServiceName:      korrel8rSvcName,
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

func findGatewayService(ctx context.Context, k client.Client, ns, nameSubstring, componentLabel string) (string, error) {
	serviceList := &corev1.ServiceList{}
	if err := k.List(ctx, serviceList, client.InNamespace(ns)); err != nil {
		return "", err
	}
	for _, service := range serviceList.Items {
		if strings.Contains(service.Name, nameSubstring) && service.Labels["app.kubernetes.io/component"] == componentLabel {
			return service.Name, nil
		}
	}
	return "", nil
}

func getTempoServiceName(ctx context.Context, k client.Client, ns string) (string, error) {
	return findGatewayService(ctx, k, ns, "gateway", "gateway")
}

func getLokiServiceName(ctx context.Context, k client.Client, ns string) (string, error) {
	return findGatewayService(ctx, k, ns, "gateway-http", "lokistack-gateway")
}
