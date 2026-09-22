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

func TestPodSecurityProfiles(t *testing.T) {
	tests := []struct {
		name            string
		profile         stack.PodSecurityProfile
		expectedSCC     string
		expectStaticIDs bool
		expectHostUsers bool
	}{
		{
			name:            "default profile preserves static IDs",
			expectedSCC:     "nonroot-v2",
			expectStaticIDs: true,
		},
		{
			name:        "restricted-v2 uses namespace IDs",
			profile:     stack.RestrictedV2PodSecurityProfile,
			expectedSCC: "restricted-v2",
		},
		{
			name:            "restricted-v3 enables pod user namespaces",
			profile:         stack.RestrictedV3PodSecurityProfile,
			expectedSCC:     "restricted-v3",
			expectHostUsers: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms := &stack.MonitoringStack{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
				Spec: stack.MonitoringStackSpec{
					PrometheusConfig: &stack.PrometheusConfig{},
					PodSecurity: stack.PodSecurityConfig{
						Profile: test.profile,
					},
				},
			}

			prometheus := newPrometheus(
				ms,
				"test-prometheus",
				"test-scrape",
				ThanosConfiguration{},
				PrometheusConfiguration{},
			)
			alertmanager := newAlertmanager(ms, "test-alertmanager", AlertmanagerConfiguration{})

			assert.Equal(t, prometheus.Spec.PodMetadata.Annotations[requiredSCCAnnotation], test.expectedSCC)
			assert.Equal(t, alertmanager.Spec.PodMetadata.Annotations[requiredSCCAnnotation], test.expectedSCC)
			assert.Equal(t, prometheus.Spec.SecurityContext.SeccompProfile.Type, corev1.SeccompProfileTypeRuntimeDefault)
			assert.Equal(t, alertmanager.Spec.SecurityContext.SeccompProfile.Type, corev1.SeccompProfileTypeRuntimeDefault)
			assert.Assert(t, *prometheus.Spec.SecurityContext.RunAsNonRoot)
			assert.Assert(t, *alertmanager.Spec.SecurityContext.RunAsNonRoot)

			if test.expectStaticIDs {
				assert.Equal(t, *prometheus.Spec.SecurityContext.RunAsUser, PrometheusUserFSGroupID)
				assert.Equal(t, *prometheus.Spec.SecurityContext.FSGroup, PrometheusUserFSGroupID)
				assert.Equal(t, *alertmanager.Spec.SecurityContext.RunAsUser, AlertmanagerUserFSGroupID)
				assert.Equal(t, *alertmanager.Spec.SecurityContext.FSGroup, AlertmanagerUserFSGroupID)
				assert.Assert(t, prometheus.Spec.SecurityContext.FSGroupChangePolicy == nil)
				assert.Assert(t, alertmanager.Spec.SecurityContext.FSGroupChangePolicy == nil)
			} else {
				assert.Assert(t, prometheus.Spec.SecurityContext.RunAsUser == nil)
				assert.Assert(t, prometheus.Spec.SecurityContext.FSGroup == nil)
				assert.Assert(t, alertmanager.Spec.SecurityContext.RunAsUser == nil)
				assert.Assert(t, alertmanager.Spec.SecurityContext.FSGroup == nil)
				assert.Equal(t, *prometheus.Spec.SecurityContext.FSGroupChangePolicy, corev1.FSGroupChangeOnRootMismatch)
				assert.Equal(t, *alertmanager.Spec.SecurityContext.FSGroupChangePolicy, corev1.FSGroupChangeOnRootMismatch)
			}

			if test.expectHostUsers {
				assert.Assert(t, prometheus.Spec.HostUsers != nil && !*prometheus.Spec.HostUsers)
				assert.Assert(t, alertmanager.Spec.HostUsers != nil && !*alertmanager.Spec.HostUsers)
			} else {
				assert.Assert(t, prometheus.Spec.HostUsers == nil)
				assert.Assert(t, alertmanager.Spec.HostUsers == nil)
			}

			prometheusRole := newPrometheusClusterRole("test-prometheus", rbacVerbs, podSecurityProfile(ms))
			alertmanagerRole := newAlertManagerClusterRole("test-alertmanager", rbacVerbs, podSecurityProfile(ms))
			assert.DeepEqual(t, prometheusRole.Rules[len(prometheusRole.Rules)-1].ResourceNames, []string{test.expectedSCC})
			assert.DeepEqual(t, alertmanagerRole.Rules[0].ResourceNames, []string{test.expectedSCC})
		})
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
