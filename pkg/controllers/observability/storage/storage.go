// Package storage materializes object-storage credentials and TLS resources for
// ObservabilityInstaller capabilities.
package storage

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

// Resources are the objects generated from an ObjectStorageSpec.
type Resources struct {
	ObjectStorage    *corev1.Secret
	ObjectStorageTLS *corev1.Secret
	ObjectStorageCA  *corev1.ConfigMap
}

// Names identifies the generated storage resources.
type Names struct {
	Secret      string
	TLSSecret   string
	CAConfigMap string
}

// Format selects the secret key names expected by the consuming operator.
// Tempo and Loki disagree on some keys for the same storage backend.
type Format int

const (
	// TempoFormat uses the key names required by tempo-operator.
	TempoFormat Format = iota
	// LokiFormat uses the key names required by loki-operator.
	LokiFormat
)

// bucketKey is the S3 bucket key name: tempo uses "bucket", loki "bucketnames".
func (f Format) bucketKey() string {
	if f == LokiFormat {
		return "bucketnames"
	}
	return "bucket"
}

// Inventory returns the complete cleanup inventory without reading referenced
// source credentials or TLS material.
func Inventory(namespace string, names Names) []client.Object {
	objects := []client.Object{
		&corev1.Secret{TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: corev1.SchemeGroupVersion.String()}, ObjectMeta: metav1.ObjectMeta{Name: names.Secret, Namespace: namespace}},
	}
	if names.TLSSecret != "" {
		objects = append(objects, &corev1.Secret{TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: corev1.SchemeGroupVersion.String()}, ObjectMeta: metav1.ObjectMeta{Name: names.TLSSecret, Namespace: namespace}})
	}
	return append(objects, &corev1.ConfigMap{TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: corev1.SchemeGroupVersion.String()}, ObjectMeta: metav1.ObjectMeta{Name: names.CAConfigMap, Namespace: namespace}})
}

// addAzureEnvironment adds the Azure cloud environment, which loki-operator
// requires and tempo-operator does not accept. The API has no environment
// field, so the public Azure cloud is assumed.
func addAzureEnvironment(data map[string][]byte, format Format) {
	if format == LokiFormat {
		data["environment"] = []byte("AzureGlobal")
	}
}

// Materialize copies referenced credentials and TLS material into resources
// owned by the operator.
func Materialize(ctx context.Context, reader client.Reader, namespace string, spec *obsv1alpha1.ObjectStorageSpec, names Names, format Format) (*Resources, error) {
	credentials := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: corev1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: names.Secret, Namespace: namespace},
	}
	if spec == nil {
		return &Resources{ObjectStorage: credentials}, nil
	}

	getSecretKey := func(ref obsv1alpha1.SecretKeySelector, description string) ([]byte, error) {
		secret := &corev1.Secret{}
		if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, secret); err != nil {
			return nil, fmt.Errorf("get %s %s: %w", description, ref.Name, err)
		}
		return secret.Data[ref.Key], nil
	}

	var tlsSecret *corev1.Secret
	resources := &Resources{ObjectStorage: credentials}
	if tls := spec.TLS; tls != nil {
		if tls.CAConfigMap != nil {
			configMap := &corev1.ConfigMap{}
			if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: tls.CAConfigMap.Name}, configMap); err != nil {
				return nil, fmt.Errorf("get object storage CA configmap %s: %w", tls.CAConfigMap.Name, err)
			}
			resources.ObjectStorageCA = &corev1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: corev1.SchemeGroupVersion.String()},
				ObjectMeta: metav1.ObjectMeta{Name: names.CAConfigMap, Namespace: namespace},
				Data:       map[string]string{"service-ca.crt": configMap.Data[tls.CAConfigMap.Key]},
			}
		}
		ensureTLSSecret := func() *corev1.Secret {
			if tlsSecret == nil {
				tlsSecret = &corev1.Secret{TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: corev1.SchemeGroupVersion.String()}, ObjectMeta: metav1.ObjectMeta{Name: names.TLSSecret, Namespace: namespace}, Data: map[string][]byte{}}
			}
			return tlsSecret
		}
		if names.TLSSecret != "" {
			if tls.CertSecret != nil {
				cert, err := getSecretKey(*tls.CertSecret, "object storage cert secret")
				if err != nil {
					return nil, err
				}
				ensureTLSSecret().Data["tls.crt"] = cert
			}
			if tls.KeySecret != nil {
				key, err := getSecretKey(*tls.KeySecret, "object storage key secret")
				if err != nil {
					return nil, err
				}
				ensureTLSSecret().Data["tls.key"] = key
			}
		}
	}
	resources.ObjectStorageTLS = tlsSecret

	switch {
	case spec.S3 != nil:
		accessKey, err := getSecretKey(spec.S3.AccessKeySecret, "S3 access key secret")
		if err != nil {
			return nil, err
		}
		credentials.Data = map[string][]byte{"access_key_id": []byte(spec.S3.AccessKeyID), "access_key_secret": accessKey, format.bucketKey(): []byte(spec.S3.Bucket), "endpoint": []byte(spec.S3.Endpoint)}
		if spec.S3.Region != "" {
			credentials.Data["region"] = []byte(spec.S3.Region)
		}
	case spec.S3STS != nil:
		credentials.Data = map[string][]byte{format.bucketKey(): []byte(spec.S3STS.Bucket), "role_arn": []byte(spec.S3STS.RoleARN), "region": []byte(spec.S3STS.Region)}
	case spec.S3CCO != nil:
		credentials.Data = map[string][]byte{format.bucketKey(): []byte(spec.S3CCO.Bucket), "region": []byte(spec.S3CCO.Region)}
	case spec.Azure != nil:
		key, err := getSecretKey(spec.Azure.AccountKeySecret, "Azure account key secret")
		if err != nil {
			return nil, err
		}
		credentials.Data = map[string][]byte{"container": []byte(spec.Azure.Container), "account_name": []byte(spec.Azure.AccountName), "account_key": key}
		addAzureEnvironment(credentials.Data, format)
	case spec.AzureWIF != nil:
		credentials.Data = map[string][]byte{"container": []byte(spec.AzureWIF.Container), "account_name": []byte(spec.AzureWIF.AccountName), "client_id": []byte(spec.AzureWIF.ClientID), "tenant_id": []byte(spec.AzureWIF.TenantID)}
		// loki-operator rejects a workload identity secret without a
		// subscription ID, tempo-operator does not accept the key.
		if format == LokiFormat {
			credentials.Data["subscription_id"] = []byte(spec.AzureWIF.SubscriptionID)
		}
		if spec.AzureWIF.Audience != "" {
			credentials.Data["audience"] = []byte(spec.AzureWIF.Audience)
		}
		addAzureEnvironment(credentials.Data, format)
	case spec.GCS != nil:
		key, err := getSecretKey(spec.GCS.KeyJSONSecret, "GCS keyJSON secret")
		if err != nil {
			return nil, err
		}
		credentials.Data = map[string][]byte{"bucketname": []byte(spec.GCS.Bucket), "key.json": key}
	case spec.GCSWIF != nil:
		key, err := getSecretKey(spec.GCSWIF.KeyJSONSecret, "GCSWIF keyJSON secret")
		if err != nil {
			return nil, err
		}
		credentials.Data = map[string][]byte{"bucketname": []byte(spec.GCSWIF.Bucket), "key.json": key}
		if spec.GCSWIF.Audience != "" {
			credentials.Data["audience"] = []byte(spec.GCSWIF.Audience)
		}
	}
	return resources, nil
}
