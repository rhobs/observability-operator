package uiplugin

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"gotest.tools/v3/assert"
)

func TestApplyTLSProfile(t *testing.T) {
	conf := &UIPluginBuildConfig{}
	conf.ApplyTLSProfile(configv1.TLSProfileSpec{
		Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
		MinTLSVersion: configv1.VersionTLS12,
	})
	assert.Equal(t, "VersionTLS12", conf.TLSMinVersion)
	assert.Assert(t, len(conf.TLSCiphers) > 0)
}
