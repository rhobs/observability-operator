package e2e

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestClusterObservabilityController(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("The tests are skipped on non-ocp cluster")
	}

	err := obsv1alpha1.AddToScheme(scheme.Scheme)
	assert.NilError(t, err)

	assertCRDExists(t, "clusterobservabilities.observability.openshift.io")

	t.Run("ClusterObservabilityLifecycle", testClusterObservabilityInstallOperators)
}

func testClusterObservabilityInstallOperators(t *testing.T) {
	ctx := context.Background()
	name := "test-cluster-observability-lifecycle"

	// Create ClusterObservability resource
	clusterObs := &obsv1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: obsv1alpha1.ClusterObservabilitySpec{
			Capabilities: &obsv1alpha1.CapabilitiesSpec{
				Tracing: obsv1alpha1.TracingSpec{
					CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
						Enabled: false,
						Operators: obsv1alpha1.OperatorsSpec{
							Install: ptr.To(true),
						},
					},
				},
			},
		},
	}

	f.CleanUp(t, func() {
		f.K8sClient.Delete(ctx, clusterObs)
	})

	// Create the resource
	err := f.K8sClient.Create(ctx, clusterObs)
	assert.NilError(t, err, "Failed to create ClusterObservability resource")

	// Verify resource exists
	var createdClusterObs obsv1alpha1.ClusterObservability
	err = f.K8sClient.Get(ctx, types.NamespacedName{Name: name}, &createdClusterObs)
	assert.NilError(t, err, "Failed to get ClusterObservability resource")

	// Verify spec is correct
	assert.Equal(t, createdClusterObs.Spec.Capabilities.Tracing.Enabled, false)

	// Delete the resource
	err = f.K8sClient.Delete(ctx, &createdClusterObs)
	assert.NilError(t, err, "Failed to delete ClusterObservability resource")

	// Verify resource is deleted
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		var deletedClusterObs obsv1alpha1.ClusterObservability
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: name}, &deletedClusterObs)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	assert.NilError(t, err, "ClusterObservability resource should be deleted")
}
