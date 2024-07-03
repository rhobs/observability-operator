package uiplugin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsVersionAheadOrEqual(t *testing.T) {
	testCases := []struct {
		clusterVersion     string
		nextClusterVersion string
		expected           bool
	}{
		{
			clusterVersion:     "v4.18",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "v4.17",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "v4.16",
			nextClusterVersion: "v4.17",
			expected:           false,
		},
		{
			clusterVersion:     "4.18",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "4.17.0-0.nightly-2024-07-09-121045",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "4.16.0-0.nightly-2024-07-09-121045",
			nextClusterVersion: "v4.17",
			expected:           false,
		},
		{
			clusterVersion:     "v4.18",
			nextClusterVersion: "",
			expected:           false,
		},
	}

	for _, tc := range testCases {
		actual := isVersionAheadOrEqual(tc.clusterVersion, tc.nextClusterVersion)
		assert.Equal(t, tc.expected, actual)
	}
}
