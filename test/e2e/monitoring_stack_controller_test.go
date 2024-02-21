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
	"k8s.io/utils/ptr"

	"github.com/rhobs/observability-operator/test/e2e/framework"

	"golang.org/x/exp/slices"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/google/go-cmp/cmp"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	monitoringstack "github.com/rhobs/observability-operator/pkg/controllers/monitoring/monitoring-stack"
	operator "github.com/rhobs/observability-operator/pkg/operator"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"

	"gotest.tools/v3/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/scheme"
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
	err := stack.AddToScheme(scheme.Scheme)
	assert.NilError(t, err, "adding stack to scheme failed")
	assertCRDExists(t,
		"prometheuses.monitoring.rhobs",
		"alertmanagers.monitoring.rhobs",
		"podmonitors.monitoring.rhobs",
		"servicemonitors.monitoring.rhobs",
		"monitoringstacks.monitoring.rhobs",
	)

	ts := []testCase{{
		name:     "Defaults are applied to Monitoring CR",
		scenario: promConfigDefaultsAreApplied,
	}, {
		name:     "Empty stack spec must create a Prometheus",
		scenario: emptyStackCreatesPrometheus,
	}, {
		name:     "resource selector nil propagates to Prometheus",
		scenario: nilResrouceSelectorPropagatesToPrometheus,
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
	}, {
		name:     "Verify ability to scale down Prometheus",
		scenario: prometheusScaleDown,
	}, {
		name:     "managed fields in Prometheus object",
		scenario: assertPrometheusManagedFields,
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

func nilResrouceSelectorPropagatesToPrometheus(t *testing.T) {
	ms := newMonitoringStack(t, "nil-selector")
	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")
	f.AssertResourceEventuallyExists(ms.Name, ms.Namespace, &monv1.Prometheus{})(t)

	updatedMS := &stack.MonitoringStack{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, updatedMS)
	err = f.UpdateWithRetry(t, updatedMS, framework.SetResourceSelector(nil))
	assert.NilError(t, err, "failed to patch monitoring stack with nil resource selector")

	prometheus := monv1.Prometheus{}
	err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, framework.DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
		if err := f.K8sClient.Get(context.Background(), types.NamespacedName{Name: updatedMS.Name, Namespace: updatedMS.Namespace}, &prometheus); errors.IsNotFound(err) {
			return false, nil
		}

		if prometheus.Spec.ServiceMonitorSelector != updatedMS.Spec.ResourceSelector {
			return false, nil
		}
		return true, nil
	})

	if wait.Interrupted(err) {
		t.Fatal(fmt.Errorf("nil ResourceSelector did not propagate to Prometheus object"))
	}
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

	err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, framework.DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
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
		"Warning",
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
	invalidN := int32(-1)
	ms := newMonitoringStack(t, "invalid-prometheus-config")
	ms.Spec.PrometheusConfig = &stack.PrometheusConfig{
		Replicas: &invalidN,
	}
	err := f.K8sClient.Create(context.Background(), ms)
	assert.ErrorContains(t, err, `invalid: spec.prometheusConfig.replicas`)

	validN := int32(1)
	ms.Spec.PrometheusConfig.Replicas = &validN
	err = f.K8sClient.Create(context.Background(), ms)
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

	// Update replica count to 1
	err = f.UpdateWithRetry(t, ms, framework.SetPrometheusReplicas(intPtr(1)))
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
	if err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		err = f.StartServicePortForward("self-scrape-prometheus", e2eTestNamespace, "9090", stopChan)
		return err == nil, nil
	}); err != nil {
		t.Fatal(fmt.Errorf("Failed to poll for port-forward: %w", err))
	}

	promClient := framework.NewPrometheusClient("http://localhost:9090")
	expectedResults := map[string]int{
		"prometheus_build_info":   2, // scrapes from both endpoints
		"alertmanager_build_info": 2,
	}
	if err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
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
		t.Fatal(fmt.Errorf("Could not query prometheus: %w", err))
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
	err = f.UpdateWithRetry(t, &updatedMS, framework.SetAlertmanagerDisabled(true))
	assert.NilError(t, err, "failed to update monitoring stack to disable alertmanager")

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
	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		err := f.StartServicePortForward("alerting-alertmanager", e2eTestNamespace, "9093", stopChan)
		return err == nil, nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
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

func prometheusScaleDown(t *testing.T) {
	numOfRep := int32(1)
	ms := newMonitoringStack(t, "prometheus-scale-down-test")
	ms.Spec.PrometheusConfig = &stack.PrometheusConfig{
		Replicas: &numOfRep,
	}

	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	prom := monv1.Prometheus{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &prom)

	assert.Equal(t, prom.Status.Replicas, int32(1))

	err = f.UpdateWithRetry(t, ms, framework.SetPrometheusReplicas(intPtr(0)))
	key := types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}
	assert.NilError(t, err, "failed to update a monitoring stack")
	err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, framework.DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
		if err := f.K8sClient.Get(context.Background(), key, &prom); errors.IsNotFound(err) {
			return false, nil
		}

		if prom.Status.Replicas != 0 {
			return false, nil
		}
		return true, nil
	})

	if wait.Interrupted(err) {
		t.Fatal(fmt.Errorf("Prometheus was not scaled down"))
	}
}

func assertPrometheusManagedFields(t *testing.T) {
	numOfRep := int32(1)
	ms := newMonitoringStack(t, "prometheus-managed-fields-test")
	var scrapeInterval monv1.Duration = "2m"
	ms.Spec.PrometheusConfig = &stack.PrometheusConfig{
		Replicas:       &numOfRep,
		ScrapeInterval: &scrapeInterval,
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimSpec{
			VolumeName: "prom-store",
		},
		RemoteWrite: []monv1.RemoteWriteSpec{
			{
				Name: "sample-remote-write",
				URL:  "https://sample-url",
			},
		},
		ExternalLabels: map[string]string{
			"key": "value",
		},
		EnableRemoteWriteReceiver: true,
	}
	ms.Spec.NamespaceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"label": "label-value",
		},
	}

	err := f.K8sClient.Create(context.Background(), ms)
	assert.NilError(t, err, "failed to create a monitoring stack")

	prom := monv1.Prometheus{}
	f.GetResourceWithRetry(t, ms.Name, ms.Namespace, &prom)

	mfs := prom.GetManagedFields()

	idx := slices.IndexFunc(mfs, func(mf metav1.ManagedFieldsEntry) bool {
		return mf.Manager == operator.ObservabilityOperatorName
	})

	if idx == -1 {
		t.Fatal(fmt.Errorf("no fields managed by observability-operator found"))
	}
	oboManagedFields := mfs[idx]
	s, err := json.MarshalIndent(oboManagedFields.FieldsV1, "", "  ")
	assert.NilError(t, err)

	var objmap map[string]interface{}
	err = json.Unmarshal(s, &objmap)
	assert.NilError(t, err)
	have := objmap["f:spec"]

	var expected map[string]interface{}
	_ = json.Unmarshal([]byte(oboManagedFieldsJson), &expected)

	assert.DeepEqual(t, have, expected)
}

// Update this json when a new Prometheus field is set by MonitoringStack
const oboManagedFieldsJson = `
{
  "f:additionalScrapeConfigs": {},
  "f:affinity": {
    "f:podAntiAffinity": {
      "f:requiredDuringSchedulingIgnoredDuringExecution": {}
    }
  },
  "f:alerting": {
    "f:alertmanagers": {}
  },
  "f:arbitraryFSAccessThroughSMs": {},
  "f:enableRemoteWriteReceiver": {},
  "f:externalLabels": {
    "f:key": {}
  },
  "f:image": {},
  "f:logLevel": {},
  "f:podMetadata": {
    "f:labels": {
      "f:app.kubernetes.io/component": {},
      "f:app.kubernetes.io/part-of": {}
    }
  },
  "f:podMonitorNamespaceSelector": {},
  "f:podMonitorSelector": {},
  "f:probeNamespaceSelector": {},
  "f:probeSelector": {},
  "f:remoteWrite": {},
  "f:replicas": {},
  "f:resources": {},
  "f:retention": {},
  "f:ruleNamespaceSelector": {},
  "f:ruleSelector": {},
  "f:rules": {
    "f:alert": {}
  },
  "f:scrapeConfigNamespaceSelector": {},
  "f:scrapeConfigSelector": {},
  "f:scrapeInterval": {},
  "f:securityContext": {
    "f:fsGroup": {},
    "f:runAsNonRoot": {},
    "f:runAsUser": {}
  },
  "f:serviceAccountName": {},
  "f:serviceMonitorNamespaceSelector": {},
  "f:serviceMonitorSelector": {},
  "f:storage": {
    "f:volumeClaimTemplate": {
      "f:metadata": {},
      "f:spec": {
        "f:resources": {},
        "f:volumeName": {}
      },
      "f:status": {}
    }
  },
  "f:thanos": {
    "f:image": {},
    "f:resources": {}
  },
  "f:tsdb": {}
}
`

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
					Interval: ptr.To(monv1.Duration("10s")),
					Rules: []monv1.Rule{
						{
							Alert: "AlwaysOn",
							Expr:  intstr.FromString("vector(1)"),
							For:   ptr.To(monv1.Duration("1s")),
						},
						{
							Alert: "NeverOn",
							Expr:  intstr.FromString("vector(1) == 0"),
							For:   ptr.To(monv1.Duration("1s")),
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

type stackModifier func(*stack.MonitoringStack)

func msResourceSelector(labels map[string]string) stackModifier {
	return func(ms *stack.MonitoringStack) {
		ms.Spec.ResourceSelector = &metav1.LabelSelector{MatchLabels: labels}
	}
}
func msNamespaceSelector(labels map[string]string) stackModifier {
	return func(ms *stack.MonitoringStack) {
		ms.Spec.NamespaceSelector = &metav1.LabelSelector{MatchLabels: labels}
	}
}

func newMonitoringStack(t *testing.T, name string, mods ...stackModifier) *stack.MonitoringStack {
	ms := &stack.MonitoringStack{
		TypeMeta: metav1.TypeMeta{
			APIVersion: stack.GroupVersion.String(),
			Kind:       "MonitoringStack",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: stack.MonitoringStackSpec{
			ResourceSelector: &metav1.LabelSelector{},
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
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, framework.DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
		key := types.NamespacedName{Name: name, Namespace: e2eTestNamespace}
		var ms stack.MonitoringStack
		err := f.K8sClient.Get(context.Background(), key, &ms)
		return errors.IsNotFound(err), nil
	})
}

// tests if a stack with a namespace selector is able to monitor
// resources from multiple namespaces
func namespaceSelectorTest(t *testing.T) {
	// as a convention, add labels to ns to indicate the stack responsible for
	// monitoring the namespaces
	// while resourceSelector uses both stack and an app label
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
	//nolint
	if pollErr := wait.Poll(15*time.Second, framework.DefaultTestTimeout, func() (bool, error) {
		err := f.StartServicePortForward(ms.Name+"-prometheus", e2eTestNamespace, "9090", stopChan)
		return err == nil, nil
	}); pollErr != nil {
		t.Fatal(pollErr)
	}

	promClient := framework.NewPrometheusClient("http://localhost:9090")
	if pollErr := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, framework.DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
		query := `version{pod="prometheus-example-app",namespace=~"test-ns-.*"}`
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

// Deploys a prometheus instance and a service pointing to the prometheus's port - 9090
// and a service-monitor to nsName namespace. nsLabels are applied to the namespace
// so that it can be monitored. resourceLabels are applied to the service monitor
func deployDemoApp(t *testing.T, nsName string, nsLabels, resourceLabels map[string]string) error {

	ns := newNamespace(t, nsName)
	ns.SetLabels(nsLabels)
	if err := f.K8sClient.Create(context.Background(), ns); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", ns, err)
	}

	// deploy a pod, service, service-monitor into that namespace
	prom := newPrometheusExampleAppPod(t, "prometheus-example-app", ns.Name)
	if err := f.K8sClient.Create(context.Background(), prom); err != nil {
		return fmt.Errorf("failed to create demo app %s/%s: %w", nsName, prom.Name, err)
	}

	svcLabels := map[string]string{
		"app.kubernetes.io/name":    prom.Name,
		"app.kubernetes.io/part-of": "prometheus",
	}
	svc := newService(t, prom.Name, ns.Name, svcLabels, prom.Labels)
	// these are prometheus ports
	svc.Spec.Ports = []corev1.ServicePort{{
		Name:       "metrics",
		Port:       8080,
		TargetPort: intstr.FromInt(8080),
	}}

	if err := f.K8sClient.Create(context.Background(), svc); err != nil {
		return fmt.Errorf("failed to create service for demo app %s/%s: %w", nsName, svc.Name, err)
	}

	svcMon := newServiceMonitor(t, ns.Name, "prometheus-example-app", resourceLabels, svcLabels, "metrics")
	if err := f.K8sClient.Create(context.Background(), svcMon); err != nil {
		return fmt.Errorf("failed to create servicemonitor for demo service %s/%s: %w", nsName, svcMon.Name, err)
	}
	return nil
}

func newServiceMonitor(t *testing.T, ns, name string, stackSelector, serviceSelector map[string]string, endpoint string) *monv1.ServiceMonitor {
	svcMon := &monv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    stackSelector,
		},
		Spec: monv1.ServiceMonitorSpec{
			Selector:  metav1.LabelSelector{MatchLabels: serviceSelector},
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

func newPrometheusExampleAppPod(t *testing.T, name, ns string) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "prometheus",
				"app.kubernetes.io/version": "multiarch-v0.4.1",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "prometheus-example-app",
				// This image is rebuild of the `prometheus-example-app` available on GitHub:
				// https://github.com/brancz/prometheus-example-app

				// The rebuild includes multi-arch support, as indicated by the link:
				// https://quay.io/repository/openshifttest/prometheus-example-app/manifest/sha256:382dc349f82d730b834515e402b48a9c7e2965d0efbc42388bd254f424f6193e

				// Additionally, this image is accessible on an OCP disconnected cluster,
				// allowing tests to be run in that environment.
				Image: "quay.io/openshifttest/prometheus-example-app@sha256:382dc349f82d730b834515e402b48a9c7e2965d0efbc42388bd254f424f6193e",
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: ptr.To(false),
					SeccompProfile: &corev1.SeccompProfile{
						Type: "RuntimeDefault",
					},
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{
							"ALL",
						},
					},
				},
				Ports: []corev1.ContainerPort{{
					Name:          "metrics",
					ContainerPort: 8080,
				}},
			}},
		},
	}

	f.CleanUp(t, func() { f.K8sClient.Delete(context.Background(), pod) })
	return pod
}
