package framework

import "sigs.k8s.io/controller-runtime/pkg/client"

type Framework struct {
	K8sClient client.Client
}
