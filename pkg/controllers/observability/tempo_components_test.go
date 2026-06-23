package observability

import (
	"testing"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestTempoStack(t *testing.T) {
	tests := []struct {
		name               string
		instance           *obsv1alpha1.ObservabilityInstaller
		wantStorageType    tempov1alpha1.ObjectStorageSecretType
		wantCredentialMode tempov1alpha1.CredentialMode
		wantTLSEnabled     bool
		wantTLSCASet       bool
		wantTLSCertSet     bool
	}{
		{
			name: "nil capabilities - does not panic",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec:       obsv1alpha1.ObservabilityInstallerSpec{},
			},
		},
		{
			name: "nil tracing - does not panic",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{},
				},
			},
		},
		{
			name: "nil storage - does not panic",
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
			name: "nil object storage spec - does not panic",
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
			wantStorageType:    tempov1alpha1.ObjectStorageSecretS3,
			wantCredentialMode: tempov1alpha1.CredentialModeStatic,
			wantTLSEnabled:     true,
			wantTLSCertSet:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tempoStack(tt.instance)

			require.NotNil(t, result)
			assert.Equal(t, tt.instance.Name, result.Name)
			assert.Equal(t, tt.instance.Namespace, result.Namespace)
			assert.Equal(t, tt.wantStorageType, result.Spec.Storage.Secret.Type)
			assert.Equal(t, tt.wantCredentialMode, result.Spec.Storage.Secret.CredentialMode)
			assert.Equal(t, tt.wantTLSEnabled, result.Spec.Storage.TLS.Enabled)

			if tt.wantTLSCASet {
				assert.NotEmpty(t, result.Spec.Storage.TLS.CA)
			}
			if tt.wantTLSCertSet {
				assert.NotEmpty(t, result.Spec.Storage.TLS.Cert)
			}
		})
	}
}
