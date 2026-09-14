package client

import (
	"bytes"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	"github.com/rhobs/observability-operator/must-gather/internal/api"
)

// Client wraps the Kubernetes client functionality needed for must-gather.
type Client struct {
	KubernetesClient kubernetes.Interface
	DynamicClient    dynamic.Interface
	config           *rest.Config
	logger           api.Logger
}

// NewClient creates a new Kubernetes client. It uses the in-cluster
// configuration when running inside a pod (as with `oc adm must-gather`) and
// falls back to the local kubeconfig otherwise.
func NewClient(logger api.Logger) (*Client, error) {
	config, err := ctrlconfig.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kube config: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		KubernetesClient: k8sClient,
		DynamicClient:    dynamicClient,
		config:           config,
		logger:           logger,
	}, nil
}

// ListPods returns the pods in the given namespace matching the label selector.
// Passing an empty namespace lists pods across all namespaces.
func (c *Client) ListPods(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	podList, err := c.KubernetesClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	return podList, nil
}

// ResourceRef is a minimal namespace/name reference to a listed resource.
type ResourceRef struct {
	Namespace string
	Name      string
}

// ListResources lists resources of the given GroupVersionResource across all
// namespaces using the dynamic client.
func (c *Client) ListResources(ctx context.Context, gvr schema.GroupVersionResource) ([]ResourceRef, error) {
	list, err := c.DynamicClient.Resource(gvr).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", gvr.Resource, err)
	}

	items := make([]ResourceRef, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, ResourceRef{
			Namespace: item.GetNamespace(),
			Name:      item.GetName(),
		})
	}
	return items, nil
}

// PodExec executes a command in a pod container and returns its stdout and
// stderr separately.
func (c *Client) PodExec(ctx context.Context, namespace, pod, container string, command []string) (stdout, stderr string, err error) {
	req := c.KubernetesClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("failed to create executor: %w", err)
	}

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: stdoutBuf,
		Stderr: stderrBuf,
	})
	if err != nil {
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("exec failed: %w", err)
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
}

// MarshalYAML converts an object to YAML bytes.
func MarshalYAML(obj interface{}) ([]byte, error) {
	return yaml.Marshal(obj)
}
