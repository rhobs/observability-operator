package monitoringstack

import (
	"testing"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

func TestNewAlertmanagerMatcherStrategy(t *testing.T) {
	for _, tc := range []struct {
		name     string
		strategy stack.AlertmanagerConfigMatcherStrategyType
		expected monv1.AlertmanagerConfigMatcherStrategyType
	}{
		{
			name:     "None",
			strategy: stack.NoneMatcherStrategy,
			expected: monv1.NoneConfigMatcherStrategyType,
		},
		{
			name:     "OnNamespace",
			strategy: stack.OnNamespaceMatcherStrategy,
			expected: monv1.OnNamespaceConfigMatcherStrategyType,
		},
		{
			name:     "OnNamespaceExceptForAlertmanagerNamespace",
			strategy: stack.OnNamespaceExceptForAlertmanagerNamespaceMatcherStrategy,
			expected: monv1.OnNamespaceExceptForAlertmanagerNamespaceConfigMatcherStrategyType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := &stack.MonitoringStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-ns",
				},
				Spec: stack.MonitoringStackSpec{
					AlertmanagerConfig: stack.AlertmanagerConfig{
						MatcherStrategy: stack.AlertmanagerConfigMatcherStrategy{
							Type: tc.strategy,
						},
					},
				},
			}

			am := newAlertmanager(ms, "test-sa", AlertmanagerConfiguration{})
			assert.Equal(t, am.Spec.AlertmanagerConfigMatcherStrategy.Type, tc.expected)
		})
	}
}
