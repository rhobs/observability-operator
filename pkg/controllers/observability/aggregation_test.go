package observability

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
)

func TestAggregateSharedIsCapabilityIndependent(t *testing.T) {
	definition := capability.Definition{
		Name: "future-capability",
		Shared: func(instance *obsv1alpha1.ObservabilityInstaller) capability.Plan {
			objects := []client.Object{
				&corev1.ConfigMap{TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"}, ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "first"}},
				&corev1.ConfigMap{TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"}, ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "second"}},
			}
			enabled := instance.Name == "enabled"
			plan := capability.Plan{Operators: capability.DesiredOperators(enabled, capability.OperatorRequirement{Name: "future", Subscription: capability.Subscription(capability.OperatorConfig{PackageName: "future"})})}
			plan.Owned = objects
			if enabled {
				plan.Desired = objects
			}
			return plan
		},
	}
	instances := []obsv1alpha1.ObservabilityInstaller{
		{ObjectMeta: metav1.ObjectMeta{Name: "enabled"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "disabled"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "enabled"}},
	}
	plan := aggregateShared([]capability.Definition{definition}, instances)
	require.Len(t, plan.Operators, 1)
	require.True(t, plan.Operators[0].Desired)
	require.Len(t, plan.Owned, 2)
	require.Len(t, plan.Desired, 2, "identity includes namespace and duplicate requests are merged")

	plan = aggregateShared([]capability.Definition{definition}, nil)
	require.Len(t, plan.Owned, 2)
	require.Empty(t, plan.Desired)
	require.False(t, plan.Operators[0].Desired)
}
