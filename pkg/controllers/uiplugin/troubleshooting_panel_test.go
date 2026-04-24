package uiplugin

import (
	"testing"

	"gotest.tools/v3/assert"
	rbacv1 "k8s.io/api/rbac/v1"
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
