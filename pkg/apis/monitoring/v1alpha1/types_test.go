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

// extractValidationRules extracts the two XValidation rules from the comment block
// immediately before the named struct in types.go.
func extractValidationRules(structName string) (nameLengthRule, nameFormatRule string, err error) {
	data, err := os.ReadFile("types.go")
	if err != nil {
		return "", "", fmt.Errorf("failed to read source file: %v", err)
	}

	content := string(data)
	pattern := regexp.MustCompile(`(?s)// ` + structName + ` [^\n]+(?:\n//[^\n]*)*\ntype ` + structName + ` struct`)
	blockMatch := pattern.FindString(content)
	if blockMatch == "" {
		return "", "", fmt.Errorf("%s struct block not found in source", structName)
	}

	rulePattern := regexp.MustCompile(`\+kubebuilder:validation:XValidation:rule="([^"]+)"`)
	matches := rulePattern.FindAllStringSubmatch(blockMatch, -1)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("expected at least 2 XValidation rules on %s, found %d", structName, len(matches))
	}

	return matches[0][1], matches[1][1], nil
}

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

func runNameValidationTests(t *testing.T, rule string, tests []struct {
	name        string
	instName    string
	expectValid bool
}) {
	t.Helper()
	env, err := newCELEnvForMetadata()
	require.NoError(t, err)

	ast, issues := env.Compile(rule)
	require.Empty(t, issues)

	program, err := env.Program(ast)
	require.NoError(t, err)

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

func TestMonitoringStackNameLengthValidation(t *testing.T) {
	lengthRule, _, err := extractValidationRules("MonitoringStack")
	require.NoError(t, err, "failed to extract name length rule from source annotation")

	runNameValidationTests(t, lengthRule, []struct {
		name        string
		instName    string
		expectValid bool
	}{
		{"short valid name", "my-stack", true},
		{"single character", "a", true},
		{"exactly 63 characters", strings.Repeat("a", 63), true},
		{"64 characters - too long", strings.Repeat("a", 64), false},
		{"253 character name", strings.Repeat("a", 253), false},
	})
}

func TestMonitoringStackNameFormatValidation(t *testing.T) {
	_, formatRule, err := extractValidationRules("MonitoringStack")
	require.NoError(t, err, "failed to extract name format rule from source annotation")

	runNameValidationTests(t, formatRule, []struct {
		name        string
		instName    string
		expectValid bool
	}{
		{"single lowercase letter", "a", true},
		{"lowercase letters and dashes", "my-stack", true},
		{"letters numbers and dashes", "stack-1", true},
		{"starts with digit", "1stack", true},
		{"ends with digit", "stack1", true},
		{"uppercase letter - invalid", "MyStack", false},
		{"starts with dash - invalid", "-stack", false},
		{"ends with dash - invalid", "stack-", false},
		{"underscore - invalid", "my_stack", false},
		{"dot separator - invalid", "my.stack", false},
	})
}

func TestThanosQuerierNameLengthValidation(t *testing.T) {
	lengthRule, _, err := extractValidationRules("ThanosQuerier")
	require.NoError(t, err, "failed to extract name length rule from source annotation")

	runNameValidationTests(t, lengthRule, []struct {
		name        string
		instName    string
		expectValid bool
	}{
		{"short valid name", "my-querier", true},
		{"single character", "a", true},
		{"exactly 63 characters", strings.Repeat("a", 63), true},
		{"64 characters - too long", strings.Repeat("a", 64), false},
		{"253 character name", strings.Repeat("a", 253), false},
	})
}

func TestThanosQuerierNameFormatValidation(t *testing.T) {
	_, formatRule, err := extractValidationRules("ThanosQuerier")
	require.NoError(t, err, "failed to extract name format rule from source annotation")

	runNameValidationTests(t, formatRule, []struct {
		name        string
		instName    string
		expectValid bool
	}{
		{"single lowercase letter", "q", true},
		{"lowercase letters and dashes", "my-querier", true},
		{"letters and numbers", "querier1", true},
		{"starts with digit", "1querier", true},
		{"ends with digit", "querier1", true},
		{"uppercase letter - invalid", "MyQuerier", false},
		{"starts with dash - invalid", "-querier", false},
		{"ends with dash - invalid", "querier-", false},
		{"underscore - invalid", "my_querier", false},
		{"dot separator - invalid", "my.querier", false},
	})
}
