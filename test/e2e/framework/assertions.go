package framework

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/prometheus/promql"
	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/instrumentation-tools/promq/prom"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const CustomForeverTestTimeout = 40 * time.Second

type AssertOption struct {
	PollInterval time.Duration
	WaitTimeout  time.Duration
}

type OptionFn func(*AssertOption)

func WithTimeout(d time.Duration) OptionFn {
	return func(o *AssertOption) {
		o.WaitTimeout = d
	}
}

func WithPollInterval(d time.Duration) OptionFn {
	return func(o *AssertOption) {
		o.PollInterval = d
	}
}

// AssertResourceNeverExists asserts that a statefulset is never created for the duration of CustomForeverTestTimeout
func (f *Framework) AssertResourceNeverExists(name, namespace string, resource client.Object, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  CustomForeverTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}

	return func(t *testing.T) {
		if err := wait.Poll(option.PollInterval, option.WaitTimeout, func() (done bool, err error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, resource); errors.IsNotFound(err) {
				return false, nil
			}

			return true, fmt.Errorf("resource %s/%s should not have been created", namespace, name)
		}); err != wait.ErrWaitTimeout {
			t.Fatal(err)
		}
	}
}

// AssertResourceEventuallyExists asserts that a resource is created duration a time period of CustomForeverTestTimeout
func (f *Framework) AssertResourceEventuallyExists(name, namespace string, resource client.Object, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  CustomForeverTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}

	return func(t *testing.T) {
		if err := wait.Poll(option.PollInterval, option.WaitTimeout, func() (done bool, err error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, resource); err == nil {
				return true, nil
			}
			return false, nil
		}); err == wait.ErrWaitTimeout {
			t.Fatal(fmt.Errorf("resource %s/%s was never created", namespace, name))
		}
	}
}

// AssertStatefulsetReady asserts that a statefulset has the desired number of pods running
func (f *Framework) AssertStatefulsetReady(name, namespace string, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  CustomForeverTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}
	return func(t *testing.T) {
		key := types.NamespacedName{Name: name, Namespace: namespace}
		if err := wait.Poll(5*time.Second, option.WaitTimeout, func() (bool, error) {
			pod := &appsv1.StatefulSet{}
			err := f.K8sClient.Get(context.Background(), key, pod)
			return err == nil && pod.Status.ReadyReplicas == *pod.Spec.Replicas, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *Framework) GetResourceWithRetry(t *testing.T, name, namespace string, obj client.Object) {
	err := wait.Poll(5*time.Second, CustomForeverTestTimeout, func() (bool, error) {
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

func assertPromQL(t *testing.T, metrics []byte, query string, expected map[string]float64) {

	now := time.Now()
	points, err := prom.ParseTextData(metrics, now)
	if err != nil {
		t.Errorf("invalid raw data: %v", err)
	}

	// TODO: logger in opts
	// TODO: query logger?
	engine := promql.NewEngine(promql.EngineOpts{
		Timeout:    100000000 * time.Second,
		MaxSamples: 1000, // TODO: find an ok value
	})

	storage := prom.NewRangeStorage()
	if err := storage.LoadData(points); err != nil {
		t.Errorf("unable to load data: %v", err)
	}

	iQuery, err := engine.NewInstantQuery(storage, query, now)
	if err != nil {
		t.Errorf("error creating query: %v", err)
	}

	res := iQuery.Exec(context.TODO())
	defer iQuery.Close()

	if res.Err != nil {
		t.Errorf("error running query: %v", res.Err)
	}

	if len(res.Warnings) > 0 {
		for _, warning := range res.Warnings {
			t.Logf("warning running query: %v", warning)
		}
	}

	vec, err := res.Vector()
	if res.Err != nil {
		t.Errorf("error converting to scalar: %v", err)
	}

	for _, v := range vec {
		s, ok := expected[v.Metric.String()]
		if !ok {
			t.Errorf("Unexpected labelset: %v\n", v.Metric.String())
		}
		if v.V != s {
			t.Errorf("%s: got %v, want: %v", v.Metric.String(), v.V, s)
		}
	}
}

// GetOperatorPod gets the operator pod assuming the operator is deployed in
// "operators" namespace.
func (f *Framework) GetOperatorPod(t *testing.T) *v1.Pod {

	// get the operator deployment
	operator := appsv1.Deployment{}
	f.AssertResourceEventuallyExists("observability-operator", "operators", &operator)(t)

	selector, err := metav1.LabelSelectorAsSelector(operator.Spec.Selector)
	if err != nil {
		t.Error(err)
	}

	// get the operator pods for the deployment
	listOptions := []client.ListOption{
		client.InNamespace(operator.Namespace),
		client.MatchingLabelsSelector{Selector: selector},
	}

	pods := v1.PodList{}
	err = f.K8sClient.List(context.Background(), &pods, listOptions...)
	if err != nil {
		t.Error("failed to get opeator pods: ", err)
	}

	if len(pods.Items) != 1 {
		t.Error("Expected 1 operator pod but got:", len(pods.Items))
	}

	return &pods.Items[0]
}

func (f *Framework) GetOperatorMetrics(t *testing.T) []byte {
	pod := f.GetOperatorPod(t)

	stopChan := make(chan struct{})
	defer close(stopChan)
	if err := wait.Poll(5*time.Second, CustomForeverTestTimeout, func() (bool, error) {
		err := f.StartPortForward(pod.Name, pod.Namespace, "8080", stopChan)
		return err == nil, nil
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get("http://localhost:8080/metrics")
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	metrics, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	return metrics
}

// AssertNoReconcileErrors asserts that there are no reconcilation errors
func (f *Framework) AssertNoReconcileErrors(t *testing.T) {
	metrics := f.GetOperatorMetrics(t)
	assertPromQL(t, metrics,
		`controller_runtime_reconcile_errors_total`,
		map[string]float64{
			`{__name__="controller_runtime_reconcile_errors_total", controller="grafana-operator"}`: 0,
			`{__name__="controller_runtime_reconcile_errors_total", controller="monitoringstack"}`:  0,
			`{__name__="controller_runtime_reconcile_errors_total", controller="thanosquerier"}`:    0,
		})
}

func (f *Framework) GetStackWhenAvailable(t *testing.T, name, namespace string) v1alpha1.MonitoringStack {
	var ms v1alpha1.MonitoringStack
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := wait.Poll(5*time.Second, CustomForeverTestTimeout, func() (bool, error) {
		err := f.K8sClient.Get(context.Background(), key, &ms)
		if err != nil {
			return false, nil
		}
		availableC := getConditionByType(ms.Status.Conditions, v1alpha1.AvailableCondition)
		if availableC != nil && availableC.Status == v1alpha1.ConditionTrue {
			return true, nil
		}
		return false, nil
	})

	if err == wait.ErrWaitTimeout {
		t.Fatal(fmt.Errorf("resource %s/%s was not available", namespace, name))
	}
	return ms
}

func (f *Framework) AssertAlertmanagerAbsent(t *testing.T, name, namespace string) {
	var am monv1.Alertmanager
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := wait.Poll(5*time.Second, CustomForeverTestTimeout, func() (bool, error) {
		err := f.K8sClient.Get(context.Background(), key, &am)
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	if err == wait.ErrWaitTimeout {
		t.Fatal(fmt.Errorf("alertmanager %s/%s is present when expected to be absent", namespace, name))
	}
}

func getConditionByType(conditions []v1alpha1.Condition, ctype v1alpha1.ConditionType) *v1alpha1.Condition {
	for _, c := range conditions {
		if c.Type == ctype {
			return &c
		}
	}
	return nil
}
