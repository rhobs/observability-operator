package uiplugin

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	osv1 "github.com/openshift/api/console/v1"
	osv1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	mchv1 "github.com/stolostron/multiclusterhub-operator/api/v1"
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
// +kubebuilder:rbac:groups=observability.openshift.io,resources=uiplugins/status;uiplugins/finalizers,verbs=get;update

// RBAC for managing observability ui plugin objects
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=list;watch;create;update;delete;patch
//+kubebuilder:rbac:groups="",resources=serviceaccounts;services;configmaps,verbs=get;list;watch;create;update;patch;delete

// RBAC for managing Console CRs
// +kubebuilder:rbac:groups=operator.openshift.io,resources=consoles,verbs=get;patch;list;watch
// +kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;list;watch;create;update;delete;patch

// RBAC for reading cluster version
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch

// RBAC for distributed tracing
// +kubebuilder:rbac:groups=tempo.grafana.com,resources=tempostacks;tempomonolithics,verbs=list

// RBAC for monitoring
// +kubebuilder:rbac:groups=operator.open-cluster-management.io,resources=multiclusterhubs,verbs=get;list;watch

// RBAC for logging view plugin
// +kubebuilder:rbac:groups=loki.grafana.com,resources=application;infrastructure;audit,verbs=get

// RBAC for korrel8r
//+kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;replicasets;statefulsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps;endpoints;events;namespaces;nodes;persistentvolumeclaims;persistentvolumes;pods;replicationcontrollers;secrets;serviceaccounts;services,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=cronjobs;jobs,verbs=get;list;watch
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses;volumeattachments,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies;ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=loki.grafana.com,resources=application;infrastructure;audit;network,verbs=get
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheuses/api,resourceNames=k8s,verbs=get;create;update
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=alertmanagers/api,resourceNames=main,verbs=get;list

const finalizerName = "uiplugin.observability.openshift.io/finalizer"

// RegisterWithManager registers the controller with Manager
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

	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&uiv1alpha1.UIPlugin{}).
		Owns(&appsv1.Deployment{}, generationChanged).
		Owns(&v1.Service{}, generationChanged).
		Owns(&v1.ServiceAccount{}, generationChanged).
		Owns(&rbacv1.Role{}, generationChanged).
		Owns(&rbacv1.RoleBinding{}, generationChanged)

	if isVersionAheadOrEqual(rm.clusterVersion, "v4.17") {
		ctrlBuilder.Owns(&osv1.ConsolePlugin{}, generationChanged)
	} else {
		ctrlBuilder.Owns(&osv1alpha1.ConsolePlugin{}, generationChanged)
	}

	ctrl, err := ctrlBuilder.Build(rm)

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

func (rm resourceManager) consolePluginCapabilityEnabled(ctx context.Context, name types.NamespacedName, clusterVersion string) bool {
	var err error

	if isVersionAheadOrEqual(clusterVersion, "v4.17") {
		consolePlugin := &osv1.ConsolePlugin{}
		err = rm.k8sClient.Get(ctx, name, consolePlugin)
	} else {
		legacyConsolePlugin := &osv1alpha1.ConsolePlugin{}
		err = rm.k8sClient.Get(ctx, name, legacyConsolePlugin)
	}

	return err == nil || !metaerrors.IsNoMatchError(err)
}

func (rm resourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("plugin", req.NamespacedName)

	if !rm.consolePluginCapabilityEnabled(ctx, req.NamespacedName, rm.clusterVersion) {
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

	// Check if the plugin is being deleted
	if !plugin.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.V(6).Info("deregistering plugin from the console")
		if err := rm.deregisterPluginFromConsole(ctx, pluginTypeToConsoleName[plugin.Spec.Type]); err != nil {
			return ctrl.Result{}, err
		}

		// Remove finalizer if present
		if slices.Contains(plugin.ObjectMeta.Finalizers, finalizerName) {
			plugin.ObjectMeta.Finalizers = slices.DeleteFunc(plugin.ObjectMeta.Finalizers, func(currentFinalizerName string) bool {
				return currentFinalizerName == finalizerName
			})
			if err := rm.k8sClient.Update(ctx, plugin); err != nil {
				return ctrl.Result{}, err
			}
		}

		logger.V(6).Info("skipping reconcile since object is already scheduled for deletion")
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !slices.Contains(plugin.ObjectMeta.Finalizers, finalizerName) {
		plugin.ObjectMeta.Finalizers = append(plugin.ObjectMeta.Finalizers, finalizerName)
		if err := rm.k8sClient.Update(ctx, plugin); err != nil {
			return ctrl.Result{}, err
		}
	}

	multiClusterHubList := &mchv1.MultiClusterHubList{}
	acmVersion := "acm version not found"
	err = rm.k8sClient.List(ctx, multiClusterHubList, &client.ListOptions{})

	// Multiple MultiClusterHub's are undefined behavior
	if err == nil && len(multiClusterHubList.Items) == 1 {
		multiClusterHub := multiClusterHubList.Items[0]
		acmVersion = multiClusterHub.Status.CurrentVersion
		if !strings.HasPrefix(acmVersion, "v") {
			acmVersion = "v" + acmVersion
		}
	}

	compatibilityInfo, err := lookupImageAndFeatures(plugin.Spec.Type, rm.clusterVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	if plugin.Annotations == nil {
		plugin.Annotations = map[string]string{}
		plugin.Annotations["observability.openshift.io/api-support"] = string(compatibilityInfo.SupportLevel)

		if err := rm.k8sClient.Update(ctx, plugin); err != nil {
			return ctrl.Result{}, err
		}
		// Upon changes to the uiplugin, reconciliation will happen anyways, so just return early
		// and let the reconciliation handle the further changes
		return ctrl.Result{}, nil
	} else if plugin.Annotations["observability.openshift.io/api-support"] != string(compatibilityInfo.SupportLevel) {
		plugin.Annotations["observability.openshift.io/api-support"] = string(compatibilityInfo.SupportLevel)

		if err := rm.k8sClient.Update(ctx, plugin); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	pluginInfo, err := PluginInfoBuilder(ctx, rm.k8sClient, plugin, rm.pluginConf, compatibilityInfo, acmVersion)

	if err != nil {
		logger.Error(err, "failed to reconcile plugin")
		return ctrl.Result{}, err
	}

	reconcilers := pluginComponentReconcilers(plugin, *pluginInfo, rm.clusterVersion)
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
		pl.Status.Conditions = []uiv1alpha1.Condition{
			{
				Type:               uiv1alpha1.ReconciledCondition,
				Status:             uiv1alpha1.ConditionFalse,
				Reason:             FailedToReconcileReason,
				Message:            recError.Error(),
				ObservedGeneration: pl.Generation,
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               uiv1alpha1.AvailableCondition,
				Status:             uiv1alpha1.ConditionFalse,
				Reason:             FailedToReconcileReason,
				ObservedGeneration: pl.Generation,
				LastTransitionTime: metav1.Now(),
			},
		}
	} else {
		pl.Status.Conditions = []uiv1alpha1.Condition{
			{
				Type:               uiv1alpha1.ReconciledCondition,
				Status:             uiv1alpha1.ConditionTrue,
				Reason:             ReconciledReason,
				Message:            ReconciledMessage,
				ObservedGeneration: pl.Generation,
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               uiv1alpha1.AvailableCondition,
				Status:             uiv1alpha1.ConditionTrue,
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
	cluster.Spec.Plugins = clusterPlugins

	if err := reconciler.NewMerger(cluster).Reconcile(ctx, rm.k8sClient, rm.scheme); err != nil {
		return err
	}

	return nil
}

func (rm resourceManager) deregisterPluginFromConsole(ctx context.Context, pluginConsoleName string) error {
	cluster := &operatorv1.Console{}
	if err := rm.k8sClient.Get(ctx, client.ObjectKey{Name: "cluster"}, cluster); err != nil {
		return err
	}

	if !slices.Contains(cluster.Spec.Plugins, pluginConsoleName) {
		return nil
	}

	clusterPlugins := slices.DeleteFunc(cluster.Spec.Plugins, func(currentPluginName string) bool {
		return currentPluginName == pluginConsoleName
	})

	// Deregister the plugin from the console
	cluster.Spec.Plugins = clusterPlugins

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
