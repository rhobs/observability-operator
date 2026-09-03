package observability

import (
	"context"
	"fmt"
	"strings"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rhobs/observability-operator/config"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

type operatorsStatus struct {
	cooNamespace string
	// Subscriptions installed in all namespaces.
	// The subscription name can be opentelemetry-product or opentelemetry-operator.
	subs []olmv1alpha1.Subscription
}

// ShouldInstall checks if the operator should be uninstalled.
// The operator should be installed only if it is not installed
func (s *operatorsStatus) ShouldInstall(operatorName string) bool {
	for _, sub := range s.subs {
		if strings.HasPrefix(sub.Name, operatorName) && sub.Labels[util.ResourceLabel] != util.OpName {
			return false
		}
	}
	return true
}

func (s *operatorsStatus) cooManages(operatorName string) *olmv1alpha1.Subscription {
	for _, sub := range s.subs {
		if strings.HasPrefix(sub.Name, operatorName) && sub.Labels[util.ResourceLabel] == util.OpName {
			return &sub
		}
	}
	return nil
}

// getReconcilers returns a list of reconcilers for the ObservabilityInstaller instance.
// The subByName is used to check if the operators are already installed, if not, they will be installed.
// The csvByName is used to uninstall the operators, the name of the CSV contains the version therefore it must be retrieved from the cluster.
// The CSV is not deleted when the subscription is deleted, so we need to delete it explicitly.
func getReconcilers(ctx context.Context, k8sReader client.Reader, instance *obsv1alpha1.ObservabilityInstaller, scheme *runtime.Scheme, opts Options, operatorsStatus operatorsStatus) ([]reconciler.Reconciler, error) {
	var reconcilers []reconciler.Reconciler
	cfg := NewOverlayConfig(scheme, opts)

	// Build the object set for the universe of all possible capabilities (for cleanup).
	allInstance := instance.DeepCopy()
	if allInstance.Spec.Capabilities == nil {
		allInstance.Spec.Capabilities = &obsv1alpha1.CapabilitiesSpec{}
	}
	if allInstance.Spec.Capabilities.Tracing == nil {
		allInstance.Spec.Capabilities.Tracing = &obsv1alpha1.TracingSpec{}
	}
	allInstance.Spec.Capabilities.Tracing.Enabled = true
	allObjects, err := ResolveInstaller(ctx, k8sReader, allInstance, cfg)
	if err != nil {
		return nil, fmt.Errorf("building full object set: %w", err)
	}

	// Handle deletion of the entire instance.
	if instance.ObjectMeta.DeletionTimestamp != nil {
		for _, obj := range allObjects {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
		if otelSub := operatorsStatus.cooManages("opentelemetry"); otelSub != nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(otelSub))
			reconcilers = append(reconcilers, reconciler.NewDeleter(
				&olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      otelSub.Status.CurrentCSV,
						Namespace: otelSub.Namespace,
					},
				}))
		}
		if tempoSub := operatorsStatus.cooManages("tempo"); tempoSub != nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(tempoSub))
			reconcilers = append(reconcilers, reconciler.NewDeleter(
				&olmv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tempoSub.Status.CurrentCSV,
						Namespace: tempoSub.Namespace,
					},
				}))
		}
		return reconcilers, nil
	}

	// Build the object set for the currently desired state.
	currentObjects, err := ResolveInstaller(ctx, k8sReader, instance, cfg)
	if err != nil {
		return nil, fmt.Errorf("building current object set: %w", err)
	}

	installedObjects := map[string]client.Object{}

	installOperators := false
	installResources := false
	if tracing := instance.Spec.GetCapabilities().GetTracing(); tracing != nil {
		installOperators = tracing.CommonCapabilitiesSpec.InstallOperators()
		installResources = tracing.Enabled
	}

	// Reconcile subscriptions before other objects so that OLM can install
	// the operators (and their CRDs) before we try to create CRD instances.
	if installOperators {
		for _, obj := range currentObjects {
			if isSubscription(obj) {
				if operatorsStatus.ShouldInstall(subscriptionPrefix(obj.GetName())) {
					reconcilers = append(reconcilers, reconciler.NewCreateUpdateReconciler(obj, instance))
				}
				installedObjects[gvkNameIdentifier(obj)] = obj
			}
		}
	}
	if installResources {
		for _, obj := range currentObjects {
			if !isSubscription(obj) {
				reconcilers = append(reconcilers, reconciler.NewUpdater(obj, instance))
				installedObjects[gvkNameIdentifier(obj)] = obj
			}
		}
	}

	// Delete not created objects.
	for _, obj := range allObjects {
		if installedObjects[gvkNameIdentifier(obj)] == nil {
			reconcilers = append(reconcilers, reconciler.NewDeleter(obj))
		}
	}
	// Delete CSV explicitly because it is not deleted when the subscription is deleted.
	// This handles the uninstall case when the capability is disabled or the operators installation is disabled.
	otelSubs := findByName(allObjects, "opentelemetry-product")
	if otelSub := operatorsStatus.cooManages("opentelemetry"); otelSub != nil && (otelSubs == nil || installedObjects[gvkNameIdentifier(otelSubs)] == nil) {
		reconcilers = append(reconcilers, reconciler.NewDeleter(otelSub))
		reconcilers = append(reconcilers, reconciler.NewDeleter(
			&olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      otelSub.Status.CurrentCSV,
					Namespace: otelSub.Namespace,
				},
			}))
	}
	tempoSubs := findByName(allObjects, "tempo-product")
	if tempoSub := operatorsStatus.cooManages("tempo"); tempoSub != nil && (tempoSubs == nil || installedObjects[gvkNameIdentifier(tempoSubs)] == nil) {
		reconcilers = append(reconcilers, reconciler.NewDeleter(tempoSub))
		reconcilers = append(reconcilers, reconciler.NewDeleter(
			&olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tempoSub.Status.CurrentCSV,
					Namespace: tempoSub.Namespace,
				},
			}))
	}

	return reconcilers, nil
}

func findByName(objects []client.Object, name string) client.Object {
	for _, obj := range objects {
		if obj.GetName() == name {
			return obj
		}
	}
	return nil
}

func NewOverlayConfig(scheme *runtime.Scheme, opts Options) OverlayConfig {
	return OverlayConfig{
		Scheme:    scheme,
		ConfigFS:  config.FS,
		Options:   opts,
		Operators: true,
		Resources: true,
	}
}

func isSubscription(obj client.Object) bool {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return gvk.Group == "operators.coreos.com" && gvk.Kind == "Subscription"
}

func subscriptionPrefix(name string) string {
	if i := strings.LastIndex(name, "-"); i > 0 {
		return name[:i]
	}
	return name
}

func gvkNameIdentifier(obj client.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetObjectKind().GroupVersionKind().String(), obj.GetName())
}
