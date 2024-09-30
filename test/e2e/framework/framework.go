package framework

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
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
}

// StartPortForward initiates a port forwarding connection to a pod on the localhost interface.
//
// The function call blocks until the port forwarding proxy server is ready to receive connections.
func (f *Framework) StartPortForward(podName string, ns string, port string, stopChan chan struct{}) error {
	roundTripper, upgrader, err := spdy.RoundTripperFor(f.Config)
	if err != nil {
		return errors.Wrap(err, "error creating RoundTripper")
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ns, podName)
	hostIP := strings.TrimLeft(f.Config.Host, "htps:/")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	readyChan := make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err := portforward.New(dialer, []string{port}, stopChan, readyChan, out, errOut)
	if err != nil {
		return errors.Wrap(err, "failed to create portforward")
	}

	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			panic(err)
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
	return f.StartPortForward(pods[0].Name, ns, port, stopChan)
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
