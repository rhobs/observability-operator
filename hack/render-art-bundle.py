#!/usr/bin/env python3
"""Render the downstream (ART) OLM bundle from the upstream bundle.

`make bundle` generates the community bundle in `bundle/`. This script
post-processes a copy of it into `bundle-art/`, the bundle consumed by ART's
`olm_bundle_konflux` builder for downstream (registry.redhat.io) builds.

It mirrors the Konflux `render_templates` customization
(github.com/rhobs/konflux-coo, bundle-patches/render_templates) but keeps
*upstream* pullspecs in place so ART can pin/replace them via
bundle-art/image-references. It does NOT do SHA pinning or registry
replacement -- ART handles that at build time.

Channel naming follows the product FBC (github.com/rhobs/konflux-coo-fbc):
package `cluster-observability-operator`, channels `stable` (default) + `fast`.

Outputs (all regenerated from scratch, so the result is deterministic):
  bundle-art/manifests/                        (copied CRDs/services/RBAC + CSV)
  bundle-art/metadata/annotations.yaml         (downstream package + channels)
  bundle-art/tests/                            (copied scorecard config)
  bundle-art/manifests/cluster-observability-operator.clusterserviceversion.yaml
  bundle-art/image-references
  bundle-art/art.yaml
  bundle-art/cluster-observability-operator.package.yaml

The operand image pullspecs below are NOT present in the upstream CSV; they must
be kept in sync with cmd/operator/main.go (defaultImages) and the vendored
prometheus-operator defaults (prometheus/alertmanager/thanos). Each `art`
(distgit) name must match the corresponding ART image config in ocp-build-data.
"""
import os
import shutil
import sys

import yaml

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SRC = os.path.join(REPO_ROOT, "bundle")
DST = os.path.join(REPO_ROOT, "bundle-art")
PKG = "cluster-observability-operator"
DEFAULT_CHANNEL = "stable"
CHANNELS = ["stable", "fast"]

# --- operand pullspecs (kept in sync with cmd/operator/main.go + prom-operator) ---
CONFIG_RELOADER = "quay.io/rhobs/obo-prometheus-config-reloader:v0.93.1-rhobs1"
ADMISSION_WEBHOOK = "quay.io/rhobs/obo-admission-webhook:v0.93.1-rhobs1"
PROMETHEUS_OPERATOR = "quay.io/rhobs/obo-prometheus-operator:v0.93.1-rhobs1"
PERSES_OPERATOR = "quay.io/openshift-observability-ui/perses-operator:v0.4.0"
ALERTMANAGER = "quay.io/prometheus/alertmanager:v0.33.1"
PROMETHEUS = "quay.io/prometheus/prometheus:v3.13.1"
THANOS = "quay.io/thanos/thanos:v0.42.2"
PERSES = "quay.io/openshift-observability-ui/perses:v0.54.0"
UI_DASHBOARDS = "quay.io/openshift-observability-ui/console-dashboards-plugin:v0.4.3"
UI_TRACING = "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v1.1.0"
UI_TRACING_PF5 = "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v0.4.3"
UI_TRACING_PF6 = "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v1.0.3"
UI_TRACING_PF4 = "quay.io/openshift-observability-ui/distributed-tracing-console-plugin:v0.3.3"
UI_LOGGING = "quay.io/openshift-observability-ui/logging-view-plugin:v6.2.1"
UI_LOGGING_PF4 = "quay.io/openshift-observability-ui/logging-view-plugin:v6.0.5"
UI_LOGGING_PF5 = "quay.io/openshift-observability-ui/logging-view-plugin:v6.1.6"
UI_TROUBLESHOOTING = "quay.io/openshift-observability-ui/troubleshooting-panel-console-plugin:v1.0.0"
UI_TROUBLESHOOTING_PF6 = "quay.io/openshift-observability-ui/troubleshooting-panel-console-plugin:v0.4.5"
UI_MONITORING = "quay.io/openshift-observability-ui/monitoring-console-plugin:v1.0.0"
UI_MONITORING_PF5 = "quay.io/openshift-observability-ui/monitoring-console-plugin:v0.4.5"
UI_MONITORING_PF6 = "quay.io/openshift-observability-ui/monitoring-console-plugin:v0.5.4"
KORREL8R = "quay.io/korrel8r/korrel8r:0.11.1"
HEALTH_ANALYZER = "quay.io/openshiftanalytics/cluster-health-analyzer:v1.1.1"


def operator_image(version):
    return f"observability-operator:{version}"


# image-references / relatedImages: (art distgit name, relatedImages name, pullspec)
# Order mirrors render_templates. `operator_image()` is filled in at runtime.
def image_table(version):
    return [
        ("cluster-observability-rhel9-operator", "cluster-observability-operator", operator_image(version)),
        ("obo-prometheus-operator-prometheus-config-reloader-rhel9", "prometheus-config-reloader", CONFIG_RELOADER),
        ("alertmanager-rhel9", "alertmanager", ALERTMANAGER),
        ("prometheus-rhel9", "prometheus", PROMETHEUS),
        ("thanos-rhel9", "thanos", THANOS),
        ("obo-prometheus-operator-admission-webhook-rhel9", "prometheus-operator-admission-webhook", ADMISSION_WEBHOOK),
        ("obo-prometheus-rhel9-operator", "prometheus-operator", PROMETHEUS_OPERATOR),
        ("dashboards-console-plugin-rhel9", "ui-dashboards", UI_DASHBOARDS),
        ("distributed-tracing-console-plugin-rhel9", "ui-tracing", UI_TRACING),
        ("distributed-tracing-console-plugin-pf5-rhel9", "ui-tracing-pf5", UI_TRACING_PF5),
        ("distributed-tracing-console-plugin-pf6-rhel9", "ui-tracing-pf6", UI_TRACING_PF6),
        ("distributed-tracing-console-plugin-pf4-rhel9", "ui-tracing-pf4", UI_TRACING_PF4),
        ("logging-console-plugin-rhel9", "ui-logging", UI_LOGGING),
        ("logging-console-plugin-pf4-rhel9", "ui-logging-pf4", UI_LOGGING_PF4),
        ("logging-console-plugin-pf5-rhel9", "ui-logging-pf5", UI_LOGGING_PF5),
        ("troubleshooting-panel-console-plugin-rhel9", "ui-troubleshooting", UI_TROUBLESHOOTING),
        ("troubleshooting-panel-console-plugin-pf6-rhel9", "ui-troubleshooting-pf6", UI_TROUBLESHOOTING_PF6),
        ("monitoring-console-plugin-rhel9", "ui-monitoring", UI_MONITORING),
        ("monitoring-console-plugin-pf5-rhel9", "ui-monitoring-pf5", UI_MONITORING_PF5),
        ("monitoring-console-plugin-pf6-rhel9", "ui-monitoring-pf6", UI_MONITORING_PF6),
        ("korrel8r-rhel9", "korrel8r", KORREL8R),
        ("cluster-health-analyzer-rhel9", "cluster-health-analyzer", HEALTH_ANALYZER),
        ("perses-rhel9", "perses", PERSES),
        ("perses-rhel9-operator", "perses-operator", PERSES_OPERATOR),
    ]


# Operator container env vars + matching --images args, in render_templates order.
OPERATOR_IMAGES = [
    ("RELATED_IMAGE_ALERTMANAGER", ALERTMANAGER, "alertmanager"),
    ("RELATED_IMAGE_PROMETHEUS", PROMETHEUS, "prometheus"),
    ("RELATED_IMAGE_THANOS", THANOS, "thanos"),
    ("RELATED_IMAGE_PERSES", PERSES, "perses"),
    ("RELATED_IMAGE_CONSOLE_DASHBOARDS_PLUGIN", UI_DASHBOARDS, "ui-dashboards"),
    ("RELATED_IMAGE_CONSOLE_DISTRIBUTED_TRACING_PLUGIN", UI_TRACING, "ui-distributed-tracing"),
    ("RELATED_IMAGE_CONSOLE_DISTRIBUTED_TRACING_PLUGIN_PF6", UI_TRACING_PF6, "ui-distributed-tracing-pf6"),
    ("RELATED_IMAGE_CONSOLE_DISTRIBUTED_TRACING_PLUGIN_PF5", UI_TRACING_PF5, "ui-distributed-tracing-pf5"),
    ("RELATED_IMAGE_CONSOLE_DISTRIBUTED_TRACING_PLUGIN_PF4", UI_TRACING_PF4, "ui-distributed-tracing-pf4"),
    ("RELATED_IMAGE_CONSOLE_LOGGING_PLUGIN", UI_LOGGING, "ui-logging"),
    ("RELATED_IMAGE_CONSOLE_LOGGING_PLUGIN_PF4", UI_LOGGING_PF4, "ui-logging-pf4"),
    ("RELATED_IMAGE_CONSOLE_LOGGING_PLUGIN_PF5", UI_LOGGING_PF5, "ui-logging-pf5"),
    ("RELATED_IMAGE_CONSOLE_TROUBLESHOOTING_PANEL_PLUGIN", UI_TROUBLESHOOTING, "ui-troubleshooting-panel"),
    ("RELATED_IMAGE_CONSOLE_TROUBLESHOOTING_PANEL_PLUGIN_PF6", UI_TROUBLESHOOTING_PF6, "ui-troubleshooting-panel-pf6"),
    ("RELATED_IMAGE_CONSOLE_MONITORING_PLUGIN", UI_MONITORING, "ui-monitoring"),
    ("RELATED_IMAGE_CONSOLE_MONITORING_PLUGIN_PF5", UI_MONITORING_PF5, "ui-monitoring-pf5"),
    ("RELATED_IMAGE_CONSOLE_MONITORING_PLUGIN_PF6", UI_MONITORING_PF6, "ui-monitoring-pf6"),
    ("RELATED_IMAGE_KORREL8R", KORREL8R, "korrel8r"),
    ("RELATED_IMAGE_CLUSTER_HEALTH_ANALYZER", HEALTH_ANALYZER, "health-analyzer"),
]

DESCRIPTION = """Cluster Observability Operator is a Go based Kubernetes operator to easily setup and manage various observability tools.
### Supported Features
- Setup multiple Highly Available Monitoring stack using Prometheus, Alertmanager and Thanos Querier
- Customizable configuration for managing Prometheus deployments
- Customizable configuration for managing Alertmanager deployments
- Customizable configuration for managing Thanos Querier deployments
- Setup console plugins
- Setup korrel8r
- Setup Perses
- Setup Cluster Health Analyzer
### Documentation
- **[Documentation](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/cluster_observability_operator/index)**
### License
Licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0)
"""

ICON = "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPHN2ZyBpZD0idXVpZC1kMWI4NDIzOC0wYzgxLTQ5MjctOGQ4Mi03OTcyN2Y5OGZjYWMiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgdmlld0JveD0iMCAwIDM4IDM4Ij4KPHRpdGxlPkNsdXN0ZXIgb2JzZXJ2YWJpbGl0eTwvdGl0bGU+CjxkZXNjPmNsb3VkPC9kZXNjPgo8bWV0YWRhdGE+PD94cGFja2V0IGJlZ2luPSLvu78iIGlkPSJXNU0wTXBDZWhpSHpyZVN6TlRjemtjOWQiPz4KPHg6eG1wbWV0YSB4bWxuczp4PSJhZG9iZTpuczptZXRhLyIgeDp4bXB0az0iQWRvYmUgWE1QIENvcmUgOC4wLWMwMDEgMS4wMDAwMDAsIDAwMDAvMDAvMDAtMDA6MDA6MDAiPgogICA8cmRmOlJERiB4bWxuczpyZGY9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkvMDIvMjItcmRmLXN5bnRheC1ucyMiPgogICAgICA8cmRmOkRlc2NyaXB0aW9uIHJkZjphYm91dD0iIgogICAgICAgICAgICB4bWxuczp4bXA9Imh0dHA6Ly9ucy5hZG9iZS5jb20veGFwLzEuMC8iCiAgICAgICAgICAgIHhtbG5zOnRpZmY9Imh0dHA6Ly9ucy5hZG9iZS5jb20vdGlmZi8xLjAvIgogICAgICAgICAgICB4bWxuczpkYz0iaHR0cDovL3B1cmwub3JnL2RjL2VsZW1lbnRzLzEuMS8iPgogICAgICAgICA8IS0tIG1ldGFkYXRhIGZpZWxkcyAtLT4KICAgICAgPC9yZGY6RGVzY3JpcHRpb24+CiAgIDwvcmRmOlJERj4KPC94OnhtcG1ldGE+Cjw/eHBhY2tldCBlbmQ9InciPz48L21ldGFkYXRhPgo8ZGVmcz48c3R5bGU+LnV1aWQtMjRiMGQ5N2ItNjgxZS00ZGE2LWExYzctNzY3MWFlNTc1MzJhe2ZpbGw6I2UwZTBlMDt9LnV1aWQtMjRiMGQ5N2ItNjgxZS00ZGE2LWExYzctNzY3MWFlNTc1MzJhLC51dWlkLTk3YzFlYzg0LTliODEtNDU2ZS05OWFhLTcxMzc1MGViNjllMCwudXVpZC1mMDMyMTc4ZS1iZDUwLTRkZTctYjc3My02NjViZmQ1YzViYjgsLnV1aWQtNGJlZjgyMGItNTZjOS00N2U3LTgyYTMtMmRiOGQ4YzdiMTEye3N0cm9rZS13aWR0aDowcHg7fS51dWlkLTk3YzFlYzg0LTliODEtNDU2ZS05OWFhLTcxMzc1MGViNjllMHtmaWxsOiMwMDA7fS51dWlkLWYwMzIxNzhlLWJkNTAtNGRlNy1iNzczLTY2NWJmZDVjNWJiOHtmaWxsOiNlMDA7fS51dWlkLTRiZWY4MjBiLTU2YzktNDdlNy04MmEzLTJkYjhkOGM3YjExMntmaWxsOiNmZmY7fTwvc3R5bGU+PC9kZWZzPgo8cmVjdCBjbGFzcz0idXVpZC00YmVmODIwYi01NmM5LTQ3ZTctODJhMy0yZGI4ZDhjN2IxMTIiIHg9IjEiIHk9IjEiIHdpZHRoPSIzNiIgaGVpZ2h0PSIzNiIgcng9IjkiIHJ5PSI5Ii8+CjxwYXRoIGNsYXNzPSJ1dWlkLTI0YjBkOTdiLTY4MWUtNGRhNi1hMWM3LTc2NzFhZTU3NTMyYSIgZD0iTTI4LDIuMjVjNC4yNzM0LDAsNy43NSwzLjQ3NjYsNy43NSw3Ljc1djE4YzAsNC4yNzM0LTMuNDc2Niw3Ljc1LTcuNzUsNy43NUgxMGMtNC4yNzM0LDAtNy43NS0zLjQ3NjYtNy43NS03Ljc1VjEwYzAtNC4yNzM0LDMuNDc2Ni03Ljc1LDcuNzUtNy43NWgxOE0yOCwxSDEwQzUuMDI5NCwxLDEsNS4wMjk0LDEsMTB2MThjMCw0Ljk3MDYsNC4wMjk0LDksOSw5aDE4YzQuOTcwNiwwLDktNC4wMjk0LDktOVYxMGMwLTQuOTcwNi00LjAyOTQtOS05LTloMFoiLz4KPHBhdGggY2xhc3M9InV1aWQtZjAzMjE3OGUtYmQ1MC00ZGU3LWI3NzMtNjY1YmZkNWM1YmI4IiBkPSJNMjEuMzc1LDE5YzAsLjM0NTIuMjgwMy42MjUuNjI1LjYyNXMuNjI1LS4yNzk4LjYyNS0uNjI1YzAtMS45OTktMS42MjYtMy42MjUtMy42MjUtMy42MjUtLjM0NDcsMC0uNjI1LjI3OTgtLjYyNS42MjVzLjI4MDMuNjI1LjYyNS42MjVjMS4zMDk2LDAsMi4zNzUsMS4wNjU0LDIuMzc1LDIuMzc1WiIvPgo8cGF0aCBjbGFzcz0idXVpZC1mMDMyMTc4ZS1iZDUwLTRkZTctYjc3My02NjViZmQ1YzViYjgiIGQ9Ik0xOSwxMy4zNzVjLS43ODIyLDAtMS41MzkxLjE1NzctMi4yNS40NjgzLS4zMTY0LjEzODItLjQ2MDkuNTA2OC0uMzIyMy44MjMyLjEzNzcuMzE2NC41MDc4LjQ1ODUuODIyMy4zMjIzLjU1MjctLjI0MTIsMS4xNDE2LS4zNjM4LDEuNzUtLjM2MzgsMi40MTIxLDAsNC4zNzUsMS45NjI0LDQuMzc1LDQuMzc1cy0xLjk2MjksNC4zNzUtNC4zNzUsNC4zNzUtNC4zNzUtMS45NjI0LTQuMzc1LTQuMzc1YzAtLjYwODkuMTIyMS0xLjE5NzMuMzYzMy0xLjc0OTUuMTM4Ny0uMzE2NC0uMDA1OS0uNjg1MS0uMzIyMy0uODIzMi0uMzE0NS0uMTM5Mi0uNjgzNi4wMDU0LS44MjIzLjMyMjMtLjMxMTUuNzExNC0uNDY4OCwxLjQ2ODMtLjQ2ODgsMi4yNTA1LDAsMy4xMDE2LDIuNTIzNCw1LjYyNSw1LjYyNSw1LjYyNXM1LjYyNS0yLjUyMzQsNS42MjUtNS42MjUtMi41MjM0LTUuNjI1LTUuNjI1LTUuNjI1WiIvPgo8cGF0aCBjbGFzcz0idXVpZC05N2MxZWM4NC05YjgxLTQ1NmUtOTlhYS03MTM3NTBlYjY5ZTAiIGQ9Ik0zMC40NjY4LDE4LjczODhjLTIuMDU2Ni00LjQ3MzEtNi41NTc2LTcuMzYzOC0xMS40NjY4LTcuMzYzOHMtOS40MTAyLDIuODkwNi0xMS40NjY4LDcuMzYzOGMtLjA3NTIuMTY2LS4wNzUyLjM1NjQsMCwuNTIyNSwyLjA1NjYsNC40NzMxLDYuNTU3Niw3LjM2MzgsMTEuNDY2OCw3LjM2MzhzOS40MTAyLTIuODkwNiwxMS40NjY4LTcuMzYzOGMuMDc1Mi0uMTY2LjA3NTItLjM1NjQsMC0uNTIyNVpNMTksMjUuMzc1Yy00LjMyNjIsMC04LjMwMDgtMi40OTI3LTEwLjIwNjEtNi4zNzUsMS45MDUzLTMuODgyMyw1Ljg3OTktNi4zNzUsMTAuMjA2MS02LjM3NXM4LjMwMDgsMi40OTI3LDEwLjIwNjEsNi4zNzVjLTEuOTA1MywzLjg4MjMtNS44Nzk5LDYuMzc1LTEwLjIwNjEsNi4zNzVaIi8+CjxwYXRoIGNsYXNzPSJ1dWlkLTk3YzFlYzg0LTliODEtNDU2ZS05OWFhLTcxMzc1MGViNjllMCIgZD0iTTE1LjQ0MjQsMTQuNTU4MWMtLjI0NDEtLjI0NDEtLjY0MDYtLjI0NDEtLjg4NDgsMC0uMjQzMi4yNDQxLS4yNDMyLjYzOTYsMCwuODgzOGw0LDRjLjEyMjEuMTIyMS4yODIyLjE4MzEuNDQyNC4xODMxcy4zMjAzLS4wNjEuNDQyNC0uMTgzMWMuMjQzMi0uMjQ0MS4yNDMyLS42Mzk2LDAtLjg4MzhsLTQtNFoiLz4KPC9zdmc+Cg=="


def find_deployment(csv, name):
    for d in csv["spec"]["install"]["spec"]["deployments"]:
        if d["name"] == name:
            return d
    raise KeyError(name)


def find_container(dep, name):
    for c in dep["spec"]["template"]["spec"]["containers"]:
        if c["name"] == name:
            return c
    raise KeyError(name)


def str_representer(dumper, data):
    if "\n" in data:
        return dumper.represent_scalar("tag:yaml.org,2002:str", data, style="|")
    return dumper.represent_scalar("tag:yaml.org,2002:str", data)


def previous_zstream(version):
    parts = version.split(".")
    if len(parts) == 3 and parts[2].isdigit() and int(parts[2]) > 0:
        parts[2] = str(int(parts[2]) - 1)
        return ".".join(parts)
    return None


def transform_csv(path, version):
    with open(path) as f:
        csv = yaml.safe_load(f)

    ann = csv["metadata"].setdefault("annotations", {})
    labels = csv["metadata"].setdefault("labels", {})

    # OpenShift feature annotations
    for k, v in [
        ("disconnected", "true"), ("fips-compliant", "false"),
        ("proxy-aware", "false"), ("tls-profiles", "false"),
        ("token-auth-aws", "false"), ("token-auth-azure", "false"),
        ("token-auth-gcp", "false"), ("cnf", "false"),
        ("cni", "false"), ("csi", "false"),
    ]:
        ann[f"features.operators.openshift.io/{k}"] = v
    ann["support"] = "Cluster Observability (https://issues.redhat.com/projects/COO/)"
    ann["operators.openshift.io/valid-subscription"] = (
        '["OpenShift Kubernetes Engine", "OpenShift Container Platform", "OpenShift Platform Plus"]'
    )

    # arch labels
    for arch in ("amd64", "arm64", "ppc64le", "s390x"):
        labels[f"operatorframework.io/arch.{arch}"] = "supported"

    # relatedImages (disconnected support)
    csv["spec"]["relatedImages"] = [
        {"name": rel, "image": pull} for (_art, rel, pull) in image_table(version)
    ]

    # perses-operator: TLS profile args
    perses = find_container(find_deployment(csv, "perses-operator"), "perses-operator")
    perses.setdefault("args", [])
    perses["args"] += ["--tls-cluster-profile", "--tls-configure-operands"]

    # obo-prometheus-operator: config-reloader via RELATED_IMAGE env
    po = find_container(find_deployment(csv, "obo-prometheus-operator"), "prometheus-operator")
    po.setdefault("args", [])
    po["args"] = [
        "--prometheus-config-reloader=$(RELATED_IMAGE_PROMETHEUS_CONFIG_RELOADER)"
        if a.startswith("--prometheus-config-reloader=") else a
        for a in po["args"]
    ]
    po.setdefault("env", [])
    po["env"].append({
        "name": "RELATED_IMAGE_PROMETHEUS_CONFIG_RELOADER",
        "value": CONFIG_RELOADER,
    })

    # observability-operator: RELATED_IMAGE envs + --images args
    op = find_container(find_deployment(csv, "observability-operator"), "operator")
    op.setdefault("args", [])
    op.setdefault("env", [])
    for env_name, pullspec, images_key in OPERATOR_IMAGES:
        op["args"].append(f"--images={images_key}=$({env_name})")
        op["env"].append({"name": env_name, "value": pullspec})
    op["args"].append("--openshift.enabled=true")

    # downstream naming / versioning
    csv["metadata"]["name"] = f"{PKG}.v{version}"
    csv["spec"]["version"] = version
    csv["spec"]["displayName"] = "Cluster Observability Operator"
    csv["spec"]["description"] = DESCRIPTION
    replaces = previous_zstream(version)
    if replaces:
        csv["spec"]["replaces"] = f"{PKG}.v{replaces}"
    ann["olm.skipRange"] = f">=0.1.0 <{version}"
    ann["operatorframework.io/suggested-namespace"] = "openshift-cluster-observability-operator"
    csv["spec"]["icon"] = [{"base64data": ICON, "mediatype": "image/svg+xml"}]

    # fix observability-operator version labels
    oo = find_deployment(csv, "observability-operator")
    oo["label"]["app.kubernetes.io/version"] = version
    oo["spec"]["template"]["metadata"]["labels"]["app.kubernetes.io/version"] = version

    yaml.add_representer(str, str_representer)
    with open(path, "w") as f:
        yaml.dump(csv, f, sort_keys=False, default_flow_style=False,
                  width=10**9, allow_unicode=True)


def write_image_references(path, version):
    lines = [
        "---",
        "# ART bundle image-references (generated by hack/render-art-bundle.py).",
        "#",
        "# ART's olm_bundle_konflux builder maps each ART (distgit) image name ->",
        "# the upstream pullspec used in the CSV, replaces every occurrence of",
        "# `from.name` with the pinned registry.redhat.io pullspec, and regenerates",
        "# spec.relatedImages. Each `from.name` MUST appear literally in the CSV.",
        "kind: ImageStream",
        "apiVersion: image.openshift.io/v1",
        "spec:",
        "  tags:",
    ]
    for art, _rel, pull in image_table(version):
        lines += [
            f"  - name: {art}",
            "    from:",
            "      kind: DockerImage",
            f"      name: {pull}",
        ]
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")


def write_art_yaml(path, version):
    replaces = previous_zstream(version)
    csv_file = f"manifests/{PKG}.clusterserviceversion.yaml"
    anchors = [
        f"olm.skipRange: '>=0.1.0 <{version}'",
        f"version: {version}",
        f"name: {PKG}.v{version}",
    ]
    if replaces:
        anchors.append(f"replaces: {PKG}.v{replaces}")
    lines = [
        "---",
        "# ART bundle metadata (generated by hack/render-art-bundle.py).",
        "#",
        "# All images are built by ART and mapped in bundle-art/image-references,",
        "# so no `external-images` section is required. The `updates` block marks the",
        "# version-bearing strings ART's rebase automation keeps in sync (search ==",
        "# replace: identity anchors rewritten on the next version bump).",
        "updates:",
        f'  - file: "{csv_file}" # relative to this file',
        "    update_list:",
    ]
    for a in anchors:
        lines += [
            f'      - search: "{a}"',
            f'        replace: "{a}"',
        ]
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")


def write_package_yaml(path, version):
    lines = [
        "---",
        "# OLM package definition for the downstream (ART) bundle.",
        "# Required by ART's olm_bundle_konflux builder (otherwise it fails with",
        "# \"IndexError: list index out of range\"). Channels mirror the product FBC",
        "# (github.com/rhobs/konflux-coo-fbc).",
        f"packageName: {PKG}",
        f"defaultChannel: {DEFAULT_CHANNEL}",
        "channels:",
    ]
    for ch in CHANNELS:
        lines += [f"- name: {ch}", f"  currentCSV: {PKG}.v{version}"]
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")


def write_annotations(path):
    lines = [
        "annotations:",
        "  # Core bundle annotations.",
        "  operators.operatorframework.io.bundle.mediatype.v1: registry+v1",
        "  operators.operatorframework.io.bundle.manifests.v1: manifests/",
        "  operators.operatorframework.io.bundle.metadata.v1: metadata/",
        f"  operators.operatorframework.io.bundle.package.v1: {PKG}",
        f"  operators.operatorframework.io.bundle.channels.v1: {','.join(CHANNELS)}",
        f"  operators.operatorframework.io.bundle.channel.default.v1: {DEFAULT_CHANNEL}",
        "  operators.operatorframework.io.metrics.builder: operator-sdk-v1.41.1",
        "  operators.operatorframework.io.metrics.mediatype.v1: metrics+v1",
        "  operators.operatorframework.io.metrics.project_layout: unknown",
        "",
        "  # Annotations for testing.",
        "  operators.operatorframework.io.test.mediatype.v1: scorecard+v1",
        "  operators.operatorframework.io.test.config.v1: tests/scorecard/",
    ]
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")


def main():
    src_csv = os.path.join(SRC, "manifests", "observability-operator.clusterserviceversion.yaml")
    if not os.path.exists(src_csv):
        sys.exit(f"error: {src_csv} not found; run `make bundle` first")

    with open(src_csv) as f:
        version = yaml.safe_load(f)["spec"]["version"]

    # 1. fresh copy of the upstream bundle
    if os.path.exists(DST):
        shutil.rmtree(DST)
    shutil.copytree(SRC, DST)

    # 2. rename + transform the CSV
    dst_csv = os.path.join(DST, "manifests", f"{PKG}.clusterserviceversion.yaml")
    os.rename(os.path.join(DST, "manifests", "observability-operator.clusterserviceversion.yaml"), dst_csv)
    transform_csv(dst_csv, version)

    # 3. downstream metadata + ART files
    write_annotations(os.path.join(DST, "metadata", "annotations.yaml"))
    write_image_references(os.path.join(DST, "image-references"), version)
    write_art_yaml(os.path.join(DST, "art.yaml"), version)
    write_package_yaml(os.path.join(DST, f"{PKG}.package.yaml"), version)

    print(f"rendered {os.path.relpath(DST, REPO_ROOT)} for version {version}")


if __name__ == "__main__":
    main()
