/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitoringstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	policyv1 "k8s.io/api/policy/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	stack "github.com/rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"

	"github.com/go-logr/logr"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

const (
	grafanaDatasourceOwnerName      = "monitoring-stack-operator/owner-name"
	grafanaDatasourceOwnerNamespace = "monitoring-stack-operator/owner-namespace"
	finalizerName                   = "monitoring-stack-grafana-ds/finalizer"
)

type reconciler struct {
	k8sClient             client.Client
	scheme                *runtime.Scheme
	logger                logr.Logger
	instanceSelectorKey   string
	instanceSelectorValue string
	grafanaDSWatchCreated bool
	controller            controller.Controller
}

// Options allows for controller options to be set
type Options struct {
	InstanceSelector string
}

// RBAC for managing monitoring stacks
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;watch;create;update
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/status,verbs=get;update

// RBAC for managing Prometheus Operator CRs
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=alertmanagers;prometheuses;servicemonitors,verbs=list;watch;create;update;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts;services;secrets,verbs=list;watch;create;update;delete
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=list;watch;create;update

// RBAC for managing Grafana CRs
//+kubebuilder:rbac:groups=integreatly.org,namespace=monitoring-stack-operator,resources=grafanadatasources,verbs=list;watch;create;update;delete

// RBAC for delegating permissions to Prometheus
//+kubebuilder:rbac:groups="",resources=pods;services;endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=extensions;networking.k8s.io,resources=ingresses,verbs=get;list;watch

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager, opts Options) error {
	split := strings.Split(opts.InstanceSelector, "=")
	if len(split) != 2 {
		return fmt.Errorf("invalid InstanceSelector: %s", opts.InstanceSelector)
	}

	r := &reconciler{
		k8sClient:             mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		logger:                ctrl.Log.WithName("monitoring-stack-operator"),
		instanceSelectorKey:   split[0],
		instanceSelectorValue: split[1],
		grafanaDSWatchCreated: false,
	}

	// We only want to trigger a reconciliation when the generation
	// of a child changes. Until we need to update our the status for our own objects,
	// we can save CPU cycles by avoiding reconciliations triggered by
	// child status changes.
	p := predicate.GenerationChangedPredicate{}

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		WithLogger(ctrl.Log).
		For(&stack.MonitoringStack{}).
		Owns(&monv1.Prometheus{}).WithEventFilter(p).
		Owns(&v1.Service{}).WithEventFilter(p).
		Owns(&v1.ServiceAccount{}).WithEventFilter(p).
		Owns(&rbacv1.Role{}).WithEventFilter(p).
		Owns(&rbacv1.RoleBinding{}).WithEventFilter(p).
		Owns(&monv1.ServiceMonitor{}).WithEventFilter(p).
		Owns(&policyv1.PodDisruptionBudget{}).WithEventFilter(p).
		Build(r)

	if err != nil {
		return err
	}
	r.controller = ctrl
	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.WithValues("stack", req.NamespacedName)
	logger.Info("Reconciling monitoring stack")
	ms, err := r.getStack(ctx, req)
	if err != nil {
		// retry since some error has occured
		return ctrl.Result{}, err
	}
	if ms == nil {
		// no such monitoring stack, so stop here
		return ctrl.Result{}, nil
	}

	patchers, err := stackComponentPatchers(ms, r.instanceSelectorKey, r.instanceSelectorValue)
	if err != nil {
		return ctrl.Result{}, err
	}
	if requeue, err := r.createGrafanaDSWatch(ctx); err != nil {
		return ctrl.Result{}, err
	} else if requeue {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if !ms.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.cleanupResources(ctx, ms)
	}

	for _, patcher := range patchers {
		err := r.reconcileObject(ctx, ms, patcher)
		// handle creation / updation errors that can happen due to a stale cache by
		// retrying after some time.
		if errors.IsAlreadyExists(err) || errors.IsConflict(err) {
			logger.V(8).Info("skipping reconcile error", "err", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return r.setupFinalizer(ctx, ms)
}

func (r *reconciler) deleteGrafanaDS(ctx context.Context, ms *stack.MonitoringStack) error {
	logger := r.logger.WithValues(stackName(ms)...)

	gds := types.NamespacedName{Namespace: ms.Namespace, Name: GrafanaDSName(ms)}
	grafanaDS := grafanav1alpha1.GrafanaDataSource{}
	if err := r.k8sClient.Get(ctx, gds, &grafanaDS); err != nil {
		// if the datasource is already deleted, take no further action
		return client.IgnoreNotFound(err)
	}

	// grafana ds exists; so delete it
	logger.WithValues("GrafanaDataSource", grafanaDS.Name).Info("Deleting GrafanaDataSource")
	err := r.k8sClient.Delete(ctx, &grafanaDS)
	return client.IgnoreNotFound(err)
}

func (r *reconciler) cleanupResources(ctx context.Context, ms *stack.MonitoringStack) (ctrl.Result, error) {
	logger := r.logger.WithValues(stackName(ms)...)

	if !controllerutil.ContainsFinalizer(ms, finalizerName) {
		logger.V(6).Info("Finalizer already removed")
		return ctrl.Result{}, nil
	}

	if err := r.deleteGrafanaDS(ctx, ms); err != nil {
		logger.V(6).Info("Could not delete GrafanaDataSource", "err", err)
		return ctrl.Result{}, err
	}

	logger.Info("Removing finalizer")
	controllerutil.RemoveFinalizer(ms, finalizerName)
	err := r.k8sClient.Update(ctx, ms)
	if errors.IsConflict(err) {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return ctrl.Result{}, err
}

func (r *reconciler) setupFinalizer(ctx context.Context, ms *stack.MonitoringStack) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(ms, finalizerName) {
		return ctrl.Result{}, nil
	}
	controllerutil.AddFinalizer(ms, finalizerName)
	if err := r.k8sClient.Update(ctx, ms); err != nil {
		if errors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func GrafanaDSName(ms *stack.MonitoringStack) string {
	return fmt.Sprintf("ms-%s-%s", ms.Namespace, ms.Name)
}

func (r *reconciler) getStack(ctx context.Context, req ctrl.Request) (*stack.MonitoringStack, error) {
	logger := r.logger.WithValues("stack", req.NamespacedName)

	ms := stack.MonitoringStack{}

	if err := r.k8sClient.Get(ctx, req.NamespacedName, &ms); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("stack could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get monitoring stack")
		return nil, err
	}

	return &ms, nil
}

func (r *reconciler) reconcileObject(ctx context.Context, ms *stack.MonitoringStack, patcher objectPatcher) error {
	existing := patcher.empty()
	gvk := existing.GetObjectKind().GroupVersionKind()
	logger := r.logger.WithValues(
		"Stack", ms.Namespace+"/"+ms.Name,
		"Component", fmt.Sprintf("%s.%s/%s", gvk.Kind, gvk.Group, gvk.Version),
		"Name", existing.GetName())

	key := types.NamespacedName{
		Name:      existing.GetName(),
		Namespace: existing.GetNamespace(),
	}
	err := r.k8sClient.Get(ctx, key, existing)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	createNew := errors.IsNotFound(err)

	desired, err := patcher.patch(existing)
	if err != nil {
		return err
	}

	if ms.Namespace == desired.GetNamespace() {
		if err := controllerutil.SetControllerReference(ms, desired, r.scheme); err != nil {
			return err
		}
	}

	if createNew {
		logger.Info("Creating stack component")
		return r.k8sClient.Create(ctx, desired)
	}

	logger.Info("Updating stack component")
	return r.k8sClient.Update(ctx, desired)
}

func (r *reconciler) createGrafanaDSWatch(ctx context.Context) (bool, error) {
	if r.grafanaDSWatchCreated {
		return false, nil
	}
	log := r.logger.WithName("create-grafana-ds-watch")
	var dataSources grafanav1alpha1.GrafanaDataSourceList

	if err := r.k8sClient.List(ctx, &dataSources, client.InNamespace("default")); err != nil {
		log.V(6).Info("grafana data source CRD is not defined")
		return true, nil
	}

	if err := r.controller.Watch(
		&source.Kind{Type: &grafanav1alpha1.GrafanaDataSource{}},
		handler.EnqueueRequestsFromMapFunc(
			func(object client.Object) []reconcile.Request {
				name := object.GetAnnotations()[grafanaDatasourceOwnerName]
				namespace := object.GetAnnotations()[grafanaDatasourceOwnerNamespace]
				namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
				return []reconcile.Request{{NamespacedName: namespacedName}}
			},
		),
	); err != nil {
		log.Error(err, "unable to create watch on grafana data source")
		return false, err
	}

	log.V(6).Info("Created watch on Grafana datasource")
	r.grafanaDSWatchCreated = true

	return false, nil

}

func stackName(ms *stack.MonitoringStack) []interface{} {
	return []interface{}{"Stack", ms.Namespace + "/" + ms.Name}
}
