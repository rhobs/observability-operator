package logging

import (
	"context"
	"fmt"

	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	clfv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	storage "github.com/rhobs/observability-operator/pkg/controllers/observability/storage"
)

func lokiStackName(name string) string {
	return fmt.Sprintf("%s-loki", name)
}

func lokiSecretName(name string) string {
	return fmt.Sprintf("coo-%s-loki", name)
}

func lokiStorageCAConfigMapName(name string) string {
	return fmt.Sprintf("coo-%s-loki-storage-ca", name)
}

func lokiStack(instance *obsv1alpha1.ObservabilityInstaller) *lokiv1.LokiStack {
	storageSpec := instance.Spec.GetCapabilities().GetLogging().GetLokiStack().GetStorage()
	lokiSpec := instance.Spec.GetCapabilities().GetLogging().GetLokiStack()

	storageType := toLokiStorageType(storageSpec)
	credentialMode := toLokiCredentialMode(storageSpec)

	var size lokiv1.LokiStackSizeType
	var storageClassName string
	var schemas []lokiv1.ObjectStorageSchema
	if lokiSpec != nil {
		size = lokiSpec.Size
		storageClassName = lokiSpec.StorageClassName
		schemas = lokiSpec.Schemas
	}

	ls := &lokiv1.LokiStack{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LokiStack",
			APIVersion: lokiv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      lokiStackName(instance.Name),
			Namespace: instance.Namespace,
		},
		Spec: lokiv1.LokiStackSpec{
			Size:             size,
			StorageClassName: storageClassName,
			Storage: lokiv1.ObjectStorageSpec{
				Schemas: schemas,
				Secret: lokiv1.ObjectStorageSecretSpec{
					Type:           storageType,
					Name:           lokiSecretName(instance.Name),
					CredentialMode: credentialMode,
				},
			},
			Tenants: &lokiv1.TenantsSpec{
				Mode: lokiv1.OpenshiftLogging,
			},
		},
	}

	// Loki's storage TLS spec requires a CA configmap name, so it can only be set
	// with a user-provided CA. Without one Loki falls back to the system CAs.
	if storageSpec != nil && storageSpec.TLS != nil && storageSpec.TLS.CAConfigMap != nil {
		ls.Spec.Storage.TLS = &lokiv1.ObjectStorageTLSSpec{
			CASpec: lokiv1.CASpec{CA: lokiStorageCAConfigMapName(instance.Name)},
		}
	}

	return ls
}

func lokiStackSecrets(ctx context.Context, k8sReader client.Reader, instance obsv1alpha1.ObservabilityInstaller) (*storage.Resources, error) {
	return storage.Materialize(ctx, k8sReader, instance.Namespace,
		instance.Spec.GetCapabilities().GetLogging().GetLokiStack().GetStorage(),
		storage.Names{
			Secret:      lokiSecretName(instance.Name),
			CAConfigMap: lokiStorageCAConfigMapName(instance.Name),
		}, storage.LokiFormat)
}

func loggingCollectorServiceAccountName(instanceName string) string {
	return fmt.Sprintf("%s-logging-collector", instanceName)
}

func loggingServiceAccount(instance *obsv1alpha1.ObservabilityInstaller) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggingCollectorServiceAccountName(instance.Name),
			Namespace: instance.Namespace,
		},
	}
}

func loggingCollectorRBAC(instance *obsv1alpha1.ObservabilityInstaller) (*rbacv1.ClusterRole, []*rbacv1.ClusterRoleBinding) {
	name := fmt.Sprintf("coo-%s-%s-logging", instance.Namespace, instance.Name)

	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"loki.grafana.com"},
				Resources: []string{"application", "audit", "infrastructure"},
				Verbs:     []string{"create"},
			},
		},
	}

	subjects := []rbacv1.Subject{{
		Kind:      "ServiceAccount",
		Name:      loggingCollectorServiceAccountName(instance.Name),
		Namespace: instance.Namespace,
	}}
	binding := func(name, roleName string) *rbacv1.ClusterRoleBinding {
		return &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRoleBinding",
				APIVersion: rbacv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{Name: name},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     roleName,
			},
			Subjects: subjects,
		}
	}

	return role, []*rbacv1.ClusterRoleBinding{
		binding(name, name),
		binding(name+"-collect-application", "collect-application-logs"),
		binding(name+"-collect-infrastructure", "collect-infrastructure-logs"),
	}
}

func clusterLogForwarderName(instanceName string) string {
	return fmt.Sprintf("%s-logging", instanceName)
}

func clusterLogForwarder(instance *obsv1alpha1.ObservabilityInstaller) *clfv1.ClusterLogForwarder {
	lokiStackNs := instance.Namespace
	lokiStackName := lokiStackName(instance.Name)

	return &clfv1.ClusterLogForwarder{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterLogForwarder",
			APIVersion: clfv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterLogForwarderName(instance.Name),
			Namespace: instance.Namespace,
		},
		Spec: clfv1.ClusterLogForwarderSpec{
			ServiceAccount: clfv1.ServiceAccount{
				Name: loggingCollectorServiceAccountName(instance.Name),
			},
			Outputs: []clfv1.OutputSpec{
				{
					Name: "default-lokistack",
					Type: clfv1.OutputTypeLokiStack,
					LokiStack: &clfv1.LokiStack{
						Target: clfv1.LokiStackTarget{
							Name:      lokiStackName,
							Namespace: lokiStackNs,
						},
						Authentication: &clfv1.LokiStackAuthentication{
							Token: &clfv1.BearerToken{
								From: clfv1.BearerTokenFromServiceAccount,
							},
						},
					},
					TLS: &clfv1.OutputTLSSpec{
						TLSSpec: clfv1.TLSSpec{
							CA: &clfv1.ValueReference{
								Key:           "service-ca.crt",
								ConfigMapName: "openshift-service-ca.crt",
							},
						},
					},
				},
			},
			Pipelines: []clfv1.PipelineSpec{
				{
					Name: "default-logstore",
					InputRefs: []string{
						"application",
						"infrastructure",
					},
					OutputRefs: []string{
						"default-lokistack",
					},
				},
			},
		},
	}
}

func toLokiStorageType(storage *obsv1alpha1.ObjectStorageSpec) lokiv1.ObjectStorageSecretType {
	if storage == nil {
		return ""
	}
	if storage.S3 != nil || storage.S3STS != nil || storage.S3CCO != nil {
		return lokiv1.ObjectStorageSecretS3
	} else if storage.Azure != nil || storage.AzureWIF != nil {
		return lokiv1.ObjectStorageSecretAzure
	} else if storage.GCS != nil || storage.GCSWIF != nil {
		return lokiv1.ObjectStorageSecretGCS
	}
	return ""
}

func toLokiCredentialMode(storage *obsv1alpha1.ObjectStorageSpec) lokiv1.CredentialMode {
	if storage == nil {
		return ""
	}
	if storage.S3 != nil || storage.Azure != nil || storage.GCS != nil {
		return lokiv1.CredentialModeStatic
	} else if storage.S3STS != nil || storage.AzureWIF != nil || storage.GCSWIF != nil {
		return lokiv1.CredentialModeToken
	} else if storage.S3CCO != nil {
		return lokiv1.CredentialModeTokenCCO
	}
	return ""
}
