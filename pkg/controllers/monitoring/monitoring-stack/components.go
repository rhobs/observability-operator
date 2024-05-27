package monitoringstack

import (
	"reflect"
	"slices"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const AdditionalScrapeConfigsSelfScrapeKey = "self-scrape-config"
const PrometheusUserFSGroupID = int64(65534)
const AlertmanagerUserFSGroupID = int64(65535)

func stackComponentReconcilers(
	ms *stack.MonitoringStack,
	instanceSelectorKey string,
	instanceSelectorValue string,
	thanos ThanosConfiguration,
	prometheus PrometheusConfiguration,
	alertmanager AlertmanagerConfiguration,
) []reconciler.Reconciler {
	prometheusName := ms.Name + "-prometheus"
	alertmanagerName := ms.Name + "-alertmanager"
	rbacVerbs := []string{"get", "list", "watch"}
	additionalScrapeConfigsSecretName := ms.Name + "-prometheus-additional-scrape-configs"
	hasNsSelector := ms.Spec.NamespaceSelector != nil
	deployAlertmanager := !ms.Spec.AlertmanagerConfig.Disabled

	return []reconciler.Reconciler{
		// Prometheus Deployment
		reconciler.NewUpdater(newServiceAccount(prometheusName, ms.Namespace), ms),
		reconciler.NewUpdater(newPrometheusClusterRole(ms, prometheusName, rbacVerbs), ms),
		reconciler.NewUpdater(newAdditionalScrapeConfigsSecret(ms, additionalScrapeConfigsSecretName), ms),
		reconciler.NewUpdater(newPrometheus(ms, prometheusName,
			additionalScrapeConfigsSecretName,
			instanceSelectorKey, instanceSelectorValue,
			thanos, prometheus), ms),
		reconciler.NewUpdater(newPrometheusService(ms, instanceSelectorKey, instanceSelectorValue), ms),
		reconciler.NewUpdater(newThanosSidecarService(ms, instanceSelectorKey, instanceSelectorValue), ms),
		reconciler.NewOptionalUpdater(newPrometheusPDB(ms, instanceSelectorKey, instanceSelectorValue), ms,
			*ms.Spec.PrometheusConfig.Replicas > 1),

		// Alertmanager Deployment
		reconciler.NewOptionalUpdater(newServiceAccount(alertmanagerName, ms.Namespace), ms, deployAlertmanager),
		// create clusterrolebinding if nsSelector's present otherwise a rolebinding
		reconciler.NewOptionalUpdater(newClusterRoleBinding(ms, prometheusName), ms, hasNsSelector),
		reconciler.NewOptionalUpdater(newRoleBindingForClusterRole(ms, prometheusName), ms, !hasNsSelector),

		reconciler.NewOptionalUpdater(newAlertManagerClusterRole(ms, alertmanagerName, rbacVerbs), ms, deployAlertmanager),

		// create clusterrolebinding if alertmanager is enabled and namespace selector is also present in MonitoringStack
		reconciler.NewOptionalUpdater(newClusterRoleBinding(ms, alertmanagerName), ms, deployAlertmanager && hasNsSelector),
		reconciler.NewOptionalUpdater(newRoleBindingForClusterRole(ms, alertmanagerName), ms, deployAlertmanager && !hasNsSelector),

		reconciler.NewOptionalUpdater(newAlertmanager(ms, alertmanagerName, instanceSelectorKey, instanceSelectorValue, alertmanager), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newAlertmanagerService(ms, instanceSelectorKey, instanceSelectorValue), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newAlertmanagerPDB(ms, instanceSelectorKey, instanceSelectorValue), ms, deployAlertmanager),
	}
}

func newPrometheusClusterRole(ms *stack.MonitoringStack, rbacResourceName string, rbacVerbs []string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: rbacResourceName,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"services", "endpoints", "pods"},
			Verbs:     rbacVerbs,
		}, {
			APIGroups: []string{"extensions", "networking.k8s.io"},
			Resources: []string{"ingresses"},
			Verbs:     rbacVerbs,
		}, {
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			ResourceNames: []string{"nonroot", "nonroot-v2"},
			Verbs:         []string{"use"},
		}},
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
	thanosCfg ThanosConfiguration,
	prometheusCfg PrometheusConfiguration,
) *monv1.Prometheus {
	prometheusSelector := ms.Spec.ResourceSelector

	config := ms.Spec.PrometheusConfig

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
				Replicas: config.Replicas,

				PodMetadata: &monv1.EmbeddedObjectMetadata{
					Labels: podLabels("prometheus", ms.Name),
				},

				// Prometheus does not use an Enum for LogLevel, so need to convert to string
				LogLevel: string(ms.Spec.LogLevel),

				Resources: ms.Spec.Resources,

				ServiceAccountName: rbacResourceName,

				ServiceMonitorSelector:          prometheusSelector,
				ServiceMonitorNamespaceSelector: ms.Spec.NamespaceSelector,
				PodMonitorSelector:              prometheusSelector,
				PodMonitorNamespaceSelector:     ms.Spec.NamespaceSelector,
				ProbeSelector:                   prometheusSelector,
				ProbeNamespaceSelector:          ms.Spec.NamespaceSelector,
				ScrapeConfigSelector:            prometheusSelector,
				ScrapeConfigNamespaceSelector:   ms.Spec.NamespaceSelector,
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
					FSGroup:      ptr.To(PrometheusUserFSGroupID),
					RunAsNonRoot: ptr.To(true),
					RunAsUser:    ptr.To(PrometheusUserFSGroupID),
				},
				RemoteWrite:               config.RemoteWrite,
				ExternalLabels:            config.ExternalLabels,
				EnableRemoteWriteReceiver: config.EnableRemoteWriteReceiver,
				EnableFeatures: func() []string {
					if config.EnableOtlpHttpReceiver != nil && *config.EnableOtlpHttpReceiver {
						return []string{"otlp-write-receiver"}
					}
					return []string{}
				}(),
			},
			Retention:             ms.Spec.Retention,
			RuleSelector:          prometheusSelector,
			RuleNamespaceSelector: ms.Spec.NamespaceSelector,
			Thanos: &monv1.ThanosSpec{
				Image: ptr.To(thanosCfg.Image),
			},
		},
	}

	if ms.Spec.PrometheusConfig.WebTLSConfig != nil {
		tlsConfig := ms.Spec.PrometheusConfig.WebTLSConfig

		prometheus.Spec.CommonPrometheusFields.Web = &monv1.PrometheusWebSpec{
			WebConfigFileFields: monv1.WebConfigFileFields{
				TLSConfig: &monv1.WebTLSConfig{
					KeySecret: corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: tlsConfig.Key.Name,
						},
						Key: tlsConfig.Key.Key,
					},
					Cert: monv1.SecretOrConfigMap{
						Secret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: tlsConfig.Cert.Name,
							},
							Key: tlsConfig.Cert.Key,
						},
					},
				},
			},
		}
		// Add a CA secret to use later for the self-scraping job
		prometheus.Spec.Secrets = append(prometheus.Spec.Secrets, tlsConfig.CA.Name)
	}

	if prometheusCfg.Image != "" {
		prometheus.Spec.CommonPrometheusFields.Image = ptr.To(prometheusCfg.Image)
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

	if config.ScrapeInterval != nil {
		prometheus.Spec.ScrapeInterval = *ms.Spec.PrometheusConfig.ScrapeInterval
	}
	if len(ms.Spec.PrometheusConfig.Secrets) > 0 {
		for _, secret := range ms.Spec.PrometheusConfig.Secrets {
			if !slices.Contains(prometheus.Spec.Secrets, secret) {
				prometheus.Spec.Secrets = append(prometheus.Spec.Secrets, secret)
			}
		}
	}

	return prometheus
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

func newRoleBindingForClusterRole(ms *stack.MonitoringStack, rbacResourceName string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbacResourceName,
			Namespace: ms.Namespace,
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  corev1.SchemeGroupVersion.Group,
			Kind:      "ServiceAccount",
			Name:      rbacResourceName,
			Namespace: ms.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     rbacResourceName,
		},
	}
	return roleBinding
}

func newClusterRoleBinding(ms *stack.MonitoringStack, rbacResourceName string) *rbacv1.ClusterRoleBinding {
	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: rbacResourceName,
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  corev1.SchemeGroupVersion.Group,
			Kind:      "ServiceAccount",
			Name:      rbacResourceName,
			Namespace: ms.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     rbacResourceName,
		},
	}
	return roleBinding
}

func newPrometheusService(ms *stack.MonitoringStack, instanceSelectorKey string, instanceSelectorValue string) *corev1.Service {
	name := ms.Name + "-prometheus"
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(name, ms.Name, instanceSelectorKey, instanceSelectorValue),
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
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
			Labels:    objectLabels(name, ms.Name, instanceSelectorKey, instanceSelectorValue),
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
	prometheusScheme := "http"
	prometheusTLSConfig := ""

	if ms.Spec.PrometheusConfig.WebTLSConfig != nil {
		promCASecret := ms.Spec.PrometheusConfig.WebTLSConfig.CA
		prometheusScheme = "https"
		prometheusTLSConfig = `
  tls_config:
    ca_file: /etc/prometheus/secrets/` + promCASecret.Name + `/` + promCASecret.Key + `
    server_name: ` + ms.Name + `-prometheus
`
	}
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
  scheme: ` + prometheusScheme +
  prometheusTLSConfig + `
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
  scheme: http
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
