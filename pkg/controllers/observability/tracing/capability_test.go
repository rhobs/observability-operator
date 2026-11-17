package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func newInstance(tracing *obsv1alpha1.TracingSpec) *obsv1alpha1.ObservabilityInstaller {
	return &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "observability"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
			Tracing: tracing,
		}},
	}
}

func keys(objects []client.Object) map[string]bool {
	result := map[string]bool{}
	for _, object := range objects {
		result[object.GetObjectKind().GroupVersionKind().Kind+"/"+object.GetNamespace()+"/"+object.GetName()] = true
	}
	return result
}

// requireSubset checks that every desired object can be cleaned up.
func requireSubset(t *testing.T, desired, inventory []client.Object) {
	t.Helper()
	inventoryKeys := keys(inventory)
	for key := range keys(desired) {
		require.True(t, inventoryKeys[key], "%s is not in the inventory", key)
	}
}

func TestTracingCapability(t *testing.T) {
	trueValue := true
	tests := []struct {
		name          string
		tracing       *obsv1alpha1.TracingSpec
		wantOperators bool
		wantOperands  bool
		deleting      bool
	}{
		{name: "capability absent"},
		{name: "capability disabled", tracing: &obsv1alpha1.TracingSpec{}},
		{
			name:     "deleting",
			deleting: true,
			tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
				Enabled: true,
			}},
		},
		{
			name: "operators only",
			tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
				Operators: &obsv1alpha1.OperatorsSpec{Install: &trueValue},
			}},
			wantOperators: true,
		},
		{
			name: "capability enabled",
			tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
				Enabled: true,
			}},
			wantOperators: true,
			wantOperands:  true,
		},
	}

	c := New(Options{})
	require.Equal(t, "tracing", c.Name())
	require.Len(t, c.Operators(), 2)
	require.NotEmpty(t, c.SharedInventory(), "shared inventory is required for cleanup")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := newInstance(tt.tracing)
			if tt.deleting {
				now := metav1.Now()
				instance.DeletionTimestamp = &now
			}
			desired, err := c.Desired(context.Background(), instance, fake.NewClientBuilder().Build())
			require.NoError(t, err)
			inventory := c.Inventory(instance)
			require.NotEmpty(t, inventory, "inventory is required for cleanup while disabled")
			requireSubset(t, desired, inventory)
			sharedDesired := c.SharedDesired(instance)
			requireSubset(t, sharedDesired, c.SharedInventory())
			if !tt.deleting {
				require.Equal(t, tt.wantOperators, c.OperatorsDesired(instance))
			}
			if tt.wantOperands {
				require.NotEmpty(t, desired)
				require.NotEmpty(t, sharedDesired)
			} else {
				require.Empty(t, desired)
				require.Empty(t, sharedDesired)
			}
		})
	}
}

func TestTracingInventoryCoversOptionalStorage(t *testing.T) {
	s3 := &obsv1alpha1.S3Spec{
		Bucket:          "tempo",
		Endpoint:        "https://s3.example.com",
		AccessKeyID:     "id",
		AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "missing", Key: "key"},
	}
	instance := newInstance(&obsv1alpha1.TracingSpec{
		CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
		Storage: &obsv1alpha1.TracingStorageSpec{
			ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{S3: s3},
		},
	})
	c := New(Options{})

	// A missing credential fails Desired but not the cleanup inventory.
	_, err := c.Desired(context.Background(), instance, fake.NewClientBuilder().Build())
	require.Error(t, err)
	inventory := keys(c.Inventory(instance))

	// Optional TLS resources are inventoried even when the spec does not use
	// them, so they are deleted when TLS is removed.
	require.True(t, inventory["Secret/observability/"+tempoStorageSecretName("test")])
	require.True(t, inventory["ConfigMap/observability/"+tempoStorageCAConfigMapName("test")])
}
