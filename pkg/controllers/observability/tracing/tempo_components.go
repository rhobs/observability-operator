package tracing

import (
	"context"
	"fmt"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	storage "github.com/rhobs/observability-operator/pkg/controllers/observability/storage"
)

const (
	tenantName = "application"
	tenantID   = "1610b0c3-c509-4592-a256-a1871353dbfb"
)

func tempoStack(instance *obsv1alpha1.ObservabilityInstaller) *tempov1alpha1.TempoStack {
	var storageType tempov1alpha1.ObjectStorageSecretType
	if oss := instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec(); oss != nil {
		storageType = toTempoStorageType(oss)
	}
	credentialMode := toTempoCredentialMode(instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec())
	tempo := &tempov1alpha1.TempoStack{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TempoStack",
			APIVersion: tempov1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoName(instance.Name),
			Namespace: instance.Namespace,
		},
		Spec: tempov1alpha1.TempoStackSpec{
			Storage: tempov1alpha1.ObjectStorageSpec{
				Secret: tempov1alpha1.ObjectStorageSecretSpec{
					Type:           storageType,
					CredentialMode: credentialMode,
					Name:           tempoSecretName(instance.Name),
				},
			},
			Template: tempov1alpha1.TempoTemplateSpec{
				Gateway: tempov1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
			},
			Tenants: &tempov1alpha1.TenantsSpec{
				Mode: tempov1alpha1.ModeOpenShift,
				Authentication: []tempov1alpha1.AuthenticationSpec{
					{
						TenantName: tenantName,
						TenantID:   tenantID,
					},
				},
			},
		},
	}

	if storageSpec := instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec(); storageSpec != nil {
		tls := storageSpec.GetTLS()
		enableTLS := tls != nil || storage.HasHTTPSEndpoint(*storageSpec)

		if enableTLS {
			tempo.Spec.Storage.TLS = tempov1alpha1.TLSSpec{
				Enabled: true,
			}
			if tls != nil {
				if tls.CAConfigMap != nil {
					tempo.Spec.Storage.TLS.CA = tempoStorageCAConfigMapName(instance.Name)
				}
				if tls.CertSecret != nil {
					tempo.Spec.Storage.TLS.Cert = tempoStorageSecretName(instance.Name)
				}
				if tls.MinVersion != "" {
					tempo.Spec.Storage.TLS.MinVersion = tls.MinVersion
				}
			}
		}
	}

	return tempo
}

func tempoName(instance string) string {
	return instance
}

func tempoStorageCAConfigMapName(name string) string {
	return fmt.Sprintf("coo-%s-tempo-storage-ca", name)
}

// tempoStorageSecretName returns the name of the secret that contains the TLS cert and key for the object storage.
func tempoStorageSecretName(name string) string {
	return fmt.Sprintf("coo-%s-tempo-storage-cert", name)
}

// tempoSecretName returns the name of the secret that contains the credentials for the object storage.
func tempoSecretName(name string) string {
	return fmt.Sprintf("coo-%s-tempo", name)
}

func tempoStackSecrets(ctx context.Context, k8sReader client.Reader, instance obsv1alpha1.ObservabilityInstaller) (*storage.Resources, error) {
	return storage.Materialize(ctx, k8sReader, instance.Namespace,
		instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec(),
		storage.Names{
			Secret:      tempoSecretName(instance.Name),
			TLSSecret:   tempoStorageSecretName(instance.Name),
			CAConfigMap: tempoStorageCAConfigMapName(instance.Name),
		}, storage.TempoFormat)
}

func toTempoStorageType(objStorage *obsv1alpha1.ObjectStorageSpec) tempov1alpha1.ObjectStorageSecretType {
	if objStorage == nil {
		return ""
	}
	if objStorage.S3 != nil || objStorage.S3STS != nil || objStorage.S3CCO != nil {
		return tempov1alpha1.ObjectStorageSecretS3
	} else if objStorage.Azure != nil || objStorage.AzureWIF != nil {
		return tempov1alpha1.ObjectStorageSecretAzure
	} else if objStorage.GCS != nil || objStorage.GCSWIF != nil {
		return tempov1alpha1.ObjectStorageSecretGCS
	}
	return ""
}

func toTempoCredentialMode(objStorage *obsv1alpha1.ObjectStorageSpec) tempov1alpha1.CredentialMode {
	if objStorage == nil {
		return ""
	}
	if objStorage.S3 != nil || objStorage.Azure != nil || objStorage.GCS != nil {
		return tempov1alpha1.CredentialModeStatic
	} else if objStorage.S3STS != nil || objStorage.AzureWIF != nil || objStorage.GCSWIF != nil {
		return tempov1alpha1.CredentialModeToken
	} else if objStorage.S3CCO != nil {
		return tempov1alpha1.CredentialModeTokenCCO
	}

	return ""
}
