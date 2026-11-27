package thanos_querier

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	msoapi "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

func TestValidateWebTLSConfig(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))

	tlsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "tls", Namespace: "test-ns"},
		Data: map[string][]byte{
			"tls.crt": []byte("cert"),
			"tls.key": []byte("key"),
			"ca.crt":  []byte("ca"),
			"empty":   {},
		},
	}

	selector := func(key string) msoapi.SecretKeySelector {
		return msoapi.SecretKeySelector{Name: "tls", Key: key}
	}

	for _, tc := range []struct {
		name    string
		tls     *msoapi.WebTLSConfig
		objects []k8sruntime.Object
		// errs lists substrings expected in the aggregated error.
		errs []string
	}{
		{
			name:    "no TLS configuration",
			tls:     nil,
			objects: nil,
		},
		{
			name: "all references valid",
			tls: &msoapi.WebTLSConfig{
				Certificate:          selector("tls.crt"),
				PrivateKey:           selector("tls.key"),
				CertificateAuthority: selector("ca.crt"),
			},
			objects: []k8sruntime.Object{tlsSecret},
		},
		{
			name: "secret not found",
			tls: &msoapi.WebTLSConfig{
				Certificate:          selector("tls.crt"),
				PrivateKey:           selector("tls.key"),
				CertificateAuthority: selector("ca.crt"),
			},
			objects: nil,
			errs: []string{
				`spec.webTLSConfig.certificate: secret "tls" not found`,
				`spec.webTLSConfig.privateKey: secret "tls" not found`,
				`spec.webTLSConfig.certificateAuthority: secret "tls" not found`,
			},
		},
		{
			name: "secret in another namespace",
			tls: &msoapi.WebTLSConfig{
				Certificate:          selector("tls.crt"),
				PrivateKey:           selector("tls.key"),
				CertificateAuthority: selector("ca.crt"),
			},
			objects: []k8sruntime.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "tls", Namespace: "other-ns"},
				Data:       tlsSecret.Data,
			}},
			errs: []string{`spec.webTLSConfig.certificate: secret "tls" not found`},
		},
		{
			name: "missing key",
			tls: &msoapi.WebTLSConfig{
				Certificate:          selector("tls.crt"),
				PrivateKey:           selector("missing.key"),
				CertificateAuthority: selector("ca.crt"),
			},
			objects: []k8sruntime.Object{tlsSecret},
			errs:    []string{`spec.webTLSConfig.privateKey: secret "tls" has no "missing.key" key`},
		},
		{
			name: "empty key",
			tls: &msoapi.WebTLSConfig{
				Certificate:          selector("empty"),
				PrivateKey:           selector("tls.key"),
				CertificateAuthority: selector("ca.crt"),
			},
			objects: []k8sruntime.Object{tlsSecret},
			errs:    []string{`spec.webTLSConfig.certificate: key "empty" of secret "tls" is empty`},
		},
		{
			name: "empty reference",
			tls: &msoapi.WebTLSConfig{
				Certificate:          msoapi.SecretKeySelector{},
				PrivateKey:           msoapi.SecretKeySelector{Name: "tls"},
				CertificateAuthority: selector("ca.crt"),
			},
			objects: []k8sruntime.Object{tlsSecret},
			errs: []string{
				"spec.webTLSConfig.certificate: both name and key are required",
				"spec.webTLSConfig.privateKey: both name and key are required",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rm := resourceManager{
				apiReader: fake.NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(tc.objects...).
					Build(),
			}
			querier := &msoapi.ThanosQuerier{
				ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "test-ns"},
				Spec:       msoapi.ThanosQuerierSpec{WebTLSConfig: tc.tls},
			}

			err := rm.validateWebTLSConfig(context.Background(), querier)
			if len(tc.errs) == 0 {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			// The reconciler relies on the error type to tell a user
			// mistake apart from a transient API failure.
			_, ok := errors.AsType[invalidWebTLSConfigError](err)
			require.True(t, ok, "expected an invalidWebTLSConfigError, got %T", err)
			for _, want := range tc.errs {
				require.ErrorContains(t, err, want)
			}
		})
	}
}
