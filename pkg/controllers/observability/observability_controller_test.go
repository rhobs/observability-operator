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
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

func TestReconcileKeepsFinalizerUntilOperandsGone(t *testing.T) {
	ctx := context.Background()
	scheme := getScheme()
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{
		Name: "installer", Namespace: "test", Finalizers: []string{finalizerName},
	}}
	operand := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "operand", Namespace: "test", Finalizers: []string{"operand-operator/finalizer"},
		Labels: map[string]string{util.ResourceLabel: util.OpName},
	}}
	sub := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{Name: "operator", Namespace: "operators", Labels: map[string]string{util.ResourceLabel: util.OpName}},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "operator"},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(instance).WithObjects(instance, operand, sub).Build()
	require.NoError(t, k8sClient.Delete(ctx, instance))
	controller := observabilityInstallerController{
		client: k8sClient, apiReader: k8sClient, scheme: scheme, logger: logr.Discard(),
		watches: newWatchRegistry(),
		capabilities: []capability.Definition{{
			Plan: func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) (capability.Plan, error) {
				return capability.Plan{Owned: []client.Object{operand}}, nil
			},
			Shared: func(*obsv1alpha1.ObservabilityInstaller) capability.Plan {
				return capability.Plan{Operators: capability.DesiredOperators(false, capability.NewOperatorRequirement("operator", capability.OperatorConfig{PackageName: "operator"}))}
			},
			UpdateStatus: func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) capability.StatusResult {
				return capability.StatusResult{}
			},
		}},
	}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(instance)}
	result, err := controller.Reconcile(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, result.RequeueAfter)
	var live obsv1alpha1.ObservabilityInstaller
	require.NoError(t, k8sClient.Get(ctx, request.NamespacedName, &live))
	require.Contains(t, live.Finalizers, finalizerName)
	condition := apimeta.FindStatusCondition(live.Status.Conditions, conditionTypeReconciled)
	require.NotNil(t, condition)
	require.Equal(t, metav1.ConditionFalse, condition.Status)
	require.Equal(t, "OperandDeletionPending", condition.Reason)
	require.Contains(t, condition.Message, "test/operand")
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{}))

	var pending corev1.ConfigMap
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(operand), &pending))
	pending.Finalizers = nil
	require.NoError(t, k8sClient.Update(ctx, &pending))
	_, err = controller.Reconcile(ctx, request)
	require.NoError(t, err)
	require.True(t, apierrors.IsNotFound(k8sClient.Get(ctx, request.NamespacedName, &live)))
	require.True(t, apierrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{})))
}

func TestReconcileDeletionWaitsUntilOperandGone(t *testing.T) {
	ctx := context.Background()
	scheme := getScheme()
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{
		Name: "installer", Namespace: "test", Finalizers: []string{finalizerName},
		DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-time.Hour)},
	}}
	operand := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "operand", Namespace: "test", Finalizers: []string{"stuck"},
		Labels: map[string]string{util.ResourceLabel: util.OpName},
	}}
	sub := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{Name: "operator", Namespace: "operators", Labels: map[string]string{util.ResourceLabel: util.OpName}},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "operator"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(instance).WithObjects(instance, operand, sub).Build()
	controller := observabilityInstallerController{
		client: c, apiReader: c, scheme: scheme, logger: logr.Discard(), watches: newWatchRegistry(),
		capabilities: []capability.Definition{{
			Plan: func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) (capability.Plan, error) {
				return capability.Plan{Owned: []client.Object{operand}}, nil
			},
			Shared: func(*obsv1alpha1.ObservabilityInstaller) capability.Plan {
				return capability.Plan{Operators: capability.DesiredOperators(false, capability.NewOperatorRequirement("operator", capability.OperatorConfig{PackageName: "operator"}))}
			},
			UpdateStatus: func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) capability.StatusResult {
				return capability.StatusResult{}
			},
		}},
	}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(instance)}
	result, err := controller.Reconcile(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, result.RequeueAfter)
	var live obsv1alpha1.ObservabilityInstaller
	require.NoError(t, c.Get(ctx, request.NamespacedName, &live))
	require.Contains(t, live.Finalizers, finalizerName)
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{}))
	var pending corev1.ConfigMap
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(operand), &pending))
	require.NotNil(t, pending.DeletionTimestamp)
	pending.Finalizers = nil
	require.NoError(t, c.Update(ctx, &pending))
	_, err = controller.Reconcile(ctx, request)
	require.NoError(t, err)
	require.True(t, apierrors.IsNotFound(c.Get(ctx, request.NamespacedName, &live)))
	require.True(t, apierrors.IsNotFound(c.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{})))
}

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

			result := controller.updateStatus(context.Background(), instance, tt.reconcileErr, operatorsStatus{})
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

func TestExternalSubscriptionCondition(t *testing.T) {
	opts := testOptions()
	opts.OpenTelemetryOperator.EquivalentPackages = []string{"opentelemetry-community"}
	definitions := capabilities(opts)
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "observability", Generation: 3},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
			Tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
		}},
	}
	operators := operatorsStatus{subs: []olmv1alpha1.Subscription{
		{ObjectMeta: metav1.ObjectMeta{Name: "otel-community", Namespace: "external"}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "opentelemetry-community"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "tempo", Namespace: "managed", Labels: map[string]string{"app.kubernetes.io/managed-by": "observability-operator"}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo-product"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "external"}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "loki-operator"}},
	}}

	updateExternalSubscriptionCondition(instance, definitions, operators)
	condition := apimeta.FindStatusCondition(instance.Status.Conditions, conditionTypeExternalOperator)
	require.NotNil(t, condition)
	require.Equal(t, metav1.ConditionTrue, condition.Status)
	require.Equal(t, "ReusingSubscription", condition.Reason)
	require.Equal(t, int64(3), condition.ObservedGeneration)
	require.Contains(t, condition.Message, "opentelemetry: external/otel-community (package opentelemetry-community)")
	require.NotContains(t, condition.Message, "tempo")
	require.NotContains(t, condition.Message, "unrelated")

	// An install-only request also uses an external subscription.
	instance.Spec.Capabilities.Tracing.Enabled = false
	install := true
	instance.Spec.Capabilities.Tracing.Operators = &obsv1alpha1.OperatorsSpec{Install: &install}
	updateExternalSubscriptionCondition(instance, definitions, operators)
	require.NotNil(t, apimeta.FindStatusCondition(instance.Status.Conditions, conditionTypeExternalOperator))

	// Once no operator is requested, the informational condition is cleared.
	instance.Spec.Capabilities.Tracing.Operators = nil
	updateExternalSubscriptionCondition(instance, definitions, operators)
	require.Nil(t, apimeta.FindStatusCondition(instance.Status.Conditions, conditionTypeExternalOperator))

	// Reusing an operator is not reported for an installer being deleted.
	instance.Spec.Capabilities.Tracing.Enabled = true
	now := metav1.Now()
	instance.DeletionTimestamp = &now
	updateExternalSubscriptionCondition(instance, definitions, operators)
	require.Nil(t, apimeta.FindStatusCondition(instance.Status.Conditions, conditionTypeExternalOperator))
}

func TestUpdateStatusPersistsExternalSubscription(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, obsv1alpha1.AddToScheme(scheme))
	install := true
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "observability"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
			Tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
				Operators: &obsv1alpha1.OperatorsSpec{Install: &install},
			}},
		}},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&obsv1alpha1.ObservabilityInstaller{}).WithObjects(instance).Build()
	controller := observabilityInstallerController{client: k8sClient, logger: logr.Discard(), capabilities: capabilities(testOptions())}
	operators := operatorsStatus{subs: []olmv1alpha1.Subscription{{
		ObjectMeta: metav1.ObjectMeta{Name: "tempo", Namespace: "other"},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "tempo-product"},
	}}}

	controller.updateStatus(context.Background(), instance, nil, operators)
	var got obsv1alpha1.ObservabilityInstaller
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(instance), &got))
	condition := apimeta.FindStatusCondition(got.Status.Conditions, conditionTypeExternalOperator)
	require.NotNil(t, condition)
	require.Contains(t, condition.Message, "tempo: other/tempo")
	require.Nil(t, apimeta.FindStatusCondition(got.Status.Conditions, conditionTypeReconciled))

	controller.updateStatus(context.Background(), &got, nil, operatorsStatus{})
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(instance), &got))
	require.Nil(t, apimeta.FindStatusCondition(got.Status.Conditions, conditionTypeExternalOperator))
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

	result := controller.updateStatus(context.Background(), instance, nil, operatorsStatus{})
	require.Equal(t, 2*time.Second, result.RequeueAfter)
	var got obsv1alpha1.ObservabilityInstaller
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(instance), &got))
	require.Empty(t, got.Status.Tempo)
	require.Empty(t, got.Status.OpenTelemetry)
}

func TestSubscriptionChanged(t *testing.T) {
	sub := func(mutate func(*olmv1alpha1.Subscription)) *olmv1alpha1.Subscription {
		s := &olmv1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tempo-product", Namespace: "operators",
				Labels: map[string]string{util.ResourceLabel: util.OpName},
			},
			Spec:   &olmv1alpha1.SubscriptionSpec{Package: "tempo-product", Channel: "stable"},
			Status: olmv1alpha1.SubscriptionStatus{CurrentCSV: "tempo-operator.v1"},
		}
		if mutate != nil {
			mutate(s)
		}
		return s
	}
	now := metav1.Now()

	tests := []struct {
		name string
		to   *olmv1alpha1.Subscription
		want bool
	}{
		{name: "unchanged", to: sub(nil)},
		{
			name: "status churn only",
			to: sub(func(s *olmv1alpha1.Subscription) {
				s.Status.LastUpdated = now
				s.Status.CatalogHealth = []olmv1alpha1.SubscriptionCatalogHealth{{Healthy: true}}
			}),
		},
		{
			name: "spec changed",
			to:   sub(func(s *olmv1alpha1.Subscription) { s.Spec.Channel = "candidate" }),
			want: true,
		},
		{
			name: "managed-by label removed",
			to:   sub(func(s *olmv1alpha1.Subscription) { delete(s.Labels, util.ResourceLabel) }),
			want: true,
		},
		{
			name: "current CSV changed",
			to:   sub(func(s *olmv1alpha1.Subscription) { s.Status.CurrentCSV = "tempo-operator.v2" }),
			want: true,
		},
		{
			name: "deletion started",
			to:   sub(func(s *olmv1alpha1.Subscription) { s.DeletionTimestamp = &now }),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, subscriptionChanged(event.UpdateEvent{ObjectOld: sub(nil), ObjectNew: tt.to}))
		})
	}

	// Non-subscription objects are never filtered out.
	require.True(t, subscriptionChanged(event.UpdateEvent{
		ObjectOld: &corev1.ConfigMap{}, ObjectNew: &corev1.ConfigMap{},
	}))
}
