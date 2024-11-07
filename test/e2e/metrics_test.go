package e2e

import (
	"fmt"
	"testing"

	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"

	"github.com/rhobs/observability-operator/test/e2e/framework"
)

func TestMetrics(t *testing.T) {
	t.Run("operator exposes metrics", func(t *testing.T) {
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
	})

	t.Run("metrics ingested in Prometheus", func(t *testing.T) {
		if !f.IsOpenshiftCluster {
			t.Skip("requires an OpenShift cluster")
		}

		err := f.AssertPromQLResult(
			t,
			fmt.Sprintf(`up{job="observability-operator",namespace="%s"} == 1`, f.OperatorNamespace),
			func(v model.Value) error {
				if v.Type() != model.ValVector {
					return fmt.Errorf("invalid value type: expecting %d, got %s", model.ValVector, v.Type())
				}

				vec := v.(model.Vector)
				if len(vec) != 1 {
					return fmt.Errorf("expecting 1 item, got %d", len(vec))
				}

				if vec[0].Value != 1.0 {
					return fmt.Errorf("expecting value 1, got %f", vec[0].Value)
				}

				return nil
			})
		assert.NilError(t, err)
	})
}
