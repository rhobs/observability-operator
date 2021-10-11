package prometheus_operator

import (
	"context"
	"fmt"
	"rhobs/monitoring-stack-operator/pkg/assets"

	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	appsv "k8s.io/api/apps/v1"
	authorizationv1 "k8s.io/api/rbac/v1"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Options struct {
	Namespace  string
	AssetsPath string
	DeployCRDs bool
}

type reconciler struct {
	logger      logr.Logger
	k8sClient   client.Client
	assetLoader *assets.Loader
	opts        Options
}

func RegisterWithManager(mgr manager.Manager, opts Options) error {
	logger := ctrlruntime.Log.WithName("prometheus-operator-controller")
	reconciler := reconciler{
		logger:      logger,
		k8sClient:   mgr.GetClient(),
		assetLoader: assets.NewLoader(opts.AssetsPath),
		opts:        opts,
	}

	ctrl, err := controller.New("prometheus-operator", mgr, controller.Options{
		Log:        logger,
		Reconciler: &reconciler,
	})
	if err != nil {
		return err
	}

	ticker := newTickerSource()
	if err := ctrl.Watch(ticker, &handler.EnqueueRequestForObject{}); err != nil {
		return nil
	}

	go ticker.run()

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request ctrlruntime.Request) (ctrlruntime.Result, error) {
	r.logger.Info("Reconciling prometheus operator")

	resources, err := r.loadStaticResources()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to load resources: %w", err)
	}

	fieldOwner := client.FieldOwner("monitoring-stack-operator")
	for _, resource := range resources {
		resource.SetNamespace(r.opts.Namespace)
		r.logger.Info("Reconciling resource",
			"Kind", resource.GetObjectKind().GroupVersionKind().Kind,
			"Name", resource.GetName())
		if err := r.k8sClient.Patch(ctx, resource, client.Apply, fieldOwner); err != nil {
			return reconcile.Result{}, err
		}
	}

	return ctrlruntime.Result{}, nil
}

func (r *reconciler) loadStaticResources() ([]client.Object, error) {
	resources := []assets.Asset{
		{
			File:   "service-account.yaml",
			Object: &v1.ServiceAccount{},
		},
		{
			File:   "cluster-role.yaml",
			Object: &authorizationv1.ClusterRole{},
		},
		{
			File:   "cluster-role-binding.yaml",
			Object: &authorizationv1.ClusterRoleBinding{},
		},
		{
			File:   "deployment.yaml",
			Object: &appsv.Deployment{},
		},
	}

	if r.opts.DeployCRDs {
		crds := []assets.Asset{
			assets.NewCRDAsset("crds/alertmanagerconfigs.yaml"),
			assets.NewCRDAsset("crds/alertmanagers.yaml"),
			assets.NewCRDAsset("crds/podmonitors.yaml"),
			assets.NewCRDAsset("crds/probes.yaml"),
			assets.NewCRDAsset("crds/prometheuses.yaml"),
			assets.NewCRDAsset("crds/prometheusrules.yaml"),
			assets.NewCRDAsset("crds/servicemonitors.yaml"),
			assets.NewCRDAsset("crds/thanosrulers.yaml"),
		}
		resources = append(crds, resources...)
	}

	return r.assetLoader.Load(resources)
}
