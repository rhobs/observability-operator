package loggingstack

import (
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	loggingv1 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	olmv1alpha2 "github.com/operator-framework/api/pkg/operators/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	stack "github.com/rhobs/observability-operator/pkg/apis/logging/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const (
	// operators
	nameClusterLoggingOperator = "cluster-logging"
	nameLokiOperator           = "loki-operator"
	namespaceLokiOperator      = "openshift-operators-redhat"
	// custom resources
	instanceClusterLogging      = "instance"
	instanceClusterLogForwarder = "instance"
	instanceLokistack           = "logging-loki"
	stackNamespace              = "openshift-logging"
)

type loggingStackReconciler struct {
	reconciler.Reconciler
	Requeue bool
}

func stackComponentReconcilers(ls *stack.LoggingStack) []loggingStackReconciler {
	withAuditLogs := ls.Spec.ForwarderSpec.WithAuditLogs
	return []loggingStackReconciler{
		// Installation Namespaces
		{
			Reconciler: reconciler.NewUpdater(newOpenShiftLoggingNamespace(ls), nil),
			Requeue:    false,
		},
		{
			Reconciler: reconciler.NewUpdater(newOpenShiftOperatorsRedHatNamespace(ls), nil),
			Requeue:    false,
		},
		// Cluster Logging Operator
		{
			Reconciler: reconciler.NewUpdater(newClusterLoggingOperatorGroup(), nil),
			Requeue:    true,
		},
		{
			Reconciler: reconciler.NewUpdater(newClusterLoggingOperatorSubscription(ls), nil),
			Requeue:    true,
		},
		// Loki Operator
		{
			Reconciler: reconciler.NewUpdater(newLokiOperatorGroup(), nil),
			Requeue:    true,
		},
		{
			Reconciler: reconciler.NewUpdater(newLokiOperatorSubscription(ls), nil),
			Requeue:    true,
		},
		// Storage Deployment
		{
			Reconciler: reconciler.NewUpdater(newLokiStack(ls), ls),
			Requeue:    true,
		},
		// Collector Deployment
		{
			Reconciler: reconciler.NewUpdater(newClusterLogging(ls), ls),
			Requeue:    true,
		},
		// Forwarder Deployment
		{
			Reconciler: reconciler.NewOptionalUpdater(newClusterLogForwarder(ls), ls, withAuditLogs),
			Requeue:    true,
		},
	}
}

func newOpenShiftLoggingNamespace(ls *stack.LoggingStack) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   stackNamespace,
			Labels: ls.Spec.MonitoringSelector,
		},
	}
}

func newOpenShiftOperatorsRedHatNamespace(ls *stack.LoggingStack) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceLokiOperator,
			Labels: ls.Spec.MonitoringSelector,
		},
	}
}

func newClusterLoggingOperatorGroup() *olmv1alpha2.OperatorGroup {
	return &olmv1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmv1alpha2.SchemeGroupVersion.String(),
			Kind:       olmv1alpha2.OperatorGroupKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameClusterLoggingOperator,
			Namespace: stackNamespace,
		},
		Spec: olmv1alpha2.OperatorGroupSpec{
			TargetNamespaces: []string{
				stackNamespace,
			},
		},
	}
}

func newClusterLoggingOperatorSubscription(ls *stack.LoggingStack) *olmv1alpha1.Subscription {
	return &olmv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmv1alpha1.SchemeGroupVersion.String(),
			Kind:       olmv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameClusterLoggingOperator,
			Namespace: stackNamespace,
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel:                ls.Spec.Subscription.Channel,
			CatalogSource:          ls.Spec.Subscription.CatalogSource,
			CatalogSourceNamespace: ls.Spec.Subscription.CatalogSourceNamespace,
			Package:                nameClusterLoggingOperator,
		},
	}
}

func newLokiOperatorGroup() *olmv1alpha2.OperatorGroup {
	return &olmv1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmv1alpha2.SchemeGroupVersion.String(),
			Kind:       olmv1alpha2.OperatorGroupKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameLokiOperator,
			Namespace: namespaceLokiOperator,
		},
		Spec: olmv1alpha2.OperatorGroupSpec{},
	}

}

func newLokiOperatorSubscription(ls *stack.LoggingStack) *olmv1alpha1.Subscription {
	return &olmv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmv1alpha1.SchemeGroupVersion.String(),
			Kind:       olmv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameLokiOperator,
			Namespace: namespaceLokiOperator,
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel:                ls.Spec.Subscription.Channel,
			CatalogSource:          ls.Spec.Subscription.CatalogSource,
			CatalogSourceNamespace: ls.Spec.Subscription.CatalogSourceNamespace,
			Package:                nameLokiOperator,
		},
	}
}

func newLokiStack(ls *stack.LoggingStack) *lokiv1.LokiStack {
	return &lokiv1.LokiStack{
		TypeMeta: metav1.TypeMeta{
			APIVersion: lokiv1.GroupVersion.String(),
			Kind:       "LokiStack",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceLokistack,
			Namespace: stackNamespace,
		},
		Spec: lokiv1.LokiStackSpec{
			ManagementState: lokiv1.ManagementStateManaged,
			Size:            ls.Spec.Storage.Size,
			Storage:         ls.Spec.Storage.Storage,
			Tenants: &lokiv1.TenantsSpec{
				Mode: lokiv1.OpenshiftLogging,
			},
		},
	}
}

func newClusterLogging(ls *stack.LoggingStack) *loggingv1.ClusterLogging {
	return &loggingv1.ClusterLogging{
		TypeMeta: metav1.TypeMeta{
			APIVersion: loggingv1.GroupVersion.String(),
			Kind:       "ClusterLogging",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceClusterLogging,
			Namespace: stackNamespace,
		},
		Spec: loggingv1.ClusterLoggingSpec{
			ManagementState: loggingv1.ManagementStateManaged,
			Collection: &loggingv1.CollectionSpec{
				Type: loggingv1.LogCollectionTypeVector,
			},
			LogStore: &loggingv1.LogStoreSpec{
				Type: loggingv1.LogStoreTypeLokiStack,
				LokiStack: loggingv1.LokiStackStoreSpec{
					Name: instanceLokistack,
				},
			},
		},
	}
}

func newClusterLogForwarder(ls *stack.LoggingStack) *loggingv1.ClusterLogForwarder {
	return &loggingv1.ClusterLogForwarder{
		TypeMeta: metav1.TypeMeta{
			APIVersion: loggingv1.GroupVersion.String(),
			Kind:       "ClusterLogForwarder",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceClusterLogForwarder,
			Namespace: stackNamespace,
		},
		Spec: loggingv1.ClusterLogForwarderSpec{
			Pipelines: []loggingv1.PipelineSpec{
				{
					InputRefs: []string{
						loggingv1.InputNameApplication,
						loggingv1.InputNameAudit,
						loggingv1.InputNameInfrastructure,
					},
					Name: "all-to-default",
					OutputRefs: []string{
						"default",
					},
				},
			},
		},
	}
}
