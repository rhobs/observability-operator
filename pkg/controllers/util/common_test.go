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

func TestTruncateLabelValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "my-resource",
			expected: "my-resource",
		},
		{
			name:     "exactly 63 characters unchanged",
			input:    strings.Repeat("a", 63),
			expected: strings.Repeat("a", 63),
		},
		{
			name:     "64 characters truncated to 63",
			input:    strings.Repeat("a", 64),
			expected: strings.Repeat("a", 63),
		},
		{
			name:     "trailing dash stripped after truncation",
			input:    strings.Repeat("a", 62) + "-xyz",
			expected: strings.Repeat("a", 62),
		},
		{
			name:     "multiple trailing dashes stripped",
			input:    strings.Repeat("a", 60) + "---xyz",
			expected: strings.Repeat("a", 60),
		},
		{
			name:     "empty string unchanged",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLabelValue(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 63)
		})
	}
}

func TestAddCommonLabels_TruncatesLongChildName(t *testing.T) {
	longChildName := "thanos-querier-" + strings.Repeat("a", 63) + "-http-conf"
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      longChildName,
			Namespace: "default",
		},
	}

	result, err := AddCommonLabels(obj, "short-owner")
	require.NoError(t, err)

	labels := result.GetLabels()
	assert.LessOrEqual(t, len(labels["app.kubernetes.io/name"]), 63)
	assert.Equal(t, "short-owner", labels["app.kubernetes.io/part-of"])
}

func TestAddCommonLabels_TruncateStripsTrailingDash(t *testing.T) {
	// Build a name where character 63 is a dash: 62 a's + "-rest"
	childName := strings.Repeat("a", 62) + "-rest-of-name"
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      childName,
			Namespace: "default",
		},
	}

	result, err := AddCommonLabels(obj, "owner")
	require.NoError(t, err)

	labelVal := result.GetLabels()["app.kubernetes.io/name"]
	assert.LessOrEqual(t, len(labelVal), 63)
	assert.False(t, strings.HasSuffix(labelVal, "-"), "label value must not end with a dash")
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
