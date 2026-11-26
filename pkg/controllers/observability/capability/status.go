package capability

import (
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

// StatusReadError converts an operand status read failure into the retry
// behavior expected by the ObservabilityInstaller controller. All read failures
// request a short retry. A missing operand, or a CRD that is not established
// yet because its operator is still installing, is treated as normal
// convergence and does not set Err; any other failure is returned in Err so the
// controller can expose it through the Reconciled condition.
//
// kind identifies the operand in the wrapped error and should be a Kubernetes
// kind such as "TempoStack" or "ClusterLogForwarder".
func StatusReadError(kind string, err error) StatusResult {
	result := StatusResult{RequeueAfter: 2 * time.Second}
	if !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		result.Err = fmt.Errorf("get %s status: %w", kind, err)
	}
	return result
}
