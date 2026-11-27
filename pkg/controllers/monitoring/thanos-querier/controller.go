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
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	msoapi "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

type resourceManager struct {
	client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	logger    logr.Logger
	thanos    ThanosConfiguration
}

type ThanosConfiguration struct {
	Image string
}

// Options allows for controller options to be set
type Options struct {
	Thanos ThanosConfiguration
}

// tlsConfigRetryPeriod is how often a ThanosQuerier with an invalid
// spec.webTLSConfig is re-evaluated. The controller doesn't watch secrets, so
// this is the only thing that picks up a fix made by the user.
const tlsConfigRetryPeriod = 30 * time.Second

// RBAC for watching monitoring stacks
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;watch

// RBAC for managing thanosquerier objects
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=thanosqueriers/finalizers,verbs=update

// RBAC for managing deployments
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;create;update;patch;delete

// RBAC for managing core resources
//+kubebuilder:rbac:groups=core,resources=services;serviceaccounts;configmaps,verbs=list;watch;create;update;patch;delete

// RBAC for managing Prometheus Operator CRs
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=servicemonitors,verbs=list;watch;create;update;patch;delete

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager, opts Options) error {
	logger := ctrl.Log.WithName("thanos-querier")

	rm := &resourceManager{
		Client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		scheme:    mgr.GetScheme(),
		logger:    logger,
		thanos:    opts.Thanos,
	}

	generationChanged := builder.WithPredicates(predicate.GenerationChangedPredicate{})
	resourceVersionChanged := builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})
	return ctrl.NewControllerManagedBy(mgr).
		For(&msoapi.ThanosQuerier{}, generationChanged).
		Owns(&appsv1.Deployment{}, generationChanged).
		Owns(&corev1.ServiceAccount{}, generationChanged).
		Owns(&corev1.Service{}, generationChanged).
		Owns(&networkingv1.NetworkPolicy{}, resourceVersionChanged).
		Owns(&corev1.ConfigMap{}, generationChanged).
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

	if err := rm.validateWebTLSConfig(ctx, querier); err != nil {
		if _, ok := errors.AsType[invalidWebTLSConfigError](err); !ok {
			return ctrl.Result{}, err
		}

		// Reconciling with an unusable TLS configuration would at best leave
		// the deployment stuck on a volume that can't be mounted, so stop
		// here and leave whatever is already deployed untouched. Only the
		// user can fix this, and since secrets aren't watched nothing will
		// notify us when they do: retry on a fixed interval instead.
		logger.Error(err, "invalid spec.webTLSConfig, skipping reconciliation")
		return ctrl.Result{RequeueAfter: tlsConfigRetryPeriod}, nil
	}

	sidecarServices, err := rm.findSidecarServices(ctx, querier)
	if client.IgnoreNotFound(err) != nil {
		// we encountered an error other then NotFound, don't try to delete
		// resources for this querier and reschedule reconcile
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	// NOTE: the TLS certificate and key referenced by spec.webTLSConfig are
	// mounted into the Thanos Querier pod as secret volumes. Thanos serves
	// them via prometheus/exporter-toolkit, which re-reads the files from
	// disk on every TLS handshake, and kubelet refreshes the mounted volume
	// shortly after the secret changes. Rotating the certificate therefore
	// needs no action from this controller, which is why it doesn't watch
	// secrets. Watching them cluster-wide used to cost hundreds of megabytes
	// of cache on clusters with many secrets.
	reconcilers := thanosComponentReconcilers(querier, sidecarServices, rm.thanos, rm.apiReader)
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

// invalidWebTLSConfigError reports that spec.webTLSConfig references secrets
// which can't be used as-is. It is distinct from a transient API error because
// retrying can only help once the user has fixed the ThanosQuerier or the
// secrets it points at.
type invalidWebTLSConfigError struct {
	err error
}

func (e invalidWebTLSConfigError) Error() string { return e.err.Error() }

func (e invalidWebTLSConfigError) Unwrap() error { return e.err }

// validateWebTLSConfig checks that every secret referenced by
// spec.webTLSConfig exists and carries the referenced key. It returns an
// invalidWebTLSConfigError listing every problem found, or a plain error if
// the secrets couldn't be read at all.
//
// Secrets are read with the uncached API reader on purpose: the operator
// doesn't cache the secrets it doesn't manage (see COO-1737).
func (rm resourceManager) validateWebTLSConfig(ctx context.Context, querier *msoapi.ThanosQuerier) error {
	if querier.Spec.WebTLSConfig == nil {
		return nil
	}

	refs := []struct {
		field    string
		selector msoapi.SecretKeySelector
	}{
		{"certificate", querier.Spec.WebTLSConfig.Certificate},
		{"privateKey", querier.Spec.WebTLSConfig.PrivateKey},
		{"certificateAuthority", querier.Spec.WebTLSConfig.CertificateAuthority},
	}

	// The 3 references usually point at the same secret, read it only once.
	// A nil value means that the secret doesn't exist.
	secrets := map[string]*corev1.Secret{}
	var invalid []error

	for _, ref := range refs {
		field, selector := "spec.webTLSConfig."+ref.field, ref.selector

		if selector.Name == "" || selector.Key == "" {
			invalid = append(invalid, fmt.Errorf("%s: both name and key are required", field))
			continue
		}

		secret, read := secrets[selector.Name]
		if !read {
			secret = &corev1.Secret{}
			err := rm.apiReader.Get(ctx, types.NamespacedName{Name: selector.Name, Namespace: querier.Namespace}, secret)
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to read secret %s/%s: %w", querier.Namespace, selector.Name, err)
			}
			if err != nil {
				secret = nil
			}
			secrets[selector.Name] = secret
		}

		if secret == nil {
			invalid = append(invalid, fmt.Errorf("%s: secret %q not found", field, selector.Name))
			continue
		}

		switch value, found := secret.Data[selector.Key]; {
		case !found:
			invalid = append(invalid, fmt.Errorf("%s: secret %q has no %q key", field, selector.Name, selector.Key))
		case len(value) == 0:
			invalid = append(invalid, fmt.Errorf("%s: key %q of secret %q is empty", field, selector.Key, selector.Name))
		}
	}

	if len(invalid) > 0 {
		return invalidWebTLSConfigError{err: errors.Join(invalid...)}
	}

	return nil
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
	logger.Info("Evaluating MonitoringStacks", "length", len(msList.Items))

	// Keep only the MonitoringStacks matching the ThanosQuerier's namespace selector.
	for _, ms := range msList.Items {
		if tQuerier.MatchesNamespace(ms.Namespace) {
			serviceName := ms.Name + "-thanos-sidecar"
			sidecarUrls = append(sidecarUrls, getEndpointUrl(serviceName, ms.Namespace))
		}
	}
	logger.Info("Found matching MonitoringStacks", "length", len(sidecarUrls))

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
