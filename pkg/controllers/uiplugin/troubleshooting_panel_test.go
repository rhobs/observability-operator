package uiplugin

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	rbacv1 "k8s.io/api/rbac/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

func findPolicyRule(rules []rbacv1.PolicyRule, apiGroup, resource string) *rbacv1.PolicyRule {
	for i := range rules {
		for _, g := range rules[i].APIGroups {
			if g != apiGroup {
				continue
			}
			for _, r := range rules[i].Resources {
				if r == resource {
					return &rules[i]
				}
			}
		}
	}
	return nil
}

func TestKorrel8rClusterRole(t *testing.T) {
	cr := korrel8rClusterRole("korrel8r")

	assert.Equal(t, cr.Name, "korrel8r-view")
	assert.Equal(t, cr.Kind, "ClusterRole")

	tests := []struct {
		name     string
		apiGroup string
		resource string
		verbs    []string
	}{
		{
			name:     "core resources",
			apiGroup: "",
			resource: "pods",
			verbs:    []string{"get", "list", "watch"},
		},
		{
			name:     "apps resources",
			apiGroup: "apps",
			resource: "deployments",
			verbs:    []string{"get", "list", "watch"},
		},
		{
			name:     "loki resources",
			apiGroup: "loki.grafana.com",
			resource: "application",
			verbs:    []string{"get"},
		},
		{
			name:     "tokenreviews for session authentication",
			apiGroup: "authentication.k8s.io",
			resource: "tokenreviews",
			verbs:    []string{"create"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule := findPolicyRule(cr.Rules, tc.apiGroup, tc.resource)
			assert.Assert(t, rule != nil, "expected rule for %s/%s", tc.apiGroup, tc.resource)
			assert.DeepEqual(t, rule.Verbs, tc.verbs)
		})
	}
}

func TestMarshalTroubleshootingPanelPluginConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *uiv1alpha1.TroubleshootingPanelConfig
		wantYAML string
		wantErr  bool
	}{
		{
			name:     "nil config returns empty string",
			cfg:      nil,
			wantYAML: "",
		},
		{
			name:     "empty config returns empty string",
			cfg:      &uiv1alpha1.TroubleshootingPanelConfig{},
			wantYAML: "",
		},
		{
			name: "timeout only",
			cfg: &uiv1alpha1.TroubleshootingPanelConfig{
				Timeout: "30s",
			},
			wantYAML: "timeout:",
		},
		{
			name: "enableAgentNavigation only",
			cfg: &uiv1alpha1.TroubleshootingPanelConfig{
				EnableAgentNavigation: true,
			},
			wantYAML: "enableAgentNavigation:",
		},
		{
			name: "both fields set",
			cfg: &uiv1alpha1.TroubleshootingPanelConfig{
				Timeout:               "1m",
				EnableAgentNavigation: true,
			},
			wantYAML: "enableAgentNavigation:",
		},
		{
			name: "enableAgentNavigation false with timeout",
			cfg: &uiv1alpha1.TroubleshootingPanelConfig{
				Timeout:               "30s",
				EnableAgentNavigation: false,
			},
			wantYAML: "timeout:",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := marshalTroubleshootingPanelPluginConfig(tc.cfg)
			if tc.wantErr {
				assert.Assert(t, err != nil)
				return
			}
			assert.NilError(t, err)
			if tc.wantYAML == "" {
				assert.Equal(t, got, "")
			} else {
				assert.Assert(t, strings.Contains(got, tc.wantYAML),
					"expected %q to contain %q", got, tc.wantYAML)
			}
		})
	}
}

func hasFeatureArg(pluginInfo *UIPluginInfo, feature string) bool {
	for _, arg := range pluginInfo.ExtraArgs {
		if strings.HasPrefix(arg, "-features=") {
			for _, f := range strings.Split(strings.TrimPrefix(arg, "-features="), ",") {
				if strings.TrimSpace(f) == feature {
					return true
				}
			}
		}
	}
	return false
}

func TestCreateTroubleshootingPanelPluginInfo_AgentNavigation(t *testing.T) {
	const (
		namespace = "openshift-operators"
		name      = "troubleshooting-panel"
		image     = "quay.io/tp-test:123"
	)

	t.Run("agent-navigation feature passed in features list", func(t *testing.T) {
		plugin := &uiv1alpha1.UIPlugin{
			Spec: uiv1alpha1.UIPluginSpec{
				Type: uiv1alpha1.TypeTroubleshootingPanel,
			},
		}
		plugin.Name = name

		info, err := createTroubleshootingPanelPluginInfo(plugin, namespace, name, image, []string{"agent-navigation"})
		assert.NilError(t, err)
		assert.Assert(t, hasFeatureArg(info, "agent-navigation"), "expected agent-navigation in features")
	})

	t.Run("no features when not enabled", func(t *testing.T) {
		plugin := &uiv1alpha1.UIPlugin{
			Spec: uiv1alpha1.UIPluginSpec{
				Type: uiv1alpha1.TypeTroubleshootingPanel,
			},
		}
		plugin.Name = name

		info, err := createTroubleshootingPanelPluginInfo(plugin, namespace, name, image, []string{})
		assert.NilError(t, err)
		assert.Assert(t, !hasFeatureArg(info, "agent-navigation"), "unexpected agent-navigation in features")
	})

	t.Run("enableAgentNavigation produces config yaml", func(t *testing.T) {
		plugin := &uiv1alpha1.UIPlugin{
			Spec: uiv1alpha1.UIPluginSpec{
				Type: uiv1alpha1.TypeTroubleshootingPanel,
				TroubleshootingPanel: &uiv1alpha1.TroubleshootingPanelConfig{
					EnableAgentNavigation: true,
				},
			},
		}
		plugin.Name = name

		info, err := createTroubleshootingPanelPluginInfo(plugin, namespace, name, image, []string{"agent-navigation"})
		assert.NilError(t, err)
		assert.Assert(t, info.ConfigMap != nil)
		configYAML := info.ConfigMap.Data["config.yaml"]
		assert.Assert(t, strings.Contains(configYAML, "enableAgentNavigation: true"),
			"expected config.yaml to contain enableAgentNavigation, got: %q", configYAML)
	})
}
