package reconciler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestClientObjectApplyConfig_MarshalJSON(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{"key": "value"},
	}
	cm.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})

	ac := &clientObjectApplyConfig{obj: cm}

	data, err := json.Marshal(ac)
	require.NoError(t, err)

	// The serialised form must contain the object's fields.
	require.Contains(t, string(data), `"test-cm"`)
	require.Contains(t, string(data), `"key"`)
}

func TestClientObjectApplyConfig_UnmarshalJSON(t *testing.T) {
	// Start with a minimal ConfigMap; unmarshal should populate it from
	// a JSON payload that resembles an API-server response.
	cm := &corev1.ConfigMap{}
	ac := &clientObjectApplyConfig{obj: cm}

	response := `{
		"apiVersion": "v1",
		"kind": "ConfigMap",
		"metadata": {
			"name": "test-cm",
			"namespace": "default",
			"resourceVersion": "12345"
		},
		"data": {"answer": "42"}
	}`

	err := json.Unmarshal([]byte(response), ac)
	require.NoError(t, err)

	require.Equal(t, "test-cm", cm.Name)
	require.Equal(t, "default", cm.Namespace)
	require.Equal(t, "12345", cm.ResourceVersion)
	require.Equal(t, map[string]string{"answer": "42"}, cm.Data)
}

func TestClientObjectApplyConfig_RoundTrip(t *testing.T) {
	original := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "round-trip",
			Namespace:       "ns",
			ResourceVersion: "999",
		},
		Data: map[string]string{"foo": "bar"},
	}
	original.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})

	ac := &clientObjectApplyConfig{obj: original}

	data, err := json.Marshal(ac)
	require.NoError(t, err)

	target := &corev1.ConfigMap{}
	targetAC := &clientObjectApplyConfig{obj: target}
	err = json.Unmarshal(data, targetAC)
	require.NoError(t, err)

	require.Equal(t, original.Name, target.Name)
	require.Equal(t, original.Namespace, target.Namespace)
	require.Equal(t, original.ResourceVersion, target.ResourceVersion)
	require.Equal(t, original.Data, target.Data)
}

func TestClientObjectApplyConfig_Getters(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "getter-test",
			Namespace: "test-ns",
		},
	}
	cm.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})

	ac := &clientObjectApplyConfig{obj: cm}

	require.Equal(t, "getter-test", *ac.GetName())
	require.Equal(t, "test-ns", *ac.GetNamespace())
	require.Equal(t, "ConfigMap", *ac.GetKind())
	require.Equal(t, "v1", *ac.GetAPIVersion())
}
