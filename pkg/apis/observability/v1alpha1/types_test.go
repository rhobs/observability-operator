package v1alpha1

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractObservabilityInstallerValidationRules reads the XValidation rules from the
// ObservabilityInstaller struct comment block in types.go.
func extractObservabilityInstallerValidationRules() (nameLengthRule, nameFormatRule string, err error) {
	data, err := os.ReadFile("types.go")
	if err != nil {
		return "", "", fmt.Errorf("failed to read source file: %v", err)
	}

	content := string(data)

	// Capture the comment block + declaration line for ObservabilityInstaller
	blockPattern := regexp.MustCompile(`(?s)ObservabilityInstaller defines.*?type ObservabilityInstaller struct`)
	blockMatch := blockPattern.FindString(content)
	if blockMatch == "" {
		return "", "", fmt.Errorf("ObservabilityInstaller struct block not found in source")
	}

	rulePattern := regexp.MustCompile(`\+kubebuilder:validation:XValidation:rule="([^"]+)"`)
	matches := rulePattern.FindAllStringSubmatch(blockMatch, -1)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("expected at least 2 XValidation rules on ObservabilityInstaller, found %d", len(matches))
	}

	return matches[0][1], matches[1][1], nil
}

// newCELEnvForMetadata creates a CEL environment where self is a nested map that
// includes metadata.name, matching the shape Kubernetes presents to root-level rules.
func newCELEnvForMetadata() (*cel.Env, error) {
	return cel.NewEnv(cel.Variable("self", cel.MapType(cel.StringType, cel.DynType)))
}

func makeMetaSelf(name string) map[string]interface{} {
	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": name,
		},
	}
}

func TestObservabilityInstallerNameLengthValidation(t *testing.T) {
	lengthRule, _, err := extractObservabilityInstallerValidationRules()
	require.NoError(t, err, "Failed to extract name length rule from source annotation")

	env, err := newCELEnvForMetadata()
	require.NoError(t, err)

	ast, issues := env.Compile(lengthRule)
	require.Empty(t, issues)

	program, err := env.Program(ast)
	require.NoError(t, err)

	tests := []struct {
		name        string
		instName    string
		expectValid bool
	}{
		{
			name:        "short valid name",
			instName:    "my-installer",
			expectValid: true,
		},
		{
			name:        "single character name",
			instName:    "a",
			expectValid: true,
		},
		{
			name:        "exactly 63 characters",
			instName:    strings.Repeat("a", 63),
			expectValid: true,
		},
		{
			name:        "64 characters - too long",
			instName:    strings.Repeat("a", 64),
			expectValid: false,
		},
		{
			// Exact name from the bug report - 108 characters
			name:        "108 character name from bug report",
			instName:    "test-very-long-name-that-exceeds-normal-kubernetes-resource-name-limits-and-should-be-validated-properly",
			expectValid: false,
		},
		{
			name:        "253 character name",
			instName:    strings.Repeat("a", 253),
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := program.Eval(map[string]interface{}{
				"self": makeMetaSelf(tt.instName),
			})
			require.NoError(t, err)

			result := out.Value().(bool)
			if tt.expectValid {
				assert.True(t, result, "Expected name %q to be valid (length %d)", tt.instName, len(tt.instName))
			} else {
				assert.False(t, result, "Expected name %q to be invalid (length %d)", tt.instName, len(tt.instName))
			}
		})
	}
}

func TestObservabilityInstallerNameFormatValidation(t *testing.T) {
	_, formatRule, err := extractObservabilityInstallerValidationRules()
	require.NoError(t, err, "Failed to extract name format rule from source annotation")

	env, err := newCELEnvForMetadata()
	require.NoError(t, err)

	ast, issues := env.Compile(formatRule)
	require.Empty(t, issues)

	program, err := env.Program(ast)
	require.NoError(t, err)

	tests := []struct {
		name        string
		instName    string
		expectValid bool
	}{
		{
			name:        "single lowercase letter",
			instName:    "a",
			expectValid: true,
		},
		{
			name:        "lowercase letters and dashes",
			instName:    "my-installer",
			expectValid: true,
		},
		{
			name:        "lowercase letters and numbers",
			instName:    "installer123",
			expectValid: true,
		},
		{
			name:        "letters numbers and dashes mixed",
			instName:    "obs-installer-1",
			expectValid: true,
		},
		{
			name:        "starts with digit",
			instName:    "1installer",
			expectValid: true,
		},
		{
			name:        "ends with digit",
			instName:    "installer1",
			expectValid: true,
		},
		{
			name:        "uppercase letter - invalid",
			instName:    "MyInstaller",
			expectValid: false,
		},
		{
			name:        "starts with dash - invalid",
			instName:    "-installer",
			expectValid: false,
		},
		{
			name:        "ends with dash - invalid",
			instName:    "installer-",
			expectValid: false,
		},
		{
			name:        "underscore - invalid",
			instName:    "my_installer",
			expectValid: false,
		},
		{
			name:        "dot separator - invalid",
			instName:    "my.installer",
			expectValid: false,
		},
		{
			name:        "108 character name from bug report",
			instName:    "test-very-long-name-that-exceeds-normal-kubernetes-resource-name-limits-and-should-be-validated-properly",
			expectValid: true, // valid format, but fails length check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := program.Eval(map[string]interface{}{
				"self": makeMetaSelf(tt.instName),
			})
			require.NoError(t, err)

			result := out.Value().(bool)
			if tt.expectValid {
				assert.True(t, result, "Expected name %q to match format", tt.instName)
			} else {
				assert.False(t, result, "Expected name %q to not match format", tt.instName)
			}
		})
	}
}
