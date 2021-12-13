package grafana_operator

import (
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// installPlanFilter filter events on v1alpha1.InstallPlan resources
type installPlanFilter struct {
	predicate.Funcs
}

// Update returns true only when the v1alpha1.Status.BundleLookup field
// is set by OLM.
func (f installPlanFilter) Update(e event.UpdateEvent) bool {
	previous, ok := e.ObjectOld.(*v1alpha1.InstallPlan)
	if !ok {
		return false
	}
	current, ok := e.ObjectNew.(*v1alpha1.InstallPlan)
	if !ok {
		return false
	}

	return len(previous.Status.BundleLookups) == 0 && len(current.Status.BundleLookups) != 0
}
