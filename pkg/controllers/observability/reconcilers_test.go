package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	clfv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

func TestOperatorsStatus(t *testing.T) {
	external := olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo-operator", Labels: map[string]string{util.ResourceLabel: "another-operator"}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}}
	managed := olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "loki-operator", Labels: map[string]string{util.ResourceLabel: util.OpName}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "loki"}}
	status := operatorsStatus{subs: []olmv1alpha1.Subscription{external, managed}}

	require.False(t, status.ShouldInstall([]string{"tempo"}), "an externally managed subscription must be reused")
	require.False(t, status.ShouldInstall([]string{"tempo-product", "tempo"}), "an equivalent package must be reused")
	require.True(t, status.ShouldInstall([]string{"loki"}), "a COO-managed subscription remains COO's responsibility")
	require.True(t, status.ShouldInstall([]string{"cluster-logging"}), "an absent operator must be installed")
	require.Nil(t, status.cooManages("tempo"), "external subscriptions must never be removed by COO")
	require.Equal(t, &managed, status.cooManages("loki"))
	require.Nil(t, status.cooManages("cluster-logging"))

	misleading := operatorsStatus{subs: []olmv1alpha1.Subscription{{
		ObjectMeta: metav1.ObjectMeta{Name: "tempo-product-backup"},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "different-package"},
	}}}
	require.True(t, misleading.ShouldInstall([]string{"tempo-product"}), "subscription names must not be prefix-matched")
}

func TestDeletionWithoutOperatorUninstallDoesNotWait(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{
		Name: "installer", Namespace: "test", DeletionTimestamp: &metav1.Time{Time: time.Now()},
	}}
	operand := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "operand", Namespace: "test"}}
	definition := capability.Definition{
		Plan: func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) (capability.Plan, error) {
			return capability.Plan{Owned: []client.Object{operand}}, nil
		},
		Shared: func(*obsv1alpha1.ObservabilityInstaller) capability.Plan { return capability.Plan{} },
	}
	reconcilers, err := getReconcilers(context.Background(), nil, instance, []capability.Definition{definition}, operatorsStatus{}, nil)
	require.NoError(t, err)
	for _, rec := range reconcilers {
		_, waits := rec.(waitForDeletion)
		require.False(t, waits)
	}
}

func TestWaitForDeletionKeepsWaiting(t *testing.T) {
	ctx := context.Background()
	scheme := getScheme()
	operand := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "operand", Namespace: "test", Finalizers: []string{"stuck"},
		Labels: map[string]string{util.ResourceLabel: util.OpName},
	}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(operand).Build()
	wait := waitForDeletion{reader: c, objects: []client.Object{operand}}
	require.ErrorIs(t, wait.Reconcile(ctx, c, scheme), errDeletionPending)
}

func TestManagedOperatorDeleters(t *testing.T) {
	tests := []struct {
		name string
		sub  *olmv1alpha1.Subscription
		want int
	}{
		{name: "operator absent", want: 0},
		{name: "external operator is preserved", sub: &olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo", Labels: map[string]string{util.ResourceLabel: "external"}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}}, want: 0},
		{name: "managed subscription without CSV", sub: &olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo", Labels: map[string]string{util.ResourceLabel: util.OpName}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}}, want: 1},
		{name: "managed subscription and CSV", sub: &olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo", Labels: map[string]string{util.ResourceLabel: util.OpName}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}, Status: olmv1alpha1.SubscriptionStatus{CurrentCSV: "tempo.v1"}}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var subscriptions []olmv1alpha1.Subscription
			if tt.sub != nil {
				subscriptions = append(subscriptions, *tt.sub)
			}
			require.Len(t, managedOperatorDeleters("tempo", operatorsStatus{subs: subscriptions}), tt.want)
		})
	}
}

func TestGetReconcilers(t *testing.T) {
	trueVal := true

	tests := []struct {
		name                   string
		instance               *obsv1alpha1.ObservabilityInstaller
		mockClient             func() *MockClient
		installedSubscriptions []olmv1alpha1.Subscription
	}{
		{
			name: "tracing capability enabled",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&olmv1alpha1.Subscription{}), mock.Anything).Return(nil)
				mockClient.On("Apply", context.Background(), mock.Anything, mock.Anything).Return(nil)
				expectLoggingDeletes(mockClient)
				return mockClient
			},
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
								Enabled:   true,
								Operators: &obsv1alpha1.OperatorsSpec{},
							},
						},
					},
				},
			},
		},
		{
			name: "tracing capability enabled, s3 storage with TLS",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&olmv1alpha1.Subscription{}), mock.Anything).Return(nil)
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&corev1.Secret{}), mock.Anything).
					Run(func(args mock.Arguments) {
						secret := args.Get(2).(*corev1.Secret)
						secret.Data = map[string][]byte{"key": []byte("secret")}
						secret.SetLabels(map[string]string{util.ResourceLabel: util.OpName})
					}).Return(nil)
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&corev1.ConfigMap{}), mock.Anything).
					Run(func(args mock.Arguments) {
						configMap := args.Get(2).(*corev1.ConfigMap)
						configMap.Data = map[string]string{"ca.crt": "test CA"}
						configMap.SetLabels(map[string]string{util.ResourceLabel: util.OpName})
					}).Return(nil)
				mockClient.On("Apply", context.Background(), mock.Anything, mock.Anything).Return(nil)
				expectLoggingDeletes(mockClient)
				return mockClient
			},
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
								Enabled:   true,
								Operators: &obsv1alpha1.OperatorsSpec{},
							},
							Storage: &obsv1alpha1.TracingStorageSpec{
								ObjectStorageSpec: &obsv1alpha1.ObjectStorageSpec{
									S3: &obsv1alpha1.S3Spec{
										Bucket:      "tempo",
										Endpoint:    "tmepo:111",
										AccessKeyID: "id",
										AccessKeySecret: obsv1alpha1.SecretKeySelector{
											Key:  "key",
											Name: "secret-name",
										},
									},
									TLS: &obsv1alpha1.TLSSpec{
										CAConfigMap: &obsv1alpha1.ConfigMapKeySelector{
											Key:  "ca.crt",
											Name: "configmap-name",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "tracing capability disabled, install operators enabled",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&olmv1alpha1.Subscription{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Namespace{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&otelv1beta1.OpenTelemetryCollector{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRole{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRoleBinding{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&tempov1alpha1.TempoStack{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&lokiv1.LokiStack{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&clfv1.ClusterLogForwarder{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ServiceAccount{}), mock.Anything, mock.Anything).Return(nil)
				return mockClient
			},
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
								Enabled: false,
								Operators: &obsv1alpha1.OperatorsSpec{
									Install: &trueVal,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "tracing capability disabled",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Delete", context.Background(), mock.IsType(&olmv1alpha1.Subscription{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&olmv1alpha1.ClusterServiceVersion{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Namespace{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&otelv1beta1.OpenTelemetryCollector{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRole{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRoleBinding{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&tempov1alpha1.TempoStack{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&lokiv1.LokiStack{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&clfv1.ClusterLogForwarder{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ServiceAccount{}), mock.Anything, mock.Anything).Return(nil)
				return mockClient
			},
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
								Enabled: false,
							},
						},
					},
				},
			},
		},
		{
			name: "tracing capability enabled, subscription already installed",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Apply", context.Background(), mock.Anything, mock.Anything).Return(nil)
				expectLoggingDeletes(mockClient)
				return mockClient
			},
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
								Enabled: true,
							},
						},
					},
				},
			},
			installedSubscriptions: []olmv1alpha1.Subscription{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "opentelemetry-operator",
						Namespace: "openshift",
					},
					Spec: &olmv1alpha1.SubscriptionSpec{Package: "otel"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-operator",
						Namespace: "openshift",
					},
					Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"},
				},
			},
		},
		{
			name: "logging capability enabled",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&olmv1alpha1.Subscription{}), mock.Anything).Return(nil)
				mockClient.On("Apply", context.Background(), mock.Anything, mock.Anything).Return(nil)
				// the tracing objects are deleted, only logging is enabled
				mockClient.On("Delete", context.Background(), mock.IsType(&otelv1beta1.OpenTelemetryCollector{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&tempov1alpha1.TempoStack{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRole{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRoleBinding{}), mock.Anything, mock.Anything).Return(nil)
				return mockClient
			},
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Logging: &obsv1alpha1.LoggingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
								Enabled:   true,
								Operators: &obsv1alpha1.OperatorsSpec{},
							},
						},
					},
				},
			},
		},
		{
			name: "empty spec",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Delete", context.Background(), mock.IsType(&olmv1alpha1.Subscription{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&olmv1alpha1.ClusterServiceVersion{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Namespace{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&otelv1beta1.OpenTelemetryCollector{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRole{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRoleBinding{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&tempov1alpha1.TempoStack{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&lokiv1.LokiStack{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&clfv1.ClusterLogForwarder{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ServiceAccount{}), mock.Anything, mock.Anything).Return(nil)
				return mockClient
			},
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
				},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{},
			},
			installedSubscriptions: []olmv1alpha1.Subscription{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockClient := test.mockClient()
			// Legacy tracing RBAC is looked up before it is added to the cleanup inventory.
			notFound := apierrors.NewNotFound(schema.GroupResource{}, "")
			mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&rbacv1.ClusterRole{}), mock.Anything).Return(notFound).Maybe()
			mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&rbacv1.ClusterRoleBinding{}), mock.Anything).Return(notFound).Maybe()
			// The UIPlugin cleanup is label-guarded: it is read and deleted only
			// when it carries COO's part-of label.
			mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything).
				Run(func(args mock.Arguments) {
					args.Get(2).(*uiv1alpha1.UIPlugin).SetLabels(map[string]string{util.PartOfLabel: util.OpName})
				}).Return(nil).Maybe()
			mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ConfigMap{}), mock.Anything).Return(nil).Maybe()
			// Managed deleters read the live object before deleting it. The
			// objects in these tests are absent unless a case sets them up.
			mockClient.On("Get", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(notFound).Maybe()

			reconcilers, err := getReconcilers(context.Background(), mockClient, test.instance, capabilities(Options{
				OpenTelemetryOperator: OperatorInstallConfig{
					Namespace:   "operators",
					PackageName: "otel",
					StartingCSV: "otel",
					Channel:     "stable",
				},
				TempoOperator: OperatorInstallConfig{
					Namespace:   "operators",
					PackageName: "tempo",
					StartingCSV: "tempo",
					Channel:     "stable",
				},
			}), operatorsStatus{
				subs: test.installedSubscriptions,
			}, []obsv1alpha1.ObservabilityInstaller{*test.instance})
			require.NoError(t, err)

			for _, rec := range reconcilers {
				// Deletion completion is tested separately with a client that
				// models objects waiting on their finalizers.
				if _, ok := rec.(waitForDeletion); ok {
					continue
				}
				err := rec.Reconcile(context.Background(), mockClient, getScheme())
				require.NoError(t, err)
			}
		})
	}

}

func TestOperatorUninstallWaitsForOperandFinalizer(t *testing.T) {
	ctx := context.Background()
	scheme := getScheme()
	operand := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "operand", Namespace: "test", Finalizers: []string{"operand-operator/finalizer"},
		Labels: map[string]string{util.ResourceLabel: util.OpName},
	}}
	sub := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{Name: "operator", Namespace: "operators", Labels: map[string]string{util.ResourceLabel: util.OpName}},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "operator"},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(operand, sub).Build()
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "installer", Namespace: "test"}}
	definitions := []capability.Definition{{
		Plan: func(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) (capability.Plan, error) {
			return capability.Plan{Owned: []client.Object{operand}}, nil
		},
		Shared: func(*obsv1alpha1.ObservabilityInstaller) capability.Plan {
			return capability.Plan{Operators: capability.DesiredOperators(false, capability.NewOperatorRequirement("operator", capability.OperatorConfig{PackageName: "operator"}))}
		},
	}}
	status := operatorsStatus{subs: []olmv1alpha1.Subscription{*sub}}
	reconcile := func() error {
		recs, err := getReconcilers(ctx, k8sClient, instance, definitions, status, []obsv1alpha1.ObservabilityInstaller{*instance})
		require.NoError(t, err)
		for _, rec := range recs {
			if err := rec.Reconcile(ctx, k8sClient, scheme); err != nil {
				return err
			}
		}
		return nil
	}

	require.ErrorIs(t, reconcile(), errDeletionPending)
	var live corev1.ConfigMap
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(operand), &live))
	require.NotNil(t, live.DeletionTimestamp, "Delete was accepted but the operand is still finalizing")
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{}), "subscription must remain installed")

	live.Finalizers = nil
	require.NoError(t, k8sClient.Update(ctx, &live))
	require.NoError(t, reconcile())
	require.True(t, apierrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{})))
}

func TestWaitForDeletionIgnoresUnownedPlugin(t *testing.T) {
	ctx := context.Background()
	scheme := getScheme()
	plugin := &uiv1alpha1.UIPlugin{ObjectMeta: metav1.ObjectMeta{Name: "tracing"}}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(plugin).Build()
	wait := waitForDeletion{reader: k8sClient, objects: []client.Object{plugin}}
	require.NoError(t, wait.Reconcile(ctx, k8sClient, scheme))
	plugin.Labels = map[string]string{util.PartOfLabel: util.OpName}
	require.NoError(t, k8sClient.Update(ctx, plugin))
	require.True(t, errors.Is(wait.Reconcile(ctx, k8sClient, scheme), errDeletionPending))
}

// expectLoggingDeletes allows deletion of the logging objects, which are removed
// when the logging capability is not enabled.
func expectLoggingDeletes(mockClient *MockClient) {
	mockClient.On("Delete", context.Background(), mock.IsType(&lokiv1.LokiStack{}), mock.Anything, mock.Anything).Return(nil)
	mockClient.On("Delete", context.Background(), mock.IsType(&clfv1.ClusterLogForwarder{}), mock.Anything, mock.Anything).Return(nil)
	mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ServiceAccount{}), mock.Anything, mock.Anything).Return(nil)
	mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
	mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRole{}), mock.Anything, mock.Anything).Return(nil)
	mockClient.On("Delete", context.Background(), mock.IsType(&rbacv1.ClusterRoleBinding{}), mock.Anything, mock.Anything).Return(nil)
	mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything).Return(nil)
}

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(uiv1alpha1.AddToScheme(scheme))
	utilruntime.Must(obsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(otelv1beta1.AddToScheme(scheme))
	utilruntime.Must(tempov1alpha1.AddToScheme(scheme))
	return scheme
}

// MockClient is a mock implementation of client.Client.
type MockClient struct {
	mock.Mock
}

var _ client.Client = &MockClient{}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.SubResourceWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(_ runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *MockClient) IsObjectNamespaced(_ runtime.Object) (bool, error) {
	return true, nil
}
