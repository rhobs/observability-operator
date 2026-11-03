package reconciler

import (
	"context"
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestUnmanagedCreateUpdateReconcilerUpdatesSubscriptionSpec(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, olmv1alpha1.AddToScheme(scheme))
	ctx := context.Background()
	key := client.ObjectKey{Name: "tempo-product", Namespace: "operators"}

	live := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.Name, Namespace: key.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: obsv1alpha1.GroupVersion.String(), Kind: "ObservabilityInstaller", Name: "old-installer", UID: types.UID("old-uid")},
				{APIVersion: "other.example/v1", Kind: "Other", Name: "other", UID: types.UID("other-uid")},
			},
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Package: "tempo-product", Channel: "old", StartingCSV: "tempo.v1",
			CatalogSource: "mirror", CatalogSourceNamespace: "mirror-ns", InstallPlanApproval: olmv1alpha1.ApprovalManual,
			Config: &olmv1alpha1.SubscriptionConfig{NodeSelector: map[string]string{"admin": "set"}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(live).Build()

	desired := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Package: "tempo-product", Channel: "new", StartingCSV: "tempo.v2",
			CatalogSource: "redhat-operators", CatalogSourceNamespace: "openshift-marketplace", InstallPlanApproval: olmv1alpha1.ApprovalAutomatic,
		},
	}
	r := NewUnmanagedCreateUpdateReconciler(desired)

	// Reconcile twice: the reconciler must keep applying the desired spec even
	// after CreateOrUpdate has overwritten its resource with the live object.
	for range 2 {
		require.NoError(t, r.Reconcile(ctx, c, scheme))

		got := &olmv1alpha1.Subscription{}
		require.NoError(t, c.Get(ctx, key, got))
		require.NotNil(t, got.Spec)
		require.Equal(t, "new", got.Spec.Channel)
		require.Equal(t, "tempo.v2", got.Spec.StartingCSV)
		require.Equal(t, "mirror", got.Spec.CatalogSource, "admin-set catalog must be preserved")
		require.Equal(t, "mirror-ns", got.Spec.CatalogSourceNamespace)
		require.Equal(t, olmv1alpha1.ApprovalManual, got.Spec.InstallPlanApproval, "admin-set approval must be preserved")
		require.NotNil(t, got.Spec.Config, "admin-set fields must be preserved")
		require.Equal(t, map[string]string{"admin": "set"}, got.Spec.Config.NodeSelector)
		require.Equal(t, []metav1.OwnerReference{{APIVersion: "other.example/v1", Kind: "Other", Name: "other", UID: types.UID("other-uid")}}, got.OwnerReferences)
	}
}

func TestCreateUpdateReconcilerCreatesMissingResource(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, olmv1alpha1.AddToScheme(scheme))
	ctx := context.Background()
	key := client.ObjectKey{Name: "otel-product", Namespace: "operators"}
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	desired := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "otel-product", Channel: "stable"},
	}
	require.NoError(t, NewUnmanagedCreateUpdateReconciler(desired).Reconcile(ctx, c, scheme))

	got := &olmv1alpha1.Subscription{}
	require.NoError(t, c.Get(ctx, key, got))
	require.Equal(t, "stable", got.Spec.Channel)
}
