package e2e

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/common/model"
	"golang.org/x/mod/semver"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

const (
	healthAnalyzerDeploymentName = "health-analyzer"
	prometheusRuleNamespace      = "openshift-monitoring"
)

func clusterHealthAnalyzer(t *testing.T) {
	skipIfClusterVersionBelow(t, "4.19")

	monv1.AddToScheme(f.K8sClient.Scheme())

	plugin := newMonitoringUIPlugin(t)
	err := f.K8sClient.Create(context.Background(), plugin)
	assert.NilError(t, err, "failed to create monitoring UIPlugin")

	t.Cleanup(func() {
		if t.Failed() {
			dumpClusterHealthAnalyzerDebug(t, plugin.Name)
		}
	})

	t.Log("Waiting for health-analyzer deployment to become ready...")
	haDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, healthAnalyzerDeploymentName, uiPluginInstallNS, &haDeployment)
	f.AssertDeploymentReady(healthAnalyzerDeploymentName, uiPluginInstallNS, framework.WithTimeout(5*time.Minute))(t)

	suffix := strconv.FormatInt(time.Now().UnixNano()%100000, 10)
	ruleName := "e2e-crashloop-" + suffix
	alertName := "E2ECrashLoop" + suffix
	deployName := "e2e-crasher-" + suffix

	rule := newCrashLoopRule(t, ruleName, alertName, deployName)
	err = f.K8sClient.Create(context.Background(), rule)
	assert.NilError(t, err, "failed to create PrometheusRule")

	dep := newCrashingDeployment(t, deployName)
	err = f.K8sClient.Create(context.Background(), dep)
	assert.NilError(t, err, "failed to create crashing deployment")

	t.Log("Waiting for pod to enter CrashLoopBackOff...")
	assertPodCrashLooping(t, deployName, e2eTestNamespace, 10*time.Second, 3*time.Minute)

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
	incidentQuery := fmt.Sprintf(`cluster_health_components_map{src_alertname="%s"}`, alertName)
	err = f.AssertPromQLResultWithOptions(t, incidentQuery,
		func(v model.Value) error {
			vec, ok := v.(model.Vector)
			if !ok || len(vec) == 0 {
				return fmt.Errorf("expected incident metric, got: %v", v)
			}
			for _, sample := range vec {
				if string(sample.Metric["src_alertname"]) != alertName {
					return fmt.Errorf("expected src_alertname=%s, got %s", alertName, sample.Metric["src_alertname"])
				}
				if string(sample.Metric["src_severity"]) != "warning" {
					return fmt.Errorf("expected src_severity=warning, got %s", sample.Metric["src_severity"])
				}
			}
			return nil
		},
		framework.WithPollInterval(30*time.Second),
		framework.WithTimeout(15*time.Minute),
	)
	assert.NilError(t, err, "incident metric for %s never appeared", alertName)
}

func newMonitoringUIPlugin(t *testing.T) *uiv1.UIPlugin {
	plugin := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: "monitoring",
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

	existing := &uiv1.UIPlugin{}
	err := f.K8sClient.Get(context.Background(), client.ObjectKey{Name: plugin.Name}, existing)
	if err == nil {
		t.Log("UIPlugin 'monitoring' already exists, deleting before recreation...")
		f.K8sClient.Delete(context.Background(), existing)
		waitForUIPluginDeletion(existing)
	} else if !errors.IsNotFound(err) {
		t.Fatalf("failed to check for existing UIPlugin: %v", err)
	}

	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), plugin)
		waitForUIPluginDeletion(plugin)
	})
	return plugin
}

func newCrashLoopRule(t *testing.T, ruleName, alertName, podPrefix string) *monv1.PrometheusRule {
	rule := &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: prometheusRuleNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "kube-prometheus",
				"app.kubernetes.io/part-of": "openshift-monitoring",
				"prometheus":                "k8s",
				"role":                      "alert-rules",
			},
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{{
				Name: "crashloop-test-" + ruleName,
				Rules: []monv1.Rule{{
					Alert: alertName,
					Expr: intstr.FromString(fmt.Sprintf(
						`max_over_time(kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff", namespace="%s", pod=~"%s.*", job="kube-state-metrics"}[5m]) >= 1`,
						e2eTestNamespace, podPrefix)),
					For:    ptr.To(monv1.Duration("1m")),
					Labels: map[string]string{"severity": "warning"},
					Annotations: map[string]string{
						"summary": "Pod is crash looping.",
					},
				}},
			}},
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), rule)
	})
	return rule
}

func newCrashingDeployment(t *testing.T, name string) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "crasher",
						Image:   "registry.access.redhat.com/ubi9-minimal:latest",
						Command: []string{"sh", "-c", "exit 1"},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1m"),
								corev1.ResourceMemory: resource.MustParse("4Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("16Mi"),
							},
						},
					}},
				},
			},
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), dep)
	})
	return dep
}

func assertPodCrashLooping(t *testing.T, deploymentName, namespace string, pollInterval, timeout time.Duration) {
	t.Helper()
	err := wait.PollUntilContextTimeout(context.Background(), pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		var pods corev1.PodList
		if err := f.K8sClient.List(ctx, &pods,
			client.InNamespace(namespace),
			client.MatchingLabels{"app": deploymentName},
		); err != nil {
			return false, nil
		}
		for i := range pods.Items {
			for _, cs := range pods.Items[i].Status.ContainerStatuses {
				if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
					return true, nil
				}
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("pod with label app=%s in %s never entered CrashLoopBackOff: %v", deploymentName, namespace, err)
	}
}

func dumpClusterHealthAnalyzerDebug(t *testing.T, pluginName string) {
	t.Helper()
	ctx := context.Background()

	t.Log("=== BEGIN DEBUG DUMP ===")

	// Dump UIPlugin status
	var plugin uiv1.UIPlugin
	if err := f.K8sClient.Get(ctx, client.ObjectKey{Name: pluginName}, &plugin); err != nil {
		t.Logf("Failed to get UIPlugin %q: %v", pluginName, err)
	} else {
		t.Logf("UIPlugin %q generation=%d, resourceVersion=%s", pluginName, plugin.Generation, plugin.ResourceVersion)
		t.Logf("UIPlugin spec.type=%s", plugin.Spec.Type)
		if plugin.Spec.Monitoring != nil {
			if plugin.Spec.Monitoring.ClusterHealthAnalyzer != nil {
				t.Logf("UIPlugin spec.monitoring.clusterHealthAnalyzer.enabled=%v", plugin.Spec.Monitoring.ClusterHealthAnalyzer.Enabled)
			}
			if plugin.Spec.Monitoring.Incidents != nil {
				t.Logf("UIPlugin spec.monitoring.incidents.enabled=%v", plugin.Spec.Monitoring.Incidents.Enabled)
			}
		}
		if len(plugin.Status.Conditions) == 0 {
			t.Log("UIPlugin has no status conditions")
		}
		for _, c := range plugin.Status.Conditions {
			t.Logf("UIPlugin condition: type=%s status=%s reason=%s message=%s", c.Type, c.Status, c.Reason, c.Message)
		}
	}

	// List all UIPlugins
	var plugins uiv1.UIPluginList
	if err := f.K8sClient.List(ctx, &plugins); err != nil {
		t.Logf("Failed to list UIPlugins: %v", err)
	} else {
		t.Logf("Total UIPlugins in cluster: %d", len(plugins.Items))
		for _, p := range plugins.Items {
			t.Logf("  UIPlugin: name=%s type=%s conditions=%d", p.Name, p.Spec.Type, len(p.Status.Conditions))
		}
	}

	// List all deployments in the operator namespace
	var deployments appsv1.DeploymentList
	if err := f.K8sClient.List(ctx, &deployments, client.InNamespace(uiPluginInstallNS)); err != nil {
		t.Logf("Failed to list deployments in %s: %v", uiPluginInstallNS, err)
	} else {
		t.Logf("Deployments in namespace %s: %d", uiPluginInstallNS, len(deployments.Items))
		for _, d := range deployments.Items {
			t.Logf("  Deployment: name=%s replicas=%d readyReplicas=%d availableReplicas=%d",
				d.Name, ptrInt32(d.Spec.Replicas), d.Status.ReadyReplicas, d.Status.AvailableReplicas)
			for _, c := range d.Status.Conditions {
				t.Logf("    condition: type=%s status=%s reason=%s message=%s",
					c.Type, c.Status, c.Reason, c.Message)
			}
		}
	}

	// List all pods in the operator namespace
	var pods corev1.PodList
	if err := f.K8sClient.List(ctx, &pods, client.InNamespace(uiPluginInstallNS)); err != nil {
		t.Logf("Failed to list pods in %s: %v", uiPluginInstallNS, err)
	} else {
		t.Logf("Pods in namespace %s: %d", uiPluginInstallNS, len(pods.Items))
		for _, p := range pods.Items {
			t.Logf("  Pod: name=%s phase=%s", p.Name, p.Status.Phase)
			for _, cs := range p.Status.ContainerStatuses {
				if cs.State.Running != nil {
					t.Logf("    container=%s ready=%v restarts=%d state=Running", cs.Name, cs.Ready, cs.RestartCount)
				} else if cs.State.Waiting != nil {
					t.Logf("    container=%s ready=%v restarts=%d state=Waiting reason=%s message=%s",
						cs.Name, cs.Ready, cs.RestartCount, cs.State.Waiting.Reason, cs.State.Waiting.Message)
				} else if cs.State.Terminated != nil {
					t.Logf("    container=%s ready=%v restarts=%d state=Terminated reason=%s exitCode=%d",
						cs.Name, cs.Ready, cs.RestartCount, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode)
				}
			}
		}
	}

	// List events in the operator namespace
	var events corev1.EventList
	if err := f.K8sClient.List(ctx, &events, client.InNamespace(uiPluginInstallNS)); err != nil {
		t.Logf("Failed to list events in %s: %v", uiPluginInstallNS, err)
	} else {
		t.Logf("Events in namespace %s: %d", uiPluginInstallNS, len(events.Items))
		for _, e := range events.Items {
			t.Logf("  Event: involvedObject=%s/%s reason=%s message=%s type=%s count=%d",
				e.InvolvedObject.Kind, e.InvolvedObject.Name, e.Reason, e.Message, e.Type, e.Count)
		}
	}

	t.Log("=== END DEBUG DUMP ===")
}

func ptrInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func skipIfClusterVersionBelow(t *testing.T, minVersion string) {
	t.Helper()
	cv := &configv1.ClusterVersion{}
	err := f.K8sClient.Get(context.Background(), client.ObjectKey{Name: "version"}, cv)
	if err != nil {
		t.Skipf("Skipping: unable to determine cluster version: %v", err)
		return
	}

	actual := cv.Status.Desired.Version
	if actual == "" {
		t.Skip("Skipping: cluster version is empty")
		return
	}
	t.Logf("Detected cluster version: %s", actual)

	if !strings.HasPrefix(actual, "v") {
		actual = "v" + actual
	}
	if !strings.HasPrefix(minVersion, "v") {
		minVersion = "v" + minVersion
	}

	canonicalActual := fmt.Sprintf("%s-0", semver.Canonical(actual))
	canonicalMin := fmt.Sprintf("%s-0", semver.Canonical(minVersion))

	if semver.Compare(canonicalActual, canonicalMin) < 0 {
		t.Skipf("Skipping: cluster version %s is below minimum required %s", cv.Status.Desired.Version, minVersion)
	}
}
