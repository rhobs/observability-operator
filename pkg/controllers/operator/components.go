package operator

import (
	"fmt"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const (
	name = "observability-operator"
)

func operatorComponentReconcilers(owner metav1.Object, namespace string) []reconciler.Reconciler {
	return []reconciler.Reconciler{
		reconciler.NewUpdater(newServiceMonitor(namespace), owner),
		reconciler.NewUpdater(newPrometheusRole(namespace), owner),
		reconciler.NewUpdater(newRoleBindingForPrometheusRole(namespace), owner),
	}
}

func newServiceMonitor(namespace string) *monv1.ServiceMonitor {
	return &monv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/component":  "operator",
				"app.kubernetes.io/name":       name,
				"app.kubernetes.io/part-of":    name,
				"openshift.io/user-monitoring": "false",
			},
		},

		Spec: monv1.ServiceMonitorSpec{
			Endpoints: []monv1.Endpoint{
				{
					Port:   "metrics",
					Scheme: ptr.To(monv1.Scheme("https")),
					HTTPConfigWithProxyAndTLSFiles: monv1.HTTPConfigWithProxyAndTLSFiles{
						HTTPConfigWithTLSFiles: monv1.HTTPConfigWithTLSFiles{
							TLSConfig: &monv1.TLSConfig{
								TLSFilesConfig: monv1.TLSFilesConfig{
									CAFile:   "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
									CertFile: "/etc/prometheus/secrets/metrics-client-certs/tls.crt",
									KeyFile:  "/etc/prometheus/secrets/metrics-client-certs/tls.key",
								},
								SafeTLSConfig: monv1.SafeTLSConfig{
									ServerName: ptr.To(fmt.Sprintf("%s.%s.svc", name, namespace)),
								},
							},
						},
					},
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component": "operator",
					"app.kubernetes.io/name":      name,
				},
			},
		},
	}
}

func newPrometheusRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-k8s",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"services", "endpoints", "pods"},
			Verbs:     []string{"get", "list", "watch"},
		}},
	}
}

func newRoleBindingForPrometheusRole(namespace string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-k8s",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  corev1.SchemeGroupVersion.Group,
			Kind:      "ServiceAccount",
			Name:      "prometheus-k8s",
			Namespace: reconciler.OpenshiftMonitoringNamespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     "prometheus-k8s",
		},
	}
	return roleBinding
}
