module rhobs/monitoring-stack-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/grafana-operator/grafana-operator/v4 v4.0.1
	github.com/operator-framework/api v0.10.3
	github.com/operator-framework/operator-lifecycle-manager v0.19.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.49.0
	github.com/prometheus/common v0.29.0 // indirect
	go.uber.org/atomic v1.8.0 // indirect
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b // indirect
	gotest.tools/v3 v3.0.3
	k8s.io/api v0.22.0
	k8s.io/apiextensions-apiserver v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.9.2
)

replace (
	// A replace directive is needed for k8s.io/client-go because Cortex (which
	// is an indirect dependency through Thanos) has a requirement on v12.0.0.
	k8s.io/api => k8s.io/api v0.21.2
	// There is no OLM release that uses 0.21.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.2
	k8s.io/client-go => k8s.io/client-go v0.21.2

)
