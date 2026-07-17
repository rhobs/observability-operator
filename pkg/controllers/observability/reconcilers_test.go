package observability

import (
	"context"
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

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
				mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
				mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
										Endpoint:    "tempo:111",
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
				mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
				mockClient.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
				mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockClient.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
						Name:      "opentelemetry-product",
						Namespace: "openshift-opentelemetry-operator",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tempo-product",
						Namespace: "openshift-tempo-operator",
					},
				},
			},
		},
		{
			name: "empty spec",
			mockClient: func() *MockClient {
				mockClient := &MockClient{}
				mockClient.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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

			reconcilers, err := getReconcilers(context.Background(), mockClient, test.instance, Options{
				COONamespace: "operators",
				OpenTelemetryOperator: OperatorInstallConfig{
					Namespace:   "openshift-opentelemetry-operator",
					PackageName: "otel",
					StartingCSV: "otel",
					Channel:     "stable",
				},
				TempoOperator: OperatorInstallConfig{
					Namespace:   "openshift-tempo-operator",
					PackageName: "tempo",
					StartingCSV: "tempo",
					Channel:     "stable",
				},
			}, operatorsStatus{
				subs: test.installedSubscriptions,
			})
			require.NoError(t, err)

			for _, rec := range reconcilers {
				err := rec.Reconcile(context.Background(), mockClient, getScheme())
				require.NoError(t, err)
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
