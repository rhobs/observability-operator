package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	clfv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func loggingLokiStackName(instanceName string) string {
	return instanceName + "-loki"
}

func TestUpdateStatus(t *testing.T) {
	const (
		name      = "example"
		namespace = "observability"
	)
	key := types.NamespacedName{Name: name, Namespace: namespace}

	tests := []struct {
		name         string
		capabilities *obsv1alpha1.CapabilitiesSpec
		initial      obsv1alpha1.ObservabilityInstallerStatus
		objects      []runtime.Object
		reconcileErr error
		want         obsv1alpha1.ObservabilityInstallerStatus
		wantRequeue  bool
	}{
		{
			name: "empty capabilities clear stale status",
			initial: obsv1alpha1.ObservabilityInstallerStatus{
				Tempo: "old-tempo", OpenTelemetry: "old-otel", LokiStack: "old-loki", Logging: "old-logging",
			},
			want: obsv1alpha1.ObservabilityInstallerStatus{},
		},
		{
			name: "disabled capabilities clear stale status",
			capabilities: &obsv1alpha1.CapabilitiesSpec{
				Tracing: &obsv1alpha1.TracingSpec{},
				Logging: &obsv1alpha1.LoggingSpec{},
			},
			initial: obsv1alpha1.ObservabilityInstallerStatus{
				Tempo: "old-tempo", OpenTelemetry: "old-otel", LokiStack: "old-loki", Logging: "old-logging",
			},
			want: obsv1alpha1.ObservabilityInstallerStatus{},
		},
		{
			name: "enabled capabilities report component status",
			capabilities: &obsv1alpha1.CapabilitiesSpec{
				Tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
				Logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
			},
			objects: []runtime.Object{
				&otelv1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
					Status:     otelv1beta1.OpenTelemetryCollectorStatus{Version: "1.2.3"},
				},
				&tempov1alpha1.TempoStack{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
					Status:     tempov1alpha1.TempoStackStatus{TempoVersion: "2.3.4"},
				},
				&lokiv1.LokiStack{ObjectMeta: metav1.ObjectMeta{Name: loggingLokiStackName(name), Namespace: namespace}},
				&clfv1.ClusterLogForwarder{
					ObjectMeta: metav1.ObjectMeta{Name: name + "-logging", Namespace: namespace},
					Status:     clfv1.ClusterLogForwarderStatus{Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}},
				},
			},
			want: obsv1alpha1.ObservabilityInstallerStatus{
				Tempo:         namespace + "/" + name + " (2.3.4)",
				OpenTelemetry: namespace + "/" + name + " (1.2.3)",
				LokiStack:     namespace + "/" + loggingLokiStackName(name),
				Logging:       namespace + "/" + name + "-logging",
			},
		},
		{
			name: "logging is not ready until forwarder is ready",
			capabilities: &obsv1alpha1.CapabilitiesSpec{
				Logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
			},
			initial: obsv1alpha1.ObservabilityInstallerStatus{Logging: "stale"},
			objects: []runtime.Object{
				&lokiv1.LokiStack{ObjectMeta: metav1.ObjectMeta{Name: loggingLokiStackName(name), Namespace: namespace}},
				&clfv1.ClusterLogForwarder{
					ObjectMeta: metav1.ObjectMeta{Name: name + "-logging", Namespace: namespace},
					Status:     clfv1.ClusterLogForwarderStatus{Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}},
				},
			},
			want: obsv1alpha1.ObservabilityInstallerStatus{LokiStack: namespace + "/" + loggingLokiStackName(name)},
		},
		{
			name:         "reconcile error sets condition",
			reconcileErr: errors.New("reconcile failed"),
			want: obsv1alpha1.ObservabilityInstallerStatus{Conditions: []metav1.Condition{{
				Type: conditionTypeReconciled, Status: metav1.ConditionFalse, Reason: conditionReasonError,
				Message: "reconcile failed",
			}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, obsv1alpha1.AddToScheme(scheme))
			require.NoError(t, otelv1beta1.AddToScheme(scheme))
			require.NoError(t, tempov1alpha1.AddToScheme(scheme))
			require.NoError(t, lokiv1.AddToScheme(scheme))
			require.NoError(t, clfv1.AddToScheme(scheme))

			instance := &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec:       obsv1alpha1.ObservabilityInstallerSpec{Capabilities: tt.capabilities},
				Status:     tt.initial,
			}
			objects := append([]runtime.Object{instance}, tt.objects...)
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&obsv1alpha1.ObservabilityInstaller{}).
				WithRuntimeObjects(objects...).Build()
			controller := observabilityInstallerController{client: k8sClient, logger: logr.Discard(), capabilities: capabilities(Options{})}

			result := controller.updateStatus(context.Background(), instance, tt.reconcileErr)
			require.Equal(t, tt.wantRequeue, result.RequeueAfter > 0)

			var got obsv1alpha1.ObservabilityInstaller
			require.NoError(t, k8sClient.Get(context.Background(), key, &got))
			if len(tt.want.Conditions) > 0 {
				require.Len(t, got.Status.Conditions, 1)
				tt.want.Conditions[0].LastTransitionTime = got.Status.Conditions[0].LastTransitionTime
				tt.want.Conditions[0].ObservedGeneration = got.Status.Conditions[0].ObservedGeneration
			}
			require.Equal(t, tt.want, got.Status)
		})
	}
}

func TestUpdateStatusRequeuesForMissingEnabledOperand(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, obsv1alpha1.AddToScheme(scheme))
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "observability"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
			Tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
		}},
		Status: obsv1alpha1.ObservabilityInstallerStatus{Tempo: "stale", OpenTelemetry: "stale"},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&obsv1alpha1.ObservabilityInstaller{}).WithRuntimeObjects(instance).Build()
	controller := observabilityInstallerController{client: k8sClient, logger: logr.Discard(), capabilities: capabilities(Options{})}

	result := controller.updateStatus(context.Background(), instance, nil)
	require.Equal(t, 2*time.Second, result.RequeueAfter)
	var got obsv1alpha1.ObservabilityInstaller
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(instance), &got))
	require.Empty(t, got.Status.Tempo)
	require.Empty(t, got.Status.OpenTelemetry)
}
