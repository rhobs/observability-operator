package logging

import (
	"context"
	"fmt"
	"slices"

	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	clfv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	storage "github.com/rhobs/observability-operator/pkg/controllers/observability/storage"
)

// Options contains only the operators used by this capability.
type Options struct {
	LokiOperator           capability.OperatorConfig
	ClusterLoggingOperator capability.OperatorConfig
}

// New constructs the logging capability.
func New(opts Options) capability.Definition {
	return capability.Definition{
		Name:         "logging",
		Shared:       func(instance *obsv1alpha1.ObservabilityInstaller) capability.Plan { return shared(instance, opts) },
		Plan:         planLogging,
		UpdateStatus: updateLoggingStatus,
		WatchTypes:   []client.Object{&lokiv1.LokiStack{}, &clfv1.ClusterLogForwarder{}},
	}
}

func planLogging(ctx context.Context, instance *obsv1alpha1.ObservabilityInstaller, reader client.Reader) (capability.Plan, error) {
	objects := []client.Object{lokiStack(instance), clusterLogForwarder(instance), loggingServiceAccount(instance)}
	collectorRole, collectorBindings := loggingCollectorRBAC(instance)
	objects = append(objects, collectorRole)
	for _, binding := range collectorBindings {
		objects = append(objects, binding)
	}
	names := storage.Names{Secret: lokiSecretName(instance.Name), CAConfigMap: lokiStorageCAConfigMapName(instance.Name)}
	plan := capability.Plan{Owned: slices.Concat(objects, storage.Inventory(instance.Namespace, names))}
	if !loggingEnabled(instance) {
		return plan, nil
	}

	// Loki's storage TLS spec accepts only a CA, see lokiStack().
	if tls := instance.Spec.GetCapabilities().GetLogging().GetLokiStack().GetStorage().GetTLS(); tls != nil && (tls.CertSecret != nil || tls.KeySecret != nil) {
		return capability.Plan{}, fmt.Errorf("LokiStack object storage TLS supports caConfigMap only, not certSecret or keySecret")
	}

	secrets, err := lokiStackSecrets(ctx, reader, *instance)
	if err != nil {
		return capability.Plan{}, fmt.Errorf("create LokiStack storage resources: %w", err)
	}
	plan.Desired = slices.Concat(objects, secrets.Objects())
	return plan, nil
}

func updateLoggingStatus(ctx context.Context, instance *obsv1alpha1.ObservabilityInstaller, reader client.Reader) capability.StatusResult {
	instance.Status.LokiStack = ""
	instance.Status.Logging = ""
	if !loggingEnabled(instance) {
		return capability.StatusResult{}
	}
	lokiStack := &lokiv1.LokiStack{}
	if err := reader.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: lokiStackName(instance.Name)}, lokiStack); err != nil {
		return capability.StatusReadError("LokiStack", err)
	}
	clf := &clfv1.ClusterLogForwarder{}
	if err := reader.Get(ctx, types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      clusterLogForwarderName(instance.Name),
	}, clf); err != nil {
		return capability.StatusReadError("ClusterLogForwarder", err)
	}

	instance.Status.LokiStack = fmt.Sprintf("%s/%s", instance.Namespace, lokiStackName(instance.Name))
	for _, condition := range clf.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
			instance.Status.Logging = fmt.Sprintf("%s/%s", instance.Namespace, clusterLogForwarderName(instance.Name))
			break
		}
	}
	return capability.StatusResult{}
}

func loggingEnabled(instance *obsv1alpha1.ObservabilityInstaller) bool {
	spec := instance.Spec.GetCapabilities().GetLogging()
	return instance.DeletionTimestamp == nil && spec != nil && spec.Enabled
}

// shared evaluates cluster-wide requirements without reading operand data.
func shared(instance *obsv1alpha1.ObservabilityInstaller, opts Options) capability.Plan {
	spec := instance.Spec.GetCapabilities().GetLogging()
	enabled, install := loggingEnabled(instance), false
	if instance.DeletionTimestamp == nil && spec != nil {
		operators := spec.GetOperators()
		install = operators != nil && operators.Install != nil && *operators.Install
	}
	plan := capability.Plan{Operators: capability.DesiredOperators(enabled || install,
		capability.NewOperatorRequirement("loki", opts.LokiOperator),
		capability.NewOperatorRequirement("cluster-logging", opts.ClusterLoggingOperator),
	)}
	plan.Owned = []client.Object{uiPlugin(instance)}
	if enabled {
		plan.Desired = plan.Owned
	}
	return plan
}
