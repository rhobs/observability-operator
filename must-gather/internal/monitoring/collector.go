package monitoring

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/rhobs/observability-operator/must-gather/internal/api"
	"github.com/rhobs/observability-operator/must-gather/internal/client"
)

const (
	// managedByLabel selects resources managed by the operator.
	managedByLabel = "app.kubernetes.io/managed-by=observability-operator"
	// partOfLabel selects resources that are part of the operator.
	partOfLabel = "app.kubernetes.io/part-of=observability-operator"
	// nameLabel selects the operator pods themselves.
	nameLabel = "app.kubernetes.io/name=observability-operator"

	promPort = "9090"
	amPort   = "9093"
)

// monitoringStackGVR is the GroupVersionResource for MonitoringStack.
var monitoringStackGVR = schema.GroupVersionResource{
	Group:    "monitoring.rhobs",
	Version:  "v1alpha1",
	Resource: "monitoringstacks",
}

// Client is the subset of the Kubernetes client functionality required by the
// monitoring collector. It is satisfied by *client.Client and allows tests to
// inject a fake implementation.
type Client interface {
	ListPods(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error)
	ListResources(ctx context.Context, gvr schema.GroupVersionResource) ([]client.ResourceRef, error)
	PodProxyGet(ctx context.Context, namespace, pod, port, endpoint string) (string, error)
}

// Collector gathers observability-operator specific monitoring information such
// as operator/operand pods and Prometheus/Alertmanager runtime state.
type Collector struct {
	client  Client
	logger  api.Logger
	destDir api.Path
}

// NewCollector creates a new monitoring collector. The destDir passed in is the
// root must-gather directory; the collector writes below
// monitoring/observability-operator.
func NewCollector(c Client, logger api.Logger, destDir api.Path) *Collector {
	return &Collector{
		client:  c,
		logger:  logger,
		destDir: destDir.Add("monitoring", "observability-operator"),
	}
}

// Name returns the name of this collector.
func (m *Collector) Name() string {
	return "MonitoringCollector"
}

// Collect performs the collection of observability-operator monitoring resources.
func (m *Collector) Collect(ctx context.Context) error {
	defer m.logger.Begin("gathering observability-operator monitoring data ...")()

	if err := m.destDir.MkdirAll(); err != nil {
		return err
	}

	// Gather operator and operand pods (best-effort).
	m.gatherOperands(ctx)

	// Discover MonitoringStacks across all namespaces.
	stacks, err := m.client.ListResources(ctx, monitoringStackGVR)
	if err != nil {
		m.logger.Warn("Failed to list MonitoringStacks: %v", err)
		return nil
	}
	m.logger.Info("Found %d MonitoringStack(s)", len(stacks))

	for _, stack := range stacks {
		m.gatherStack(ctx, stack.Namespace, stack.Name)
	}

	return nil
}

// gatherOperands collects the operator deployment pods and all operand pods.
func (m *Collector) gatherOperands(ctx context.Context) {
	// Operand pods: managed-by and part-of observability-operator.
	var operands []byte
	for _, selector := range []string{managedByLabel, partOfLabel} {
		pods, err := m.client.ListPods(ctx, "", selector)
		if err != nil {
			m.logger.Warn("Failed to list pods (%s): %v", selector, err)
			continue
		}
		data, err := client.MarshalYAML(pods)
		if err != nil {
			m.logger.Warn("Failed to marshal pods (%s): %v", selector, err)
			continue
		}
		if len(operands) > 0 {
			operands = append(operands, []byte("---\n")...)
		}
		operands = append(operands, data...)
	}
	if err := m.destDir.Add("operants.yaml").WriteFile(operands); err != nil {
		m.logger.Warn("Failed to write operants.yaml: %v", err)
	}

	// Operator pods.
	operatorPods, err := m.client.ListPods(ctx, "", nameLabel)
	if err != nil {
		m.logger.Warn("Failed to list operator pods: %v", err)
		return
	}
	data, err := client.MarshalYAML(operatorPods)
	if err != nil {
		m.logger.Warn("Failed to marshal operator pods: %v", err)
		return
	}
	if err := m.destDir.Add("operator.yaml").WriteFile(data); err != nil {
		m.logger.Warn("Failed to write operator.yaml: %v", err)
	}
}

// gatherStack collects Prometheus and Alertmanager state for a single stack.
func (m *Collector) gatherStack(ctx context.Context, ns, name string) {
	m.logger.Info("Gathering MonitoringStack %s/%s", ns, name)

	// Prometheus data served from the first ready replica.
	m.promGet(ctx, "alertmanagers", ns, name)
	m.promGet(ctx, "rules", ns, name)
	m.promGet(ctx, "status/config", ns, name)
	m.promGet(ctx, "status/flags", ns, name)

	// Per-replica state.
	m.promGetFromReplicas(ctx, "status/runtimeinfo", ns, name, "status/runtimeinfo")
	m.promGetFromReplicas(ctx, "targets?state=active", ns, name, "targets-active")
	m.promGetFromReplicas(ctx, "status/tsdb", ns, name, "status/tsdb")

	// Alertmanager state.
	m.alertmanagerGet(ctx, "status", ns, name)
}

// promPods returns the Prometheus pods that belong to the given stack.
func (m *Collector) promPods(ctx context.Context, ns, name string) []corev1.Pod {
	selector := fmt.Sprintf("app.kubernetes.io/part-of=%s,app.kubernetes.io/component=prometheus", name)
	pods, err := m.client.ListPods(ctx, ns, selector)
	if err != nil {
		m.logger.Warn("Failed to list Prometheus pods for %s/%s: %v", ns, name, err)
		return nil
	}
	return pods.Items
}

// alertmanagerPods returns the Alertmanager pods that belong to the given stack.
func (m *Collector) alertmanagerPods(ctx context.Context, ns, name string) []corev1.Pod {
	selector := fmt.Sprintf("app.kubernetes.io/part-of=%s,app.kubernetes.io/component=alertmanager", name)
	pods, err := m.client.ListPods(ctx, ns, selector)
	if err != nil {
		m.logger.Warn("Failed to list Alertmanager pods for %s/%s: %v", ns, name, err)
		return nil
	}
	return pods.Items
}

// promGet queries the Prometheus API on the first ready replica and stores the
// result under <ns>/<name>/prometheus/<object>.json.
func (m *Collector) promGet(ctx context.Context, object, ns, name string) {
	pod := firstReadyPod(m.promPods(ctx, ns, name))
	if pod == "" {
		m.logger.Warn("No ready Prometheus pod found for %s/%s", ns, name)
		return
	}

	resultPath := m.destDir.Add(ns, name, "prometheus", object)
	m.get(ctx, ns, pod, promPort, "v1", object, resultPath)
}

// promGetFromReplicas queries the Prometheus API on every replica of a stack and
// stores the result under <ns>/<name>/prometheus/<pod>/<path>.json.
func (m *Collector) promGetFromReplicas(ctx context.Context, object, ns, name, path string) {
	pods := m.promPods(ctx, ns, name)
	if len(pods) == 0 {
		m.logger.Warn("No Prometheus pods found for %s/%s", ns, name)
		return
	}
	for _, pod := range pods {
		if !podReady(pod) {
			continue
		}
		resultPath := m.destDir.Add(ns, name, "prometheus", pod.Name, path)
		m.get(ctx, ns, pod.Name, promPort, "v1", object, resultPath)
	}
}

// alertmanagerGet queries the Alertmanager API on the first ready replica and
// stores the result under <ns>/<name>/alertmanager/<object>.json.
func (m *Collector) alertmanagerGet(ctx context.Context, object, ns, name string) {
	pod := firstReadyPod(m.alertmanagerPods(ctx, ns, name))
	if pod == "" {
		m.logger.Warn("No ready Alertmanager pod found for %s/%s", ns, name)
		return
	}

	resultPath := m.destDir.Add(ns, name, "alertmanager", object)
	m.get(ctx, ns, pod, amPort, "v2", object, resultPath)
}

// get requests the pod's API through the Kubernetes proxy and writes stdout to
// <resultPath>.json and any stderr/error to <resultPath>.stderr.
func (m *Collector) get(ctx context.Context, ns, pod, port, apiVersion, object string, resultPath api.Path) {
	m.logger.Info("Getting %s from %s", object, pod)

	endpoint := fmt.Sprintf("/api/%s/%s", apiVersion, object)
	stdout, err := m.client.PodProxyGet(ctx, ns, pod, port, endpoint)
	stderr := ""
	if err != nil {
		m.logger.Warn("Failed to get %s from %s: %v", object, pod, err)
		stderr = err.Error()
	}

	if writeErr := resultPath.WithSuffix(".json").WriteFile([]byte(stdout)); writeErr != nil {
		m.logger.Warn("Failed to write %s.json: %v", object, writeErr)
	}
	if stderr != "" {
		if writeErr := resultPath.WithSuffix(".stderr").WriteFile([]byte(stderr)); writeErr != nil {
			m.logger.Warn("Failed to write %s.stderr: %v", object, writeErr)
		}
	}
}

// firstReadyPod returns the name of the first Running pod with all containers
// ready, or an empty string when none is found.
func firstReadyPod(pods []corev1.Pod) string {
	for _, pod := range pods {
		if podReady(pod) {
			return pod.Name
		}
	}
	return ""
}

func podReady(pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning || len(pod.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}
	return true
}
