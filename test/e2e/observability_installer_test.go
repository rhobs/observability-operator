package e2e

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net/http"
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
	//go:embed traces_telemetrygen.yaml
	telemetrygenManifest string
	//go:embed traces_verify.yaml
	verifyTracesManifests string
)

func TestObservabilityInstallerController(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("The tests are skipped on non-ocp cluster")
	}

	assertCRDExists(t, "observabilityinstallers.observability.openshift.io")

	t.Run("ObservabilityInstallerTracing", testObservabilityInstallerTracing)
}

func testObservabilityInstallerTracing(t *testing.T) {
	ctx := context.Background()

	// The ObservabilityInstaller installs operators via subscriptions,
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
		obj := deployManifest(t, doc)
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

	// Create ObservabilityInstaller resource
	obsInstaller := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coo",
			Namespace: operandNamespace.Name,
		},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{
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
		f.K8sClient.Delete(ctx, obsInstaller)
	})

	// Create the resource
	err = f.K8sClient.Create(ctx, obsInstaller)
	assert.NilError(t, err, "Failed to create ObservabilityInstaller resource")

	// Verify resource exists
	var createdClusterObs obsv1alpha1.ObservabilityInstaller
	err = f.K8sClient.Get(ctx, types.NamespacedName{Name: "coo", Namespace: operandNamespace.Name}, &createdClusterObs)
	assert.NilError(t, err, "Failed to get ObservabilityInstaller resource")

	// Wait for Tempo to be ready
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		var instance obsv1alpha1.ObservabilityInstaller
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
		var instance obsv1alpha1.ObservabilityInstaller
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

	fmt.Println("---> Tempo and OpenTelemetry Collector are ready")

	stopChan := make(chan struct{})
	defer close(stopChan)
	if err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		err = f.StartServicePortForward("tempo-coo-ingester", operandNamespace.Name, "3200", stopChan)
		return err == nil, nil
	}); err != nil {
		require.NoError(t, err)
	}

	// Check readiness endpoint
	ctx, cancel := context.WithTimeout(ctx, time.Minute*1)
	defer cancel()

	// Load and configure mTLS certificates like the original job
	tr := http.DefaultTransport.(*http.Transport).Clone()
	var mtlsSecret corev1.Secret
	if err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "tempo-coo-gateway-mtls", Namespace: operandNamespace.Name}, &mtlsSecret); err == nil {
		clientCert, clientKey := mtlsSecret.Data["tls.crt"], mtlsSecret.Data["tls.key"]
		if cert, err := tls.X509KeyPair(clientCert, clientKey); err == nil {
			tr.TLSClientConfig = &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true, // equivalent to curl -k
			}
		} else {
			tr.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true, // fallback if cert parsing fails
			}
		}
	} else {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // fallback if secret not found
		}
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, time.Minute*1, true, func(ctx context.Context) (bool, error) {
		t.Log("getting tempo readiness endpoint")
		req, err := http.NewRequestWithContext(ctx, "GET", "https://localhost:3200/ready", nil)
		if err != nil {
			return false, nil
		}

		httpClient := &http.Client{Transport: tr, Timeout: 30 * time.Second}
		resp, err := httpClient.Do(req)
		if err != nil {
			t.Logf("tempo readiness check failed: %v", err)
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			t.Log("SUCCESS: Tempo service is ready!")
			return true, nil
		}

		t.Logf("tempo readiness check returned status: %d", resp.StatusCode)
		return false, nil
	})
	require.NoError(t, err)

	telemetrygenObj := deployManifest(t, telemetrygenManifest)
	f.CleanUp(t, func() { f.K8sClient.Delete(ctx, telemetrygenObj) })
	f.CleanUp(t, func() { f.K8sClient.Delete(ctx, telemetrygenObj) })
	err = jobHasCompleted(t, telemetrygenObj.GetName(), operandNamespace.Name, time.Minute)
	require.NoError(t, err)

	for _, doc := range strings.Split(verifyTracesManifests, "---") {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		obj := deployManifest(t, doc)
		f.CleanUp(t, func() { f.K8sClient.Delete(ctx, obj) })
	}
	err = jobHasCompleted(t, "verify-traces-traceql-grpc", operandNamespace.Name, time.Minute)
	require.NoError(t, err)

	// Delete the resource
	err = f.K8sClient.Delete(ctx, &createdClusterObs)
	assert.NilError(t, err, "Failed to delete ObservabilityInstaller resource")

	// Verify resource is deleted
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		var deletedClusterObs obsv1alpha1.ObservabilityInstaller
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: "coo", Namespace: operandNamespace.Name}, &deletedClusterObs)
		if apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, nil
	})
	require.NoError(t, err)
}

func deployManifest(t *testing.T, manifest string) client.Object {
	// Create an unstructured object to decode the YAML into
	obj := &unstructured.Unstructured{}
	// Use the YAML decoder to convert the YAML string into the unstructured object
	decoder := yamlutil.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 100)
	err := decoder.Decode(&obj)
	require.NoError(t, err)

	// Use the client to create the object in the Kubernetes cluster
	err = f.K8sClient.Create(context.Background(), obj)
	if !apierrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}
	return obj
}

func jobHasCompleted(t *testing.T, name, ns string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(t.Context(), 1*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		var job batchv1.Job
		err := f.K8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, &job)
		if apierrors.IsNotFound(err) {
			t.Logf("job %s not found", name)
			return false, nil
		}

		if job.Status.Succeeded > 0 {
			return true, nil
		}

		t.Logf("job %s didn't succeed yet", name)
		return false, nil
	})
}
