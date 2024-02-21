package framework

import (
	"context"
	"testing"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	"gotest.tools/v3/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

type StackFn func(monitoringStack *stack.MonitoringStack)

func SetPrometheusReplicas(replicas *int32) StackFn {
	return func(ms *stack.MonitoringStack) {
		ms.Spec.PrometheusConfig.Replicas = replicas
	}
}
func SetResourceSelector(resourceSelector *v1.LabelSelector) StackFn {
	return func(ms *stack.MonitoringStack) {
		ms.Spec.ResourceSelector = resourceSelector
	}
}
func SetAlertmanagerDisabled(disabled bool) StackFn {
	return func(ms *stack.MonitoringStack) {
		ms.Spec.AlertmanagerConfig.Disabled = disabled
	}
}

// Update monitoringstack with retry
func (f *Framework) UpdateWithRetry(t *testing.T, ms *stack.MonitoringStack, fns ...StackFn) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		key := types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}
		err := f.K8sClient.Get(context.Background(), key, ms)
		assert.NilError(t, err, "failed to get a monitoring stack")
		for _, fn := range fns {
			fn(ms)
		}
		err = f.K8sClient.Update(context.Background(), ms)
		return err
	})
	return err
}
