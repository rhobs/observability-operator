package monitoring

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/rhobs/observability-operator/must-gather/internal/api"
	"github.com/rhobs/observability-operator/must-gather/internal/client"
)

// proxyCall records the arguments of a PodProxyGet invocation.
type proxyCall struct {
	namespace string
	pod       string
	port      string
	endpoint  string
}

// fakeClient is a test double for the monitoring.Client interface.
type fakeClient struct {
	// pods returns pods keyed by "<namespace>|<labelSelector>".
	pods map[string][]corev1.Pod
	// resources are the MonitoringStacks returned by ListResources.
	resources []client.ResourceRef

	proxyCalls []proxyCall
	// proxyFunc lets a test customise proxy output/errors.
	proxyFunc func(call proxyCall) (string, error)
}

func (f *fakeClient) ListPods(_ context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	key := namespace + "|" + labelSelector
	return &corev1.PodList{Items: f.pods[key]}, nil
}

func (f *fakeClient) ListResources(_ context.Context, _ schema.GroupVersionResource) ([]client.ResourceRef, error) {
	return f.resources, nil
}

func (f *fakeClient) PodProxyGet(_ context.Context, namespace, pod, port, endpoint string) (string, error) {
	call := proxyCall{namespace: namespace, pod: pod, port: port, endpoint: endpoint}
	f.proxyCalls = append(f.proxyCalls, call)
	if f.proxyFunc != nil {
		return f.proxyFunc(call)
	}
	return "{}", nil
}

func runningPod(name string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Ready: true},
			},
		},
	}
}

func newTestCollector(t *testing.T, fc *fakeClient) (*Collector, string) {
	t.Helper()
	tmp := t.TempDir()
	c := NewCollector(fc, api.NewLogger(&bytes.Buffer{}), api.NewPath(tmp))
	return c, tmp
}

func fileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(data), want), "file %s = %q, want contains %q", path, string(data), want)
}

func TestName(t *testing.T) {
	c := NewCollector(&fakeClient{}, api.NewLogger(&bytes.Buffer{}), api.NewPath("x"))
	assert.Equal(t, "MonitoringCollector", c.Name())
}

func TestCollectWritesOperandAndOperatorFiles(t *testing.T) {
	fc := &fakeClient{
		pods: map[string][]corev1.Pod{
			"|" + managedByLabel: {runningPod("operand-managed")},
			"|" + partOfLabel:    {runningPod("operand-partof")},
			"|" + nameLabel:      {runningPod("operator-0")},
		},
	}
	c, tmp := newTestCollector(t, fc)

	err := c.Collect(context.Background())
	assert.NilError(t, err)

	base := filepath.Join(tmp, "monitoring", "observability-operator")
	// operants.yaml contains both managed-by and part-of results, separated by a YAML doc marker.
	fileContains(t, filepath.Join(base, "operants.yaml"), "operand-managed")
	fileContains(t, filepath.Join(base, "operants.yaml"), "operand-partof")
	fileContains(t, filepath.Join(base, "operants.yaml"), "---")
	fileContains(t, filepath.Join(base, "operator.yaml"), "operator-0")
}

func TestCollectGathersPrometheusAndAlertmanager(t *testing.T) {
	const ns, name = "team-a", "stack-a"
	promSel := fmt.Sprintf("%s|app.kubernetes.io/part-of=%s,app.kubernetes.io/component=prometheus", ns, name)
	amSel := fmt.Sprintf("%s|app.kubernetes.io/part-of=%s,app.kubernetes.io/component=alertmanager", ns, name)

	fc := &fakeClient{
		resources: []client.ResourceRef{{Namespace: ns, Name: name}},
		pods: map[string][]corev1.Pod{
			promSel: {runningPod("prometheus-stack-a-0"), runningPod("prometheus-stack-a-1")},
			amSel:   {runningPod("alertmanager-stack-a-0")},
		},
		proxyFunc: func(call proxyCall) (string, error) {
			return fmt.Sprintf("output-from-%s", call.pod), nil
		},
	}
	c, tmp := newTestCollector(t, fc)

	err := c.Collect(context.Background())
	assert.NilError(t, err)

	promDir := filepath.Join(tmp, "monitoring", "observability-operator", ns, name, "prometheus")
	amDir := filepath.Join(tmp, "monitoring", "observability-operator", ns, name, "alertmanager")

	// First-ready-replica queries.
	fileContains(t, filepath.Join(promDir, "alertmanagers.json"), "output-from-prometheus-stack-a-0")
	fileContains(t, filepath.Join(promDir, "rules.json"), "output-from-prometheus-stack-a-0")
	fileContains(t, filepath.Join(promDir, "status", "config.json"), "output-from-prometheus-stack-a-0")
	fileContains(t, filepath.Join(promDir, "status", "flags.json"), "output-from-prometheus-stack-a-0")

	// Per-replica queries produce a directory per pod.
	for _, replica := range []string{"prometheus-stack-a-0", "prometheus-stack-a-1"} {
		fileContains(t, filepath.Join(promDir, replica, "status", "runtimeinfo.json"), "output-from-"+replica)
		fileContains(t, filepath.Join(promDir, replica, "targets-active.json"), "output-from-"+replica)
		fileContains(t, filepath.Join(promDir, replica, "status", "tsdb.json"), "output-from-"+replica)
	}

	// Alertmanager status.
	fileContains(t, filepath.Join(amDir, "status.json"), "output-from-alertmanager-stack-a-0")

	// Alertmanager proxy requests must target the stack's namespace and port.
	var amCall *proxyCall
	for i := range fc.proxyCalls {
		if fc.proxyCalls[i].port == amPort {
			amCall = &fc.proxyCalls[i]
			break
		}
	}
	assert.Assert(t, amCall != nil, "expected an alertmanager proxy call")
	assert.Equal(t, ns, amCall.namespace)
	assert.Equal(t, "/api/v2/status", amCall.endpoint)
}

func TestGetWritesStderrOnError(t *testing.T) {
	const ns, name = "team-a", "stack-a"
	promSel := fmt.Sprintf("%s|app.kubernetes.io/part-of=%s,app.kubernetes.io/component=prometheus", ns, name)

	fc := &fakeClient{
		resources: []client.ResourceRef{{Namespace: ns, Name: name}},
		pods: map[string][]corev1.Pod{
			promSel: {runningPod("prometheus-stack-a-0")},
		},
		proxyFunc: func(_ proxyCall) (string, error) {
			return "", fmt.Errorf("boom")
		},
	}
	c, tmp := newTestCollector(t, fc)

	err := c.Collect(context.Background())
	assert.NilError(t, err)

	promDir := filepath.Join(tmp, "monitoring", "observability-operator", ns, name, "prometheus")
	// stderr file written with the captured stderr.
	fileContains(t, filepath.Join(promDir, "rules.stderr"), "boom")
}

func TestGatherStackSkipsWhenNoReadyPods(t *testing.T) {
	const ns, name = "team-a", "stack-a"
	promSel := fmt.Sprintf("%s|app.kubernetes.io/part-of=%s,app.kubernetes.io/component=prometheus", ns, name)
	fc := &fakeClient{
		resources: []client.ResourceRef{{Namespace: ns, Name: name}},
		pods: map[string][]corev1.Pod{
			promSel: {
				{ObjectMeta: metav1.ObjectMeta{Name: "pending"}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
				{ObjectMeta: metav1.ObjectMeta{Name: "no-statuses"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
			},
		},
	}
	c, _ := newTestCollector(t, fc)

	err := c.Collect(context.Background())
	assert.NilError(t, err)
	// No proxy calls should have happened.
	assert.Equal(t, 0, len(fc.proxyCalls))
}

func TestFirstReadyPod(t *testing.T) {
	notRunning := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pending"},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	notReady := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "running-notready"},
		Status: corev1.PodStatus{
			Phase:             corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{Ready: false}},
		},
	}
	noStatuses := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "running-without-statuses"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	ready := runningPod("running-ready")

	assert.Equal(t, "", firstReadyPod(nil))
	assert.Equal(t, "", firstReadyPod([]corev1.Pod{notRunning, notReady, noStatuses}))
	assert.Equal(t, "running-ready", firstReadyPod([]corev1.Pod{notRunning, notReady, noStatuses, ready}))
}

// TestProxyRequestShape verifies the prometheus queries use the expected paths.
func TestProxyRequestShape(t *testing.T) {
	const ns, name = "team-a", "stack-a"
	promSel := fmt.Sprintf("%s|app.kubernetes.io/part-of=%s,app.kubernetes.io/component=prometheus", ns, name)

	fc := &fakeClient{
		resources: []client.ResourceRef{{Namespace: ns, Name: name}},
		pods: map[string][]corev1.Pod{
			promSel: {runningPod("prometheus-stack-a-0")},
		},
	}
	c, _ := newTestCollector(t, fc)
	assert.NilError(t, c.Collect(context.Background()))

	var promURLs []string
	for _, call := range fc.proxyCalls {
		if call.port == promPort {
			promURLs = append(promURLs, call.endpoint)
		}
	}
	sort.Strings(promURLs)

	for _, want := range []string{
		"/api/v1/alertmanagers",
		"/api/v1/rules",
		"/api/v1/status/config",
		"/api/v1/status/flags",
		"/api/v1/status/runtimeinfo",
		"/api/v1/targets?state=active",
		"/api/v1/status/tsdb",
	} {
		found := false
		for _, got := range promURLs {
			if strings.Contains(got, want) {
				found = true
				break
			}
		}
		assert.Assert(t, found, "expected a prometheus query for %q, got %v", want, promURLs)
	}
}
