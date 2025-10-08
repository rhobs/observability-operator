package v1alpha1

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractValidationRuleFromSource extracts the XValidation rule from the TracingObjectStorageSpec struct annotation
func extractValidationRuleFromSource() (string, error) {
	// Read the source file directly since kubebuilder annotations are in comments
	sourceFile := "tracing.go"

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %v", err)
	}

	content := string(data)

	// Find the TracingObjectStorageSpec struct and extract the XValidation rule
	structPattern := regexp.MustCompile(`// TracingObjectStorageSpec[^{]*?// \+kubebuilder:validation:XValidation:rule="([^"]+)"`)
	match := structPattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", fmt.Errorf("XValidation rule not found in source")
	}

	return match[1], nil
}

// structToMap converts a struct to a map[string]interface{} for CEL evaluation
func structToMap(v interface{}) (map[string]interface{}, error) {
	// Marshal to JSON then unmarshal to map to respect json tags and handle nil pointers properly
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %v", err)
	}

	return result, nil
}

func TestTracingObjectStorageSpecValidation(t *testing.T) {
	// Extract the validation rule directly from the source code annotation
	validationRule, err := extractValidationRuleFromSource()
	require.NoError(t, err, "Failed to extract validation rule from source annotation")

	env, err := cel.NewEnv(cel.Variable("self", cel.MapType(cel.StringType, cel.DynType)))
	require.NoError(t, err)

	ast, issues := env.Compile(validationRule)
	require.Empty(t, issues)

	program, err := env.Program(ast)
	require.NoError(t, err)

	tests := []struct {
		name        string
		spec        TracingObjectStorageSpec
		expectValid bool
	}{
		{
			name:        "no storage types specified - invalid",
			spec:        TracingObjectStorageSpec{},
			expectValid: false,
		},
		{
			name: "only S3 specified",
			spec: TracingObjectStorageSpec{
				S3: &S3Spec{
					Bucket:          "test-bucket",
					Endpoint:        "test-endpoint",
					AccessKeyID:     "test-access-key",
					AccessKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
			},
			expectValid: true,
		},
		{
			name: "only S3STS specified",
			spec: TracingObjectStorageSpec{
				S3STS: &S3STSpec{
					Bucket:  "test-bucket",
					RoleARN: "arn:aws:iam::123456789012:role/test-role",
				},
			},
			expectValid: true,
		},
		{
			name: "only S3CCO specified",
			spec: TracingObjectStorageSpec{
				S3CCO: &S3CCOSpec{
					Bucket: "test-bucket",
				},
			},
			expectValid: true,
		},
		{
			name: "only Azure specified",
			spec: TracingObjectStorageSpec{
				Azure: &AzureSpec{
					Container:        "test-container",
					AccountName:      "test-account",
					AccountKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
			},
			expectValid: true,
		},
		{
			name: "only AzureWIF specified",
			spec: TracingObjectStorageSpec{
				AzureWIF: &AzureWIFSpec{
					Container:   "test-container",
					AccountName: "test-account",
					ClientID:    "test-client-id",
					TenantID:    "test-tenant-id",
				},
			},
			expectValid: true,
		},
		{
			name: "only GCS specified",
			spec: TracingObjectStorageSpec{
				GCS: &GCSSpec{
					Bucket:        "test-bucket",
					KeyJSONSecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
			},
			expectValid: true,
		},
		{
			name: "only GCSWIFSpec specified",
			spec: TracingObjectStorageSpec{
				GCSSTSSpec: &GCSWIFSpec{
					Bucket:        "test-bucket",
					KeyJSONSecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
			},
			expectValid: true,
		},
		{
			name: "S3 and S3STS specified - invalid",
			spec: TracingObjectStorageSpec{
				S3: &S3Spec{
					Bucket:          "test-bucket",
					Endpoint:        "test-endpoint",
					AccessKeyID:     "test-access-key",
					AccessKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
				S3STS: &S3STSpec{
					Bucket:  "test-bucket",
					RoleARN: "arn:aws:iam::123456789012:role/test-role",
				},
			},
			expectValid: false,
		},
		{
			name: "S3 and Azure specified - invalid",
			spec: TracingObjectStorageSpec{
				S3: &S3Spec{
					Bucket:          "test-bucket",
					Endpoint:        "test-endpoint",
					AccessKeyID:     "test-access-key",
					AccessKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
				Azure: &AzureSpec{
					Container:        "test-container",
					AccountName:      "test-account",
					AccountKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
			},
			expectValid: false,
		},
		{
			name: "Azure and GCS specified - invalid",
			spec: TracingObjectStorageSpec{
				Azure: &AzureSpec{
					Container:        "test-container",
					AccountName:      "test-account",
					AccountKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
				GCS: &GCSSpec{
					Bucket:        "test-bucket",
					KeyJSONSecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
			},
			expectValid: false,
		},
		{
			name: "three storage types specified - invalid",
			spec: TracingObjectStorageSpec{
				S3: &S3Spec{
					Bucket:          "test-bucket",
					Endpoint:        "test-endpoint",
					AccessKeyID:     "test-access-key",
					AccessKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
				Azure: &AzureSpec{
					Container:        "test-container",
					AccountName:      "test-account",
					AccountKeySecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
				GCS: &GCSSpec{
					Bucket:        "test-bucket",
					KeyJSONSecret: SecretKeySelector{Name: "test-secret", Key: "key"},
				},
			},
			expectValid: false,
		},
		{
			name: "S3CCO and AzureWIF specified - invalid",
			spec: TracingObjectStorageSpec{
				S3CCO: &S3CCOSpec{
					Bucket: "test-bucket",
				},
				AzureWIF: &AzureWIFSpec{
					Container:   "test-container",
					AccountName: "test-account",
					ClientID:    "test-client-id",
					TenantID:    "test-tenant-id",
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

func TestTracingSpecValidation(t *testing.T) {
	// Extract the validation rule directly from the source code annotation
	validationRule, err := extractTracingSpecValidationRule()
	require.NoError(t, err, "Failed to extract validation rule from source annotation")

	env, err := cel.NewEnv(cel.Variable("self", cel.MapType(cel.StringType, cel.DynType)))
	require.NoError(t, err)

	ast, issues := env.Compile(validationRule)
	require.Empty(t, issues)

	program, err := env.Program(ast)
	require.NoError(t, err)

	tests := []struct {
		name        string
		spec        TracingSpec
		expectValid bool
	}{
		{
			name: "tracing disabled - valid",
			spec: TracingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: false,
				},
			},
			expectValid: true,
		},
		{
			name: "tracing disabled with storage - valid",
			spec: TracingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: false,
				},
				Storage: TracingStorageSpec{
					ObjectStorageSpec: TracingObjectStorageSpec{
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
			name: "tracing enabled with storage - valid",
			spec: TracingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: true,
				},
				Storage: TracingStorageSpec{
					ObjectStorageSpec: TracingObjectStorageSpec{
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
			name: "tracing enabled without storage - invalid",
			spec: TracingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: true,
				},
			},
			expectValid: false,
		},
		{
			name: "tracing enabled with empty storage - invalid",
			spec: TracingSpec{
				CommonCapabilitiesSpec: CommonCapabilitiesSpec{
					Enabled: true,
				},
				Storage: TracingStorageSpec{
					ObjectStorageSpec: TracingObjectStorageSpec{},
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

// extractTracingSpecValidationRule extracts the XValidation rule from the TracingSpec struct annotation
func extractTracingSpecValidationRule() (string, error) {
	// Read the source file directly since kubebuilder annotations are in comments
	sourceFile := "tracing.go"

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %v", err)
	}

	content := string(data)

	// Find the TracingSpec struct and extract the XValidation rule
	structPattern := regexp.MustCompile(`// TracingSpec[^{]*?// \+kubebuilder:validation:XValidation:rule="([^"]+)"`)
	match := structPattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", fmt.Errorf("XValidation rule not found in source")
	}

	return match[1], nil
}

