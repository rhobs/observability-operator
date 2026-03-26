package uiplugin

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/library-go/pkg/crypto"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

type tlsArgsStatus struct {
	minVersion   string // value after -tls-min-version=, empty if absent
	cipherSuites string // value after -tls-cipher-suites=, empty if absent
}

func extractTLSArgs(pluginInfo *UIPluginInfo) tlsArgsStatus {
	var status tlsArgsStatus
	for _, arg := range pluginInfo.ExtraArgs {
		if strings.HasPrefix(arg, "-tls-min-version=") {
			status.minVersion = strings.TrimPrefix(arg, "-tls-min-version=")
		}
		if strings.HasPrefix(arg, "-tls-cipher-suites=") {
			status.cipherSuites = strings.TrimPrefix(arg, "-tls-cipher-suites=")
		}
	}
	return status
}

func getDistributedTracingPluginInfo(plugin *uiv1alpha1.UIPlugin, tlsProfile *configv1.TLSProfileSpec) (*UIPluginInfo, error) {
	const (
		namespace = "openshift-operators"
		name      = "distributed-tracing"
		image     = "quay.io/distributed-tracing-foo-test:123"
	)

	return createDistributedTracingPluginInfo(plugin, namespace, name, image, []string{}, tlsProfile)
}

var tracingPlugin = &uiv1alpha1.UIPlugin{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "observability.openshift.io/v1alpha1",
		Kind:       "UIPlugin",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "distributed-tracing-plugin",
	},
	Spec: uiv1alpha1.UIPluginSpec{
		Type: "distributed-tracing",
	},
}

func TestCreateDistributedTracingPluginInfo(t *testing.T) {
	intermediateProfile := configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
	modernProfile := configv1.TLSProfiles[configv1.TLSProfileModernType]
	oldProfile := configv1.TLSProfiles[configv1.TLSProfileOldType]

	type testCase struct {
		name             string
		tlsProfile       *configv1.TLSProfileSpec
		expectedTLSArgs  tlsArgsStatus
	}

	testCases := []testCase{
		{
			name:       "nil TLS profile",
			tlsProfile: nil,
			expectedTLSArgs: tlsArgsStatus{
				minVersion:   "",
				cipherSuites: "",
			},
		},
		{
			name:       "Intermediate profile (TLS 1.2 + ciphers)",
			tlsProfile: intermediateProfile,
			expectedTLSArgs: tlsArgsStatus{
				minVersion:   string(intermediateProfile.MinTLSVersion),
				cipherSuites: strings.Join(crypto.OpenSSLToIANACipherSuites(intermediateProfile.Ciphers), ","),
			},
		},
		{
			name:       "Modern profile (TLS 1.3)",
			tlsProfile: modernProfile,
			expectedTLSArgs: tlsArgsStatus{
				minVersion:   string(modernProfile.MinTLSVersion),
				cipherSuites: strings.Join(crypto.OpenSSLToIANACipherSuites(modernProfile.Ciphers), ","),
			},
		},
		{
			name:       "Old profile (TLS 1.0 + ciphers)",
			tlsProfile: oldProfile,
			expectedTLSArgs: tlsArgsStatus{
				minVersion:   string(oldProfile.MinTLSVersion),
				cipherSuites: strings.Join(crypto.OpenSSLToIANACipherSuites(oldProfile.Ciphers), ","),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pluginInfo, err := getDistributedTracingPluginInfo(tracingPlugin, tc.tlsProfile)
			assert.NilError(t, err, "getDistributedTracingPluginInfo returned an unexpected error")

			// Verify TLS args
			actualTLS := extractTLSArgs(pluginInfo)
			assert.Equal(t, actualTLS.minVersion, tc.expectedTLSArgs.minVersion, "TLS min version mismatch")
			assert.Equal(t, actualTLS.cipherSuites, tc.expectedTLSArgs.cipherSuites, "TLS cipher suites mismatch")

			// Verify plugin-config-path is always present
			hasConfigPath := false
			for _, arg := range pluginInfo.ExtraArgs {
				if arg == "-plugin-config-path=/etc/plugin/config/config.yaml" {
					hasConfigPath = true
					break
				}
			}
			assert.Assert(t, hasConfigPath, "ExtraArgs must contain -plugin-config-path=/etc/plugin/config/config.yaml")

			// Verify a single "backend" proxy is present
			assert.Equal(t, len(pluginInfo.Proxies), 1, "Expected exactly one proxy")
			assert.Equal(t, pluginInfo.Proxies[0].Alias, "backend", "Expected proxy alias to be 'backend'")

			// Verify ConfigMap is non-nil
			assert.Assert(t, pluginInfo.ConfigMap != nil, "ConfigMap must not be nil")

			// Verify ClusterRoles and ClusterRoleBindings are present
			assert.Assert(t, len(pluginInfo.ClusterRoles) > 0, "ClusterRoles must be present")
			assert.Assert(t, len(pluginInfo.ClusterRoleBindings) > 0, "ClusterRoleBindings must be present")

			// Verify plugin name, image, namespace flow through correctly
			assert.Equal(t, pluginInfo.Name, tracingPlugin.Name, "Plugin name mismatch")
			assert.Equal(t, pluginInfo.Image, "quay.io/distributed-tracing-foo-test:123", "Plugin image mismatch")
			assert.Equal(t, pluginInfo.ResourceNamespace, "openshift-operators", "Plugin namespace mismatch")
		})
	}
}
