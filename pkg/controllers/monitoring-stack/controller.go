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

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	stack "rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"

	"github.com/go-logr/logr"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type reconciler struct {
	k8sClient             client.Client
	scheme                *runtime.Scheme
	logger                logr.Logger
	instanceSelectorKey   string
	instanceSelectorValue string
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
	}

	// We only want to trigger a reconciliation when the generation
	// of a child changes. Until we need to update our the status for our own objects,
	// we can save CPU cycles by avoiding reconciliations triggered by
	// child status changes.
	p := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&stack.MonitoringStack{}).
		Owns(&monv1.Prometheus{}).WithEventFilter(p).
		Owns(&v1.ServiceAccount{}).WithEventFilter(p).
		Owns(&rbacv1.Role{}).WithEventFilter(p).
		Owns(&rbacv1.RoleBinding{}).WithEventFilter(p).
		Owns(&monv1.ServiceMonitor{}).WithEventFilter(p).
		Complete(r)
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

	for _, patcher := range patchers {
		if err := r.reconcileObject(ctx, ms, patcher); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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
	logger := r.logger.WithValues(
		"stack", ms.Namespace+"/"+ms.Name,
		"GVK", existing.GetObjectKind().GroupVersionKind(),
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
