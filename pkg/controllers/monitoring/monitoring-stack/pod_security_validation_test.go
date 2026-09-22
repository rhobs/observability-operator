package monitoringstack

import (
	"context"
	"errors"
	"testing"

	securityv1 "github.com/openshift/api/security/v1"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

type noSCCReader struct{}

func (noSCCReader) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return &meta.NoKindMatchError{GroupKind: schema.GroupKind{Group: securityv1.GroupName, Kind: "SecurityContextConstraints"}}
}

func (noSCCReader) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return nil
}

func TestValidateUserNamespaceIDRange(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "start and length", value: "1000/10000"},
		{name: "start and end", value: "1000-10999"},
		{name: "multiple ranges", value: "1000/10000,20000-29999"},
		{name: "maximum ID", value: "65535/1"},
		{name: "empty", wantErr: true},
		{name: "invalid format", value: "1000", wantErr: true},
		{name: "zero length", value: "1000/0", wantErr: true},
		{name: "root ID", value: "0/1000", wantErr: true},
		{name: "reversed range", value: "2000-1000", wantErr: true},
		{name: "range exceeds maximum", value: "65000/1000", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateUserNamespaceIDRange(test.value)
			if test.wantErr {
				assert.Assert(t, err != nil)
				return
			}
			assert.NilError(t, err)
		})
	}
}

func TestValidatePodSecurityProfile(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NilError(t, corev1.AddToScheme(scheme))
	assert.NilError(t, securityv1.Install(scheme))

	restrictedV2 := &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "restricted-v2"}}
	restrictedV3 := &securityv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "restricted-v3"}}
	validNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "valid",
			Annotations: map[string]string{
				"openshift.io/sa.scc.uid-range":           "1000/10000",
				"openshift.io/sa.scc.supplemental-groups": "1000/10000",
			},
		},
	}
	invalidNamespace := validNamespace.DeepCopy()
	invalidNamespace.Name = "invalid"
	invalidNamespace.Annotations["openshift.io/sa.scc.uid-range"] = "1000000000/10000"

	reader := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(restrictedV2, restrictedV3, validNamespace, invalidNamespace).
		Build()

	tests := []struct {
		name       string
		profile    stack.PodSecurityProfile
		namespace  string
		wantReason string
	}{
		{
			name:      "restricted-v2 SCC is available",
			profile:   stack.RestrictedV2PodSecurityProfile,
			namespace: "valid",
		},
		{
			name:      "restricted-v3 accepts user namespace ranges",
			profile:   stack.RestrictedV3PodSecurityProfile,
			namespace: "valid",
		},
		{
			name:       "restricted-v3 rejects host ID ranges",
			profile:    stack.RestrictedV3PodSecurityProfile,
			namespace:  "invalid",
			wantReason: invalidNamespaceIDRangeReason,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms := &stack.MonitoringStack{
				ObjectMeta: metav1.ObjectMeta{Namespace: test.namespace},
				Spec: stack.MonitoringStackSpec{
					PodSecurity: stack.PodSecurityConfig{Profile: test.profile},
				},
			}
			err := validatePodSecurityProfile(t.Context(), reader, ms)
			if test.wantReason == "" {
				assert.NilError(t, err)
				return
			}

			var profileError *podSecurityValidationError
			assert.Assert(t, errors.As(err, &profileError))
			assert.Equal(t, profileError.reason, test.wantReason)
		})
	}
}

func TestValidatePodSecurityProfileWithoutOpenShift(t *testing.T) {
	legacyStack := &stack.MonitoringStack{}
	assert.NilError(t, validatePodSecurityProfile(t.Context(), noSCCReader{}, legacyStack))

	restrictedStack := &stack.MonitoringStack{
		Spec: stack.MonitoringStackSpec{
			PodSecurity: stack.PodSecurityConfig{Profile: stack.RestrictedV2PodSecurityProfile},
		},
	}
	err := validatePodSecurityProfile(t.Context(), noSCCReader{}, restrictedStack)
	var profileError *podSecurityValidationError
	assert.Assert(t, errors.As(err, &profileError))
	assert.Equal(t, profileError.reason, unsupportedPlatformReason)
}

func TestValidatePodSecurityProfileWithoutRequiredSCC(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NilError(t, securityv1.Install(scheme))
	reader := fake.NewClientBuilder().WithScheme(scheme).Build()
	ms := &stack.MonitoringStack{
		Spec: stack.MonitoringStackSpec{
			PodSecurity: stack.PodSecurityConfig{Profile: stack.RestrictedV2PodSecurityProfile},
		},
	}

	err := validatePodSecurityProfile(t.Context(), reader, ms)
	var profileError *podSecurityValidationError
	assert.Assert(t, errors.As(err, &profileError))
	assert.Equal(t, profileError.reason, sccUnavailableReason)
}
