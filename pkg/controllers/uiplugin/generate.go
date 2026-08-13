package uiplugin

import (
	"context"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

// UIPluginBuildConfig carries the parameters needed to resolve a UIPlugin CR
// into its operand object set, usable both by the running operator and the
// offline generator (which has no cluster).
type UIPluginBuildConfig struct {
	Images         map[string]string
	Namespace      string
	ClusterVersion string
	TLSCiphers     []string
	TLSMinVersion  string
	// Pre-resolved values for plugins that need cluster state.
	LokiServiceNames  map[string]string
	TempoServiceNames map[string]string
	// Nil in the offline generator; set by the controller.
	DynamicClient dynamic.Interface
}

// ApplyTLSProfile sets TLS fields from an OpenShift TLS security profile,
// converting OpenSSL cipher names to IANA.
func (c *UIPluginBuildConfig) ApplyTLSProfile(profile configv1.TLSProfileSpec) {
	c.TLSMinVersion = string(profile.MinTLSVersion)
	c.TLSCiphers = libgocrypto.OpenSSLToIANACipherSuites(profile.Ciphers)
}

// GenerateUIPluginObjects builds the operands for a UIPlugin CR, used by
// both the reconciler and offline generator to produce identical resources.
func GenerateUIPluginObjects(ctx context.Context, plugin *uiv1alpha1.UIPlugin, conf UIPluginBuildConfig, logger logr.Logger) ([]client.Object, *UIPluginInfo, error) {
	pluginInfo, err := buildPluginInfo(ctx, plugin, conf, logger)
	if pluginInfo == nil {
		return nil, nil, err
	}

	var objects []client.Object
	for _, rec := range pluginComponentReconcilers(plugin, *pluginInfo, conf.ClusterVersion, logger) {
		objects = append(objects, rec.Desired()...)
	}

	return objects, pluginInfo, err
}
