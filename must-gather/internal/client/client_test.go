package client

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/rhobs/observability-operator/must-gather/internal/api"
)

func testClient(t *testing.T, objs ...runtime.Object) *Client {
	t.Helper()
	return &Client{
		KubernetesClient: k8sfake.NewSimpleClientset(objs...),
		logger:           api.NewLogger(newDiscard()),
	}
}

func newDiscard() *discardWriter { return &discardWriter{} }

type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func pod(namespace, name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
	}
}

func TestListPodsFiltersByLabelAndNamespace(t *testing.T) {
	c := testClient(t,
		pod("ns1", "a", map[string]string{"app": "x"}),
		pod("ns1", "b", map[string]string{"app": "y"}),
		pod("ns2", "c", map[string]string{"app": "x"}),
	)

	// All namespaces, label app=x
	all, err := c.ListPods(context.Background(), "", "app=x")
	assert.NilError(t, err)
	assert.Equal(t, 2, len(all.Items))

	// Single namespace, label app=x
	scoped, err := c.ListPods(context.Background(), "ns1", "app=x")
	assert.NilError(t, err)
	assert.Equal(t, 1, len(scoped.Items))
	assert.Equal(t, "a", scoped.Items[0].Name)
}

func TestListResources(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "monitoring.rhobs", Version: "v1alpha1", Resource: "monitoringstacks"}

	scheme := runtime.NewScheme()
	ms := &unstructured.Unstructured{}
	ms.SetGroupVersionKind(schema.GroupVersionKind{Group: "monitoring.rhobs", Version: "v1alpha1", Kind: "MonitoringStack"})
	ms.SetNamespace("team-a")
	ms.SetName("stack-a")

	ms2 := ms.DeepCopy()
	ms2.SetNamespace("team-b")
	ms2.SetName("stack-b")

	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme,
		map[schema.GroupVersionResource]string{gvr: "MonitoringStackList"},
		ms, ms2,
	)

	c := &Client{
		DynamicClient: dynClient,
		logger:        api.NewLogger(newDiscard()),
	}

	refs, err := c.ListResources(context.Background(), gvr)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(refs))

	got := map[string]string{}
	for _, r := range refs {
		got[r.Name] = r.Namespace
	}
	assert.Equal(t, "team-a", got["stack-a"])
	assert.Equal(t, "team-b", got["stack-b"])
}

func TestMarshalYAML(t *testing.T) {
	p := pod("ns", "name", map[string]string{"k": "v"})
	data, err := MarshalYAML(p)
	assert.NilError(t, err)
	assert.Assert(t, len(data) > 0)
}
