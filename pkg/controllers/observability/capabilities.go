package observability

import (
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/logging"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/tracing"
)

// capabilities is the only registration point for capability implementations.
func capabilities(opts Options) []capability.Definition {
	return []capability.Definition{
		tracing.New(tracing.Options{OpenTelemetryOperator: opts.OpenTelemetryOperator, TempoOperator: opts.TempoOperator}),
		logging.New(logging.Options{LokiOperator: opts.LokiOperator, ClusterLoggingOperator: opts.ClusterLoggingOperator}),
	}
}

// aggregateShared merges requirements using OR semantics. It never materializes
// operand configuration or reads source credentials for another installer.
func aggregateShared(definitions []capability.Definition, instances []obsv1alpha1.ObservabilityInstaller) capability.Plan {
	var result capability.Plan
	operatorIndex := map[string]int{}
	ownedKeys, desiredKeys := map[string]bool{}, map[string]bool{}
	for _, definition := range definitions {
		// An empty installer supplies inventory even when there are no users.
		plans := []capability.Plan{definition.Shared(&obsv1alpha1.ObservabilityInstaller{})}
		for i := range instances {
			if instances[i].DeletionTimestamp == nil {
				plans = append(plans, definition.Shared(&instances[i]))
			}
		}
		for _, plan := range plans {
			for _, operator := range plan.Operators {
				key := gvkNameIdentifier(operator.Subscription)
				if index, exists := operatorIndex[key]; exists {
					result.Operators[index].Desired = result.Operators[index].Desired || operator.Desired
				} else {
					operatorIndex[key] = len(result.Operators)
					result.Operators = append(result.Operators, operator)
				}
			}
			for _, object := range plan.Owned {
				key := gvkNameIdentifier(object)
				if !ownedKeys[key] {
					ownedKeys[key] = true
					result.Owned = append(result.Owned, object)
				}
			}
			for _, object := range plan.Desired {
				key := gvkNameIdentifier(object)
				if !desiredKeys[key] {
					desiredKeys[key] = true
					result.Desired = append(result.Desired, object)
				}
			}
		}
	}
	return result
}
