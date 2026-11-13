package capability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDesiredOperators(t *testing.T) {
	tests := []struct {
		name             string
		enabled          bool
		installOperators bool
		wantOperators    bool
	}{
		{name: "disabled"},
		{name: "operators only", installOperators: true, wantOperators: true},
		{name: "enabled", enabled: true, wantOperators: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			operators := DesiredOperators(tt.enabled || tt.installOperators, OperatorRequirement{Name: "operator"})
			require.Len(t, operators, 1)
			require.Equal(t, tt.wantOperators, operators[0].Desired)
		})
	}
}
