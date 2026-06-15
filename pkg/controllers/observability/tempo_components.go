package observability

import (
	"context"
	"encoding/json"
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
	tenantID   = "1610b0c3-c509-4592-a256-a1871353dbfb"

	tempoStackComponent = "../components/stores/tempostack/resources"
	consoleComponent    = "../components/console/resources"

	tempoStackBaseName = "tempostack"
)

func addTempoStack(overlay *Overlay, instance *obsv1alpha1.ObservabilityInstaller) error {
	overlay.AddComponent(tempoStackComponent)

	patch := tempoStackPatch(instance)
	patchYAML, err := marshalYAML(patch)
	if err != nil {
		return fmt.Errorf("marshaling TempoStack patch: %w", err)
	}
	overlay.AddPatch("patches/tempostack.yaml", patchYAML)
	return nil
}

func addUIPlugin(overlay *Overlay) {
	overlay.AddComponent(consoleComponent)
}

func addTempoSecrets(ctx context.Context, overlay *Overlay, k8sClient client.Client, k8sReader client.Reader, instance *obsv1alpha1.ObservabilityInstaller) error {
	secrets, err := tempoStackSecrets(ctx, k8sClient, k8sReader, *instance)
	if err != nil {
		return err
	}

	if secrets.objectStorage != nil {
		yaml, err := marshalYAML(secrets.objectStorage)
		if err != nil {
			return fmt.Errorf("marshaling tempo secret: %w", err)
		}
		overlay.AddResource("resources/tempo-secret.yaml", yaml)
	}
	if secrets.objectStorageTLSSecret != nil {
		yaml, err := marshalYAML(secrets.objectStorageTLSSecret)
		if err != nil {
			return fmt.Errorf("marshaling tempo TLS secret: %w", err)
		}
		overlay.AddResource("resources/tempo-tls-secret.yaml", yaml)
	}
	if secrets.objectStorageCAConfigMap != nil {
		yaml, err := marshalYAML(secrets.objectStorageCAConfigMap)
		if err != nil {
			return fmt.Errorf("marshaling tempo CA configmap: %w", err)
		}
		overlay.AddResource("resources/tempo-ca-configmap.yaml", yaml)
	}
	return nil
}

// tempoStackPatch returns a strategic merge patch for the base TempoStack manifest.
// It sets only dynamic fields; hard-coded values are in the base manifest.
func tempoStackPatch(instance *obsv1alpha1.ObservabilityInstaller) *tempov1alpha1.TempoStack {
	patch := &tempov1alpha1.TempoStack{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TempoStack",
			APIVersion: tempov1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: tempoStackBaseName,
		},
		Spec: tempov1alpha1.TempoStackSpec{
			Storage: tempov1alpha1.ObjectStorageSpec{
				Secret: tempov1alpha1.ObjectStorageSecretSpec{
					Name: tempoSecretName(instance.Name),
				},
			},
		},
	}

	if oss := instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec(); oss != nil {
		patch.Spec.Storage.Secret.Type = toTempoStorageType(oss)
		patch.Spec.Storage.Secret.CredentialMode = toTempoCredentialMode(oss)

		tls := oss.GetTLS()
		enableTLS := tls != nil || s3hasHTTPSEndpoint(*oss)

		if enableTLS {
			patch.Spec.Storage.TLS = tempov1alpha1.TLSSpec{
				Enabled: true,
			}
			if tls != nil {
				if tls.CAConfigMap != nil {
					patch.Spec.Storage.TLS.CA = tempoStorageCAConfigMapName(instance.Name)
				}
				if tls.CertSecret != nil {
					patch.Spec.Storage.TLS.Cert = tempoStorageSecretName(instance.Name)
				}
				if tls.MinVersion != "" {
					patch.Spec.Storage.TLS.MinVersion = tls.MinVersion
				}
			}
		}
	}

	return patch
}

func tempoName(instance string) string {
	return instance
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
	objectStorage            *corev1.Secret
	objectStorageTLSSecret   *corev1.Secret
	objectStorageCAConfigMap *corev1.ConfigMap
}

func tempoStackSecrets(ctx context.Context, k8sClient client.Client, k8sReader client.Reader, instance obsv1alpha1.ObservabilityInstaller) (*tempoSecrets, error) {
	var objectStorageCAConfMap *corev1.ConfigMap
	var objectStorageTLSSecret *corev1.Secret

	if tlsSpec := instance.Spec.GetCapabilities().GetTracing().GetStorage().GetObjectStorageSpec().GetTLS(); tlsSpec != nil {
		if tlsSpec.CAConfigMap != nil {
			caConfigMap := &corev1.ConfigMap{}
			err := k8sReader.Get(ctx, client.ObjectKey{
				Namespace: instance.Namespace,
				Name:      tlsSpec.CAConfigMap.Name,
			}, caConfigMap)
			if err != nil {
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
			err := k8sReader.Get(ctx, client.ObjectKey{
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
					Name:      tempoStorageSecretName(instance.Name),
					Namespace: instance.Namespace,
				},
			}
			objectStorageTLSSecret.Data = map[string][]byte{}
			objectStorageTLSSecret.Data["tls.crt"] = certSecret.Data[tlsSpec.CertSecret.Key]
		}
		if tlsSpec.KeySecret != nil {
			certSecret := &corev1.Secret{}
			err := k8sReader.Get(ctx, client.ObjectKey{
				Namespace: instance.Namespace,
				Name:      tlsSpec.KeySecret.Name,
			}, certSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to get object storage cert secret %s: %w", tlsSpec.KeySecret.Name, err)
			}

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
				"region":   []byte(objectStorageSpec.S3STS.Region),
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
		} else if objectStorageSpec.GCSWIF != nil {
			keyJSONSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, client.ObjectKey{
				Namespace: instance.Namespace,
				Name:      objectStorageSpec.GCSWIF.KeyJSONSecret.Name,
			}, keyJSONSecret)
			if err != nil {
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

// marshalYAML marshals a Kubernetes object to YAML via JSON tags.
func marshalYAML(obj interface{}) ([]byte, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return jsonToYAML(data)
}
