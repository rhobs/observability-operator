package framework

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

// default ForeverTestTimeout is 30, some test fail because they take more than 30s
// change to custom in order to let the test finish withouth errors
const DefaultTestTimeout = 40 * time.Second

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

// AssertResourceNeverExists asserts that a resource is never created for the
// duration of the timeout
func (f *Framework) AssertResourceNeverExists(name, namespace string, resource client.Object, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}

	return func(t *testing.T) {
		//nolint
		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, resource); apierrors.IsNotFound(err) {
				return false, nil
			}

			return true, fmt.Errorf("resource %s/%s should not have been created", namespace, name)
		}); !wait.Interrupted(err) {
			t.Fatal(err)
		}
	}
}

// AssertResourceAbsent asserts that a resource is not present or, if present, is deleted
// within the timeout
func (f *Framework) AssertResourceAbsent(name, namespace string, resource client.Object, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}

	return func(t *testing.T) {
		//nolint
		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, resource); apierrors.IsNotFound(err) {
				return true, nil
			}

			return false, fmt.Errorf("resource %s/%s should not be present", namespace, name)
		}); wait.Interrupted(err) {
			t.Fatal(err)
		}
	}
}

// AssertResourceEventuallyExists asserts that a resource is created duration a time period of customForeverTestTimeout
func (f *Framework) AssertResourceEventuallyExists(name, namespace string, resource client.Object, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}

	return func(t *testing.T) {
		t.Helper()
		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, resource); err == nil {
				return true, nil
			}
			return false, nil
		}); wait.Interrupted(err) {
			t.Fatal(fmt.Errorf("resource %s/%s was never created", namespace, name))
		}
	}
}

// AssertStatefulsetReady asserts that a statefulset has the desired number of pods running
func (f *Framework) AssertStatefulsetReady(name, namespace string, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}
	return func(t *testing.T) {
		t.Helper()
		key := types.NamespacedName{Name: name, Namespace: namespace}
		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {
			pod := &appsv1.StatefulSet{}
			err := f.K8sClient.Get(context.Background(), key, pod)
			return err == nil && pod.Status.ReadyReplicas == *pod.Spec.Replicas, nil
		}); err != nil {
			t.Fatal(fmt.Errorf("statefulset %s was never ready with %v", name, err))
		}
	}
}

// AssertDeploymentReady asserts that a deployment has the desired number of pods running
func (f *Framework) AssertDeploymentReady(name, namespace string, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}
	return func(t *testing.T) {
		t.Helper()
		key := types.NamespacedName{Name: name, Namespace: namespace}
		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {
			deployment := &appsv1.Deployment{}
			err := f.K8sClient.Get(context.Background(), key, deployment)
			return err == nil && deployment.Status.ReadyReplicas == *deployment.Spec.Replicas, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *Framework) GetResourceWithRetry(t *testing.T, name, namespace string, obj client.Object) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {
		key := types.NamespacedName{Name: name, Namespace: namespace}

		if err := f.K8sClient.Get(context.Background(), key, obj); apierrors.IsNotFound(err) {
			// retry
			return false, nil
		}

		return true, nil
	})

	if wait.Interrupted(err) {
		t.Fatal(fmt.Errorf("resource %s/%s was never created", namespace, name))
	}
}

func ParseMetrics(metrics []byte) (model.Vector, error) {
	sDecoder := expfmt.SampleDecoder{
		Dec: expfmt.NewDecoder(
			bytes.NewReader(metrics),
			expfmt.NewFormat(expfmt.FormatType(expfmt.TypeTextPlain)),
		),
		Opts: &expfmt.DecodeOptions{
			Timestamp: model.TimeFromUnixNano(0),
		},
	}

	var (
		samples    model.Vector
		decSamples = make(model.Vector, 0, 50)
	)
	for {
		err := sDecoder.Decode(&decSamples)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		samples = append(samples, decSamples...)
		decSamples = decSamples[:0]
	}

	return samples, nil
}

func assertSamples(t *testing.T, metrics []byte, expected map[string]float64) {
	t.Helper()

	samples, err := ParseMetrics(metrics)
	if err != nil {
		t.Errorf("error decoding samples: %s", err)
	}

	for _, s := range samples {
		expectedVal, ok := expected[s.Metric.String()]
		if !ok {
			t.Errorf("Unexpected labelset: %v\n", s.Metric.String())
		}
		if float64(s.Value) != expectedVal {
			t.Errorf("%s: got %v, want: %v", s.Metric.String(), s.Value, expectedVal)
		}
	}
}

// GetOperatorPod gets the operator's pod.
func (f *Framework) GetOperatorPod(t *testing.T) *v1.Pod {
	// get the operator deployment
	operator := appsv1.Deployment{}
	f.AssertResourceEventuallyExists("observability-operator", f.OperatorNamespace, &operator)(t)

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

type HTTPOptions struct {
	scheme string
}

func WithHTTPS() func(*HTTPOptions) {
	return func(o *HTTPOptions) {
		o.scheme = "https"
	}
}

func (f *Framework) GetPodMetrics(pod *v1.Pod, opts ...func(*HTTPOptions)) ([]byte, error) {
	var (
		pollErr error
		b       []byte
	)
	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
		b, pollErr = f.getPodMetrics(ctx, pod, opts...)
		if pollErr != nil {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w: %w", err, pollErr)
	}

	return b, nil
}

func (f *Framework) getPodMetrics(ctx context.Context, pod *v1.Pod, opts ...func(*HTTPOptions)) ([]byte, error) {
	var (
		stopChan = make(chan struct{})
		errChan  = make(chan error, 1)
	)
	defer func() {
		select {
		case err := <-errChan:
			fmt.Println("port-forward error:", err.Error())
		default:
		}

		close(stopChan)
	}()

	err := f.StartPortForward(pod.Name, pod.Namespace, "8080", stopChan, errChan)
	if err != nil {
		return nil, fmt.Errorf("failed to start port-forwarding: %w", err)
	}

	httpOptions := HTTPOptions{
		scheme: "http",
	}
	for _, o := range opts {
		o(&httpOptions)
	}

	// The /metrics endpoint shouldn't need more than 5 seconds to send a response.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s://localhost:8080/metrics", httpOptions.scheme), nil)
	if err != nil {
		return nil, err
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		ServerName: fmt.Sprintf("observability-operator.%s.svc", pod.Namespace),
		RootCAs:    f.RootCA,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			fmt.Printf("client cert: %#v\n", f.MetricsClientCert)
			return f.MetricsClientCert, nil
		},
	}

	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get a response from %q: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code from %q: got %d", req.URL.String(), resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// AssertNoReconcileErrors asserts that there are no reconcilation errors
func (f *Framework) AssertNoReconcileErrors(t *testing.T) {
	t.Helper()

	pod := f.GetOperatorPod(t)

	metrics, err := f.GetPodMetrics(pod)
	if err != nil {
		t.Fatalf("pod %s/%s: %s", pod.Namespace, pod.Name, err)
	}

	assertSamples(t, metrics,
		map[string]float64{
			`{__name__="controller_runtime_reconcile_errors_total", controller="monitoringstack"}`: 0,
			`{__name__="controller_runtime_reconcile_errors_total", controller="thanosquerier"}`:   0,
		})
}

func (f *Framework) AssertNoEventWithReason(t *testing.T, reason string) {
	t.Helper()

	c, err := f.getKubernetesClient()
	if err != nil {
		t.Fatalf("failed to get kubenetes client with error: %s", err)
	}

	evts, err := c.EventsV1().Events("").List(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("reason=%s", reason),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(evts.Items) > 0 {
		t.Logf("expected 0 event with reason=%q, got %d", reason, len(evts.Items))
		for i, e := range evts.Items {
			t.Logf("event[%d]: %s", i, e.String())
		}
		t.FailNow()
	}
}

func (f *Framework) GetStackWhenAvailable(t *testing.T, name, namespace string) v1alpha1.MonitoringStack {
	var ms v1alpha1.MonitoringStack
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	var lastErr error

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, DefaultTestTimeout*2, true, func(ctx context.Context) (bool, error) {
		lastErr = nil
		err := f.K8sClient.Get(context.Background(), key, &ms)
		if err != nil {
			lastErr = err
			return false, nil
		}
		availableC := getConditionByType(ms.Status.Conditions, v1alpha1.AvailableCondition)
		if availableC != nil && availableC.Status == v1alpha1.ConditionTrue {
			return true, nil
		}
		return false, nil
	})

	if wait.Interrupted(err) {
		t.Fatal(fmt.Errorf("MonitoringStack %s/%s was not available - err: %w |  %v", namespace, name, lastErr, ms.Status.Conditions))
	}
	return ms
}

func (f *Framework) AssertAlertmanagerAbsent(t *testing.T, name, namespace string) {
	var am monv1.Alertmanager
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
		err := f.K8sClient.Get(context.Background(), key, &am)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	if wait.Interrupted(err) {
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

// AssertPrometheusReplicaStatus asserts that prometheus is scaled correctly duration a time period of customForeverTestTimeout
func (f *Framework) AssertPrometheusReplicaStatus(name, namespace string, expectedReplicas int32, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}
	prom := monv1.Prometheus{}
	return func(t *testing.T) {
		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, false, func(ctx context.Context) (bool, error) {
			key := types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			}
			if err := f.K8sClient.Get(context.Background(), key, &prom); apierrors.IsNotFound(err) {
				return false, nil
			}
			if prom.Status.Replicas != expectedReplicas {
				return false, nil
			}
			return true, nil

		}); wait.Interrupted(err) {
			t.Fatal(fmt.Errorf("Prometheus %s/%s has unexpected number of replicas, got %d, expected %d", namespace, name, prom.Status.Replicas, expectedReplicas))
		}
	}
}
