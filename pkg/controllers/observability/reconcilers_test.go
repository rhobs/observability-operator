package observability

import (
	"context"
	"errors"
	"testing"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type uninventoriedInstallerCapability struct{ fakeCapability }

func (uninventoriedInstallerCapability) Desired(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) ([]client.Object, error) {
	return []client.Object{configMap("test-namespace")}, nil
}

func TestGetReconcilersRejectsUninventoriedInstallerResource(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-namespace"}}
	c := uninventoriedInstallerCapability{fakeCapability{name: "broken"}}
	reconcilers, err := getReconcilers(context.Background(), fake.NewClientBuilder().WithScheme(getScheme()).Build(),
		instance, []capability.Capability{c}, operatorsStatus{}, []obsv1alpha1.ObservabilityInstaller{*instance})
	require.ErrorContains(t, err, "not in its inventory")
	require.ErrorContains(t, err, "broken")
	require.Empty(t, reconcilers)
}

func TestGetReconcilersPreservesSharedInventoryOnPlanningError(t *testing.T) {
	ctx := context.Background()
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	shared := configMap("test")
	planErr := errors.New("missing credential")
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(shared).Build()
	reconcilers, err := getReconcilers(ctx, c, instance, []capability.Capability{
		failingDesiredCapability{fakeCapability: fakeCapability{name: "broken", shared: []client.Object{shared}}, err: planErr},
	}, operatorsStatus{}, []obsv1alpha1.ObservabilityInstaller{*instance})
	require.ErrorIs(t, err, planErr)
	for _, rec := range reconcilers {
		require.NoError(t, rec.Reconcile(ctx, c, getScheme()))
	}
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(shared), &corev1.ConfigMap{}))
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
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ConfigMap{}), mock.Anything, mock.Anything).Return(nil)
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
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&corev1.Secret{}), mock.Anything).Return(nil)
				mockClient.On("Get", context.Background(), mock.Anything, mock.IsType(&corev1.ConfigMap{}), mock.Anything).Return(nil)
				mockClient.On("Apply", context.Background(), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ConfigMap{}), mock.Anything, mock.Anything).Return(nil)
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
								ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{
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
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ConfigMap{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything, mock.Anything).Return(nil)
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
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ConfigMap{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything, mock.Anything).Return(nil)
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
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.Secret{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ConfigMap{}), mock.Anything, mock.Anything).Return(nil)
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
					Spec: &olmv1alpha1.SubscriptionSpec{Package: "opentelemetry-operator"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-operator",
						Namespace: "openshift",
					},
					Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo-operator"},
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
				mockClient.On("Delete", context.Background(), mock.IsType(&corev1.ConfigMap{}), mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", context.Background(), mock.IsType(&uiv1alpha1.UIPlugin{}), mock.Anything, mock.Anything).Return(nil)
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

			reconcilers, err := getReconcilers(context.Background(), mockClient, test.instance, capabilities(Options{
				COONamespace: "operators",
				OpenTelemetryOperator: OperatorInstallConfig{
					Namespace:          "operators",
					PackageName:        "otel",
					StartingCSV:        "otel",
					Channel:            "stable",
					EquivalentPackages: []string{"opentelemetry-operator"},
				},
				TempoOperator: OperatorInstallConfig{
					Namespace:          "operators",
					PackageName:        "tempo",
					StartingCSV:        "tempo",
					Channel:            "stable",
					EquivalentPackages: []string{"tempo-operator"},
				},
			}), operatorsStatus{
				subs: test.installedSubscriptions,
			}, []obsv1alpha1.ObservabilityInstaller{*test.instance})
			require.NoError(t, err)

			for _, rec := range reconcilers {
				err := rec.Reconcile(context.Background(), mockClient, getScheme())
				require.NoError(t, err)
			}
		})
	}

}

func TestMissingCRDDeleter(t *testing.T) {
	target := &otelv1beta1.OpenTelemetryCollector{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	otherErr := errors.New("delete failed")
	for _, tt := range []struct {
		name      string
		deleteErr error
		wantErr   error
	}{
		{name: "missing object"},
		{name: "missing CRD", deleteErr: &meta.NoKindMatchError{GroupKind: schema.GroupKind{Kind: "OpenTelemetryCollector"}}},
		{name: "other error", deleteErr: otherErr, wantErr: otherErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(getScheme()).WithInterceptorFuncs(interceptor.Funcs{
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					if tt.deleteErr != nil {
						return tt.deleteErr
					}
					return c.Delete(ctx, obj, opts...)
				},
			}).Build()
			err := missingCRDDeleter{reconciler.NewDeleter(target)}.Reconcile(context.Background(), c, getScheme())
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
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
