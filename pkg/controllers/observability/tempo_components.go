package observability

import (
	"fmt"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

const (
	tempoName  = "coo"
	tenantName = "prod"
	tenantID   = "1610b0c3-c509-4592-a256-a1871353dbfb"
)

func tempoStack(storage obsv1alpha1.StorageSpec, ns string) *tempov1alpha1.TempoStack {
	return &tempov1alpha1.TempoStack{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TempoStack",
			APIVersion: tempov1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoName,
			Namespace: ns,
		},
		Spec: tempov1alpha1.TempoStackSpec{
			Storage: tempov1alpha1.ObjectStorageSpec{
				Secret: tempov1alpha1.ObjectStorageSecretSpec{
					Name: tempoSecretName(storage.Secret.Name),
					Type: tempov1alpha1.ObjectStorageSecretType(storage.Secret.Type),
				},
			},
			Template: tempov1alpha1.TempoTemplateSpec{
				Gateway: tempov1alpha1.TempoGatewaySpec{
					Enabled: true,
				},
			},
			Tenants: &tempov1alpha1.TenantsSpec{
				Mode: tempov1alpha1.ModeOpenShift,
				Authentication: []tempov1alpha1.AuthenticationSpec{
					{
						TenantName: tenantName,
						TenantID:   tenantID,
					},
				},
			},
		},
	}
}

func tempoSecretName(name string) string {
	return fmt.Sprintf("coo-%s", name)
}

func tempoStackSecret(storage obsv1alpha1.StorageSpec, ns string, storageSecret corev1.Secret) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempoSecretName(storage.Secret.Name),
			Namespace: ns,
		},
		Data: map[string][]byte{
			"bucket":            storageSecret.Data["bucket"],
			"endpoint":          storageSecret.Data["endpoint"],
			"access_key_id":     storageSecret.Data["access_key_id"],
			"access_key_secret": storageSecret.Data["access_key_secret"],
		},
	}
}

func uiPlugin() *uiv1alpha1.UIPlugin {
	return &uiv1alpha1.UIPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UIPlugin",
			APIVersion: uiv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "distributed-tracing",
		},
		Spec: uiv1alpha1.UIPluginSpec{
			Type: uiv1alpha1.TypeDistributedTracing,
		},
	}
}
