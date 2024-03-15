package status

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type conditionHelper struct {
	Type               string
	Status             string
	Reason             string
	ObservedGeneration int64
	Message            string
	LastTransitionTime metav1.Time
}

func convert[T comparable](v interface{}) T {
	var r T
	converted, ok := v.(T)
	if !ok {
		return r
	}
	return converted
}

// Operand is a wrapper type around client.Object
// It is a helper type to evaluate status condtions
// in generic fashion
type Operand struct {
	name                string
	affectsAvailability bool
	affectsReconciled   bool
	Object              client.Object
}

func NewOperand(obj client.Object, affectsStackAvailability bool, affectsStackReconciled bool) *Operand {
	name := obj.GetObjectKind().GroupVersionKind().Kind
	return &Operand{
		Object:              obj,
		name:                name,
		affectsAvailability: affectsStackAvailability,
		affectsReconciled:   affectsStackReconciled,
	}
}

// getConditionByType converts the operand object to unstructured and
// then tries to find conidtion with provided type.
func (o *Operand) getConditionByType(ctype string) (*conditionHelper, error) {
	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o.Object)
	if err != nil {
		return nil, err
	}

	untypedCon, ok, err := unstructured.NestedSlice(unstrObj, "status", "conditions")
	if !ok {
		return nil, fmt.Errorf("conditions not available")
	}
	if err != nil {
		return nil, err
	}

	for _, untypedC := range untypedCon {
		cMap, ok := untypedC.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("converting to map[string]interface{}: %v", untypedC)
		}

		if t, ok := cMap["type"]; ok {
			if t == ctype {
				return &conditionHelper{
					Type:               convert[string](t),
					Status:             convert[string](cMap["status"]),
					ObservedGeneration: convert[int64](cMap["observedGeneration"]),
					Message:            convert[string](cMap["message"]),
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("can't find any condition with type %s ", ctype)
}
