package uiplugin

import (
	"bytes"
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	"github.com/openshift/library-go/pkg/crypto"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func createDistributedTracingPluginInfo(plugin *uiv1alpha1.UIPlugin, namespace, name, image string, features []string, tlsProfile *configv1.TLSProfileSpec) (*UIPluginInfo, error) {
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

	extraArgs = append(extraArgs, tlsProfileArgs(tlsProfile)...)

	pluginInfo := &UIPluginInfo{
		Image:             image,
		Name:              plugin.Name,
		ConsoleName:       pluginTypeToConsoleName[plugin.Spec.Type],
		DisplayName:       "Distributed Tracing Console Plugin",
		ResourceNamespace: namespace,
		ExtraArgs:         extraArgs,
		LegacyProxies: []osv1alpha1.ConsolePluginProxy{
			{
				Type:      osv1alpha1.ProxyTypeService,
				Alias:     "backend",
				Authorize: true,
				Service: osv1alpha1.ConsolePluginProxyServiceConfig{
					Name:      name,
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
						Name:      name,
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
						Resources: []string{"tempostacks", "tempomonolithics"},
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

// tlsProfileArgs returns container args for the given TLS profile spec:
// -tls-min-version=<version> and -tls-cipher-suites=<comma-separated-ciphers>.
// Ciphers are converted from OpenSSL names (used by OpenShift API) to IANA
// names (expected by k8s.io/component-base/cli/flag).
func tlsProfileArgs(spec *configv1.TLSProfileSpec) []string {
	if spec == nil {
		return nil
	}

	args := []string{
		fmt.Sprintf("-tls-min-version=%s", spec.MinTLSVersion),
	}

	if len(spec.Ciphers) > 0 {
		ianaCiphers := crypto.OpenSSLToIANACipherSuites(spec.Ciphers)
		args = append(args, fmt.Sprintf("-tls-cipher-suites=%s", strings.Join(ianaCiphers, ",")))
	}

	return args
}
