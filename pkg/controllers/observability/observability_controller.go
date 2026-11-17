package observability

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
)

const (
	finalizerName = "observability.openshift.io/observabilityinstaller"

	conditionReasonError    = "ReconcileError"
	conditionTypeReconciled = "Reconciled"
)

// RBAC for the ObservabilityInstaller CRD
// +kubebuilder:rbac:groups=observability.openshift.io,resources=observabilityinstallers,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=observabilityinstallers/status;observabilityinstallers/finalizers,verbs=get;update;delete;patch

// RBAC for installing operators
// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;list;watch;create;update;patch;delete

// RBAC for OTEL
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/status,verbs=get;list;watch

// RBAC for Tempo
// +kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=tempo.grafana.com,resources=application,resourceNames=traces,verbs=create

type observabilityInstallerController struct {
	client client.Client
	// Use the reader to access config maps which are not cached
	apiReader       client.Reader
	scheme          *runtime.Scheme
	logger          logr.Logger
	Options         Options
	controller      controller.TypedController[reconcile.Request]
	cache           cache.Cache
	discoveryClient *discovery.DiscoveryClient
	watches         map[client.Object]*sync.Once
	capabilities    []capability.Capability
}

var _ reconcile.TypedReconciler[reconcile.Request] = (*observabilityInstallerController)(nil)

func (o observabilityInstallerController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	o.logger.Info("Reconcile called", "request", request)

	instance, err := o.getInstance(ctx, request)
	if err != nil {
		// retry since some error has occurred
		return ctrl.Result{}, err
	}
	if instance == nil {
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(instance, finalizerName) {
		patch := client.MergeFrom(instance.DeepCopy())
		controllerutil.AddFinalizer(instance, finalizerName)
		if err := o.client.Patch(ctx, instance, patch); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
	}

	subs := &olmv1alpha1.SubscriptionList{}
	// List all subscriptions to figure out if the operators are already installed
	err = o.apiReader.List(ctx, subs, &client.ListOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}
	instances := &obsv1alpha1.ObservabilityInstallerList{}
	if err := o.apiReader.List(ctx, instances); err != nil {
		return ctrl.Result{}, err
	}
	reconcilers, err := getReconcilers(ctx, o.apiReader, instance, o.capabilities, operatorsStatus{
		cooNamespace: o.Options.COONamespace,
		subs:         subs.Items,
	}, instances.Items)
	// Planning errors are reported, but must not block independent work.
	var retryConflict bool
	for _, reconciler := range reconcilers {
		reconcileErr := reconciler.Reconcile(ctx, o.client, o.scheme)
		// handle creation / update errors that can happen due to a stale cache by
		// retrying after some time.
		if apierrors.IsAlreadyExists(reconcileErr) || apierrors.IsConflict(reconcileErr) {
			o.logger.V(1).Info("skipping reconcile error", "err", reconcileErr)
			retryConflict = true
			continue
		}
		if reconcileErr != nil {
			err = errors.Join(err, reconcileErr)
		}
	}
	if err != nil {
		return o.updateStatus(ctx, instance, err), err
	}
	if retryConflict {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}

	// No new operand watches are needed while deleting. Complete cleanup before
	// releasing the finalizer, independently of another installer's shared applies.
	if instance.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(instance, finalizerName) {
			patch := client.MergeFrom(instance.DeepCopy())
			controllerutil.RemoveFinalizer(instance, finalizerName)
			if err := o.client.Patch(ctx, instance, patch); err != nil {
				if apierrors.IsNotFound(err) {
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, err
			}
		}
		return o.updateStatus(ctx, instance, nil), nil
	}

	groups, err := o.discoveryClient.ServerGroups()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get server groups: %w", err)
	}
	for _, group := range groups.Groups {
		for watchType, once := range o.watches {
			gvk, err := apiutil.GVKForObject(watchType, o.scheme)
			if err != nil {
				o.logger.Error(err, "Failed to determine watch type")
				continue
			}
			if group.Name == gvk.Group {
				once.Do(func() {
					if err := o.controller.Watch(source.Kind[client.Object](o.cache, watchType, handler.EnqueueRequestsFromMapFunc(o.triggerReconcile))); err != nil {
						o.logger.Error(err, "Failed to watch resources", "kind", gvk.Kind)
					}
				})
			}
		}
	}

	return o.updateStatus(ctx, instance, nil), nil
}

func (o observabilityInstallerController) triggerReconcile(ctx context.Context, _ client.Object) []reconcile.Request {
	instances := &obsv1alpha1.ObservabilityInstallerList{}
	listOps := &client.ListOptions{}
	err := o.client.List(ctx, instances, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(instances.Items))
	for i, item := range instances.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func (o observabilityInstallerController) getInstance(ctx context.Context, req ctrl.Request) (*obsv1alpha1.ObservabilityInstaller, error) {
	instance := obsv1alpha1.ObservabilityInstaller{}
	if err := o.client.Get(ctx, req.NamespacedName, &instance); err != nil {
		if apierrors.IsNotFound(err) {
			o.logger.V(3).Info("instance could not be found; may be marked for deletion")
			return nil, nil
		}
		return nil, err
	}

	return &instance, nil
}

func (o observabilityInstallerController) updateStatus(ctx context.Context, instance *obsv1alpha1.ObservabilityInstaller, reconcileErr error) reconcile.Result {
	var requeueAfter time.Duration
	for _, capability := range o.capabilities {
		result := capability.UpdateStatus(ctx, instance, o.client)
		if result.RequeueAfter > 0 && (requeueAfter == 0 || result.RequeueAfter < requeueAfter) {
			requeueAfter = result.RequeueAfter
		}
		if reconcileErr == nil && result.Err != nil {
			reconcileErr = fmt.Errorf("evaluate %s status: %w", capability.Name(), result.Err)
		}
	}

	if reconcileErr != nil {
		instance.Status.Conditions = []metav1.Condition{
			{
				Reason:             conditionReasonError,
				Type:               conditionTypeReconciled,
				Status:             metav1.ConditionFalse,
				Message:            reconcileErr.Error(),
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: instance.GetGeneration(),
			},
		}
	} else {
		apimeta.RemoveStatusCondition(&instance.Status.Conditions, conditionTypeReconciled)
	}

	err := o.client.Status().Update(ctx, instance)
	if err != nil {
		o.logger.Error(err, "failed to update status")
		return ctrl.Result{RequeueAfter: 2 * time.Second}
	}

	return ctrl.Result{RequeueAfter: requeueAfter}
}

type Options struct {
	COONamespace          string
	OpenTelemetryOperator OperatorInstallConfig
	TempoOperator         OperatorInstallConfig
}

// OperatorInstallConfig is the OLM configuration accepted by capability constructors.
type OperatorInstallConfig = capability.OperatorConfig

func RegisterWithManager(mgr ctrl.Manager, opts Options) error {
	logger := ctrl.Log.WithName("cluster-observability")

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	controller := &observabilityInstallerController{
		client:          mgr.GetClient(),
		apiReader:       mgr.GetAPIReader(),
		scheme:          mgr.GetScheme(),
		logger:          logger,
		Options:         opts,
		watches:         map[client.Object]*sync.Once{},
		capabilities:    capabilities(opts),
		discoveryClient: discoveryClient,
		cache:           mgr.GetCache(),
	}

	for _, capability := range controller.capabilities {
		for _, watchType := range capability.WatchTypes() {
			controller.watches[watchType] = &sync.Once{}
		}
	}

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		For(&obsv1alpha1.ObservabilityInstaller{}).
		Owns(&olmv1alpha1.Subscription{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Namespace{}).
		Owns(&uiv1alpha1.UIPlugin{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Named("cluster-observability").
		Build(controller)
	if err != nil {
		return err
	}

	controller.controller = ctrl
	return nil
}
