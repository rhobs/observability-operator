package monitoringstack

import (
	"strings"
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

func TestNewAlertmanagerSetsResources(t *testing.T) {
	amResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
	ms := &stack.MonitoringStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
		Spec: stack.MonitoringStackSpec{
			AlertmanagerConfig: stack.AlertmanagerConfig{
				Resources: amResources,
			},
		},
	}

	am := newAlertmanager(ms, "test-alertmanager", AlertmanagerConfiguration{})
	assert.DeepEqual(t, amResources, am.Spec.Resources)
}

func TestNewPrometheusSetsThanosSidecarResources(t *testing.T) {
	promResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	thanosResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
	ms := &stack.MonitoringStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
		Spec: stack.MonitoringStackSpec{
			Resources: promResources,
			PrometheusConfig: &stack.PrometheusConfig{
				ThanosResources: thanosResources,
			},
			AlertmanagerConfig: stack.AlertmanagerConfig{Disabled: true},
		},
	}

	prom := newPrometheus(ms, "test-prometheus", "test-scrape",
		ThanosConfiguration{Image: "thanos:latest"},
		PrometheusConfiguration{})

	assert.DeepEqual(t, thanosResources, prom.Spec.Thanos.Resources)
	assert.DeepEqual(t, promResources, prom.Spec.Resources)
}

func TestWebTLSConfigSetsMinAndMaxVersion(t *testing.T) {
	tls := &stack.WebTLSConfig{
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
		MinVersion: "TLS13",
		MaxVersion: "TLS13",
	}

	ms := &stack.MonitoringStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
		Spec: stack.MonitoringStackSpec{
			PrometheusConfig: &stack.PrometheusConfig{
				WebTLSConfig: tls,
			},
			AlertmanagerConfig: stack.AlertmanagerConfig{
				WebTLSConfig: tls,
			},
		},
	}

	prom := newPrometheus(ms, "test-prometheus", "test-scrape",
		ThanosConfiguration{}, PrometheusConfiguration{})
	assert.Assert(t, prom.Spec.Web != nil)
	assert.Assert(t, prom.Spec.Web.TLSConfig != nil)
	assert.Equal(t, "TLS13", *prom.Spec.Web.TLSConfig.MinVersion)
	assert.Equal(t, "TLS13", *prom.Spec.Web.TLSConfig.MaxVersion)

	am := newAlertmanager(ms, "test-alertmanager", AlertmanagerConfiguration{})
	assert.Assert(t, am.Spec.Web != nil)
	assert.Assert(t, am.Spec.Web.TLSConfig != nil)
	assert.Equal(t, "TLS13", *am.Spec.Web.TLSConfig.MinVersion)
	assert.Equal(t, "TLS13", *am.Spec.Web.TLSConfig.MaxVersion)

	assert.Assert(t, prom.Spec.Alerting != nil)
	assert.Assert(t, len(prom.Spec.Alerting.Alertmanagers) > 0)
	amEndpointTLS := prom.Spec.Alerting.Alertmanagers[0].TLSConfig
	assert.Assert(t, amEndpointTLS != nil)
	assert.Equal(t, monv1.TLSVersion13, *amEndpointTLS.MinVersion)
	assert.Equal(t, monv1.TLSVersion13, *amEndpointTLS.MaxVersion)

	scrape := newAdditionalScrapeConfigsSecret(ms, "scrape")
	scrapeCfg := scrape.StringData[AdditionalScrapeConfigsSelfScrapeKey]
	assert.Assert(t, strings.Contains(scrapeCfg, "min_version: TLS13"))
	assert.Assert(t, strings.Contains(scrapeCfg, "max_version: TLS13"))
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
