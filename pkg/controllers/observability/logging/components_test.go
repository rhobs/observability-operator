package logging

import (
	"testing"

	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	clfv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestLokiStack(t *testing.T) {
	tests := []struct {
		name               string
		instance           *obsv1alpha1.ObservabilityInstaller
		wantStorageType    lokiv1.ObjectStorageSecretType
		wantCredentialMode lokiv1.CredentialMode
		wantTLSEnabled     bool
		wantTLSCASet       bool
	}{
		{
			name: "nil capabilities - does not panic",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec:       obsv1alpha1.ObservabilityInstallerSpec{},
			},
		},
		{
			name: "nil logging - does not panic",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{},
				},
			},
		},
		{
			name: "nil lokiStack - does not panic",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Logging: &obsv1alpha1.LoggingSpec{},
					},
				},
			},
		},
		{
			name: "nil storage - does not panic",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Logging: &obsv1alpha1.LoggingSpec{
							LokiStack: &obsv1alpha1.LokiStackSpec{},
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
						Logging: &obsv1alpha1.LoggingSpec{
							LokiStack: &obsv1alpha1.LokiStackSpec{
								Size:             lokiv1.SizeOneXPico,
								StorageClassName: "standard",
								ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
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
			wantStorageType:    lokiv1.ObjectStorageSecretS3,
			wantCredentialMode: lokiv1.CredentialModeStatic,
		},
		{
			name: "Azure storage - sets Azure type and static credential mode",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Logging: &obsv1alpha1.LoggingSpec{
							LokiStack: &obsv1alpha1.LokiStackSpec{
								Size:             lokiv1.SizeOneXPico,
								StorageClassName: "standard",
								ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
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
			wantStorageType:    lokiv1.ObjectStorageSecretAzure,
			wantCredentialMode: lokiv1.CredentialModeStatic,
		},
		{
			name: "GCS storage - sets GCS type and static credential mode",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Logging: &obsv1alpha1.LoggingSpec{
							LokiStack: &obsv1alpha1.LokiStackSpec{
								Size:             lokiv1.SizeOneXPico,
								StorageClassName: "standard",
								ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
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
			wantStorageType:    lokiv1.ObjectStorageSecretGCS,
			wantCredentialMode: lokiv1.CredentialModeStatic,
		},
		{
			// Loki's TLS spec requires a CA configmap name, so an HTTPS endpoint
			// without a user CA must leave TLS unset and use the system CAs.
			name: "S3 with HTTPS endpoint and no CA - leaves TLS unset",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Logging: &obsv1alpha1.LoggingSpec{
							LokiStack: &obsv1alpha1.LokiStackSpec{
								Size:             lokiv1.SizeOneXPico,
								StorageClassName: "standard",
								ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
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
			wantStorageType:    lokiv1.ObjectStorageSecretS3,
			wantCredentialMode: lokiv1.CredentialModeStatic,
		},
		{
			name: "S3 with explicit TLS CA configmap - sets TLS CA reference",
			instance: &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{
					Capabilities: &obsv1alpha1.CapabilitiesSpec{
						Logging: &obsv1alpha1.LoggingSpec{
							LokiStack: &obsv1alpha1.LokiStackSpec{
								Size:             lokiv1.SizeOneXPico,
								StorageClassName: "standard",
								ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
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
			wantStorageType:    lokiv1.ObjectStorageSecretS3,
			wantCredentialMode: lokiv1.CredentialModeStatic,
			wantTLSEnabled:     true,
			wantTLSCASet:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lokiStack(tt.instance)

			require.NotNil(t, result)
			assert.Equal(t, lokiStackName(tt.instance.Name), result.Name)
			assert.Equal(t, tt.instance.Namespace, result.Namespace)
			assert.Equal(t, tt.wantStorageType, result.Spec.Storage.Secret.Type)
			assert.Equal(t, tt.wantCredentialMode, result.Spec.Storage.Secret.CredentialMode)

			if result.Spec.Storage.TLS != nil {
				assert.True(t, tt.wantTLSEnabled, "unexpected TLS enabled")
			}
			if tt.wantTLSEnabled {
				assert.NotNil(t, result.Spec.Storage.TLS, "expected TLS to be set")
			}
			if tt.wantTLSCASet {
				assert.NotEmpty(t, result.Spec.Storage.TLS.CA)
			}
		})
	}
}

func TestClusterLogForwarder(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{
			Capabilities: &obsv1alpha1.CapabilitiesSpec{
				Logging: &obsv1alpha1.LoggingSpec{
					CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
						Enabled: true,
					},
					LokiStack: &obsv1alpha1.LokiStackSpec{
						Size:             lokiv1.SizeOneXPico,
						StorageClassName: "standard",
					},
				},
			},
		},
	}

	result := clusterLogForwarder(instance)

	require.NotNil(t, result)
	assert.Equal(t, clusterLogForwarderName(instance.Name), result.Name)
	assert.Equal(t, instance.Namespace, result.Namespace)
	assert.Equal(t, instance.Name+"-logging-collector", result.Spec.ServiceAccount.Name)

	require.Len(t, result.Spec.Outputs, 1)
	assert.Equal(t, "default-lokistack", result.Spec.Outputs[0].Name)
	assert.Equal(t, clfv1.OutputTypeLokiStack, result.Spec.Outputs[0].Type)
	assert.Equal(t, lokiStackName(instance.Name), result.Spec.Outputs[0].LokiStack.Target.Name)
	assert.Equal(t, instance.Namespace, result.Spec.Outputs[0].LokiStack.Target.Namespace)

	require.Len(t, result.Spec.Pipelines, 1)
	assert.Equal(t, "default-logstore", result.Spec.Pipelines[0].Name)
	assert.Equal(t, []string{"application", "infrastructure"}, result.Spec.Pipelines[0].InputRefs)
	assert.Equal(t, []string{"default-lokistack"}, result.Spec.Pipelines[0].OutputRefs)
}

func TestLoggingServiceAccount(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
	}

	result := loggingServiceAccount(instance)

	require.NotNil(t, result)
	assert.Equal(t, "test-logging-collector", result.Name)
	assert.Equal(t, "test-ns", result.Namespace)
}

func TestLoggingCollectorRBAC(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test-ns"},
	}

	role, bindings := loggingCollectorRBAC(instance)

	require.NotNil(t, role)
	require.Len(t, bindings, 3)

	assert.Equal(t, "coo-test-ns-test-logging", role.Name)
	require.Len(t, role.Rules, 1)
	assert.Equal(t, []string{"loki.grafana.com"}, role.Rules[0].APIGroups)
	assert.Equal(t, []string{"application", "audit", "infrastructure"}, role.Rules[0].Resources)
	assert.Equal(t, []string{"create"}, role.Rules[0].Verbs)

	wantBindings := []struct {
		name    string
		roleRef string
	}{
		{name: "coo-test-ns-test-logging", roleRef: "coo-test-ns-test-logging"},
		{name: "coo-test-ns-test-logging-collect-application", roleRef: "collect-application-logs"},
		{name: "coo-test-ns-test-logging-collect-infrastructure", roleRef: "collect-infrastructure-logs"},
	}
	for i, want := range wantBindings {
		assert.Equal(t, want.name, bindings[i].Name)
		assert.Equal(t, want.roleRef, bindings[i].RoleRef.Name)
		require.Len(t, bindings[i].Subjects, 1)
		assert.Equal(t, "test-logging-collector", bindings[i].Subjects[0].Name)
		assert.Equal(t, "test-ns", bindings[i].Subjects[0].Namespace)
	}
}

func TestToLokiStorageType(t *testing.T) {
	tests := []struct {
		name    string
		storage *obsv1alpha1.ObjectStorageSpec
		want    lokiv1.ObjectStorageSecretType
	}{
		{"nil", nil, ""},
		{"S3", &obsv1alpha1.ObjectStorageSpec{S3: &obsv1alpha1.S3Spec{}}, lokiv1.ObjectStorageSecretS3},
		{"S3STS", &obsv1alpha1.ObjectStorageSpec{S3STS: &obsv1alpha1.S3STSpec{}}, lokiv1.ObjectStorageSecretS3},
		{"S3CCO", &obsv1alpha1.ObjectStorageSpec{S3CCO: &obsv1alpha1.S3CCOSpec{}}, lokiv1.ObjectStorageSecretS3},
		{"Azure", &obsv1alpha1.ObjectStorageSpec{Azure: &obsv1alpha1.AzureSpec{}}, lokiv1.ObjectStorageSecretAzure},
		{"AzureWIF", &obsv1alpha1.ObjectStorageSpec{AzureWIF: &obsv1alpha1.AzureWIFSpec{}}, lokiv1.ObjectStorageSecretAzure},
		{"GCS", &obsv1alpha1.ObjectStorageSpec{GCS: &obsv1alpha1.GCSSpec{}}, lokiv1.ObjectStorageSecretGCS},
		{"GCSWIF", &obsv1alpha1.ObjectStorageSpec{GCSWIF: &obsv1alpha1.GCSWIFSpec{}}, lokiv1.ObjectStorageSecretGCS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLokiStorageType(tt.storage)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToLokiCredentialMode(t *testing.T) {
	tests := []struct {
		name    string
		storage *obsv1alpha1.ObjectStorageSpec
		want    lokiv1.CredentialMode
	}{
		{"nil", nil, ""},
		{"S3 static", &obsv1alpha1.ObjectStorageSpec{S3: &obsv1alpha1.S3Spec{}}, lokiv1.CredentialModeStatic},
		{"Azure static", &obsv1alpha1.ObjectStorageSpec{Azure: &obsv1alpha1.AzureSpec{}}, lokiv1.CredentialModeStatic},
		{"GCS static", &obsv1alpha1.ObjectStorageSpec{GCS: &obsv1alpha1.GCSSpec{}}, lokiv1.CredentialModeStatic},
		{"S3STS token", &obsv1alpha1.ObjectStorageSpec{S3STS: &obsv1alpha1.S3STSpec{}}, lokiv1.CredentialModeToken},
		{"AzureWIF token", &obsv1alpha1.ObjectStorageSpec{AzureWIF: &obsv1alpha1.AzureWIFSpec{}}, lokiv1.CredentialModeToken},
		{"GCSWIF token", &obsv1alpha1.ObjectStorageSpec{GCSWIF: &obsv1alpha1.GCSWIFSpec{}}, lokiv1.CredentialModeToken},
		{"S3CCO token-cco", &obsv1alpha1.ObjectStorageSpec{S3CCO: &obsv1alpha1.S3CCOSpec{}}, lokiv1.CredentialModeTokenCCO},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLokiCredentialMode(tt.storage)
			assert.Equal(t, tt.want, got)
		})
	}
}
