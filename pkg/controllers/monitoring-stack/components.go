package monitoringstack

import (
	"fmt"
	stack "rhobs/monitoring-stack-operator/pkg/apis/v1alpha1"
	grafana_operator "rhobs/monitoring-stack-operator/pkg/controllers/grafana-operator"

	"k8s.io/apimachinery/pkg/runtime"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const AdditionalScrapeConfigsSelfScrapeKey = "self-scrape-config"

type emptyObjectFunc func() client.Object

type patchObjectFunc func(object client.Object) (client.Object, error)

type objectPatcher struct {
	empty emptyObjectFunc
	patch patchObjectFunc
}

type ObjectTypeError struct {
	expectedType string
	actualType   string
}

func NewObjectTypeError(expected runtime.Object, actual runtime.Object) error {
	gvk := func(o runtime.Object) string {
		gvk := o.GetObjectKind().GroupVersionKind()
		return gvk.Group + "/" + gvk.Version + " " + gvk.Kind
	}

	return ObjectTypeError{expectedType: gvk(expected), actualType: gvk(actual)}
}

func (e ObjectTypeError) Error() string {
	return fmt.Sprintf("object type %q is not %q", e.actualType, e.expectedType)
}

func stackComponentPatchers(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) ([]objectPatcher, error) {
	rbacResourceName := ms.Name + "-prometheus"
	rbacVerbs := []string{"get", "list", "watch"}
	additionalScrapeConfigsSecretName := ms.Name + "-prometheus-additional-scrape-configs"

	return []objectPatcher{
		{
			empty: func() client.Object {
				sa := newServiceAccount(ms)
				return &corev1.ServiceAccount{
					TypeMeta:   sa.TypeMeta,
					ObjectMeta: sa.ObjectMeta,
				}
			},
			patch: func(object client.Object) (client.Object, error) {
				return newServiceAccount(ms), nil
			},
		},
		{
			empty: func() client.Object {
				role := newRole(ms, rbacResourceName, rbacVerbs)
				return &rbacv1.Role{
					TypeMeta:   role.TypeMeta,
					ObjectMeta: role.ObjectMeta,
				}
			},
			patch: func(object client.Object) (client.Object, error) {
				role := newRole(ms, rbacResourceName, rbacVerbs)

				if object == nil {
					return role, nil
				}

				desired, ok := object.DeepCopyObject().(*rbacv1.Role)
				if !ok {
					return nil, NewObjectTypeError(role, object)
				}

				desired.Rules = role.Rules
				return desired, nil
			},
		},
		{
			empty: func() client.Object {
				rb := newRoleBinding(ms, rbacResourceName)
				return &rbacv1.RoleBinding{
					TypeMeta:   rb.TypeMeta,
					ObjectMeta: rb.ObjectMeta,
				}
			},
			patch: func(object client.Object) (client.Object, error) {
				roleBinding := newRoleBinding(ms, rbacResourceName)

				if object == nil {
					return roleBinding, nil
				}

				desired, ok := object.DeepCopyObject().(*rbacv1.RoleBinding)
				if !ok {
					return nil, NewObjectTypeError(roleBinding, object)
				}

				desired.Subjects = roleBinding.Subjects
				desired.RoleRef = roleBinding.RoleRef
				return desired, nil
			},
		},
		{
			empty: func() client.Object {
				secret := newAdditionalScrapeConfigsSecret(additionalScrapeConfigsSecretName, ms.Namespace)
				return &corev1.Secret{
					TypeMeta:   secret.TypeMeta,
					ObjectMeta: secret.ObjectMeta,
				}
			},
			patch: func(object client.Object) (client.Object, error) {
				secret := newAdditionalScrapeConfigsSecret(additionalScrapeConfigsSecretName, ms.Namespace)
				if object == nil {
					return secret, nil
				}

				desired, ok := object.DeepCopyObject().(*corev1.Secret)
				if !ok {
					return nil, NewObjectTypeError(secret, object)
				}

				desired.StringData = secret.StringData

				return desired, nil
			},
		},
		{
			empty: func() client.Object {
				prometheus := newPrometheus(ms, rbacResourceName, additionalScrapeConfigsSecretName, instanceSelectorKey, instanceSelectorValue)
				return &monv1.Prometheus{
					TypeMeta:   prometheus.TypeMeta,
					ObjectMeta: prometheus.ObjectMeta,
				}
			},
			patch: func(object client.Object) (client.Object, error) {
				prometheus := newPrometheus(ms, rbacResourceName, additionalScrapeConfigsSecretName, instanceSelectorKey, instanceSelectorValue)

				if object == nil {
					return prometheus, nil
				}

				desired, ok := object.DeepCopyObject().(*monv1.Prometheus)
				if !ok {
					return nil, NewObjectTypeError(prometheus, object)
				}

				desired.Spec = prometheus.Spec
				return desired, nil
			},
		},

		{
			empty: func() client.Object {
				dataSource := newGrafanaDataSource(ms)
				return &grafanav1alpha1.GrafanaDataSource{
					TypeMeta:   dataSource.TypeMeta,
					ObjectMeta: dataSource.ObjectMeta,
				}
			},
			patch: func(object client.Object) (client.Object, error) {
				dataSource := newGrafanaDataSource(ms)
				if object == nil {
					return dataSource, nil
				}

				desired, ok := object.DeepCopyObject().(*grafanav1alpha1.GrafanaDataSource)
				if !ok {
					return nil, NewObjectTypeError(dataSource, object)
				}

				desired.Spec = dataSource.Spec
				return desired, nil
			},
		},
	}, nil
}

func newGrafanaDataSource(ms *stack.MonitoringStack) *grafanav1alpha1.GrafanaDataSource {
	datasourceName := fmt.Sprintf("ms-%s-%s", ms.GetNamespace(), ms.GetName())
	prometheusURL := fmt.Sprintf("prometheus-operated.%s:9090", ms.GetNamespace())
	return &grafanav1alpha1.GrafanaDataSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "integreatly.org/v1alpha1",
			Kind:       "GrafanaDataSource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      datasourceName,
			Namespace: grafana_operator.Namespace,
		},
		Spec: grafanav1alpha1.GrafanaDataSourceSpec{
			Name: datasourceName,
			Datasources: []grafanav1alpha1.GrafanaDataSourceFields{
				{
					Name:    datasourceName,
					Type:    "prometheus",
					Access:  "proxy",
					Url:     prometheusURL,
					Version: 1,
				},
			},
		},
	}
}

func newRole(ms *stack.MonitoringStack, rbacResourceName string, rbacVerbs []string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbacResourceName,
			Namespace: ms.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "pods"},
				Verbs:     rbacVerbs,
			},
			{
				APIGroups: []string{"extensions", "networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     rbacVerbs,
			},
		},
	}
}

func newServiceAccount(ms *stack.MonitoringStack) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name + "-prometheus",
			Namespace: ms.Namespace,
		},
	}
}

func newPrometheus(
	ms *stack.MonitoringStack,
	rbacResourceName string,
	additionalScrapeConfigsSecretName string,
	instanceSelectorKey string,
	instanceSelectorValue string,
) *monv1.Prometheus {
	prometheusSelector := ms.Spec.ResourceSelector
	if prometheusSelector == nil {
		prometheusSelector = &metav1.LabelSelector{}
	}
	prometheus := &monv1.Prometheus{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Prometheus",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name,
			Namespace: ms.Namespace,
			Labels:    prometheusLabels(ms.Name, instanceSelectorKey, instanceSelectorValue),
		},

		Spec: monv1.PrometheusSpec{
			// Prometheus does not use an Enum for LogLevel, so need to convert to string
			LogLevel: string(ms.Spec.LogLevel),

			Retention: ms.Spec.Retention,
			Resources: ms.Spec.Resources,

			ServiceAccountName: rbacResourceName,

			ServiceMonitorSelector:          prometheusSelector,
			ServiceMonitorNamespaceSelector: nil,
			PodMonitorSelector:              prometheusSelector,
			PodMonitorNamespaceSelector:     nil,

			// Prometheus should be configured for self-scraping through a static job.
			// It avoids the need to synthesize a ServiceMonitor with labels that will match
			// what the user defines in the monitoring stacks's resourceSelector field.
			AdditionalScrapeConfigs: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: additionalScrapeConfigsSecretName,
				},
				Key: AdditionalScrapeConfigsSelfScrapeKey,
			},
		},
	}
	return prometheus
}

func newRoleBinding(ms *stack.MonitoringStack, rbacResourceName string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbacResourceName,
			Namespace: ms.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      rbacResourceName,
				Namespace: ms.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     rbacResourceName,
		},
	}
	return roleBinding
}

func newAdditionalScrapeConfigsSecret(name string, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: map[string]string{
			AdditionalScrapeConfigsSelfScrapeKey: `- job_name: prometheus-self
  honor_labels: true
  kubernetes_sd_configs:
  - role: endpoints
    namespaces:
      names:
      - ` + namespace + `
  relabel_configs:
  - source_labels:
    - job
    target_label: __tmp_prometheus_job_name
  - action: keep
    source_labels:
    - __meta_kubernetes_service_label_operated_prometheus
    regex: "true"
  - action: keep
    source_labels:
    - __meta_kubernetes_endpoint_port_name
    regex: web
  - source_labels:
    - __meta_kubernetes_endpoint_address_target_kind
    - __meta_kubernetes_endpoint_address_target_name
    separator: ;
    regex: Node;(.*)
    replacement: ${1}
    target_label: node
  - source_labels:
    - __meta_kubernetes_endpoint_address_target_kind
    - __meta_kubernetes_endpoint_address_target_name
    separator: ;
    regex: Pod;(.*)
    replacement: ${1}
    target_label: pod
  - source_labels:
    - __meta_kubernetes_namespace
    target_label: namespace
  - source_labels:
    - __meta_kubernetes_service_name
    target_label: service
  - source_labels:
    - __meta_kubernetes_pod_name
    target_label: pod
  - source_labels:
    - __meta_kubernetes_pod_container_name
    target_label: container
  - source_labels:
    - __meta_kubernetes_service_name
    target_label: job
    replacement: ${1}
  - target_label: endpoint
    replacement: web
  - source_labels:
    - __address__
    target_label: __tmp_hash
    modulus: 1
    action: hashmod
  - source_labels:
    - __tmp_hash
    regex: 0
    action: keep`,
		},
	}
}

func prometheusLabels(msName string, instanceSelectorKey string, instanceSelectorValue string) map[string]string {
	return map[string]string{
		instanceSelectorKey:                    instanceSelectorValue,
		"monitoring.rhobs.io/monitoring-stack": msName,
	}
}
