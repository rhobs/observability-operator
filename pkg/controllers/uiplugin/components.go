package uiplugin

import (
	"bytes"
	"embed"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"

	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
)

const (
	port                 = 9443
	DefaultLokiStackName = "logging-loki"
	OpenshiftLoggingNs   = "openshift-logging"
	OpenshiftNetobservNs = "netobserv"
	OpenshiftTracingNs   = "openshift-tracing"

	annotationPrefix = "observability.openshift.io/ui-plugin-"

	serviceAccountSuffix  = "-sa"
	clusterRoleSuffix     = "-cr"
	clusterRoleBindSuffix = "-crb"

	defaultTempoService = "tempo-platform-gateway"
)

var (
	defaultNodeSelector = map[string]string{
		"kubernetes.io/os": "linux",
	}

	hashSeparator = []byte("\n")

	//go:embed config/korrel8r.yaml
	korrel8rConfigYAMLTmplFile embed.FS

	korrel8rConfigTmpl = template.Must(template.ParseFS(korrel8rConfigYAMLTmplFile, "config/korrel8r.yaml"))
)

func IsVersionAheadOrEqual(currentVersion, version string) bool {
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}
	if version == "" {
		return false
	}

	canonicalMinVersion := fmt.Sprintf("%s-0", semver.Canonical(version))

	return semver.Compare(currentVersion, canonicalMinVersion) >= 0
}

func computeConfigMapHash(cm *corev1.ConfigMap) string {
	keys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := fnv.New32a()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write(hashSeparator)
		h.Write([]byte(cm.Data[k]))
		h.Write(hashSeparator)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func createNodeSelectorAndTolerations(config *uiv1alpha1.DeploymentConfig) (map[string]string, []corev1.Toleration) {
	if config == nil {
		return defaultNodeSelector, nil
	}

	nodeSelector := config.NodeSelector
	if nodeSelector == nil {
		nodeSelector = defaultNodeSelector
	}

	return nodeSelector, config.Tolerations
}

func marshalTimeoutConfig(timeout string) (string, error) {
	if timeout == "" {
		return "", nil
	}
	pluginCfg := struct {
		Timeout string `yaml:"timeout"`
	}{
		Timeout: timeout,
	}
	buf := &bytes.Buffer{}
	if err := yaml.NewEncoder(buf).Encode(pluginCfg); err != nil {
		return "", err
	}
	return buf.String(), nil
}
