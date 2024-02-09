package observabilityuiplugin

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	operatorv1 "github.com/openshift/api/operator/v1"
	observabilityui "github.com/rhobs/observability-operator/pkg/apis/observabilityui/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type resourceManager struct {
	k8sClient  client.Client
	scheme     *runtime.Scheme
	logger     logr.Logger
	controller controller.Controller
}

// RBAC for managing ObservabilityUIPlugins
// +kubebuilder:rbac:groups=observabilityui.rhobs,resources=observabilityuiplugins,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=observabilityui.rhobs,resources=observabilityuiplugins/status,verbs=get;update

// RBAC for managing Console CRs
// +kubebuilder:rbac:groups=operator.openshift.io,resources=consoles,verbs=get;patch

func RegisterWithStackManager(mgr ctrl.Manager) error {
	rm := &resourceManager{
		k8sClient: mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		logger:    ctrl.Log.WithName("observability-operator").WithName("logging-stack").WithName("custom-resources-operator"),
	}

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		For(&observabilityui.ObservabilityUIPlugin{}).
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

	ls, err := rm.getUIPlugin(ctx, req)
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

	var pluginName string
	switch ls.Spec.Type {
	case observabilityui.TypeLogs:
		pluginName = "logging-view-plugin"
	}

	cluster := &operatorv1.Console{
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
			Plugins: []string{
				pluginName,
			},
		},
	}

	if err := reconciler.NewMerger(cluster).Reconcile(ctx, rm.k8sClient, rm.scheme); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (rm resourceManager) updateStatus(ctx context.Context, req ctrl.Request, plugin *observabilityui.ObservabilityUIPlugin, recError error) ctrl.Result {

	return ctrl.Result{}
}

func (rm resourceManager) getUIPlugin(ctx context.Context, req ctrl.Request) (*observabilityui.ObservabilityUIPlugin, error) {
	logger := rm.logger.WithValues("stack", req.NamespacedName)

	plugin := observabilityui.ObservabilityUIPlugin{}

	if err := rm.k8sClient.Get(ctx, req.NamespacedName, &plugin); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("stack could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get ObservabilityUIPlugin")
		return nil, err

	}

	return &plugin, nil
}
