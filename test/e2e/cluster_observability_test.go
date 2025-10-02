package e2e

import (
	"context"
	_ "embed"
	"regexp"
	"strings"
	"testing"
	"time"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

var (
	//go:embed traces_minio.yaml
	minioManifests string
	//go:embed traces_verify.yaml
	verifyTracesManifests string
)

func TestClusterObservabilityController(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("The tests are skipped on non-ocp cluster")
	}

	//assert.NilError(t, obsv1alpha1.AddToScheme(scheme.Scheme))
	//assert.NilError(t, olmv1alpha1.AddToScheme(scheme.Scheme))

	assertCRDExists(t, "clusterobservabilities.observability.openshift.io")

	t.Run("ClusterObservabilityTracing", testClusterObservabilityTracing)
}

func testClusterObservabilityTracing(t *testing.T) {
	ctx := context.Background()

	// The ClusterObservability installs operators via subscriptions,
	// therefore it is necessary to change the COO subscription to Automatic approval
	subs := &olmv1alpha1.SubscriptionList{}
	errSubs := f.K8sClient.List(ctx, subs, &client.ListOptions{
		Namespace: f.OperatorNamespace,
	})
	require.NoError(t, errSubs)
	for _, sub := range subs.Items {
		sub.Spec.InstallPlanApproval = olmv1alpha1.ApprovalAutomatic
		err := f.K8sClient.Update(ctx, &sub)
		require.NoError(t, err)
	}

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
		if apierrors.IsAlreadyExists(err) {
			continue
		}
		require.NoError(t, err)
		f.CleanUp(t, func() { f.K8sClient.Delete(ctx, obj) })
	}

	operandNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tracing-observability",
		},
	}
	err := f.K8sClient.Create(ctx, operandNamespace)
	require.NoError(t, err)
	f.CleanUp(t, func() { f.K8sClient.Delete(ctx, operandNamespace) })

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: operandNamespace.Name,
		},
		Data: map[string][]byte{
			"access_key_secret": []byte("supersecret"),
		},
	}
	err = f.K8sClient.Create(ctx, secret)
	require.NoError(t, err)
	f.CleanUp(t, func() { f.K8sClient.Delete(ctx, secret) })

	// Create ClusterObservability resource
	clusterObs := &obsv1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coo",
			Namespace: operandNamespace.Name,
		},
		Spec: obsv1alpha1.ClusterObservabilitySpec{
			Capabilities: &obsv1alpha1.CapabilitiesSpec{
				Tracing: obsv1alpha1.TracingSpec{
					CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
						Enabled: true,
					},
					Storage: obsv1alpha1.TracingStorageSpec{
						ObjectStorageSpec: obsv1alpha1.TracingObjectStorageSpec{
							S3: &obsv1alpha1.S3Spec{
								Bucket:      "tempo",
								Endpoint:    "http://minio.minio.svc:9000",
								AccessKeyID: "tempo",
								AccessKeySecret: obsv1alpha1.SecretKeySelector{
									Key:  "access_key_secret",
									Name: "minio",
								},
							},
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
	err = f.K8sClient.Create(ctx, clusterObs)
	assert.NilError(t, err, "Failed to create ClusterObservability resource")

	// Verify resource exists
	var createdClusterObs obsv1alpha1.ClusterObservability
	err = f.K8sClient.Get(ctx, types.NamespacedName{Name: "coo", Namespace: operandNamespace.Name}, &createdClusterObs)
	assert.NilError(t, err, "Failed to get ClusterObservability resource")

	// Wait for Tempo to be ready
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		var instance obsv1alpha1.ClusterObservability
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "coo", Namespace: operandNamespace.Name}, &instance)
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		t.Logf("Tempo status: %s\n", instance.Status.Tempo)
		r, _ := regexp.Compile(`tracing-observability/coo \([0-9]+.*\)`)
		if r.MatchString(instance.Status.Tempo) {
			return true, nil
		}

		return false, nil
	})
	require.NoError(t, err)

	// Wait for collector to be ready
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		var instance obsv1alpha1.ClusterObservability
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "coo", Namespace: operandNamespace.Name}, &instance)
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		t.Logf("OTEL status: %s\n", instance.Status.OpenTelemetry)
		r, _ := regexp.Compile(`tracing-observability/coo \([0-9]+.*\)`)
		if r.MatchString(instance.Status.OpenTelemetry) {
			return true, nil
		}

		return false, nil
	})
	require.NoError(t, err)

	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "generate-traces-grpc",
			Namespace: operandNamespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "telemetrygen",
							Image: "ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.129.0",
							Args: []string{
								"traces",
								"--otlp-endpoint=coo-collector.tracing-observability.svc.cluster.local:4317",
								"--service=grpc",
								"--otlp-insecure",
								"--traces=10",
							},
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}
	err = f.K8sClient.Create(ctx, &job)
	require.NoError(t, err)

	// Check if the job succeeded
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		var job batchv1.Job
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "generate-traces-grpc", Namespace: operandNamespace.Name}, &job)
		if apierrors.IsNotFound(err) {
			t.Logf("trace generation job not found")
			return false, nil
		}

		if job.Status.Succeeded > 0 {
			return true, nil
		}

		t.Logf("trace generation job didn't succeed yet")
		return false, nil
	})
	require.NoError(t, err)

	for _, doc := range strings.Split(verifyTracesManifests, "---") {
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

	// Check if the verify traces job succeeded
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		var job batchv1.Job
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "verify-traces-traceql-grpc", Namespace: operandNamespace.Name}, &job)
		if apierrors.IsNotFound(err) {
			t.Logf("trace verification job not found")
			return false, nil
		}

		if job.Status.Succeeded > 0 {
			return true, nil
		}

		t.Logf("trace verification job didn't succeed yet")
		return false, nil
	})
	require.NoError(t, err)

	// Delete the resource
	err = f.K8sClient.Delete(ctx, &createdClusterObs)
	assert.NilError(t, err, "Failed to delete ClusterObservability resource")

	// Verify resource is deleted
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		var deletedClusterObs obsv1alpha1.ClusterObservability
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "coo", Namespace: operandNamespace.Name}, &deletedClusterObs)
		if apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, nil
	})
	require.NoError(t, err)
}
