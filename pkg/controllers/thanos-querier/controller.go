/*
Copyright 2021.
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
	"errors"
	"fmt"
	"time"

	msoapi "github.com/rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
)

type reconciler struct {
	client.Client
	scheme *runtime.Scheme
	logger logr.Logger
}

// RBAC for watching monitoring stacks
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=get;list;watch

// RBAC for managing thanosquerier objects
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers/finalizers,verbs=update

// RBAC for managing deployments
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

// RBAC for managing services
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager) error {

	logger := ctrl.Log.WithName("thanos-querier")
	r := &reconciler{
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
			&source.Kind{Type: &corev1.Service{}},
			handler.EnqueueRequestsFromMapFunc(r.findQueriersForService),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&source.Kind{Type: &msoapi.MonitoringStack{}},
			handler.EnqueueRequestsFromMapFunc(r.findQueriersForMonitoringStack),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.WithValues("thanosQuerier", req.NamespacedName)
	logger.Info("Reconciling thanos Querier")

	tQuerier, err := r.getQuerierSpec(ctx, req)
	if err != nil {
		// retry since some error has occured
		return ctrl.Result{}, err
	}
	if tQuerier == nil {
		logger.Info("No ThanosQuerier spec found")
		return ctrl.Result{}, nil
	}
	sidecarServices, err := r.findSidecarServices(ctx, tQuerier)

	if client.IgnoreNotFound(err) != nil {
		// we encountered an error other then NotFound, don't try to delete
		// resources for this querier and reschedule reconcile
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}
	if len(sidecarServices) == 0 {
		// either no MonitoringStack for this querier exists or it doesn't
		// expose a thanos-side car service
		// Clean up any resources that might have been created and return
		logger.Info("No thanos-sidecar services found, deleting any existing resources")
		r.maybeDeleteResources(ctx, tQuerier)
		return ctrl.Result{}, nil
	}

	components := thanosComponents(tQuerier, sidecarServices)

	controllerLabels := map[string]string{
		"app.kubernetes.io/managed-by": "monitoring-stack-operator",
	}
	for _, c := range components {
		logger.WithValues("component", c.GetObjectKind().GroupVersionKind())
		err := r.reconcileComponent(ctx, tQuerier, ensureLabels(c, controllerLabels))
		if err != nil {
			logger.Error(err, "Error in reconcile")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// Given a ThanosQuerier object, find the matching MonitoringStacks, extract the
// sidecar service and return a list of urls for those sidecar services.
func (r reconciler) findSidecarServices(ctx context.Context, tQuerier *msoapi.ThanosQuerier) ([]string, error) {
	logger := r.logger.WithValues("selector", tQuerier.Spec.Selector)

	msList := &msoapi.MonitoringStackList{}
	selector, _ := metav1.LabelSelectorAsSelector(&tQuerier.Spec.Selector)
	opts := []client.ListOption{
		client.MatchingLabelsSelector{Selector: selector},
	}

	sidecarUrls := []string{}
	if err := r.List(ctx, msList, opts...); err != nil {
		logger.Info("Couldn't find any MonitoringStack")
		return sidecarUrls, err
	}
	logger.Info("Found MonitoringStacks list", "length", len(msList.Items))
	for _, ms := range msList.Items {
		svcList := &corev1.ServiceList{}
		serviceSelector := labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/name":    fmt.Sprintf("%s-thanos-sidecar", ms.Name),
			"app.kubernetes.io/part-of": ms.Name,
		})
		opts := []client.ListOption{
			client.MatchingLabelsSelector{Selector: serviceSelector},
		}
		if len(tQuerier.Spec.NamespaceSelector.MatchNames) == 0 {
			err := r.List(ctx, svcList, opts...)
			if err != nil {
				logger.Info("No sidecar services found")
				return sidecarUrls, err
			}
			logger.Info("Found services", "count", len(svcList.Items))
			sidecarUrls = append(sidecarUrls, getEndpointUrl(svcList)...)
		} else {
			for _, ns := range tQuerier.Spec.NamespaceSelector.MatchNames {
				err := r.List(ctx, svcList, append(opts, client.ListOption(client.InNamespace(ns)))...)
				if err != nil {
					logger.Info("No sidecar services found", "namespace", ns)
					continue
				}
				logger.Info("Found services", "namespace", ns, "count", len(svcList.Items))
				sidecarUrls = append(sidecarUrls, getEndpointUrl(svcList)...)
			}
		}
	}
	return sidecarUrls, nil
}

// Given a Service object, return a url to use as value for --store/--endpoint.
func getEndpointUrl(svcList *corev1.ServiceList) []string {
	var ret []string
	for _, svc := range svcList.Items {
		ret = append(ret, fmt.Sprintf("dnssrv+_grpc._tcp.%s.%s.svc.cluster.local", svc.Name, svc.Namespace))
	}
	return ret
}

// Given a reconcile request, get a ThanosQuerier object.
func (r reconciler) getQuerierSpec(ctx context.Context, req ctrl.Request) (*msoapi.ThanosQuerier, error) {
	tQuerier := msoapi.ThanosQuerier{}

	err := r.Get(ctx, req.NamespacedName, &tQuerier)
	return &tQuerier, client.IgnoreNotFound(err)
}

// For a given Service, retrieve the owning object if it is of Kind
// MonitoringStack. If such an owner can be found, pass it to
// findQueriersForMonitoringStack().
func (r reconciler) findQueriersForService(svc client.Object) []reconcile.Request {
	logger := r.logger.WithValues("service", svc.GetNamespace()+"/"+svc.GetName())
	logger.Info("Watched Service changed, checking for matching querier")
	svcOwners := svc.GetOwnerReferences()
	logger.Info("Found", "OwnerRefs", svcOwners)
	err, msName := findMonStackServiceOwner(svcOwners)
	if err != nil {
		logger.Info("Owner is not a MonitoringStack, ignoring")
		return []reconcile.Request{}
	}

	ms := &msoapi.MonitoringStack{}
	if err := r.Get(context.Background(), types.NamespacedName{Name: msName, Namespace: svc.GetNamespace()}, ms); err != nil {
		logger.Info("Failed to get Monitoring Stack for")
		return []reconcile.Request{}
	}
	return r.findQueriersForMonitoringStack(ms)
}

// Find all ThanosQueriers, whose Selector fits the given MonitoringStack and
// return a list of reconcile requests, one for each ThanosQuerier.
func (r reconciler) findQueriersForMonitoringStack(ms client.Object) []reconcile.Request {
	logger := r.logger.WithValues("Monitoring Stack", ms.GetNamespace()+"/"+ms.GetName())
	logger.Info("watched MonitoringStack changed, checking for matching querier")
	queriers := &msoapi.ThanosQuerierList{}
	err := r.List(context.TODO(), queriers, &client.ListOptions{})
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

// Filter a list of OwnerReferences and return the first owner of Kind
// MonitoringStack.
func findMonStackServiceOwner(owners []metav1.OwnerReference) (error, string) {
	for _, owner := range owners {
		// This should be compared to a MonitoringStack{} object, but in
		// this case Kind is empty, so use hardcoded string for now
		if owner.Kind == "MonitoringStack" {
			return nil, owner.Name
		}
	}
	return errors.New("Owner is not of kind MonitoringStack"), ""
}

// Create an object after setting the owner reference to the passed
// ThanosQueriers.
// Uses server-side-apply.
func (r reconciler) reconcileComponent(ctx context.Context, thanos *msoapi.ThanosQuerier, component client.Object) error {
	if thanos.Namespace == component.GetNamespace() {
		if err := controllerutil.SetControllerReference(thanos, component, r.scheme); err != nil {
			return err
		}
	}

	if err := r.Patch(ctx, component, client.Apply, client.ForceOwnership, client.FieldOwner("thanos-querier-controller")); err != nil {
		return err
	}
	return nil
}

// Delete all resources attached to the passed ThanosQuerier, if they still
// exist.
func (r reconciler) maybeDeleteResources(ctx context.Context, thanos *msoapi.ThanosQuerier) {
	logger := r.logger.WithValues("thanosQuerier", thanos.GetNamespace()+"/"+thanos.GetName())
	logger.Info("Deleting resources created by")
	for _, gvk := range []schema.GroupVersionKind{
		{
			Group:   "apps",
			Kind:    "Deployment",
			Version: "v1",
		},
		{
			Group:   "",
			Kind:    "ServiceAccount",
			Version: "v1",
		},
		{
			Group:   "monitoring.coreos.com",
			Kind:    "ServiceMonitor",
			Version: "v1",
		},
	} {
		resource := &unstructured.Unstructured{}
		resource.SetGroupVersionKind(gvk)
		if err := r.DeleteAllOf(ctx, resource, client.InNamespace(thanos.Namespace), client.MatchingLabels{"app.kubernetes.io/part-of": "thanos-querier-" + thanos.Name}); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info("Resources not found, nothing to do")
			} else {
				logger.Error(err, "DeleteAllOf failed", "resource", resource)
			}
		}
	}
	srv := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thanos-querier-" + thanos.Name,
			Namespace: thanos.Namespace,
		},
	}
	if err := r.Delete(ctx, srv); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Resources not found, nothing to do")
		} else {
			logger.Error(err, "Delete failed", "resource", srv)
		}
	}
}

func ensureLabels(obj client.Object, wantLabels map[string]string) client.Object {
	labels := obj.GetLabels()
	if labels == nil {
		obj.SetLabels(wantLabels)
		return obj
	}
	for name, val := range wantLabels {
		if _, ok := labels[name]; !ok {
			labels[name] = val
		}
	}
	return obj
}
