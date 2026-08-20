package generator

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gotest.tools/v3/assert"
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
