package observability

import (
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOperatorsStatus(t *testing.T) {
	external := olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo-operator", Labels: map[string]string{util.ResourceLabel: "another-operator"}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}}
	managed := olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "loki-operator", Labels: map[string]string{util.ResourceLabel: util.OpName}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "loki"}}
	status := operatorsStatus{subs: []olmv1alpha1.Subscription{external, managed}}

	require.False(t, status.ShouldInstall([]string{"tempo"}), "an externally managed subscription must be reused")
	require.False(t, status.ShouldInstall([]string{"tempo-product", "tempo"}), "an equivalent package must be reused")
	require.True(t, status.ShouldInstall([]string{"loki"}), "a COO-managed subscription remains COO's responsibility")
	require.True(t, status.ShouldInstall([]string{"cluster-logging"}), "an absent operator must be installed")
	require.Nil(t, status.cooManages("tempo"), "external subscriptions must never be removed by COO")
	require.Equal(t, &managed, status.cooManages("loki"))
	require.Nil(t, status.cooManages("cluster-logging"))

	misleading := operatorsStatus{subs: []olmv1alpha1.Subscription{{
		ObjectMeta: metav1.ObjectMeta{Name: "tempo-product-backup"},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "different-package"},
	}}}
	require.True(t, misleading.ShouldInstall([]string{"tempo-product"}), "subscription names must not be prefix-matched")
}

func TestManagedOperatorDeleters(t *testing.T) {
	tests := []struct {
		name string
		sub  *olmv1alpha1.Subscription
		want int
	}{
		{name: "operator absent", want: 0},
		{name: "external operator is preserved", sub: &olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo", Labels: map[string]string{util.ResourceLabel: "external"}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}}, want: 0},
		{name: "managed subscription without CSV", sub: &olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo", Labels: map[string]string{util.ResourceLabel: util.OpName}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}}, want: 1},
		{name: "managed subscription and CSV", sub: &olmv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{Name: "tempo", Labels: map[string]string{util.ResourceLabel: util.OpName}}, Spec: &olmv1alpha1.SubscriptionSpec{Package: "tempo"}, Status: olmv1alpha1.SubscriptionStatus{CurrentCSV: "tempo.v1"}}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var subscriptions []olmv1alpha1.Subscription
			if tt.sub != nil {
				subscriptions = append(subscriptions, *tt.sub)
			}
			require.Len(t, managedOperatorDeleters("tempo", operatorsStatus{subs: subscriptions}), tt.want)
		})
	}
}
