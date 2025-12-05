package framework

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
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

// AssertStatefulSetContainerHasArg asserts that a specific container in a StatefulSet's
// Pod template contains the expected command-line argument.
func (f *Framework) AssertStatefulSetContainerHasArg(t *testing.T, name, namespace, containerName, expectedArg string, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}

	return func(t *testing.T) {
		t.Helper()
		statefulSet := &appsv1.StatefulSet{}
		key := types.NamespacedName{Name: name, Namespace: namespace}

		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {

			if err := f.K8sClient.Get(ctx, key, statefulSet); apierrors.IsNotFound(err) {
				return false, nil
			}

			var container *v1.Container
			for i, c := range statefulSet.Spec.Template.Spec.Containers {
				if c.Name == containerName {
					container = &statefulSet.Spec.Template.Spec.Containers[i]
					break
				}
			}

			if container == nil {
				return false, fmt.Errorf("container %q not found in StatefulSet template", containerName)
			}

			for _, arg := range container.Args {
				if arg == expectedArg {
					return true, nil
				}
			}

			t.Logf("StatefulSet %s container %q args are missing %q. Retrying...", name, containerName, expectedArg)
			return false, nil
		}); wait.Interrupted(err) {
			t.Fatalf("StatefulSet %s failed to contain argument %q in container %q within timeout. Final args: %v",
				name, expectedArg, containerName, statefulSet.Spec.Template.Spec.Containers[0].Args)
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

// AssertDeploymentReadyAndStable asserts that a deployment has the desired number of pods running for 2 consecutive polls 5 seconds appart
func (f *Framework) AssertDeploymentReadyAndStable(name, namespace string, fns ...OptionFn) func(t *testing.T) {
	option := AssertOption{
		PollInterval: 5 * time.Second,
		WaitTimeout:  DefaultTestTimeout,
	}
	for _, fn := range fns {
		fn(&option)
	}
	return func(t *testing.T) {
		key := types.NamespacedName{Name: name, Namespace: namespace}
		if err := wait.PollUntilContextTimeout(context.Background(), option.PollInterval, option.WaitTimeout, true, func(ctx context.Context) (bool, error) {
			deployment := &appsv1.Deployment{}
			err := f.K8sClient.Get(context.Background(), key, deployment)
			if err == nil && deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
				time.Sleep(5 * time.Second)
				err := f.K8sClient.Get(context.Background(), key, deployment)
				return err == nil && deployment.Status.ReadyReplicas == *deployment.Spec.Replicas, nil
			} else {
				return false, nil
			}
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
		t.Errorf("failed to get operator pods: %s", err)
	}

	if len(pods.Items) != 1 {
		t.Errorf("Expected 1 operator pod but got: %d", len(pods.Items))
	}

	return &pods.Items[0]
}

type HTTPOptions struct {
	scheme  string
	port    string
	method  string
	path    string
	body    string
	timeout time.Duration
}

func WithHTTPS() func(*HTTPOptions) {
	return func(o *HTTPOptions) {
		o.scheme = "https"
	}
}

func WithPort(p string) func(*HTTPOptions) {
	return func(o *HTTPOptions) {
		o.port = p
	}
}

func WithMethod(m string) func(*HTTPOptions) {
	return func(o *HTTPOptions) {
		o.method = m
	}
}

func WithPath(p string) func(*HTTPOptions) {
	return func(o *HTTPOptions) {
		o.path = p
	}
}

func WithBody(b string) func(*HTTPOptions) {
	return func(o *HTTPOptions) {
		o.body = b
	}
}

// GetPodMetrics requests the /metrics endpoint from the pod.
func (f *Framework) GetPodMetrics(pod *v1.Pod, opts ...func(*HTTPOptions)) ([]byte, error) {
	var (
		pollErr error
		b       []byte
	)
	opts = append(opts, WithPath("/metrics"), WithPort("8080"))
	if err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, DefaultTestTimeout, true, func(ctx context.Context) (bool, error) {
		b, pollErr = f.getRequest(ctx, pod, opts...)
		if pollErr != nil {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w: %w", err, pollErr)
	}

	return b, nil
}

// AssertPromQLResult evaluates the PromQL expression against the in-cluster
// Prometheus stack.
// It returns an error if the request fails. Otherwise the result is passed to
// the callback function for additional checks.
func (f *Framework) AssertPromQLResult(t *testing.T, expr string, callback func(model.Value) error) error {
	t.Helper()
	var (
		pollErr error
		v       model.Value
	)
	if err := wait.PollUntilContextTimeout(context.Background(), 20*time.Second, 3*DefaultTestTimeout, true, func(context.Context) (bool, error) {
		v, pollErr = f.getPromQLResult(context.Background(), expr)
		if pollErr != nil {
			t.Logf("error from getPromQLResult(): %s", pollErr)
			return false, nil
		}

		pollErr = callback(v)
		if pollErr != nil {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return fmt.Errorf("failed to assert query %q: %w: %w", expr, err, pollErr)
	}

	return nil
}

// Copied from github.com/prometheus/client_golang/blob/api/prometheus/v1/api.go
type apiResponse struct {
	Status    string      `json:"status"`
	Result    queryResult `json:"data"`
	ErrorType string      `json:"errorType"`
	Error     string      `json:"error"`
	Warnings  []string    `json:"warnings,omitempty"`
}

type queryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result interface{}     `json:"result"`

	// The decoded value.
	v model.Value
}

func (qr *queryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		err = json.Unmarshal(v.Result, &sv)
		qr.v = &sv

	case model.ValVector:
		var vv model.Vector
		err = json.Unmarshal(v.Result, &vv)
		qr.v = vv

	case model.ValMatrix:
		var mv model.Matrix
		err = json.Unmarshal(v.Result, &mv)
		qr.v = mv

	default:
		err = fmt.Errorf("unexpected value type %q", v.Type)
	}
	return err
}

func (f *Framework) getPromQLResult(ctx context.Context, expr string) (model.Value, error) {
	pods, err := f.getPodsForService("prometheus-k8s", "openshift-monitoring")
	if err != nil {
		return nil, fmt.Errorf("failed to get prometheus pod: %w", err)
	}

	if len(pods) == 0 {
		return nil, fmt.Errorf("no Prometheus pods found")
	}

	data := url.Values{}
	data.Set("query", expr)
	b, err := f.getRequest(
		ctx,
		&pods[0],
		WithPort("9090"),
		WithMethod("POST"),
		WithPath("/api/v1/query"),
		WithBody(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query prometheus: %w", err)
	}

	var r apiResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("failed to parse prometheus response: %w", err)
	}

	if r.Status != "success" {
		return nil, fmt.Errorf("%q: %s (%s)", expr, r.ErrorType, r.Error)
	}

	return r.Result.v, nil
}

// getRequest makes an HTTP request to the pod via port-forward.
func (f *Framework) getRequest(ctx context.Context, pod *v1.Pod, opts ...func(*HTTPOptions)) ([]byte, error) {
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

	httpOptions := HTTPOptions{
		scheme:  "http",
		method:  "GET",
		timeout: 4 * time.Second,
	}
	for _, o := range opts {
		o(&httpOptions)
	}

	err := f.StartPortForward(pod.Name, pod.Namespace, httpOptions.port, stopChan, errChan)
	if err != nil {
		return nil, fmt.Errorf("failed to start port-forwarding: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, httpOptions.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx,
		httpOptions.method,
		httpOptions.scheme+"://"+path.Join(fmt.Sprintf("localhost:%s", httpOptions.port), httpOptions.path),
		strings.NewReader(httpOptions.body),
	)
	if err != nil {
		return nil, err
	}
	if req.Method == "POST" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{
		ServerName: fmt.Sprintf("observability-operator.%s.svc", pod.Namespace),
		RootCAs:    f.RootCA,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return f.MetricsClientCert, nil
		},
	}

	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get a response from %q: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid status code from %q: got %d (%q)", req.URL.String(), resp.StatusCode, string(b))
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
