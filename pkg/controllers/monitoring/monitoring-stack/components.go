package monitoringstack

import (
	"fmt"
	"path/filepath"
	"reflect"

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

const (
	AdditionalScrapeConfigsSelfScrapeKey = "self-scrape-config"
	PrometheusUserFSGroupID              = int64(65534)
	AlertmanagerUserFSGroupID            = int64(65535)

	prometheusSecretsMountPoint = "/etc/prometheus/secrets"
)

var (
	rbacVerbs = []string{"get", "list", "watch"}
)

func stackComponentCleanup(ms *stack.MonitoringStack) []reconciler.Reconciler {
	prometheusName := ms.Name + "-prometheus"
	alertmanagerName := ms.Name + "-alertmanager"
	return []reconciler.Reconciler{
		reconciler.NewDeleter(newPrometheusClusterRole(prometheusName, rbacVerbs)),
		reconciler.NewDeleter(newClusterRoleBinding(ms, prometheusName)),
		reconciler.NewDeleter(newRoleBindingForClusterRole(ms, prometheusName)),
		reconciler.NewDeleter(newAlertManagerClusterRole(alertmanagerName, rbacVerbs)),
		reconciler.NewDeleter(newClusterRoleBinding(ms, alertmanagerName)),
		reconciler.NewDeleter(newRoleBindingForClusterRole(ms, alertmanagerName)),
	}
}

func stackComponentReconcilers(
	ms *stack.MonitoringStack,
	thanos ThanosConfiguration,
	prometheus PrometheusConfiguration,
	alertmanager AlertmanagerConfiguration,
) []reconciler.Reconciler {
	prometheusName := ms.Name + "-prometheus"
	alertmanagerName := ms.Name + "-alertmanager"
	additionalScrapeConfigsSecretName := ms.Name + "-self-scrape"
	hasNsSelector := ms.Spec.NamespaceSelector != nil
	createCRB := hasNsSelector && ms.Spec.CreateClusterRoleBindings == stack.CreateClusterRoleBindings
	deployAlertmanager := !ms.Spec.AlertmanagerConfig.Disabled

	return []reconciler.Reconciler{
		// Create RBAC
		reconciler.NewUpdater(newServiceAccount(prometheusName, ms.Namespace), ms),
		reconciler.NewOptionalUpdater(newServiceAccount(alertmanagerName, ms.Namespace), ms, deployAlertmanager),

		reconciler.NewUpdater(newPrometheusClusterRole(prometheusName, rbacVerbs), ms),
		// create clusterrolebinding if nsSelector's present otherwise a rolebinding
		reconciler.NewOptionalUpdater(newClusterRoleBinding(ms, prometheusName), ms, createCRB),
		reconciler.NewOptionalUpdater(newRoleBindingForClusterRole(ms, prometheusName), ms, !hasNsSelector),

		reconciler.NewOptionalUpdater(newAlertManagerClusterRole(alertmanagerName, rbacVerbs), ms, deployAlertmanager),
		// create clusterrolebinding if alertmanager is enabled and namespace selector is also present in MonitoringStack
		reconciler.NewOptionalUpdater(newClusterRoleBinding(ms, alertmanagerName), ms, deployAlertmanager && createCRB),
		reconciler.NewOptionalUpdater(newRoleBindingForClusterRole(ms, alertmanagerName), ms, deployAlertmanager && !hasNsSelector),

		// Prometheus Deployment
		reconciler.NewUpdater(newPrometheus(ms, prometheusName,
			additionalScrapeConfigsSecretName,
			thanos, prometheus), ms),
		reconciler.NewUpdater(newPrometheusService(ms), ms),
		reconciler.NewUpdater(newThanosSidecarService(ms), ms),
		reconciler.NewUpdater(newAdditionalScrapeConfigsSecret(ms, additionalScrapeConfigsSecretName), ms),
		reconciler.NewOptionalUpdater(newPrometheusPDB(ms), ms,
			*ms.Spec.PrometheusConfig.Replicas > 1),

		// Alertmanager Deployment
		reconciler.NewOptionalUpdater(newAlertmanager(ms, alertmanagerName, alertmanager), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newAlertmanagerService(ms), ms, deployAlertmanager),
		reconciler.NewOptionalUpdater(newAlertmanagerPDB(ms), ms, deployAlertmanager && *ms.Spec.AlertmanagerConfig.Replicas > 1),
	}
}

func newPrometheusClusterRole(rbacResourceName string, rbacVerbs []string) *rbacv1.ClusterRole {
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
				NodeSelector:                    ms.Spec.NodeSelector,
				Tolerations:                     ms.Spec.Tolerations,
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
				EnableOTLPReceiver:        config.EnableOtlpHttpReceiver,
			},
			Retention:             ms.Spec.Retention,
			RetentionSize:         ms.Spec.RetentionSize,
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
							Name: tlsConfig.PrivateKey.Name,
						},
						Key: tlsConfig.PrivateKey.Key,
					},
					Cert: monv1.SecretOrConfigMap{
						Secret: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: tlsConfig.Certificate.Name,
							},
							Key: tlsConfig.Certificate.Key,
						},
					},
				},
			},
		}
		// Add a CA secret to use later for the self-scraping job
		prometheus.Spec.Secrets = append(prometheus.Spec.Secrets, tlsConfig.CertificateAuthority.Name)
	}

	if prometheusCfg.Image != "" {
		prometheus.Spec.CommonPrometheusFields.Image = ptr.To(prometheusCfg.Image)
	}

	if !ms.Spec.AlertmanagerConfig.Disabled {
		prometheus.Spec.Alerting = &monv1.AlertingSpec{
			Alertmanagers: []monv1.AlertmanagerEndpoints{
				{
					Name:      ms.Name + "-alertmanager",
					Namespace: ptr.To(ms.Namespace),
					Scheme:    ptr.To(monv1.Scheme("http")),
					Port:      intstr.FromString("web"),
				},
			},
		}
		if ms.Spec.AlertmanagerConfig.WebTLSConfig != nil {
			caSecret := ms.Spec.AlertmanagerConfig.WebTLSConfig.CertificateAuthority

			prometheus.Spec.Secrets = append(prometheus.Spec.Secrets, caSecret.Name)

			prometheus.Spec.Alerting.Alertmanagers[0].Scheme = ptr.To(monv1.Scheme("https"))
			prometheus.Spec.Alerting.Alertmanagers[0].TLSConfig = &monv1.TLSConfig{
				SafeTLSConfig: monv1.SafeTLSConfig{
					ServerName: ptr.To(ms.Name + "-alertmanager"),
				},
				TLSFilesConfig: monv1.TLSFilesConfig{
					CAFile: filepath.Join(prometheusSecretsMountPoint, caSecret.Name, caSecret.Key),
				},
			}
		}
	}

	if config.ScrapeInterval != nil {
		prometheus.Spec.ScrapeInterval = *ms.Spec.PrometheusConfig.ScrapeInterval
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

func newPrometheusService(ms *stack.MonitoringStack) *corev1.Service {
	name := ms.Name + "-prometheus"
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
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

func newThanosSidecarService(ms *stack.MonitoringStack) *corev1.Service {
	name := ms.Name + "-thanos-sidecar"
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
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
	var (
		prometheusScheme     = "http"
		prometheusCAFile     string
		prometheusServerName string

		alertmanagerScheme     = "http"
		alertmanagerCAFile     string
		alertmanagerServerName string
	)

	if ms.Spec.PrometheusConfig.WebTLSConfig != nil {
		promCASecret := ms.Spec.PrometheusConfig.WebTLSConfig.CertificateAuthority
		prometheusScheme = "https"
		prometheusCAFile = filepath.Join(prometheusSecretsMountPoint, promCASecret.Name, promCASecret.Key)
		prometheusServerName = fmt.Sprintf("%s-prometheus", ms.Name)
	}

	if ms.Spec.AlertmanagerConfig.WebTLSConfig != nil {
		amCASecret := ms.Spec.AlertmanagerConfig.WebTLSConfig.CertificateAuthority
		alertmanagerScheme = "https"
		alertmanagerCAFile = filepath.Join(prometheusSecretsMountPoint, amCASecret.Name, amCASecret.Key)
		alertmanagerServerName = fmt.Sprintf("%s-alertmanager", ms.Name)
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
			AdditionalScrapeConfigsSelfScrapeKey: fmt.Sprintf(`
- job_name: prometheus-self
  scheme: %s
  tls_config:
    ca_file: %q
    server_name: %q
  relabel_configs:
  - action: keep
    source_labels:
    - __meta_kubernetes_service_label_app_kubernetes_io_name
    regex: %s
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
      - %s
- job_name: alertmanager-self
  scrape_interval: 30s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: %s
  tls_config:
    ca_file: %q
    server_name: %q
  relabel_configs:
  - source_labels:
    - __meta_kubernetes_service_label_app_kubernetes_io_name
    separator: ;
    regex: %s
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
    namespaces:
      names:
      - %s`,
				prometheusScheme,
				prometheusCAFile,
				prometheusServerName,
				fmt.Sprintf("%s-prometheus", ms.Name),
				ms.Namespace,
				alertmanagerScheme,
				alertmanagerCAFile,
				alertmanagerServerName,
				fmt.Sprintf("%s-alertmanager", ms.Name),
				ms.Namespace,
			),
		},
	}
}

func newPrometheusPDB(ms *stack.MonitoringStack) *policyv1.PodDisruptionBudget {
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

func podLabels(component string, msName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": component,
		"app.kubernetes.io/part-of":   msName,
	}
}
