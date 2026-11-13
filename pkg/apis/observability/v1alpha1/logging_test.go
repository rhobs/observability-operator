package v1alpha1

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractLoggingSpecValidationRule extracts the XValidation rule from the LoggingSpec struct annotation
func extractLoggingSpecValidationRule() (string, error) {
	// Read the source file directly since kubebuilder annotations are in comments
	sourceFile := "logging.go"

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %v", err)
	}

	content := string(data)

	// Find the LoggingSpec struct and extract the XValidation rule
	structPattern := regexp.MustCompile(`// LoggingSpec[^{]*?// \+kubebuilder:validation:XValidation:rule="([^"]+)"`)
	match := structPattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", fmt.Errorf("XValidation rule not found in source")
	}

	return match[1], nil
}

func TestLoggingSpecValidation(t *testing.T) {
	// Extract the validation rule directly from the source code annotation
	validationRule, err := extractLoggingSpecValidationRule()
	require.NoError(t, err, "Failed to extract validation rule from source annotation")

	env, err := cel.NewEnv(cel.Variable("self", cel.MapType(cel.StringType, cel.DynType)))
	require.NoError(t, err)

	ast, issues := env.Compile(validationRule)
	require.Empty(t, issues)

	program, err := env.Program(ast)
	require.NoError(t, err)

	tests := []struct {
		name        string
		spec        LoggingSpec
		expectValid bool
	}{
		{
			name: "logging disabled - valid",
			spec: LoggingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: false,
				},
			},
			expectValid: true,
		},
		{
			name: "logging disabled without lokiStack - valid",
			spec: LoggingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: false,
				},
				LokiStack: &LokiStackSpec{
					StorageClassName: "gp3",
					Size:             "1x.small",
				},
			},
			expectValid: true,
		},
		{
			name: "logging enabled with lokiStack and object storage - valid",
			spec: LoggingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: true,
				},
				LokiStack: &LokiStackSpec{
					StorageClassName: "gp3",
					Size:             "1x.small",
					ObjectStorage: &ObjectStorageSpec{
						S3: &S3Spec{
							Bucket:          "test-bucket",
							Endpoint:        "test-endpoint",
							AccessKeyID:     "test-access-key",
							AccessKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
						},
					},
				},
			},
			expectValid: true,
		},
		{
			name: "logging enabled without lokiStack - invalid",
			spec: LoggingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: true,
				},
			},
			expectValid: false,
		},
		{
			name: "logging enabled with lokiStack but no object storage - invalid",
			spec: LoggingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: true,
				},
				LokiStack: &LokiStackSpec{
					StorageClassName: "gp3",
					Size:             "1x.small",
				},
			},
			expectValid: false,
		},
		{
			name: "logging enabled with lokiStack and empty object storage - invalid",
			spec: LoggingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: true,
				},
				LokiStack: &LokiStackSpec{
					StorageClassName: "gp3",
					Size:             "1x.small",
					ObjectStorage:    &ObjectStorageSpec{},
				},
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert the struct to a map for CEL evaluation
			selfMap, err := structToMap(tt.spec)
			require.NoError(t, err, "Failed to convert struct to map")

			out, _, err := program.Eval(map[string]interface{}{
				"self": selfMap,
			})
			require.NoError(t, err)

			result := out.Value().(bool)
			if tt.expectValid {
				assert.True(t, result, "Expected configuration to be valid")
			} else {
				assert.False(t, result, "Expected configuration to be invalid")
			}
		})
	}
}
