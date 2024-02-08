package loggingstack

import (
	stack "github.com/rhobs/observability-operator/pkg/apis/logging/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

func stackComponentReconcilers(ls *stack.LoggingStack) []reconciler.Reconciler {
	return []reconciler.Reconciler{}
}
