package images

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		images  map[string]string
		wantErr string
	}{
		{
			name:   "nil map returns defaults",
			images: nil,
		},
		{
			name:   "empty map returns defaults",
			images: map[string]string{},
		},
		{
			name:   "valid override",
			images: map[string]string{"korrel8r": "example.com/korrel8r:latest"},
		},
		{
			name:    "unknown image name",
			images:  map[string]string{"no-such-image": "example.com/nope:v1"},
			wantErr: "unknown",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Validate(tc.images)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, len(DefaultImages), len(result))
			for k, v := range tc.images {
				require.Equal(t, v, result[k])
			}
		})
	}
}
