package e2e

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	msov1 "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

func TestThanosQuerierController(t *testing.T) {
	assertCRDExists(t, "thanosqueriers.monitoring.rhobs")

	ts := []testCase{
		{
			name:     "Create resources for single monitoring stack",
			scenario: singleStackWithSidecar,
		},
		{
			name:     "Delete resources if matched monitoring stack is deleted",
			scenario: stackWithSidecarGetsDeleted,
		},
		{
			name:     "Create resources for single monitoring stack with web endpoint TLS",
			scenario: singleStackWithSidecarTLS,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func stackWithSidecarGetsDeleted(t *testing.T) {
	tq, ms := newThanosStackCombo(t, "tq-ms-combo")
	err := f.K8sClient.Create(context.Background(), tq)
	assert.NilError(t, err, "failed to create a thanos querier")
	err = f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	// delete MonitoringStack
	f.K8sClient.Delete(context.Background(), ms)
	waitForStackDeletion(ms.Name)
	// thanos-queroer deployment and service should get deleted as well
	name := "thanos-querier-" + tq.Name
	waitForDeploymentDeletion(name)
	waitForServiceDeletion(name)
}

func singleStackWithSidecar(t *testing.T) {
	tq, ms := newThanosStackCombo(t, "tq-ms-combo")

	// Create the MonitoringStack first to ensure that the DNS entries will be
	// populated when the Thanos Querier starts.
	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")
	_ = f.GetStackWhenAvailable(t, ms.Name, ms.Namespace)

	err = f.K8sClient.Create(context.Background(), tq)
	assert.NilError(t, err, "failed to create a thanos querier")

	name := "thanos-querier-" + tq.Name
	thanosDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, name, tq.Namespace, &thanosDeployment)

	thanosService := corev1.Service{}
	f.GetResourceWithRetry(t, name, tq.Namespace, &thanosService)

	f.AssertDeploymentReady(name, tq.Namespace, framework.WithTimeout(5*time.Minute))(t)

	var lastErr error
	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		result, err := f.QueryPrometheusService(ctx, framework.NamespacedName{Name: name, Namespace: e2eTestNamespace}, 10902, "prometheus_build_info")
		if err != nil {
			lastErr = err
			return false, nil
		}

		vector, ok := result.(model.Vector)
		if !ok {
			lastErr = fmt.Errorf("unexpected result type %T", result)
			return false, nil
		}

		if len(vector) != 2 {
			lastErr = fmt.Errorf("expected 2 results for prometheus_build_info, got %d", len(vector))
			return false, nil
		}

		return true, nil
	}); wait.Interrupted(err) {
		t.Fatalf("querying thanos did not yield expected results: %s", lastErr)
	}
}

func singleStackWithSidecarTLS(t *testing.T) {
	comboName := "tq-ms-combo-tls"
	querierName := "thanos-querier-" + comboName

	certs, key, err := cert.GenerateSelfSignedCertKey(querierName, []net.IP{}, []string{})
	assert.NilError(t, err)

	thanosKey := string(key)
	thanosCerts := strings.SplitAfter(string(certs), "-----END CERTIFICATE-----")

	tlsSecretName := "thanos-test-tls-secret"

	thanosTLSSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsSecretName,
			Namespace: e2eTestNamespace,
		},
		StringData: map[string]string{
			"tls.key": thanosKey,
			"tls.crt": thanosCerts[0],
			"ca.crt":  thanosCerts[1],
		},
	}
	err = f.K8sClient.Create(context.Background(), &thanosTLSSecret)
	assert.NilError(t, err)

	tq, ms := newThanosStackCombo(t, comboName)

	// Create the MonitoringStack first to ensure that the DNS entries will be
	// populated when the Thanos Querier starts.
	err = f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")
	_ = f.GetStackWhenAvailable(t, ms.Name, ms.Namespace)

	tq.Spec.WebTLSConfig = &msov1.WebTLSConfig{
		PrivateKey: msov1.SecretKeySelector{
			Name: tlsSecretName,
			Key:  "tls.key",
		},
		Certificate: msov1.SecretKeySelector{
			Name: tlsSecretName,
			Key:  "tls.crt",
		},
		CertificateAuthority: msov1.SecretKeySelector{
			Name: tlsSecretName,
			Key:  "ca.crt",
		},
	}
	err = f.K8sClient.Create(context.Background(), tq)
	assert.NilError(t, err, "failed to create a thanos querier")

	thanosDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, querierName, tq.Namespace, &thanosDeployment)

	thanosService := corev1.Service{}
	f.GetResourceWithRetry(t, querierName, tq.Namespace, &thanosService)

	f.AssertDeploymentReadyAndStable(querierName, tq.Namespace, framework.WithTimeout(5*time.Minute))(t)

	// Assert prometheus instance can be queried
	stopChan := make(chan struct{})
	defer close(stopChan)
	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		err = f.StartServicePortForward(querierName, e2eTestNamespace, "10902", stopChan)
		return err == nil, nil
	}); wait.Interrupted(err) {
		t.Fatal("timeout waiting for port-forward")
	}

	promClient, err := framework.NewTLSPrometheusClient("https://localhost:10902", thanosCerts[1], querierName)
	if err != nil {
		t.Fatal(fmt.Errorf("Failed to create prometheus client: %s", err))
	}
	expectedResults := map[string]int{
		"prometheus_build_info": 2, // must return from both prometheus pods
	}
	var lastErr error
	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		correct := 0
		for query, value := range expectedResults {
			result, err := promClient.Query(query)
			if err != nil {
				return false, nil
			}

			if len(result.Data.Result) == 0 {
				return false, nil
			}

			if len(result.Data.Result) > value {
				lastErr = fmt.Errorf("invalid result for query %s, got %d, want %d", query, len(result.Data.Result), value)
				return true, lastErr
			}

			if len(result.Data.Result) != value {
				return false, nil
			}

			correct++
		}

		return correct == len(expectedResults), nil
	}); wait.Interrupted(err) {
		t.Fatal(fmt.Errorf("querying thanos did not yield expected results: %w", lastErr))
	}
}

func newThanosQuerier(t *testing.T, name string, selector map[string]string) *msov1.ThanosQuerier {
	tq := &msov1.ThanosQuerier{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: msov1.ThanosQuerierSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selector,
			},
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), tq)
		waitForThanosQuerierDeletion(tq)
	})

	return tq
}

func waitForThanosQuerierDeletion(tq *msov1.ThanosQuerier) error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, wait.ForeverTestTimeout, true, func(ctx context.Context) (done bool, err error) {
		err = f.K8sClient.Get(context.Background(),
			types.NamespacedName{Name: tq.Name, Namespace: tq.Namespace},
			tq)
		return errors.IsNotFound(err), nil
	})
}

func waitForDeploymentDeletion(name string) error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, wait.ForeverTestTimeout, true, func(ctx context.Context) (done bool, err error) {
		var dep appsv1.Deployment
		err = f.K8sClient.Get(context.Background(),
			types.NamespacedName{Name: name, Namespace: e2eTestNamespace},
			&dep)
		return errors.IsNotFound(err), nil
	})
}

func waitForServiceDeletion(name string) error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, wait.ForeverTestTimeout, true, func(ctx context.Context) (done bool, err error) {
		var svc corev1.Service
		err = f.K8sClient.Get(context.Background(),
			types.NamespacedName{Name: name, Namespace: e2eTestNamespace},
			&svc)
		return errors.IsNotFound(err), nil
	})
}

func newThanosStackCombo(t *testing.T, name string) (*msov1.ThanosQuerier, *msov1.MonitoringStack) {
	labels := map[string]string{"stack": "mso-e2e"}
	tq := ensureLabels(newThanosQuerier(t, name, labels), labels)
	ms := ensureLabels(newMonitoringStack(t, name), labels)
	return tq.(*msov1.ThanosQuerier), ms.(*msov1.MonitoringStack)
}

func ensureLabels(obj client.Object, wantLabels map[string]string) client.Object {
	labels := obj.GetLabels()
	if labels == nil {
		obj.SetLabels(wantLabels)
		return obj
	}
	for name, val := range wantLabels {
		if _, ok := labels[name]; !ok {
			labels[name] = val
		}
	}
	return obj
}
