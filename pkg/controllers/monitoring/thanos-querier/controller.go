/*
Copyright 2022.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package thanos_querier

import (
	"context"
	"fmt"
	"time"

	msoapi "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
)

type resourceManager struct {
	client.Client
	scheme *runtime.Scheme
	logger logr.Logger
}

// RBAC for watching monitoring stacks
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;watch

// RBAC for managing thanosquerier objects
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers/finalizers,verbs=update

// RBAC for managing deployments
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;update;patch;delete

// RBAC for managing core resources
//+kubebuilder:rbac:groups=core,resources=services;serviceaccounts,verbs=list;watch;create;update;patch;delete

// RBAC for managing Prometheus Operator CRs
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=servicemonitors,verbs=list;watch;create;update;patch;delete

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager) error {
	logger := ctrl.Log.WithName("thanos-querier")
	rm := &resourceManager{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		logger: logger,
	}

	p := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&msoapi.ThanosQuerier{}).
		Owns(&appsv1.Deployment{}).WithEventFilter(p).
		Owns(&corev1.ServiceAccount{}).WithEventFilter(p).
		Owns(&corev1.Service{}).WithEventFilter(p).
		Watches(
			&msoapi.MonitoringStack{},
			handler.EnqueueRequestsFromMapFunc(rm.findQueriersForMonitoringStack),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(rm)
}

func (rm resourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("querier", req.NamespacedName)
	logger.Info("Reconciling Thanos Querier")

	querier := &msoapi.ThanosQuerier{}
	err := rm.Get(ctx, req.NamespacedName, querier)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	sidecarServices, err := rm.findSidecarServices(ctx, querier)
	if client.IgnoreNotFound(err) != nil {
		// we encountered an error other then NotFound, don't try to delete
		// resources for this querier and reschedule reconcile
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	reconcilers := thanosComponentReconcilers(querier, sidecarServices)
	for _, reconciler := range reconcilers {
		err := reconciler.Reconcile(ctx, rm, rm.scheme)
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

// Given a ThanosQuerier object, find the matching MonitoringStacks, extract the
// sidecar service and return a list of urls for those sidecar services.
func (rm resourceManager) findSidecarServices(ctx context.Context, tQuerier *msoapi.ThanosQuerier) ([]string, error) {
	logger := rm.logger.WithValues("selector", tQuerier.Spec.Selector)

	msList := &msoapi.MonitoringStackList{}
	selector, _ := metav1.LabelSelectorAsSelector(&tQuerier.Spec.Selector)
	opts := []client.ListOption{
		client.MatchingLabelsSelector{Selector: selector},
	}

	var sidecarUrls []string
	if err := rm.List(ctx, msList, opts...); err != nil {
		logger.Info("Couldn't find any MonitoringStack")
		return sidecarUrls, err
	}
	logger.Info("Found MonitoringStacks list", "length", len(msList.Items))
	for _, ms := range msList.Items {
		if tQuerier.MatchesNamespace(ms.Namespace) {
			serviceName := ms.Name + "-thanos-sidecar"
			sidecarUrls = append(sidecarUrls, getEndpointUrl(serviceName, ms.Namespace))
		}
	}

	return sidecarUrls, nil
}

// Given a Service object, return a url to use as value for --store/--endpoint.
func getEndpointUrl(serviceName string, namespace string) string {
	return fmt.Sprintf("dnssrv+_grpc._tcp.%s.%s.svc.cluster.local", serviceName, namespace)
}

// Find all ThanosQueriers, whose Selector fits the given MonitoringStack and
// return a list of reconcile requests, one for each ThanosQuerier.
func (rm resourceManager) findQueriersForMonitoringStack(ctx context.Context, ms client.Object) []reconcile.Request {
	logger := rm.logger.WithValues("Monitoring Stack", ms.GetNamespace()+"/"+ms.GetName())
	logger.Info("watched MonitoringStack changed, checking for matching querier")
	queriers := &msoapi.ThanosQuerierList{}
	err := rm.List(ctx, queriers, &client.ListOptions{})
	if err != nil {
		logger.Error(err, "Failed to list Thanosqueriers")
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for _, item := range queriers.Items {
		sel, err := metav1.LabelSelectorAsSelector(&item.Spec.Selector)
		if err != nil {
			return []reconcile.Request{}
		}
		if sel.Matches(labels.Set(ms.GetLabels())) {
			logger.Info("Found querier, scheduling sync")
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			})
		}
	}
	return requests
}
