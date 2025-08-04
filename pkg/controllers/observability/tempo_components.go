package observability

import (
	"context"
	"fmt"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

const (
	tempoName  = "coo"
	tenantName = "prod"
	tenantID   = "1610b0c3-c509-4592-a256-a1871353dbfb"
)

func tempoStack(storage obsv1alpha1.StorageSpec, ns string, cobs string) *tempov1alpha1.TempoStack {
	return &tempov1alpha1.TempoStack{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TempoStack",
			APIVersion: tempov1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoName,
			Namespace: ns,
		},
		Spec: tempov1alpha1.TempoStackSpec{
			Storage: tempov1alpha1.ObjectStorageSpec{
				TLS: tempov1alpha1.TLSSpec{
					Enabled:    storage.ObjectStorageSpec.TLS.Enabled,
					CA:         storage.ObjectStorageSpec.TLS.CA,
					Cert:       storage.ObjectStorageSpec.TLS.Cert,
					MinVersion: storage.ObjectStorageSpec.TLS.MinVersion,
				},
				Secret: tempov1alpha1.ObjectStorageSecretSpec{
					Type:           toTempoStorageType(storage.ObjectStorageSpec),
					CredentialMode: toTempoCredentialMode(storage.ObjectStorageSpec),
					Name:           tempoSecretName(cobs),
				},
			},
			StorageClassName: storage.PersistentVolumeClaimSpec.StorageClassName,
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
}

func tempoSecretName(name string) string {
	return fmt.Sprintf("coo-%s", name)
}

func tempoStackSecret(ctx context.Context, k8sClient client.Client, instance obsv1alpha1.ClusterObservability, tempoNamespace string, operatorNamespace string) (*corev1.Secret, error) {
	tempoSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoSecretName(instance.Name),
			Namespace: tempoNamespace,
		},
	}
	if instance.Spec.Storage.ObjectStorageSpec.S3 != nil {
		accessKeySecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: operatorNamespace,
			Name:      instance.Spec.Storage.ObjectStorageSpec.S3.AccessKeySecret.SecretName,
		}, accessKeySecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get S3 access key secret %s: %w", instance.Spec.Storage.ObjectStorageSpec.S3.AccessKeySecret.SecretName, err)
		}

		tempoSecret.Data = map[string][]byte{
			"bucket":            []byte(instance.Spec.Storage.ObjectStorageSpec.S3.Bucket),
			"endpoint":          []byte(instance.Spec.Storage.ObjectStorageSpec.S3.Endpoint),
			"access_key_id":     []byte(instance.Spec.Storage.ObjectStorageSpec.S3.AccessKeyID),
			"access_key_secret": accessKeySecret.Data[instance.Spec.Storage.ObjectStorageSpec.S3.AccessKeySecret.Key],
		}

		if instance.Spec.Storage.ObjectStorageSpec.S3.Region != "" {
			tempoSecret.Data["region"] = []byte(instance.Spec.Storage.ObjectStorageSpec.S3.Region)
		}
	} else if instance.Spec.Storage.ObjectStorageSpec.S3STS != nil {
		tempoSecret.Data = map[string][]byte{
			"bucket":   []byte(instance.Spec.Storage.ObjectStorageSpec.S3STS.Bucket),
			"role_arn": []byte(instance.Spec.Storage.ObjectStorageSpec.S3STS.RoleARN),
			"region":   []byte(instance.Spec.Storage.ObjectStorageSpec.S3STS.RoleARN),
		}
	} else if instance.Spec.Storage.ObjectStorageSpec.S3CCO != nil {
		tempoSecret.Data = map[string][]byte{
			"bucket": []byte(instance.Spec.Storage.ObjectStorageSpec.S3CCO.Bucket),
			"region": []byte(instance.Spec.Storage.ObjectStorageSpec.S3CCO.Region),
		}
	} else if instance.Spec.Storage.ObjectStorageSpec.Azure != nil {
		accountKeySecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: operatorNamespace,
			Name:      instance.Spec.Storage.ObjectStorageSpec.Azure.AccountKey.SecretName,
		}, accountKeySecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get Azure account key secret %s: %w", instance.Spec.Storage.ObjectStorageSpec.Azure.AccountKey.SecretName, err)
		}

		tempoSecret.Data = map[string][]byte{
			"container":    []byte(instance.Spec.Storage.ObjectStorageSpec.Azure.Container),
			"account_name": []byte(instance.Spec.Storage.ObjectStorageSpec.Azure.AccountName),
			"account_key":  accountKeySecret.Data[instance.Spec.Storage.ObjectStorageSpec.Azure.AccountKey.Key],
		}
	} else if instance.Spec.Storage.ObjectStorageSpec.AzureWIF != nil {
		tempoSecret.Data = map[string][]byte{
			"container":    []byte(instance.Spec.Storage.ObjectStorageSpec.AzureWIF.Container),
			"account_name": []byte(instance.Spec.Storage.ObjectStorageSpec.AzureWIF.AccountName),
			"client_id":    []byte(instance.Spec.Storage.ObjectStorageSpec.AzureWIF.ClientID),
			"tenant_id":    []byte(instance.Spec.Storage.ObjectStorageSpec.AzureWIF.TenantID),
		}
		if instance.Spec.Storage.ObjectStorageSpec.AzureWIF.Audience != "" {
			tempoSecret.Data["audience"] = []byte(instance.Spec.Storage.ObjectStorageSpec.AzureWIF.Audience)
		}
	} else if instance.Spec.Storage.ObjectStorageSpec.GCS != nil {
		accountKeySecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: operatorNamespace,
			Name:      instance.Spec.Storage.ObjectStorageSpec.GCS.KeyJSON.SecretName,
		}, accountKeySecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get GCS keyJSON secret %s: %w", instance.Spec.Storage.ObjectStorageSpec.GCS.KeyJSON.SecretName, err)
		}

		tempoSecret.Data = map[string][]byte{
			"bucket":   []byte(instance.Spec.Storage.ObjectStorageSpec.GCS.Bucket),
			"key.json": accountKeySecret.Data[instance.Spec.Storage.ObjectStorageSpec.GCS.KeyJSON.Key],
		}
	}

	return tempoSecret, nil
}

func uiPlugin() *uiv1alpha1.UIPlugin {
	return &uiv1alpha1.UIPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UIPlugin",
			APIVersion: uiv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "distributed-tracing",
		},
		Spec: uiv1alpha1.UIPluginSpec{
			Type: uiv1alpha1.TypeDistributedTracing,
		},
	}
}

func toTempoStorageType(objStorage obsv1alpha1.ObjectStorage) tempov1alpha1.ObjectStorageSecretType {
	if objStorage.S3 != nil || objStorage.S3STS != nil || objStorage.S3CCO != nil {
		return tempov1alpha1.ObjectStorageSecretS3
	} else if objStorage.Azure != nil || objStorage.AzureWIF != nil {
		return tempov1alpha1.ObjectStorageSecretAzure
	} else if objStorage.GCS != nil {
		return tempov1alpha1.ObjectStorageSecretGCS
	}
	return ""
}

func toTempoCredentialMode(objStorage obsv1alpha1.ObjectStorage) tempov1alpha1.CredentialMode {
	if objStorage.S3 != nil || objStorage.Azure != nil || objStorage.GCS != nil {
		return tempov1alpha1.CredentialModeStatic
	} else if objStorage.S3STS != nil || objStorage.AzureWIF != nil {
		return tempov1alpha1.CredentialModeToken
	} else if objStorage.S3CCO != nil {
		return tempov1alpha1.CredentialModeTokenCCO
	}

	return ""
}
