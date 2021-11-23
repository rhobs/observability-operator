package framework

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	Config    *rest.Config
	K8sClient client.Client
	Retain    bool
}

// StartPortForward initiates a port forwarding connection to a pod on the localhost interface.
//
// The function call blocks until the port forwarding proxy server is ready to receive connections.
func (f *Framework) StartPortForward(podName string, ns string, port string, stopChan chan struct{}) error {
	roundTripper, upgrader, err := spdy.RoundTripperFor(f.Config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", ns, podName)
	hostIP := strings.TrimLeft(f.Config.Host, "htps:/")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	readyChan := make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)
	forwarder, err := portforward.New(dialer, []string{port}, stopChan, readyChan, out, errOut)
	if err != nil {
		return err
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

func (f *Framework) CleanUp(t *testing.T, cleanupFunc func()) {
	t.Cleanup(func() {
		testSucceeded := !t.Failed()
		if testSucceeded || !f.Retain {
			cleanupFunc()
		}
	})
}
