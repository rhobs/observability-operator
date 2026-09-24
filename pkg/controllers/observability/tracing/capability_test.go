package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestTracingCapabilityPlan(t *testing.T) {
	trueValue := true
	tests := []struct {
		name          string
		tracing       *obsv1alpha1.TracingSpec
		wantOperators bool
		wantOperands  bool
		deleting      bool
	}{
		{name: "capability absent"},
		{name: "capability disabled", tracing: &obsv1alpha1.TracingSpec{}},
		{
			name: "disabled with missing storage credentials",
			tracing: &obsv1alpha1.TracingSpec{Storage: &obsv1alpha1.TracingStorageSpec{ObjectStorageSpec: &obsv1alpha1.ObjectStorageSpec{
				S3: &obsv1alpha1.S3Spec{AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "missing", Key: "key"}},
			}}},
		},
		{
			name:     "deleting with missing storage credentials",
			deleting: true,
			tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}, Storage: &obsv1alpha1.TracingStorageSpec{ObjectStorageSpec: &obsv1alpha1.ObjectStorageSpec{
				S3: &obsv1alpha1.S3Spec{AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "missing", Key: "key"}},
			}}},
		},
		{
			name: "operators only",
			tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
				Operators: &obsv1alpha1.OperatorsSpec{Install: &trueValue},
			}},
			wantOperators: true,
		},
		{
			name: "capability enabled",
			tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
				Enabled: true,
			}},
			wantOperators: true,
			wantOperands:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &obsv1alpha1.ObservabilityInstaller{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "observability"},
				Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
					Tracing: tt.tracing,
				}},
			}
			if tt.deleting {
				now := metav1.Now()
				instance.DeletionTimestamp = &now
			}
			plan, err := planTracing(context.Background(), instance, fake.NewClientBuilder().Build())
			require.NoError(t, err)
			require.Equal(t, "tracing", New(Options{}).Name)
			operators := shared(instance, Options{}).Operators
			require.Len(t, operators, 2)
			for _, operator := range operators {
				require.Equal(t, tt.wantOperators, operator.Desired, operator.Name)
			}
			require.NotEmpty(t, plan.Owned, "owned inventory is required for cleanup while disabled")
			if tt.wantOperands {
				require.NotEmpty(t, plan.Desired)
			} else {
				require.Empty(t, plan.Desired)
			}
		})
	}
}

func TestTracingCapabilityPlanCleansUpLegacyRBAC(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "observability"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
			Tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}},
		}},
	}
	key := func(obj client.Object) string {
		return obj.GetObjectKind().GroupVersionKind().Kind + "/" + obj.GetName()
	}
	legacyNames := []string{
		"ClusterRole/coo-otelcol-test-components",
		"ClusterRoleBinding/coo-otelcol-test-components",
		"ClusterRole/coo-otelcol-test-tempo",
		"ClusterRoleBinding/coo-otelcol-test-tempo",
	}

	for _, tt := range []struct {
		name     string
		partOf   string
		wantOwns bool
	}{
		{name: "legacy objects of this installer are cleaned up", partOf: "test", wantOwns: true},
		// A legacy name can be the current name of another installer, here
		// installer "otelcol-test" in namespace "coo".
		{name: "objects of another installer are left alone", partOf: "otelcol-test", wantOwns: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			for _, object := range legacyOTelCollectorRBACNames("test") {
				object.SetLabels(map[string]string{"app.kubernetes.io/part-of": tt.partOf})
				builder = builder.WithObjects(object)
			}
			plan, err := planTracing(context.Background(), instance, builder.Build())
			require.NoError(t, err)

			owned := map[string]bool{}
			for _, obj := range plan.Owned {
				owned[key(obj)] = true
			}
			for _, name := range legacyNames {
				require.Equal(t, tt.wantOwns, owned[name], "%s in owned inventory", name)
			}
			for _, obj := range plan.Desired {
				require.NotContains(t, legacyNames, key(obj), "%s must not be desired", key(obj))
			}
		})
	}
}
