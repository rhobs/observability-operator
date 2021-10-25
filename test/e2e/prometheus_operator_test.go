package e2e

import (
	"context"
	"testing"

	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	prometheusStsName   = "prometheus-prometheus"
	alertmanagerStsName = "alertmanager-alertmanager"
	thanosRulerStsName  = "thanos-ruler-thanosruler"
	ownedResourceLabels = map[string]string{
		"app.kubernetes.io/managed-by": "monitoring-stack-operator",
	}
)

type testCase struct {
	name     string
	scenario func(t *testing.T)
}

func TestPrometheusOperatorForNonOwnedResources(t *testing.T) {
	resources := []client.Object{
		newPrometheus(nil),
		newAlertmanager(nil),
		newThanosRuler(nil),
	}
	defer deleteResources(resources...)

	ts := []testCase{
		{
			name: "Operator should create Prometheus Operator CRDs",
			scenario: func(t *testing.T) {
				f.AssertResourceEventuallyExists("prometheuses.monitoring.coreos.com", "", &apiextensionsv1.CustomResourceDefinition{})(t)
				f.AssertResourceEventuallyExists("alertmanagers.monitoring.coreos.com", "", &apiextensionsv1.CustomResourceDefinition{})(t)
				f.AssertResourceEventuallyExists("thanosrulers.monitoring.coreos.com", "", &apiextensionsv1.CustomResourceDefinition{})(t)
			},
		},
		{
			name:     "Create prometheus operator resources",
			scenario: createResources(resources...),
		},
		{
			name: "Operator should not reconcile resources which it does not own",
			scenario: func(t *testing.T) {
				t.Run("Prometheus never exists", func(t *testing.T) {
					t.Parallel()
					f.AssertResourceNeverExists(prometheusStsName, e2eTestNamespace, &appsv1.StatefulSet{})(t)
				})
				t.Run("Alertmanager never exists", func(t *testing.T) {
					t.Parallel()
					f.AssertResourceNeverExists(alertmanagerStsName, e2eTestNamespace, &appsv1.StatefulSet{})(t)
				})
				t.Run("Thanos Ruler never exists", func(t *testing.T) {
					t.Parallel()
					f.AssertResourceNeverExists(thanosRulerStsName, e2eTestNamespace, &appsv1.StatefulSet{})(t)
				})
			},
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func TestPrometheusOperatorForOwnedResources(t *testing.T) {
	resources := []client.Object{
		newPrometheus(ownedResourceLabels),
		newAlertmanager(ownedResourceLabels),
		newThanosRuler(ownedResourceLabels),
	}
	defer deleteResources(resources...)

	ts := []testCase{
		{
			name:     "Create prometheus operator resources",
			scenario: createResources(resources...),
		},
		{
			name: "Operator should reconcile resources which it does owns",
			scenario: func(t *testing.T) {
				t.Run("Prometheus eventually exists", func(t *testing.T) {
					t.Parallel()
					f.AssertResourceEventuallyExists(prometheusStsName, e2eTestNamespace, &appsv1.StatefulSet{})(t)
				})
				t.Run("Alertmanager eventually exists", func(t *testing.T) {
					t.Parallel()
					f.AssertResourceEventuallyExists(alertmanagerStsName, e2eTestNamespace, &appsv1.StatefulSet{})(t)
				})
				t.Run("Thanos Ruler eventually exists", func(t *testing.T) {
					t.Parallel()
					f.AssertResourceEventuallyExists(thanosRulerStsName, e2eTestNamespace, &appsv1.StatefulSet{})(t)
				})
			},
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func newPrometheus(labels map[string]string) *v1.Prometheus {
	return &v1.Prometheus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "Prometheus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus",
			Namespace: e2eTestNamespace,
			Labels:    labels,
		},
	}
}

func newAlertmanager(labels map[string]string) *v1.Alertmanager {
	return &v1.Alertmanager{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "Prometheus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alertmanager",
			Namespace: e2eTestNamespace,
			Labels:    labels,
		},
	}
}

func newThanosRuler(labels map[string]string) *v1.ThanosRuler {
	return &v1.ThanosRuler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "Prometheus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thanosruler",
			Namespace: e2eTestNamespace,
			Labels:    labels,
		},
		Spec: v1.ThanosRulerSpec{
			QueryEndpoints: []string{"127.0.0.1"},
		},
	}
}

func createResources(resource ...client.Object) func(t *testing.T) {
	return func(t *testing.T) {
		for _, resource := range resource {
			if err := f.K8sClient.Create(context.Background(), resource); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func deleteResources(resource ...client.Object) {
	for _, resource := range resource {
		f.K8sClient.Delete(context.Background(), resource)
	}
}
