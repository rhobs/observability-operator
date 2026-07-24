package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhobs/observability-operator/config"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildOverlayTracing(t *testing.T) {
	tests := []struct {
		name           string
		instance       *obsv1alpha1.ObservabilityInstaller
		wantObjects    bool
		wantObjectName string
	}{
		{
			name: "nil capabilities - produces empty overlay",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec:       obsv1alpha1.ObservabilityInstallerSpec{},
			},
		},
		{
			name: "tracing disabled - produces empty overlay",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{},
					},
				},
			},
		},
		{
			name: "tracing enabled with S3 storage - produces overlay objects",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
							Storage: &obsv1alpha1.TracingStorageSpec{
								ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{
									S3: &obsv1alpha1.S3Spec{
										Bucket:   "test-bucket",
										Endpoint: "http://minio:9000",
									},
								},
							},
						},
					},
				},
			},
			wantObjects:    true,
			wantObjectName: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay, err := BuildOverlay(tt.instance, OverlayConfig{ConfigFS: config.FS, COONamespace: "test-namespace"})
			require.NoError(t, err)

			objects, err := overlay.Build()
			require.NoError(t, err)

			if !tt.wantObjects {
				assert.Empty(t, objects)
				return
			}

			require.NotEmpty(t, objects)

			var found bool
			for _, obj := range objects {
				if obj.GetObjectKind().GroupVersionKind().Kind == "TempoStack" {
					found = true
					assert.Equal(t, tt.wantObjectName, obj.GetName())
				}
			}
			assert.True(t, found, "expected TempoStack object in overlay output")
		})
	}
}

func TestBuildOverlayContainsExpectedResources(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{
			Capabilities: &obsv1alpha1.CapabilitiesSpec{
				Tracing: &obsv1alpha1.TracingSpec{
					CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
					Storage: &obsv1alpha1.TracingStorageSpec{
						ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{
							S3: &obsv1alpha1.S3Spec{
								Bucket:   "test-bucket",
								Endpoint: "http://minio:9000",
							},
						},
					},
				},
			},
		},
	}

	overlay, err := BuildOverlay(instance, OverlayConfig{ConfigFS: config.FS, COONamespace: "test-namespace"})
	require.NoError(t, err)

	objects, err := overlay.Build()
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, obj := range objects {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		kinds[kind] = append(kinds[kind], obj.GetName())
	}

	assert.Contains(t, kinds, "TempoStack")
	assert.Contains(t, kinds, "OpenTelemetryCollector")
	assert.Contains(t, kinds, "UIPlugin")
	assert.Contains(t, kinds, "Subscription")
	assert.Contains(t, kinds, "ClusterRole")
	assert.Contains(t, kinds, "ClusterRoleBinding")
	assert.NotContains(t, kinds, "Namespace")
	assert.NotContains(t, kinds, "OperatorGroup")
}

func TestBuildOverlaySubscriptionPatch(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{
			Capabilities: &obsv1alpha1.CapabilitiesSpec{
				Tracing: &obsv1alpha1.TracingSpec{
					CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
					Storage: &obsv1alpha1.TracingStorageSpec{
						ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{
							S3: &obsv1alpha1.S3Spec{
								Bucket:   "test-bucket",
								Endpoint: "http://minio:9000",
							},
						},
					},
				},
			},
		},
	}

	cfg := OverlayConfig{
		ConfigFS:     config.FS,
		COONamespace: "test-namespace",
		OpenTelemetryOperator: OperatorInstallConfig{
			Namespace:   "openshift-opentelemetry-operator",
			StartingCSV: "opentelemetry-operator.v0.100.0",
		},
		TempoOperator: OperatorInstallConfig{
			Namespace:   "openshift-tempo-operator",
			StartingCSV: "tempo-operator.v0.10.0",
		},
	}

	overlay, err := BuildOverlay(instance, cfg)
	require.NoError(t, err)

	yamlOut, err := overlay.BuildYAML()
	require.NoError(t, err)

	yamlStr := string(yamlOut)
	assert.Contains(t, yamlStr, "opentelemetry-operator.v0.100.0")
	assert.Contains(t, yamlStr, "tempo-operator.v0.10.0")
}
