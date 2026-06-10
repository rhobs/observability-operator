package util

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ResourceLabel = "app.kubernetes.io/managed-by"
	OpName        = "observability-operator"
)

// AddCommonLabels sets standard observability-operator labels on obj.
// It returns an error if name exceeds 63 characters, the Kubernetes label value limit.
func AddCommonLabels(obj client.Object, name string) (client.Object, error) {
	if len(name) > 63 {
		return nil, fmt.Errorf("resource name %q exceeds the 63-character limit for Kubernetes label values", name)
	}
	labels := obj.GetLabels()
	want := map[string]string{
		"app.kubernetes.io/part-of": name,
		"app.kubernetes.io/name":    obj.GetName(),
		ResourceLabel:               OpName,
	}
	if labels == nil {
		obj.SetLabels(want)
		return obj, nil
	}
	for name, val := range want {
		if _, ok := labels[name]; !ok {
			labels[name] = val
		}
	}
	return obj, nil
}
