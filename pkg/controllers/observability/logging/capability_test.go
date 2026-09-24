package logging

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
)

func TestLoggingCapabilityRejectsUnsupportedStorageTLS(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "observability"},
		Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
			Logging: &obsv1alpha1.LoggingSpec{
				CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
				LokiStack: &obsv1alpha1.LokiStackSpec{ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
					TLS: &obsv1alpha1.TLSSpec{CertSecret: &obsv1alpha1.SecretKeySelector{Name: "cert", Key: "tls.crt"}},
				}},
			},
		}},
	}
	_, err := planLogging(context.Background(), instance, fake.NewClientBuilder().Build())
	require.ErrorContains(t, err, "caConfigMap only")
}

func TestLoggingCapabilityPlan(t *testing.T) {
	trueValue := true
	tests := []struct {
		name          string
		logging       *obsv1alpha1.LoggingSpec
		wantOperators bool
		wantOperands  bool
		deleting      bool
	}{
		{name: "capability absent"},
		{name: "capability disabled", logging: &obsv1alpha1.LoggingSpec{}},
		{
			name: "disabled with missing storage credentials",
			logging: &obsv1alpha1.LoggingSpec{LokiStack: &obsv1alpha1.LokiStackSpec{ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
				S3: &obsv1alpha1.S3Spec{AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "missing", Key: "key"}},
			}}},
		},
		{
			name:     "deleting with missing storage credentials",
			deleting: true,
			logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}, LokiStack: &obsv1alpha1.LokiStackSpec{ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
				S3: &obsv1alpha1.S3Spec{AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "missing", Key: "key"}},
			}}},
		},
		{
			name: "operators only",
			logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
				Operators: &obsv1alpha1.OperatorsSpec{Install: &trueValue},
			}},
			wantOperators: true,
		},
		{
			name: "capability enabled",
			logging: &obsv1alpha1.LoggingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{
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
					Logging: tt.logging,
				}},
			}
			if tt.deleting {
				now := metav1.Now()
				instance.DeletionTimestamp = &now
			}
			plan, err := planLogging(context.Background(), instance, fake.NewClientBuilder().Build())
			require.NoError(t, err)
			require.Equal(t, "logging", New(Options{}).Name)
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
