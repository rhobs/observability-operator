package framework

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// AssertResourceNeverExists asserts that a statefulset is never created for the duration of wait.ForeverTestTimeout
func (f *Framework) AssertResourceNeverExists(name string, namespace string, resource client.Object) func(t *testing.T) {
	return func(t *testing.T) {
		if err := wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (done bool, err error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, resource); errors.IsNotFound(err) {
				return false, nil
			}

			return true, fmt.Errorf("statefulset %s/%s should not have been created", namespace, name)
		}); err != wait.ErrWaitTimeout {
			t.Fatal(err)
		}
	}
}

// AssertResourceEventuallyExists asserts that a statefulset is created duration a time period of wait.ForeverTestTimeout
func (f *Framework) AssertResourceEventuallyExists(name string, namespace string, resource client.Object) func(t *testing.T) {
	return func(t *testing.T) {
		if err := wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (done bool, err error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, resource); err == nil {
				return true, nil
			}
			return false, nil
		}); err == wait.ErrWaitTimeout {
			t.Fatal(fmt.Errorf("statefulset %s/%s was never created", namespace, name))
		}
	}
}

// AssertStatefulsetReady asserts that a statefulset has the desired number of pods running
func (f *Framework) AssertStatefulsetReady(name string, namespace string) func(t *testing.T) {
	return func(t *testing.T) {
		key := types.NamespacedName{Name: name, Namespace: namespace}
		if err := wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (bool, error) {
			pod := &appsv1.StatefulSet{}
			err := f.K8sClient.Get(context.Background(), key, pod)
			return err == nil && pod.Status.ReadyReplicas == *pod.Spec.Replicas, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *Framework) GetResourceWithRetry(t *testing.T, name, namespace string, obj client.Object) {
	err := wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (bool, error) {
		key := types.NamespacedName{Name: name, Namespace: namespace}

		if err := f.K8sClient.Get(context.Background(), key, obj); errors.IsNotFound(err) {
			// retry
			return false, nil
		}

		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		t.Fatal(fmt.Errorf("resource %s/%s was never created", namespace, name))
	}
}
