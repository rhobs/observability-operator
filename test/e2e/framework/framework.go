package framework

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	kubernetes         kubernetes.Interface
	Config             *rest.Config
	K8sClient          client.Client
	Retain             bool
	IsOpenshiftCluster bool
	RootCA             *x509.CertPool
	MetricsClientCert  *tls.Certificate
	OperatorNamespace  string
}

// Setup finalizes the initilization of the Framework object by setting
// parameters which are specific to OpenShift.
func (f *Framework) Setup() error {
	clusterVersion := &configv1.ClusterVersion{}
	if err := f.K8sClient.Get(context.Background(), client.ObjectKey{Name: "version"}, clusterVersion); err != nil {
		if meta.IsNoMatchError(err) {
			return nil
		}

		return fmt.Errorf("failed to get clusterversion %w", err)
	}

	f.IsOpenshiftCluster = true

	// Load the service CA operator's certificate authority.
	var (
		cm  v1.ConfigMap
		key = client.ObjectKey{
			Namespace: "openshift-config",
			Name:      "openshift-service-ca.crt",
		}
	)
	if err := f.K8sClient.Get(context.Background(), key, &cm); err != nil {
		return err
	}

	b, found := cm.Data["service-ca.crt"]
	if !found {
		return errors.New("failed to find 'service-ca.crt'")
	}

	rootCA := x509.NewCertPool()
	if !rootCA.AppendCertsFromPEM([]byte(b)) {
		return errors.New("invalid service CA")
	}
	f.RootCA = rootCA

	// Load the prometheus-k8s TLS client certificate.
	var s v1.Secret
	key = client.ObjectKey{
		Namespace: "openshift-monitoring",
		Name:      "metrics-client-certs",
	}
	if err := f.K8sClient.Get(context.Background(), key, &s); err != nil {
		return err
	}

	cert, found := s.Data["tls.crt"]
	if !found {
		return errors.New("failed to find TLS client certificate")
	}

	k, found := s.Data["tls.key"]
	if !found {
		return errors.New("failed to find TLS client key")
	}

	c, err := tls.X509KeyPair(cert, k)
	if err != nil {
		return err
	}
	f.MetricsClientCert = &c

	return nil
}

// StartPortForward initiates a port forwarding connection to a pod on the localhost interface.
//
// The function call blocks until the port forwarding proxy server is ready to
// receive connections. The errChan parameter can be used to retrieve errors
// happening after the port-fowarding connection is in place.
func (f *Framework) StartPortForward(podName string, ns string, port string, stopChan chan struct{}, errChan chan error) error {
	roundTripper, upgrader, err := spdy.RoundTripperFor(f.Config)
	if err != nil {
		return fmt.Errorf("error creating RoundTripper: %w", err)
	}

	u := fmt.Sprintf("https://%s", strings.TrimPrefix(strings.TrimPrefix(f.Config.Host, "http://"), "https://"))
	serverURL, err := url.Parse(u)
	if err != nil {
		return err
	}
	serverURL.Path = path.Join(
		serverURL.Path,
		fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ns, podName),
	)

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)

	var (
		readyChan = make(chan struct{}, 1)
		out       = &bytes.Buffer{}
	)
	forwarder, err := portforward.New(dialer, []string{port}, stopChan, readyChan, out, out)
	if err != nil {
		return fmt.Errorf("failed to create portforward: %w", err)
	}
	defer func() {
		if out.Len() > 0 {
			fmt.Println(out.String())
		}
	}()

	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			if errChan == nil {
				return
			}

			select {
			case errChan <- err:
			default:
			}
		}
	}()

	<-readyChan
	return nil
}

// StartServicePortForward initiates a port forwarding connection to a service on the localhost interface.
//
// The function call blocks until the port forwarding proxy server is ready to receive connections.
func (f *Framework) StartServicePortForward(serviceName string, ns string, port string, stopChan chan struct{}) error {
	pods, err := f.getPodsForService(serviceName, ns)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("no pods found for service %s/%s", serviceName, ns)
	}

	return f.StartPortForward(pods[0].Name, ns, port, stopChan, nil)
}

func (f *Framework) GetStatefulSetPods(name string, namespace string) ([]corev1.Pod, error) {
	var svc appsv1.StatefulSet
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := f.K8sClient.Get(context.Background(), key, &svc); err != nil {
		return nil, err
	}

	selector := svc.Spec.Template.ObjectMeta.Labels
	var pods corev1.PodList
	if err := f.K8sClient.List(context.Background(), &pods, client.MatchingLabels(selector)); err != nil {
		return nil, err
	}

	return pods.Items, nil
}

func (f *Framework) getPodsForService(name string, namespace string) ([]corev1.Pod, error) {
	var svc corev1.Service
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := f.K8sClient.Get(context.Background(), key, &svc); err != nil {
		return nil, err
	}

	selector := svc.Spec.Selector
	var pods corev1.PodList
	if err := f.K8sClient.List(context.Background(), &pods, client.MatchingLabels(selector)); err != nil {
		return nil, err
	}

	return pods.Items, nil
}

func (f *Framework) getKubernetesClient() (kubernetes.Interface, error) {
	if f.kubernetes == nil {
		c, err := kubernetes.NewForConfig(f.Config)
		if err != nil {
			return nil, err
		}
		f.kubernetes = c
	}

	return f.kubernetes, nil
}

func (f *Framework) Evict(pod *corev1.Pod, gracePeriodSeconds int64) error {
	delOpts := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	eviction := &policyv1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyv1.SchemeGroupVersion.String(),
			Kind:       "Eviction",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: &delOpts,
	}

	c, err := f.getKubernetesClient()
	if err != nil {
		return err
	}
	return c.PolicyV1().Evictions(pod.Namespace).Evict(context.Background(), eviction)
}

func (f *Framework) CleanUp(t *testing.T, cleanupFunc func()) {
	t.Cleanup(func() {
		testSucceeded := !t.Failed()
		if testSucceeded || !f.Retain {
			cleanupFunc()
		}
	})
}
