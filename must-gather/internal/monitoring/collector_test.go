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

// execCall records the arguments of a PodExec invocation.
type execCall struct {
	namespace string
	pod       string
	container string
	command   []string
}

// fakeClient is a test double for the monitoring.Client interface.
type fakeClient struct {
	// pods returns pods keyed by "<namespace>|<labelSelector>".
	pods map[string][]corev1.Pod
	// resources are the MonitoringStacks returned by ListResources.
	resources []client.ResourceRef

	execCalls []execCall
	// execFunc lets a test customise exec output/errors.
	execFunc func(call execCall) (stdout, stderr string, err error)
}

func (f *fakeClient) ListPods(_ context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	key := namespace + "|" + labelSelector
	return &corev1.PodList{Items: f.pods[key]}, nil
}

func (f *fakeClient) ListResources(_ context.Context, _ schema.GroupVersionResource) ([]client.ResourceRef, error) {
	return f.resources, nil
}

func (f *fakeClient) PodExec(_ context.Context, namespace, pod, container string, command []string) (string, string, error) {
	call := execCall{namespace: namespace, pod: pod, container: container, command: command}
	f.execCalls = append(f.execCalls, call)
	if f.execFunc != nil {
		return f.execFunc(call)
	}
	return "{}", "", nil
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

func TestCollectWritesOperantAndOperatorFiles(t *testing.T) {
	fc := &fakeClient{
		pods: map[string][]corev1.Pod{
			"|" + managedByLabel: {runningPod("operant-managed")},
			"|" + partOfLabel:    {runningPod("operant-partof")},
			"|" + nameLabel:      {runningPod("operator-0")},
		},
	}
	c, tmp := newTestCollector(t, fc)

	err := c.Collect(context.Background())
	assert.NilError(t, err)

	base := filepath.Join(tmp, "monitoring", "observability-operator")
	// operants.yaml contains both managed-by and part-of results, separated by a YAML doc marker.
	fileContains(t, filepath.Join(base, "operants.yaml"), "operant-managed")
	fileContains(t, filepath.Join(base, "operants.yaml"), "operant-partof")
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
		execFunc: func(call execCall) (string, string, error) {
			return fmt.Sprintf("output-from-%s", call.pod), "", nil
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

	// Alertmanager exec must target the stack's namespace and container.
	var amCall *execCall
	for i := range fc.execCalls {
		if fc.execCalls[i].container == amContainer {
			amCall = &fc.execCalls[i]
			break
		}
	}
	assert.Assert(t, amCall != nil, "expected an alertmanager exec call")
	assert.Equal(t, ns, amCall.namespace)
	assert.Assert(t, strings.Contains(strings.Join(amCall.command, " "), "9093/api/v2/status"))
}

func TestCurlWritesStderrOnError(t *testing.T) {
	const ns, name = "team-a", "stack-a"
	promSel := fmt.Sprintf("%s|app.kubernetes.io/part-of=%s,app.kubernetes.io/component=prometheus", ns, name)

	fc := &fakeClient{
		resources: []client.ResourceRef{{Namespace: ns, Name: name}},
		pods: map[string][]corev1.Pod{
			promSel: {runningPod("prometheus-stack-a-0")},
		},
		execFunc: func(_ execCall) (string, string, error) {
			return "", "boom", fmt.Errorf("exec failed")
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
	fc := &fakeClient{
		resources: []client.ResourceRef{{Namespace: ns, Name: name}},
		// No pods registered -> nothing ready.
		pods: map[string][]corev1.Pod{},
	}
	c, _ := newTestCollector(t, fc)

	err := c.Collect(context.Background())
	assert.NilError(t, err)
	// No exec calls should have happened.
	assert.Equal(t, 0, len(fc.execCalls))
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
	ready := runningPod("running-ready")

	assert.Equal(t, "", firstReadyPod(nil))
	assert.Equal(t, "", firstReadyPod([]corev1.Pod{notRunning, notReady}))
	assert.Equal(t, "running-ready", firstReadyPod([]corev1.Pod{notRunning, notReady, ready}))
}

// TestExecCommandShape verifies the prometheus queries use the expected URLs.
func TestExecCommandShape(t *testing.T) {
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
	for _, call := range fc.execCalls {
		if call.container == promContainer {
			promURLs = append(promURLs, strings.Join(call.command, " "))
		}
	}
	sort.Strings(promURLs)

	for _, want := range []string{
		"9090/api/v1/alertmanagers",
		"9090/api/v1/rules",
		"9090/api/v1/status/config",
		"9090/api/v1/status/flags",
		"9090/api/v1/status/runtimeinfo",
		"9090/api/v1/targets?state=active",
		"9090/api/v1/status/tsdb",
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
