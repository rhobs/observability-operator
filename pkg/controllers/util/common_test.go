package util

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCommonLabels(t *testing.T) {
	tests := []struct {
		name        string
		ownerName   string
		expectError bool
	}{
		{
			name:        "short valid name",
			ownerName:   "my-stack",
			expectError: false,
		},
		{
			name:        "exactly 63 characters",
			ownerName:   strings.Repeat("a", 63),
			expectError: false,
		},
		{
			name:        "64 characters - too long",
			ownerName:   strings.Repeat("a", 64),
			expectError: true,
		},
		{
			name:        "108 character name from bug report",
			ownerName:   "test-very-long-name-that-exceeds-normal-kubernetes-resource-name-limits-and-should-be-validated-properly",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "default",
				},
			}

			result, err := AddCommonLabels(obj, tt.ownerName)
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				labels := result.GetLabels()
				assert.Equal(t, tt.ownerName, labels["app.kubernetes.io/part-of"])
				assert.Equal(t, "test-resource", labels["app.kubernetes.io/name"])
				assert.Equal(t, OpName, labels[ResourceLabel])
			}
		})
	}
}

func TestAddCommonLabels_PreservesExistingLabels(t *testing.T) {
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				"existing-label":            "existing-value",
				"app.kubernetes.io/part-of": "already-set",
			},
		},
	}

	result, err := AddCommonLabels(obj, "new-owner")
	require.NoError(t, err)

	labels := result.GetLabels()
	assert.Equal(t, "existing-value", labels["existing-label"], "existing label must be preserved")
	assert.Equal(t, "already-set", labels["app.kubernetes.io/part-of"], "pre-set part-of must not be overwritten")
	assert.Equal(t, "test", labels["app.kubernetes.io/name"])
	assert.Equal(t, OpName, labels[ResourceLabel])
}

func TestAddCommonLabels_NilLabels(t *testing.T) {
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	result, err := AddCommonLabels(obj, "my-owner")
	require.NoError(t, err)

	labels := result.GetLabels()
	assert.Equal(t, "my-owner", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "test", labels["app.kubernetes.io/name"])
	assert.Equal(t, OpName, labels[ResourceLabel])
}
