package capability

import (
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Subscription(config OperatorConfig) *olmv1alpha1.Subscription {
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

// OperatorConfig describes an OLM subscription requested by a capability.
type OperatorConfig struct {
	Namespace   string
	PackageName string
	StartingCSV string
	Channel     string
	// EquivalentPackages are other packages providing the same operator, for
	// example the community build. A subscription to any of them means the
	// operator is already installed and COO must not install its own.
	EquivalentPackages []string
}
