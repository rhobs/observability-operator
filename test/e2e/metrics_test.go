package e2e

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/rhobs/observability-operator/test/e2e/framework"
)

func TestOperatorMetrics(t *testing.T) {
	pod := f.GetOperatorPod(t)

	var opts []func(*framework.HTTPOptions)
	if f.IsOpenshiftCluster {
		opts = append(opts, framework.WithHTTPS())
	}

	metrics, err := f.GetPodMetrics(pod, opts...)
	assert.NilError(t, err)

	v, err := framework.ParseMetrics(metrics)
	assert.NilError(t, err)

	assert.Assert(t, len(v) > 0, "no metrics")
}
