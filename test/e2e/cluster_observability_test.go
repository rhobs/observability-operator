package e2e

import (
	"context"
	_ "embed"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

var (
	//go:embed minio.yaml
	minioManifests string
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

	for _, doc := range strings.Split(minioManifests, "---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		// Create an unstructured object to decode the YAML into
		obj := &unstructured.Unstructured{}

		// Use the YAML decoder to convert the YAML string into the unstructured object
		decoder := yamlutil.NewYAMLOrJSONDecoder(strings.NewReader(doc), 100)
		err := decoder.Decode(&obj)
		require.NoError(t, err)

		// Use the client to create the object in the Kubernetes cluster
		err = f.K8sClient.Create(context.Background(), obj)
		require.NoError(t, err)
		f.CleanUp(t, func() { f.K8sClient.Delete(ctx, obj) })
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: f.OperatorNamespace,
		},
		Data: map[string][]byte{
			"endpoint":          []byte("http://minio.minio.svc:9000"),
			"bucket":            []byte("tempo"),
			"access_key_id":     []byte("tempo"),
			"access_key_secret": []byte("supersecret"),
		},
	}
	err := f.K8sClient.Create(ctx, secret)
	require.NoError(t, err)
	f.CleanUp(t, func() { f.K8sClient.Delete(ctx, secret) })

	// Create ClusterObservability resource
	clusterObs := &obsv1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: obsv1alpha1.ClusterObservabilitySpec{
			Storage: obsv1alpha1.StorageSpec{
				Secret: obsv1alpha1.SecretSpec{
					Name: "minio",
					Type: "s3",
				},
			},
			Capabilities: &obsv1alpha1.CapabilitiesSpec{
				Tracing: obsv1alpha1.TracingSpec{
					CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
						Enabled: true,
					},
				},
			},
		},
	}

	f.CleanUp(t, func() {
		f.K8sClient.Delete(ctx, clusterObs)
	})

	// Create the resource
	err = f.K8sClient.Create(ctx, clusterObs)
	assert.NilError(t, err, "Failed to create ClusterObservability resource")

	// Verify resource exists
	var createdClusterObs obsv1alpha1.ClusterObservability
	err = f.K8sClient.Get(ctx, types.NamespacedName{Name: name}, &createdClusterObs)
	assert.NilError(t, err, "Failed to get ClusterObservability resource")

	// TODO implement a data test - report traces and check the collector if export was successful
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 60*time.Second, true, func(ctx context.Context) (bool, error) {
		var instance obsv1alpha1.ClusterObservability
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: name}, &instance)
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		r, _ := regexp.Compile("cluster-observability2/coo ([0-9].*)")
		if r.MatchString(instance.Status.Tempo) {
			return true, nil
		}
		return false, nil
	})

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
