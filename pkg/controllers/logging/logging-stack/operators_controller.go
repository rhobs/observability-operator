package loggingstack

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	olmv1 "github.com/operator-framework/api/pkg/operators/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	stack "github.com/rhobs/observability-operator/pkg/apis/logging/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type operatorsResourceManager struct {
	k8sClient         client.Client
	scheme            *runtime.Scheme
	logger            logr.Logger
	controller        controller.Controller
	operatorInstalled chan struct{}
}

// RBAC for managing OperatorLifecycleManager CRs
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=operatorgroups,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;list;watch;delete

func RegisterWithOperatorsManager(mgr ctrl.Manager, c chan struct{}) error {
	rm := &operatorsResourceManager{
		k8sClient:         mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		logger:            ctrl.Log.WithName("observability-operator").WithName("logging-stack").WithName("logging-operators-operator"),
		operatorInstalled: c,
	}

	// We only want to trigger a reconciliation when the generation
	// of a child changes. Until we need to update our the status for our own objects,
	// we can save CPU cycles by avoiding reconciliations triggered by
	// child status changes. The only exception is Prometheus resources, where we want to
	// be notified about changes in their status.
	generationChanged := builder.WithPredicates(predicate.GenerationChangedPredicate{})

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		For(&stack.LoggingStack{}).
		Owns(&olmv1.OperatorGroup{}, generationChanged).
		Owns(&olmv1alpha1.Subscription{}, generationChanged).
		Build(rm)
	if err != nil {
		return err
	}
	rm.controller = ctrl

	return nil
}

func (rm operatorsResourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("stack", req.NamespacedName)
	logger.Info("Reconciling logging stack")

	ls, err := rm.getStack(ctx, req)
	if err != nil {
		// retry since some error has occured
		return ctrl.Result{}, err
	}

	if ls == nil || !ls.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.V(6).Info("skipping reconcile since object is already schedule for deletion and cleanup logging operator install resources")

		res, err := rm.deleteManagedOperators(ctx, logger)
		if err != nil {
			return res, err
		}

		return ctrl.Result{}, nil
	}

	reconcilers := operatorsComponentReconcilers(ls)
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

	if res, ok := rm.operatorsInstalled(ctx, logger); !ok {
		return res, nil
	}

	// Signal stack operator to start
	select {
	case rm.operatorInstalled <- struct{}{}:
		return ctrl.Result{}, nil
	default:
		logger.V(6).Info("skipping sending operatorInstalled signal as channel closed")
	}

	return ctrl.Result{}, nil
}
func (rm operatorsResourceManager) updateStatus(ctx context.Context, req ctrl.Request, ls *stack.LoggingStack, recError error) ctrl.Result {

	return ctrl.Result{}
}

func (rm operatorsResourceManager) operatorsInstalled(ctx context.Context, logger logr.Logger) (ctrl.Result, bool) {
	// Cluster Logging Operator
	_, cloCSV, err := rm.getOperatorResources(ctx, logger, nameClusterLoggingOperator, stackNamespace)
	if err != nil {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, false
	}

	// Loki Operator
	_, loCSV, err := rm.getOperatorResources(ctx, logger, nameLokiOperator, namespaceLokiOperator)
	if err != nil {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, false
	}

	var (
		isCLOInstalled = cloCSV.Status.Phase == olmv1alpha1.CSVPhaseSucceeded
		isLOInstalled  = loCSV.Status.Phase == olmv1alpha1.CSVPhaseSucceeded
	)

	if isCLOInstalled && isLOInstalled {
		return ctrl.Result{}, true
	}

	return ctrl.Result{RequeueAfter: 2 * time.Second}, false
}

func (rm operatorsResourceManager) getOperatorResources(ctx context.Context, logger logr.Logger, operatorName, operatorNs string) (*olmv1alpha1.Subscription, *olmv1alpha1.ClusterServiceVersion, error) {
	// Cluster Logging Operator
	sub := &olmv1alpha1.Subscription{}
	subKey := client.ObjectKey{Name: operatorName, Namespace: operatorNs}
	if err := rm.k8sClient.Get(ctx, subKey, sub); err != nil {
		logger.Error(err, "failed to get subscription", "operator-name", operatorName)
		return nil, nil, err
	}

	csv := &olmv1alpha1.ClusterServiceVersion{}
	csvKey := client.ObjectKey{Name: sub.Status.CurrentCSV, Namespace: operatorNs}
	if err := rm.k8sClient.Get(ctx, csvKey, csv); err != nil {
		logger.Error(err, "failed to get clusterserviceversion", "operator-name", operatorName)
		return nil, nil, err
	}

	return sub, csv, nil
}

func (rm operatorsResourceManager) deleteManagedOperators(ctx context.Context, logger logr.Logger) (ctrl.Result, error) {
	cloSub, cloCSV, err := rm.getOperatorResources(ctx, logger, nameClusterLoggingOperator, stackNamespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	loSub, loCSV, err := rm.getOperatorResources(ctx, logger, nameLokiOperator, namespaceLokiOperator)
	if err != nil {
		return ctrl.Result{}, err
	}

	deleters := []reconciler.Reconciler{
		reconciler.NewDeleter(cloCSV),
		reconciler.NewDeleter(cloSub),
		reconciler.NewDeleter(loCSV),
		reconciler.NewDeleter(loSub),
	}
	for _, deleter := range deleters {
		err := deleter.Reconcile(ctx, rm.k8sClient, rm.scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (rm operatorsResourceManager) getStack(ctx context.Context, req ctrl.Request) (*stack.LoggingStack, error) {
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
