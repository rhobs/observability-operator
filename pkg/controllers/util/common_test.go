package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestAddCommonLabels(t *testing.T) {
	tests := []struct {
		name       string
		labels     map[string]string
		wantPartOf string
	}{
		{
			name:       "nil labels",
			labels:     nil,
			wantPartOf: "my-owner",
		},
		{
			name:       "empty labels",
			labels:     map[string]string{},
			wantPartOf: "my-owner",
		},
		{
			name:       "existing labels preserved",
			labels:     map[string]string{"app.kubernetes.io/part-of": "custom", "extra": "val"},
			wantPartOf: "custom",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Labels: tc.labels}}
			result := AddCommonLabels(obj, "my-owner")
			labels := result.GetLabels()
			require.Equal(t, tc.wantPartOf, labels["app.kubernetes.io/part-of"])
			require.Equal(t, "cm", labels["app.kubernetes.io/name"])
			require.Equal(t, OpName, labels[ResourceLabel])
			if tc.labels != nil {
				for k, v := range tc.labels {
					require.Equal(t, v, labels[k])
				}
			}
		})
	}
}

func TestMarshalCopy(t *testing.T) {
	src := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: "ns"},
		Data:       map[string]string{"key": "value"},
	}
	dst := &corev1.ConfigMap{}
	require.NoError(t, MarshalCopy(dst, src))
	require.Equal(t, src.Name, dst.Name)
	require.Equal(t, src.Namespace, dst.Namespace)
	require.Equal(t, src.Data, dst.Data)
}

func TestCompareObjects(t *testing.T) {
	obj := func(ns, group, kind, name string) *corev1.ConfigMap {
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
		cm.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Kind: kind})
		return cm
	}

	tests := []struct {
		name string
		a, b *corev1.ConfigMap
		want int
	}{
		{"same", obj("ns", "g", "Kind", "a"), obj("ns", "g", "Kind", "a"), 0},
		{"namespace first", obj("", "", "Namespace", "x"), obj("", "", "ConfigMap", "x"), -1},
		{"namespace before namespaced", obj("", "", "Namespace", "ns"), obj("ns", "", "ConfigMap", "x"), -1},
		{"namespace differs", obj("a-ns", "", "Kind", "x"), obj("b-ns", "", "Kind", "x"), -1},
		{"group differs", obj("ns", "a.io", "Kind", "x"), obj("ns", "b.io", "Kind", "x"), -1},
		{"kind differs", obj("ns", "g", "A", "x"), obj("ns", "g", "B", "x"), -1},
		{"name differs", obj("ns", "g", "Kind", "a"), obj("ns", "g", "Kind", "b"), -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CompareObjects(tc.a, tc.b)
			switch {
			case tc.want < 0:
				require.Less(t, got, 0)
			case tc.want > 0:
				require.Greater(t, got, 0)
			default:
				require.Equal(t, 0, got)
			}
		})
	}
}
