package framework

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"rhobs/monitoring-stack-operator/test/e2e/framework/prom"

	_ "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/promql"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func assertPromQL(t *testing.T, metrics []byte, query string, expected map[string]float64) {

	now := time.Now()
	points, err := prom.ParseTextData(metrics, now)
	if err != nil {
		t.Errorf("invalid raw data: %v", err)
	}

	// TODO: logger in opts
	// TODO: query logger?
	engine := promql.NewEngine(promql.EngineOpts{
		Timeout:    100000000 * time.Second, // this is what context is supposed to be for :-/
		MaxSamples: 1000,                    // TODO: find an ok value
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
			t.Errorf("Didn't expect this label combo: %v\n", v.Metric.String())
		}
		if v.V != s {
			t.Errorf("%s: got %v, want: %v", v.Metric.String(), v.V, s)
		}
	}
}

// GetOperatorPod gets the operator pod assuming the default deployment; i.e
// operator is deployed in default namespace
func (f *Framework) GetOperatorPod(t *testing.T) *v1.Pod {

	// get the operator deployment
	operator := appsv1.Deployment{}
	f.AssertResourceEventuallyExists("monitoring-stack-operator", "default", &operator)(t)

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

// AssertResourceNeverExists asserts that a statefulset is never created for the duration of wait.ForeverTestTimeout
func (f *Framework) AssertNoReconcileErrors(t *testing.T) {
	pod := f.GetOperatorPod(t)

	stopChan := make(chan struct{})
	defer close(stopChan)
	if err := wait.Poll(5*time.Second, wait.ForeverTestTimeout, func() (bool, error) {
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
	assertPromQL(t, metrics,
		`controller_runtime_reconcile_errors_total`,
		map[string]float64{
			`{__name__="controller_runtime_reconcile_errors_total", controller="grafana-operator"}`: 0,
			`{__name__="controller_runtime_reconcile_errors_total", controller="monitoringstack"}`:  0,
		})
}
