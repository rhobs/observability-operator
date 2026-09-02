package util

import (
	"encoding/json"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ResourceLabel = "app.kubernetes.io/managed-by"
	OpName        = "observability-operator"
)

// Identifier is a global identifying string of the form GVK/name.namespace
func Identifier(obj client.Object) string {
	return fmt.Sprintf("%s/%s.%s", obj.GetObjectKind().GroupVersionKind().String(), obj.GetName(), obj.GetNamespace())
}

func AddCommonLabels(obj client.Object, name string) client.Object {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	want := map[string]string{
		"app.kubernetes.io/part-of": name,
		"app.kubernetes.io/name":    obj.GetName(),
		ResourceLabel:               OpName,
	}
	for k, v := range want {
		if _, ok := labels[k]; !ok {
			labels[k] = v
		}
	}
	obj.SetLabels(labels)
	return obj
}

// MarshalCopy copies a client.Object (structured or unstructured) into dst
// via JSON round-trip.
func MarshalCopy(dst, src client.Object) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// CompareObjects orders Namespace objects first, then by namespace, API group,
// kind, then name. Callers must populate TypeMeta so Group and Kind are available.
func CompareObjects(a, b client.Object) int {
	aNS := a.GetObjectKind().GroupVersionKind().Kind == "Namespace"
	bNS := b.GetObjectKind().GroupVersionKind().Kind == "Namespace"
	if aNS != bNS {
		if aNS {
			return -1
		}
		return 1
	}
	if c := strings.Compare(a.GetNamespace(), b.GetNamespace()); c != 0 {
		return c
	}
	if c := strings.Compare(a.GetObjectKind().GroupVersionKind().Group, b.GetObjectKind().GroupVersionKind().Group); c != 0 {
		return c
	}
	if c := strings.Compare(a.GetObjectKind().GroupVersionKind().Kind, b.GetObjectKind().GroupVersionKind().Kind); c != 0 {
		return c
	}
	return strings.Compare(a.GetName(), b.GetName())
}
