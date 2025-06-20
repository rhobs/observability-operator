package observability

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	go_yaml "github.com/goccy/go-yaml"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

const (
	otelCollectorName = "coo"
)

var (
	//go:embed collector.yaml
	collectorConfigFS       embed.FS
	collectorConfigTemplate = template.Must(template.ParseFS(collectorConfigFS, "collector.yaml"))
)

type templateOptions struct {
	Namespace   string
	TempoTenant string
	TempoName   string
}

func otelCollector(ns string) (*otelv1beta1.OpenTelemetryCollector, error) {
	w := bytes.NewBuffer(nil)
	err := collectorConfigTemplate.Execute(w, templateOptions{Namespace: ns, TempoName: tempoName, TempoTenant: tenantName})
	if err != nil {
		return nil, err
	}
	cfgStr, err := io.ReadAll(w)
	if err != nil {
		return nil, err
	}

	// Convert YAML to JSON and unmarshal into OpenTelemetryCollector Config
	collectorJson, err := go_yaml.YAMLToJSON(cfgStr)
	if err != nil {
		return nil, err
	}
	cfg := &otelv1beta1.Config{}
	err = json.Unmarshal(collectorJson, cfg)
	if err != nil {
		return nil, err
	}

	return &otelv1beta1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenTelemetryCollector",
			APIVersion: otelv1beta1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      otelCollectorName,
			Namespace: ns,
		},
		Spec: otelv1beta1.OpenTelemetryCollectorSpec{
			Config: *cfg,
			// Fixes updater failed to patch: OpenTelemetryCollector.opentelemetry.io \"otel-tracing\" is invalid: [spec.upgradeStrategy: Unsupported value: \"\": supported values: \"automatic\", \"none\", <nil>: Invalid value: \"null\":
			UpgradeStrategy: otelv1beta1.UpgradeStrategyAutomatic,
			Mode:            otelv1beta1.ModeDeployment,
		},
	}, nil
}

func otelCollectorComponentsRBAC(ns string) (*rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding) {
	name := "coo-otel-collector-components"
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			// required by the k8sattributes processor
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "namespaces", "nodes"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
	binding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("%s-collector", otelCollectorName),
				Namespace: ns,
			},
		},
	}
	return role, binding
}

func otelCollectorTempoRBAC(ns string) (*rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding) {
	name := "coo-otel-collector-tempo"
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"tempo.grafana.com"},
				Resources:     []string{tenantName},
				ResourceNames: []string{"traces"},
				Verbs:         []string{"create"},
			},
		},
	}

	binding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("%s-collector", otelCollectorName),
				Namespace: ns,
			},
		},
	}

	return role, binding
}
