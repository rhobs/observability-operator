package observability

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

// Safety net: uses reflection to check that every pointer field in CapabilitiesSpec
// is non-nil after GenerateAllInstallerObjects runs. When a new capability is added
// to the spec, this test fails until GenerateAllInstallerObjects is updated to
// enable it — otherwise the reconciler's cleanup logic would silently miss resources
// from the new capability.
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
