package monitoringstack

import (
	"reflect"

	"github.com/rhobs/observability-operator/pkg/reconciler"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AdditionalScrapeConfigsSelfScrapeKey = "self-scrape-config"
const PrometheusUserFSGroupID = 65534
const AlertmanagerUserFSGroupID = 65535

func stackComponentReconcilers(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) []reconciler.Reconciler {
	prometheusRBACResourceName := ms.Name + "-prometheus"
	rbacVerbs := []string{"get", "list", "watch"}
	additionalScrapeConfigsSecretName := ms.Name + "-prometheus-additional-scrape-configs"

	alertmanagerRBACResourceName := ms.Name + "-alertmanager"
	deployAlertmanager := !ms.Spec.AlertmanagerConfig.Disabled

	return []reconciler.Reconciler{
		reconciler.NewUpdater(newServiceAccount(prometheusRBACResourceName, ms.Namespace), ms),
		reconciler.NewUpdater(newPrometheusRole(ms, prometheusRBACResourceName, rbacVerbs), ms),
		reconciler.NewUpdater(newRoleBinding(ms, prometheusRBACResourceName), ms),
		reconciler.NewUpdater(newAdditionalScrapeConfigsSecret(ms, additionalScrapeConfigsSecretName), ms),
		reconciler.NewUpdater(newPrometheus(ms, prometheusRBACResourceName, additionalScrapeConfigsSecretName, instanceSelectorKey, instanceSelectorValue), ms),
		reconciler.NewUpdater(newPrometheusService(ms, instanceSelectorKey, instanceSelectorValue), ms),
		reconciler.NewUpdater(newThanosSidecarService(ms, instanceSelectorKey, instanceSelectorValue), ms),
		reconciler.NewOptionalUpdater(newPrometheusPDB(ms, instanceSelectorKey, instanceSelectorValue), ms,
			*ms.Spec.PrometheusConfig.Replicas > 1),

		// Alertmanager
		reconciler.NewOptionalUpdater(newServiceAccount(alertmanagerRBACResourceName, ms.Namespace), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newAlertmanagerRole(ms, alertmanagerRBACResourceName, rbacVerbs), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newRoleBinding(ms, alertmanagerRBACResourceName), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(
			newAlertmanager(ms, alertmanagerRBACResourceName, instanceSelectorKey, instanceSelectorValue),
			ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newAlertmanagerService(ms, instanceSelectorKey, instanceSelectorValue), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newAlertmanagerPDB(ms, instanceSelectorKey, instanceSelectorValue), ms, deployAlertmanager),
	}
}

func newPrometheusRole(ms *stack.MonitoringStack, rbacResourceName string, rbacVerbs []string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
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
			{
				APIGroups:     []string{"security.openshift.io"},
				Resources:     []string{"securitycontextconstraints"},
				ResourceNames: []string{"nonroot", "nonroot-v2"},
				Verbs:         []string{"use"},
			},
		},
	}
}

func newServiceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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

	config := ms.Spec.PrometheusConfig
	// NOTE: using hash due to upstream PO bug - 4944
	tlsSecretName := ms.PrometheusHash() + "-tls"
	skipValidation := `tls_config: { insecure_skip_verify: true, }`

	prometheus := &monv1.Prometheus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "Prometheus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ms.Name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(ms.Name, ms.Name, instanceSelectorKey, instanceSelectorValue),
		},

		Spec: monv1.PrometheusSpec{
			CommonPrometheusFields: monv1.CommonPrometheusFields{
				Containers: []v1.Container{{
					Name: "thanos-sidecar",
					Args: []string{
						"sidecar",
						"--prometheus.url=https://localhost:9090/",
						"--prometheus.http-client", skipValidation,
						"--grpc-address=:10901",
						"--http-address=:10902",
						"--log.level=debug",
					},
				}},
				Replicas: config.Replicas,

				PodMetadata: &monv1.EmbeddedObjectMetadata{
					Labels: podLabels("prometheus", ms.Name),
				},

				// Prometheus does not use an Enum for LogLevel, so need to convert to string
				LogLevel: string(ms.Spec.LogLevel),

				Resources: ms.Spec.Resources,

				ServiceAccountName: rbacResourceName,

				ServiceMonitorSelector:          prometheusSelector,
				ServiceMonitorNamespaceSelector: nil,
				PodMonitorSelector:              prometheusSelector,
				PodMonitorNamespaceSelector:     nil,
				Affinity: &corev1.Affinity{
					PodAntiAffinity: &corev1.PodAntiAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
							{
								TopologyKey: "kubernetes.io/hostname",
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: podLabels("prometheus", ms.Name),
								},
							},
						},
					},
				},

				// Prometheus should be configured for self-scraping through a static job.
				// It avoids the need to synthesize a ServiceMonitor with labels that will match
				// what the user defines in the monitoring stacks's resourceSelector field.
				AdditionalScrapeConfigs: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: additionalScrapeConfigsSecretName,
					},
					Key: AdditionalScrapeConfigsSelfScrapeKey,
				},
				Storage: storageForPVC(config.PersistentVolumeClaim),
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup:      pointer.Int64(PrometheusUserFSGroupID),
					RunAsNonRoot: pointer.Bool(true),
					RunAsUser:    pointer.Int64(PrometheusUserFSGroupID),
				},
				RemoteWrite:               config.RemoteWrite,
				ExternalLabels:            config.ExternalLabels,
				EnableRemoteWriteReceiver: config.EnableRemoteWriteReceiver,
				Secrets:                   []string{tlsSecretName},
				Web: &monv1.PrometheusWebSpec{
					TLSConfig: webTLSConfigForServiceCA(tlsSecretName)},
			},
			Retention:             ms.Spec.Retention,
			RuleSelector:          prometheusSelector,
			RuleNamespaceSelector: nil,
			Thanos: &monv1.ThanosSpec{
				BaseImage: stringPtr("quay.io/thanos/thanos"),
				Version:   stringPtr("v0.24.0"),
			},
		},
	}

	if !ms.Spec.AlertmanagerConfig.Disabled {
		prometheus.Spec.Alerting = &monv1.AlertingSpec{
			Alertmanagers: []monv1.AlertmanagerEndpoints{
				{
					APIVersion: "v2",
					Name:       ms.Name + "-alertmanager",
					Namespace:  ms.Namespace,
					Scheme:     "http",
					Port:       intstr.FromString("web"),
				},
			},
		}
	}

	return prometheus
}

func webTLSConfigForServiceCA(secret string) *monv1.WebTLSConfig {

	return &monv1.WebTLSConfig{
		Cert: monv1.SecretOrConfigMap{
			Secret: &corev1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: secret},
				Key:                  "tls.crt", // set by service-ca operator
			},
		},
		KeySecret: corev1.SecretKeySelector{
			LocalObjectReference: v1.LocalObjectReference{Name: secret},
			Key:                  "tls.key",
		},
	}
}

func storageForPVC(pvc *corev1.PersistentVolumeClaimSpec) *monv1.StorageSpec {
	if pvc == nil {
		return nil
	}

	if reflect.DeepEqual(*pvc, corev1.PersistentVolumeClaimSpec{}) {
		return nil
	}

	return &monv1.StorageSpec{
		VolumeClaimTemplate: monv1.EmbeddedPersistentVolumeClaim{
			Spec: *pvc,
		},
	}
}

func newRoleBinding(ms *stack.MonitoringStack, rbacResourceName string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbacResourceName,
			Namespace: ms.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  corev1.SchemeGroupVersion.Group,
				Kind:      "ServiceAccount",
				Name:      rbacResourceName,
				Namespace: ms.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     rbacResourceName,
		},
	}
	return roleBinding
}

func newPrometheusService(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) *corev1.Service {
	name := ms.PrometheusName()

	// NOTE: workaround for PO bug: https://github.com/prometheus-operator/prometheus-operator/issues/4944
	tlsSecretName := ms.PrometheusHash() + "-tls"

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(name, ms.Name, instanceSelectorKey, instanceSelectorValue),
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": tlsSecretName,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: podLabels("prometheus", ms.Name),
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
				},
			},
		},
	}
}

func newThanosSidecarService(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) *corev1.Service {
	name := ms.Name + "-thanos-sidecar"
	// NOTE: workaround for PO bug: https://github.com/prometheus-operator/prometheus-operator/issues/4944
	tlsSecretName := ms.PrometheusHash() + "-tls"
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(name, ms.Name, instanceSelectorKey, instanceSelectorValue),
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": tlsSecretName,
			},
		},
		Spec: corev1.ServiceSpec{

			// NOTE: Setting this to "None" makes a "headless service" (no virtual
			// IP), which is useful when direct endpoint connections are preferred
			// and proxying is not required.
			// This is a required for thanos service-discovery to work correctly
			ClusterIP: "None",

			Selector: podLabels("prometheus", ms.Name),
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Port:       10901,
					TargetPort: intstr.FromString("grpc"),
				},
			},
		},
	}
}

func newAdditionalScrapeConfigsSecret(ms *stack.MonitoringStack, name string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
		},
		StringData: map[string]string{
			AdditionalScrapeConfigsSelfScrapeKey: `
- job_name: prometheus-self
  honor_labels: true
  scheme: https
  tls_config:
    insecure_skip_verify: true
  relabel_configs:
  - action: keep
    source_labels:
    - __meta_kubernetes_service_label_app_kubernetes_io_name
    regex: ` + ms.Name + `-prometheus
  - action: keep
    source_labels:
    - __meta_kubernetes_endpoint_port_name
    regex: web
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
  - target_label: endpoint
    replacement: web
  kubernetes_sd_configs:
  - role: endpoints
    namespaces:
      names:
      - ` + ms.Namespace + `
- job_name: alertmanager-self
  honor_timestamps: true
  scrape_interval: 30s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: https
  tls_config:
    insecure_skip_verify: true
  follow_redirects: true
  relabel_configs:
  - source_labels:
    - __meta_kubernetes_service_label_app_kubernetes_io_name
    separator: ;
    regex: ` + ms.Name + `-alertmanager
    replacement: $1
    action: keep
  - source_labels: [__meta_kubernetes_endpoint_port_name]
    separator: ;
    regex: web
    replacement: $1
    action: keep
  - source_labels: [__meta_kubernetes_namespace]
    separator: ;
    regex: (.*)
    target_label: namespace
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_service_name]
    separator: ;
    regex: (.*)
    target_label: service
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    separator: ;
    regex: (.*)
    target_label: pod
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_pod_container_name]
    separator: ;
    regex: (.*)
    target_label: container
    replacement: $1
    action: replace
  - separator: ;
    regex: (.*)
    target_label: endpoint
    replacement: web
    action: replace
  kubernetes_sd_configs:
  - role: endpoints
    kubeconfig_file: ""
    follow_redirects: true
    namespaces:
      names:
      - ` + ms.Namespace,
		},
	}
}

func newPrometheusPDB(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) *policyv1.PodDisruptionBudget {
	name := ms.Name + "-prometheus"
	selector := podLabels("prometheus", ms.Name)

	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyv1.SchemeGroupVersion.String(),
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(name, ms.Name, instanceSelectorKey, instanceSelectorValue),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
		},
	}
}

func objectLabels(name string, msName string, instanceSelectorKey string, instanceSelectorValue string) map[string]string {
	return map[string]string{
		instanceSelectorKey:         instanceSelectorValue,
		"app.kubernetes.io/name":    name,
		"app.kubernetes.io/part-of": msName,
	}
}

func podLabels(component string, msName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": component,
		"app.kubernetes.io/part-of":   msName,
	}
}

func stringPtr(s string) *string {
	return &s
}
