package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/logging"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/tracing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCapabilityStatusReportsReadErrors(t *testing.T) {
	tests := []struct {
		name       string
		capability capability.Definition
		instance   *obsv1alpha1.ObservabilityInstaller
		objectType interface{}
	}{
		{
			name:       "tracing",
			capability: tracing.New(tracing.Options{}),
			instance: enabledStatusInstance(&obsv1alpha1.CapabilitiesSpec{
				Tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
			}),
			objectType: &otelv1beta1.OpenTelemetryCollector{},
		},
		{
			name:       "logging",
			capability: logging.New(logging.Options{}),
			instance: enabledStatusInstance(&obsv1alpha1.CapabilitiesSpec{
				Logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
			}),
			objectType: &lokiv1.LokiStack{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &MockClient{}
			reader.On("Get", context.Background(), mock.Anything, mock.IsType(tt.objectType), mock.Anything).Return(errors.New("API unavailable"))

			result := tt.capability.UpdateStatus(context.Background(), tt.instance, reader)
			require.ErrorContains(t, result.Err, "API unavailable")
			require.Equal(t, 2*time.Second, result.RequeueAfter)
		})
	}
}

func enabledStatusInstance(capabilities *obsv1alpha1.CapabilitiesSpec) *obsv1alpha1.ObservabilityInstaller {
	return &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "observability"},
		Spec:       obsv1alpha1.ObservabilityInstallerSpec{Capabilities: capabilities},
	}
}
