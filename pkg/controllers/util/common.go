package util

import (
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ResourceLabel = "app.kubernetes.io/managed-by"
	PartOfLabel   = "app.kubernetes.io/part-of"
	OpName        = "observability-operator"
)

func AddCommonLabels(obj client.Object, name string) client.Object {
	labels := obj.GetLabels()
	want := map[string]string{
		PartOfLabel:              name,
		"app.kubernetes.io/name": obj.GetName(),
		ResourceLabel:            OpName,
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

// IsOperatorManaged reports whether obj was created or adopted by COO. It is
// used to guard updates and deletions of resources whose names are derivable
// from an installer: a user-created resource that happens to share such a name
// must never be adopted or deleted.
func IsOperatorManaged(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels[ResourceLabel] == OpName || labels[PartOfLabel] == OpName {
		return true
	}
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == "ObservabilityInstaller" && strings.HasPrefix(ref.APIVersion, "observability.openshift.io/") {
			return true
		}
	}
	return false
}
