package e2e

import (
	"context"
	"testing"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestPrometheusRuleWebhook(t *testing.T) {
	assertCRDExists(t,
		"prometheusrules.monitoring.rhobs",
	)
	ts := []testCase{{
		name:     "Valid PrometheusRules are accepted",
		scenario: validPrometheusRuleIsAccepted,
	}, {
		name:     "Invalid PrometheusRules are rejected",
		scenario: invalidPrometheusRuleIsRejected,
	}}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func validPrometheusRuleIsAccepted(t *testing.T) {
	rule := newSinglePrometheusRule(t, "valid-rule",
		`increase(controller_runtime_reconcile_errors_total{job="foobar"}[15m]) > 0`,
	)
	err := f.K8sClient.Create(context.Background(), rule)
	assert.NilError(t, err, `failed to create a valid log`)
}

func invalidPrometheusRuleIsRejected(t *testing.T) {
	rule := newSinglePrometheusRule(t, "valid-rule", `FOOBAR({job="foobar"}[15m]) > 0`)
	err := f.K8sClient.Create(context.Background(), rule)
	assert.ErrorContains(t, err, `denied the request: Rules are not valid`)
}

func newSinglePrometheusRule(t *testing.T, name, expr string) *monv1.PrometheusRule {
	rule := &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{{
				Name: "single-rule-group",
				Rules: []monv1.Rule{{
					Alert: "alert name",
					Expr:  intstr.FromString(expr),
					For:   ptr.To(monv1.Duration("15m")),
				}},
			}},
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), rule)
	})

	return rule
}
