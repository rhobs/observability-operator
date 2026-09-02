package observability

import (
	"testing"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/rhobs/observability-operator/config"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestBuildOverlayTracing(t *testing.T) {
	tests := []struct {
		name               string
		instance           *obsv1alpha1.ObservabilityInstaller
		wantTempoStack     bool
		wantStorageType    tempov1alpha1.ObjectStorageSecretType
		wantCredentialMode tempov1alpha1.CredentialMode
		wantTLSEnabled     bool
		wantTLSCASet       bool
		wantTLSCertSet     bool
	}{
		{
			name: "nil capabilities - produces empty overlay",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec:       obsv1alpha1.ObservabilityInstallerSpec{},
			},
		},
		{
			name: "nil tracing - produces empty overlay",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{},
				},
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
			name: "nil object storage spec - produces empty overlay",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							Storage: &obsv1alpha1.TracingStorageSpec{},
						},
					},
				},
			},
		},
		{
			name: "S3 storage - sets S3 type and static credential mode",
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
			wantTempoStack:     true,
			wantStorageType:    tempov1alpha1.ObjectStorageSecretS3,
			wantCredentialMode: tempov1alpha1.CredentialModeStatic,
		},
		{
			name: "S3STS storage - sets S3 type and token credential mode",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
							Storage: &obsv1alpha1.TracingStorageSpec{
								ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{
									S3STS: &obsv1alpha1.S3STSpec{
										Bucket:  "test-bucket",
										RoleARN: "arn:aws:iam::123:role/test",
										Region:  "us-east-1",
									},
								},
							},
						},
					},
				},
			},
			wantTempoStack:     true,
			wantStorageType:    tempov1alpha1.ObjectStorageSecretS3,
			wantCredentialMode: tempov1alpha1.CredentialModeToken,
		},
		{
			name: "Azure storage - sets Azure type and static credential mode",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
							Storage: &obsv1alpha1.TracingStorageSpec{
								ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{
									Azure: &obsv1alpha1.AzureSpec{
										Container:   "test-container",
										AccountName: "test-account",
										AccountKeySecret: obsv1alpha1.SecretKeySelector{
											Name: "secret",
											Key:  "key",
										},
									},
								},
							},
						},
					},
				},
			},
			wantTempoStack:     true,
			wantStorageType:    tempov1alpha1.ObjectStorageSecretAzure,
			wantCredentialMode: tempov1alpha1.CredentialModeStatic,
		},
		{
			name: "GCS storage - sets GCS type and static credential mode",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Tracing: &obsv1alpha1.TracingSpec{
							CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
							Storage: &obsv1alpha1.TracingStorageSpec{
								ObjectStorageSpec: &obsv1alpha1.TracingObjectStorageSpec{
									GCS: &obsv1alpha1.GCSSpec{
										Bucket: "test-bucket",
										KeyJSONSecret: obsv1alpha1.SecretKeySelector{
											Name: "secret",
											Key:  "key.json",
										},
									},
								},
							},
						},
					},
				},
			},
			wantTempoStack:     true,
			wantStorageType:    tempov1alpha1.ObjectStorageSecretGCS,
			wantCredentialMode: tempov1alpha1.CredentialModeStatic,
		},
		{
			name: "S3 with HTTPS endpoint - enables TLS automatically",
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
										Endpoint: "https://s3.amazonaws.com",
									},
								},
							},
						},
					},
				},
			},
			wantTempoStack:     true,
			wantStorageType:    tempov1alpha1.ObjectStorageSecretS3,
			wantCredentialMode: tempov1alpha1.CredentialModeStatic,
			wantTLSEnabled:     true,
		},
		{
			name: "S3 with explicit TLS CA configmap - sets TLS CA reference",
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
									TLS: &obsv1alpha1.TLSSpec{
										CAConfigMap: &obsv1alpha1.ConfigMapKeySelector{
											Name: "ca-configmap",
											Key:  "ca.crt",
										},
									},
								},
							},
						},
					},
				},
			},
			wantTempoStack:     true,
			wantStorageType:    tempov1alpha1.ObjectStorageSecretS3,
			wantCredentialMode: tempov1alpha1.CredentialModeStatic,
			wantTLSEnabled:     true,
			wantTLSCASet:       true,
		},
		{
			name: "S3 with explicit TLS cert secret - sets TLS cert reference",
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
									TLS: &obsv1alpha1.TLSSpec{
										CertSecret: &obsv1alpha1.SecretKeySelector{
											Name: "cert-secret",
											Key:  "tls.crt",
										},
									},
								},
							},
						},
					},
				},
			},
			wantTempoStack:     true,
			wantStorageType:    tempov1alpha1.ObjectStorageSecretS3,
			wantCredentialMode: tempov1alpha1.CredentialModeStatic,
			wantTLSEnabled:     true,
			wantTLSCertSet:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay, err := BuildInstallerOverlay(tt.instance, OverlayConfig{
				Scheme:    getScheme(),
				ConfigFS:  config.FS,
				Options:   Options{COONamespace: "test-namespace"},
				Operators: true,
				Resources: true})
			require.NoError(t, err)

			objects, err := overlay.Build()
			require.NoError(t, err)

			if !tt.wantTempoStack {
				for _, obj := range objects {
					assert.NotEqual(t, "TempoStack", obj.GetObjectKind().GroupVersionKind().Kind,
						"unexpected TempoStack in overlay output")
				}
				return
			}

			var tempo *unstructured.Unstructured
			for _, obj := range objects {
				if obj.GetObjectKind().GroupVersionKind().Kind == "TempoStack" {
					tempo = obj.(*unstructured.Unstructured)
					break
				}
			}
			require.NotNil(t, tempo, "expected TempoStack object in overlay output")
			assert.Equal(t, tt.instance.Name, tempo.GetName())
			assert.Equal(t, tt.instance.Namespace, tempo.GetNamespace())

			storageType, _, _ := unstructured.NestedString(tempo.Object, "spec", "storage", "secret", "type")
			credentialMode, _, _ := unstructured.NestedString(tempo.Object, "spec", "storage", "secret", "credentialMode")
			assert.Equal(t, string(tt.wantStorageType), storageType)
			assert.Equal(t, string(tt.wantCredentialMode), credentialMode)

			tlsEnabled, _, _ := unstructured.NestedBool(tempo.Object, "spec", "storage", "tls", "enabled")
			assert.Equal(t, tt.wantTLSEnabled, tlsEnabled)

			if tt.wantTLSCASet {
				tlsCA, _, _ := unstructured.NestedString(tempo.Object, "spec", "storage", "tls", "ca")
				assert.NotEmpty(t, tlsCA)
			}
			if tt.wantTLSCertSet {
				tlsCert, _, _ := unstructured.NestedString(tempo.Object, "spec", "storage", "tls", "cert")
				assert.NotEmpty(t, tlsCert)
			}
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

	overlay, err := BuildInstallerOverlay(instance, OverlayConfig{
		Scheme:    getScheme(),
		ConfigFS:  config.FS,
		Options:   Options{COONamespace: "test-namespace"},
		Operators: true,
		Resources: true})
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
		Scheme:   getScheme(),
		ConfigFS: config.FS,
		Options: Options{
			COONamespace: "test-namespace",
			OpenTelemetryOperator: OperatorInstallConfig{
				Namespace:   "openshift-tracing",
				StartingCSV: "opentelemetry-operator.v0.100.0",
			},
			TempoOperator: OperatorInstallConfig{
				Namespace:   "openshift-tracing",
				StartingCSV: "tempo-operator.v0.10.0",
			},
		},
		Operators: true,
		Resources: true,
	}

	overlay, err := BuildInstallerOverlay(instance, cfg)
	require.NoError(t, err)

	yamlOut, err := overlay.BuildYAML()
	require.NoError(t, err)

	yamlStr := string(yamlOut)
	assert.Contains(t, yamlStr, "opentelemetry-operator.v0.100.0")
	assert.Contains(t, yamlStr, "tempo-operator.v0.10.0")
}
