package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestInventory(t *testing.T) {
	objects := Inventory("test", Names{Secret: "credentials", TLSSecret: "tls", CAConfigMap: "ca"})
	require.Len(t, objects, 3)
	require.Equal(t, "credentials", objects[0].GetName())
	require.Equal(t, "tls", objects[1].GetName())
	require.Equal(t, "ca", objects[2].GetName())

	objects = Inventory("test", Names{Secret: "credentials", CAConfigMap: "ca"})
	require.Len(t, objects, 2)
	require.Equal(t, "credentials", objects[0].GetName())
	require.Equal(t, "ca", objects[1].GetName())
}

func TestMaterializeOmitsUnconfiguredTLSSecret(t *testing.T) {
	resources, err := Materialize(context.Background(), fake.NewClientBuilder().Build(), "test", &obsv1alpha1.ObjectStorageSpec{
		TLS: &obsv1alpha1.TLSSpec{
			CertSecret: &obsv1alpha1.SecretKeySelector{Name: "missing", Key: "tls.crt"},
			KeySecret:  &obsv1alpha1.SecretKeySelector{Name: "missing", Key: "tls.key"},
		},
	}, Names{Secret: "credentials", CAConfigMap: "ca"}, LokiFormat)
	require.NoError(t, err)
	require.Nil(t, resources.ObjectStorageTLS)
}

func TestMaterializeS3(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: "test"},
		Data:       map[string][]byte{"key": []byte("secret")},
	}).Build()

	resources, err := Materialize(context.Background(), reader, "test", &obsv1alpha1.ObjectStorageSpec{
		S3: &obsv1alpha1.S3Spec{
			Bucket: "bucket", Endpoint: "https://s3.example", AccessKeyID: "id",
			AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "source", Key: "key"},
		},
	}, Names{Secret: "target"}, TempoFormat)
	require.NoError(t, err)
	require.Equal(t, "target", resources.ObjectStorage.Name)
	require.Equal(t, []byte("bucket"), resources.ObjectStorage.Data["bucket"])
	require.Equal(t, []byte("secret"), resources.ObjectStorage.Data["access_key_secret"])
}

func TestMaterializeFormatKeys(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: "test"},
		Data:       map[string][]byte{"key": []byte("secret")},
	}).Build()

	s3 := &obsv1alpha1.ObjectStorageSpec{S3: &obsv1alpha1.S3Spec{
		Bucket: "bucket", Endpoint: "https://s3.example", AccessKeyID: "id",
		AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "source", Key: "key"},
	}}
	azure := &obsv1alpha1.ObjectStorageSpec{Azure: &obsv1alpha1.AzureSpec{
		Container: "container", AccountName: "account",
		AccountKeySecret: obsv1alpha1.SecretKeySelector{Name: "source", Key: "key"},
	}}

	// Loki and Tempo name the same fields differently.
	loki, err := Materialize(context.Background(), reader, "test", s3, Names{Secret: "target"}, LokiFormat)
	require.NoError(t, err)
	require.Equal(t, []byte("bucket"), loki.ObjectStorage.Data["bucketnames"])
	require.NotContains(t, loki.ObjectStorage.Data, "bucket")

	tempo, err := Materialize(context.Background(), reader, "test", s3, Names{Secret: "target"}, TempoFormat)
	require.NoError(t, err)
	require.NotContains(t, tempo.ObjectStorage.Data, "bucketnames")

	// Loki requires an Azure cloud environment, Tempo rejects it.
	loki, err = Materialize(context.Background(), reader, "test", azure, Names{Secret: "target"}, LokiFormat)
	require.NoError(t, err)
	require.Equal(t, []byte("AzureGlobal"), loki.ObjectStorage.Data["environment"])

	tempo, err = Materialize(context.Background(), reader, "test", azure, Names{Secret: "target"}, TempoFormat)
	require.NoError(t, err)
	require.NotContains(t, tempo.ObjectStorage.Data, "environment")

	// Loki requires an Azure subscription ID for workload identity, Tempo rejects it.
	azureWIF := &obsv1alpha1.ObjectStorageSpec{AzureWIF: &obsv1alpha1.AzureWIFSpec{
		Container: "container", AccountName: "account", ClientID: "client", TenantID: "tenant", SubscriptionID: "subscription",
	}}
	loki, err = Materialize(context.Background(), reader, "test", azureWIF, Names{Secret: "target"}, LokiFormat)
	require.NoError(t, err)
	require.Equal(t, []byte("subscription"), loki.ObjectStorage.Data["subscription_id"])

	tempo, err = Materialize(context.Background(), reader, "test", azureWIF, Names{Secret: "target"}, TempoFormat)
	require.NoError(t, err)
	require.NotContains(t, tempo.ObjectStorage.Data, "subscription_id")
}
