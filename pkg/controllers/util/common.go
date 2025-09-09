package util

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ResourceLabel = "app.kubernetes.io/managed-by"
	OpName        = "observability-operator"
)

func AddCommonLabels(obj client.Object, name string) client.Object {
	labels := obj.GetLabels()
	want := map[string]string{
		"app.kubernetes.io/part-of": name,
		"app.kubernetes.io/name":    obj.GetName(),
		ResourceLabel:               OpName,
	}
	if labels == nil {
		obj.SetLabels(want)
		return obj
	}
	for name, val := range want {
		if _, ok := labels[name]; !ok {
			labels[name] = val
		}
	}
	return obj
}
