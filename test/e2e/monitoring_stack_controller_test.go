package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/rhobs/monitoring-stack-operator/test/e2e/framework"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/util/wait"

	stack "github.com/rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"gotest.tools/v3/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type alert struct {
	Labels map[string]string
}

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
		}, {
			name:     "Prometheus stacks can scrape themselves",
			scenario: assertPrometheusScrapesItself,
		}, {
			name:     "Alertmanager receives alerts from the Prometheus instance",
			scenario: assertAlertmanagerReceivesAlerts,
		}, {
			name:     "Alertmanager instances are on different nodes",
			scenario: assertAlertmanagersAreOnDifferentNodes,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func emptyStackCreatesPrometheus(t *testing.T) {
	ms := newMonitoringStack(t, "empty-stack")
	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	// Creating an Empty monitoring stack must create a Prometheus with defaults applied
	prometheus := monv1.Prometheus{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &prometheus)
}

func reconcileStack(t *testing.T) {
	ms := newMonitoringStack(t, "reconcile-test")
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
	ms := newMonitoringStack(t, "revert-test")
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
}

func validateStackLogLevel(t *testing.T) {
	invalidLogLevels := []string{
		"foobar",
		"xyz",
		"Info",
		"Debug",
	}
	ms := newMonitoringStack(t, "invalid-loglevel-stack")
	for _, v := range invalidLogLevels {
		ms.Spec.LogLevel = stack.LogLevel(v)
		err := f.K8sClient.Create(context.Background(), ms)
		assert.ErrorContains(t, err, `spec.logLevel: Unsupported value`)
	}

	validMS := newMonitoringStack(t, "valid-loglevel")
	validMS.Spec.LogLevel = "debug"
	err := f.K8sClient.Create(context.Background(), validMS)
	assert.NilError(t, err, `debug is a valid loglevel`)
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

	ms := newMonitoringStack(t, "invalid-retention")
	for _, v := range invalidRetention {
		ms.Spec.Retention = v
		err := f.K8sClient.Create(context.Background(), ms)
		assert.ErrorContains(t, err, `spec.retention: Invalid value`)
	}

	validMS := newMonitoringStack(t, "valid-retention")
	validMS.Spec.Retention = "100h"

	err := f.K8sClient.Create(context.Background(), validMS)
	assert.NilError(t, err, `100h is a valid retention period`)
}

func assertPrometheusScrapesItself(t *testing.T) {
	ms := newMonitoringStack(t, "self-scrape")
	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err)
	f.AssertStatefulsetReady("prometheus-self-scrape", e2eTestNamespace, framework.WithTimeout(5*time.Minute))(t)

	stopChan := make(chan struct{})
	defer close(stopChan)
	if err := wait.Poll(5*time.Second, 2*time.Minute, func() (bool, error) {
		err = f.StartServicePortForward("self-scrape-prometheus", e2eTestNamespace, "9090", stopChan)
		return err == nil, nil
	}); err != nil {
		t.Fatal(err)
	}

	promClient := framework.NewPrometheusClient("http://localhost:9090")
	expectedResults := map[string]int{
		"prometheus_build_info":   1,
		"alertmanager_build_info": 2,
	}
	if err := wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
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
				resultErr := fmt.Errorf("invalid result for query %s, got %d, want %d", query, len(result.Data.Result), value)
				return true, resultErr
			}

			if len(result.Data.Result) != value {
				return false, nil
			}

			correct++
		}

		return correct == len(expectedResults), nil
	}); err != nil {
		t.Fatal(err)
	}
}

func assertAlertmanagersAreOnDifferentNodes(t *testing.T) {
	ms := newMonitoringStack(t, "alerting")
	if err := f.K8sClient.Create(context.Background(), ms); err != nil {
		t.Fatal(err)
	}
	f.AssertStatefulsetReady("alertmanager-alerting", e2eTestNamespace, framework.WithTimeout(2*time.Minute))(t)

	var pods corev1.PodList
	if err := f.K8sClient.List(context.Background(), &pods); err != nil {
		t.Fatal(err)
	}
	nodes := make([]string, 0, 2)
	for _, pod := range pods.Items {
		alertmanagerPod := false
		for _, owner := range pod.OwnerReferences {
			if owner.Name == "alertmanager-alerting" && owner.Kind == "StatefulSet" {
				alertmanagerPod = true
			}
		}
		if alertmanagerPod {
			nodes = append(nodes, pod.Spec.NodeName)
		}
	}

	if len(nodes) != 2 || nodes[0] == nodes[1] {
		t.Fatal(fmt.Errorf("expected alertmanager pods to run on different nodes"))
	}
}

func assertAlertmanagerReceivesAlerts(t *testing.T) {
	ms := newMonitoringStack(t, "alerting")
	if err := f.K8sClient.Create(context.Background(), ms); err != nil {
		t.Fatal(err)
	}

	rule := newAlerts(t)
	if err := f.K8sClient.Create(context.Background(), rule); err != nil {
		t.Fatal(err)
	}
	f.AssertStatefulsetReady("alertmanager-alerting", e2eTestNamespace, framework.WithTimeout(2*time.Minute))(t)

	stopChan := make(chan struct{})
	defer close(stopChan)
	if err := wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		err := f.StartServicePortForward("alerting-alertmanager", e2eTestNamespace, "9093", stopChan)
		return err == nil, nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		alerts, err := getAlertmanagerAlerts()
		if err != nil {
			return false, nil
		}

		if len(alerts) == 0 {
			return false, nil
		}

		if len(alerts) != 1 {
			return true, fmt.Errorf("too many alerts fired")
		}

		if alerts[0].Labels["alertname"] == "AlwaysOn" {
			return true, nil
		}

		return true, fmt.Errorf("wrong alert firing, got %s, want %s", alerts[0].Labels["alertname"], "AlwaysOn")
	}); err != nil {
		t.Fatal(err)
	}
}

func getAlertmanagerAlerts() ([]alert, error) {
	client := http.Client{}
	resp, err := client.Get("http://localhost:9093/api/v2/alerts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var alerts []alert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return nil, err
	}

	return alerts, nil
}

func newAlerts(t *testing.T) *monv1.PrometheusRule {
	rule := &monv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "always-on",
			Namespace: e2eTestNamespace,
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{
				{
					Name:     "Test",
					Interval: "10s",
					Rules: []monv1.Rule{
						{
							Alert: "AlwaysOn",
							Expr:  intstr.FromString("vector(1)"),
							For:   "1s",
						},
						{
							Alert: "NeverOn",
							Expr:  intstr.FromString("vector(1) == 0"),
							For:   "1s",
						},
					},
				},
			},
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), rule)
	})

	return rule
}

func newMonitoringStack(t *testing.T, name string) *stack.MonitoringStack {
	ms := &stack.MonitoringStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
	}
	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), ms)
		waitForStackDeletion(name)
	})

	return ms
}

func waitForStackDeletion(name string) error {
	return wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (bool, error) {
		key := types.NamespacedName{Name: name, Namespace: e2eTestNamespace}
		var ms stack.MonitoringStack
		err := f.K8sClient.Get(context.Background(), key, &ms)
		return errors.IsNotFound(err), nil
	})
}
