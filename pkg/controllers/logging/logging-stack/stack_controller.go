package loggingstack

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	loggingv1 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	stack "github.com/rhobs/observability-operator/pkg/apis/logging/v1alpha1"
	observabilityuiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observabilityui/v1alpha1"
)

type resourceManager struct {
	k8sClient  client.Client
	scheme     *runtime.Scheme
	logger     logr.Logger
	controller controller.Controller
}

// RBAC for managing logging stacks
// +kubebuilder:rbac:groups=logging.rhobs,resources=loggingstacks,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=logging.rhobs,resources=loggingstacks/status,verbs=get;update

// RBAC for managing ObservabilityUI Plugin
// +kubebuilder:rbac:groups=observabilityui.rhobs,resources=observabilityuiplugins,verbs=get;list;watch;create;update

// RBAC for managing ClusterLoggingOperator CRs
// +kubebuilder:rbac:groups=logging.openshift.io,resources=clusterloggings,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=logging.openshift.io,resources=clusterlogforwarders,verbs=get;list;watch;create;update;delete;patch

// RBAC for managing LokiOperator CRs
// +kubebuilder:rbac:groups=loki.grafana.com,resources=alertingrules,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=loki.grafana.com,resources=lokistacks,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=loki.grafana.com,resources=recordingrules,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=loki.grafana.com,resources=rulerconfigs,verbs=get;list;watch;create;update;delete;patch

func RegisterWithStackManager(mgr ctrl.Manager) error {
	rm := &resourceManager{
		k8sClient: mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		logger:    ctrl.Log.WithName("observability-operator").WithName("logging-stack").WithName("custom-resources-operator"),
	}
	// We want to trigger reconciliation when the resource version
	// of a child changes to update the status about changes in
	// ClusterLogging, ClusterLogForwarder and LokiStack status.
	resourceVersionChanged := builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		For(&stack.LoggingStack{}).
		Owns(&lokiv1.LokiStack{}, resourceVersionChanged).
		Owns(&loggingv1.ClusterLogging{}, resourceVersionChanged).
		Owns(&loggingv1.ClusterLogForwarder{}, resourceVersionChanged).
		Owns(&observabilityuiv1alpha1.ObservabilityUIPlugin{}, resourceVersionChanged).
		Build(rm)
	if err != nil {
		return err
	}
	rm.controller = ctrl

	return nil
}

func (rm resourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("stack", req.NamespacedName)
	logger.Info("Reconciling logging stack")

	ls, err := rm.getStack(ctx, req)
	if err != nil {
		// retry since some error has occured
		return ctrl.Result{}, err
	}

	if ls == nil {
		// no such logging stack, so stop here
		return ctrl.Result{}, nil
	}

	if !ls.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.V(6).Info("skipping reconcile since object is already schedule for deletion")
		return ctrl.Result{}, nil
	}

	reconcilers := stackComponentReconcilers(ls)
	for _, reconciler := range reconcilers {
		err := reconciler.Reconcile(ctx, rm.k8sClient, rm.scheme)
		// handle create / update errors that can happen due to a stale cache by
		// retrying after some time.
		if errors.IsAlreadyExists(err) || errors.IsConflict(err) {
			logger.V(3).Info("skipping reconcile error", "err", err)

			if reconciler.Requeue {
				return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
			}
			continue
		}
		if err != nil {
			return rm.updateStatus(ctx, req, ls, err), err
		}
	}

	return ctrl.Result{}, nil
}
func (rm resourceManager) updateStatus(ctx context.Context, req ctrl.Request, ls *stack.LoggingStack, recError error) ctrl.Result {
	// var prom monv1.Prometheus
	// logger := rm.logger.WithValues("stack", req.NamespacedName)
	// key := client.ObjectKey{
	//	Name:      ms.Name,
	//	Namespace: ms.Namespace,
	// }
	// err := rm.k8sClient.Get(ctx, key, &prom)
	// if err != nil {
	//	logger.Info("Failed to get prometheus object", "err", err)
	//	return ctrl.Result{RequeueAfter: 2 * time.Second}
	// }
	// ms.Status.Conditions = updateConditions(ms, prom, recError)
	// err = rm.k8sClient.Status().Update(ctx, ms)
	// if err != nil {
	//	logger.Info("Failed to update status", "err", err)
	//	return ctrl.Result{RequeueAfter: 2 * time.Second}
	// }
	return ctrl.Result{}
}

func (rm resourceManager) getStack(ctx context.Context, req ctrl.Request) (*stack.LoggingStack, error) {
	logger := rm.logger.WithValues("stack", req.NamespacedName)

	ls := stack.LoggingStack{}

	if err := rm.k8sClient.Get(ctx, req.NamespacedName, &ls); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("stack could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get logging stack")
		return nil, err

	}

	return &ls, nil
}
