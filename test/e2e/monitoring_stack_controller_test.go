package e2e

import (
	"context"
	"testing"

	stack "rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"gotest.tools/v3/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func assertCRDExists(t *testing.T, crds ...string) {
	for _, crd := range crds {
		f.AssertResourceEventuallyExists(crd, "", &apiextensionsv1.CustomResourceDefinition{})(t)
	}
}

func TestMonitoringStackController(t *testing.T) {
	assertCRDExists(t,
		"prometheuses.monitoring.coreos.com",
		"monitoringstacks.monitoring.rhobs",
	)

	ts := []testCase{
		{
			name:     "Empty stack spec must create a Prometheus",
			scenario: emptyStackCreatesPrometheus,
		}, {
			name:     "stack spec are reflected in Prometheus",
			scenario: reconcileStack,
		}, {
			name:     "invalid loglevels are rejected",
			scenario: validateStackLogLevel,
		}, {
			name:     "invalid retention is rejected",
			scenario: validateStackRetention,
		}, {
			name:     "Controller reverts back changes to Prometheus",
			scenario: reconcileRevertsManualChanges,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func emptyStackCreatesPrometheus(t *testing.T) {
	ms := newMonitoringStack("empty-stack")
	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	// Creating an Empty monitoring stack must create a Prometheus with defaults applied
	prometheus := monv1.Prometheus{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &prometheus)

	expected := monv1.PrometheusSpec{
		Retention: "120h",
		LogLevel:  "info",
	}

	assert.DeepEqual(t, expected, prometheus.Spec)

	// cleanup
	f.K8sClient.Delete(context.Background(), ms)
}

func reconcileStack(t *testing.T) {
	ms := newMonitoringStack("reconcile-test")
	ms.Spec.LogLevel = "debug"
	ms.Spec.Retention = "1h"
	ms.Spec.ResourceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"system":   "foobar",
			"resource": "test",
		},
	}

	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	// Creating an Empty monitoring stack must create a Prometheus with defaults applied
	generated := monv1.Prometheus{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &generated)

	expected := monv1.PrometheusSpec{
		LogLevel:               string(ms.Spec.LogLevel),
		Retention:              ms.Spec.Retention,
		ServiceMonitorSelector: ms.Spec.ResourceSelector.DeepCopy(),
	}

	assert.DeepEqual(t, expected.ServiceMonitorSelector, generated.Spec.ServiceMonitorSelector)
	assert.Equal(t, expected.LogLevel, generated.Spec.LogLevel)
	assert.Equal(t, expected.Retention, generated.Spec.Retention)

}

func reconcileRevertsManualChanges(t *testing.T) {
	ms := newMonitoringStack("revert-test")
	ms.Spec.LogLevel = "debug"
	ms.Spec.Retention = "1h"
	ms.Spec.ResourceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"system":   "foobar",
			"resource": "test",
		},
	}

	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	// Creating an Empty monitoring stack must create a Prometheus with defaults applied
	generated := monv1.Prometheus{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &generated)

	// update the prometheus created by monitoring-stack controller

	modified := generated.DeepCopy()
	modified.Spec.ServiceMonitorSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"system": "modified",
			"add":    "another",
			// resource label is deleted but should get reverted
		},
	}

	err = f.K8sClient.Update(context.Background(), modified)
	assert.NilError(t, err, "failed to update a prometheus")

	reconciled := monv1.Prometheus{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &reconciled)

	assert.DeepEqual(t, generated.Spec, reconciled.Spec)

	// cleanup
	f.K8sClient.Delete(context.Background(), ms)
}

func validateStackLogLevel(t *testing.T) {
	invalidLogLevels := []string{
		"foobar",
		"xyz",
		"Info",
		"Debug",
	}
	ms := newMonitoringStack("invalid-loglevel-stack")
	for _, v := range invalidLogLevels {

		ms.Spec.LogLevel = stack.LogLevel(v)
		err := f.K8sClient.Create(context.Background(), ms)
		assert.ErrorContains(t, err, `spec.logLevel: Unsupported value`)
	}

	validMS := newMonitoringStack("valid-loglevel")
	validMS.Spec.LogLevel = "debug"
	err := f.K8sClient.Create(context.Background(), validMS)
	assert.NilError(t, err, `debug is a valid loglevel`)

	// cleanup
	err = f.K8sClient.Delete(context.Background(), validMS)
	assert.NilError(t, err, `deletion error`)
}

func validateStackRetention(t *testing.T) {
	invalidRetention := []string{
		"100days",
		"100ducks",
		"100 days",
		"100 hours",
		"100 h",
		"100 s",
		"100d   ",
	}

	ms := newMonitoringStack("invalid-retention")
	for _, v := range invalidRetention {
		ms.Spec.Retention = v
		err := f.K8sClient.Create(context.Background(), ms)
		assert.ErrorContains(t, err, `spec.retention: Invalid value`)
	}

	validMS := newMonitoringStack("valid-retention")
	validMS.Spec.Retention = "100h"

	err := f.K8sClient.Create(context.Background(), validMS)
	assert.NilError(t, err, `100h is a valid retention period`)

	// cleanup
	err = f.K8sClient.Delete(context.Background(), validMS)
	assert.NilError(t, err, `deletion error`)
}

func newMonitoringStack(name string) *stack.MonitoringStack {
	return &stack.MonitoringStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
	}
}
