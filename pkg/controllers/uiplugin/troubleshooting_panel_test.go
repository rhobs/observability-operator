package uiplugin

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
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

func newTroubleshootingPanelPlugin(cfg *uiv1alpha1.TroubleshootingPanelConfig) *uiv1alpha1.UIPlugin {
	return &uiv1alpha1.UIPlugin{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "observability.openshift.io/v1alpha1",
			Kind:       "UIPlugin",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "troubleshooting-panel",
		},
		Spec: uiv1alpha1.UIPluginSpec{
			Type:                 uiv1alpha1.TypeTroubleshootingPanel,
			TroubleshootingPanel: cfg,
		},
	}
}

func getTroubleshootingPanelPluginInfo(plugin *uiv1alpha1.UIPlugin, features []string, clusterVersion string, logger logr.Logger) (*UIPluginInfo, error) {
	return createTroubleshootingPanelPluginInfo(plugin, "openshift-operators", plugin.Name, "quay.io/tp-test:latest", features, clusterVersion, logger)
}

func findFeaturesArg(args []string) string {
	for _, arg := range args {
		if v, ok := strings.CutPrefix(arg, "-features="); ok {
			return v
		}
	}
	return ""
}

func TestCreateTroubleshootingPanelPluginInfo(t *testing.T) {
	logger := ctrl.Log.WithName("troubleshooting-tests")
	t.Run("no features when EnableAgentNavigation is false", func(t *testing.T) {
		plugin := newTroubleshootingPanelPlugin(&uiv1alpha1.TroubleshootingPanelConfig{
			Timeout: "30s",
		})
		info, err := getTroubleshootingPanelPluginInfo(plugin, nil, "v4.19.0", logger)
		assert.NilError(t, err)
		assert.Equal(t, findFeaturesArg(info.ExtraArgs), "")
	})

	t.Run("agent-navigation feature is not set in < 4.22", func(t *testing.T) {
		plugin := newTroubleshootingPanelPlugin(&uiv1alpha1.TroubleshootingPanelConfig{
			EnableAgentNavigation: true,
			Timeout:               "30s",
		})
		info, err := getTroubleshootingPanelPluginInfo(plugin, nil, "v4.19.0", logger)
		assert.NilError(t, err)
		assert.Equal(t, findFeaturesArg(info.ExtraArgs), "")
	})

	t.Run("agent-navigation feature is set", func(t *testing.T) {
		plugin := newTroubleshootingPanelPlugin(&uiv1alpha1.TroubleshootingPanelConfig{
			EnableAgentNavigation: true,
			Timeout:               "30s",
		})
		info, err := getTroubleshootingPanelPluginInfo(plugin, nil, "v4.22.0", logger)
		assert.NilError(t, err)
		assert.Equal(t, findFeaturesArg(info.ExtraArgs), "agent-navigation")
	})

	t.Run("nil TroubleshootingPanel config", func(t *testing.T) {
		plugin := newTroubleshootingPanelPlugin(nil)
		info, err := getTroubleshootingPanelPluginInfo(plugin, nil, "v4.19.0", logger)
		assert.NilError(t, err)
		assert.Equal(t, findFeaturesArg(info.ExtraArgs), "")
	})

	t.Run("config yaml includes timeout", func(t *testing.T) {
		plugin := newTroubleshootingPanelPlugin(&uiv1alpha1.TroubleshootingPanelConfig{
			Timeout: "5m",
		})
		info, err := getTroubleshootingPanelPluginInfo(plugin, nil, "v4.19.0", logger)
		assert.NilError(t, err)
		assert.Assert(t, info.ConfigMap != nil)
		assert.Assert(t, strings.Contains(info.ConfigMap.Data["config.yaml"], "timeout: 5m"))
	})

	t.Run("proxies are configured for korrel8r", func(t *testing.T) {
		plugin := newTroubleshootingPanelPlugin(nil)
		info, err := getTroubleshootingPanelPluginInfo(plugin, nil, "v4.19.0", logger)
		assert.NilError(t, err)
		assert.Equal(t, len(info.Proxies), 1)
		assert.Equal(t, info.Proxies[0].Alias, "korrel8r")
		assert.Equal(t, info.Proxies[0].ServiceName, "korrel8r")
		assert.Equal(t, info.Proxies[0].ServiceNamespace, "openshift-operators")
	})

	t.Run("multiple features are comma-joined", func(t *testing.T) {
		plugin := newTroubleshootingPanelPlugin(nil)
		info, err := getTroubleshootingPanelPluginInfo(plugin, []string{"agent-navigation", "other-feature"}, "v4.22.0", logger)
		assert.NilError(t, err)
		assert.Equal(t, findFeaturesArg(info.ExtraArgs), "agent-navigation,other-feature")
	})
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
