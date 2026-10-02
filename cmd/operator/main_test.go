package main

import (
	"context"
	"errors"
	"flag"
	"testing"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	"gotest.tools/v3/assert"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/rhobs/observability-operator/pkg/operator"
)

func TestFlagWasSet(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "unset"},
		{name: "explicitly enabled", args: []string{"--openshift.enabled=true"}, want: true},
		{name: "explicitly disabled", args: []string{"--openshift.enabled=false"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			flags.Bool("openshift.enabled", false, "")
			assert.NilError(t, flags.Parse(tt.args))
			assert.Equal(t, flagWasSet(flags, "openshift.enabled"), tt.want)
		})
	}
}

func TestTryOpenShift(t *testing.T) {
	setupLog = logr.Discard()
	profile := configv1.TLSProfileSpec{
		MinTLSVersion: configv1.VersionTLS12,
		Ciphers:       []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
	}
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.APIServerSpec{TLSSecurityProfile: &configv1.TLSSecurityProfile{
			Type: configv1.TLSProfileCustomType,
			Custom: &configv1.CustomTLSProfile{
				TLSProfileSpec: profile,
			},
		}},
	}
	clusterVersion := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Status: configv1.ClusterVersionStatus{
			Desired: configv1.Release{Version: "4.19.3"},
		},
	}

	calls := 0
	newClient := func() (client.Client, error) {
		calls++
		return fake.NewClientBuilder().WithScheme(operator.NewOpenShiftScheme()).WithObjects(apiServer, clusterVersion).Build(), nil
	}

	data, enabled, err := tryOpenShift(context.Background(), newClient, false)

	assert.NilError(t, err)
	assert.Assert(t, enabled)
	assert.Equal(t, calls, 1)
	assert.DeepEqual(t, data.TLSProfile, profile)
	assert.Equal(t, data.Version, "4.19.3")
}

func TestTryOpenShiftAbsentAPI(t *testing.T) {
	setupLog = logr.Discard()
	// On a non-OpenShift cluster the API group is not served, so Get fails with a
	// no-match error.
	newClient := func() (client.Client, error) {
		return fake.NewClientBuilder().
			WithScheme(operator.NewOpenShiftScheme()).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return &apimeta.NoKindMatchError{GroupKind: schema.GroupKind{Group: "config.openshift.io", Kind: "APIServer"}}
				},
			}).Build(), nil
	}

	data, enabled, err := tryOpenShift(context.Background(), newClient, false)
	assert.NilError(t, err)
	assert.Assert(t, !enabled)
	assert.DeepEqual(t, data, openShiftData{})
}

func TestTryOpenShiftReturnsErrors(t *testing.T) {
	setupLog = logr.Discard()
	tests := []struct {
		name             string
		openShiftEnabled bool
		newClient        clientFunc
		wantError        string
	}{
		{
			name:             "explicitly enabled requires OpenShift",
			openShiftEnabled: true,
			newClient: func() (client.Client, error) {
				return fake.NewClientBuilder().WithScheme(operator.NewOpenShiftScheme()).Build(), nil
			},
			wantError: "failed to fetch TLS profile",
		},
		{
			// The API group is served but the singleton is missing: an OpenShift
			// cluster in a bad state, not a plain Kubernetes cluster.
			name: "auto-detection rejects a missing APIServer object",
			newClient: func() (client.Client, error) {
				return fake.NewClientBuilder().WithScheme(operator.NewOpenShiftScheme()).Build(), nil
			},
			wantError: "failed to fetch TLS profile",
		},
		{
			name: "auto-detection rejects unexpected errors",
			newClient: func() (client.Client, error) {
				return nil, errors.New("client failed")
			},
			wantError: "client failed",
		},
		{
			name: "auto-detection requires cluster version after finding OpenShift API",
			newClient: func() (client.Client, error) {
				return fake.NewClientBuilder().WithScheme(operator.NewOpenShiftScheme()).WithObjects(
					&configv1.APIServer{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}},
				).Build(), nil
			},
			wantError: "failed to fetch cluster version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, enabled, err := tryOpenShift(context.Background(), tt.newClient, tt.openShiftEnabled)
			assert.ErrorContains(t, err, tt.wantError)
			assert.Assert(t, !enabled)
			assert.DeepEqual(t, data, openShiftData{})
		})
	}
}
