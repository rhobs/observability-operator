package e2e

import (
	"testing"
)

func TestOperatorReconcileErrors(t *testing.T) {
	// assertCRDExists(t,
	// "prometheuses.monitoring.coreos.com",
	// "monitoringstacks.monitoring.rhobs",
	// )
	f.AssertNoReconcileErrors(t)
}
