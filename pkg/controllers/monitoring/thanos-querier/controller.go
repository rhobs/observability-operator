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
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
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
	scheme *runtime.Scheme
	logger logr.Logger
	thanos ThanosConfiguration
}

type ThanosConfiguration struct {
	Image string
}

// Options allows for controller options to be set
type Options struct {
	Thanos ThanosConfiguration
}

const (
	thanosTLSPrivateKeySecretNameField           = ".spec.webTLSConfig.privateKey.name"
	thanosTLSCertificateSecretNameField          = ".spec.webTLSConfig.certificate.name"
	thanosTLSCertificateAuthoritySecretNameField = ".spec.webTLSConfig.certificateAuthority.name"
)

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
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=list;watch

// RBAC for managing Prometheus Operator CRs
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=servicemonitors,verbs=list;watch;create;update;patch;delete

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager, opts Options) error {
	logger := ctrl.Log.WithName("thanos-querier")

	rm := &resourceManager{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		logger: logger,
		thanos: opts.Thanos,
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &msoapi.ThanosQuerier{}, thanosTLSPrivateKeySecretNameField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*msoapi.ThanosQuerier)
		if cr.Spec.WebTLSConfig == nil {
			return nil
		}
		return []string{cr.Spec.WebTLSConfig.PrivateKey.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &msoapi.ThanosQuerier{}, thanosTLSCertificateSecretNameField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*msoapi.ThanosQuerier)
		if cr.Spec.WebTLSConfig == nil {
			return nil
		}
		return []string{cr.Spec.WebTLSConfig.Certificate.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &msoapi.ThanosQuerier{}, thanosTLSCertificateAuthoritySecretNameField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*msoapi.ThanosQuerier)
		if cr.Spec.WebTLSConfig == nil {
			return nil
		}
		return []string{cr.Spec.WebTLSConfig.CertificateAuthority.Name}
	}); err != nil {
		return err
	}

	p := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&msoapi.ThanosQuerier{}).
		Owns(&appsv1.Deployment{}).WithEventFilter(p).
		Owns(&corev1.ServiceAccount{}).WithEventFilter(p).
		Owns(&corev1.Service{}).WithEventFilter(p).
		Owns(&corev1.ConfigMap{}).WithEventFilter(p).
		Watches(
			&msoapi.MonitoringStack{},
			handler.EnqueueRequestsFromMapFunc(rm.findQueriersForMonitoringStack),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(rm.findQueriersForTLSSecrets),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
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

	tlsHashes := map[string]string{}
	if querier.Spec.WebTLSConfig != nil {
		secretSelectors := []msoapi.SecretKeySelector{
			querier.Spec.WebTLSConfig.CertificateAuthority,
			querier.Spec.WebTLSConfig.Certificate,
			querier.Spec.WebTLSConfig.PrivateKey,
		}
		for _, secretSelector := range secretSelectors {
			hash, err := rm.hashOfTLSSecret(secretSelector, querier.Namespace)
			if err != nil {
				return ctrl.Result{}, err
			}
			tlsHashes[fmt.Sprintf("%s-%s", secretSelector.Name, secretSelector.Key)] = hash
		}
	}

	reconcilers := thanosComponentReconcilers(querier, sidecarServices, rm.thanos, tlsHashes)
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

func (rm resourceManager) hashOfTLSSecret(selector msoapi.SecretKeySelector, namespace string) (string, error) {
	var secret corev1.Secret
	err := rm.Get(context.Background(), types.NamespacedName{
		Name:      selector.Name,
		Namespace: namespace,
	}, &secret)
	if err != nil {
		return "", fmt.Errorf("Couldn't get TLS secret %s: %s", selector.Name, err)
	}

	hash := sha256.Sum256(secret.Data[selector.Key])
	return rand.SafeEncodeString(fmt.Sprint(hash)), nil
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

// Find all ThanosQueriers, whose TLS secrets fit the given Secret and
// return a list of reconcile requests, one for each ThanosQuerier.
func (rm resourceManager) findQueriersForTLSSecrets(ctx context.Context, src client.Object) []reconcile.Request {
	requests := []reconcile.Request{}

	thanosWatchFields := []string{
		thanosTLSCertificateAuthoritySecretNameField,
		thanosTLSCertificateSecretNameField,
		thanosTLSPrivateKeySecretNameField,
	}

	for _, field := range thanosWatchFields {
		crList := &msoapi.ThanosQuerierList{}
		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, src.GetName()),
			Namespace:     src.GetNamespace(),
		}
		err := rm.Client.List(ctx, crList, listOps)
		if err != nil {
			rm.logger.Error(err, "Failed to list Thanosqueriers")
			return []reconcile.Request{}
		}

		for _, item := range crList.Items {
			logger := rm.logger.WithValues("Secret", src.GetNamespace()+"/"+src.GetName())
			logger.Info("Found a querier whose secret changed, scheduling sync")
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
