package observability

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

const (
	tenantName = "application"
)

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

func s3hasHTTPSEndpoint(storageSpec obsv1alpha1.TracingObjectStorageSpec) bool {
	if storageSpec.S3 == nil {
		return false
	}
	endpoint := strings.TrimSpace(storageSpec.S3.Endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "https")
}

type tempoSecrets struct {
	objectStorage *corev1.Secret

	objectStorageTLSSecret   *corev1.Secret
	objectStorageCAConfigMap *corev1.ConfigMap
}

func tempoStackSecrets(ctx context.Context, k8sReader client.Reader, instance obsv1alpha1.ObservabilityInstaller) (*tempoSecrets, error) {
	var objectStorageCAConfMap *corev1.ConfigMap
	var objectStorageTLSSecret *corev1.Secret

	if tlsSpec := instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec().GetTLS(); tlsSpec != nil {
		if tlsSpec.CAConfigMap != nil {
			caConfigMap := &corev1.ConfigMap{}
			key := client.ObjectKey{Namespace: instance.Namespace, Name: tlsSpec.CAConfigMap.Name}
			if err := k8sReader.Get(ctx, key, caConfigMap); err != nil {
				return nil, fmt.Errorf("failed to get object storage CA configmap %s: %w", tlsSpec.CAConfigMap.Name, err)
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
			key := client.ObjectKey{Namespace: instance.Namespace, Name: tlsSpec.CertSecret.Name}
			if err := k8sReader.Get(ctx, key, certSecret); err != nil {
				return nil, fmt.Errorf("failed to get object storage cert secret %s: %w", tlsSpec.CertSecret.Name, err)
			}

			objectStorageTLSSecret = &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      tempoStorageSecretName(instance.Name),
					Namespace: instance.Namespace,
				},
			}
			objectStorageTLSSecret.Data = map[string][]byte{}
			objectStorageTLSSecret.Data["tls.crt"] = certSecret.Data[tlsSpec.CertSecret.Key]
		}
		if tlsSpec.KeySecret != nil {
			certSecret := &corev1.Secret{}
			key := client.ObjectKey{Namespace: instance.Namespace, Name: tlsSpec.KeySecret.Name}
			if err := k8sReader.Get(ctx, key, certSecret); err != nil {
				return nil, fmt.Errorf("failed to get object storage key secret %s: %w", tlsSpec.KeySecret.Name, err)
			}

			// Set only if the cert was found, which initialized the secret
			if objectStorageTLSSecret != nil {
				objectStorageTLSSecret.Data["tls.key"] = certSecret.Data[tlsSpec.KeySecret.Key]
			}
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
	if objectStorageSpec := instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec(); objectStorageSpec != nil {
		if objectStorageSpec.S3 != nil {
			accessKeySecret := &corev1.Secret{}
			key := client.ObjectKey{Namespace: instance.Namespace, Name: objectStorageSpec.S3.AccessKeySecret.Name}
			if err := k8sReader.Get(ctx, key, accessKeySecret); err != nil {
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
				"region":   []byte(objectStorageSpec.S3STS.Region),
			}
		} else if objectStorageSpec.S3CCO != nil {
			tempoSecret.Data = map[string][]byte{
				"bucket": []byte(objectStorageSpec.S3CCO.Bucket),
				"region": []byte(objectStorageSpec.S3CCO.Region),
			}
		} else if objectStorageSpec.Azure != nil {
			accountKeySecret := &corev1.Secret{}
			key := client.ObjectKey{Namespace: instance.Namespace, Name: objectStorageSpec.Azure.AccountKeySecret.Name}
			if err := k8sReader.Get(ctx, key, accountKeySecret); err != nil {
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
			key := client.ObjectKey{Namespace: instance.Namespace, Name: objectStorageSpec.GCS.KeyJSONSecret.Name}
			if err := k8sReader.Get(ctx, key, keyJSONSecret); err != nil {
				return nil, fmt.Errorf("failed to get GCS keyJSON secret %s: %w", objectStorageSpec.GCS.KeyJSONSecret.Name, err)
			}

			tempoSecret.Data = map[string][]byte{
				"bucketname": []byte(objectStorageSpec.GCS.Bucket),
				"key.json":   keyJSONSecret.Data[objectStorageSpec.GCS.KeyJSONSecret.Key],
			}
		} else if objectStorageSpec.GCSWIF != nil {
			keyJSONSecret := &corev1.Secret{}
			key := client.ObjectKey{Namespace: instance.Namespace, Name: objectStorageSpec.GCSWIF.KeyJSONSecret.Name}
			if err := k8sReader.Get(ctx, key, keyJSONSecret); err != nil {
				return nil, fmt.Errorf("failed to get GCSWIF keyJSON secret %s: %w", objectStorageSpec.GCSWIF.KeyJSONSecret.Name, err)
			}

			tempoSecret.Data = map[string][]byte{
				"bucketname": []byte(objectStorageSpec.GCSWIF.Bucket),
				"key.json":   keyJSONSecret.Data[objectStorageSpec.GCSWIF.KeyJSONSecret.Key],
			}
			if objectStorageSpec.GCSWIF.Audience != "" {
				tempoSecret.Data["audience"] = []byte(objectStorageSpec.GCSWIF.Audience)
			}
		}
	}

	return &tempoSecrets{
		objectStorage:            tempoSecret,
		objectStorageTLSSecret:   objectStorageTLSSecret,
		objectStorageCAConfigMap: objectStorageCAConfMap,
	}, nil
}

func BuildTempoSecrets(ctx context.Context, k8sReader client.Reader, instance obsv1alpha1.ObservabilityInstaller) ([]client.Object, error) {
	secrets, err := tempoStackSecrets(ctx, k8sReader, instance)
	if err != nil {
		return nil, err
	}
	var objects []client.Object
	if secrets.objectStorage != nil {
		objects = append(objects, secrets.objectStorage)
	}
	if secrets.objectStorageTLSSecret != nil {
		objects = append(objects, secrets.objectStorageTLSSecret)
	}
	if secrets.objectStorageCAConfigMap != nil {
		objects = append(objects, secrets.objectStorageCAConfigMap)
	}
	return objects, nil
}

func toTempoStorageType(objStorage *obsv1alpha1.TracingObjectStorageSpec) tempov1alpha1.ObjectStorageSecretType {
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

func toTempoCredentialMode(objStorage *obsv1alpha1.TracingObjectStorageSpec) tempov1alpha1.CredentialMode {
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
