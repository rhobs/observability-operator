package e2e

import (
	"context"
	grafana_operator "rhobs/monitoring-stack-operator/pkg/controllers/grafana-operator"
	"testing"

	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"

	"github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	grafanaDeploymentName = "grafana-deployment"
	operatorNamespace     = "monitoring-stack-operator"
)

func TestControllerRestoresDeletedResources(t *testing.T) {
	resourceName := "monitoring-stack-operator-grafana-operator"
	ts := []testCase{
		{
			name: "Operator should create a Grafana Operator through OLM",
			scenario: func(t *testing.T) {
				f.AssertResourceEventuallyExists(operatorNamespace, "", &v1.Namespace{})(t)
				f.AssertResourceEventuallyExists(resourceName, operatorNamespace, &olmv1alpha1.Subscription{})(t)
				f.AssertResourceEventuallyExists(resourceName, operatorNamespace, &olmv1.OperatorGroup{})(t)
			},
		},
		{
			name: "Operator should restore deleted OLM Subscription",
			scenario: func(t *testing.T) {
				if err := f.K8sClient.Delete(context.Background(), grafana_operator.NewSubscription()); err != nil {
					t.Fatal(err)
				}
				f.AssertResourceEventuallyExists(resourceName, operatorNamespace, &olmv1alpha1.Subscription{})(t)
			},
		},
		{
			name: "Operator should restore deleted OLM OperatorGroup",
			scenario: func(t *testing.T) {
				if err := f.K8sClient.Delete(context.Background(), grafana_operator.NewOperatorGroup()); err != nil {
					t.Fatal(err)
				}
				f.AssertResourceEventuallyExists(resourceName, operatorNamespace, &olmv1.OperatorGroup{})(t)
			},
		},
		{
			name: "Operator should restore deleted namespace",
			scenario: func(t *testing.T) {
				if err := f.K8sClient.Delete(context.Background(), grafana_operator.NewNamespace()); err != nil {
					t.Fatal(err)
				}
				f.AssertResourceEventuallyExists(operatorNamespace, "", &v1.Namespace{})(t)
				f.AssertResourceEventuallyExists(resourceName, operatorNamespace, &olmv1alpha1.Subscription{})(t)
				f.AssertResourceEventuallyExists(resourceName, operatorNamespace, &olmv1.OperatorGroup{})(t)
			},
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func TestDefaultGrafanaInstanceIsCreated(t *testing.T) {
	ts := []testCase{
		{
			name: "Operator should reconcile resources in its own namespace",
			scenario: func(t *testing.T) {
				f.AssertResourceEventuallyExists(grafanaDeploymentName, operatorNamespace, &appsv1.Deployment{})(t)
			},
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func TestGrafanaOperatorForResourcesOutsideOfItsOwnNamespace(t *testing.T) {
	resources := []client.Object{
		newGrafana(e2eTestNamespace),
	}
	defer deleteResources(resources...)

	ts := []testCase{
		{
			name:     "Create grafana resource",
			scenario: createResources(resources...),
		},
		{
			name: "Operator should not reconcile resources outside of its own namespace",
			scenario: func(t *testing.T) {
				f.AssertResourceNeverExists(grafanaDeploymentName, e2eTestNamespace, &appsv1.Deployment{})(t)
			},
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func newGrafana(namespace string) *v1alpha1.Grafana {
	return &v1alpha1.Grafana{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "Grafana",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: namespace,
		},
	}
}
