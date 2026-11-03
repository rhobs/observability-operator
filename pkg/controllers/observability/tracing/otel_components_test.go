package tracing

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestOTelCollectorRBACNames(t *testing.T) {
	instances := []*obsv1alpha1.ObservabilityInstaller{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "ns1", Name: "obs-foo"}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "ns1-obs", Name: "foo"}},
	}
	for _, build := range []struct {
		name string
		get  func(*obsv1alpha1.ObservabilityInstaller) (string, string, string)
	}{
		{"components", func(i *obsv1alpha1.ObservabilityInstaller) (string, string, string) {
			role, binding := otelCollectorComponentsRBAC(i)
			return role.Name, binding.Name, binding.RoleRef.Name
		}},
		{"tempo", func(i *obsv1alpha1.ObservabilityInstaller) (string, string, string) {
			role, binding := otelCollectorTempoRBAC(i)
			return role.Name, binding.Name, binding.RoleRef.Name
		}},
	} {
		t.Run(build.name, func(t *testing.T) {
			first, binding, ref := build.get(instances[0])
			if first != binding || first != ref {
				t.Fatalf("role %q, binding %q, roleRef %q differ", first, binding, ref)
			}
			second, binding, ref := build.get(instances[1])
			if second != binding || second != ref || first == second {
				t.Fatalf("names collide or references differ: %q, %q, %q, %q", first, second, binding, ref)
			}
			again, _, _ := build.get(instances[0])
			if first != again {
				t.Fatalf("non-deterministic name: %q != %q", first, again)
			}
		})
	}
}
