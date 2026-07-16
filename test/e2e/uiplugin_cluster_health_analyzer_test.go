package e2e

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

const (
	healthAnalyzerDeploymentName = "health-analyzer"
	prometheusRuleNamespace      = "e2e-health-analyzer-rules"
)

func clusterHealthAnalyzer(t *testing.T) {
	f.SkipIfClusterVersionBelow(t, "4.19")

	err := monv1.AddToScheme(f.K8sClient.Scheme())
	assert.NilError(t, err, "failed to add monv1 to scheme")

	f.DumpOnFailure(t, f.DebugNamespaces(uiPluginInstallNS))

	plugin := resetMonitoringUIPlugin(t)
	err = f.K8sClient.Create(t.Context(), plugin)
	assert.NilError(t, err, "failed to create monitoring UIPlugin")

	f.DumpOnFailure(t, f.DebugUIPlugin(plugin.Name))
	t.Log("Waiting for health-analyzer deployment to become ready...")
	haDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, healthAnalyzerDeploymentName, uiPluginInstallNS, &haDeployment)
	f.AssertDeploymentReady(healthAnalyzerDeploymentName, uiPluginInstallNS, framework.WithTimeout(5*time.Minute))(t)

	// Use a unique suffix so re-runs don't conflict with leftover rules from prior executions.
	suffix := strconv.FormatInt(time.Now().UnixNano()%100000, 10)
	ruleName := "e2e-health-analyzer-" + suffix
	alertName := "E2EHealthAnalyzer" + suffix

	createRuleNamespace(t, prometheusRuleNamespace)

	rule := newAlwaysFiringRule(t, ruleName, alertName)
	err = f.K8sClient.Create(t.Context(), rule)
	assert.NilError(t, err, "failed to create PrometheusRule")

	t.Log("Waiting for alert to fire in Prometheus...")
	alertQuery := fmt.Sprintf(`ALERTS{alertname="%s",alertstate="firing"}`, alertName)
	err = f.AssertPromQLResultWithOptions(t, alertQuery,
		func(v model.Value) error {
			vec, ok := v.(model.Vector)
			if !ok || len(vec) == 0 {
				return fmt.Errorf("expected firing alert, got: %v", v)
			}
			return nil
		},
		framework.WithPollInterval(30*time.Second),
		framework.WithTimeout(10*time.Minute),
	)
	assert.NilError(t, err, "alert %s never fired", alertName)

	t.Log("Waiting for cluster-health-analyzer to expose incident metric...")
	incidentQuery := fmt.Sprintf(`cluster_health_components_map{src_alertname="%s",src_severity="warning"}`, alertName)
	err = f.AssertPromQLResultWithOptions(t, incidentQuery,
		func(v model.Value) error {
			vec, ok := v.(model.Vector)
			if !ok || len(vec) == 0 {
				return fmt.Errorf("expected incident metric, got: %v", v)
			}
			return nil
		},
		framework.WithPollInterval(30*time.Second),
		framework.WithTimeout(15*time.Minute),
	)
	assert.NilError(t, err, "incident metric for %s never appeared", alertName)
}

func resetMonitoringUIPlugin(t *testing.T) *uiv1.UIPlugin {
	plugin := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: uiv1.MonitoringPluginName,
		},
		Spec: uiv1.UIPluginSpec{
			Type: uiv1.TypeMonitoring,
			Monitoring: &uiv1.MonitoringConfig{
				ClusterHealthAnalyzer: &uiv1.ClusterHealthAnalyzerReference{
					Enabled: true,
				},
			},
		},
	}

	deleteUIPluginIfExists(t, plugin.Name)

	f.CleanUp(t, func() {
		ctx := context.WithoutCancel(t.Context())
		if err := f.K8sClient.Delete(ctx, plugin); err != nil && !errors.IsNotFound(err) {
			t.Logf("warning: failed to delete UIPlugin during cleanup: %v", err)
		}
		waitForUIPluginDeletion(plugin)
	})
	return plugin
}

func deleteUIPluginIfExists(t *testing.T, name string) {
	t.Helper()
	plugin := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	err := f.K8sClient.Delete(t.Context(), plugin)
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		t.Fatalf("failed to delete existing UIPlugin: %v", err)
	}
	t.Log("UIPlugin already existed, waiting for deletion...")
	waitForUIPluginDeletion(plugin)
}

func createRuleNamespace(t *testing.T, name string) {
	t.Helper()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"openshift.io/cluster-monitoring": "true",
			},
		},
	}
	if err := f.K8sClient.Create(t.Context(), ns); err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("failed to create rule namespace %s: %v", name, err)
	}
	f.DumpOnFailure(t, f.DebugNamespaces(name))
	f.CleanUp(t, func() {
		ctx := context.WithoutCancel(t.Context())
		f.K8sClient.Delete(ctx, ns)
	})
}

func newAlwaysFiringRule(t *testing.T, ruleName, alertName string) *monv1.PrometheusRule {
	rule := &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: prometheusRuleNamespace,
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{{
				Name: "health-analyzer-test-" + ruleName,
				Rules: []monv1.Rule{{
					Alert:  alertName,
					Expr:   intstr.FromString(`vector(1)`),
					Labels: map[string]string{"severity": "warning"},
					Annotations: map[string]string{
						"summary": "E2E static test alert for cluster health analyzer.",
					},
				}},
			}},
		},
	}
	f.CleanUp(t, func() {
		ctx := context.WithoutCancel(t.Context())
		if err := f.K8sClient.Delete(ctx, rule); err != nil && !errors.IsNotFound(err) {
			t.Logf("warning: failed to delete PrometheusRule during cleanup: %v", err)
		}
	})
	return rule
}
