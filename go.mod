module rhobs/monitoring-stack-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.49.0
	github.com/prometheus/common v0.29.0 // indirect
	go.uber.org/atomic v1.8.0 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	golang.org/x/tools v0.1.3 // indirect
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog/v2 v2.9.0 // indirect
	k8s.io/utils v0.0.0-20210629042839-4a2b36d8d73f // indirect
	sigs.k8s.io/controller-runtime v0.9.2
)

// A replace directive is needed for k8s.io/client-go because Cortex (which
// is an indirect dependency through Thanos) has a requirement on v12.0.0.
replace k8s.io/client-go => k8s.io/client-go v0.21.2
