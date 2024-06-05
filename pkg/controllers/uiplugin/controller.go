package uiplugin

import (
	"context"
	"slices"
	"time"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaerrors "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/rhobs/observability-operator/pkg/apis/shared"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type resourceManager struct {
	k8sClient      client.Client
	scheme         *runtime.Scheme
	logger         logr.Logger
	controller     controller.Controller
	pluginConf     UIPluginsConfiguration
	clusterVersion string
}

type UIPluginsConfiguration struct {
	Images             map[string]string
	ResourcesNamespace string
}

type Options struct {
	PluginsConf UIPluginsConfiguration
}

const (
	AvailableReason         = "UIPluginAvailable"
	ReconciledReason        = "UIPluginReconciled"
	FailedToReconcileReason = "UIPluginFailedToReconciled"
	ReconciledMessage       = "Plugin reconciled successfully"
	NoReason                = "None"
)

// RBAC for managing UIPlugins
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins/status,verbs=get;update

// RBAC for managing observability ui plugin objects
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=list;watch;create;update;delete;patch
//+kubebuilder:rbac:groups="",resources=serviceaccounts;services;configmaps,verbs=get;list;watch;create;update;patch;delete

// RBAC for managing Console CRs
// +kubebuilder:rbac:groups=operator.openshift.io,resources=consoles,verbs=get;patch;list;watch
// +kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;list;watch;create;update;delete;patch

// RBAC for reading cluster version
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch

func RegisterWithManager(mgr ctrl.Manager, opts Options) error {
	logger := ctrl.Log.WithName("observability-ui")

	clusterVersion, err := getClusterVersion(mgr.GetAPIReader())

	if err != nil {
		logger.Error(err, "failed to get cluster version")
		return err
	}

	rm := &resourceManager{
		k8sClient:      mgr.GetClient(),
		scheme:         mgr.GetScheme(),
		logger:         logger,
		pluginConf:     opts.PluginsConf,
		clusterVersion: clusterVersion.Status.Desired.Version,
	}

	generationChanged := builder.WithPredicates(predicate.GenerationChangedPredicate{})

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		For(&uiv1alpha1.UIPlugin{}).
		Owns(&appsv1.Deployment{}, generationChanged).
		Owns(&v1.Service{}, generationChanged).
		Owns(&v1.ServiceAccount{}, generationChanged).
		Owns(&rbacv1.Role{}, generationChanged).
		Owns(&rbacv1.RoleBinding{}, generationChanged).
		Owns(&osv1alpha1.ConsolePlugin{}, generationChanged).
		Build(rm)
	if err != nil {
		return err
	}
	rm.controller = ctrl

	return nil
}

func getClusterVersion(k8client client.Reader) (*configv1.ClusterVersion, error) {
	clusterVersion := &configv1.ClusterVersion{}
	key := client.ObjectKey{Name: "version"}
	if err := k8client.Get(context.TODO(), key, clusterVersion); err != nil {
		return nil, err
	}
	return clusterVersion, nil
}

func (rm resourceManager) consolePluginCapabilityEnabled(ctx context.Context, name types.NamespacedName) bool {
	current := &osv1alpha1.ConsolePlugin{}
	err := rm.k8sClient.Get(ctx, name, current)

	return err == nil || !metaerrors.IsNoMatchError(err)
}

func (rm resourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("plugin", req.NamespacedName)

	if !rm.consolePluginCapabilityEnabled(ctx, req.NamespacedName) {
		logger.Info("Cluster console plugin not supported or not accessible. Skipping observability UI plugin reconciliation")
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling observability UI plugin")

	plugin, err := rm.getUIPlugin(ctx, req)
	if err != nil {
		// retry since some error has occured
		return ctrl.Result{}, err
	}

	if plugin == nil {
		// no such obs ui plugin, so stop here
		return ctrl.Result{}, nil
	}

	if !plugin.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.V(6).Info("skipping reconcile since object is already schedule for deletion")
		return ctrl.Result{}, nil
	}

	pluginInfo, err := PluginInfoBuilder(plugin, rm.pluginConf, rm.clusterVersion)

	if err != nil {
		logger.Error(err, "failed to reconcile plugin")
		return ctrl.Result{}, err
	}

	reconcilers := pluginComponentReconcilers(plugin, *pluginInfo)
	for _, reconciler := range reconcilers {
		err := reconciler.Reconcile(ctx, rm.k8sClient, rm.scheme)
		// handle creation / updation errors that can happen due to a stale cache by
		// retrying after some time.
		if apierrors.IsAlreadyExists(err) || apierrors.IsConflict(err) {
			logger.V(8).Info("skipping reconcile error", "err", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		if err != nil {
			return rm.updateStatus(ctx, req, plugin, err), err
		}
	}

	if err := rm.registerPluginWithConsole(ctx, pluginInfo); err != nil {
		return rm.updateStatus(ctx, req, plugin, err), err
	}

	return rm.updateStatus(ctx, req, plugin, nil), nil
}

func (rm resourceManager) updateStatus(ctx context.Context, req ctrl.Request, pl *uiv1alpha1.UIPlugin, recError error) ctrl.Result {
	logger := rm.logger.WithValues("plugin", req.NamespacedName)

	if recError != nil {
		pl.Status.Conditions = []shared.Condition{
			{
				Type:               shared.ReconciledCondition,
				Status:             shared.ConditionFalse,
				Reason:             FailedToReconcileReason,
				Message:            recError.Error(),
				ObservedGeneration: pl.Generation,
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               shared.AvailableCondition,
				Status:             shared.ConditionFalse,
				Reason:             FailedToReconcileReason,
				ObservedGeneration: pl.Generation,
				LastTransitionTime: metav1.Now(),
			},
		}
	} else {
		pl.Status.Conditions = []shared.Condition{
			{
				Type:               shared.ReconciledCondition,
				Status:             shared.ConditionTrue,
				Reason:             ReconciledReason,
				Message:            ReconciledMessage,
				ObservedGeneration: pl.Generation,
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               shared.AvailableCondition,
				Status:             shared.ConditionTrue,
				Reason:             AvailableReason,
				ObservedGeneration: pl.Generation,
				LastTransitionTime: metav1.Now(),
			},
		}
	}

	err := rm.k8sClient.Status().Update(ctx, pl)
	if err != nil {
		logger.Info("Failed to update status", "err", err)
		return ctrl.Result{RequeueAfter: 2 * time.Second}
	}

	return ctrl.Result{}
}

func (rm resourceManager) registerPluginWithConsole(ctx context.Context, pluginInfo *UIPluginInfo) error {
	cluster := &operatorv1.Console{}
	if err := rm.k8sClient.Get(ctx, client.ObjectKey{Name: "cluster"}, cluster); err != nil {
		return err
	}

	if slices.Contains(cluster.Spec.Plugins, pluginInfo.ConsoleName) {
		return nil
	}

	clusterPlugins := append(cluster.Spec.Plugins, pluginInfo.ConsoleName)

	// Register the plugin with the console
	cluster = &operatorv1.Console{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorv1.GroupVersion.String(),
			Kind:       "Console",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: operatorv1.ConsoleSpec{
			OperatorSpec: operatorv1.OperatorSpec{
				ManagementState: operatorv1.Managed,
			},
			Plugins: clusterPlugins,
		},
	}

	if err := reconciler.NewMerger(cluster).Reconcile(ctx, rm.k8sClient, rm.scheme); err != nil {
		return err
	}

	return nil
}

func (rm resourceManager) getUIPlugin(ctx context.Context, req ctrl.Request) (*uiv1alpha1.UIPlugin, error) {
	logger := rm.logger.WithValues("plugin", req.NamespacedName)

	plugin := uiv1alpha1.UIPlugin{}

	if err := rm.k8sClient.Get(ctx, req.NamespacedName, &plugin); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(3).Info("stack could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get UIPlugin")
		return nil, err
	}

	return &plugin, nil
}
