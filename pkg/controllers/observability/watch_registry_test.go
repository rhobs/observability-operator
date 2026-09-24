package observability

import (
	"errors"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/logging"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/tracing"
	"testing"

	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	clfv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	lokiStackGVK  = lokiv1.GroupVersion.WithKind("LokiStack")
	tempoStackGVK = tempov1alpha1.GroupVersion.WithKind("TempoStack")
)

func TestWatchRegistry(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, lokiv1.AddToScheme(scheme))
	require.NoError(t, tempov1alpha1.AddToScheme(scheme))
	registry := newWatchRegistry()
	objects := []client.Object{&lokiv1.LokiStack{}, &tempov1alpha1.TempoStack{}}

	var registered []client.Object
	register := func(object client.Object) error {
		registered = append(registered, object)
		return nil
	}

	require.NoError(t, registry.RegisterAvailable(objects, map[schema.GroupVersionKind]bool{lokiStackGVK: true}, scheme, register))
	require.Len(t, registered, 1)
	require.IsType(t, &lokiv1.LokiStack{}, registered[0])

	// A successful watch is not registered twice; a newly available API is.
	require.NoError(t, registry.RegisterAvailable(objects, map[schema.GroupVersionKind]bool{lokiStackGVK: true, tempoStackGVK: true}, scheme, register))
	require.Len(t, registered, 2)
	require.IsType(t, &tempov1alpha1.TempoStack{}, registered[1])
}

func TestWatchRegistryRetriesFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, lokiv1.AddToScheme(scheme))
	registry := newWatchRegistry()
	attempts := 0
	register := func(client.Object) error {
		attempts++
		if attempts == 1 {
			return errors.New("temporary failure")
		}
		return nil
	}

	groups := map[schema.GroupVersionKind]bool{lokiStackGVK: true}
	require.Error(t, registry.RegisterAvailable([]client.Object{&lokiv1.LokiStack{}}, groups, scheme, register))
	require.NoError(t, registry.RegisterAvailable([]client.Object{&lokiv1.LokiStack{}}, groups, scheme, register))
	require.Equal(t, 2, attempts)
}

func TestCapabilitiesDeclareWatchTypes(t *testing.T) {
	tracing := tracing.New(tracing.Options{}).WatchTypes
	require.Len(t, tracing, 2)
	require.IsType(t, &otelv1beta1.OpenTelemetryCollector{}, tracing[0])
	require.IsType(t, &tempov1alpha1.TempoStack{}, tracing[1])

	logging := logging.New(logging.Options{}).WatchTypes
	require.Len(t, logging, 2)
	require.IsType(t, &lokiv1.LokiStack{}, logging[0])
	require.IsType(t, &clfv1.ClusterLogForwarder{}, logging[1])
}
