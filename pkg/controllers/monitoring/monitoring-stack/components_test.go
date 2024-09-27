package monitoringstack

import (
	"testing"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

func TestStorageSpec(t *testing.T) {
	validPVCSpec := &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("200Mi"),
			},
		},
	}

	tt := []struct {
		pvc      *corev1.PersistentVolumeClaimSpec
		expected *monv1.StorageSpec
	}{
		{pvc: nil, expected: nil},
		{pvc: &corev1.PersistentVolumeClaimSpec{}, expected: nil},
		{
			pvc: validPVCSpec,
			expected: &monv1.StorageSpec{
				VolumeClaimTemplate: v1.EmbeddedPersistentVolumeClaim{
					Spec: *validPVCSpec,
				},
			},
		},
	}

	for _, tc := range tt {
		actual := storageForPVC(tc.pvc)
		assert.DeepEqual(t, tc.expected, actual)
	}
}

func TestNewAdditionalScrapeConfigsSecret(t *testing.T) {
	for _, tc := range []struct {
		name       string
		spec       stack.MonitoringStackSpec
		goldenFile string
	}{
		{
			name: "no-tls",
			spec: stack.MonitoringStackSpec{
				PrometheusConfig:   &stack.PrometheusConfig{},
				AlertmanagerConfig: stack.AlertmanagerConfig{},
			},
			goldenFile: "no-tls",
		},
		{
			name: "with-tls",
			spec: stack.MonitoringStackSpec{
				PrometheusConfig: &stack.PrometheusConfig{
					WebTLSConfig: &stack.WebTLSConfig{
						PrivateKey: stack.SecretKeySelector{
							Name: "prometheus-tls",
							Key:  "key.pem",
						},
						Certificate: stack.SecretKeySelector{
							Name: "prometheus-tls",
							Key:  "cert.pem",
						},
						CertificateAuthority: stack.SecretKeySelector{
							Name: "prometheus-tls",
							Key:  "ca.pem",
						},
					},
				},
				AlertmanagerConfig: stack.AlertmanagerConfig{
					WebTLSConfig: &stack.WebTLSConfig{
						PrivateKey: stack.SecretKeySelector{
							Name: "alertmanager-tls",
							Key:  "key.pem",
						},
						Certificate: stack.SecretKeySelector{
							Name: "alertmanager-tls",
							Key:  "cert.pem",
						},
						CertificateAuthority: stack.SecretKeySelector{
							Name: "alertmanager-tls",
							Key:  "ca.pem",
						},
					},
				},
			},
			goldenFile: "tls",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := stack.MonitoringStack{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ms-" + tc.name,
					Namespace: "ns-" + tc.name,
				},
				Spec: tc.spec,
			}
			s := newAdditionalScrapeConfigsSecret(&ms, tc.name)
			assert.Equal(t, s.Name, tc.name)
			golden.Assert(t, s.StringData[AdditionalScrapeConfigsSelfScrapeKey], tc.goldenFile)
		})
	}
}
