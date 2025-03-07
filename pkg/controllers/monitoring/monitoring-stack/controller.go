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
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/assets"
	"github.com/rhobs/observability-operator/pkg/controllers/monitoring/utils"
)

type resourceManager struct {
	k8sClient             client.Client
	scheme                *runtime.Scheme
	logger                logr.Logger
	instanceSelectorKey   string
	instanceSelectorValue string
	controller            controller.Controller
	prometheus            PrometheusConfiguration
	alertmanager          AlertmanagerConfiguration
	thanos                ThanosConfiguration
}

type PrometheusConfiguration struct {
	Image string
}

type AlertmanagerConfiguration struct {
	Image string
}

type ThanosConfiguration struct {
	Image string
}

// Options allows for controller options to be set
type Options struct {
	InstanceSelector string
	Prometheus       PrometheusConfiguration
	Alertmanager     AlertmanagerConfiguration
	Thanos           ThanosConfiguration
}

const finalizerName = "monitoring.observability.openshift.io/finalizer"

// RBAC for managing monitoring stacks
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=list;watch;create;update
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/status,verbs=get;update

// RBAC for managing Prometheus Operator CRs
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=alertmanagers;prometheuses;servicemonitors,verbs=list;watch;create;update;delete;patch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=list;watch;create;update;delete;patch
//+kubebuilder:rbac:groups="",resources=serviceaccounts;services;secrets,verbs=list;watch;create;update;delete;patch
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=list;watch;create;update;delete;patch

// RBAC for delegating permissions to Prometheus
//+kubebuilder:rbac:groups="",resources=pods;services;endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=extensions;networking.k8s.io,resources=ingresses,verbs=get;list;watch

// RBAC for delegating the use of SCC nonroot-v2 (for OpenShift >= 4.11) and nonroot (for OpenShift < 4.11)
//+kubebuilder:rbac:groups="security.openshift.io",resources=securitycontextconstraints,resourceNames=nonroot;nonroot-v2,verbs=use

// RegisterWithManager registers the controller with Manager
func RegisterWithManager(mgr ctrl.Manager, opts Options) error {
	split := strings.Split(opts.InstanceSelector, "=")
	if len(split) != 2 {
		return fmt.Errorf("invalid InstanceSelector: %s", opts.InstanceSelector)
	}

	rm := &resourceManager{
		k8sClient:             mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		logger:                ctrl.Log.WithName("observability-operator"),
		instanceSelectorKey:   split[0],
		instanceSelectorValue: split[1],
		thanos:                opts.Thanos,
		prometheus:            opts.Prometheus,
		alertmanager:          opts.Alertmanager,
	}
	// We only want to trigger a reconciliation when the generation
	// of a child changes. Until we need to update our the status for our own objects,
	// we can save CPU cycles by avoiding reconciliations triggered by
	// child status changes. The only exception is Prometheus resources, where we want to
	// be notified about changes in their status.
	generationChanged := builder.WithPredicates(predicate.GenerationChangedPredicate{})

	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		For(&stack.MonitoringStack{}).
		Owns(&monv1.Prometheus{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&monv1.Alertmanager{}, generationChanged).
		Owns(&v1.Service{}, generationChanged).
		Owns(&v1.ServiceAccount{}, generationChanged).
		Owns(&rbacv1.Role{}, generationChanged).
		Owns(&rbacv1.RoleBinding{}, generationChanged).
		Owns(&monv1.ServiceMonitor{}, generationChanged).
		Owns(&policyv1.PodDisruptionBudget{}, generationChanged).
		Build(rm)

	if err != nil {
		return err
	}
	rm.controller = ctrl
	return nil
}

func (rm resourceManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := rm.logger.WithValues("stack", req.NamespacedName)
	logger.Info("Reconciling monitoring stack")

	gRPCSecret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      assets.GRPCSecretName,
			Namespace: req.Namespace,
		},
		Data: map[string][]byte{},
	}
	err := rm.k8sClient.Get(ctx,
                types.NamespacedName{
                        Name: assets.GRPCSecretName,
                        Namespace: req.Namespace,
                },
                &gRPCSecret)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	rotate, err := assets.RotateGRPCSecret(&gRPCSecret, logger)
	if err != nil {
		return ctrl.Result{}, err
	}
	if rotate {
		err = rm.k8sClient.Update(ctx, &gRPCSecret)
		if errors.IsNotFound(err) {
			err = rm.k8sClient.Create(ctx, &gRPCSecret)
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	ms, err := rm.getStack(ctx, req)
	if err != nil {
		// retry since some error has occured
		return ctrl.Result{}, err
	}

	if ms == nil {
		// no such monitoring stack, so stop here
		return ctrl.Result{}, nil
	}

	// Check if the plugin is being deleted
	if !ms.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.V(6).Info("removing cluster scoped resources")

		reconcilers := stackComponentCleanup(ms)
		for _, reconciler := range reconcilers {
			err := reconciler.Reconcile(ctx, rm.k8sClient, rm.scheme)
			if err != nil {
				logger.Error(err, "failed to cleanup monitoring stack")
			}
		}

		// Remove finalizer if present
		if slices.Contains(ms.ObjectMeta.Finalizers, finalizerName) {
			ms.ObjectMeta.Finalizers = slices.DeleteFunc(ms.ObjectMeta.Finalizers, func(currentFinalizerName string) bool {
				return currentFinalizerName == finalizerName
			})
			if err := rm.k8sClient.Update(ctx, ms); err != nil {
				return ctrl.Result{}, err
			}
		}

		logger.V(6).Info("skipping reconcile since object is already scheduled for deletion")
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !slices.Contains(ms.ObjectMeta.Finalizers, finalizerName) {
		ms.ObjectMeta.Finalizers = append(ms.ObjectMeta.Finalizers, finalizerName)
		if err := rm.k8sClient.Update(ctx, ms); err != nil {
			return ctrl.Result{}, err
		}
	}
	// querier <---> sidecar mTLS hashes
	mTLSSecretKeys := []string{"prometheus-server.key", "prometheus-server.crt", "ca.crt"}
	tlsHashes := map[string]string{}
	for _, key := range mTLSSecretKeys {
		hash, err := utils.HashOfTLSSecret(assets.GRPCSecretName, key, ms.Namespace, rm.k8sClient)
		if err != nil {
			return ctrl.Result{}, err
		}
		tlsHashes[fmt.Sprintf("%s-%s", assets.GRPCSecretName, key)] = hash
	}

	reconcilers := stackComponentReconcilers(ms,
		rm.instanceSelectorKey,
		rm.instanceSelectorValue,
		rm.thanos,
		rm.prometheus,
		rm.alertmanager,
		tlsHashes,
	)
	for _, reconciler := range reconcilers {
		err := reconciler.Reconcile(ctx, rm.k8sClient, rm.scheme)
		// handle create / update errors that can happen due to a stale cache by
		// retrying after some time.
		if errors.IsAlreadyExists(err) || errors.IsConflict(err) {
			logger.V(3).Info("skipping reconcile error", "err", err)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		if err != nil {
			return rm.updateStatus(ctx, req, ms, err), err
		}
	}

	return rm.updateStatus(ctx, req, ms, nil), nil
}

func (rm resourceManager) updateStatus(ctx context.Context, req ctrl.Request, ms *stack.MonitoringStack, recError error) ctrl.Result {
	var prom monv1.Prometheus
	logger := rm.logger.WithValues("stack", req.NamespacedName)
	key := client.ObjectKey{
		Name:      ms.Name,
		Namespace: ms.Namespace,
	}
	err := rm.k8sClient.Get(ctx, key, &prom)
	if err != nil {
		logger.Info("Failed to get prometheus object", "err", err)
		return ctrl.Result{RequeueAfter: 2 * time.Second}
	}
	ms.Status.Conditions = updateConditions(ms, prom, recError)
	err = rm.k8sClient.Status().Update(ctx, ms)
	if err != nil {
		logger.Info("Failed to update status", "err", err)
		return ctrl.Result{RequeueAfter: 2 * time.Second}
	}
	return ctrl.Result{}
}

func (rm resourceManager) getStack(ctx context.Context, req ctrl.Request) (*stack.MonitoringStack, error) {
	logger := rm.logger.WithValues("stack", req.NamespacedName)

	ms := stack.MonitoringStack{}

	if err := rm.k8sClient.Get(ctx, req.NamespacedName, &ms); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("stack could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get monitoring stack")
		return nil, err
	}

	return &ms, nil
}
