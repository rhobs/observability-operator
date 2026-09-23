package generator

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestFallbackReaderStringData verifies that a Secret provided with stringData
// is served to readers with its values under data, matching what the API
// server would return for the same secret, so object storage secrets are
// populated offline.
func TestFallbackReaderStringData(t *testing.T) {
	r := NewFallbackReader(nil, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "minio-secret", Namespace: "sample"},
		StringData: map[string]string{"access_key_secret": "supersecret"},
	})

	got := &corev1.Secret{}
	err := r.Get(context.Background(), client.ObjectKey{Name: "minio-secret", Namespace: "sample"}, got)
	assert.NilError(t, err)
	assert.Equal(t, string(got.Data["access_key_secret"]), "supersecret")
	assert.Equal(t, len(got.StringData), 0)
}

// TestFallbackReaderData verifies that a Secret provided with base64 data is
// served unchanged.
func TestFallbackReaderData(t *testing.T) {
	r := NewFallbackReader(nil, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "minio-secret", Namespace: "sample"},
		Data:       map[string][]byte{"access_key_secret": []byte("supersecret")},
	})

	got := &corev1.Secret{}
	err := r.Get(context.Background(), client.ObjectKey{Name: "minio-secret", Namespace: "sample"}, got)
	assert.NilError(t, err)
	assert.Equal(t, string(got.Data["access_key_secret"]), "supersecret")
}

func TestFallbackReaderGetNotFound(t *testing.T) {
	r := NewFallbackReader(nil)
	got := &corev1.Secret{}
	err := r.Get(context.Background(), client.ObjectKey{Name: "missing", Namespace: "ns"}, got)
	assert.ErrorContains(t, err, "not found")
}

func TestFallbackReaderGetFallback(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NilError(t, corev1.AddToScheme(scheme))

	clusterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-secret", Namespace: "ns"},
		Data:       map[string][]byte{"key": []byte("from-cluster")},
	}
	fakeClient := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(clusterSecret).Build()

	r := NewFallbackReader(fakeClient, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "local-secret", Namespace: "ns"},
		Data:       map[string][]byte{"key": []byte("from-preload")},
	})

	// Preloaded secret is returned directly.
	got := &corev1.Secret{}
	assert.NilError(t, r.Get(context.Background(), client.ObjectKey{Name: "local-secret", Namespace: "ns"}, got))
	assert.Equal(t, string(got.Data["key"]), "from-preload")

	// Non-preloaded secret falls back to the underlying reader.
	got = &corev1.Secret{}
	assert.NilError(t, r.Get(context.Background(), client.ObjectKey{Name: "cluster-secret", Namespace: "ns"}, got))
	assert.Equal(t, string(got.Data["key"]), "from-cluster")
}

func TestFallbackReaderListDelegates(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NilError(t, corev1.AddToScheme(scheme))

	s1 := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns"}}
	fakeClient := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(s1).Build()

	r := NewFallbackReader(fakeClient)
	list := &corev1.SecretList{}
	assert.NilError(t, r.List(context.Background(), list, client.InNamespace("ns")))
	assert.Equal(t, len(list.Items), 1)
}

func TestFallbackReaderListNilReader(t *testing.T) {
	r := NewFallbackReader(nil)
	list := &corev1.SecretList{}
	assert.NilError(t, r.List(context.Background(), list))
	assert.Equal(t, len(list.Items), 0)
}
