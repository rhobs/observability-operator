module rhobs/monitoring-stack-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/grafana-operator/grafana-operator/v4 v4.0.1 // indirect
	github.com/prometheus-operator/prometheus-operator v0.49.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.49.0
	k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/kustomize/kustomize/v3 v3.10.0 // indirect
)

// A replace directive is needed for k8s.io/client-go because Cortex (which
// is an indirect dependency through Thanos) has a requirement on v12.0.0.
replace k8s.io/client-go => k8s.io/client-go v0.21.2
