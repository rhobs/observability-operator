// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	korrel8rdeployName                  = "korrel8r"
	OperatorNamespace      = "operators"
)

var createOrUpdateDeployment = builder.WithPredicates(predicate.Funcs{
	UpdateFunc:  func(e event.UpdateEvent) bool { 
		return e.ObjectNew.GetNamespace() == OperatorNamespace && e.ObjectNew.GetName() == korrel8rdeployName },
	CreateFunc:  func(e event.CreateEvent) bool { 
		return e.Object.GetNamespace() == OperatorNamespace && e.Object.GetName() == korrel8rdeployName },
	DeleteFunc:  func(e event.DeleteEvent) bool { return false },
	GenericFunc: func(e event.GenericEvent) bool { return false },
})

// Korrel8rReconciler reconciles a Korrel8r object
type resourceManager struct {
	k8sClient  client.Client
	scheme     *runtime.Scheme
	logger     logr.Logger
	controller controller.Controller
	korrel8rconf Korrel8rConfiguration
	version      string
}

type Korrel8rConfiguration struct {
	Image string
}

// Options allows for controller options to be set
type Options struct {
	Korrel8rconf Korrel8rConfiguration
}
// List permissions needed by Reconcile - used to generate role.yaml
//
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=configmaps;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
//

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager, opts Options) error {

	rm := &resourceManager{
		k8sClient:    mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		logger:       ctrl.Log.WithName("observability-operator"),
		korrel8rconf: opts.Korrel8rconf,
	}

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		Named("Korrel8r").
		Watches(&corev1.Service{}, &handler.EnqueueRequestForObject{}, createOrUpdateDeployment).
		Watches(&appsv1.Deployment{}, &handler.EnqueueRequestForObject{}, createOrUpdateDeployment).
		Build(rm)

	if err != nil {
		return err
	}
	rm.controller = ctrl
	return nil
}

func (rm resourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("stack", req.NamespacedName)
	logger.Info("Reconciling Korrel8r stack")
	korrel8rDeploy := &appsv1.Deployment{}
	if err := rm.k8sClient.Get(ctx, req.NamespacedName, korrel8rDeploy); err != nil {
		if err != nil {
			// retry since some error has occured
			return ctrl.Result{}, err
		}
	}

	korrel8rSvc := &corev1.Service{}
	if err := rm.k8sClient.Get(ctx, req.NamespacedName, korrel8rSvc); err != nil {
		if err != nil {
			// retry since some error has occured
			return ctrl.Result{}, err
		}
	}

	// korrel8rRoute := &routev1.Route{}
	// if err := rm.k8sClient.Get(ctx, req.NamespacedName, korrel8rRoute); err != nil {
	// 	if err != nil {
	// 		// retry since some error has occured
	// 		return ctrl.Result{}, err
	// 	}
	// }

	reconcilers := korrel8rComponentReconcilers(korrel8rDeploy, korrel8rSvc, rm.korrel8rconf, OperatorNamespace)
	for _, reconciler := range reconcilers {
		err := reconciler.Reconcile(ctx, rm.k8sClient, rm.scheme)
		// handle creation / updation errors that can happen due to a stale cache by
		// retrying after some time.
		if apierrors.IsAlreadyExists(err) || apierrors.IsConflict(err) {
			logger.V(8).Info("skipping reconcile error", "err", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
