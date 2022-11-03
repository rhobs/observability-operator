package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"

	"github.com/rhobs/observability-operator/test/e2e/framework"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/google/go-cmp/cmp"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	monitoringstack "github.com/rhobs/observability-operator/pkg/controllers/monitoring/monitoring-stack"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"

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
		"prometheuses.monitoring.rhobs",
		"alertmanagers.monitoring.rhobs",
		"podmonitors.monitoring.rhobs",
		"monitoringstacks.monitoring.rhobs",
	)

	ts := []testCase{{
		name:     "Defaults are applied to Monitoring CR",
		scenario: promConfigDefaultsAreApplied,
	}, {
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
		name:     "single prometheus replica has no pdb",
		scenario: singlePrometheusReplicaHasNoPDB,
	}, {
		name:     "Prometheus stacks can scrape themselves",
		scenario: assertPrometheusScrapesItself,
	}, {
		name:     "Alertmanager receives alerts from the Prometheus instance",
		scenario: assertAlertmanagerReceivesAlerts,
	}, {
		name: "Alertmanager runs in HA mode",
		scenario: func(t *testing.T) {
			stackName := "alerting"
			assertAlertmanagerCreated(t, stackName)
			pods, err := f.GetStatefulSetPods("alertmanager-"+stackName, e2eTestNamespace)
			if err != nil {
				t.Fatal(err)
			}
			assertAlertmanagersAreOnDifferentNodes(t, pods)
			assertAlertmanagersAreResilientToDisruption(t, pods)
		},
	}, {
		name:     "invalid Prometheus replicas numbers",
		scenario: validatePrometheusConfig,
	}, {
		name:     "Alertmanager disabled",
		scenario: assertAlertmanagerNotDeployed,
	}, {
		name:     "Alertmanager deployed and removed",
		scenario: assertAlertmanagerDeployedAndRemoved,
	}, {
		name:     "Verify multi-namespace support",
		scenario: namespaceSelectorTest,
	}}

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

func promConfigDefaultsAreApplied(t *testing.T) {
	tests := []struct {
		name     string
		config   *stack.PrometheusConfig
		expected int32
	}{
		// creating an empty stack should have 2 replicas
		{"empty-stack", nil, 2},

		// creating a stack with replicas explictly set must honour that
		{"explict-replica", &stack.PrometheusConfig{Replicas: intPtr(1)}, 1},

		// creating an stack with a partical config (no replicas) should default to 2
		{
			name: "partial-config",
			config: &stack.PrometheusConfig{
				RemoteWrite: []monv1.RemoteWriteSpec{
					{URL: "https://foobar"},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.name)
			ms := newMonitoringStack(t, tt.name)
			ms.Spec.PrometheusConfig = tt.config

			err := f.K8sClient.Create(context.Background(), ms)
			assert.NilError(t, err, "failed to create a monitoring stack")

			created := stack.MonitoringStack{}
			f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &created)

			assert.Equal(t, tt.expected, *created.Spec.PrometheusConfig.Replicas)
		})

	}

}

func intPtr(i int32) *int32 {
	return &i
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
		Retention: ms.Spec.Retention,
		CommonPrometheusFields: monv1.CommonPrometheusFields{
			LogLevel:               string(ms.Spec.LogLevel),
			ServiceMonitorSelector: ms.Spec.ResourceSelector.DeepCopy(),
		},
	}

	assert.DeepEqual(t, expected.ServiceMonitorSelector, generated.Spec.ServiceMonitorSelector)
	assert.Equal(t, expected.LogLevel, generated.Spec.LogLevel)
	assert.Equal(t, expected.Retention, generated.Spec.Retention)

	availableMs := f.GetStackWhenAvailable(t, ms.Name, ms.Namespace)
	availableC := getConditionByType(availableMs.Status.Conditions, stack.AvailableCondition)
	assertCondition(t, availableC, monitoringstack.AvailableReason, stack.AvailableCondition, availableMs)
	reconciledC := getConditionByType(availableMs.Status.Conditions, stack.ReconciledCondition)
	assertCondition(t, reconciledC, monitoringstack.ReconciledReason, stack.ReconciledCondition, availableMs)
}

func assertCondition(t *testing.T, c *stack.Condition, reason string, ctype stack.ConditionType, ms stack.MonitoringStack) {
	assert.Check(t, c != nil, "failed to find %s status condition for %s monitoring stack", ctype, ms.Name)
	assert.Check(t, c.Status == stack.ConditionTrue, "unexpected %s condition status", ctype)
	assert.Check(t, c.Reason == reason, "unexpected %s condition reason", ctype)
}

func getConditionByType(conditions []stack.Condition, ctype stack.ConditionType) *stack.Condition {
	for _, c := range conditions {
		if c.Type == ctype {
			return &c
		}
	}
	return nil
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

	err = wait.Poll(5*time.Second, time.Minute, func() (bool, error) {
		reconciled := monv1.Prometheus{}
		key := types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}

		if err := f.K8sClient.Get(context.Background(), key, &reconciled); errors.IsNotFound(err) {
			// retry
			return false, nil
		}

		if diff := cmp.Diff(generated.Spec, reconciled.Spec); diff != "" {
			t.Logf("Mismatch in prometheus spec, retrying (-want +got):\n%s", diff)
			// retry
			return false, nil
		}
		return true, nil
	})
	assert.NilError(t, err, "failed to revert manual changes to prometheus spec")
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
	invalidRetention := []monv1.Duration{
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

func validatePrometheusConfig(t *testing.T) {
	invalidReplicasValues := []int32{-1, 0}
	ms := newMonitoringStack(t, "invalid-prometheus-config")
	for _, v := range invalidReplicasValues {
		ms.Spec.PrometheusConfig = &stack.PrometheusConfig{
			Replicas: &v,
		}
		err := f.K8sClient.Create(context.Background(), ms)
		assert.ErrorContains(t, err, `invalid: spec.prometheusConfig.replicas`)
	}

	validN := int32(1)
	ms.Spec.PrometheusConfig.Replicas = &validN
	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, `1 is a valid replica count`)
}

func singlePrometheusReplicaHasNoPDB(t *testing.T) {
	// asserts that no prometheus pdb is created for stacks with replicas set to 1

	// Initially, ensure that pdb is created by default for the default stack.
	// This should later be removed when replicas is set to 1
	ms := newMonitoringStack(t, "single-replica")

	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	// ensure pdb is created for default stack
	pdb := policyv1.PodDisruptionBudget{}
	pdbName := ms.Name + "-prometheus"
	f.AssertResourceEventuallyExists(pdbName, ms.Namespace, &pdb)(t)

	// Update replica count to 1 and assert that pdb is removed
	key := types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}
	err = f.K8sClient.Get(context.Background(), key, ms)
	assert.NilError(t, err, "failed to get a monitoring stack")

	ms.Spec.PrometheusConfig.Replicas = intPtr(1)
	err = f.K8sClient.Update(context.Background(), ms)
	assert.NilError(t, err, "failed to update monitoring stack")

	// ensure there is no pdb
	f.AssertResourceNeverExists(pdbName, ms.Namespace, &pdb)(t)
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
		"prometheus_build_info":   2, // scrapes from both endpoints
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

func assertAlertmanagerNotDeployed(t *testing.T) {
	ms := newMonitoringStack(t, "no-alertmanager", func(ms *stack.MonitoringStack) {
		ms.Spec.AlertmanagerConfig.Disabled = true
	})
	if err := f.K8sClient.Create(context.Background(), ms); err != nil {
		t.Fatal(err)
	}
	_ = f.GetStackWhenAvailable(t, ms.Name, ms.Namespace)
	f.AssertAlertmanagerAbsent(t, ms.Name, ms.Namespace)
}

func assertAlertmanagerDeployedAndRemoved(t *testing.T) {
	ms := newMonitoringStack(t, "alertmanager-deployed-and-removed")
	if err := f.K8sClient.Create(context.Background(), ms); err != nil {
		t.Fatal(err)
	}
	updatedMS := f.GetStackWhenAvailable(t, ms.Name, ms.Namespace)
	var am monv1.Alertmanager
	key := types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}
	err := f.K8sClient.Get(context.Background(), key, &am)
	assert.NilError(t, err)

	updatedMS.Spec.AlertmanagerConfig.Disabled = true
	err = f.K8sClient.Update(context.Background(), &updatedMS)
	assert.NilError(t, err)

	f.AssertAlertmanagerAbsent(t, updatedMS.Name, updatedMS.Namespace)
}

func assertAlertmanagerCreated(t *testing.T, name string) {
	ms := newMonitoringStack(t, name)
	if err := f.K8sClient.Create(context.Background(), ms); err != nil {
		t.Fatal(err)
	}
	f.AssertStatefulsetReady("alertmanager-"+name, e2eTestNamespace, framework.WithTimeout(2*time.Minute))(t)
}

func assertAlertmanagersAreOnDifferentNodes(t *testing.T, pods []corev1.Pod) {
	nodeAllocations := make(map[string]struct{})
	for _, pod := range pods {
		if _, ok := nodeAllocations[pod.Spec.NodeName]; ok {
			err := fmt.Errorf("expected alertmanager pods to run on different nodes")
			t.Fatal(err)
		}
		nodeAllocations[pod.Spec.NodeName] = struct{}{}
	}
}

func assertAlertmanagersAreResilientToDisruption(t *testing.T, pods []corev1.Pod) {
	for i, pod := range pods {
		lastPod := i == len(pods)-1
		err := f.Evict(&pod, 0)
		if lastPod && err == nil {
			t.Fatal("expected an error when evicting the last pod, got nil")
		}
		if !lastPod && err != nil {
			t.Fatalf("expected no error when evicting pod with index %d, got %v", i, err)
		}
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

type stackMod func(*stack.MonitoringStack)

func msResourceSelector(labels map[string]string) stackMod {
	return func(ms *stack.MonitoringStack) {
		ms.Spec.ResourceSelector = &metav1.LabelSelector{MatchLabels: labels}
	}
}
func msNamespaceSelector(labels map[string]string) stackMod {
	return func(ms *stack.MonitoringStack) {
		ms.Spec.NamespaceSelector = &metav1.LabelSelector{MatchLabels: labels}
	}
}

func newMonitoringStack(t *testing.T, name string, mods ...stackMod) *stack.MonitoringStack {
	ms := &stack.MonitoringStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
	}
	for _, mod := range mods {
		mod(ms)
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

func namespaceSelectorTest(t *testing.T) {
	// as a convention
	// add labels to ns to indicate the stack responsible for monitoring the ns
	// resourceSelector uses both stack and an app label
	stackName := "multi-ns"
	nsLabels := map[string]string{"monitoring.rhobs/stack": stackName}
	resourceLabels := map[string]string{
		"monitoring.rhobs/stack": stackName,
		"app":                    "demo",
	}

	ms := newMonitoringStack(t, stackName,
		msResourceSelector(resourceLabels),
		msNamespaceSelector(nsLabels))

	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	namespaces := []string{"test-ns-1", "test-ns-2", "test-ns-3"}

	for _, ns := range namespaces {
		err := deployDemoApp(t, ns, nsLabels, resourceLabels)
		assert.NilError(t, err, "%s: deploying demo app failed", ns)
	}

	stopChan := make(chan struct{})
	defer close(stopChan)
	if pollErr := wait.Poll(5*time.Second, 2*time.Minute, func() (bool, error) {
		err := f.StartServicePortForward(ms.Name+"-prometheus", e2eTestNamespace, "9090", stopChan)
		return err == nil, nil
	}); pollErr != nil {
		t.Fatal(pollErr)
	}

	promClient := framework.NewPrometheusClient("http://localhost:9090")
	if pollErr := wait.Poll(5*time.Second, 5*time.Minute, func() (bool, error) {
		query := `prometheus_build_info{namespace=~"test-ns-.*"}`
		result, err := promClient.Query(query)
		if err != nil {
			return false, nil
		}

		if len(result.Data.Result) != len(namespaces) {
			return false, nil
		}

		return true, nil
	}); pollErr != nil {
		t.Fatal(pollErr)
	}
}

func deployDemoApp(t *testing.T, nsName string, nsLabels, resourceLabels map[string]string) error {

	// now deploy a prometheus instance into test-ns
	ns := newNamespace(t, nsName)
	ns.SetLabels(nsLabels)
	if err := f.K8sClient.Create(context.Background(), ns); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", ns, err)
	}

	// deploy a pod, service, service-monitor into that namespace
	app := newPrometheusPod(t, "prometheus", ns.Name, resourceLabels)
	if err := f.K8sClient.Create(context.Background(), app); err != nil {
		return fmt.Errorf("failed to create demo app %s/%s: %w", nsName, app.Name, err)
	}

	svcLabels := map[string]string{"service": app.Name}
	svc := newService(t, app.Name, ns.Name, svcLabels, app.Labels)
	// these are prometheus ports
	svc.Spec.Ports = []corev1.ServicePort{{
		Name:       "metrics",
		Port:       9090,
		TargetPort: intstr.FromInt(9090),
	}}

	if err := f.K8sClient.Create(context.Background(), svc); err != nil {
		return fmt.Errorf("failed to create service for demo app %s/%s: %w", nsName, svc.Name, err)
	}

	svcMon := newServiceMonitor(t, ns.Name, "prometheus", resourceLabels, svcLabels, "metrics")
	if err := f.K8sClient.Create(context.Background(), svcMon); err != nil {
		return fmt.Errorf("failed to create servicemonitor for demo service %s/%s: %w", nsName, svcMon.Name, err)
	}
	return nil
}

func newServiceMonitor(t *testing.T, ns, name string, labels, svcLabels map[string]string, endpoint string) *monv1.ServiceMonitor {
	svcMon := &monv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: monv1.ServiceMonitorSpec{
			Selector:  metav1.LabelSelector{MatchLabels: svcLabels},
			Endpoints: []monv1.Endpoint{{Port: endpoint}},
		},
	}
	f.CleanUp(t, func() { f.K8sClient.Delete(context.Background(), svcMon) })
	return svcMon
}

func newNamespace(t *testing.T, name string) *corev1.Namespace {
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	f.CleanUp(t, func() { f.K8sClient.Delete(context.Background(), ns) })
	return ns
}

func newService(t *testing.T, name, namespace string, labels, selector map[string]string) *corev1.Service {
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
		},
	}

	f.CleanUp(t, func() { f.K8sClient.Delete(context.Background(), svc) })
	return svc
}

func newPrometheusPod(t *testing.T, name, ns string, labels map[string]string) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "prometheus",
				Image: "quay.io/prometheus/prometheus:v2.39.1",
				Ports: []corev1.ContainerPort{{
					Name:          "metrics",
					ContainerPort: 9090,
				}},
			}},
		},
	}
	pod.Labels["app.kubernetes.io/name"] = name

	f.CleanUp(t, func() {
		f.K8sClient.Delete(context.Background(), pod)
	})
	return pod
}
