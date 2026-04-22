package uiplugin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsVersionAheadOrEqual(t *testing.T) {
	testCases := []struct {
		clusterVersion     string
		nextClusterVersion string
		expected           bool
	}{
		{
			clusterVersion:     "v4.18",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "v4.17",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "v4.16",
			nextClusterVersion: "v4.17",
			expected:           false,
		},
		{
			clusterVersion:     "4.18",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "4.17.0-0.nightly-2024-07-09-121045",
			nextClusterVersion: "v4.17",
			expected:           true,
		},
		{
			clusterVersion:     "4.16.0-0.nightly-2024-07-09-121045",
			nextClusterVersion: "v4.17",
			expected:           false,
		},
		{
			clusterVersion:     "v4.18",
			nextClusterVersion: "",
			expected:           false,
		},
	}

	for _, tc := range testCases {
		actual := isVersionAheadOrEqual(tc.clusterVersion, tc.nextClusterVersion)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestNewDeploymentTLSArgs(t *testing.T) {
	testCases := []struct {
		name          string
		tlsMinVersion string
		tlsCiphers    []string
		extraArgs     []string
		expectArgs    []string
		notExpectArgs []string
	}{
		{
			name:          "TLS profile with min version and ciphers",
			tlsMinVersion: "VersionTLS12",
			tlsCiphers:    []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"},
			expectArgs: []string{
				"-tls-min-version=VersionTLS12",
				"-tls-cipher-suites=TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384",
			},
		},
		{
			name:          "no TLS profile",
			tlsMinVersion: "",
			tlsCiphers:    nil,
			notExpectArgs: []string{
				"-tls-min-version=",
				"-tls-cipher-suites=",
			},
		},
		{
			name:          "TLS min version only",
			tlsMinVersion: "VersionTLS13",
			tlsCiphers:    nil,
			expectArgs: []string{
				"-tls-min-version=VersionTLS13",
			},
			notExpectArgs: []string{
				"-tls-cipher-suites=",
			},
		},
		{
			name:          "TLS args appear after extra args",
			tlsMinVersion: "VersionTLS12",
			tlsCiphers:    []string{"TLS_AES_128_GCM_SHA256"},
			extraArgs:     []string{"-plugin-config-path=/etc/plugin/config/config.yaml"},
			expectArgs: []string{
				"-plugin-config-path=/etc/plugin/config/config.yaml",
				"-tls-min-version=VersionTLS12",
				"-tls-cipher-suites=TLS_AES_128_GCM_SHA256",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := UIPluginInfo{
				Name:          "test-plugin",
				Image:         "test-image:latest",
				ExtraArgs:     tc.extraArgs,
				TLSMinVersion: tc.tlsMinVersion,
				TLSCiphers:    tc.tlsCiphers,
			}

			deploy := newDeployment(info, "test-ns", nil)
			args := deploy.Spec.Template.Spec.Containers[0].Args

			// Always expect base args
			assert.Assert(t, containsArg(args, "-port=9443"))
			assert.Assert(t, containsArg(args, "-cert=/var/serving-cert/tls.crt"))
			assert.Assert(t, containsArg(args, "-key=/var/serving-cert/tls.key"))

			for _, expected := range tc.expectArgs {
				assert.Assert(t, containsArg(args, expected), "expected arg %q not found in %v", expected, args)
			}

			for _, notExpected := range tc.notExpectArgs {
				assert.Assert(t, !containsArgPrefix(args, notExpected), "unexpected arg prefix %q found in %v", notExpected, args)
			}

			// Verify ordering: TLS args should come after extra args
			if len(tc.extraArgs) > 0 && tc.tlsMinVersion != "" {
				extraArgIdx := indexOfArg(args, tc.extraArgs[0])
				tlsArgIdx := indexOfArg(args, "-tls-min-version="+tc.tlsMinVersion)
				assert.Assert(t, extraArgIdx < tlsArgIdx, "extra args should appear before TLS args")
			}
		})
	}
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func containsArgPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if len(arg) >= len(prefix) && arg[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func indexOfArg(args []string, target string) int {
	for i, arg := range args {
		if arg == target {
			return i
		}
	}
	return -1
}

func TestPluginComponentReconcilersTLSProfile(t *testing.T) {
	testCases := []struct {
		name          string
		tlsMinVersion string
		tlsCiphers    []string
		expectTLS     bool
	}{
		{
			name:          "TLS profile applied when SupportsTLSProfile populates fields",
			tlsMinVersion: "VersionTLS12",
			tlsCiphers:    []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"},
			expectTLS:     true,
		},
		{
			name:          "no TLS args when plugin does not support TLS profile",
			tlsMinVersion: "",
			tlsCiphers:    nil,
			expectTLS:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := UIPluginInfo{
				Name:              "test-plugin",
				ConsoleName:       "test-console-plugin",
				Image:             "test-image:latest",
				ResourceNamespace: "test-ns",
				ExtraArgs:         []string{"-config-path=/opt/app-root/config"},
				TLSMinVersion:     tc.tlsMinVersion,
				TLSCiphers:        tc.tlsCiphers,
			}

			deploy := newDeployment(info, "test-ns", nil)
			args := deploy.Spec.Template.Spec.Containers[0].Args

			if tc.expectTLS {
				assert.Assert(t, containsArg(args, "-tls-min-version="+tc.tlsMinVersion),
					"expected -tls-min-version arg in %v", args)
				assert.Assert(t, containsArgPrefix(args, "-tls-cipher-suites="),
					"expected -tls-cipher-suites arg in %v", args)

				// TLS args must come after plugin-specific extra args
				extraIdx := indexOfArg(args, "-config-path=/opt/app-root/config")
				tlsIdx := indexOfArg(args, "-tls-min-version="+tc.tlsMinVersion)
				assert.Assert(t, extraIdx < tlsIdx,
					"TLS args (idx %d) should appear after extra args (idx %d)", tlsIdx, extraIdx)
			} else {
				assert.Assert(t, !containsArgPrefix(args, "-tls-min-version="),
					"unexpected -tls-min-version arg in %v", args)
				assert.Assert(t, !containsArgPrefix(args, "-tls-cipher-suites="),
					"unexpected -tls-cipher-suites arg in %v", args)
			}

			// Base args must always be present regardless of TLS
			assert.Assert(t, containsArg(args, "-port=9443"))
			assert.Assert(t, containsArg(args, "-cert=/var/serving-cert/tls.crt"))
			assert.Assert(t, containsArg(args, "-key=/var/serving-cert/tls.key"))
			assert.Assert(t, containsArg(args, "-config-path=/opt/app-root/config"))
		})
	}
}
