package monitoringstack

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestStorageSpec(t *testing.T) {

	validPVCSpec := &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.ResourceRequirements{
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
