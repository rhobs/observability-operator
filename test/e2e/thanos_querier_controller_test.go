package e2e

import (
	"context"
	"testing"
	"time"

	msov1 "github.com/rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"
	"github.com/rhobs/monitoring-stack-operator/test/e2e/framework"

	"gotest.tools/v3/assert"

	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestThanosQuerierController(t *testing.T) {
	assertCRDExists(t,
		"thanosquerier.monitoring.rhobs",
	)

	ts := []testCase{
		{
			name:     "Create resources for single monitoring stack",
			scenario: singleStackWithSidecar,
		},
		{
			name:     "Don't create any resources if selector matches nothing",
			scenario: noStack,
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

func noStack(t *testing.T) {
	tq := newThanosQuerier(t, "no-stack", map[string]string{"doesnt": "exist"})
	err := f.K8sClient.Create(context.Background(), tq)
	assert.NilError(t, err, "failed to create a thanos querier")

	name := "thanos-querier-" + tq.Name
	thanosDeployment := appsv1.Deployment{}
	f.AssertResourceNeverExists(name, tq.Namespace, &thanosDeployment, framework.WithTimeout(15*time.Second))(t)
	thanosService := corev1.Service{}
	f.AssertResourceNeverExists(name, tq.Namespace, &thanosService, framework.WithTimeout(15*time.Second))(t)
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

	// Creating a basic combo must create a thanos deplouyment and a service
	name := "thanos-querier-" + tq.Name
	thanosDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, name, tq.Namespace, &thanosDeployment)
	thanosService := corev1.Service{}
	f.GetResourceWithRetry(t, name, tq.Namespace, &thanosService)
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
	return wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (bool, error) {
		err := f.K8sClient.Get(context.Background(),
			types.NamespacedName{Name: tq.Name, Namespace: tq.Namespace},
			tq)
		return errors.IsNotFound(err), nil
	})
}

func waitForDeploymentDeletion(name string) error {
	return wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (bool, error) {
		var dep appsv1.Deployment
		err := f.K8sClient.Get(context.Background(),
			types.NamespacedName{Name: name, Namespace: e2eTestNamespace},
			&dep)
		return errors.IsNotFound(err), nil
	})
}

func waitForServiceDeletion(name string) error {
	return wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (bool, error) {
		var svc corev1.Service
		err := f.K8sClient.Get(context.Background(),
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
