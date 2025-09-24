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
	tenantName = "application"
	tenantID   = "1610b0c3-c509-4592-a256-a1871353dbfb"
)

func tempoStack(instance *obsv1alpha1.ObservabilityInstaller) *tempov1alpha1.TempoStack {
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
					Type:           toTempoStorageType(instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec),
					CredentialMode: toTempoCredentialMode(instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec),
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

	if instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec.TLS != nil {
		tempo.Spec.Storage.TLS = tempov1alpha1.TLSSpec{
			Enabled:    true,
			CA:         tempoStorageCAConfigMapName(instance.Name),
			Cert:       tempoStorageSecretName(instance.Name),
			MinVersion: instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec.TLS.MinVersion,
		}
	}

	return tempo
}

func tempoName(instance string) string {
	return fmt.Sprintf("%s", instance)
}

func tempoStorageCAConfigMapName(name string) string {
	return fmt.Sprintf("coo-%s-tempo-storage-ca", name)
}

func tempoStorageSecretName(name string) string {
	return fmt.Sprintf("coo-%s-tempo-storage-cert", name)
}

func tempoSecretName(name string) string {
	return fmt.Sprintf("coo-%s-tempo", name)
}

type tempoSecrets struct {
	objectStorage *corev1.Secret

	objectStorageTLSSecret   *corev1.Secret
	objectStorageCAConfigMap *corev1.ConfigMap
}

func tempoStackSecrets(ctx context.Context, k8sClient client.Client, instance obsv1alpha1.ObservabilityInstaller) (*tempoSecrets, error) {
	var objectStorageCAConfMap *corev1.ConfigMap
	var objectStorageTLSSecret *corev1.Secret

	if instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec.TLS != nil {
		tlsSpec := instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec.TLS
		if tlsSpec.CAConfigMap != nil {
			caConfigMap := &corev1.ConfigMap{}
			err := k8sClient.Get(ctx, client.ObjectKey{
				Namespace: instance.Namespace,
				Name:      tlsSpec.CAConfigMap.Name,
			}, caConfigMap)
			if err != nil {
				return nil, fmt.Errorf("failed to get object storage CA configmap %s: %w", instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec.TLS.CAConfigMap.Name, err)
			}

			objectStorageCAConfMap = &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      tempoStorageCAConfigMapName(instance.Name),
					Namespace: instance.Namespace,
				},
				Data: map[string]string{
					"service-ca.crt": caConfigMap.Data[tlsSpec.CAConfigMap.Key],
				},
			}
		}

		if tlsSpec.CertSecret != nil {
			certSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, client.ObjectKey{
				Namespace: instance.Namespace,
				Name:      tlsSpec.CertSecret.Name,
			}, certSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to get object storage cert secret %s: %w", tlsSpec.CertSecret.Name, err)
			}

			objectStorageTLSSecret = &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      tempoSecretName(instance.Name),
					Namespace: instance.Namespace,
				},
			}
			objectStorageTLSSecret.Data["tls.crt"] = certSecret.Data[tlsSpec.CertSecret.Key]
		}
		if tlsSpec.KeySecret != nil {
			certSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, client.ObjectKey{
				Namespace: instance.Namespace,
				Name:      tlsSpec.KeySecret.Name,
			}, certSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to get object storage cert secret %s: %w", tlsSpec.KeySecret.Name, err)
			}

			objectStorageTLSSecret.Data["tls.key"] = certSecret.Data[tlsSpec.KeySecret.Key]
		}
	}

	tempoSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoSecretName(instance.Name),
			Namespace: instance.Namespace,
		},
	}
	objectStorageSpec := instance.Spec.Capabilities.Tracing.Storage.ObjectStorageSpec
	if objectStorageSpec.S3 != nil {
		accessKeySecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      objectStorageSpec.S3.AccessKeySecret.Name,
		}, accessKeySecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get S3 access key secret %s: %w", objectStorageSpec.S3.AccessKeySecret.Name, err)
		}

		tempoSecret.Data = map[string][]byte{
			"bucket":            []byte(objectStorageSpec.S3.Bucket),
			"endpoint":          []byte(objectStorageSpec.S3.Endpoint),
			"access_key_id":     []byte(objectStorageSpec.S3.AccessKeyID),
			"access_key_secret": accessKeySecret.Data[objectStorageSpec.S3.AccessKeySecret.Key],
		}

		if objectStorageSpec.S3.Region != "" {
			tempoSecret.Data["region"] = []byte(objectStorageSpec.S3.Region)
		}
	} else if objectStorageSpec.S3STS != nil {
		tempoSecret.Data = map[string][]byte{
			"bucket":   []byte(objectStorageSpec.S3STS.Bucket),
			"role_arn": []byte(objectStorageSpec.S3STS.RoleARN),
			"region":   []byte(objectStorageSpec.S3STS.RoleARN),
		}
	} else if objectStorageSpec.S3CCO != nil {
		tempoSecret.Data = map[string][]byte{
			"bucket": []byte(objectStorageSpec.S3CCO.Bucket),
			"region": []byte(objectStorageSpec.S3CCO.Region),
		}
	} else if objectStorageSpec.Azure != nil {
		accountKeySecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      objectStorageSpec.Azure.AccountKeySecret.Name,
		}, accountKeySecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get Azure account key secret %s: %w", objectStorageSpec.Azure.AccountKeySecret.Name, err)
		}

		tempoSecret.Data = map[string][]byte{
			"container":    []byte(objectStorageSpec.Azure.Container),
			"account_name": []byte(objectStorageSpec.Azure.AccountName),
			"account_key":  accountKeySecret.Data[objectStorageSpec.Azure.AccountKeySecret.Key],
		}
	} else if objectStorageSpec.AzureWIF != nil {
		tempoSecret.Data = map[string][]byte{
			"container":    []byte(objectStorageSpec.AzureWIF.Container),
			"account_name": []byte(objectStorageSpec.AzureWIF.AccountName),
			"client_id":    []byte(objectStorageSpec.AzureWIF.ClientID),
			"tenant_id":    []byte(objectStorageSpec.AzureWIF.TenantID),
		}
		if objectStorageSpec.AzureWIF.Audience != "" {
			tempoSecret.Data["audience"] = []byte(objectStorageSpec.AzureWIF.Audience)
		}
	} else if objectStorageSpec.GCS != nil {
		keyJSONSecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      objectStorageSpec.GCS.KeyJSONSecret.Name,
		}, keyJSONSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get GCS keyJSON secret %s: %w", objectStorageSpec.GCS.KeyJSONSecret.Name, err)
		}

		tempoSecret.Data = map[string][]byte{
			"bucketname": []byte(objectStorageSpec.GCS.Bucket),
			"key.json":   keyJSONSecret.Data[objectStorageSpec.GCS.KeyJSONSecret.Key],
		}
	} else if objectStorageSpec.GCSSTSSpec != nil {
		keyJSONSecret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: instance.Namespace,
			Name:      objectStorageSpec.GCSSTSSpec.KeyJSONSecret.Name,
		}, keyJSONSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to get GCSSTS keyJSON secret %s: %w", objectStorageSpec.GCSSTSSpec.KeyJSONSecret.Name, err)
		}

		tempoSecret.Data = map[string][]byte{
			"bucketname": []byte(objectStorageSpec.GCSSTSSpec.Bucket),
			"key.json":   keyJSONSecret.Data[objectStorageSpec.GCSSTSSpec.KeyJSONSecret.Key],
		}
		if objectStorageSpec.GCSSTSSpec.Audience != "" {
			tempoSecret.Data["audience"] = []byte(objectStorageSpec.GCSSTSSpec.Audience)
		}
	}

	return &tempoSecrets{
		objectStorage:            tempoSecret,
		objectStorageTLSSecret:   objectStorageTLSSecret,
		objectStorageCAConfigMap: objectStorageCAConfMap,
	}, nil
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

func toTempoStorageType(objStorage obsv1alpha1.TracingObjectStorageSpec) tempov1alpha1.ObjectStorageSecretType {
	if objStorage.S3 != nil || objStorage.S3STS != nil || objStorage.S3CCO != nil {
		return tempov1alpha1.ObjectStorageSecretS3
	} else if objStorage.Azure != nil || objStorage.AzureWIF != nil {
		return tempov1alpha1.ObjectStorageSecretAzure
	} else if objStorage.GCS != nil || objStorage.GCSSTSSpec != nil {
		return tempov1alpha1.ObjectStorageSecretGCS
	}
	return ""
}

func toTempoCredentialMode(objStorage obsv1alpha1.TracingObjectStorageSpec) tempov1alpha1.CredentialMode {
	if objStorage.S3 != nil || objStorage.Azure != nil || objStorage.GCS != nil {
		return tempov1alpha1.CredentialModeStatic
	} else if objStorage.S3STS != nil || objStorage.AzureWIF != nil || objStorage.GCSSTSSpec != nil {
		return tempov1alpha1.CredentialModeToken
	} else if objStorage.S3CCO != nil {
		return tempov1alpha1.CredentialModeTokenCCO
	}

	return ""
}
