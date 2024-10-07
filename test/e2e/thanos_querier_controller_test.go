package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	err := f.K8sClient.Create(context.Background(), tq)
	assert.NilError(t, err, "failed to create a thanos querier")
	err = f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	// Creating a basic combo must create a thanos deployment and a service
	name := "thanos-querier-" + tq.Name
	thanosDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, name, tq.Namespace, &thanosDeployment)

	thanosService := corev1.Service{}
	f.GetResourceWithRetry(t, name, tq.Namespace, &thanosService)

	f.AssertDeploymentReady(name, tq.Namespace, framework.WithTimeout(5*time.Minute))(t)
	// Assert prometheus instance can be queried
	stopChan := make(chan struct{})
	defer close(stopChan)
	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		err = f.StartServicePortForward(name, e2eTestNamespace, "10902", stopChan)
		return err == nil, nil
	}); wait.Interrupted(err) {
		t.Fatal("timeout waiting for port-forward")
	}

	promClient := framework.NewPrometheusClient("http://localhost:10902")
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
