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
// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;list;watch

func RegisterWithOperatorsManager(mgr ctrl.Manager, c chan struct{}) error {
	rm := &operatorsResourceManager{
		k8sClient:         mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		logger:            ctrl.Log.WithName("observability-operator").WithName("logging-stack"),
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

	if ls == nil {
		// no such logging stack, so stop here
		return ctrl.Result{}, nil
	}

	if !ls.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.V(6).Info("skipping reconcile since object is already schedule for deletion")
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

	if res, ok := rm.operatorsInstalled(ctx, ls); !ok {
		return res, nil
	}

	// Signal stack operator to start
	rm.operatorInstalled <- struct{}{}

	return ctrl.Result{}, nil
}
func (rm operatorsResourceManager) updateStatus(ctx context.Context, req ctrl.Request, ls *stack.LoggingStack, recError error) ctrl.Result {
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

func (rm operatorsResourceManager) operatorsInstalled(ctx context.Context, ls *stack.LoggingStack) (ctrl.Result, bool) {
	logger := rm.logger.WithValues("stack", client.ObjectKeyFromObject(ls))

	var (
		isCLOInstalled = false
		isLOInstalled  = false
	)

	// Cluster Logging Operator
	cloSub := &olmv1alpha1.Subscription{}
	cloSubKey := client.ObjectKeyFromObject(newClusterLoggingOperatorSubscription(ls))
	if err := rm.k8sClient.Get(ctx, cloSubKey, cloSub); err != nil {
		logger.Error(err, "failed to get cluster-logging subscription")
		return ctrl.Result{RequeueAfter: 2 * time.Second}, false
	}

	cloCSV := &olmv1alpha1.ClusterServiceVersion{}
	cloCSVKey := client.ObjectKey{Name: cloSub.Status.CurrentCSV, Namespace: cloSub.Namespace}
	if err := rm.k8sClient.Get(ctx, cloCSVKey, cloCSV); err != nil {
		logger.Error(err, "failed to get cluster-logging CSV")
		return ctrl.Result{RequeueAfter: 2 * time.Second}, false
	}

	isCLOInstalled = cloCSV.Status.Phase == olmv1alpha1.CSVPhaseSucceeded

	// Loki Operator
	loSub := &olmv1alpha1.Subscription{}
	loSubKey := client.ObjectKeyFromObject(newLokiOperatorSubscription(ls))
	if err := rm.k8sClient.Get(ctx, loSubKey, loSub); err != nil {
		logger.Error(err, "failed to get loki-operator subscription")
		return ctrl.Result{RequeueAfter: 2 * time.Second}, false
	}

	loCSV := &olmv1alpha1.ClusterServiceVersion{}
	loCSVKey := client.ObjectKey{Name: loSub.Status.CurrentCSV, Namespace: loSub.Namespace}
	if err := rm.k8sClient.Get(ctx, loCSVKey, loCSV); err != nil {
		logger.Error(err, "failed to get loki-operator CSV")
		return ctrl.Result{RequeueAfter: 2 * time.Second}, false
	}

	isLOInstalled = loCSV.Status.Phase == olmv1alpha1.CSVPhaseSucceeded

	return ctrl.Result{}, isCLOInstalled && isLOInstalled
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
