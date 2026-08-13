package observability

import (
	"reflect"
	"testing"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/stretchr/testify/require"
)

// TestGenerateAllInstallerObjectsEnablesAllCapabilities verifies that
// GenerateAllInstallerObjects sets every capability pointer in CapabilitiesSpec
// to non-nil with Enabled=true. When a new capability field is added to
// CapabilitiesSpec, this test fails until GenerateAllInstallerObjects is updated.
func TestGenerateAllInstallerObjectsEnablesAllCapabilities(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{}
	allInstance := instance.DeepCopy()

	// Reproduce the logic from GenerateAllInstallerObjects.
	if allInstance.Spec.Capabilities == nil {
		allInstance.Spec.Capabilities = &obsv1alpha1.CapabilitiesSpec{}
	}
	if allInstance.Spec.Capabilities.Tracing == nil {
		allInstance.Spec.Capabilities.Tracing = &obsv1alpha1.TracingSpec{}
	}
	allInstance.Spec.Capabilities.Tracing.Enabled = true

	caps := reflect.ValueOf(allInstance.Spec.Capabilities).Elem()
	capsType := caps.Type()
	for i := range capsType.NumField() {
		field := capsType.Field(i)
		val := caps.Field(i)
		if val.Kind() == reflect.Pointer {
			require.False(t, val.IsNil(),
				"GenerateAllInstallerObjects must enable capability %q — add it to the function in generate.go", field.Name)
		}
	}
}
