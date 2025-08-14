package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

const (
	finalizerName = "observability.openshift.io/clusterobservability"

	conditionReasonError    = "ReconcileError"
	conditionTypeReconciled = "Reconciled"
)

// RBAC for the ClusterObservability CRD
// +kubebuilder:rbac:groups=observability.openshift.io,resources=clusterobservabilities,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=clusterobservabilities/status;clusterobservabilities/finalizers,verbs=get;update;delete;patch

// RBAC for installing operators
// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;list;watch;create;update;patch;delete

// RBAC for OTEL
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/status,verbs=get;list;watch

// RBAC for Tempo
// +kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces;secrets;configmaps,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=tempo.grafana.com,resources=prod,resourceNames=traces,verbs=create

type clusterObservabilityController struct {
	client          client.Client
	scheme          *runtime.Scheme
	logger          logr.Logger
	Options         Options
	controller      controller.TypedController[reconcile.Request]
	cache           cache.Cache
	discoveryClient *discovery.DiscoveryClient
	watchOTELcol    *sync.Once
	watchTempo      *sync.Once
}

var _ reconcile.TypedReconciler[reconcile.Request] = (*clusterObservabilityController)(nil)

func (o clusterObservabilityController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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
		if controllerutil.AddFinalizer(instance, finalizerName) {
			err := o.client.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	csvs := &olmv1alpha1.ClusterServiceVersionList{}
	err = o.client.List(ctx, csvs, &client.ListOptions{Namespace: o.Options.COONamespace})
	if err != nil {
		o.logger.Error(err, "Failed to list csvs")
		return ctrl.Result{}, err
	}
	subs := &olmv1alpha1.SubscriptionList{}
	err = o.client.List(ctx, subs, &client.ListOptions{})
	if err != nil {
		o.logger.Error(err, "Failed to list subscriptions")
		return ctrl.Result{}, err
	}
	reconcilers, err := getReconcilers(ctx, o.client, instance, o.Options, operatorsStatus{
		cooNamespace: o.Options.COONamespace,
		csvs:         csvs.Items,
		subs:         subs.Items,
	})
	if err != nil {
		o.logger.Error(err, "Failed to get reconcilers")
		return ctrl.Result{}, err
	}
	for _, reconciler := range reconcilers {
		reconcileErr := reconciler.Reconcile(ctx, o.client, o.scheme)
		// handle creation / update errors that can happen due to a stale cache by
		// retrying after some time.
		if apierrors.IsAlreadyExists(err) || apierrors.IsConflict(err) {
			o.logger.V(1).Info("skipping reconcile error", "err", reconcileErr)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		if reconcileErr != nil {
			o.logger.Error(reconcileErr, "Failed to reconcile")
			return o.updateStatus(ctx, instance, err), err
		}
	}

	groups, err := o.discoveryClient.ServerGroups()
	if err != nil {
		o.logger.Error(err, "Failed to get server groups / CDRs")
		return ctrl.Result{}, fmt.Errorf("failed to get server groups: %w", err)
	}
	for _, group := range groups.Groups {
		if group.Name == "opentelemetry.io" {
			o.watchOTELcol.Do(func() {
				if err := o.controller.Watch(source.Kind[client.Object](o.cache, &otelv1beta1.OpenTelemetryCollector{}, handler.EnqueueRequestsFromMapFunc(o.triggerReconcile))); err != nil {
					o.logger.Error(err, "Failed to watch OpenTelemetryCollector resources")
				}
			})
		}

		if group.Name == "tempo.grafana.com" {
			o.watchTempo.Do(func() {
				if err := o.controller.Watch(source.Kind[client.Object](o.cache, &tempov1alpha1.TempoStack{}, handler.EnqueueRequestsFromMapFunc(o.triggerReconcile))); err != nil {
					o.logger.Error(err, "Failed to watch TempoStack resources")
				}
			})
		}
	}

	// We have a deletion, short circuit and let the deletion happen
	if instance.ObjectMeta.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(instance, finalizerName) {
			// Once all finalizers have been
			// removed, the object will be deleted.
			if controllerutil.RemoveFinalizer(instance, finalizerName) {
				err := o.client.Update(ctx, instance)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	return o.updateStatus(ctx, instance, nil), nil
}

func (o clusterObservabilityController) triggerReconcile(ctx context.Context, _ client.Object) []reconcile.Request {
	instances := &obsv1alpha1.ClusterObservabilityList{}
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

func (o clusterObservabilityController) getInstance(ctx context.Context, req ctrl.Request) (*obsv1alpha1.ClusterObservability, error) {
	instance := obsv1alpha1.ClusterObservability{}
	if err := o.client.Get(ctx, req.NamespacedName, &instance); err != nil {
		if apierrors.IsNotFound(err) {
			o.logger.V(3).Info("instance could not be found; may be marked for deletion")
			return nil, nil
		}
		o.logger.Error(err, "failed to get cluster observability instance")
		return nil, err
	}

	return &instance, nil
}

func (o clusterObservabilityController) updateStatus(ctx context.Context, instance *obsv1alpha1.ClusterObservability, reconcileErr error) reconcile.Result {
	if instance.Spec.Capabilities != nil {
		capabilities := instance.Spec.Capabilities
		if capabilities.Tracing.Enabled {
			otelcol := &otelv1beta1.OpenTelemetryCollector{}
			err := o.client.Get(ctx, types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      otelCollectorName(instance.Name),
			}, otelcol)
			if err != nil {
				return ctrl.Result{RequeueAfter: 2 * time.Second}
			}
			tempo := &tempov1alpha1.TempoStack{}
			err = o.client.Get(ctx, types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      tempoName(instance.Name),
			}, tempo)
			if err != nil {
				return ctrl.Result{RequeueAfter: 2 * time.Second}
			}

			instance.Status.Tempo = fmt.Sprintf("%s/%s (%s)", instance.Namespace, tempoName(instance.Name), tempo.Status.TempoVersion)
			instance.Status.OpenTelemetry = fmt.Sprintf("%s/%s (%s)", instance.Namespace, otelCollectorName(instance.Name), otelcol.Status.Version)
		}
	} else {
		instance.Status.Tempo = ""
		instance.Status.OpenTelemetry = ""
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
	}

	err := o.client.Status().Update(ctx, instance)
	if err != nil {
		o.logger.Error(err, "failed to update status")
		return ctrl.Result{RequeueAfter: 2 * time.Second}
	}

	return ctrl.Result{}
}

type Options struct {
	COONamespace          string
	OpenTelemetryOperator OperatorInstallConfig
	TempoOperator         OperatorInstallConfig
}

type OperatorInstallConfig struct {
	Namespace   string
	PackageName string
	StartingCSV string
	Channel     string
}

func RegisterWithManager(mgr ctrl.Manager, opts Options) error {
	logger := ctrl.Log.WithName("cluster-observability")

	// TODO remove once the ClusterObservability feature is tech-preview
	// Check if the ClusterObservability CRD is installed, if not, do not install the controller
	clObs := &obsv1alpha1.ClusterObservability{}
	getClObsErr := mgr.GetAPIReader().Get(context.Background(), types.NamespacedName{
		Name:      "does-not-exist-on-purpose",
		Namespace: opts.COONamespace,
	}, clObs)
	if !apierrors.IsNotFound(getClObsErr) {
		logger.Info("not installing cluster observability controller, ClusterObservability CRD is not installed")
		return nil
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	controller := &clusterObservabilityController{
		client:          mgr.GetClient(),
		scheme:          mgr.GetScheme(),
		logger:          logger,
		Options:         opts,
		watchOTELcol:    &sync.Once{},
		watchTempo:      &sync.Once{},
		discoveryClient: discoveryClient,
		cache:           mgr.GetCache(),
	}

	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&obsv1alpha1.ClusterObservability{}).
		Owns(&olmv1alpha1.Subscription{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Namespace{}).
		Owns(&uiv1alpha1.UIPlugin{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Named("cluster-observability")

	controller.controller, err = ctrlBuilder.Build(controller)
	if err != nil {
		return err
	}
	return nil
}
