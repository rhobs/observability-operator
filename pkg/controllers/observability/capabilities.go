package observability

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/tracing"
)

// capabilities is the only registration point for capability implementations.
func capabilities(opts Options) []capability.Capability {
	return []capability.Capability{
		tracing.New(tracing.Options{OpenTelemetryOperator: opts.OpenTelemetryOperator, TempoOperator: opts.TempoOperator}),
	}
}

// operatorPlan is an operator requirement with its aggregated desired state.
type operatorPlan struct {
	capability.OperatorRequirement
	desired bool
}

// sharedPlan is the cluster-wide state aggregated across installers.
type sharedPlan struct {
	operators []operatorPlan
	desired   []client.Object
	owned     []client.Object
}

// aggregateShared merges requirements using OR semantics. It never materializes
// operand configuration or reads source credentials for another installer.
func aggregateShared(capabilities []capability.Capability, instances []obsv1alpha1.ObservabilityInstaller) (sharedPlan, error) {
	var result sharedPlan
	operatorIndex := map[string]int{}
	desiredKeys := map[string]bool{}
	for _, c := range capabilities {
		inventory := c.SharedInventory()
		result.owned = append(result.owned, inventory...)
		operatorsDesired := false
		for i := range instances {
			if instances[i].DeletionTimestamp != nil {
				continue
			}
			operatorsDesired = operatorsDesired || c.OperatorsDesired(&instances[i])
			desired := c.SharedDesired(&instances[i])
			if err := checkInventory(c.Name(), desired, inventory); err != nil {
				return sharedPlan{}, err
			}
			for _, object := range desired {
				if key := gvkNameIdentifier(object); !desiredKeys[key] {
					desiredKeys[key] = true
					result.desired = append(result.desired, object)
				}
			}
		}
		// Capabilities may share an operator; it is desired if any capability needs it.
		for _, operator := range c.Operators() {
			key := gvkNameIdentifier(operator.Subscription)
			if index, exists := operatorIndex[key]; exists {
				result.operators[index].desired = result.operators[index].desired || operatorsDesired
			} else {
				operatorIndex[key] = len(result.operators)
				result.operators = append(result.operators, operatorPlan{OperatorRequirement: operator, desired: operatorsDesired})
			}
		}
	}
	return result, nil
}

// checkInventory ensures desired resources can be cleaned up later.
func checkInventory(name string, desired, inventory []client.Object) error {
	keys := map[string]bool{}
	for _, object := range inventory {
		keys[gvkNameIdentifier(object)] = true
	}
	for _, object := range desired {
		if key := gvkNameIdentifier(object); !keys[key] {
			return fmt.Errorf("capability %s: desired resource %s is not in its inventory", name, key)
		}
	}
	return nil
}
