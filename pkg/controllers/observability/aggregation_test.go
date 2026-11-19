package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
)

// fakeCapability is enabled for installers named enabledFor.
type fakeCapability struct {
	name       string
	operator   string
	shared     []client.Object
	enabledFor string
}

func (f fakeCapability) Name() string { return f.name }
func (f fakeCapability) Inventory(*obsv1alpha1.ObservabilityInstaller) []client.Object {
	return nil
}
func (f fakeCapability) Desired(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) ([]client.Object, error) {
	return nil, nil
}
func (f fakeCapability) SharedInventory() []client.Object { return f.shared }
func (f fakeCapability) SharedDesired(instance *obsv1alpha1.ObservabilityInstaller) []client.Object {
	if instance.Name == f.enabledFor {
		return f.shared
	}
	return nil
}
func (f fakeCapability) Operators() []capability.OperatorRequirement {
	return []capability.OperatorRequirement{{Name: f.operator, Subscription: capability.Subscription(capability.OperatorConfig{PackageName: f.operator})}}
}
func (f fakeCapability) OperatorsDesired(instance *obsv1alpha1.ObservabilityInstaller) bool {
	return instance.Name == f.enabledFor
}
func (f fakeCapability) UpdateStatus(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) capability.StatusResult {
	return capability.StatusResult{}
}
func (f fakeCapability) WatchTypes() []client.Object { return nil }

func configMap(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"}, ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: namespace}}
}

func TestAggregateSharedIsCapabilityIndependent(t *testing.T) {
	c := fakeCapability{name: "future-capability", operator: "future", enabledFor: "enabled", shared: []client.Object{configMap("first"), configMap("second")}}
	instances := []obsv1alpha1.ObservabilityInstaller{
		{ObjectMeta: metav1.ObjectMeta{Name: "enabled"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "disabled"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "enabled"}},
	}
	plan, err := aggregateShared([]capability.Capability{c}, instances)
	require.NoError(t, err)
	require.Len(t, plan.operators, 1)
	require.True(t, plan.operators[0].desired)
	require.Len(t, plan.owned, 2)
	require.Len(t, plan.desired, 2, "identity includes namespace and duplicate requests are merged")

	plan, err = aggregateShared([]capability.Capability{c}, nil)
	require.NoError(t, err)
	require.Len(t, plan.owned, 2)
	require.Empty(t, plan.desired)
	require.False(t, plan.operators[0].desired)

	now := metav1.Now()
	deleting := []obsv1alpha1.ObservabilityInstaller{{ObjectMeta: metav1.ObjectMeta{Name: "enabled", DeletionTimestamp: &now}}}
	plan, err = aggregateShared([]capability.Capability{c}, deleting)
	require.NoError(t, err)
	require.Empty(t, plan.desired, "deleting installers do not keep shared resources")
	require.False(t, plan.operators[0].desired)
}

func TestAggregateSharedMergesOperatorsAcrossCapabilities(t *testing.T) {
	// Only the second capability is enabled; the shared operator must stay desired.
	disabled := fakeCapability{name: "disabled", operator: "shared-operator"}
	enabled := fakeCapability{name: "enabled", operator: "shared-operator", enabledFor: "enabled"}
	instances := []obsv1alpha1.ObservabilityInstaller{{ObjectMeta: metav1.ObjectMeta{Name: "enabled"}}}
	plan, err := aggregateShared([]capability.Capability{disabled, enabled}, instances)
	require.NoError(t, err)
	require.Len(t, plan.operators, 1)
	require.True(t, plan.operators[0].desired)
}

func TestAggregateSharedRejectsUninventoriedResources(t *testing.T) {
	c := uninventoried{fakeCapability{name: "broken", operator: "broken", enabledFor: "enabled"}}
	_, err := aggregateShared([]capability.Capability{c}, []obsv1alpha1.ObservabilityInstaller{{ObjectMeta: metav1.ObjectMeta{Name: "enabled"}}})
	require.ErrorContains(t, err, "not in its inventory")
}

// uninventoried requests a shared resource it does not inventory.
type uninventoried struct{ fakeCapability }

func (uninventoried) SharedDesired(*obsv1alpha1.ObservabilityInstaller) []client.Object {
	return []client.Object{configMap("x")}
}
