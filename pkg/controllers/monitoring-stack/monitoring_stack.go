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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.rhobs,resources=monitoringstacks/finalizers,verbs=update
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheuses,verbs=get;list;watch;create;update;patch;delete

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

	return ctrl.NewControllerManagedBy(mgr).
		For(&stack.MonitoringStack{}).
		Owns(&monv1.Prometheus{}).
		Complete(r)
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.logger.WithValues("stack", req.NamespacedName)

	ms, err := r.getStack(ctx, req)
	if err != nil {
		// retry since some error has occured
		return ctrl.Result{}, err
	}
	if ms == nil {
		// no such monitoring stack, so stop here
		return ctrl.Result{}, nil
	}

	// check if the Prometheus CR exits for the ms
	prometheus := monv1.Prometheus{}

	// monitoring-stack and prometheus has 1-1 mapping
	err = r.k8sClient.Get(ctx, types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}, &prometheus)
	logger.V(10).Info("monitoring stack", "err", err, "prom", prometheus)

	if errors.IsNotFound(err) {
		return r.createNewPrometheus(ctx, ms)
	}
	if err != nil {
		logger.Error(err, "error getting promethues")
		return ctrl.Result{}, err
	}

	// if prometheus already exits update it
	return r.updatePrometheus(ctx, ms, &prometheus)

	// TODO(sthaha): Add a retry after a delay and update status once
	// PrometheusOperator implements Status
}

func (r *reconciler) createNewPrometheus(ctx context.Context, ms *stack.MonitoringStack) (ctrl.Result, error) {
	logger := r.logger.WithValues("stack name", ms.Name, "namespace", ms.Namespace, "action", "create")
	prometheus := r.prometheusForStack(ms)

	logger.V(8).Info("going to create prometheus", "prometheus", prometheus.Spec, "ms", ms.Spec)

	if err := ctrl.SetControllerReference(ms, prometheus, r.scheme); err != nil {
		logger.Error(err, "failed set controller reference")
		return ctrl.Result{}, err
	}

	if err := r.k8sClient.Create(ctx, prometheus); err != nil {
		logger.Error(err, "failed to create prometheus cr")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) updatePrometheus(ctx context.Context, ms *stack.MonitoringStack, prometheus *monv1.Prometheus) (ctrl.Result, error) {
	logger := r.logger.WithValues("stack name", ms.Name, "namespace", ms.Namespace, "action", "update")
	logger.V(8).Info("going to update prometheus", "prometheus", prometheus.Spec, "ms", ms.Spec)

	expected := r.prometheusForStack(ms)

	logger.V(8).Info("update prometheus ", "from", prometheus.Spec, "to", expected.Spec)

	prometheus.Spec = expected.Spec
	if err := ctrl.SetControllerReference(ms, prometheus, r.scheme); err != nil {
		logger.Error(err, "failed set controller reference")
		return ctrl.Result{}, err
	}

	if err := r.k8sClient.Update(ctx, prometheus); err != nil {
		logger.Error(err, "failed to update prometheus cr")
		return ctrl.Result{}, err
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

func (r *reconciler) prometheusForStack(ms *stack.MonitoringStack) *monv1.Prometheus {
	spec := ms.Spec
	p := &monv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name,
			Namespace: ms.Namespace,
			Labels:    r.labelsForPrometheus(ms.Name),
		},

		Spec: monv1.PrometheusSpec{
			// Prometheus does not use an Enum for LogLevel, so need to convert to string
			LogLevel: string(spec.LogLevel),

			Retention: spec.Retention,
			Resources: spec.Resources,

			ServiceMonitorSelector:          ms.Spec.ResourceSelector,
			ServiceMonitorNamespaceSelector: ms.Spec.ResourceSelector,
			PodMonitorSelector:              ms.Spec.ResourceSelector,
			PodMonitorNamespaceSelector:     ms.Spec.ResourceSelector,
		},
	}
	return p
}

func (r reconciler) labelsForPrometheus(msName string) map[string]string {
	return map[string]string{
		r.instanceSelectorKey:                  r.instanceSelectorValue,
		"monitoring.rhobs.io/monitoring-stack": msName,
	}
}
