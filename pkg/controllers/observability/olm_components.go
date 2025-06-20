package observability

import (
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func subscription(config OperatorInstallConfig) *olmv1alpha1.Subscription {
	return &olmv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       olmv1alpha1.SubscriptionKind,
			APIVersion: olmv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.PackageName,
			Namespace: config.Namespace,
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			CatalogSource:          "redhat-operators",
			CatalogSourceNamespace: "openshift-marketplace",
			Package:                config.PackageName,
			Channel:                config.Channel,
			StartingCSV:            config.StartingCSV,
			InstallPlanApproval:    olmv1alpha1.ApprovalAutomatic,
		},
	}
}
