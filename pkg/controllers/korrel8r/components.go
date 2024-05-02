package korrel8r

import (
	//"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	//"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rhobs/observability-operator/pkg/reconciler"
)

const (
	port                  = 8443
	serviceAccountSuffix  = "-sa"
	servingCertVolumeName = "serving-cert"
	korrel8rObjName                  = "korrel8r"
)

func korrel8rComponentReconcilers(korrel8rDeploy *appsv1.Deployment, korrel8rSvc *corev1.Service, korrel8rCfg Korrel8rConfiguration, namespace string) []reconciler.Reconciler {

	return []reconciler.Reconciler{
		reconciler.NewUpdater(newKorrel8rService(korrel8rObjName, namespace), korrel8rSvc),
		reconciler.NewUpdater(newKorrel8rDeployment(korrel8rObjName, namespace, korrel8rCfg), korrel8rDeploy),
	}
}

func newKorrel8rDeployment(name string, namespace string, korrel8rCfg Korrel8rConfiguration) *appsv1.Deployment {
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    componentLabels(name),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: componentLabels(name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels:    componentLabels(name),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: name + serviceAccountSuffix,
					Containers: []corev1.Container{
						{
							Name:    name,
							Image:   korrel8rCfg.Image,
							Command: []string{"korrel8r", "web", "--config", "/etc/korrel8r/openshift-svc.yaml"},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             ptr.To(true),
								AllowPrivilegeEscalation: ptr.To(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
				},
			},
		},
	}
	return deploy
}

func newKorrel8rService(name string, namespace string) *corev1.Service {
	annotations := map[string]string{
		"service.alpha.openshift.io/serving-cert-secret-name": name,
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      componentLabels(name),
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					Name:       "web",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(port),
				},
			},
			Selector: componentLabels(name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func componentLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/part-of":    "Korrel8r",
		"app.kubernetes.io/managed-by": "observability-operator",
	}
}
