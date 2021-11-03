package framework

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Framework struct {
	Config    *rest.Config
	K8sClient client.Client
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
