/*
Copyright 2024.

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

package operator_controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type resourceManager struct {
	k8sClient  client.Client
	scheme     *runtime.Scheme
	logger     logr.Logger
	controller controller.Controller
	namespace  string
}

// RBAC for managing Prometheus Operator CRs
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=list;watch;create;update;delete;patch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=list;create;update;patch

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager, namespace string) error {

	rm := &resourceManager{
		k8sClient: mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		logger:    ctrl.Log.WithName("observability-operator"),
		namespace: namespace,
	}
	// We only want to trigger a reconciliation when the generation
	// of a child changes. Until we need to update our the status for our own objects,
	// we can save CPU cycles by avoiding reconciliations triggered by
	// child status changes.
	generationChanged := builder.WithPredicates(predicate.GenerationChangedPredicate{})

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		Owns(&monv1.ServiceMonitor{}, generationChanged).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(rm.operatorDeployment),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Build(rm)

	if err != nil {
		return err
	}
	rm.controller = ctrl
	return nil
}

func (rm resourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("operator", req.NamespacedName)
	logger.Info("Reconciling operator resources")

	op := &appsv1.Deployment{}
	err := rm.k8sClient.Get(ctx, req.NamespacedName, op)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	reconcilers := operatorComponentReconcilers(op, rm.namespace)
	for _, reconciler := range reconcilers {
		err := reconciler.Reconcile(ctx, rm.k8sClient, rm.scheme)
		// handle create / update errors that can happen due to a stale cache by
		// retrying after some time.
		if errors.IsAlreadyExists(err) || errors.IsConflict(err) {
			logger.V(3).Info("skipping reconcile error", "err", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
func (rm resourceManager) operatorDeployment(ctx context.Context, ms client.Object) []reconcile.Request {
	var requests []reconcile.Request
	op := &appsv1.Deployment{}
	err := rm.k8sClient.Get(ctx, types.NamespacedName{Name: "observability-operator", Namespace: rm.namespace}, op)
	if errors.IsNotFound(err) {
		return requests
	}
	if err != nil {
		return requests
	}
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      op.GetName(),
			Namespace: op.GetNamespace(),
		},
	})
	return requests
}
